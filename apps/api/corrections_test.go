package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v9"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
)

func TestRejectsBadRequests(t *testing.T) {
	es, err := elasticsearch.NewTyped(elasticsearch.WithAddresses("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	vocab, err := search.NewVocabulary(es)
	if err != nil {
		t.Fatal(err)
	}
	long := `"` + strings.Repeat("a", maxCorrectionBytes+1) + `"`

	tests := []struct {
		name    string
		handler http.HandlerFunc
		body    string
		want    int
	}{
		{"correction with invalid JSON", handleCorrect(vocab), "{", http.StatusBadRequest},
		{"correction that is too long", handleCorrect(vocab), `{"text":` + long + `}`, http.StatusRequestEntityTooLarge},
		{"no terms", handleAddTerms(vocab), `{"terms":[]}`, http.StatusBadRequest},
		{"only blank terms", handleAddTerms(vocab), `{"terms":["  "]}`, http.StatusBadRequest},
		{"term that is too long", handleAddTerms(vocab), `{"terms":["` + strings.Repeat("a", maxTermBytes+1) + `"]}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
