package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// DefaultTerms seeds the vocabulary with cloud native names Whisper tends to
// mishear, until talks bring their own.
var DefaultTerms = []string{
	"Argo CD", "Backstage", "CNCF", "Cilium", "containerd", "cosign", "Crossplane",
	"Dash0", "eBPF", "Elasticsearch", "Envoy", "etcd", "Falco", "Flux", "Gitar",
	"Grafana", "gRPC", "Helm", "Hubble", "Istio", "Jaeger", "KEDA", "Kibana",
	"Knative", "KubeCon", "kubectl", "Kubernetes", "Linkerd", "OpenTelemetry",
	"Prometheus", "Sigstore", "SPIFFE", "Talos", "WebAssembly",
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

type vocabularyTerm struct {
	Term string `json:"term"`
}

// Add stores terms, keyed by their lowercase form so adding one twice is
// harmless. It waits for a refresh so the terms correct the next request.
func (v *Vocabulary) Add(ctx context.Context, terms []string) error {
	if len(terms) == 0 {
		return nil
	}
	bulk := v.es.Bulk().Index(v.index).Refresh(refresh.Waitfor)
	for _, term := range terms {
		id := strings.ToLower(term)
		if err := bulk.IndexOp(types.IndexOperation{Id_: &id}, vocabularyTerm{Term: term}); err != nil {
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

func (v *Vocabulary) soundAlikes(ctx context.Context, phrases []string) ([][]string, error) {
	if len(phrases) == 0 {
		return nil, nil
	}
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	for _, p := range phrases {
		if err := enc.Encode(map[string]any{}); err != nil {
			return nil, err
		}
		query := map[string]any{
			"size":    3,
			"_source": []string{"term"},
			"query":   map[string]any{"match": map[string]any{"term.sound": p}},
		}
		if err := enc.Encode(query); err != nil {
			return nil, err
		}
	}

	res, err := v.es.Msearch().Index(v.index).Raw(&body).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("match vocabulary: %w", err)
	}
	out := make([][]string, len(phrases))
	for i, item := range res.Responses {
		switch item := item.(type) {
		case *types.MultiSearchItem:
			for _, hit := range item.Hits.Hits {
				var t vocabularyTerm
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
