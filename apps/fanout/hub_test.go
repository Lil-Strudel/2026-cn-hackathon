package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
)

func testHub(t *testing.T) *hub {
	t.Helper()
	h, err := newHub()
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func encode(t *testing.T, c captions.Caption) []byte {
	t.Helper()
	data, err := captions.Encode(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDispatchReachesOnlyThatTalksClients(t *testing.T) {
	h := testHub(t)
	demo := h.subscribe(t.Context(), "demo")
	other := h.subscribe(t.Context(), "other")

	h.dispatch(t.Context(), encode(t, captions.Caption{TalkID: "demo", ID: "s1", Text: "hello"}))

	select {
	case data := <-demo.messages:
		var got captions.Caption
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if got.Text != "hello" || got.ID != "s1" {
			t.Fatalf("got %+v", got)
		}
	default:
		t.Fatal("demo client got nothing")
	}
	if len(other.messages) != 0 {
		t.Fatal("other talk received demo's caption")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	h := testHub(t)
	c := h.subscribe(t.Context(), "demo")
	h.unsubscribe(t.Context(), c)

	h.dispatch(t.Context(), encode(t, captions.Caption{TalkID: "demo", Text: "hello"}))

	if len(c.messages) != 0 {
		t.Fatal("unsubscribed client received a caption")
	}
	if len(h.talks) != 0 {
		t.Fatalf("empty talk left behind: %v", h.talks)
	}
}

func TestSlowClientIsDropped(t *testing.T) {
	h := testHub(t)
	slow := h.subscribe(t.Context(), "demo")
	msg := encode(t, captions.Caption{TalkID: "demo", Text: "hello"})

	for range clientBuffer + 1 {
		h.dispatch(t.Context(), msg)
	}

	for range clientBuffer {
		<-slow.messages
	}
	if _, ok := <-slow.messages; ok {
		t.Fatal("slow client's channel should be closed")
	}
	h.unsubscribe(t.Context(), slow)
}

func TestServeEventsStreamsCaptions(t *testing.T) {
	h := testHub(t)
	shutdown, cancel := context.WithCancel(t.Context())
	server := httptest.NewServer(newMux(shutdown, redis.NewClient(&redis.Options{}), h))
	defer server.Close()

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/events/talks/demo", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}

	lines := bufio.NewScanner(res.Body)
	readUntil := func(prefix string) string {
		t.Helper()
		for lines.Scan() {
			if strings.HasPrefix(lines.Text(), prefix) {
				return lines.Text()
			}
		}
		t.Fatalf("stream ended before %q: %v", prefix, lines.Err())
		return ""
	}

	readUntil(": connected")
	h.dispatch(t.Context(), encode(t, captions.Caption{TalkID: "demo", ID: "s1", Text: "hello", Final: true}))
	readUntil("event: caption")
	data := strings.TrimPrefix(readUntil("data: "), "data: ")
	if !strings.Contains(data, `"corrections":[]`) {
		t.Fatalf("corrections must reach the client as an array: %s", data)
	}
	var got captions.Caption
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatal(err)
	}
	if got.Text != "hello" || !got.Final {
		t.Fatalf("got %+v", got)
	}

	cancel()
	done := make(chan struct{})
	go func() {
		for lines.Scan() {
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stream stayed open after shutdown")
	}
}

func TestServeEventsRejectsInvalidTalkID(t *testing.T) {
	h := testHub(t)
	mux := newMux(t.Context(), redis.NewClient(&redis.Options{}), h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/events/talks/Not_Valid", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d", rec.Code)
	}
}
