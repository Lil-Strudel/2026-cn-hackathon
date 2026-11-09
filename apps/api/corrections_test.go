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
		{"only blank terms", handleAddTerms(vocab), `{"terms":[{"term":"  ","context":["Sonar"]}]}`, http.StatusBadRequest},
		{"term that is too long", handleAddTerms(vocab), `{"terms":[{"term":"` + long[1:maxTermBytes+2] + `"}]}`, http.StatusBadRequest},
		{"context word that is too long", handleAddTerms(vocab), `{"terms":[{"term":"Gitar","context":["` + long[1:maxTermBytes+2] + `"]}]}`, http.StatusBadRequest},
		{"too many context words", handleAddTerms(vocab), `{"terms":[{"term":"Gitar","context":[` + strings.Repeat(`"Sonar",`, maxContextWords) + `"CI"]}]}`, http.StatusBadRequest},
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

func TestRejectsBadTermsToRemove(t *testing.T) {
	es, err := elasticsearch.NewTyped(elasticsearch.WithAddresses("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	vocab, err := search.NewVocabulary(es)
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{" ", strings.Repeat("a", maxTermBytes+1)} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/", nil)
		req.SetPathValue("term", term)
		handleRemoveTerm(vocab)(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("removing %q: status = %d, want %d", term, rec.Code, http.StatusBadRequest)
		}
	}
}
