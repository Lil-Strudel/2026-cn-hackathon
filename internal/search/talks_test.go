package search

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/refresh"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/notes"
)

func TestSlug(t *testing.T) {
	tests := map[string]string{
		"  Café: eBPF & You! ":    "cafe-ebpf-you",
		"Guitar demo":             "guitar-demo",
		"🚀🚀":                      "talk",
		strings.Repeat("ab ", 30): strings.TrimSuffix(strings.Repeat("ab-", 16), "-"),
	}
	for title, want := range tests {
		got := slug(title)
		if got != want {
			t.Errorf("slug(%q) = %q, want %q", title, got, want)
		}
		if !captions.ValidTalkID(got + "-100") {
			t.Errorf("slug(%q) = %q leaves no room for a numbered id", title, got)
		}
	}
}

func TestTalks(t *testing.T) {
	es, ctx := testClient(t)
	prefix := fmt.Sprintf("test-%d-", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = es.Indices.Delete(prefix + TalksIndex + "," + prefix + CaptionsIndex).Do(context.Background())
	})
	for name, mapping := range map[string]string{TalksIndex: talksMapping, CaptionsIndex: captionsMapping} {
		if _, err := es.Indices.Create(prefix + name).Raw(strings.NewReader(mapping)).Do(ctx); err != nil {
			t.Fatal(err)
		}
	}
	talks, err := newTalks(es, prefix+TalksIndex, prefix+CaptionsIndex)
	if err != nil {
		t.Fatal(err)
	}

	guitar, err := talks.Create(ctx, NewTalk{Title: "Guitar demo", Speaker: "Aaron", Description: "Live captions for a guitar lesson."})
	if err != nil {
		t.Fatal(err)
	}
	if guitar.ID != "guitar-demo" || guitar.Status != TalkLive || guitar.EndedAt != nil {
		t.Errorf("created %+v", guitar)
	}
	again, err := talks.Create(ctx, NewTalk{Title: "Guitar demo!", Speaker: "Someone else"})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != "guitar-demo-2" {
		t.Errorf("second talk with the same title got id %q, want guitar-demo-2", again.ID)
	}

	got, err := talks.Get(ctx, "guitar-demo")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != guitar {
		t.Errorf("get = %+v, want %+v", got, guitar)
	}

	notesOnly := newTalkNotes(es, prefix+TalksIndex)
	if err := notesOnly.Store(ctx, "untracked", notes.Notes{Summary: "Guitar strings stretch."}); err != nil {
		t.Fatal(err)
	}
	if got, err := talks.Get(ctx, "untracked"); err != nil || got != nil {
		t.Errorf("notes without a talk: get = %+v, err = %v", got, err)
	}
	if got, err := talks.Get(ctx, "missing"); err != nil || got != nil {
		t.Errorf("missing talk: get = %+v, err = %v", got, err)
	}

	if _, err := es.Indices.Refresh().Index(prefix + TalksIndex).Do(ctx); err != nil {
		t.Fatal(err)
	}
	list, err := talks.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != "guitar-demo-2" || list[1].ID != "guitar-demo" {
		t.Errorf("list = %+v, want the two talks, newest first", list)
	}

	quiet, err := talks.Create(ctx, NewTalk{Title: "Tuning by ear", Speaker: "Lena"})
	if err != nil {
		t.Fatal(err)
	}
	line := captions.Caption{TalkID: quiet.ID, ID: "line-1", At: captions.FormatTime(time.Now()), Text: "Then I picked up my guitar again."}
	if err := newCaptions(es, prefix+CaptionsIndex).Store(ctx, line, line.Text); err != nil {
		t.Fatal(err)
	}
	if _, err := es.Indices.Refresh().Index(prefix + CaptionsIndex).Do(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := es.Index(prefix + CaptionsIndex).Id("orphan").Refresh(refresh.True).Document(captionDoc{TalkID: "untracked", Text: "guitar"}).Do(ctx); err != nil {
		t.Fatal(err)
	}

	hits, err := talks.Search(ctx, "guitar")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(hits))
	for i, hit := range hits {
		ids[i] = hit.Talk.ID
	}
	if len(hits) != 3 || hits[2].Talk.ID != quiet.ID {
		t.Fatalf("search hits %v, want both guitar demos then the talk that only says it", ids)
	}
	described := hits[0]
	if described.Talk.ID != "guitar-demo" {
		described = hits[1]
	}
	if !hasHighlight(described.Highlights, "description", "<mark>guitar</mark> lesson") {
		t.Errorf("description highlights = %+v", described.Highlights)
	}
	said := hits[2].Highlights
	if len(said) != 1 || said[0].Field != "transcript" || said[0].LineID != "line-1" || said[0].At != line.At ||
		said[0].Fragment != "Then I picked up my <mark>guitar</mark> again." {
		t.Errorf("transcript highlights = %+v", said)
	}
}

func hasHighlight(highlights []Highlight, field, fragment string) bool {
	for _, h := range highlights {
		if h.Field == field && strings.Contains(h.Fragment, fragment) {
			return true
		}
	}
	return false
}
