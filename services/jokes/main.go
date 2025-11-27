// Jokes Service - Returns random jokes
// Routes:
//   GET /healthz         -> health check
//   GET /api/v1/joke     -> returns a random joke

package main

import (
	"context"
	"math/rand"
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
	logger      *zap.Logger
	tracer      trace.Tracer
	meter       metric.Meter
	jokesServed metric.Int64Counter
	jokeLatency metric.Float64Histogram
)

var jokes = []string{
	"Why do programmers hate nature? It has too many bugs.",
	"I told my computer I needed a break, and it said 'No problem — I'll go to sleep.'",
	"Debugging is like being the detective in a crime movie where you are also the murderer.",
	"Why do Java developers wear glasses? Because they don't C#.",
	"To understand recursion, you must first understand recursion.",
	"There are 10 types of people: those who understand binary and those who don't.",
	"Why did the programmer quit? Because they didn't get arrays.",
	"A SQL query walks into a bar, walks up to two tables and asks: 'Can I join you?'",
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
			semconv.ServiceName("jokes-service"),
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

	tracer = tp.Tracer("jokes-service")

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}
}

func initMetrics() {
	meter = otel.Meter("jokes-service")

	var err error
	jokesServed, err = meter.Int64Counter(
		"jokes.served",
		metric.WithDescription("Total number of jokes served"),
		metric.WithUnit("{joke}"),
	)
	if err != nil {
		logger.Fatal("Failed to create jokes counter", zap.Error(err))
	}

	jokeLatency, err = meter.Float64Histogram(
		"jokes.latency",
		metric.WithDescription("Joke retrieval latency"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		logger.Fatal("Failed to create latency histogram", zap.Error(err))
	}
}

func getRandomJoke(ctx context.Context) string {
	_, span := tracer.Start(ctx, "getRandomJoke")
	defer span.End()

	start := time.Now()

	// Simulate some processing
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)))

	joke := jokes[rand.Intn(len(jokes))]

	span.SetAttributes(
		attribute.String("joke.content", joke),
		attribute.Int("joke.length", len(joke)),
	)

	duration := time.Since(start).Milliseconds()
	jokeLatency.Record(ctx, float64(duration))

	logger.Info("Joke retrieved",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Int("joke_length", len(joke)),
		zap.Int64("duration_ms", duration),
	)

	return joke
}

func notifyAnalytics(ctx context.Context, joke string) {
	_, span := tracer.Start(ctx, "notifyAnalytics")
	defer span.End()

	analyticsService := os.Getenv("ANALYTICS_SERVICE_URL")
	if analyticsService == "" {
		analyticsService = "analytics-service.default.svc.cluster.local"
	}

	// Make async call to analytics service
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		req, _ := http.NewRequest("POST", "http://"+analyticsService+"/internal/track", nil)
		req.Header.Set("X-Joke-Length", string(rune(len(joke))))

		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

		resp, err := client.Do(req)
		if err != nil {
			logger.Warn("Failed to notify analytics", zap.Error(err))
			return
		}
		defer resp.Body.Close()
	}()
}

func main() {
	initLogger()
	defer logger.Sync()

	shutdown := initTracer()
	defer shutdown()

	initMetrics()

	r := gin.Default()
	r.Use(otelgin.Middleware("jokes-service"))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "jokes-service",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/api/v1/joke", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)

		logger.Info("Joke requested",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("client_ip", c.ClientIP()),
		)

		joke := getRandomJoke(ctx)

		// Increment counter
		jokesServed.Add(ctx, 1)

		// Notify analytics asynchronously
		notifyAnalytics(ctx, joke)

		c.JSON(http.StatusOK, gin.H{
			"joke":      joke,
			"service":   "jokes-service",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	logger.Info("Starting Jokes Service", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
