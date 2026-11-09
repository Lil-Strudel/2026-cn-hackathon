package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
)

const (
	TalksIndex      = "talks"
	VocabularyIndex = "vocabulary"
	CaptionsIndex   = "captions"
)

const talksMapping = `{
  "mappings": {
    "properties": {
      "title":    { "type": "text", "analyzer": "english" },
      "abstract": { "type": "text", "analyzer": "english" },
      "speakers": { "type": "keyword" }
    }
  }
}`

// The "sound" subfield strips spaces before encoding, so a term Whisper splits
// into several words ("silly um") gets the same code as the term it misheard
// ("Cilium"). Encoding word by word gives SL AM against SLM and misses it.
const vocabularyMapping = `{
  "settings": {
    "analysis": {
      "char_filter": {
        "letters_and_digits": { "type": "pattern_replace", "pattern": "[^\\p{L}\\p{N}]", "replacement": "" }
      },
      "filter": {
        "double_metaphone": { "type": "phonetic", "encoder": "double_metaphone", "replace": true, "max_code_len": 8 }
      },
      "analyzer": {
        "sound": {
          "tokenizer": "keyword",
          "char_filter": ["letters_and_digits"],
          "filter": ["lowercase", "double_metaphone"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "term": {
        "type": "text",
        "fields": {
          "keyword": { "type": "keyword" },
          "sound":   { "type": "text", "analyzer": "sound" }
        }
      },
      "context":  { "type": "text", "analyzer": "english" },
      "talk_ids": { "type": "keyword" }
    }
  }
}`

const captionsMapping = `{
  "mappings": {
    "properties": {
      "talk_id":      { "type": "keyword" },
      "start_ms":     { "type": "long" },
      "end_ms":       { "type": "long" },
      "text":         { "type": "text", "analyzer": "english" },
      "whisper_text": { "type": "text", "index": false },
      "corrections":  { "type": "object", "enabled": false }
    }
  }
}`

var indices = map[string]string{
	TalksIndex:      talksMapping,
	VocabularyIndex: vocabularyMapping,
	CaptionsIndex:   captionsMapping,
}

// EnsureIndices creates any index that doesn't exist yet and adds new fields
// to the ones that do. Changing an existing field needs the index deleted
// first.
func EnsureIndices(ctx context.Context, es *elasticsearch.TypedClient) error {
	return ensureIndices(ctx, es, "")
}

func ensureIndices(ctx context.Context, es *elasticsearch.TypedClient, prefix string) error {
	for name, mapping := range indices {
		_, err := es.Indices.Create(prefix + name).Raw(strings.NewReader(mapping)).Do(ctx)
		if err == nil {
			continue
		}
		if !alreadyExists(err) {
			return fmt.Errorf("create index %s: %w", prefix+name, err)
		}
		if err := addFields(ctx, es, prefix+name, mapping); err != nil {
			return err
		}
	}
	return nil
}

func addFields(ctx context.Context, es *elasticsearch.TypedClient, index, mapping string) error {
	var parts struct {
		Mappings json.RawMessage `json:"mappings"`
	}
	if err := json.Unmarshal([]byte(mapping), &parts); err != nil {
		return fmt.Errorf("read mapping of %s: %w", index, err)
	}
	if _, err := es.Indices.PutMapping(index).Raw(bytes.NewReader(parts.Mappings)).Do(ctx); err != nil {
		return fmt.Errorf("update mapping of %s: %w", index, err)
	}
	return nil
}

// Several api replicas start at once and race to create the same index.
func alreadyExists(err error) bool {
	var esErr *types.ElasticsearchError
	return errors.As(err, &esErr) && esErr.ErrorCause.Type == "resource_already_exists_exception"
}
