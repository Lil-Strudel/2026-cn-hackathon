package transcribe

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Fake turns every two seconds of audio into a canned line, so the caption
// path runs without an OpenAI key. The lines mishear cloud native terms the
// way Whisper does, so vocabulary correction shows up too.
type Fake struct{}

const fakeSegmentBytes = 2 * SampleRate * 2

var fakeLines = []string{
	"Welcome, and thanks for joining us in the hallway track.",
	"Our pods talk to each other over silly um and eBPF.",
	"The fanout service scales on connected clients with Kedah.",
	"Every request is traced from the browser down with OpenTelemetry.",
	"And all of it runs on cooper netties in a homelab.",
}

func (Fake) Start(context.Context, Options) (Session, error) {
	return &fakeSession{events: make(chan Event, 64)}, nil
}

type fakeSession struct {
	mu       sync.Mutex
	events   chan Event
	received int
	closed   bool
}

func (s *fakeSession) Events() <-chan Event { return s.events }

// Append emits half of a line after one second of audio and the whole line,
// final, after two.
func (s *fakeSession) Append(_ context.Context, pcm []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("session closed")
	}
	const half = fakeSegmentBytes / 2
	before := s.received
	s.received += len(pcm)
	for n := before/half + 1; n <= s.received/half; n++ {
		line := (n - 1) / 2
		words := strings.Fields(fakeLines[line%len(fakeLines)])
		event := Event{SegmentID: fmt.Sprintf("fake_%d", line), Text: strings.Join(words, " ")}
		if n%2 == 1 {
			event.Text = strings.Join(words[:len(words)/2], " ")
		} else {
			event.Final = true
			event.CommittedAt = time.Now()
		}
		s.events <- event
	}
	return nil
}

func (s *fakeSession) Close(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.events)
	}
	return nil
}
