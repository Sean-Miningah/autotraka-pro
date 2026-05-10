package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracer bootstraps the OpenTelemetry tracer provider with an OTLP/gRPC
// exporter, batch span processor, and env-var-driven sampling.
func InitTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithExportTimeout(30*time.Second),
		),
		sdktrace.WithSampler(samplerFromEnv()),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// ShutdownTracer gracefully flushes and shuts down the tracer provider.
func ShutdownTracer(ctx context.Context, tp *sdktrace.TracerProvider) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return tp.Shutdown(ctx)
}

// samplerFromEnv reads OTEL_TRACES_SAMPLER / OTEL_TRACES_SAMPLER_ARG and
// returns the matching sdktrace.Sampler. Falls back to ParentBased(AlwaysOn).
func samplerFromEnv() sdktrace.Sampler {
	s := os.Getenv("OTEL_TRACES_SAMPLER")
	arg := os.Getenv("OTEL_TRACES_SAMPLER_ARG")

	switch s {
	case "traceidratio":
		ratio, _ := strconv.ParseFloat(arg, 64)
		return sdktrace.TraceIDRatioBased(ratio)
	case "parentbased_traceidratio":
		ratio, _ := strconv.ParseFloat(arg, 64)
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	case "always_off":
		return sdktrace.NeverSample()
	case "always_on":
		return sdktrace.AlwaysSample()
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}
