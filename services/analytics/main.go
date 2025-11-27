// Analytics Service - Tracks joke statistics and metrics
// Routes:
//   GET /healthz            -> health check
//   GET /api/v1/stats       -> returns joke statistics
//   POST /internal/track    -> internal endpoint for tracking (called by jokes service)

package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger        *zap.Logger
	tracer        trace.Tracer
	meter         metric.Meter
	trackingCount metric.Int64Counter

	// In-memory stats (in production, use a database)
	stats      = &Stats{requests: 0, totalJokes: 0}
	statsMutex sync.RWMutex
)

type Stats struct {
	requests   int64
	totalJokes int64
	lastUpdate time.Time
}

func initLogger() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	logger, err = config.Build()
	if err != nil {
		panic(err)
	}
}

func initTracer() func() {
	ctx := context.Background()

	signozEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if signozEndpoint == "" {
		signozEndpoint = "signoz-otel-collector.platform.svc.cluster.local:4317"
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(signozEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		logger.Fatal("Failed to create trace exporter", zap.Error(err))
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("analytics-service"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		logger.Fatal("Failed to create resource", zap.Error(err))
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

	tracer = tp.Tracer("analytics-service")

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}
}

func initMetrics() {
	meter = otel.Meter("analytics-service")

	var err error
	trackingCount, err = meter.Int64Counter(
		"analytics.tracks",
		metric.WithDescription("Number of analytics events tracked"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		logger.Fatal("Failed to create tracking counter", zap.Error(err))
	}
}

func trackEvent(ctx context.Context) {
	_, span := tracer.Start(ctx, "trackEvent")
	defer span.End()

	statsMutex.Lock()
	defer statsMutex.Unlock()

	stats.requests++
	stats.totalJokes++
	stats.lastUpdate = time.Now()

	trackingCount.Add(ctx, 1)

	span.SetAttributes(
		attribute.Int64("stats.requests", stats.requests),
		attribute.Int64("stats.total_jokes", stats.totalJokes),
	)

	logger.Info("Event tracked",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Int64("total_requests", stats.requests),
		zap.Int64("total_jokes", stats.totalJokes),
	)
}

func getStats(ctx context.Context) map[string]interface{} {
	_, span := tracer.Start(ctx, "getStats")
	defer span.End()

	statsMutex.RLock()
	defer statsMutex.RUnlock()

	result := map[string]interface{}{
		"total_requests": stats.requests,
		"total_jokes":    stats.totalJokes,
		"last_update":    stats.lastUpdate.Format(time.RFC3339),
		"uptime_seconds": time.Since(stats.lastUpdate).Seconds(),
	}

	span.SetAttributes(
		attribute.Int64("stats.requests", stats.requests),
		attribute.Int64("stats.total_jokes", stats.totalJokes),
	)

	logger.Info("Stats retrieved",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Int64("total_requests", stats.requests),
	)

	return result
}

func main() {
	initLogger()
	defer logger.Sync()

	shutdown := initTracer()
	defer shutdown()

	initMetrics()

	// Initialize stats
	stats.lastUpdate = time.Now()

	r := gin.Default()
	r.Use(otelgin.Middleware("analytics-service"))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "analytics-service",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/api/v1/stats", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)

		logger.Info("Stats requested",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("client_ip", c.ClientIP()),
		)

		statistics := getStats(ctx)
		c.JSON(http.StatusOK, statistics)
	})

	r.POST("/internal/track", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)

		logger.Info("Track event received",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
		)

		trackEvent(ctx)
		c.JSON(http.StatusOK, gin.H{"status": "tracked"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	logger.Info("Starting Analytics Service", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
