package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"go.opentelemetry.io/otel/codes"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/transcribe"
)

type transcript struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Final bool   `json:"final"`
}

// ServeTranscription transcribes a microphone like ServeHTTP, but sends the
// text back on the socket instead of publishing and storing it, so anyone can
// try transcription and correction without captioning a talk. Correction is
// left to POST /api/corrections, which the page calls on what was heard.
func (a *audioHandler) ServeTranscription(w http.ResponseWriter, r *http.Request) {
	a.active.Add(1)
	defer a.active.Done()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(maxAudioFrame)

	ctx := context.WithoutCancel(r.Context())
	ctx, span := a.tracer.Start(ctx, "transcription.session")
	defer span.End()

	session, err := a.transcriber.Start(ctx, transcribe.Options{Language: "en"})
	if err != nil {
		slog.ErrorContext(ctx, "start transcription", "err", err)
		span.SetStatus(codes.Error, "start transcription")
		_ = conn.Close(websocket.StatusInternalError, "transcription unavailable")
		return
	}

	sent := make(chan struct{})
	go func() {
		defer close(sent)
		for event := range session.Events() {
			data, err := json.Marshal(transcript{ID: event.SegmentID, Text: event.Text, Final: event.Final})
			if err != nil {
				continue
			}
			// Events are drained even once writing fails, so Close isn't blocked.
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				span.RecordError(err)
			}
		}
	}()

	status, reason := a.relayAudio(ctx, conn, session)

	drainCtx, cancel := context.WithTimeout(ctx, drainTimeout)
	defer cancel()
	if err := session.Close(drainCtx); err != nil {
		slog.WarnContext(ctx, "close transcription", "err", err)
	}
	<-sent
	_ = conn.Close(status, reason)
}
