package search

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/captions"
)

func TestCaptionHistoryIsOrderedAndScopedToTheTalk(t *testing.T) {
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
	store := newCaptions(es, prefix+CaptionsIndex)

	second := captions.Caption{
		TalkID: "demo", ID: "s1-item_2", At: "2026-11-09T10:00:07.250Z", Text: "Then Cilium.", Final: true,
		Corrections: []captions.Correction{{Start: 5, End: 11, Heard: "silly um"}},
	}
	first := captions.Caption{
		TalkID: "demo", ID: "s1-item_1", At: "2026-11-09T10:00:03.000Z", Text: "First.", Final: true,
		Corrections: []captions.Correction{},
	}
	other := captions.Caption{TalkID: "other", ID: "s2-item_1", At: "2026-11-09T10:00:01.000Z", Text: "Elsewhere.", Final: true}
	for _, line := range []captions.Caption{second, first, other} {
		if err := store.Store(ctx, line, line.Text); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := es.Indices.Refresh().Index(prefix + CaptionsIndex).Do(ctx); err != nil {
		t.Fatal(err)
	}

	got, err := store.History(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if want := []captions.Caption{first, second}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}

	none, err := store.History(ctx, "nobody")
	if err != nil {
		t.Fatal(err)
	}
	if none == nil || len(none) != 0 {
		t.Fatalf("an unknown talk should have an empty, non-nil history: %#v", none)
	}
}
