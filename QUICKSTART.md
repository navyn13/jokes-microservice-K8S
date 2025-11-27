# Quick Start Guide

Get up and running with the Jokes Microservices in 5 minutes!

## Prerequisites Check

```bash
# Check Docker
docker --version

# Check Docker Compose
docker-compose --version

# Check Go (optional, for local builds)
go version

# Check kubectl (for K8s deployment)
kubectl version --client
```

## Option 1: Local Development (Recommended for Learning)

### Step 1: Start Services

```bash
make local-up
```

This will start:
- 4 microservices (Gateway, Jokes, Analytics, User)
- SigNoz observability platform
- OpenTelemetry Collector

Wait ~60 seconds for all services to be ready.

### Step 2: Test the APIs

```bash
./scripts/test-apis.sh
```

Or manually:

```bash
# Get a joke
curl http://localhost:8000/api/v1/joke

# Add to favorites
curl -X POST http://localhost:8000/api/v1/favorite \
  -H "Content-Type: application/json" \
  -d '{"joke":"Why do programmers hate nature?","user_id":"user123"}'

# Get statistics
curl http://localhost:8000/api/v1/stats
```

### Step 3: Explore SigNoz

Open http://localhost:3301 in your browser.

#### View Traces
1. Click **"Traces"** in the left sidebar
2. You'll see all requests traced across services
3. Click on any trace to see:
   - Complete request flow
   - Time spent in each service
   - Logs correlated with the trace

#### View Metrics
1. Click **"Metrics"** in the left sidebar
2. Explore pre-built dashboards
3. Create custom queries:
   - `jokes_served_total`
   - `http_server_request_count`
   - `http_server_request_duration`

#### View Logs
1. Click **"Logs"** in the left sidebar
2. Search logs by:
   - Service name
   - Trace ID
   - Log level
   - Custom fields

#### Service Map
1. Click **"Service Map"** to visualize service dependencies
2. See request flow and error rates

### Step 4: Generate Load

Generate traffic to see more interesting data:

```bash
./scripts/load-test.sh http://localhost:8000 100
```

Refresh SigNoz to see:
- Increased request rates
- Latency distributions
- Service dependency patterns

### Step 5: Stop Services

```bash
make local-down
```

## Option 2: Kubernetes Deployment

### Prerequisites
- Kubernetes cluster running (minikube, kind, or cloud)
- kubectl configured
- Docker images built and pushed

### Step 1: Build and Push Images

```bash
# Login to Docker Hub (or your registry)
docker login

# Build and push all images
make docker-push
```

### Step 2: Deploy to Kubernetes

```bash
make k8s-deploy
```

### Step 3: Wait for Pods

```bash
# Check platform namespace (SigNoz)
kubectl get pods -n platform

# Check default namespace (services)
kubectl get pods -n default
```

Wait until all pods are Running (may take 2-3 minutes).

### Step 4: Access Services

#### For Minikube:

```bash
# Access API Gateway
minikube service api-gateway -n default

# Access SigNoz
minikube service signoz-frontend -n platform
```

#### For Other K8s:

```bash
# Port forward API Gateway
kubectl port-forward -n default svc/api-gateway 8000:80

# Port forward SigNoz
kubectl port-forward -n platform svc/signoz-frontend 3301:3301
```

### Step 5: Test APIs

```bash
# Get NodePort
kubectl get svc api-gateway -n default

# Test (replace 30080 with your NodePort)
./scripts/test-apis.sh http://localhost:30080
```

### Step 6: Scale Services

```bash
# Manual scaling
kubectl scale deployment jokes-service -n default --replicas=5

# Check HPA status
kubectl get hpa -n default
```

### Step 7: Clean Up

```bash
make k8s-delete
```

## Understanding the Output

### Test Script Output

```bash
$ ./scripts/test-apis.sh

1. Health Check
-----------------------------------
{
  "status": "healthy",
  "service": "api-gateway",
  "timestamp": "2025-11-27T10:30:45Z"
}

2. Get Random Joke
-----------------------------------
{
  "joke": "Why do programmers hate nature? It has too many bugs.",
  "service": "jokes-service",
  "timestamp": "2025-11-27T10:30:46Z"
}
```

### SigNoz Traces View

You'll see traces like:
```
GET /api/v1/joke
├─ api-gateway (10ms)
│  └─ proxy_to_/api/v1/joke (45ms)
│     └─ jokes-service (40ms)
│        ├─ getRandomJoke (35ms)
│        └─ notifyAnalytics (2ms)
│           └─ analytics-service (1ms)
```

## Troubleshooting

### Services not starting in Docker Compose

```bash
# Check logs
docker-compose logs otel-collector
docker-compose logs api-gateway

# Restart specific service
docker-compose restart api-gateway
```

### Traces not appearing in SigNoz

1. Wait 60 seconds for data to appear
2. Check OTEL Collector logs:
   ```bash
   docker-compose logs otel-collector
   ```
3. Verify endpoint configuration:
   ```bash
   docker-compose exec api-gateway env | grep OTEL
   ```

### Kubernetes pods not starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n default

# Check logs
kubectl logs <pod-name> -n default

# Check events
kubectl get events -n default --sort-by='.lastTimestamp'
```

### Cannot access services in Kubernetes

```bash
# Check services
kubectl get svc -n default
kubectl get svc -n platform

# Test internal connectivity
kubectl run curl --image=curlimages/curl -it --rm -- sh
# Then: curl http://api-gateway.default.svc.cluster.local/healthz
```

## Next Steps

1. **Explore SigNoz Features**
   - Create custom dashboards
   - Set up alerts
   - Explore trace flamegraphs

2. **Modify Services**
   - Add new endpoints
   - Add custom metrics
   - Implement new features

3. **Learn OpenTelemetry**
   - Add custom spans
   - Add span attributes
   - Implement baggage propagation

4. **Practice Troubleshooting**
   - Introduce errors
   - Simulate latency
   - Practice using traces to debug

5. **Advanced Topics**
   - Sampling strategies
   - Tail-based sampling
   - Log correlation
   - Metric alerts

## Useful Commands

```bash
# View all containers
docker-compose ps

# Follow logs of all services
docker-compose logs -f

# Restart a service
docker-compose restart jokes-service

# View Kubernetes resources
kubectl get all -n default
kubectl get all -n platform

# Scale a deployment
kubectl scale deployment jokes-service --replicas=5 -n default

# Get pod logs
kubectl logs -f <pod-name> -n default

# Execute command in pod
kubectl exec -it <pod-name> -n default -- sh
```

## Learning Resources

- Read `ARCHITECTURE.md` for detailed system design
- Check service code in `services/` directory
- Review K8s manifests in `k8s/` directory
- Explore OpenTelemetry configuration in `otel-collector-config.yaml`

## Getting Help

- Check logs: `docker-compose logs` or `kubectl logs`
- Verify connectivity: Test health endpoints
- Review configuration: Environment variables
- Check SigNoz docs: https://signoz.io/docs/

Happy learning! 🚀

