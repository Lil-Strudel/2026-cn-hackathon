package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/httpserver"
)

type captionHistory interface {
	History(ctx context.Context, talkID string) ([]captions.Caption, error)
}

// handleCaptionHistory serves a talk's final lines so far, so a viewer who
// joins late or reloads sees what was already said.
func handleCaptionHistory(history captionHistory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		talkID := r.PathValue("talkID")
		if !captions.ValidTalkID(talkID) {
			http.Error(w, "invalid talk id", http.StatusNotFound)
			return
		}
		lines, err := history.History(r.Context(), talkID)
		if err != nil {
			slog.ErrorContext(r.Context(), "caption history", "talk_id", talkID, "err", err)
			http.Error(w, "caption history unavailable", http.StatusBadGateway)
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, lines)
	}
}
