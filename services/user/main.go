// User Service - Manages user preferences and favorites
// Routes:
//   GET /healthz              -> health check
//   POST /api/v1/favorite     -> add a favorite joke
//   GET /api/v1/favorites     -> get all favorite jokes

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
	logger          *zap.Logger
	tracer          trace.Tracer
	meter           metric.Meter
	favoritesCount  metric.Int64Counter
	
	// In-memory storage (in production, use a database)
	favorites      []Favorite
	favoritesMutex sync.RWMutex
)

type Favorite struct {
	ID        string    `json:"id"`
	Joke      string    `json:"joke"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type FavoriteRequest struct {
	Joke   string `json:"joke" binding:"required"`
	UserID string `json:"user_id" binding:"required"`
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
			semconv.ServiceName("user-service"),
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

	tracer = tp.Tracer("user-service")

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}
}

func initMetrics() {
	meter = otel.Meter("user-service")

	var err error
	favoritesCount, err = meter.Int64Counter(
		"user.favorites.added",
		metric.WithDescription("Number of favorites added"),
		metric.WithUnit("{favorite}"),
	)
	if err != nil {
		logger.Fatal("Failed to create favorites counter", zap.Error(err))
	}
}

func addFavorite(ctx context.Context, req FavoriteRequest) Favorite {
	_, span := tracer.Start(ctx, "addFavorite")
	defer span.End()

	favoritesMutex.Lock()
	defer favoritesMutex.Unlock()

	fav := Favorite{
		ID:        time.Now().Format("20060102150405"),
		Joke:      req.Joke,
		UserID:    req.UserID,
		CreatedAt: time.Now(),
	}

	favorites = append(favorites, fav)
	favoritesCount.Add(ctx, 1)

	span.SetAttributes(
		attribute.String("favorite.id", fav.ID),
		attribute.String("favorite.user_id", fav.UserID),
		attribute.Int("favorites.total", len(favorites)),
	)

	logger.Info("Favorite added",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("favorite_id", fav.ID),
		zap.String("user_id", fav.UserID),
		zap.Int("total_favorites", len(favorites)),
	)

	return fav
}

func getFavorites(ctx context.Context, userID string) []Favorite {
	_, span := tracer.Start(ctx, "getFavorites")
	defer span.End()

	favoritesMutex.RLock()
	defer favoritesMutex.RUnlock()

	var userFavorites []Favorite
	for _, fav := range favorites {
		if userID == "" || fav.UserID == userID {
			userFavorites = append(userFavorites, fav)
		}
	}

	span.SetAttributes(
		attribute.String("query.user_id", userID),
		attribute.Int("results.count", len(userFavorites)),
	)

	logger.Info("Favorites retrieved",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("user_id", userID),
		zap.Int("count", len(userFavorites)),
	)

	return userFavorites
}

func main() {
	initLogger()
	defer logger.Sync()

	shutdown := initTracer()
	defer shutdown()

	initMetrics()

	favorites = make([]Favorite, 0)

	r := gin.Default()
	r.Use(otelgin.Middleware("user-service"))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "user-service",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	r.POST("/api/v1/favorite", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)

		var req FavoriteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.Error("Invalid request",
				zap.String("trace_id", span.SpanContext().TraceID().String()),
				zap.Error(err),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		logger.Info("Favorite request received",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("user_id", req.UserID),
		)

		favorite := addFavorite(ctx, req)
		c.JSON(http.StatusCreated, favorite)
	})

	r.GET("/api/v1/favorites", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)

		userID := c.Query("user_id")

		logger.Info("Favorites list requested",
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("user_id", userID),
		)

		userFavorites := getFavorites(ctx, userID)
		c.JSON(http.StatusOK, gin.H{
			"favorites": userFavorites,
			"count":     len(userFavorites),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	logger.Info("Starting User Service", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

