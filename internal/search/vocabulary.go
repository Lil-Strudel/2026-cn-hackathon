package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/refresh"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentation = "github.com/Lil-Strudel/2026-cn-hackathon/internal/search"

// Term is a vocabulary entry. A term is not used when one of its Except words
// is said near it, for names that sound like everyday words: "Gitar" sounds
// like "guitar" and "Qatar", but "playing a guitar" should stay a guitar.
type Term struct {
	Term   string   `json:"term"`
	Except []string `json:"except,omitempty"`
}

// DefaultTerms seeds the vocabulary with cloud native names Whisper tends to
// mishear, until talks bring their own.
var DefaultTerms = append(plainTerms(
	"Argo CD", "Backstage", "CNCF", "Cilium", "containerd", "cosign", "Crossplane",
	"Dash0", "eBPF", "Elasticsearch", "Envoy", "etcd", "Falco", "Flux",
	"Grafana", "gRPC", "Helm", "Hubble", "Istio", "Jaeger", "KEDA", "Kibana",
	"KubeCon", "kubectl", "Kubernetes", "Linkerd", "OpenTelemetry",
	"Prometheus", "Sigstore", "SPIFFE", "Talos", "WebAssembly",
), Term{Term: "Gitar", Except: []string{
	"play", "music", "song", "band", "chord", "acoustic", "electric", "bass",
	"country", "Doha", "Airways", "flight", "travel",
}})

func plainTerms(names ...string) []Term {
	terms := make([]Term, len(names))
	for i, name := range names {
		terms[i] = Term{Term: name}
	}
	return terms
}

type Vocabulary struct {
	es          *elasticsearch.TypedClient
	index       string
	tracer      trace.Tracer
	corrections metric.Int64Counter
}

func NewVocabulary(es *elasticsearch.TypedClient) (*Vocabulary, error) {
	return newVocabulary(es, VocabularyIndex)
}

func newVocabulary(es *elasticsearch.TypedClient, index string) (*Vocabulary, error) {
	corrections, err := otel.Meter(instrumentation).Int64Counter("captions.corrections",
		metric.WithDescription("Words or phrases replaced by a vocabulary term"))
	if err != nil {
		return nil, err
	}
	return &Vocabulary{es: es, index: index, tracer: otel.Tracer(instrumentation), corrections: corrections}, nil
}

// Add stores terms, keyed by their lowercase form, so adding a term again
// replaces its exception words. It waits for a refresh so the terms correct the next
// request.
func (v *Vocabulary) Add(ctx context.Context, terms []Term) error {
	if len(terms) == 0 {
		return nil
	}
	bulk := v.es.Bulk().Index(v.index).Refresh(refresh.Waitfor)
	for _, term := range terms {
		id := strings.ToLower(term.Term)
		if err := bulk.IndexOp(types.IndexOperation{Id_: &id}, term); err != nil {
			return err
		}
	}
	res, err := bulk.Do(ctx)
	if err != nil {
		return fmt.Errorf("add terms: %w", err)
	}
	if res.Errors {
		return fmt.Errorf("add terms: some terms were rejected")
	}
	return nil
}

// Seed stores the terms that aren't in the vocabulary yet and leaves the rest
// alone, so starting the api never undoes exception words someone set.
func (v *Vocabulary) Seed(ctx context.Context, terms []Term) error {
	bulk := v.es.Bulk().Index(v.index).Refresh(refresh.Waitfor)
	for _, term := range terms {
		id := strings.ToLower(term.Term)
		if err := bulk.CreateOp(types.CreateOperation{Id_: &id}, term); err != nil {
			return err
		}
	}
	res, err := bulk.Do(ctx)
	if err != nil {
		return fmt.Errorf("seed terms: %w", err)
	}
	for _, item := range res.Items {
		for _, result := range item {
			if result.Error != nil && result.Status != http.StatusConflict {
				return fmt.Errorf("seed terms: %s", result.Error.Type)
			}
		}
	}
	return nil
}

// maxTerms bounds List. A vocabulary is names typed in by hand, far fewer than
// this.
const maxTerms = 1000

// List returns every term, sorted the way people read them, ignoring case.
func (v *Vocabulary) List(ctx context.Context) ([]Term, error) {
	ctx, span := v.tracer.Start(ctx, "vocabulary.list")
	defer span.End()

	res, err := v.es.Search().Index(v.index).Size(maxTerms).Do(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list terms: %w", err)
	}
	terms := make([]Term, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		var t Term
		if err := json.Unmarshal(hit.Source_, &t); err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}
	slices.SortFunc(terms, func(a, b Term) int {
		return strings.Compare(strings.ToLower(a.Term), strings.ToLower(b.Term))
	})
	span.SetAttributes(attribute.Int("vocabulary.terms", len(terms)))
	return terms, nil
}

func (v *Vocabulary) Empty(ctx context.Context) (bool, error) {
	res, err := v.es.Count().Index(v.index).Do(ctx)
	if err != nil {
		return false, fmt.Errorf("count terms: %w", err)
	}
	return res.Count == 0, nil
}

// Remove deletes a term, if it exists, and waits for a refresh so the next
// request is no longer corrected to it.
func (v *Vocabulary) Remove(ctx context.Context, term string) error {
	ctx, span := v.tracer.Start(ctx, "vocabulary.remove", trace.WithAttributes(attribute.String("vocabulary.term", term)))
	defer span.End()

	res, err := v.es.Delete(v.index, strings.ToLower(term)).Refresh(refresh.Waitfor).Do(ctx)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("remove term: %w", err)
	}
	span.SetAttributes(attribute.String("vocabulary.result", res.Result.String()))
	return nil
}

func (v *Vocabulary) Correct(ctx context.Context, text string) (Correction, error) {
	ctx, span := v.tracer.Start(ctx, "vocabulary.correct")
	defer span.End()

	c, err := correct(ctx, text, v.soundAlikes)
	if err != nil {
		span.RecordError(err)
		return Correction{}, err
	}
	corrected := 0
	for _, s := range c.Segments {
		if s.Heard != "" {
			corrected++
		}
	}
	span.SetAttributes(attribute.Int("captions.corrections", corrected))
	v.corrections.Add(ctx, int64(corrected))
	return c, nil
}

func (v *Vocabulary) soundAlikes(ctx context.Context, lookups []lookup) ([][]string, error) {
	if len(lookups) == 0 {
		return nil, nil
	}
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	for _, l := range lookups {
		if err := enc.Encode(map[string]any{}); err != nil {
			return nil, err
		}
		query := map[string]any{
			"size":    3,
			"_source": []string{"term"},
			"query":   soundAlikeQuery(l),
		}
		if err := enc.Encode(query); err != nil {
			return nil, err
		}
	}

	res, err := v.es.Msearch().Index(v.index).Raw(&body).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("match vocabulary: %w", err)
	}
	out := make([][]string, len(lookups))
	for i, item := range res.Responses {
		switch item := item.(type) {
		case *types.MultiSearchItem:
			for _, hit := range item.Hits.Hits {
				var t Term
				if err := json.Unmarshal(hit.Source_, &t); err != nil {
					return nil, err
				}
				out[i] = append(out[i], t.Term)
			}
		case *types.ErrorResponseBase:
			return nil, fmt.Errorf("match vocabulary: %s", item.Error.Type)
		}
	}
	return out, nil
}

func soundAlikeQuery(l lookup) map[string]any {
	return map[string]any{"bool": map[string]any{
		"must":     map[string]any{"match": map[string]any{"term.sound": l.Phrase}},
		"must_not": map[string]any{"match": map[string]any{"except": l.Window}},
	}}
}
