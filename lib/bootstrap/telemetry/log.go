package telemetry

import (
	"context"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
)

func InitLog(ctx context.Context, logger *zap.Logger, serviceName, httpEndpoint string) func() {
	var exp, err = otlploghttp.New(ctx, otlploghttp.WithEndpoint(httpEndpoint), otlploghttp.WithInsecure())
	fatal(logger, err, "crate log exporter error")
	var logProvider = log.NewLoggerProvider(
		log.WithResource(newResource(ctx, serviceName)),
		log.WithProcessor(log.NewBatchProcessor(exp)),
	)
	global.SetLoggerProvider(logProvider)
	wrapLogger := otelzap.New(
		logger,                               // zap实例，按需配置
		otelzap.WithMinLevel(zap.DebugLevel), // 指定日志级别
	)
	undo := otelzap.ReplaceGlobals(wrapLogger)
	return func() {
		_ = logProvider.Shutdown(context.Background())
		_ = wrapLogger.Sync()
		undo()
	}
}
