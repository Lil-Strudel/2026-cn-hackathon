package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
)

type fakeTalks struct {
	talks   map[string]search.Talk
	created []search.NewTalk
	queries []string
	err     error
}

func (f *fakeTalks) List(context.Context) ([]search.Talk, error) {
	var list []search.Talk
	for _, talk := range f.talks {
		list = append(list, talk)
	}
	return list, f.err
}

func (f *fakeTalks) Get(_ context.Context, id string) (*search.Talk, error) {
	if talk, ok := f.talks[id]; ok {
		return &talk, f.err
	}
	return nil, f.err
}

func (f *fakeTalks) Create(_ context.Context, n search.NewTalk) (search.Talk, error) {
	f.created = append(f.created, n)
	return search.Talk{ID: "guitar-demo", Title: n.Title, Speaker: n.Speaker, Status: search.TalkLive}, f.err
}

func (f *fakeTalks) End(_ context.Context, id string) (*search.Talk, error) {
	talk, ok := f.talks[id]
	if !ok || f.err != nil {
		return nil, f.err
	}
	if talk.EndedAt == nil {
		now := time.Now()
		talk.Status, talk.EndedAt = search.TalkFinished, &now
		f.talks[id] = talk
	}
	return &talk, nil
}

func (f *fakeTalks) Search(_ context.Context, query string) ([]search.SearchHit, error) {
	f.queries = append(f.queries, query)
	return nil, f.err
}

type captionCounts map[string]int64

func (c captionCounts) Count(_ context.Context, talkID string) (int64, error) {
	return c[talkID], nil
}

func serveTalks(t *testing.T, store talkStore, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serveEnd(t, store, captionCounts{"guitar-demo": 3}, nil, method, target, body)
}

func serveEnd(t *testing.T, store talkStore, lines captionCounter, ended *[]string, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	onEnd := func(_ context.Context, talkID string) {
		if ended != nil {
			*ended = append(*ended, talkID)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/talks", handleListTalks(store))
	mux.HandleFunc("POST /api/talks", handleCreateTalk(store))
	mux.HandleFunc("GET /api/talks/{talkID}", handleGetTalk(store))
	mux.HandleFunc("POST /api/talks/{talkID}/end", handleEndTalk(store, lines, onEnd))
	mux.HandleFunc("GET /api/search", handleSearchTalks(store))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body)))
	return rec
}

func TestCreateTalkTrimsAndStores(t *testing.T) {
	store := &fakeTalks{}
	rec := serveTalks(t, store, http.MethodPost, "/api/talks", `{"title":"  Guitar demo ","speaker":" Aaron ","description":""}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	if len(store.created) != 1 || store.created[0] != (search.NewTalk{Title: "Guitar demo", Speaker: "Aaron"}) {
		t.Errorf("stored %+v", store.created)
	}
	var talk search.Talk
	if err := json.Unmarshal(rec.Body.Bytes(), &talk); err != nil || talk.ID != "guitar-demo" {
		t.Errorf("response %s, err %v", rec.Body, err)
	}
}

func TestCreateTalkRejectsInvalidInput(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":         `title=Guitar`,
		"no title":         `{"title":"  ","speaker":"Aaron"}`,
		"no speaker":       `{"title":"Guitar demo"}`,
		"long title":       `{"title":"` + strings.Repeat("é", maxTitleRunes+1) + `","speaker":"Aaron"}`,
		"long description": `{"title":"Guitar demo","speaker":"Aaron","description":"` + strings.Repeat("a", maxDescriptionRunes+1) + `"}`,
	} {
		store := &fakeTalks{}
		if rec := serveTalks(t, store, http.MethodPost, "/api/talks", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, rec.Code)
		}
		if len(store.created) != 0 {
			t.Errorf("%s: stored %+v", name, store.created)
		}
	}
}

func TestCreateTalkAcceptsLongestFormInput(t *testing.T) {
	body := `{"title":"` + strings.Repeat("é", maxTitleRunes) + `","speaker":"` + strings.Repeat("é", maxSpeakerRunes) +
		`","description":"` + strings.Repeat("é", maxDescriptionRunes) + `"}`
	if rec := serveTalks(t, &fakeTalks{}, http.MethodPost, "/api/talks", body); rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
}

func TestGetTalk(t *testing.T) {
	store := &fakeTalks{talks: map[string]search.Talk{"guitar-demo": {ID: "guitar-demo"}}}
	for target, want := range map[string]int{
		"/api/talks/guitar-demo": http.StatusOK,
		"/api/talks/missing":     http.StatusNotFound,
		"/api/talks/Not%20Valid": http.StatusNotFound,
	} {
		if rec := serveTalks(t, store, http.MethodGet, target, ""); rec.Code != want {
			t.Errorf("GET %s: got %d, want %d", target, rec.Code, want)
		}
	}
}

func TestEndTalk(t *testing.T) {
	store := &fakeTalks{talks: map[string]search.Talk{"guitar-demo": {ID: "guitar-demo", Status: search.TalkLive}}}
	var ended []string
	rec := serveEnd(t, store, captionCounts{"guitar-demo": 3}, &ended, http.MethodPost, "/api/talks/guitar-demo/end", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	var talk search.Talk
	if err := json.Unmarshal(rec.Body.Bytes(), &talk); err != nil || talk.Status != search.TalkFinished || talk.EndedAt == nil {
		t.Errorf("response %s, err %v", rec.Body, err)
	}
	if len(ended) != 1 || ended[0] != "guitar-demo" {
		t.Errorf("notes started for %v, want [guitar-demo]", ended)
	}
	for target, want := range map[string]int{
		"/api/talks/missing/end":     http.StatusNotFound,
		"/api/talks/Not%20Valid/end": http.StatusNotFound,
	} {
		if rec := serveTalks(t, store, http.MethodPost, target, ""); rec.Code != want {
			t.Errorf("POST %s: got %d, want %d", target, rec.Code, want)
		}
	}
}

func TestATalkWithoutCaptionsStaysLive(t *testing.T) {
	store := &fakeTalks{talks: map[string]search.Talk{"guitar-demo": {ID: "guitar-demo", Status: search.TalkLive}}}
	var ended []string
	rec := serveEnd(t, store, captionCounts{}, &ended, http.MethodPost, "/api/talks/guitar-demo/end", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
	}
	if store.talks["guitar-demo"].Status != search.TalkLive || len(ended) != 0 {
		t.Errorf("talk %+v ended, notes started for %v", store.talks["guitar-demo"], ended)
	}
}

func TestListTalksIsAnArrayWhenEmpty(t *testing.T) {
	rec := serveTalks(t, &fakeTalks{}, http.MethodGet, "/api/talks", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("got %d %q, want 200 []", rec.Code, rec.Body)
	}
}

func TestSearchTalks(t *testing.T) {
	store := &fakeTalks{}
	if rec := serveTalks(t, store, http.MethodGet, "/api/search?q=%20%20", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("blank query: got %d %q", rec.Code, rec.Body)
	}
	if rec := serveTalks(t, store, http.MethodGet, "/api/search?q="+strings.Repeat("a", maxQueryRunes+1), ""); rec.Code != http.StatusBadRequest {
		t.Errorf("long query: got %d, want 400", rec.Code)
	}
	if rec := serveTalks(t, store, http.MethodGet, "/api/search?q=+guitar+", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("no hits: got %d %q", rec.Code, rec.Body)
	}
	if len(store.queries) != 1 || store.queries[0] != "guitar" {
		t.Errorf("searched %q", store.queries)
	}
}

func TestTalkStoreFailureIsBadGateway(t *testing.T) {
	store := &fakeTalks{err: errors.New("elasticsearch down")}
	for _, req := range [][3]string{
		{http.MethodGet, "/api/talks", ""},
		{http.MethodPost, "/api/talks", `{"title":"Guitar demo","speaker":"Aaron"}`},
		{http.MethodGet, "/api/search?q=guitar", ""},
		{http.MethodPost, "/api/talks/guitar-demo/end", ""},
	} {
		if rec := serveTalks(t, store, req[0], req[1], req[2]); rec.Code != http.StatusBadGateway {
			t.Errorf("%s %s: got %d, want 502", req[0], req[1], rec.Code)
		}
	}
}
