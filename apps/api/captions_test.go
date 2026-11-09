package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
)

type fakeHistory struct {
	lines []captions.Caption
	err   error
}

func (f fakeHistory) History(context.Context, string) ([]captions.Caption, error) {
	return f.lines, f.err
}

func getHistory(t *testing.T, history captionHistory, talkID string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/talks/{talkID}/captions", handleCaptionHistory(history))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/talks/"+talkID+"/captions", nil))
	return rec
}

func TestCaptionHistory(t *testing.T) {
	line := captions.Caption{TalkID: "demo", ID: "s1-item_1", At: "2026-11-09T10:00:03.000Z", Text: "Hello.", Final: true, Corrections: []captions.Correction{}}
	rec := getHistory(t, fakeHistory{lines: []captions.Caption{line}}, "demo")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	var got []captions.Caption
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != line.ID {
		t.Fatalf("got %+v", got)
	}
}

func TestCaptionHistoryErrors(t *testing.T) {
	if rec := getHistory(t, fakeHistory{}, "Bad_ID"); rec.Code != http.StatusNotFound {
		t.Errorf("invalid talk id: got %d", rec.Code)
	}
	if rec := getHistory(t, fakeHistory{err: errors.New("down")}, "demo"); rec.Code != http.StatusBadGateway {
		t.Errorf("store failure: got %d", rec.Code)
	}
}
