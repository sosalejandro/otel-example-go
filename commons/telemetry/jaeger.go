package telemetry

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Returns a new OpenTelemetry resource describing this application.
func newResource(ctx context.Context) *resource.Resource {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(semconv.ServiceNameKey.String(os.Getenv("SERVICE_NAME")),
			attribute.String("environment", os.Getenv("GO_ENV")),
		),
	)
	if err != nil {
		log.Fatalf("%s: %v", "Failed to create resource", err)
	}
	return res
}

// Creates Jaeger exporter
func exporterToJaeger() (*jaeger.Exporter, error) {
	return jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(os.Getenv("OPEN_TELEMETRY_COLLECTOR_URL")),
		),
	)
}

// Initiates OpenTelemetry provider sending data to OpenTelemetry Collector.
func InitProviderWithJaegerExporter(ctx context.Context) (func(context.Context) error, error) {
	exp, err := exporterToJaeger()
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}
	tp := trace.NewTracerProvider(
		trace.WithSampler(GetSampler()),
		trace.WithBatcher(exp),
		trace.WithResource(newResource(ctx)),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
