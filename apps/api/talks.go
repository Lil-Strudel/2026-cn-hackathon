package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/httpserver"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
)

// The same limits as the new talk form.
const (
	maxTitleRunes       = 120
	maxSpeakerRunes     = 80
	maxDescriptionRunes = 1000
	maxNewTalkBytes     = 16 << 10
	maxQueryRunes       = 200
)

type talkStore interface {
	List(ctx context.Context) ([]search.Talk, error)
	Get(ctx context.Context, id string) (*search.Talk, error)
	Create(ctx context.Context, n search.NewTalk) (search.Talk, error)
	Search(ctx context.Context, query string) ([]search.SearchHit, error)
}

func handleListTalks(talks talkStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := talks.List(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "list talks", "err", err)
			http.Error(w, "talks unavailable", http.StatusBadGateway)
			return
		}
		if list == nil {
			list = []search.Talk{}
		}
		httpserver.WriteJSON(w, http.StatusOK, list)
	}
}

func handleGetTalk(talks talkStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		talkID := r.PathValue("talkID")
		if !captions.ValidTalkID(talkID) {
			http.Error(w, "talk not found", http.StatusNotFound)
			return
		}
		talk, err := talks.Get(r.Context(), talkID)
		if err != nil {
			slog.ErrorContext(r.Context(), "get talk", "talk_id", talkID, "err", err)
			http.Error(w, "talk unavailable", http.StatusBadGateway)
			return
		}
		if talk == nil {
			http.Error(w, "talk not found", http.StatusNotFound)
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, talk)
	}
}

func handleCreateTalk(talks talkStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var n search.NewTalk
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxNewTalkBytes)).Decode(&n); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		n.Title, n.Speaker, n.Description = strings.TrimSpace(n.Title), strings.TrimSpace(n.Speaker), strings.TrimSpace(n.Description)
		if msg := invalidTalk(n); msg != "" {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		talk, err := talks.Create(r.Context(), n)
		if err != nil {
			slog.ErrorContext(r.Context(), "create talk", "err", err)
			http.Error(w, "creating the talk failed", http.StatusBadGateway)
			return
		}
		slog.InfoContext(r.Context(), "talk created", "talk_id", talk.ID)
		httpserver.WriteJSON(w, http.StatusCreated, talk)
	}
}

func invalidTalk(n search.NewTalk) string {
	switch {
	case n.Title == "" || utf8.RuneCountInString(n.Title) > maxTitleRunes:
		return "title must be 1 to 120 characters"
	case n.Speaker == "" || utf8.RuneCountInString(n.Speaker) > maxSpeakerRunes:
		return "speaker must be 1 to 80 characters"
	case utf8.RuneCountInString(n.Description) > maxDescriptionRunes:
		return "description must be at most 1000 characters"
	}
	return ""
}

func handleSearchTalks(talks talkStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			httpserver.WriteJSON(w, http.StatusOK, []search.SearchHit{})
			return
		}
		if utf8.RuneCountInString(query) > maxQueryRunes {
			http.Error(w, "query must be at most 200 characters", http.StatusBadRequest)
			return
		}
		hits, err := talks.Search(r.Context(), query)
		if err != nil {
			slog.ErrorContext(r.Context(), "search talks", "err", err)
			http.Error(w, "search unavailable", http.StatusBadGateway)
			return
		}
		if hits == nil {
			hits = []search.SearchHit{}
		}
		httpserver.WriteJSON(w, http.StatusOK, hits)
	}
}
