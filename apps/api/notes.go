package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/httpserver"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/notes"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
)

const (
	// Too little speech to say anything about, and not worth an LLM call.
	minTranscriptWords = 30
	// About five hours of speech, far more than a talk.
	maxTranscriptBytes = 300_000
	writeTimeout       = 2 * time.Minute
)

var errTranscriptTooLong = errors.New("transcript is too long to summarise")

type notesStore interface {
	Get(ctx context.Context, talkID string) (*notes.Notes, error)
	Store(ctx context.Context, talkID string, n notes.Notes) error
}

type talkLookup interface {
	Get(ctx context.Context, id string) (*search.Talk, error)
}

type talkNotes struct {
	history captionHistory
	store   notesStore
	talks   talkLookup
	// writer is nil without an OpenAI key; notes then stay unwritten.
	writer   notes.Writer
	inflight singleflight.Group
}

// notesBody is the web app's TalkNotes: both fields are null until written.
type notesBody struct {
	Summary *string  `json:"summary"`
	Notes   []string `json:"notes"`
}

func bodyOf(n *notes.Notes) notesBody {
	if n == nil {
		return notesBody{}
	}
	points := n.Notes
	if points == nil {
		points = []string{}
	}
	return notesBody{Summary: &n.Summary, Notes: points}
}

func (t *talkNotes) handleGet(w http.ResponseWriter, r *http.Request) {
	talkID := r.PathValue("talkID")
	if !captions.ValidTalkID(talkID) {
		http.Error(w, "invalid talk id", http.StatusNotFound)
		return
	}
	stored, err := t.store.Get(r.Context(), talkID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get notes", "talk_id", talkID, "err", err)
		http.Error(w, "notes unavailable", http.StatusBadGateway)
		return
	}
	if stored != nil || t.writer == nil {
		httpserver.WriteJSON(w, http.StatusOK, bodyOf(stored))
		return
	}
	// A live talk gets no notes yet, or they would cover only its first half.
	// Ending a talk starts its notes; this writes them again only if that
	// write failed or the api restarted during it.
	talk, err := t.talks.Get(r.Context(), talkID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get talk for notes", "talk_id", talkID, "err", err)
		http.Error(w, "notes unavailable", http.StatusBadGateway)
		return
	}
	if talk == nil || talk.Status != search.TalkFinished {
		httpserver.WriteJSON(w, http.StatusOK, bodyOf(nil))
		return
	}
	t.respond(w, r, talkID, false)
}

// start writes a just-ended talk's notes in the background, so they are ready,
// or already being written, when viewers of the finished talk ask for them.
func (t *talkNotes) start(ctx context.Context, talkID string) {
	if t.writer == nil {
		return
	}
	go func() {
		if _, err := t.write(ctx, talkID, false); err != nil {
			slog.ErrorContext(ctx, "write notes", "talk_id", talkID, "err", err)
		}
	}()
}

func (t *talkNotes) handleRewrite(w http.ResponseWriter, r *http.Request) {
	talkID := r.PathValue("talkID")
	if !captions.ValidTalkID(talkID) {
		http.Error(w, "invalid talk id", http.StatusNotFound)
		return
	}
	if t.writer == nil {
		http.Error(w, "notes need OPENAI_API_KEY", http.StatusServiceUnavailable)
		return
	}
	t.respond(w, r, talkID, true)
}

func (t *talkNotes) respond(w http.ResponseWriter, r *http.Request, talkID string, force bool) {
	n, err := t.write(r.Context(), talkID, force)
	if err != nil {
		slog.ErrorContext(r.Context(), "write notes", "talk_id", talkID, "err", err)
		status := http.StatusBadGateway
		if errors.Is(err, errTranscriptTooLong) {
			status = http.StatusUnprocessableEntity
		}
		http.Error(w, "writing notes failed", status)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, bodyOf(n))
}

// write returns nil notes, without calling the LLM, when the transcript is too
// short. Unless forced, it returns notes already stored instead of writing
// them again. Concurrent calls for one talk share a single LLM request, which
// outlives the request that started it.
func (t *talkNotes) write(ctx context.Context, talkID string, force bool) (*notes.Notes, error) {
	key := talkID
	if force {
		key += "/force"
	}
	v, err, _ := t.inflight.Do(key, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), writeTimeout)
		defer cancel()

		if !force {
			stored, err := t.store.Get(ctx, talkID)
			if err != nil || stored != nil {
				return stored, err
			}
		}
		lines, err := t.history.History(ctx, talkID)
		if err != nil {
			return (*notes.Notes)(nil), err
		}
		transcript := transcriptOf(lines)
		if len(strings.Fields(transcript)) < minTranscriptWords {
			return (*notes.Notes)(nil), nil
		}
		if len(transcript) > maxTranscriptBytes {
			return (*notes.Notes)(nil), errTranscriptTooLong
		}

		n, err := t.writer.Write(ctx, transcript)
		if err != nil {
			return (*notes.Notes)(nil), err
		}
		if err := t.store.Store(ctx, talkID, n); err != nil {
			slog.ErrorContext(ctx, "store notes", "talk_id", talkID, "err", err)
		}
		return &n, nil
	})
	return v.(*notes.Notes), err
}

func transcriptOf(lines []captions.Caption) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line.Text)
		b.WriteByte('\n')
	}
	return b.String()
}
