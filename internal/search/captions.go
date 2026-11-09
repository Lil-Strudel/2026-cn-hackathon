package search

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/sortorder"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
)

// maxHistory is Elasticsearch's default result window, and hours of captions.
const maxHistory = 10000

// Captions stores the final lines of every talk's transcript.
type Captions struct {
	es     *elasticsearch.TypedClient
	index  string
	tracer trace.Tracer
}

func NewCaptions(es *elasticsearch.TypedClient) *Captions {
	return newCaptions(es, CaptionsIndex)
}

func newCaptions(es *elasticsearch.TypedClient, index string) *Captions {
	return &Captions{es: es, index: index, tracer: otel.Tracer(instrumentation)}
}

type captionDoc struct {
	TalkID      string                `json:"talk_id"`
	StartMs     int64                 `json:"start_ms"`
	EndMs       int64                 `json:"end_ms"`
	Text        string                `json:"text"`
	WhisperText string                `json:"whisper_text"`
	Corrections []captions.Correction `json:"corrections"`
}

// Store saves a final line under its ID, so storing it again overwrites it.
// heard is the text as transcribed, before correction.
func (c *Captions) Store(ctx context.Context, line captions.Caption, heard string) error {
	ctx, span := c.tracer.Start(ctx, "captions.store", trace.WithAttributes(
		attribute.String("talk.id", line.TalkID),
		attribute.String("caption.id", line.ID),
	))
	defer span.End()

	start, err := time.Parse(time.RFC3339, line.At)
	if err != nil {
		return fmt.Errorf("caption time: %w", err)
	}
	_, err = c.es.Index(c.index).Id(line.ID).Document(captionDoc{
		TalkID:      line.TalkID,
		StartMs:     start.UnixMilli(),
		EndMs:       time.Now().UnixMilli(),
		Text:        line.Text,
		WhisperText: heard,
		Corrections: line.Corrections,
	}).Do(ctx)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("store caption: %w", err)
	}
	return nil
}

// Count returns how many final lines a talk has stored.
func (c *Captions) Count(ctx context.Context, talkID string) (int64, error) {
	ctx, span := c.tracer.Start(ctx, "captions.count", trace.WithAttributes(attribute.String("talk.id", talkID)))
	defer span.End()

	res, err := c.es.Count().Index(c.index).
		Query(&types.Query{Term: map[string]types.TermQuery{"talk_id": {Value: talkID}}}).
		Do(ctx)
	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("count captions: %w", err)
	}
	span.SetAttributes(attribute.Int64("captions.lines", res.Count))
	return res.Count, nil
}

// History returns a talk's stored lines in the order they were said.
func (c *Captions) History(ctx context.Context, talkID string) ([]captions.Caption, error) {
	ctx, span := c.tracer.Start(ctx, "captions.history", trace.WithAttributes(attribute.String("talk.id", talkID)))
	defer span.End()

	res, err := c.es.Search().Index(c.index).
		Query(&types.Query{Term: map[string]types.TermQuery{"talk_id": {Value: talkID}}}).
		Sort(&types.SortOptions{SortOptions: map[string]types.FieldSort{"start_ms": {Order: &sortorder.Asc}}}).
		Size(maxHistory).
		Do(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("caption history: %w", err)
	}

	lines := make([]captions.Caption, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		var doc captionDoc
		if err := json.Unmarshal(hit.Source_, &doc); err != nil {
			return nil, err
		}
		corrections := doc.Corrections
		if corrections == nil {
			corrections = []captions.Correction{}
		}
		id := ""
		if hit.Id_ != nil {
			id = *hit.Id_
		}
		lines = append(lines, captions.Caption{
			TalkID:      doc.TalkID,
			ID:          id,
			At:          captions.FormatTime(time.UnixMilli(doc.StartMs)),
			Text:        doc.Text,
			Final:       true,
			Corrections: corrections,
		})
	}
	span.SetAttributes(attribute.Int("captions.lines", len(lines)))
	return lines, nil
}
