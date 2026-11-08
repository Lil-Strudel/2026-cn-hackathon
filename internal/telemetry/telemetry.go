package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Setup installs global OpenTelemetry providers configured from the standard
// OTEL_* environment variables, plus a JSON slog logger carrying trace context.
// Exporting is skipped when no OTLP endpoint is configured, so local runs
// without a collector stay quiet.
func Setup(ctx context.Context, serviceName string) (shutdown func(context.Context) error, err error) {
	slog.SetDefault(slog.New(traceContextHandler{slog.NewJSONHandler(os.Stdout, nil)}))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	if !exportConfigured() {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, err
	}
	res, err = resource.Merge(res, resource.Environment())
	if err != nil {
		return nil, err
	}

	spans, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, err
	}
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(spans), sdktrace.WithResource(res))
	otel.SetTracerProvider(tracerProvider)

	reader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, errors.Join(err, tracerProvider.Shutdown(ctx))
	}
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithResource(res))
	otel.SetMeterProvider(meterProvider)

	if err := runtime.Start(); err != nil {
		return nil, errors.Join(err, tracerProvider.Shutdown(ctx), meterProvider.Shutdown(ctx))
	}

	return func(ctx context.Context) error {
		return errors.Join(tracerProvider.Shutdown(ctx), meterProvider.Shutdown(ctx))
	}, nil
}

func exportConfigured() bool {
	for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_TRACES_EXPORTER"} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}
