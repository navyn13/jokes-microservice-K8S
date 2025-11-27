#!/bin/bash

# Load testing script to generate traffic for observability
# Usage: ./scripts/load-test.sh [base-url] [requests]

BASE_URL=${1:-http://localhost:8000}
REQUESTS=${2:-100}

echo "=================================="
echo "Load Testing Microservices"
echo "Base URL: $BASE_URL"
echo "Requests: $REQUESTS"
echo "=================================="

for i in $(seq 1 $REQUESTS); do
  echo "Request $i/$REQUESTS"
  
  # Get a joke
  curl -s "$BASE_URL/api/v1/joke" > /dev/null
  
  # 30% chance to add to favorites
  if [ $((RANDOM % 10)) -lt 3 ]; then
    USER_ID="user$((RANDOM % 10))"
    curl -s -X POST "$BASE_URL/api/v1/favorite" \
      -H "Content-Type: application/json" \
      -d "{\"joke\":\"Test joke $i\",\"user_id\":\"$USER_ID\"}" > /dev/null
  fi
  
  # 20% chance to check stats
  if [ $((RANDOM % 10)) -lt 2 ]; then
    curl -s "$BASE_URL/api/v1/stats" > /dev/null
  fi
  
  # Small delay between requests
  sleep 0.1
done

echo ""
echo "=================================="
echo "Load test completed!"
echo "Total requests: $REQUESTS"
echo "=================================="
echo ""
echo "Check SigNoz UI for:"
echo "- Trace patterns and latency distribution"
echo "- Service dependency graphs"
echo "- Error rates and anomalies"
echo ""

