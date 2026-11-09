package transcribe

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// fakeRealtime serves one Realtime session: it hands each client event to
// onEvent, which replies with scripted server events.
func fakeRealtime(t *testing.T, onEvent func(event map[string]any) []string) *OpenAI {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = conn.CloseNow() }()
		conn.SetReadLimit(1 << 20)
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var event map[string]any
			if err := json.Unmarshal(data, &event); err != nil {
				t.Error(err)
				return
			}
			for _, reply := range onEvent(event) {
				if err := conn.Write(r.Context(), websocket.MessageText, []byte(reply)); err != nil {
					return
				}
			}
		}
	}))
	t.Cleanup(server.Close)
	return &OpenAI{APIKey: "test-key", URL: "ws" + strings.TrimPrefix(server.URL, "http")}
}

func collect(t *testing.T, events <-chan Event, n int) []Event {
	t.Helper()
	var got []Event
	for len(got) < n {
		select {
		case e, ok := <-events:
			if !ok {
				t.Fatalf("events closed after %d of %d: %+v", len(got), n, got)
			}
			got = append(got, e)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out after %d of %d events: %+v", len(got), n, got)
		}
	}
	return got
}

func TestOpenAISessionMapsRealtimeEvents(t *testing.T) {
	audio := []byte{1, 2, 3, 4}
	var config map[string]any
	o := fakeRealtime(t, func(event map[string]any) []string {
		switch event["type"] {
		case "session.update":
			config = event
		case "input_audio_buffer.append":
			got, _ := base64.StdEncoding.DecodeString(event["audio"].(string))
			if !bytes.Equal(got, audio) {
				t.Errorf("audio: got %v, want %v", got, audio)
			}
			return []string{
				`{"type":"input_audio_buffer.committed","item_id":"item_1"}`,
				`{"type":"conversation.item.input_audio_transcription.delta","item_id":"item_1","delta":"Hello"}`,
				`{"type":"conversation.item.input_audio_transcription.delta","item_id":"item_1","delta":" world"}`,
				`{"type":"input_audio_buffer.committed","item_id":"item_2"}`,
				`{"type":"conversation.item.input_audio_transcription.completed","item_id":"item_2","transcript":"Second."}`,
				`{"type":"conversation.item.input_audio_transcription.completed","item_id":"item_1","transcript":"Hello world."}`,
			}
		case "input_audio_buffer.commit":
			return []string{`{"type":"error","error":{"code":"input_audio_buffer_commit_empty","message":"buffer empty"}}`}
		}
		return nil
	})

	s, err := o.Start(t.Context(), Options{Language: "en", Prompt: "Kubernetes, KEDA"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Append(t.Context(), audio); err != nil {
		t.Fatal(err)
	}

	got := collect(t, s.Events(), 4)
	want := []Event{
		{SegmentID: "item_1", Text: "Hello"},
		{SegmentID: "item_1", Text: "Hello world"},
		{SegmentID: "item_2", Text: "Second.", Final: true},
		{SegmentID: "item_1", Text: "Hello world.", Final: true},
	}
	for i := range want {
		if got[i].CommittedAt.IsZero() {
			t.Errorf("event %d has no commit time", i)
		}
		got[i].CommittedAt = time.Time{}
		if got[i] != want[i] {
			t.Errorf("event %d: got %+v, want %+v", i, got[i], want[i])
		}
	}

	transcription := config["session"].(map[string]any)["audio"].(map[string]any)["input"].(map[string]any)["transcription"].(map[string]any)
	if transcription["model"] != openAIModel || transcription["language"] != "en" || transcription["prompt"] != "Kubernetes, KEDA" {
		t.Errorf("transcription config: %v", transcription)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil {
		t.Fatal("Close waited for its deadline instead of the empty-commit error")
	}
	if _, ok := <-s.Events(); ok {
		t.Fatal("events not closed")
	}
}

func TestOpenAISessionCloseFlushesTrailingSpeech(t *testing.T) {
	o := fakeRealtime(t, func(event map[string]any) []string {
		if event["type"] == "input_audio_buffer.commit" {
			return []string{
				`{"type":"input_audio_buffer.committed","item_id":"item_9"}`,
				`{"type":"conversation.item.input_audio_transcription.completed","item_id":"item_9","transcript":"Thank you."}`,
			}
		}
		return nil
	})
	s, err := o.Start(t.Context(), Options{})
	if err != nil {
		t.Fatal(err)
	}

	var got []Event
	done := make(chan struct{})
	go func() {
		for e := range s.Events() {
			got = append(got, e)
		}
		close(done)
	}()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
	<-done
	if len(got) != 1 || got[0].Text != "Thank you." || !got[0].Final {
		t.Fatalf("got %+v", got)
	}
}

func TestOpenAIStartFailsWithBadKey(t *testing.T) {
	o := fakeRealtime(t, func(map[string]any) []string { return nil })
	o.APIKey = "wrong"
	if _, err := o.Start(t.Context(), Options{}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestFakeEmitsPartialThenFinal(t *testing.T) {
	s, _ := Fake{}.Start(t.Context(), Options{})
	if err := s.Append(t.Context(), make([]byte, fakeSegmentBytes)); err != nil {
		t.Fatal(err)
	}
	got := collect(t, s.Events(), 2)
	if got[0].Final || !got[1].Final || got[0].SegmentID != got[1].SegmentID {
		t.Fatalf("got %+v", got)
	}
	_ = s.Close(t.Context())
}
