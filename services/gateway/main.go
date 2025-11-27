// API Gateway Service - Entry point for all microservices
// Routes:
//   GET /healthz          -> health check
//   GET /api/v1/joke      -> get random joke (proxies to jokes-service)
//   POST /api/v1/favorite -> add favorite joke (proxies to user-service)
//   GET /api/v1/stats     -> get analytics (proxies to analytics-service)

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	logger         *zap.Logger
	tracer         trace.Tracer
	meter          metric.Meter
	requestCount   metric.Int64Counter
	requestLatency metric.Float64Histogram
)

func initLogger() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	logger, err = config.Build()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
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
			semconv.ServiceName("api-gateway"),
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

	tracer = tp.Tracer("api-gateway")

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}
}

func initMetrics() {
	meter = otel.Meter("api-gateway")

	var err error
	requestCount, err = meter.Int64Counter(
		"http.server.request_count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		logger.Fatal("Failed to create request counter", zap.Error(err))
	}

	requestLatency, err = meter.Float64Histogram(
		"http.server.request_duration",
		metric.WithDescription("HTTP request latency"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		logger.Fatal("Failed to create latency histogram", zap.Error(err))
	}
}

func proxyRequest(c *gin.Context, serviceURL, path string) {
	ctx := c.Request.Context()

	// Create child span for proxy request
	_, span := tracer.Start(ctx, fmt.Sprintf("proxy_to_%s", path))
	defer span.End()

	start := time.Now()

	// Build target URL
	targetURL := fmt.Sprintf("http://%s%s", serviceURL, path)

	logger.Info("Proxying request",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("target", targetURL),
		zap.String("method", c.Request.Method),
	)

	// Create new request
	req, err := http.NewRequestWithContext(ctx, c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		logger.Error("Failed to create proxy request",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Propagate headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to proxy request",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.Error(err),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Service unavailable"})
		return
	}
	defer resp.Body.Close()

	// Record metrics
	duration := time.Since(start).Milliseconds()
	requestLatency.Record(ctx, float64(duration),
		metric.WithAttributes(
			attribute.String("service", serviceURL),
			attribute.Int("status_code", resp.StatusCode),
		),
	)

	// Copy response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	logger.Info("Proxy request completed",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Int("status_code", resp.StatusCode),
		zap.Int64("duration_ms", duration),
	)

	c.Data(resp.StatusCode, "application/json", body)
}

func main() {
	initLogger()
	defer logger.Sync()

	shutdown := initTracer()
	defer shutdown()

	initMetrics()

	r := gin.Default()
	r.Use(otelgin.Middleware("api-gateway"))

	// Middleware for metrics
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start).Milliseconds()
		requestCount.Add(c.Request.Context(), 1,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("path", c.Request.URL.Path),
				attribute.Int("status_code", c.Writer.Status()),
			),
		)
		requestLatency.Record(c.Request.Context(), float64(duration),
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("path", c.Request.URL.Path),
			),
		)
	})

	// Health check
	r.GET("/healthz", func(c *gin.Context) {
		logger.Info("Health check")
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "api-gateway",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Proxy to jokes service
	r.GET("/api/v1/joke", func(c *gin.Context) {
		jokesService := os.Getenv("JOKES_SERVICE_URL")
		if jokesService == "" {
			jokesService = "jokes-service.default.svc.cluster.local"
		}
		proxyRequest(c, jokesService, "/api/v1/joke")
	})

	// Proxy to user service
	r.POST("/api/v1/favorite", func(c *gin.Context) {
		userService := os.Getenv("USER_SERVICE_URL")
		if userService == "" {
			userService = "user-service.default.svc.cluster.local"
		}
		proxyRequest(c, userService, "/api/v1/favorite")
	})

	// Proxy to analytics service
	r.GET("/api/v1/stats", func(c *gin.Context) {
		analyticsService := os.Getenv("ANALYTICS_SERVICE_URL")
		if analyticsService == "" {
			analyticsService = "analytics-service.default.svc.cluster.local"
		}
		proxyRequest(c, analyticsService, "/api/v1/stats")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Starting API Gateway", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
