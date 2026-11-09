package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/transcribe"
)

// fakeVocabulary corrects "silly um" to "Cilium", like the real vocabulary.
type fakeVocabulary struct{}

func (fakeVocabulary) Correct(_ context.Context, text string) (search.Correction, error) {
	before, after, found := strings.Cut(text, "silly um")
	if !found {
		return search.Correction{Text: text, Segments: []search.Segment{{Text: text}}}, nil
	}
	return search.Correction{
		Text:     before + "Cilium" + after,
		Segments: []search.Segment{{Text: before}, {Text: "Cilium", Heard: "silly um"}, {Text: after}},
	}, nil
}

type recorder struct {
	mu        sync.Mutex
	published []captions.Caption
	stored    []storedCaption
}

type storedCaption struct {
	line  captions.Caption
	heard string
}

func (r *recorder) Store(_ context.Context, line captions.Caption, heard string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stored = append(r.stored, storedCaption{line, heard})
	return nil
}

func (r *recorder) publish(_ context.Context, c captions.Caption) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.published = append(r.published, c)
	return nil
}

func (r *recorder) captions() []captions.Caption {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]captions.Caption(nil), r.published...)
}

// startAPI returns a function that builds a talk's audio URL.
func startAPI(t *testing.T, shutdown context.Context) (*recorder, func(talkID string) string) {
	t.Helper()
	rec := &recorder{}
	audio, err := newAudioHandler(shutdown, transcribe.Fake{}, fakeVocabulary{}, rec.publish, rec)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/talks/{talkID}/audio", audio)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return rec, func(talkID string) string {
		return "ws" + strings.TrimPrefix(server.URL, "http") + "/api/talks/" + talkID + "/audio"
	}
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, res, err := websocket.Dial(t.Context(), url, nil)
	if res != nil && res.Body != nil {
		_ = res.Body.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

func eventually(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAudioIsTranscribedCorrectedAndPublished(t *testing.T) {
	rec, audioURL := startAPI(t, t.Context())
	conn := dial(t, audioURL("demo"))

	twoLines := 2 * 2 * transcribe.SampleRate * 2
	frame := make([]byte, 9600)
	for sent := 0; sent < twoLines; sent += len(frame) {
		if err := conn.Write(t.Context(), websocket.MessageBinary, frame); err != nil {
			t.Fatal(err)
		}
	}
	eventually(t, func() bool { return len(rec.captions()) == 4 })
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}

	got := rec.captions()
	for i, c := range got {
		if c.TalkID != "demo" || c.Final != (i%2 == 1) || c.Corrections == nil {
			t.Errorf("caption %d: %+v", i, c)
		}
	}
	partial, final := got[2], got[3]
	if partial.ID != final.ID || partial.At != final.At {
		t.Errorf("a line's partial and final must share id and at: %+v, %+v", partial, final)
	}
	if got[0].ID == final.ID || got[0].At > final.At {
		t.Errorf("lines out of order: %+v before %+v", got[0], final)
	}
	if strings.Contains(partial.Text, "Cilium") || len(partial.Corrections) != 0 {
		t.Errorf("partials go out as heard: %+v", partial)
	}
	want := []captions.Correction{{Start: 33, End: 39, Heard: "silly um"}}
	if !strings.Contains(final.Text, "over Cilium and eBPF") || !reflect.DeepEqual(final.Corrections, want) {
		t.Errorf("final not corrected: %+v", final)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.stored) != 2 {
		t.Fatalf("want only the 2 finals stored, got %d", len(rec.stored))
	}
	if s := rec.stored[1]; !reflect.DeepEqual(s.line, final) || !strings.Contains(s.heard, "over silly um and") {
		t.Errorf("stored %+v", s)
	}
}

func TestCaptionCorrectionsCountUTF16(t *testing.T) {
	got := captionCorrections([]search.Segment{
		{Text: "Café 🚀 "},
		{Text: "Cilium", Heard: "silly um"},
		{Text: "!"},
	})
	want := []captions.Correction{{Start: 8, End: 14, Heard: "silly um"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAudioRejectsNonPCMFrames(t *testing.T) {
	_, audioURL := startAPI(t, t.Context())
	conn := dial(t, audioURL("demo"))

	if err := conn.Write(t.Context(), websocket.MessageBinary, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	_, _, err := conn.Read(t.Context())
	if websocket.CloseStatus(err) != websocket.StatusUnsupportedData {
		t.Fatalf("got %v", err)
	}
}

func TestAudioClosesSpeakersOnShutdown(t *testing.T) {
	shutdown, cancel := context.WithCancel(t.Context())
	_, audioURL := startAPI(t, shutdown)
	conn := dial(t, audioURL("demo"))

	cancel()
	ctx, stop := context.WithTimeout(t.Context(), 2*time.Second)
	defer stop()
	_, _, err := conn.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusGoingAway {
		t.Fatalf("got %v", err)
	}
}

func TestAudioRejectsInvalidTalkID(t *testing.T) {
	_, audioURL := startAPI(t, t.Context())
	_, res, err := websocket.Dial(t.Context(), audioURL("Bad_ID"), nil)
	if res != nil && res.Body != nil {
		_ = res.Body.Close()
	}
	if err == nil || res == nil || res.StatusCode != http.StatusNotFound {
		t.Fatalf("got %v, %v", res, err)
	}
}

func TestFinalsAreStoredWhenPublishingFails(t *testing.T) {
	rec := &recorder{}
	failing := func(context.Context, captions.Caption) error { return errors.New("redis down") }
	audio, err := newAudioHandler(t.Context(), transcribe.Fake{}, fakeVocabulary{}, failing, rec)
	if err != nil {
		t.Fatal(err)
	}
	audio.publishEvent(t.Context(), captions.Caption{TalkID: "demo", ID: "s-1", Text: "over silly um", Final: true}, time.Time{})

	if len(rec.stored) != 1 || rec.stored[0].line.Text != "over Cilium" {
		t.Fatalf("stored %+v", rec.stored)
	}
}

// trailingTranscriber holds back its only line until Close, like a speaker
// cut off mid-sentence whose last words are still being transcribed.
type trailingTranscriber struct{}

func (trailingTranscriber) Start(context.Context, transcribe.Options) (transcribe.Session, error) {
	return &trailingSession{events: make(chan transcribe.Event, 1)}, nil
}

type trailingSession struct{ events chan transcribe.Event }

func (s *trailingSession) Append(context.Context, []byte) error { return nil }
func (s *trailingSession) Events() <-chan transcribe.Event      { return s.events }

func (s *trailingSession) Close(context.Context) error {
	time.Sleep(100 * time.Millisecond)
	s.events <- transcribe.Event{SegmentID: "last", Text: "Thank you.", Final: true}
	close(s.events)
	return nil
}

func TestShutdownWaitsForTheLastCaption(t *testing.T) {
	shutdown, stopServer := context.WithCancel(t.Context())
	requests, cancelRequests := context.WithCancel(t.Context())

	var mu sync.Mutex
	var published []captions.Caption
	publish := func(ctx context.Context, c captions.Caption) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		published = append(published, c)
		return nil
	}
	audio, err := newAudioHandler(shutdown, trailingTranscriber{}, fakeVocabulary{}, publish, &recorder{})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/talks/{talkID}/audio", audio)
	server := httptest.NewUnstartedServer(mux)
	server.Config.BaseContext = func(net.Listener) context.Context { return requests }
	server.Start()
	t.Cleanup(server.Close)

	conn := dial(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/talks/demo/audio")
	go func() { _, _, _ = conn.Read(context.Background()) }()

	// What httpserver.Run does on SIGTERM: shut down, then cancel request contexts.
	stopServer()
	cancelRequests()

	wait, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	audio.Wait(wait)
	if wait.Err() != nil {
		t.Fatal("Wait timed out")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(published) != 1 || published[0].Text != "Thank you." {
		t.Fatalf("the last caption was lost: %+v", published)
	}
}
