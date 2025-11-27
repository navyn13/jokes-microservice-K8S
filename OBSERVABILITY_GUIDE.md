# Observability Guide - OpenTelemetry + SigNoz

This guide explains how to use SigNoz to observe your microservices and practice OpenTelemetry concepts.

## OpenTelemetry Implementation

### What's Instrumented

Every service includes:

#### 1. Automatic Instrumentation (via otelgin)
```go
r.Use(otelgin.Middleware("service-name"))
```

This automatically captures:
- HTTP request/response
- Request method, path, status code
- Request duration
- Errors and exceptions

#### 2. Manual Instrumentation

**Creating Spans:**
```go
ctx, span := tracer.Start(ctx, "operation-name")
defer span.End()
```

**Adding Attributes:**
```go
span.SetAttributes(
    attribute.String("key", "value"),
    attribute.Int("count", 123),
)
```

**Recording Metrics:**
```go
counter.Add(ctx, 1,
    metric.WithAttributes(
        attribute.String("label", "value"),
    ),
)
```

### Trace Context Propagation

#### How It Works

```
Client Request
    │
    ├─ Trace ID: abc123...
    └─ Parent Span: (none)
        │
        ▼
    API Gateway
        │
        ├─ Trace ID: abc123...
        ├─ Span ID: xyz789...
        └─ Parent: (none)
            │
            ├─ W3C traceparent header
            ├─ traceparent: 00-abc123-xyz789-01
            │
            ▼
        Jokes Service
            │
            ├─ Trace ID: abc123... (same!)
            ├─ Span ID: def456...
            └─ Parent: xyz789... (gateway)
                │
                ├─ traceparent: 00-abc123-def456-01
                │
                ▼
            Analytics Service
                │
                ├─ Trace ID: abc123... (same!)
                ├─ Span ID: ghi789...
                └─ Parent: def456... (jokes)
```

## SigNoz UI Guide

### 1. Traces Tab 📊

#### Finding Traces

1. **Filter by Service:**
   - Click "Service" dropdown
   - Select `api-gateway`, `jokes-service`, etc.

2. **Filter by Operation:**
   - Select specific HTTP operations
   - Example: `GET /api/v1/joke`

3. **Filter by Status:**
   - Success (2xx)
   - Client Error (4xx)
   - Server Error (5xx)

4. **Time Range:**
   - Last 15 minutes (default)
   - Custom range

#### Analyzing a Trace

Click on any trace to see:

```
Trace Timeline:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│ api-gateway (45ms)
│ ├─ proxy_to_/api/v1/joke (40ms)
│ │  └─ jokes-service::GET /api/v1/joke (38ms)
│ │     ├─ getRandomJoke (30ms)
│ │     └─ notifyAnalytics (3ms)
│ │        └─ analytics-service::POST /internal/track (2ms)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**What to Look For:**
- ⏱️ **Latency Hotspots**: Which service/operation is slowest?
- 🔴 **Errors**: Red spans indicate failures
- 🔗 **Dependencies**: How services call each other
- 📊 **Timing**: Sequential vs. parallel operations

#### Trace Details

Each span shows:
- **Duration**: Time spent
- **Attributes**: Custom data (joke content, user ID, etc.)
- **Events**: Important moments
- **Status**: Ok, Error
- **Span Kind**: Internal, Server, Client

### 2. Service Map 🗺️

Visual representation of service dependencies.

**Example Map:**
```
        ┌─────────────┐
        │   Client    │
        └──────┬──────┘
               │
        ┌──────▼──────────┐
        │  API Gateway    │
        └──────┬──────────┘
               │
       ┌───────┼────────┐
       │       │        │
┌──────▼──┐ ┌─▼────┐ ┌─▼─────────┐
│  Jokes  │ │ User │ │ Analytics │
└──────┬──┘ └──────┘ └─────▲─────┘
       │                    │
       └────────────────────┘
```

**Information Shown:**
- Request rate (requests/second)
- Error rate (%)
- P99 latency
- Service health

**Use Cases:**
- Understand service architecture
- Identify bottlenecks
- Spot error propagation
- Plan optimizations

### 3. Metrics Tab 📈

#### Pre-Built Metrics

System automatically collects:
- `http_server_request_count`
- `http_server_request_duration`
- `http_server_active_requests`

#### Custom Business Metrics

Our services export:

**Jokes Service:**
- `jokes_served_total` - Counter
- `jokes_latency_milliseconds` - Histogram

**Analytics Service:**
- `analytics_tracks_total` - Counter

**User Service:**
- `user_favorites_added_total` - Counter

#### Creating Queries

**Example 1: Request Rate**
```
Query: http_server_request_count
Aggregation: Rate
Group By: service_name, http_method
```

**Example 2: Latency Percentiles**
```
Query: http_server_request_duration
Aggregation: P95, P99
Group By: service_name
```

**Example 3: Error Rate**
```
Query: http_server_request_count{http_status_code=~"5.."}
Aggregation: Rate
```

#### Creating Dashboards

1. Click "Dashboards" → "New Dashboard"
2. Add panels:
   - **Requests/sec** - Line chart
   - **Latency** - Line chart (P50, P95, P99)
   - **Error Rate** - Line chart
   - **Active Services** - Stat
   - **Jokes Served** - Counter

3. Save and share!

### 4. Logs Tab 📝

#### Log Structure

Each log entry includes:
```json
{
  "timestamp": "2025-11-27T10:30:45.123Z",
  "level": "info",
  "msg": "Joke requested",
  "trace_id": "abc123...",
  "span_id": "xyz789...",
  "service": "jokes-service",
  "client_ip": "192.168.1.100"
}
```

#### Searching Logs

**By Trace ID:**
```
trace_id = "abc123..."
```

**By Service:**
```
service = "jokes-service"
```

**By Level:**
```
level = "error"
```

**Combined:**
```
service = "api-gateway" AND level = "error"
```

#### Log-Trace Correlation

1. Find an error in logs
2. Click on the trace ID
3. Opens the full trace
4. See what happened across all services

### 5. Alerts 🔔

#### Setting Up Alerts

**Example: High Error Rate**
```
Query: http_server_request_count{http_status_code=~"5.."}
Condition: Rate > 10 requests/sec
Duration: 5 minutes
Action: Slack notification
```

**Example: High Latency**
```
Query: http_server_request_duration
Condition: P95 > 1000ms
Duration: 5 minutes
Action: Email notification
```

## Common Observability Patterns

### Pattern 1: Finding Slow Requests

1. Go to **Traces**
2. Sort by **Duration** (descending)
3. Click on slowest trace
4. Identify which span is slow
5. Check span attributes for clues

### Pattern 2: Debugging Errors

1. Go to **Traces**
2. Filter by **Status: Error**
3. Click on failed trace
4. Check error message and stack trace
5. Look at preceding spans for context
6. Search **Logs** by trace ID for details

### Pattern 3: Analyzing Service Dependencies

1. Go to **Service Map**
2. Click on a service
3. See incoming/outgoing requests
4. Check error rates per connection
5. Identify bottleneck services

### Pattern 4: Monitoring Business Metrics

1. Go to **Metrics**
2. Query custom metrics:
   - `jokes_served_total`
   - `user_favorites_added_total`
3. Create time-series charts
4. Set up alerts for anomalies

### Pattern 5: Correlating Logs and Traces

1. See error in application
2. Go to **Logs**
3. Find error log entry
4. Click on trace ID
5. See full distributed trace
6. Understand root cause

## Practice Exercises

### Exercise 1: Trace a Request

1. Send a request:
   ```bash
   curl http://localhost:8000/api/v1/joke
   ```

2. In SigNoz, find the trace
3. Answer:
   - How long did the total request take?
   - Which service was slowest?
   - How many services were involved?
   - Was Analytics called synchronously or asynchronously?

### Exercise 2: Create a Dashboard

Create a dashboard with:
1. Total requests/second (all services)
2. P95 latency per service
3. Error rate
4. Jokes served counter
5. Active services

### Exercise 3: Find Performance Issues

1. Run load test:
   ```bash
   ./scripts/load-test.sh http://localhost:8000 1000
   ```

2. In SigNoz:
   - Find slowest traces
   - Identify bottleneck services
   - Check resource utilization
   - Propose optimizations

### Exercise 4: Debug an Error

1. Modify jokes service to randomly fail:
   ```go
   if rand.Intn(10) == 0 {
       c.JSON(500, gin.H{"error": "Random failure"})
       return
   }
   ```

2. Generate requests
3. Find failed traces in SigNoz
4. Analyze error patterns
5. Correlate with logs

### Exercise 5: Monitor Custom Metrics

1. Add a new metric to jokes service:
   ```go
   jokeLengthHistogram.Record(ctx, float64(len(joke)))
   ```

2. Query in SigNoz Metrics
3. Create visualization
4. Set up alert for unusually long jokes

## Advanced Topics

### Sampling Strategies

**Always Sample (Current):**
```go
sdktrace.WithSampler(sdktrace.AlwaysSample())
```

**Sample 10%:**
```go
sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1))
```

**Parent-based:**
```go
sdktrace.WithSampler(sdktrace.ParentBased(
    sdktrace.TraceIDRatioBased(0.1),
))
```

### Custom Span Events

```go
span.AddEvent("Cache miss", trace.WithAttributes(
    attribute.String("key", "joke:123"),
))
```

### Baggage Propagation

```go
// Set baggage
ctx = baggage.ContextWithValues(ctx,
    attribute.String("user.id", "user123"),
)

// Read baggage in another service
bag := baggage.FromContext(ctx)
userID := bag.Member("user.id").Value()
```

### Context Propagation in Goroutines

```go
go func() {
    // Create new context from parent
    ctx, span := tracer.Start(context.Background(), "async-work")
    defer span.End()
    
    // Do work with traced context
    doWork(ctx)
}()
```

## Troubleshooting Observability

### Traces Not Appearing

**Check:**
1. OTEL Collector running?
   ```bash
   docker-compose ps otel-collector
   ```

2. Correct endpoint?
   ```bash
   echo $OTEL_EXPORTER_OTLP_ENDPOINT
   ```

3. Collector receiving data?
   ```bash
   docker-compose logs otel-collector | grep "TracesExporter"
   ```

### Metrics Not Updating

**Check:**
1. Metrics being recorded in code?
2. Metric pipeline configured in collector?
3. ClickHouse connection working?

### Logs Missing Trace IDs

**Ensure:**
1. Using span context for logging:
   ```go
   span := trace.SpanFromContext(ctx)
   logger.Info("Message",
       zap.String("trace_id", span.SpanContext().TraceID().String()),
   )
   ```

## Resources

### Official Docs
- [OpenTelemetry Docs](https://opentelemetry.io/docs/)
- [SigNoz Docs](https://signoz.io/docs/)
- [OTEL Go SDK](https://pkg.go.dev/go.opentelemetry.io/otel)

### Tutorials
- [Getting Started with OTEL](https://opentelemetry.io/docs/instrumentation/go/getting-started/)
- [SigNoz Tutorials](https://signoz.io/blog/)

### Community
- [OTEL Slack](https://cloud-native.slack.com/archives/C01NR1YLSE7)
- [SigNoz Slack](https://signoz.io/slack)

## Next Steps

1. ✅ Start services and generate traffic
2. ✅ Explore all SigNoz tabs
3. ✅ Complete practice exercises
4. ⬜ Create custom dashboards
5. ⬜ Set up alerts
6. ⬜ Add more instrumentation
7. ⬜ Experiment with sampling
8. ⬜ Practice debugging with traces

Happy observing! 🔭

