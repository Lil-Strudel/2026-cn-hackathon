package main

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf16"

	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/transcribe"
)

const (
	instrumentationName = "github.com/Lil-Strudel/2026-cn-hackathon/apps/api"
	maxAudioFrame       = 64 << 10
	drainTimeout        = 5 * time.Second
)

type corrector interface {
	Correct(ctx context.Context, text string) (search.Correction, error)
}

type publishFunc func(ctx context.Context, c captions.Caption) error

func redisPublisher(rdb *redis.Client) publishFunc {
	return func(ctx context.Context, c captions.Caption) error {
		data, err := captions.Encode(ctx, c)
		if err != nil {
			return err
		}
		return rdb.Publish(ctx, captions.Channel(c.TalkID), data).Err()
	}
}

// audioHandler takes a speaker's microphone over a WebSocket, transcribes it
// and publishes the captions. Binary frames carry mono PCM16 at
// transcribe.SampleRate. The speaker watches captions over SSE like everyone
// else, so nothing is sent back on the socket.
type audioHandler struct {
	shutdown    context.Context
	transcriber transcribe.Transcriber
	corrector   corrector
	publish     publishFunc

	tracer     trace.Tracer
	sessions   metric.Int64UpDownCounter
	audioBytes metric.Int64Counter
	published  metric.Int64Counter
	latency    metric.Float64Histogram
}

func newAudioHandler(shutdown context.Context, t transcribe.Transcriber, c corrector, publish publishFunc) (*audioHandler, error) {
	meter := otel.Meter(instrumentationName)
	sessions, err := meter.Int64UpDownCounter("captions.sessions.active",
		metric.WithDescription("Speakers currently streaming audio"), metric.WithUnit("{session}"))
	if err != nil {
		return nil, err
	}
	audioBytes, err := meter.Int64Counter("captions.audio.received",
		metric.WithDescription("PCM audio received from speakers"), metric.WithUnit("By"))
	if err != nil {
		return nil, err
	}
	published, err := meter.Int64Counter("captions.published",
		metric.WithDescription("Caption updates published, partial and final"), metric.WithUnit("{caption}"))
	if err != nil {
		return nil, err
	}
	latency, err := meter.Float64Histogram("captions.transcription.latency",
		metric.WithDescription("Time from the end of a line's speech to its final caption"), metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.25, 0.5, 0.75, 1, 1.5, 2, 3, 5, 10))
	if err != nil {
		return nil, err
	}
	return &audioHandler{
		shutdown:    shutdown,
		transcriber: t,
		corrector:   c,
		publish:     publish,
		tracer:      otel.Tracer(instrumentationName),
		sessions:    sessions,
		audioBytes:  audioBytes,
		published:   published,
		latency:     latency,
	}, nil
}

func (a *audioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	talkID := r.PathValue("talkID")
	if !captions.ValidTalkID(talkID) {
		http.Error(w, "invalid talk id", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(maxAudioFrame)

	// Segment IDs restart with every transcription session, so line IDs are
	// prefixed with one per speaker connection.
	sessionID := rand.Text()
	ctx, span := a.tracer.Start(r.Context(), "caption.session", trace.WithAttributes(
		attribute.String("talk.id", talkID),
		attribute.String("caption.session_id", sessionID),
	))
	defer span.End()
	a.sessions.Add(ctx, 1)
	defer a.sessions.Add(context.WithoutCancel(ctx), -1)
	slog.InfoContext(ctx, "speaker connected", "talk_id", talkID, "session_id", sessionID)

	session, err := a.transcriber.Start(ctx, transcribe.Options{Language: "en"})
	if err != nil {
		slog.ErrorContext(ctx, "start transcription", "talk_id", talkID, "err", err)
		span.SetStatus(codes.Error, "start transcription")
		_ = conn.Close(websocket.StatusInternalError, "transcription unavailable")
		return
	}

	published := make(chan struct{})
	go func() {
		defer close(published)
		started := map[string]time.Time{}
		for event := range session.Events() {
			at, ok := started[event.SegmentID]
			if !ok {
				at = time.Now()
				started[event.SegmentID] = at
			}
			if event.Final {
				delete(started, event.SegmentID)
			}
			a.publishEvent(ctx, captions.Caption{
				TalkID: talkID,
				ID:     sessionID + "-" + event.SegmentID,
				At:     captions.FormatTime(at),
				Text:   event.Text,
				Final:  event.Final,
			}, event.CommittedAt)
		}
	}()

	status, reason := a.relayAudio(ctx, conn, session)

	drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), drainTimeout)
	defer cancel()
	if err := session.Close(drainCtx); err != nil {
		slog.WarnContext(ctx, "close transcription", "talk_id", talkID, "err", err)
	}
	<-published
	_ = conn.Close(status, reason)
	slog.InfoContext(ctx, "speaker disconnected", "talk_id", talkID, "session_id", sessionID)
}

// relayAudio forwards audio frames until the speaker hangs up or the server
// shuts down, and returns how to close the socket.
func (a *audioHandler) relayAudio(ctx context.Context, conn *websocket.Conn, session transcribe.Session) (websocket.StatusCode, string) {
	// Cancelling a Read drops the connection without a close frame, so on
	// shutdown close the socket instead and let the Read fail.
	stop := context.AfterFunc(a.shutdown, func() {
		_ = conn.Close(websocket.StatusGoingAway, "server shutting down")
	})
	defer stop()

	for {
		kind, data, err := conn.Read(ctx)
		if err != nil {
			if a.shutdown.Err() != nil {
				return websocket.StatusGoingAway, "server shutting down"
			}
			if s := websocket.CloseStatus(err); s != websocket.StatusNormalClosure && s != websocket.StatusGoingAway {
				slog.WarnContext(ctx, "speaker connection lost", "err", err)
			}
			return websocket.StatusNormalClosure, ""
		}
		if kind != websocket.MessageBinary || len(data)%2 != 0 {
			return websocket.StatusUnsupportedData, "expected PCM16 binary frames"
		}
		a.audioBytes.Add(ctx, int64(len(data)))
		if err := session.Append(ctx, data); err != nil {
			slog.ErrorContext(ctx, "append audio", "err", err)
			return websocket.StatusInternalError, "transcription failed"
		}
	}
}

func (a *audioHandler) publishEvent(ctx context.Context, c captions.Caption, committedAt time.Time) {
	ctx, span := a.tracer.Start(ctx, "caption.publish", trace.WithSpanKind(trace.SpanKindProducer), trace.WithAttributes(
		attribute.String("talk.id", c.TalkID),
		attribute.String("caption.id", c.ID),
		attribute.Bool("caption.final", c.Final),
	))
	defer span.End()

	c.Corrections = []captions.Correction{}
	if c.Final {
		c.Text, c.Corrections = a.correct(ctx, c.Text)
		if !committedAt.IsZero() {
			a.latency.Record(ctx, time.Since(committedAt).Seconds())
		}
	}

	if err := a.publish(ctx, c); err != nil {
		slog.ErrorContext(ctx, "publish caption", "talk_id", c.TalkID, "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish caption")
		return
	}
	a.published.Add(ctx, 1, metric.WithAttributes(attribute.Bool("caption.final", c.Final)))
}

// correct runs only on final lines: each correction costs Elasticsearch
// queries, too many to spend on every partial. It falls back to the text as
// heard, because a caption with a misheard word beats a missing caption.
func (a *audioHandler) correct(ctx context.Context, text string) (string, []captions.Correction) {
	correction, err := a.corrector.Correct(ctx, text)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.WarnContext(ctx, "correct caption", "err", err)
		}
		return text, []captions.Correction{}
	}
	return correction.Text, captionCorrections(correction.Segments)
}

func captionCorrections(segments []search.Segment) []captions.Correction {
	out := []captions.Correction{}
	offset := 0
	for _, s := range segments {
		length := len(utf16.Encode([]rune(s.Text)))
		if s.Heard != "" {
			out = append(out, captions.Correction{Start: offset, End: offset + length, Heard: s.Heard})
		}
		offset += length
	}
	return out
}
