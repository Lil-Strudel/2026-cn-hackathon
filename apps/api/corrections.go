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
	maxExceptWords     = 20
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
			Terms []search.Term `json:"terms"`
		}
		maxBody := maxNewTerms * (maxExceptWords + 1) * maxTermBytes * 2
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(maxBody))).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		var terms []search.Term
		for _, t := range req.Terms {
			if t, ok := cleanTerm(t); ok {
				terms = append(terms, t)
			}
		}
		if len(terms) == 0 || len(terms) > maxNewTerms {
			http.Error(w, "send between 1 and 50 terms", http.StatusBadRequest)
			return
		}
		for _, t := range terms {
			if len(t.Except) > maxExceptWords {
				http.Error(w, "a term has more than 20 exception words", http.StatusBadRequest)
				return
			}
			for _, word := range append([]string{t.Term}, t.Except...) {
				if len(word) > maxTermBytes {
					http.Error(w, "a term or exception word is too long", http.StatusBadRequest)
					return
				}
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

func handleListTerms(vocab *search.Vocabulary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		terms, err := vocab.List(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "list terms", "err", err)
			http.Error(w, "listing terms failed", http.StatusBadGateway)
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, map[string]any{"terms": terms})
	}
}

func handleRemoveTerm(vocab *search.Vocabulary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term := strings.TrimSpace(r.PathValue("term"))
		if term == "" || len(term) > maxTermBytes {
			http.Error(w, "a term is between 1 and 64 bytes", http.StatusBadRequest)
			return
		}
		if err := vocab.Remove(r.Context(), term); err != nil {
			slog.ErrorContext(r.Context(), "remove term", "err", err)
			http.Error(w, "removing the term failed", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func cleanTerm(t search.Term) (search.Term, bool) {
	clean := search.Term{Term: strings.TrimSpace(t.Term)}
	for _, word := range t.Except {
		if word = strings.TrimSpace(word); word != "" {
			clean.Except = append(clean.Except, word)
		}
	}
	return clean, clean.Term != ""
}
