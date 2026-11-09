package transcribe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	openAIRealtimeURL = "wss://api.openai.com/v1/realtime?intent=transcription"
	openAIModel       = "gpt-4o-transcribe"
)

// OpenAI transcribes with a Realtime transcription session. Its server-side
// voice activity detection decides where segments end.
type OpenAI struct {
	APIKey string
	URL    string
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{APIKey: apiKey, URL: openAIRealtimeURL}
}

func (o *OpenAI) Start(ctx context.Context, opts Options) (Session, error) {
	conn, res, err := websocket.Dial(ctx, o.URL, &websocket.DialOptions{
		HTTPClient: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		HTTPHeader: http.Header{"Authorization": {"Bearer " + o.APIKey}},
	})
	if res != nil && res.Body != nil {
		_ = res.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("dial openai realtime: %w", err)
	}
	conn.SetReadLimit(1 << 20)

	s := &openAISession{
		conn:    conn,
		events:  make(chan Event, 64),
		done:    make(chan struct{}),
		drained: make(chan struct{}),
		items:   map[string]*segment{},
	}
	if err := s.send(ctx, sessionUpdate(opts)); err != nil {
		_ = conn.CloseNow()
		return nil, fmt.Errorf("configure openai session: %w", err)
	}
	go s.read(context.WithoutCancel(ctx))
	return s, nil
}

func sessionUpdate(opts Options) map[string]any {
	transcription := map[string]any{"model": openAIModel}
	if opts.Language != "" {
		transcription["language"] = opts.Language
	}
	if opts.Prompt != "" {
		transcription["prompt"] = opts.Prompt
	}
	return map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"type": "transcription",
			"audio": map[string]any{
				"input": map[string]any{
					"format":          map[string]any{"type": "audio/pcm", "rate": SampleRate},
					"noise_reduction": map[string]any{"type": "near_field"},
					"transcription":   transcription,
					"turn_detection": map[string]any{
						"type":                "server_vad",
						"threshold":           0.5,
						"prefix_padding_ms":   300,
						"silence_duration_ms": 500,
					},
				},
			},
		},
	}
}

type segment struct {
	text        strings.Builder
	committedAt time.Time
}

type openAISession struct {
	conn    *websocket.Conn
	events  chan Event
	done    chan struct{}
	drained chan struct{}

	closing    atomic.Bool
	drainOnce  sync.Once
	items      map[string]*segment
	commitSeen bool
}

type serverEvent struct {
	Type       string `json:"type"`
	ItemID     string `json:"item_id"`
	Delta      string `json:"delta"`
	Transcript string `json:"transcript"`
	Error      *struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *openAISession) Events() <-chan Event { return s.events }

func (s *openAISession) Append(ctx context.Context, pcm []byte) error {
	return s.send(ctx, map[string]string{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(pcm),
	})
}

func (s *openAISession) Close(ctx context.Context) error {
	s.closing.Store(true)
	if err := s.send(ctx, map[string]string{"type": "input_audio_buffer.commit"}); err == nil {
		select {
		case <-s.drained:
		case <-s.done:
		case <-ctx.Done():
		}
	}
	err := s.conn.Close(websocket.StatusNormalClosure, "")
	<-s.done
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (s *openAISession) send(ctx context.Context, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

func (s *openAISession) read(ctx context.Context) {
	defer close(s.done)
	defer close(s.events)
	for {
		_, data, err := s.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure && !s.closing.Load() {
				slog.WarnContext(ctx, "openai realtime connection ended", "err", err)
			}
			return
		}
		var event serverEvent
		if err := json.Unmarshal(data, &event); err != nil {
			slog.WarnContext(ctx, "undecodable openai event", "err", err)
			continue
		}
		s.handle(ctx, event)
		if s.closing.Load() && s.commitSeen && len(s.items) == 0 {
			s.drainOnce.Do(func() { close(s.drained) })
		}
	}
}

func (s *openAISession) segment(itemID string) *segment {
	seg, ok := s.items[itemID]
	if !ok {
		seg = &segment{}
		s.items[itemID] = seg
	}
	return seg
}

func (s *openAISession) handle(ctx context.Context, event serverEvent) {
	switch event.Type {
	case "input_audio_buffer.committed":
		s.segment(event.ItemID).committedAt = time.Now()
		if s.closing.Load() {
			s.commitSeen = true
		}
	case "conversation.item.input_audio_transcription.delta":
		seg := s.segment(event.ItemID)
		seg.text.WriteString(event.Delta)
		s.events <- Event{SegmentID: event.ItemID, Text: seg.text.String(), CommittedAt: seg.committedAt}
	case "conversation.item.input_audio_transcription.completed":
		seg := s.segment(event.ItemID)
		delete(s.items, event.ItemID)
		s.events <- Event{SegmentID: event.ItemID, Text: event.Transcript, Final: true, CommittedAt: seg.committedAt}
	case "conversation.item.input_audio_transcription.failed":
		seg := s.segment(event.ItemID)
		delete(s.items, event.ItemID)
		slog.WarnContext(ctx, "openai transcription failed", "item_id", event.ItemID, "error", event.Error)
		if seg.text.Len() > 0 {
			s.events <- Event{SegmentID: event.ItemID, Text: seg.text.String(), Final: true, CommittedAt: seg.committedAt}
		}
	case "error":
		// The commit sent by Close is rejected when the buffer holds no new
		// speech, which is the normal way a session ends.
		if s.closing.Load() {
			s.commitSeen = true
			return
		}
		slog.WarnContext(ctx, "openai realtime error", "error", event.Error)
	}
}
