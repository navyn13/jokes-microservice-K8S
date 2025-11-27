#!/bin/bash

# Build all microservices
set -e

echo "=================================="
echo "Building All Microservices"
echo "=================================="

SERVICES=("gateway" "jokes" "analytics" "user")

for service in "${SERVICES[@]}"; do
  echo ""
  echo "Building $service service..."
  cd "services/$service"
  
  # Download dependencies
  go mod download
  
  # Build the service
  CGO_ENABLED=0 GOOS=linux go build -o "${service}-server" .
  
  echo "✓ $service service built successfully"
  cd ../..
done

echo ""
echo "=================================="
echo "All services built successfully!"
echo "=================================="

