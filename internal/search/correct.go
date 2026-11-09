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
)

var wordPattern = regexp.MustCompile(`[\p{L}\p{N}]+(?:'\p{L}+)?`)

type phrase struct {
	first, last int
	text        string
}

// matchFunc returns, for each phrase, vocabulary terms that sound like it.
type matchFunc func(ctx context.Context, phrases []string) ([][]string, error)

type replacement struct {
	last       int
	term       string
	similarity float64
}

func correct(ctx context.Context, text string, match matchFunc) (Correction, error) {
	words := wordPattern.FindAllStringIndex(text, -1)
	candidates := phrases(text, words)

	texts := make([]string, len(candidates))
	for i, c := range candidates {
		texts[i] = c.text
	}
	terms, err := match(ctx, texts)
	if err != nil {
		return Correction{}, err
	}

	best := make(map[int]replacement)
	for i, c := range candidates {
		for _, term := range terms[i] {
			s, ok := soundsRight(c.text, term)
			if ok && s > best[c.first].similarity {
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
			p := phrase{first: first, last: last, text: text[words[first][0]:words[last][1]]}
			if len(letters(p.text)) >= minLetters {
				out = append(out, p)
			}
		}
	}
	return out
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
