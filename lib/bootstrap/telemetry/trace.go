package telemetry

import (
	"context"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
	"time"
)

type ShutdownWrap = func()

var Tracer trace.Tracer

func GrpcClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}

func GrpcServerHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

func InitTracer(ctx context.Context, logger *zap.Logger, serviceName, url string) (shutdown ShutdownWrap) {

	prop := newPropagator()

	otel.SetTextMapPropagator(prop)

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(url),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	exporter, err := otlptrace.New(ctx, traceClient)
	fatal(logger, err, "create trace exporter error")

	tracerProvider, err := newTraceProvider(ctx, exporter, serviceName)
	fatal(logger, err, "create trace provider error")

	otel.SetTracerProvider(tracerProvider)

	Tracer = tracerProvider.Tracer(serviceName)

	shutdown = func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
	return
}
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(ctx context.Context, exporter sdktrace.SpanExporter, serviceName string) (*sdktrace.TracerProvider, error) {

	processor := sdktrace.NewBatchSpanProcessor(exporter)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(processor),
		sdktrace.WithSampler(newSampler("dev")),
		sdktrace.WithResource(newResource(ctx, serviceName)),
	)
	return traceProvider, nil
}

func newZipkinExporter(url string) (*zipkin.Exporter, error) {
	return zipkin.New(url)
}

// 采样器将决定哪些跨度会被采集
func newSampler(mode string) sdktrace.Sampler {
	return sdktrace.AlwaysSample()
}

// 资源会存储额外信息
func newResource(ctx context.Context, serviceName string) *resource.Resource {
	if val, err := resource.New(ctx, resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.TelemetrySDKLanguageGo,
		)); err == nil {
		return val
	}
	return nil
}
func fatal(logger *zap.Logger, err error, message string) {
	if err != nil {
		logger.Fatal(message, zap.Error(err))
	}
}
