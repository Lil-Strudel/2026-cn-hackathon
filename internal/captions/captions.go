package captions

import (
	"context"
	"encoding/json"
	"regexp"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Caption is one line of a talk's transcript, in the shape the web app's
// CaptionLine expects. A line is published repeatedly while it is spoken:
// each partial carries the whole text so far under the same ID, and the last
// one has Final set.
type Caption struct {
	TalkID string `json:"talkId"`
	ID     string `json:"id"`
	// At is when the line was said, formatted by FormatTime.
	At          string       `json:"at"`
	Text        string       `json:"text"`
	Final       bool         `json:"final"`
	Corrections []Correction `json:"corrections"`
}

// Correction marks a term the vocabulary corrected. Start and End are UTF-16
// offsets into Text, because that is how JavaScript indexes strings.
type Correction struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Heard string `json:"heard"`
}

// FormatTime matches JavaScript's toISOString. Clients order lines by
// comparing At as text, which RFC 3339's trimmed fractions would break.
func FormatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

const channelPrefix = "captions:"

// ChannelPattern matches the Redis channel of every talk.
const ChannelPattern = channelPrefix + "*"

var talkIDPattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

func ValidTalkID(id string) bool {
	return talkIDPattern.MatchString(id)
}

// Channel is the Redis pub/sub channel a talk's captions are published on.
func Channel(talkID string) string {
	return channelPrefix + talkID
}

type envelope struct {
	Caption Caption           `json:"caption"`
	Trace   map[string]string `json:"trace,omitempty"`
}

// Encode wraps c for Redis together with the trace context in ctx, so the
// fanout span that delivers it joins the trace of the api span that published it.
func Encode(ctx context.Context, c Caption) ([]byte, error) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return json.Marshal(envelope{Caption: c, Trace: carrier})
}

// Decode reverses Encode, returning a context that carries the publisher's trace context.
func Decode(ctx context.Context, data []byte) (context.Context, Caption, error) {
	var e envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return ctx, Caption{}, err
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(e.Trace)), e.Caption, nil
}
