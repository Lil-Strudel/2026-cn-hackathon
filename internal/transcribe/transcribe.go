package transcribe

import (
	"context"
	"time"
)

// SampleRate is the rate of the mono PCM16 little-endian audio a Session accepts.
const SampleRate = 24000

// Event is an update to one segment of speech. Text is the whole transcript
// of the segment so far; the last event of a segment has Final set.
type Event struct {
	SegmentID string
	Text      string
	Final     bool
	// CommittedAt is when the segment's audio was closed off, zero if unknown.
	CommittedAt time.Time
}

type Options struct {
	Language string
	Prompt   string
}

type Transcriber interface {
	Start(ctx context.Context, opts Options) (Session, error)
}

type Session interface {
	Append(ctx context.Context, pcm []byte) error
	// Events is closed once the session has ended.
	Events() <-chan Event
	// Close flushes audio not yet transcribed, waits until ctx is done for its
	// events, and ends the session.
	Close(ctx context.Context) error
}
