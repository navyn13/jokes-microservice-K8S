#!/bin/bash

# Test script for microservices APIs
# Usage: ./scripts/test-apis.sh [base-url]

BASE_URL=${1:-http://localhost:8000}

echo "=================================="
echo "Testing Jokes Microservices"
echo "Base URL: $BASE_URL"
echo "=================================="

echo ""
echo "1. Health Check"
echo "-----------------------------------"
curl -s "$BASE_URL/healthz" | jq .

echo ""
echo "2. Get Random Joke (Request #1)"
echo "-----------------------------------"
JOKE1=$(curl -s "$BASE_URL/api/v1/joke")
echo "$JOKE1" | jq .
JOKE_TEXT=$(echo "$JOKE1" | jq -r '.joke')

echo ""
echo "3. Get Random Joke (Request #2)"
echo "-----------------------------------"
curl -s "$BASE_URL/api/v1/joke" | jq .

echo ""
echo "4. Get Random Joke (Request #3)"
echo "-----------------------------------"
curl -s "$BASE_URL/api/v1/joke" | jq .

echo ""
echo "5. Add Favorite Joke"
echo "-----------------------------------"
curl -s -X POST "$BASE_URL/api/v1/favorite" \
  -H "Content-Type: application/json" \
  -d "{\"joke\":\"$JOKE_TEXT\",\"user_id\":\"user123\"}" | jq .

echo ""
echo "6. Add Another Favorite"
echo "-----------------------------------"
curl -s -X POST "$BASE_URL/api/v1/favorite" \
  -H "Content-Type: application/json" \
  -d '{"joke":"Test joke from script","user_id":"user456"}' | jq .

echo ""
echo "7. Get Analytics Statistics"
echo "-----------------------------------"
curl -s "$BASE_URL/api/v1/stats" | jq .

echo ""
echo "=================================="
echo "Test completed!"
echo "=================================="
echo ""
echo "Now check SigNoz UI at http://localhost:3301"
echo "- Navigate to Traces to see distributed tracing"
echo "- Check Metrics for custom metrics"
echo "- View Logs with trace correlation"
echo ""

