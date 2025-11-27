.PHONY: help build build-all docker-build docker-push k8s-deploy k8s-delete test local-up local-down

# Default target
help:
	@echo "Available targets:"
	@echo "  build-all        - Build all microservices"
	@echo "  docker-build     - Build all Docker images"
	@echo "  docker-push      - Push all Docker images to registry"
	@echo "  local-up         - Start local environment with docker-compose"
	@echo "  local-down       - Stop local environment"
	@echo "  k8s-deploy       - Deploy to Kubernetes"
	@echo "  k8s-delete       - Delete from Kubernetes"
	@echo "  test             - Run test requests"
	@echo "  logs             - View logs from all services"

# Build all services
build-all:
	@echo "Building all services..."
	cd services/gateway && go mod tidy && go build -o gateway-server .
	cd services/jokes && go mod tidy && go build -o jokes-server .
	cd services/analytics && go mod tidy && go build -o analytics-server .
	cd services/user && go mod tidy && go build -o user-server .
	@echo "All services built successfully!"

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker build -t navyn13/api-gateway:latest ./services/gateway
	docker build -t navyn13/jokes-service:latest ./services/jokes
	docker build -t navyn13/analytics-service:latest ./services/analytics
	docker build -t navyn13/user-service:latest ./services/user
	@echo "All Docker images built successfully!"

# Push Docker images
docker-push: docker-build
	@echo "Pushing Docker images..."
	docker push navyn13/api-gateway:latest
	docker push navyn13/jokes-service:latest
	docker push navyn13/analytics-service:latest
	docker push navyn13/user-service:latest
	@echo "All Docker images pushed successfully!"

# Start local environment
local-up:
	@echo "Starting local environment..."
	docker-compose up -d
	@echo "Local environment started!"
	@echo "Access SigNoz UI at: http://localhost:3301"
	@echo "Access API Gateway at: http://localhost:8000"

# Stop local environment
local-down:
	@echo "Stopping local environment..."
	docker-compose down
	@echo "Local environment stopped!"

# Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/signoz.yaml
	@echo "Waiting for SigNoz to be ready..."
	sleep 30
	kubectl apply -f k8s/gateway.yaml
	kubectl apply -f k8s/jokes-service.yaml
	kubectl apply -f k8s/analytics-service.yaml
	kubectl apply -f k8s/user-service.yaml
	@echo "Deployment complete!"
	@echo "Check status with: kubectl get pods -n default && kubectl get pods -n platform"

# Delete from Kubernetes
k8s-delete:
	@echo "Deleting from Kubernetes..."
	kubectl delete -f k8s/gateway.yaml --ignore-not-found=true
	kubectl delete -f k8s/jokes-service.yaml --ignore-not-found=true
	kubectl delete -f k8s/analytics-service.yaml --ignore-not-found=true
	kubectl delete -f k8s/user-service.yaml --ignore-not-found=true
	kubectl delete -f k8s/signoz.yaml --ignore-not-found=true
	@echo "Deletion complete!"

# Run test requests
test:
	@echo "Testing API Gateway..."
	@echo "\n1. Health Check:"
	curl -s http://localhost:8000/healthz | jq .
	@echo "\n2. Get a random joke:"
	curl -s http://localhost:8000/api/v1/joke | jq .
	@echo "\n3. Add to favorites:"
	curl -s -X POST http://localhost:8000/api/v1/favorite \
		-H "Content-Type: application/json" \
		-d '{"joke":"Test joke","user_id":"user123"}' | jq .
	@echo "\n4. Get statistics:"
	curl -s http://localhost:8000/api/v1/stats | jq .

# View logs
logs:
	docker-compose logs -f api-gateway jokes-service analytics-service user-service

# Clean everything
clean:
	@echo "Cleaning up..."
	docker-compose down -v
	rm -f services/gateway/gateway-server
	rm -f services/jokes/jokes-server
	rm -f services/analytics/analytics-server
	rm -f services/user/user-server
	@echo "Cleanup complete!"

