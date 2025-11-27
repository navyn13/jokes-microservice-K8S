# Jokes Microservice with OpenTelemetry & SigNoz

A complete microservices architecture demonstrating distributed tracing, metrics, and logging with OpenTelemetry and SigNoz.

## Architecture

This project consists of 4 microservices:

1. **API Gateway** (Port 8080) - Entry point for all requests
2. **Jokes Service** (Port 8081) - Returns random programming jokes
3. **Analytics Service** (Port 8082) - Tracks joke request statistics
4. **User Service** (Port 8083) - Manages user preferences and favorites

All services are instrumented with OpenTelemetry to send:
- **Traces**: Distributed request tracing across services
- **Metrics**: Custom business metrics and system metrics
- **Logs**: Structured logs with trace context

## Observability Stack

- **OpenTelemetry Collector**: Receives and processes telemetry data
- **SigNoz**: Complete observability platform
  - Frontend UI (Port 3301)
  - Query Service (Port 8080)
  - ClickHouse for data storage

## Prerequisites

### For Local Development (Docker Compose)
- Docker and Docker Compose
- Go 1.22+ (for local builds)
- curl and jq (for testing)

### For Kubernetes Deployment
- Kubernetes cluster (minikube, kind, or cloud provider)
- kubectl configured
- Docker for building images

## Quick Start

### Local Development with Docker Compose

1. **Start all services:**
```bash
make local-up
```

2. **Access the services:**
- SigNoz UI: http://localhost:3301
- API Gateway: http://localhost:8000

3. **Test the APIs:**
```bash
make test
```

4. **View logs:**
```bash
make logs
```

5. **Stop services:**
```bash
make local-down
```

### Kubernetes Deployment

1. **Build and push Docker images:**
```bash
make docker-build
make docker-push
```

2. **Deploy to Kubernetes:**
```bash
make k8s-deploy
```

3. **Check deployment status:**
```bash
kubectl get pods -n default
kubectl get pods -n platform
```

4. **Access services:**
```bash
# For minikube
minikube service api-gateway -n default
minikube service signoz-frontend -n platform

# Or use port-forward
kubectl port-forward -n default svc/api-gateway 8000:80
kubectl port-forward -n platform svc/signoz-frontend 3301:3301
```

5. **Delete deployment:**
```bash
make k8s-delete
```

## API Endpoints

### API Gateway (http://localhost:8000)

- `GET /healthz` - Health check
- `GET /api/v1/joke` - Get a random joke
- `POST /api/v1/favorite` - Add a favorite joke
  ```bash
  curl -X POST http://localhost:8000/api/v1/favorite \
    -H "Content-Type: application/json" \
    -d '{"joke":"Why do programmers hate nature?","user_id":"user123"}'
  ```
- `GET /api/v1/stats` - Get analytics statistics

### Direct Service Access (Docker Compose)

- Jokes Service: http://localhost:8081/api/v1/joke
- Analytics Service: http://localhost:8082/api/v1/stats
- User Service: http://localhost:8083/api/v1/favorites

## Observability Features

### Traces
- End-to-end request tracing across all microservices
- Service dependencies visualization
- Latency analysis
- Error tracking

### Metrics
- HTTP request counts and latency
- Custom business metrics:
  - `jokes.served` - Total jokes served
  - `analytics.tracks` - Analytics events tracked
  - `user.favorites.added` - Favorites added
- Resource utilization

### Logs
- Structured JSON logs
- Trace ID correlation
- Log levels: Info, Warn, Error
- Contextual information

## SigNoz Features to Explore

1. **Traces Tab**: View distributed traces
   - Filter by service, operation, or status
   - See complete request flow
   - Identify bottlenecks

2. **Metrics Tab**: Monitor service metrics
   - Request rates and latency
   - Custom business metrics
   - Service health

3. **Logs Tab**: Search and analyze logs
   - Filter by trace ID, service, severity
   - Correlate logs with traces
   - Full-text search

4. **Service Map**: Visualize service dependencies
   - Request flow between services
   - Error rates per service
   - Latency at each hop

## Project Structure

```
.
├── services/
│   ├── gateway/          # API Gateway service
│   ├── jokes/            # Jokes service
│   ├── analytics/        # Analytics service
│   └── user/             # User service
├── k8s/                  # Kubernetes manifests
│   ├── namespace.yaml
│   ├── signoz.yaml       # SigNoz deployment
│   ├── gateway.yaml
│   ├── jokes-service.yaml
│   ├── analytics-service.yaml
│   └── user-service.yaml
├── docker-compose.yaml   # Local development
├── otel-collector-config.yaml
├── Makefile              # Build and deployment commands
└── README.md
```

## Development

### Building Locally

```bash
# Build all services
make build-all

# Build specific service
cd services/gateway && go build -o gateway-server .
```

### Running Tests

```bash
# Run automated tests
make test

# Manual testing
curl http://localhost:8000/api/v1/joke
```

### Viewing Telemetry Data

1. Open SigNoz UI at http://localhost:3301
2. Navigate to:
   - **Traces** → Filter by service name
   - **Metrics** → Create custom dashboards
   - **Logs** → Search and filter logs

## Troubleshooting

### Services not starting
```bash
docker-compose logs <service-name>
kubectl logs -n default <pod-name>
```

### Cannot connect to SigNoz
- Ensure OTEL collector is running
- Check endpoint configuration in environment variables
- Verify network connectivity

### Traces not appearing
- Wait 30-60 seconds for data to appear
- Check OTEL collector logs
- Verify service instrumentation

## Configuration

### Environment Variables

Each service accepts:
- `PORT` - Service port
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint
- Service-specific URLs for inter-service communication

### OpenTelemetry Configuration

See `otel-collector-config.yaml` for collector configuration:
- Receivers (OTLP gRPC/HTTP)
- Processors (batch, memory_limiter, resource)
- Exporters (logging, otlp, prometheus)

## Performance Tuning

### Horizontal Pod Autoscaling (K8s)
- API Gateway: 2-10 replicas (70% CPU)
- Jokes Service: 3-10 replicas (60% CPU)
- Analytics Service: 2-8 replicas (70% CPU)
- User Service: 2-8 replicas (70% CPU)

### Resource Limits
Configured in Kubernetes manifests based on service requirements.

## Learning Resources

- [OpenTelemetry Docs](https://opentelemetry.io/docs/)
- [SigNoz Docs](https://signoz.io/docs/)
- [Go OTEL Instrumentation](https://opentelemetry.io/docs/instrumentation/go/)
- [Kubernetes HPA](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)

## License

MIT License - Feel free to use this for learning and experimentation!

## Contributing

This is a learning project. Feel free to:
- Add more microservices
- Implement additional observability features
- Create dashboards in SigNoz
- Add alerting rules
- Experiment with sampling strategies

Happy observability practice! 🚀

