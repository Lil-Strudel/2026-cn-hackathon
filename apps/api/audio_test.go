package main

import (
	"context"
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
	audio, err := newAudioHandler(shutdown, transcribe.Fake{}, fakeVocabulary{}, rec.publish)
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
