package search

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
)

func fakeMatch(soundAlikes map[string][]string) matchFunc {
	return func(_ context.Context, lookups []lookup) ([][]string, error) {
		out := make([][]string, len(lookups))
		for i, l := range lookups {
			out[i] = soundAlikes[l.Phrase]
		}
		return out, nil
	}
}

func TestCorrect(t *testing.T) {
	match := fakeMatch(map[string][]string{
		"silly um":       {"Cilium"},
		"cooper netties": {"Kubernetes"},
		"cat":            {"KEDA"},
		"Kubernetes":     {"Kubernetes"},
	})
	tests := []struct {
		name, text, want string
		segments         []Segment
	}{
		{
			name:     "replaces a split term",
			text:     "we run silly um.",
			want:     "we run Cilium.",
			segments: []Segment{{Text: "we run "}, {Text: "Cilium", Heard: "silly um"}, {Text: "."}},
		},
		{
			name: "replaces several terms",
			text: "silly um on cooper netties",
			want: "Cilium on Kubernetes",
			segments: []Segment{
				{Text: "Cilium", Heard: "silly um"}, {Text: " on "}, {Text: "Kubernetes", Heard: "cooper netties"},
			},
		},
		{
			name:     "keeps a sound-alike that is spelled too differently",
			text:     "the cat",
			want:     "the cat",
			segments: []Segment{{Text: "the cat"}},
		},
		{
			name:     "keeps a term already spelled right",
			text:     "Kubernetes",
			want:     "Kubernetes",
			segments: []Segment{{Text: "Kubernetes"}},
		},
		{
			name: "handles empty text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := correct(context.Background(), tt.text, match)
			if err != nil {
				t.Fatal(err)
			}
			if got.Text != tt.want {
				t.Errorf("text = %q, want %q", got.Text, tt.want)
			}
			if !slices.Equal(got.Segments, tt.segments) {
				t.Errorf("segments = %+v, want %+v", got.Segments, tt.segments)
			}
		})
	}
}

func TestWindow(t *testing.T) {
	tests := []struct {
		name, text, phrase, want string
	}{
		{"stops at commas", "I am playing a guitar, in the country Qatar, and I am using Sonar Guitar.", "guitar", "I am playing a"},
		{"stays inside its clause", "I am playing a guitar, in the country Qatar, and I am using Sonar Guitar.", "Qatar", "in the country"},
		{"reaches the word before", "I am playing a guitar, in the country Qatar, and I am using Sonar Guitar.", "Guitar", "I am using Sonar"},
		{"takes four words each side", "I am playing a guitar in the country Qatar and I am using Sonar Guitar", "Qatar", "guitar in the country and I am using"},
		{"at the start", "silly um is fast", "silly um", "is fast"},
		{"alone", "Kedah", "Kedah", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen []lookup
			record := func(_ context.Context, lookups []lookup) ([][]string, error) {
				seen = lookups
				return make([][]string, len(lookups)), nil
			}
			if _, err := correct(context.Background(), tt.text, record); err != nil {
				t.Fatal(err)
			}
			i := slices.IndexFunc(seen, func(l lookup) bool { return l.Phrase == tt.phrase })
			if i < 0 {
				t.Fatalf("no lookup for %q in %+v", tt.phrase, seen)
			}
			if got := seen[i].Window; got != tt.want {
				t.Errorf("window = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSimilarity(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"silly um", "Cilium", true},
		{"cooper netties", "Kubernetes", true},
		{"Kedah", "KEDA", true},
		{"cat", "KEDA", false},
	}
	for _, tt := range tests {
		if _, ok := soundsRight(tt.a, tt.b); ok != tt.want {
			t.Errorf("soundsRight(%q, %q) = %v, want %v", tt.a, tt.b, ok, tt.want)
		}
	}
}

func TestVocabularyCorrect(t *testing.T) {
	es, ctx := testClient(t)
	prefix := fmt.Sprintf("test-%d-", time.Now().UnixNano())
	t.Cleanup(func() { _, _ = es.Indices.Delete(prefix + "*").Do(context.Background()) })
	if err := ensureIndices(ctx, es, prefix); err != nil {
		t.Fatal(err)
	}
	vocab, err := newVocabulary(es, prefix+VocabularyIndex)
	if err != nil {
		t.Fatal(err)
	}
	if err := vocab.Add(ctx, DefaultTerms); err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"we route traffic with silly um and hubble":                                 "we route traffic with Cilium and Hubble",
		"it runs on cooper netties":                                                 "it runs on Kubernetes",
		"scale it with Kedah":                                                       "scale it with KEDA",
		"the cat sat on the mat and said okay":                                      "the cat sat on the mat and said okay",
		"thanks everyone for coming to this talk":                                   "thanks everyone for coming to this talk",
		"I am playing a guitar, in the country Qatar, and I am using Sonar Guitar.": "I am playing a guitar, in the country Qatar, and I am using Sonar Gitar.",
		"I am playing a guitar in the country Qitar":                                "I am playing a guitar in the country Qitar",
		"open a pull request and Gitar reviews it":                                  "open a pull request and Gitar reviews it",
		"Guitar reviewed my pull request":                                           "Gitar reviewed my pull request",
	}
	for heard, want := range tests {
		got, err := vocab.Correct(ctx, heard)
		if err != nil {
			t.Fatal(err)
		}
		if got.Text != want {
			t.Errorf("Correct(%q) = %q, want %q", heard, got.Text, want)
		}
	}
}

func TestVocabularySeedKeepsExistingTerms(t *testing.T) {
	es, ctx := testClient(t)
	prefix := fmt.Sprintf("test-%d-", time.Now().UnixNano())
	t.Cleanup(func() { _, _ = es.Indices.Delete(prefix + "*").Do(context.Background()) })
	if err := ensureIndices(ctx, es, prefix); err != nil {
		t.Fatal(err)
	}
	vocab, err := newVocabulary(es, prefix+VocabularyIndex)
	if err != nil {
		t.Fatal(err)
	}
	if err := vocab.Add(ctx, []Term{{Term: "Helm", Context: []string{"chart"}}}); err != nil {
		t.Fatal(err)
	}
	if err := vocab.Seed(ctx, DefaultTerms); err != nil {
		t.Fatal(err)
	}

	for heard, want := range map[string]string{
		"we install it with helm":        "we install it with helm",
		"the helm chart sets it up":      "the Helm chart sets it up",
		"we route traffic with silly um": "we route traffic with Cilium",
	} {
		got, err := vocab.Correct(ctx, heard)
		if err != nil {
			t.Fatal(err)
		}
		if got.Text != want {
			t.Errorf("Correct(%q) = %q, want %q", heard, got.Text, want)
		}
	}
}
