package search

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/notes"
)

// TalkNotes keeps each talk's written notes on its document in the talks
// index, so talk search matches them alongside the title and abstract.
type TalkNotes struct {
	es     *elasticsearch.TypedClient
	index  string
	tracer trace.Tracer
}

func NewTalkNotes(es *elasticsearch.TypedClient) *TalkNotes {
	return newTalkNotes(es, TalksIndex)
}

func newTalkNotes(es *elasticsearch.TypedClient, index string) *TalkNotes {
	return &TalkNotes{es: es, index: index, tracer: otel.Tracer(instrumentation)}
}

type notesDoc struct {
	Summary   string    `json:"summary"`
	Notes     []string  `json:"notes"`
	WrittenAt time.Time `json:"notes_written_at"`
}

// Get returns a talk's notes, or nil when none have been written.
func (t *TalkNotes) Get(ctx context.Context, talkID string) (*notes.Notes, error) {
	ctx, span := t.tracer.Start(ctx, "notes.get", trace.WithAttributes(attribute.String("talk.id", talkID)))
	defer span.End()

	res, err := t.es.Get(t.index, talkID).SourceIncludes_("summary", "notes", "notes_written_at").Do(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get notes: %w", err)
	}
	if !res.Found {
		return nil, nil
	}
	var doc notesDoc
	if err := json.Unmarshal(res.Source_, &doc); err != nil {
		return nil, fmt.Errorf("get notes: %w", err)
	}
	if doc.WrittenAt.IsZero() {
		return nil, nil
	}
	span.SetAttributes(attribute.Bool("notes.found", true))
	return &notes.Notes{Summary: doc.Summary, Notes: doc.Notes}, nil
}

// Store saves a talk's notes, creating its document if the talk has none yet
// and leaving its other fields alone.
func (t *TalkNotes) Store(ctx context.Context, talkID string, n notes.Notes) error {
	ctx, span := t.tracer.Start(ctx, "notes.store", trace.WithAttributes(attribute.String("talk.id", talkID)))
	defer span.End()

	doc := notesDoc{Summary: n.Summary, Notes: n.Notes, WrittenAt: time.Now().UTC()}
	if _, err := t.es.Update(t.index, talkID).Doc(doc).DocAsUpsert(true).Do(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("store notes: %w", err)
	}
	return nil
}
