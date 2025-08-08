package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
	"time"
)

func InitMeter(ctx context.Context, logger *zap.Logger, serviceName, url string) func() {
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(url),
		otlpmetricgrpc.WithInsecure())
	fatal(logger, err, "create meter exporter error")
	meterProvider, err := newMeterProvider(ctx, metricExporter, serviceName)
	fatal(logger, err, "create meter provider error")
	otel.SetMeterProvider(meterProvider)
	return func() {
		_ = meterProvider.Shutdown(context.Background())
	}
}

func newMeterProvider(ctx context.Context, exp *otlpmetricgrpc.Exporter, serviceName string) (*metric.MeterProvider, error) {

	res := newResource(ctx, serviceName)
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exp,
			metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}
