package search

import (
	"context"
	"regexp"
	"strings"
	"unicode"
)

// Segment is a piece of corrected text. Heard holds what Whisper wrote when
// the piece was corrected, and is empty when the text was kept as heard.
type Segment struct {
	Text  string `json:"text"`
	Heard string `json:"heard,omitempty"`
}

type Correction struct {
	Text     string    `json:"text"`
	Segments []Segment `json:"segments"`
}

const (
	maxPhraseWords = 3
	minLetters     = 3
	// Sounding alike isn't enough on its own: "cat" and "KEDA" share the code
	// KT. Requiring the spelling to be half the same keeps "silly um" →
	// "Cilium" (0.67) and rejects "cat" → "KEDA" (0.25).
	minSimilarity = 0.5
	// A term with context words needs one of them close by. Four words reach
	// "using Sonar Guitar" but not, in "the country Qatar and I am using
	// Sonar", from Qatar to Sonar.
	contextWords = 4
)

var (
	wordPattern = regexp.MustCompile(`[\p{L}\p{N}]+(?:'\p{L}+)?`)
	// A context word in the next clause is about something else.
	clauseBreak = regexp.MustCompile(`[.,;:!?]`)
)

// lookup is a phrase to match against the vocabulary, with the words around
// it that a term's context words are checked against.
type lookup struct {
	Phrase string
	Window string
}

type phrase struct {
	first, last int
	lookup
}

// matchFunc returns, for each lookup, vocabulary terms that sound like its
// phrase.
type matchFunc func(ctx context.Context, lookups []lookup) ([][]string, error)

type replacement struct {
	last       int
	term       string
	similarity float64
}

func correct(ctx context.Context, text string, match matchFunc) (Correction, error) {
	words := wordPattern.FindAllStringIndex(text, -1)
	candidates := phrases(text, words)

	lookups := make([]lookup, len(candidates))
	for i, c := range candidates {
		lookups[i] = c.lookup
	}
	terms, err := match(ctx, lookups)
	if err != nil {
		return Correction{}, err
	}

	best := make(map[int]replacement)
	for i, c := range candidates {
		for _, term := range terms[i] {
			s, ok := soundsRight(c.Phrase, term)
			if !ok {
				continue
			}
			cur, seen := best[c.first]
			if !seen || s > cur.similarity && !(cur.similarity == 1 && cur.last < c.last) {
				best[c.first] = replacement{last: c.last, term: term, similarity: s}
			}
		}
	}

	var segments []Segment
	var out strings.Builder
	cursor := 0
	add := func(s Segment) {
		segments = append(segments, s)
		out.WriteString(s.Text)
	}
	for i := 0; i < len(words); i++ {
		r, ok := best[i]
		if !ok {
			continue
		}
		start, end := words[i][0], words[r.last][1]
		if start > cursor {
			add(Segment{Text: text[cursor:start]})
		}
		if heard := text[start:end]; heard == r.term {
			add(Segment{Text: heard})
		} else {
			add(Segment{Text: r.term, Heard: heard})
		}
		cursor = end
		i = r.last
	}
	if cursor < len(text) {
		add(Segment{Text: text[cursor:]})
	}
	return Correction{Text: out.String(), Segments: segments}, nil
}

func phrases(text string, words [][]int) []phrase {
	var out []phrase
	for first := range words {
		for last := first; last < len(words) && last < first+maxPhraseWords; last++ {
			heard := text[words[first][0]:words[last][1]]
			if len(letters(heard)) >= minLetters {
				out = append(out, phrase{first: first, last: last, lookup: lookup{
					Phrase: heard,
					Window: window(text, words, first, last),
				}})
			}
		}
	}
	return out
}

func window(text string, words [][]int, first, last int) string {
	joined := func(a, b int) bool {
		return !clauseBreak.MatchString(text[words[a][1]:words[b][0]])
	}
	var before, after []string
	for i := first - 1; i >= 0 && len(before) < contextWords && joined(i, i+1); i-- {
		before = append([]string{text[words[i][0]:words[i][1]]}, before...)
	}
	for i := last + 1; i < len(words) && len(after) < contextWords && joined(i-1, i); i++ {
		after = append(after, text[words[i][0]:words[i][1]])
	}
	return strings.Join(append(before, after...), " ")
}

func soundsRight(heard, term string) (float64, bool) {
	s := similarity(letters(heard), letters(term))
	return s, s >= minSimilarity
}

func letters(s string) []rune {
	var out []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, unicode.ToLower(r))
		}
	}
	return out
}

func similarity(a, b []rune) float64 {
	longest := max(len(a), len(b))
	if longest == 0 {
		return 0
	}
	return 1 - float64(levenshtein(a, b))/float64(longest)
}

func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
