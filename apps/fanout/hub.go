package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
)

const (
	instrumentationName = "github.com/Lil-Strudel/2026-cn-hackathon/apps/fanout"
	clientBuffer        = 64
	pingInterval        = 15 * time.Second
)

type client struct {
	talkID string
	// messages is closed by the hub when it drops the client for falling behind.
	messages chan []byte
}

// hub delivers captions from Redis to every connected client of a talk. Each
// pod holds a single pattern subscription and fans out in memory, so Redis
// load does not grow with the number of viewers.
type hub struct {
	mu    sync.Mutex
	talks map[string]map[*client]struct{}

	tracer    trace.Tracer
	clients   metric.Int64UpDownCounter
	delivered metric.Int64Counter
	dropped   metric.Int64Counter
}

func newHub() (*hub, error) {
	meter := otel.Meter(instrumentationName)
	clients, err := meter.Int64UpDownCounter("fanout.clients",
		metric.WithDescription("Clients connected to a caption stream"), metric.WithUnit("{client}"))
	if err != nil {
		return nil, err
	}
	delivered, err := meter.Int64Counter("fanout.captions.delivered",
		metric.WithDescription("Captions queued for delivery to a client"), metric.WithUnit("{caption}"))
	if err != nil {
		return nil, err
	}
	dropped, err := meter.Int64Counter("fanout.clients.dropped",
		metric.WithDescription("Clients disconnected for falling behind"), metric.WithUnit("{client}"))
	if err != nil {
		return nil, err
	}
	return &hub{
		talks:     map[string]map[*client]struct{}{},
		tracer:    otel.Tracer(instrumentationName),
		clients:   clients,
		delivered: delivered,
		dropped:   dropped,
	}, nil
}

func (h *hub) subscribe(ctx context.Context, talkID string) *client {
	c := &client{talkID: talkID, messages: make(chan []byte, clientBuffer)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.talks[talkID] == nil {
		h.talks[talkID] = map[*client]struct{}{}
	}
	h.talks[talkID][c] = struct{}{}
	h.clients.Add(ctx, 1)
	return c
}

func (h *hub) unsubscribe(ctx context.Context, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.remove(ctx, c)
}

// remove reports whether c was still subscribed. The caller holds h.mu.
func (h *hub) remove(ctx context.Context, c *client) bool {
	talk := h.talks[c.talkID]
	if _, ok := talk[c]; !ok {
		return false
	}
	delete(talk, c)
	if len(talk) == 0 {
		delete(h.talks, c.talkID)
	}
	h.clients.Add(ctx, -1)
	return true
}

// dispatch queues one Redis message for every client of its talk. A client
// whose buffer is full is dropped rather than allowed to stall the others.
func (h *hub) dispatch(ctx context.Context, payload []byte) {
	ctx, caption, err := captions.Decode(ctx, payload)
	if err != nil {
		slog.WarnContext(ctx, "undecodable caption message", "err", err)
		return
	}
	ctx, span := h.tracer.Start(ctx, "caption.fanout",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("talk.id", caption.TalkID),
			attribute.String("caption.id", caption.ID),
			attribute.Bool("caption.final", caption.Final),
		))
	defer span.End()

	data, err := json.Marshal(caption)
	if err != nil {
		slog.ErrorContext(ctx, "encode caption", "err", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	var delivered, dropped int64
	for c := range h.talks[caption.TalkID] {
		select {
		case c.messages <- data:
			delivered++
		default:
			h.remove(ctx, c)
			close(c.messages)
			dropped++
		}
	}
	span.SetAttributes(attribute.Int64("fanout.delivered", delivered), attribute.Int64("fanout.dropped", dropped))
	h.delivered.Add(ctx, delivered)
	if dropped > 0 {
		h.dropped.Add(ctx, dropped)
		slog.WarnContext(ctx, "dropped slow clients", "talk_id", caption.TalkID, "count", dropped)
	}
}

// run dispatches Redis messages until ctx is cancelled. go-redis reconnects
// and resubscribes on its own when the connection drops.
func (h *hub) run(ctx context.Context, rdb *redis.Client) error {
	pubsub := rdb.PSubscribe(ctx, captions.ChannelPattern)
	defer func() { _ = pubsub.Close() }()
	messages := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-messages:
			if !ok {
				return fmt.Errorf("redis subscription closed")
			}
			h.dispatch(ctx, []byte(msg.Payload))
		}
	}
}

// serveEvents streams a talk's captions as Server-Sent Events. Streams also
// end when shutdown is cancelled, because the server only cancels request
// contexts after draining, which an open stream would otherwise block.
func (h *hub) serveEvents(shutdown context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		talkID := r.PathValue("talkID")
		if !captions.ValidTalkID(talkID) {
			http.Error(w, "invalid talk id", http.StatusNotFound)
			return
		}

		ctx := r.Context()
		c := h.subscribe(ctx, talkID)
		defer h.unsubscribe(context.WithoutCancel(ctx), c)

		rc := http.NewResponseController(w)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		write := func(format string, args ...any) error {
			if _, err := fmt.Fprintf(w, format, args...); err != nil {
				return err
			}
			return rc.Flush()
		}

		if err := write("retry: 2000\n: connected\n\n"); err != nil {
			return
		}

		ping := time.NewTicker(pingInterval)
		defer ping.Stop()
		for {
			var err error
			select {
			case <-ctx.Done():
				return
			case <-shutdown.Done():
				return
			case data, ok := <-c.messages:
				if !ok {
					return
				}
				err = write("event: caption\ndata: %s\n\n", data)
			case <-ping.C:
				err = write(": ping\n\n")
			}
			if err != nil {
				return
			}
		}
	}
}
