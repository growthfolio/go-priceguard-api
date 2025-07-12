package infrastructure

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingConfig configuração do tracing
type TracingConfig struct {
	Enabled     bool   `json:"enabled"`
	ServiceName string `json:"service_name"`
	Endpoint    string `json:"endpoint"`
	Environment string `json:"environment"`
}

// TracingManager gerencia a configuração de distributed tracing
type TracingManager struct {
	tracer         oteltrace.Tracer
	tracerProvider *trace.TracerProvider
	logger         *zap.Logger
	config         TracingConfig
}

// NewTracingManager cria uma nova instância do TracingManager
func NewTracingManager(config TracingConfig, logger *zap.Logger) (*TracingManager, error) {
	if !config.Enabled {
		logger.Info("Distributed tracing is disabled")
		return &TracingManager{
			config: config,
			logger: logger,
		}, nil
	}

	// Criar resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Configurar exportador OTLP
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(config.Endpoint),
		otlptracehttp.WithInsecure(), // Para desenvolvimento
	)
	if err != nil {
		return nil, err
	}

	// Configurar tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // Para desenvolvimento
	)

	// Definir como provider global
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Criar tracer
	tracer := tp.Tracer(config.ServiceName)

	logger.Info("Distributed tracing initialized",
		zap.String("service", config.ServiceName),
		zap.String("endpoint", config.Endpoint),
	)

	return &TracingManager{
		tracer:         tracer,
		tracerProvider: tp,
		logger:         logger,
		config:         config,
	}, nil
}

// GetTracer retorna o tracer configurado
func (tm *TracingManager) GetTracer() oteltrace.Tracer {
	return tm.tracer
}

// StartSpan inicia um novo span
func (tm *TracingManager) StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if !tm.config.Enabled || tm.tracer == nil {
		return ctx, oteltrace.SpanFromContext(ctx)
	}
	return tm.tracer.Start(ctx, name, opts...)
}

// Shutdown finaliza o tracing de forma limpa
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if tm.tracerProvider == nil {
		return nil
	}

	// Timeout para shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := tm.tracerProvider.Shutdown(shutdownCtx); err != nil {
		tm.logger.Error("Failed to shutdown tracer provider", zap.Error(err))
		return err
	}

	tm.logger.Info("Tracing shutdown completed")
	return nil
}

// NewDefaultTracingManager cria um TracingManager com configuração padrão
func NewDefaultTracingManager(logger *zap.Logger) (*TracingManager, error) {
	config := TracingConfig{
		Enabled:     getEnvBool("ENABLE_TRACING", false),
		ServiceName: getEnvString("OTEL_SERVICE_NAME", "priceguard-api"),
		Endpoint:    getEnvString("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318/v1/traces"),
		Environment: getEnvString("APP_ENV", "development"),
	}

	return NewTracingManager(config, logger)
}

// Helper functions
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
