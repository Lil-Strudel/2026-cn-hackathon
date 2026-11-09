package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
)

func TestMappingsAreValidJSON(t *testing.T) {
	for name, mapping := range indices {
		if !json.Valid([]byte(mapping)) {
			t.Errorf("%s mapping is not valid JSON", name)
		}
	}
}

func TestSoundMatchesMisheardTerms(t *testing.T) {
	es, ctx := testClient(t)
	prefix := fmt.Sprintf("test-%d-", time.Now().UnixNano())
	t.Cleanup(func() {
		names := make([]string, 0, len(indices))
		for name := range indices {
			names = append(names, prefix+name)
		}
		_, _ = es.Indices.Delete(strings.Join(names, ",")).Do(context.Background())
	})

	if err := ensureIndices(ctx, es, prefix); err != nil {
		t.Fatal(err)
	}
	if err := ensureIndices(ctx, es, prefix); err != nil {
		t.Fatalf("second run should tolerate existing indices: %v", err)
	}

	for _, pair := range [][2]string{
		{"Cilium", "silly um"},
		{"Kubernetes", "cooper netties"},
		{"KEDA", "Kedah"},
		{"Gitar", "guitar"},
		{"Gitar", "Qatar"},
	} {
		term, heard := sound(ctx, t, es, prefix, pair[0]), sound(ctx, t, es, prefix, pair[1])
		if !slices.ContainsFunc(term, func(code string) bool { return slices.Contains(heard, code) }) {
			t.Errorf("%q sounds like %v, %q like %v", pair[0], term, pair[1], heard)
		}
	}
}

func TestEnsureIndicesAddsNewFields(t *testing.T) {
	es, ctx := testClient(t)
	index := fmt.Sprintf("test-%d-%s", time.Now().UnixNano(), VocabularyIndex)
	t.Cleanup(func() { _, _ = es.Indices.Delete(index).Do(context.Background()) })

	exceptField := `"except":   { "type": "text", "analyzer": "english" },`
	if !strings.Contains(vocabularyMapping, exceptField) {
		t.Fatal("vocabulary mapping no longer has the except field this test removes")
	}
	before := strings.Replace(vocabularyMapping, exceptField, "", 1)
	if _, err := es.Indices.Create(index).Raw(strings.NewReader(before)).Do(ctx); err != nil {
		t.Fatal(err)
	}
	if err := addFields(ctx, es, index, vocabularyMapping); err != nil {
		t.Fatal(err)
	}
	res, err := es.Indices.GetFieldMapping("except").Index(index).Do(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(res[index].Mappings) == 0 {
		t.Errorf("except field was not added to %s", index)
	}
}

func sound(ctx context.Context, t *testing.T, es *elasticsearch.TypedClient, prefix, text string) []string {
	t.Helper()
	res, err := es.Indices.Analyze().Index(prefix + VocabularyIndex).Field("term.sound").Text(text).Do(ctx)
	if err != nil {
		t.Fatal(err)
	}
	codes := make([]string, len(res.Tokens))
	for i, token := range res.Tokens {
		codes[i] = token.Token
	}
	return codes
}

// CI has no Elasticsearch, so tests that need one skip when it isn't reachable.
func testClient(t *testing.T) (*elasticsearch.TypedClient, context.Context) {
	t.Helper()
	url := os.Getenv("ELASTICSEARCH_URL")
	if url == "" {
		url = "http://localhost:9200"
	}
	es, err := elasticsearch.NewTyped(elasticsearch.WithAddresses(url))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	if ok, err := es.Ping().Do(ctx); err != nil || !ok {
		t.Skipf("elasticsearch not reachable at %s", url)
	}
	return es, ctx
}
