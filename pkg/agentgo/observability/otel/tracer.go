// Package otel provides OpenTelemetry tracing integration for AgentGo.
// It exposes a tracer provider factory and hook implementations that plug into
// the existing agent hook system without modifying agent.go.
package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	defaultEndpoint     = "http://localhost:4318"
	defaultSamplingRate = 1.0
)

// Config holds configuration for the OTel tracer provider.
type Config struct {
	// Endpoint is the OTLP HTTP collector endpoint (default: http://localhost:4318).
	Endpoint string

	// Headers are optional HTTP headers sent with every export request (e.g. auth tokens).
	Headers map[string]string

	// ServiceName is the logical name of the service (required).
	ServiceName string

	// ServiceVersion is the version of the service (e.g. "1.0.0").
	ServiceVersion string

	// SamplingRate controls the fraction of traces that are sampled [0.0, 1.0].
	// Default is 1.0 (sample everything). Uses ParentBased(TraceIDRatioBased).
	SamplingRate float64

	// ResourceAttributes are additional key/value pairs attached to every span
	// as resource attributes (e.g. deployment.environment, host.name).
	ResourceAttributes map[string]string
}

// NewTracerProvider creates a configured TracerProvider that exports via OTLP HTTP.
// Returns the provider, a shutdown function (call it on program exit), and any error.
//
// Example:
//
//	tp, shutdown, err := otel.NewTracerProvider(ctx, otel.Config{
//	    ServiceName:  "my-agent",
//	    SamplingRate: 0.1,
//	})
//	if err != nil { log.Fatal(err) }
//	defer shutdown(context.Background())
func NewTracerProvider(ctx context.Context, cfg Config) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}
	if cfg.SamplingRate == 0 {
		cfg.SamplingRate = defaultSamplingRate
	}

	// Build OTLP HTTP exporter options.
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Endpoint),
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("otel: failed to create OTLP HTTP exporter: %w", err)
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("otel: failed to build resource: %w", err)
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplingRate))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	return tp, shutdown, nil
}

// buildResource constructs the OTel resource from Config fields.
func buildResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.ServiceVersion))
	}
	for k, v := range cfg.ResourceAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	return resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithTelemetrySDK(),
		resource.WithOS(),
	)
}
