#!/bin/bash

BASE_URL="http://localhost:8080"

echo "🔍 Testing Jnana Yoga Knowledge API..."
echo

# Test health
echo "1. Health Check:"
curl -s "$BASE_URL/health" | jq '.' || echo "Health check failed"
echo

# Test creating a concept
echo "2. Creating a Concept:"
CONCEPT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/concepts" \
  -H "Content-Type: application/json" \
  -d '{"name": "System Dynamics", "summary": "Study of complex systems", "mastery_level": 7}')
echo "Response: $CONCEPT_RESPONSE"
echo

# Extract concept ID if successful
if [[ $CONCEPT_RESPONSE == *"id"* ]]; then
  CONCEPT_ID=$(echo $CONCEPT_RESPONSE | jq -r '.id')
  echo "3. Getting Concept by ID ($CONCEPT_ID):"
  curl -s "$BASE_URL/api/v1/concepts/$CONCEPT_ID" | jq '.' || echo "Failed to get concept"
  echo
fi

echo "✅ API testing complete!"