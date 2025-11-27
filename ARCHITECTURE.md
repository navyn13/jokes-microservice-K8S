# Architecture Documentation

## System Overview

This microservices architecture demonstrates a production-ready observability setup using OpenTelemetry and SigNoz.

```
┌─────────────────────────────────────────────────────────────────┐
│                         External Client                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ HTTP Requests
                             ▼
                    ┌────────────────────┐
                    │   API Gateway      │
                    │   (Port 8080)      │
                    └────────┬───────────┘
                             │
                 ┌───────────┼───────────┐
                 │           │           │
          ┌──────▼────┐ ┌───▼─────┐ ┌──▼──────────┐
          │  Jokes    │ │  User   │ │ Analytics   │
          │  Service  │ │ Service │ │  Service    │
          │ (8081)    │ │ (8083)  │ │  (8082)     │
          └──────┬────┘ └────┬────┘ └──────┬──────┘
                 │           │             │
                 └───────────┴─────────────┘
                             │
                             │ OTLP (gRPC)
                             ▼
                    ┌────────────────────┐
                    │  OTEL Collector    │
                    │   (Port 4317)      │
                    └────────┬───────────┘
                             │
                             │ Export Data
                             ▼
                    ┌────────────────────┐
                    │   ClickHouse DB    │
                    │   (Port 9000)      │
                    └────────┬───────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
         ┌──────▼────────┐      ┌────────▼────────┐
         │ Query Service │      │  SigNoz UI      │
         │  (Port 8080)  │◄─────┤  (Port 3301)    │
         └───────────────┘      └─────────────────┘
```

## Service Responsibilities

### 1. API Gateway
**Purpose**: Single entry point for all client requests

**Responsibilities**:
- Route requests to appropriate backend services
- Request validation
- Trace context propagation
- Load balancing (when multiple replicas)

**Key Features**:
- OpenTelemetry instrumentation for incoming requests
- Automatic trace propagation to downstream services
- Custom metrics for gateway-specific operations
- Structured logging with trace IDs

**Endpoints**:
- `GET /healthz` - Health check
- `GET /api/v1/joke` - Proxy to Jokes Service
- `POST /api/v1/favorite` - Proxy to User Service
- `GET /api/v1/stats` - Proxy to Analytics Service

### 2. Jokes Service
**Purpose**: Provides random programming jokes

**Responsibilities**:
- Maintain joke database (in-memory)
- Select and return random jokes
- Notify analytics service asynchronously

**Key Features**:
- Custom metric: `jokes.served` counter
- Custom metric: `jokes.latency` histogram
- Async communication with Analytics Service
- Simulated processing latency for demo

**Data Flow**:
1. Receive request from Gateway
2. Select random joke from collection
3. Notify Analytics Service (async)
4. Return joke to client

### 3. Analytics Service
**Purpose**: Track and analyze joke request patterns

**Responsibilities**:
- Record joke request events
- Maintain statistics (in-memory)
- Provide analytics data

**Key Features**:
- Custom metric: `analytics.tracks` counter
- In-memory statistics storage
- Internal tracking endpoint
- Real-time stats calculation

**Data Tracked**:
- Total requests
- Total jokes served
- Last update timestamp
- Uptime

### 4. User Service
**Purpose**: Manage user preferences and favorites

**Responsibilities**:
- Store user favorite jokes
- Manage user preferences
- Provide favorite joke listings

**Key Features**:
- Custom metric: `user.favorites.added` counter
- In-memory storage (can be extended to DB)
- User-specific data filtering
- Timestamp tracking

**Data Model**:
```go
type Favorite struct {
    ID        string
    Joke      string
    UserID    string
    CreatedAt time.Time
}
```

## Observability Architecture

### OpenTelemetry Instrumentation

Each service includes:

1. **Trace Provider**
   - Configured with OTLP gRPC exporter
   - Sends to SigNoz OTEL Collector
   - Always sampling (for demo purposes)
   - Resource attributes (service name, version, environment)

2. **Metrics Provider**
   - Custom business metrics
   - HTTP metrics (via otelgin middleware)
   - Periodic export to collector

3. **Logging**
   - Structured JSON logs with Zap
   - Trace ID injection for correlation
   - ISO 8601 timestamps
   - Multiple severity levels

### Trace Propagation

```
Client Request
    │
    ▼
API Gateway [Trace ID: abc123]
    │
    ├─► Jokes Service [Trace ID: abc123, Span ID: xyz789]
    │       │
    │       └─► Analytics Service [Trace ID: abc123, Span ID: def456]
    │
    └─► User Service [Trace ID: abc123, Span ID: ghi123]
```

**Key Points**:
- Trace ID maintained across all services
- Each service creates child spans
- Context propagated via HTTP headers
- W3C Trace Context standard

### Custom Metrics

#### Gateway Service
- `http.server.request_count` - Total requests
- `http.server.request_duration` - Request latency

#### Jokes Service
- `jokes.served` - Number of jokes served
- `jokes.latency` - Joke retrieval time

#### Analytics Service
- `analytics.tracks` - Events tracked

#### User Service
- `user.favorites.added` - Favorites added

### Log Correlation

All logs include:
- `timestamp` - ISO 8601 format
- `level` - Log severity
- `msg` - Log message
- `trace_id` - For correlation with traces
- Service-specific context

Example:
```json
{
  "timestamp": "2025-11-27T10:30:45.123Z",
  "level": "info",
  "msg": "Joke requested",
  "trace_id": "abc123...",
  "client_ip": "192.168.1.100"
}
```

## Communication Patterns

### Synchronous Communication
- API Gateway ↔ All Services (HTTP/REST)
- Uses context propagation for tracing
- Timeout: 10 seconds

### Asynchronous Communication
- Jokes Service → Analytics Service
- Fire-and-forget pattern
- Non-blocking operations
- Timeout: 2 seconds

## Data Flow Examples

### Example 1: Get Joke Request

```
1. Client → API Gateway: GET /api/v1/joke
   ├─ Gateway creates trace span
   └─ Log: "Joke requested"

2. API Gateway → Jokes Service: GET /api/v1/joke
   ├─ Propagate trace context
   └─ Jokes Service creates child span

3. Jokes Service:
   ├─ Select random joke
   ├─ Record metric: jokes.served++
   ├─ Log: "Joke retrieved"
   └─ Async notify Analytics Service

4. Jokes Service → Analytics Service: POST /internal/track
   ├─ Fire-and-forget
   └─ Analytics records event

5. Jokes Service → API Gateway: Response with joke
   └─ Include trace headers

6. API Gateway → Client: Response with joke
   ├─ Record latency metric
   └─ Log: "Request completed"
```

### Example 2: Add Favorite

```
1. Client → API Gateway: POST /api/v1/favorite
2. API Gateway → User Service: POST /api/v1/favorite
3. User Service:
   ├─ Validate request
   ├─ Store favorite
   ├─ Record metric: user.favorites.added++
   └─ Return success
4. API Gateway → Client: Response
```

## Deployment Architecture

### Docker Compose (Local Development)
- All services in single bridge network
- Direct service-to-service communication
- Persistent volumes for ClickHouse

### Kubernetes (Production)
- Namespace separation:
  - `default` - Application services
  - `platform` - Observability stack
- Service discovery via DNS
- Horizontal Pod Autoscaling
- Resource limits and requests

## Scaling Strategy

### Horizontal Scaling
- API Gateway: 2-10 replicas (CPU-based)
- Jokes Service: 3-10 replicas (CPU-based)
- Analytics Service: 2-8 replicas (CPU-based)
- User Service: 2-8 replicas (CPU-based)

### Load Distribution
- Kubernetes Service (ClusterIP) for internal services
- Round-robin load balancing
- Health checks for readiness/liveness

## Security Considerations

### In This Demo
- No authentication (for simplicity)
- Insecure TLS to OTEL collector
- In-memory data storage

### Production Recommendations
- Add API authentication (JWT, OAuth)
- Enable TLS for all communications
- Use proper database with encryption
- Implement rate limiting
- Add network policies in K8s
- Secret management (Vault, K8s Secrets)

## Performance Characteristics

### Expected Latency
- API Gateway: < 50ms
- Jokes Service: 10-60ms (with simulated delay)
- Analytics Service: < 20ms
- User Service: < 30ms

### Throughput
- API Gateway: ~1000 req/s per replica
- Backend Services: ~500 req/s per replica

### Resource Usage
- Gateway: 128-256 MB RAM
- Services: 64-128 MB RAM
- OTEL Collector: 512 MB - 1 GB RAM
- ClickHouse: 512 MB - 4 GB RAM

## Observability Best Practices Demonstrated

1. **Distributed Tracing**
   - End-to-end request tracking
   - Service dependency mapping
   - Latency breakdown by service

2. **Metrics Collection**
   - RED metrics (Rate, Errors, Duration)
   - Custom business metrics
   - Resource utilization

3. **Log Aggregation**
   - Centralized logging
   - Structured logs
   - Trace correlation

4. **Service Discovery**
   - DNS-based discovery in K8s
   - Environment-based configuration

5. **Health Checks**
   - Readiness probes
   - Liveness probes
   - Health endpoints

## Future Enhancements

1. **Persistence**
   - Add PostgreSQL/MongoDB
   - Migrate from in-memory storage

2. **Caching**
   - Add Redis for favorites
   - Cache jokes in gateway

3. **Authentication**
   - JWT-based auth
   - User management

4. **Advanced Observability**
   - Custom dashboards in SigNoz
   - Alert rules
   - SLO tracking

5. **Resilience**
   - Circuit breakers
   - Retry logic
   - Fallback responses

6. **Testing**
   - Unit tests
   - Integration tests
   - Load testing suite

