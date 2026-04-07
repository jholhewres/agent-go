package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewStdoutTracerProvider creates a TracerProvider that writes spans to stdout.
// Useful for local development and testing without a running collector.
//
// Example:
//
//	tp, shutdown := otel.NewStdoutTracerProvider()
//	defer shutdown(context.Background())
//	tracer := tp.Tracer("my-service")
func NewStdoutTracerProvider() (*sdktrace.TracerProvider, func(context.Context) error, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, fmt.Errorf("otel: failed to create stdout exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	return tp, shutdown, nil
}
