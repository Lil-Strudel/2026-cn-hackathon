package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/notes"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
)

type transcriptHistory []captions.Caption

func (f transcriptHistory) History(context.Context, string) ([]captions.Caption, error) {
	return f, nil
}

type fakeNotesStore struct {
	mu    sync.Mutex
	saved map[string]notes.Notes
}

func (f *fakeNotesStore) Get(_ context.Context, talkID string) (*notes.Notes, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n, ok := f.saved[talkID]; ok {
		return &n, nil
	}
	return nil, nil
}

func (f *fakeNotesStore) Store(_ context.Context, talkID string, n notes.Notes) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saved == nil {
		f.saved = map[string]notes.Notes{}
	}
	f.saved[talkID] = n
	return nil
}

type fakeWriter struct {
	calls   atomic.Int32
	release chan struct{}
	err     error
}

func (f *fakeWriter) Write(context.Context, string) (notes.Notes, error) {
	f.calls.Add(1)
	if f.release != nil {
		<-f.release
	}
	if f.err != nil {
		return notes.Notes{}, f.err
	}
	return notes.Notes{Summary: "About Cilium.", Notes: []string{"eBPF routes traffic"}}, nil
}

type talkStatus search.TalkStatus

func (s talkStatus) Get(_ context.Context, id string) (*search.Talk, error) {
	if s == "" {
		return nil, nil
	}
	return &search.Talk{ID: id, Status: search.TalkStatus(s)}, nil
}

const (
	ended = talkStatus(search.TalkFinished)
	live  = talkStatus(search.TalkLive)
)

func spoken(words int) transcriptHistory {
	return transcriptHistory{{
		TalkID: "keynote",
		At:     "2026-11-09T15:00:00.000Z",
		Text:   strings.TrimSpace(strings.Repeat("word ", words)),
	}}
}

func serveNotes(t *testing.T, tn *talkNotes, method, talkID string) (int, notesBody) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/talks/{talkID}/notes", tn.handleGet)
	mux.HandleFunc("POST /api/talks/{talkID}/notes", tn.handleRewrite)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/api/talks/"+talkID+"/notes", nil))
	var body notesBody
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, body
}

func TestNotesWrittenOnceAndStored(t *testing.T) {
	writer := &fakeWriter{}
	store := &fakeNotesStore{}
	tn := &talkNotes{history: spoken(100), store: store, talks: ended, writer: writer}

	for range 2 {
		code, body := serveNotes(t, tn, http.MethodGet, "keynote")
		if code != http.StatusOK || body.Summary == nil || *body.Summary != "About Cilium." || len(body.Notes) != 1 {
			t.Fatalf("code = %d, body = %+v", code, body)
		}
	}
	if got := writer.calls.Load(); got != 1 {
		t.Errorf("writer called %d times, want 1", got)
	}
	if _, ok := store.saved["keynote"]; !ok {
		t.Error("notes were not stored")
	}
}

func TestNotesNotWritten(t *testing.T) {
	tests := []struct {
		name    string
		history transcriptHistory
		talk    talkStatus
		writer  bool
	}{
		{"without an OpenAI key", spoken(100), ended, false},
		{"for a short transcript", spoken(minTranscriptWords - 1), ended, true},
		{"while the talk is live", spoken(100), live, true},
		{"for a missing talk", spoken(100), "", true},
		{"without captions", transcriptHistory{}, ended, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &fakeWriter{}
			tn := &talkNotes{history: tt.history, store: &fakeNotesStore{}, talks: tt.talk}
			if tt.writer {
				tn.writer = writer
			}
			code, body := serveNotes(t, tn, http.MethodGet, "keynote")
			if code != http.StatusOK || body.Summary != nil || body.Notes != nil {
				t.Errorf("code = %d, body = %+v, want nulls", code, body)
			}
			if writer.calls.Load() != 0 {
				t.Error("writer was called")
			}
		})
	}
}

func TestRewriteReplacesStoredNotesOfALiveTalk(t *testing.T) {
	writer := &fakeWriter{}
	store := &fakeNotesStore{saved: map[string]notes.Notes{"keynote": {Summary: "old"}}}
	tn := &talkNotes{history: spoken(100), store: store, talks: live, writer: writer}

	code, body := serveNotes(t, tn, http.MethodPost, "keynote")
	if code != http.StatusOK || body.Summary == nil || *body.Summary != "About Cilium." {
		t.Fatalf("code = %d, body = %+v", code, body)
	}
	if store.saved["keynote"].Summary != "About Cilium." {
		t.Error("rewritten notes were not stored")
	}
}

func TestConcurrentViewersShareOneWrite(t *testing.T) {
	writer := &fakeWriter{release: make(chan struct{})}
	tn := &talkNotes{history: spoken(100), store: &fakeNotesStore{}, talks: ended, writer: writer}

	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			if code, _ := serveNotes(t, tn, http.MethodGet, "keynote"); code != http.StatusOK {
				t.Errorf("code = %d", code)
			}
		})
	}
	for writer.calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	close(writer.release)
	wg.Wait()
	if got := writer.calls.Load(); got != 1 {
		t.Errorf("writer called %d times, want 1", got)
	}
}

func TestNotesErrors(t *testing.T) {
	failing := &fakeWriter{err: errors.New("openai down")}
	tn := &talkNotes{history: spoken(100), store: &fakeNotesStore{}, talks: ended, writer: failing}
	if code, _ := serveNotes(t, tn, http.MethodGet, "keynote"); code != http.StatusBadGateway {
		t.Errorf("writer failure: code = %d, want 502", code)
	}
	if code, _ := serveNotes(t, tn, http.MethodGet, "Not_A_Talk"); code != http.StatusNotFound {
		t.Errorf("invalid talk id: code = %d, want 404", code)
	}
	tn.writer = nil
	if code, _ := serveNotes(t, tn, http.MethodPost, "keynote"); code != http.StatusServiceUnavailable {
		t.Errorf("rewrite without a key: code = %d, want 503", code)
	}
	tn.history = spoken(maxTranscriptBytes/5 + 1)
	tn.writer = &fakeWriter{}
	if code, _ := serveNotes(t, tn, http.MethodPost, "keynote"); code != http.StatusUnprocessableEntity {
		t.Errorf("transcript too long: code = %d, want 422", code)
	}
}

func TestEndingATalkWritesItsNotesOnce(t *testing.T) {
	writer := &fakeWriter{release: make(chan struct{})}
	store := &fakeNotesStore{}
	tn := &talkNotes{history: spoken(100), store: store, talks: ended, writer: writer}

	tn.start(t.Context(), "keynote")
	for writer.calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	viewer := make(chan notesBody)
	go func() {
		_, body := serveNotes(t, tn, http.MethodGet, "keynote")
		viewer <- body
	}()
	time.Sleep(20 * time.Millisecond)
	close(writer.release)

	if body := <-viewer; body.Summary == nil || *body.Summary != "About Cilium." {
		t.Fatalf("a viewer during the write got %+v", body)
	}
	tn.start(t.Context(), "keynote")
	if code, body := serveNotes(t, tn, http.MethodGet, "keynote"); code != http.StatusOK || body.Summary == nil {
		t.Fatalf("code = %d, body = %+v", code, body)
	}
	time.Sleep(20 * time.Millisecond)
	if got := writer.calls.Load(); got != 1 {
		t.Errorf("writer called %d times, want 1", got)
	}
}
