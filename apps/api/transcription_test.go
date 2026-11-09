package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/transcribe"
)

func TestTranscriptionIsSentBackAndNotPublished(t *testing.T) {
	rec := &recorder{}
	audio, err := newAudioHandler(t.Context(), transcribe.Fake{}, fakeVocabulary{}, rec.publish, rec)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(audio.ServeTranscription))
	t.Cleanup(server.Close)
	conn := dial(t, "ws"+strings.TrimPrefix(server.URL, "http"))

	lineAndAHalf := 3 * transcribe.SampleRate * 2
	frame := make([]byte, 9600)
	for sent := 0; sent < lineAndAHalf; sent += len(frame) {
		if err := conn.Write(t.Context(), websocket.MessageBinary, frame); err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.Write(t.Context(), websocket.MessageText, []byte(endOfAudio)); err != nil {
		t.Fatal(err)
	}

	var got []transcript
	for {
		_, data, err := conn.Read(t.Context())
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		var line transcript
		if err := json.Unmarshal(data, &line); err != nil {
			t.Fatal(err)
		}
		got = append(got, line)
	}

	if len(got) != 3 || got[0].Final || !got[1].Final || got[2].Final || got[0].ID != got[1].ID || got[1].ID == got[2].ID {
		t.Fatalf("want a partial and final of one line, then a partial of the next: %+v", got)
	}
	if !strings.Contains(got[1].Text, "hallway track") {
		t.Errorf("final %+v", got[1])
	}
	if len(rec.captions()) != 0 || len(rec.stored) != 0 {
		t.Errorf("trying transcription must not publish or store captions")
	}
}
