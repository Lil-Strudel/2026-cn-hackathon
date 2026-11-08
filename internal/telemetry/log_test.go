package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestTraceContextHandlerAddsIDs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(traceContextHandler{slog.NewJSONHandler(&buf, nil)})

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))

	logger.InfoContext(ctx, "hello")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	if line["trace_id"] != traceID.String() || line["span_id"] != spanID.String() {
		t.Fatalf("trace context missing from log line: %v", line)
	}
}

func TestTraceContextHandlerWithoutSpan(t *testing.T) {
	var buf bytes.Buffer
	slog.New(traceContextHandler{slog.NewJSONHandler(&buf, nil)}).Info("hello")

	if bytes.Contains(buf.Bytes(), []byte("trace_id")) {
		t.Fatalf("unexpected trace_id without a span: %s", buf.String())
	}
}
