package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/httpserver"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
)

// A caption chunk is a few seconds of speech; anything far longer is misuse,
// and every word costs up to three vocabulary queries.
const (
	maxCorrectionBytes = 2000
	maxNewTerms        = 50
	maxTermBytes       = 64
)

func handleCorrect(vocab *search.Vocabulary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*maxCorrectionBytes)).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if len(req.Text) > maxCorrectionBytes {
			http.Error(w, "text is too long", http.StatusRequestEntityTooLarge)
			return
		}
		correction, err := vocab.Correct(r.Context(), req.Text)
		if err != nil {
			slog.ErrorContext(r.Context(), "correct text", "err", err)
			http.Error(w, "correction failed", http.StatusBadGateway)
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, correction)
	}
}

func handleAddTerms(vocab *search.Vocabulary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Terms []string `json:"terms"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxNewTerms*maxTermBytes*2)).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		var terms []string
		for _, t := range req.Terms {
			if t = strings.TrimSpace(t); t != "" {
				terms = append(terms, t)
			}
		}
		if len(terms) == 0 || len(terms) > maxNewTerms {
			http.Error(w, "send between 1 and 50 terms", http.StatusBadRequest)
			return
		}
		for _, t := range terms {
			if len(t) > maxTermBytes {
				http.Error(w, "a term is too long", http.StatusBadRequest)
				return
			}
		}
		if err := vocab.Add(r.Context(), terms); err != nil {
			slog.ErrorContext(r.Context(), "add terms", "err", err)
			http.Error(w, "adding terms failed", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
