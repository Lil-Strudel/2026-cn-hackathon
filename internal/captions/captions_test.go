package captions

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestEncodeDecodeCarriesCaptionAndTraceContext(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))
	want := Caption{
		TalkID: "demo", ID: "s1-item_1", At: "2026-11-09T10:00:00.000Z", Text: "Hello Cilium", Final: true,
		Corrections: []Correction{{Start: 6, End: 12, Heard: "silly um"}},
	}

	data, err := Encode(ctx, want)
	if err != nil {
		t.Fatal(err)
	}
	gotCtx, got, err := Decode(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("caption: got %+v, want %+v", got, want)
	}
	if sc := trace.SpanContextFromContext(gotCtx); sc.TraceID() != traceID || sc.SpanID() != spanID {
		t.Fatalf("trace context not carried: %v", sc)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	if _, _, err := Decode(context.Background(), []byte("not json")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidTalkID(t *testing.T) {
	tests := map[string]bool{
		"demo":                  true,
		"kubecon-2026-keynote":  true,
		"":                      false,
		"Demo":                  false,
		"a/b":                   false,
		"talk*":                 false,
		strings.Repeat("a", 64): true,
		strings.Repeat("a", 65): false,
	}
	for id, want := range tests {
		if got := ValidTalkID(id); got != want {
			t.Errorf("ValidTalkID(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestFormatTimeSortsAsText(t *testing.T) {
	base := time.Date(2026, 11, 9, 10, 0, 5, 0, time.FixedZone("MST", -7*3600))
	earlier, later := FormatTime(base.Add(100*time.Millisecond)), FormatTime(base.Add(120*time.Millisecond))
	if earlier != "2026-11-09T17:00:05.100Z" {
		t.Fatalf("got %q", earlier)
	}
	if earlier >= later {
		t.Fatalf("%q should sort before %q", earlier, later)
	}
}
