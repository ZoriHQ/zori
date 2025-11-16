package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName           string
	ServiceVersion        string
	Environment           string
	OTLPEndpoint          string
	Enabled               bool
	HTTPSamplingRate      float64
	IngestionSamplingRate float64
}

type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	config         Config
}

func NewProvider(cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{
			tracerProvider: sdktrace.NewTracerProvider(),
			config:         cfg,
		}, nil
	}

	if cfg.HTTPSamplingRate == 0 {
		cfg.HTTPSamplingRate = 1.0
	}
	if cfg.IngestionSamplingRate == 0 {
		cfg.IngestionSamplingRate = 0.1
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		tracerProvider: tp,
		config:         cfg,
	}, nil
}

func (p *Provider) Tracer(name string) trace.Tracer {
	return p.tracerProvider.Tracer(name)
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}
	return p.tracerProvider.Shutdown(ctx)
}

func (p *Provider) HTTPSampler() sdktrace.Sampler {
	return sdktrace.TraceIDRatioBased(p.config.HTTPSamplingRate)
}

func (p *Provider) IngestionSampler() sdktrace.Sampler {
	return sdktrace.TraceIDRatioBased(p.config.IngestionSamplingRate)
}

func (p *Provider) ShouldSampleIngestion(traceID trace.TraceID) bool {
	if !p.config.Enabled {
		return false
	}

	sampler := p.IngestionSampler()
	params := sdktrace.SamplingParameters{
		TraceID: traceID,
	}

	result := sampler.ShouldSample(params)
	return result.Decision == sdktrace.RecordAndSample
}

const (
	AttrOrgID     = attribute.Key("zori.org_id")
	AttrProjectID = attribute.Key("zori.project_id")
	AttrVisitorID = attribute.Key("zori.visitor_id")
	AttrEventType = attribute.Key("zori.event_type")
	AttrUserID    = attribute.Key("zori.user_id")
)

func StartSpan(ctx context.Context, tracer trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, opts...)
}

func StartIngestionSpan(ctx context.Context, tracer trace.Tracer, name, orgID, projectID string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			AttrOrgID.String(orgID),
			AttrProjectID.String(projectID),
		),
	)
}

func StartHTTPSpan(ctx context.Context, tracer trace.Tracer, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
	}

	if len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}

	return tracer.Start(ctx, name, opts...)
}

func RecordError(span trace.Span, err error) {
	if err != nil && span != nil {
		span.RecordError(err)
	}
}

func SetStatus(span trace.Span, err error) {
	if span == nil {
		return
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
}
