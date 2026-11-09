package search

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/notes"
)

func TestTalkNotes(t *testing.T) {
	es, ctx := testClient(t)
	index := fmt.Sprintf("test-%d-%s", time.Now().UnixNano(), TalksIndex)
	t.Cleanup(func() { _, _ = es.Indices.Delete(index).Do(context.Background()) })
	if _, err := es.Indices.Create(index).Raw(strings.NewReader(talksMapping)).Do(ctx); err != nil {
		t.Fatal(err)
	}
	store := newTalkNotes(es, index)

	if got, err := store.Get(ctx, "kubecon-keynote"); err != nil || got != nil {
		t.Fatalf("before writing: notes = %+v, err = %v", got, err)
	}

	title := map[string]string{"title": "Cilium in production"}
	if _, err := es.Index(index).Id("kubecon-keynote").Document(title).Do(ctx); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Get(ctx, "kubecon-keynote"); err != nil || got != nil {
		t.Fatalf("talk without notes: notes = %+v, err = %v", got, err)
	}

	want := notes.Notes{Summary: "Cilium replaces kube-proxy.", Notes: []string{"eBPF routes traffic"}}
	if err := store.Store(ctx, "kubecon-keynote", want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "kubecon-keynote")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Summary != want.Summary || !slices.Equal(got.Notes, want.Notes) {
		t.Errorf("notes = %+v, want %+v", got, want)
	}

	doc, err := es.Get(index, "kubecon-keynote").Do(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var kept map[string]any
	if err := json.Unmarshal(doc.Source_, &kept); err != nil {
		t.Fatal(err)
	}
	if kept["title"] != "Cilium in production" {
		t.Errorf("storing notes changed the talk: %v", kept)
	}
}
