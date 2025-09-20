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
CONCEPT_ID=""
if [[ $CONCEPT_RESPONSE == *"id"* ]]; then
  CONCEPT_ID=$(echo $CONCEPT_RESPONSE | jq -r '.id')
  echo "3. Getting Concept by ID ($CONCEPT_ID):"
  curl -s "$BASE_URL/api/v1/concepts/$CONCEPT_ID" | jq '.' || echo "Failed to get concept"
  echo
fi

# Test creating an essay
echo "4. Creating an Essay:"
ESSAY_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/essays" \
  -H "Content-Type: application/json" \
  -d '{"title": "Understanding Complex Systems", "content": "This essay explores how system dynamics can help us understand complex adaptive systems..."}')
echo "Response: $ESSAY_RESPONSE"
echo

# Extract essay ID if successful
ESSAY_ID=""
if [[ $ESSAY_RESPONSE == *"id"* ]]; then
  ESSAY_ID=$(echo $ESSAY_RESPONSE | jq -r '.id')
  echo "5. Getting Essay by ID ($ESSAY_ID):"
  curl -s "$BASE_URL/api/v1/essays/$ESSAY_ID" | jq '.' || echo "Failed to get essay"
  echo
fi

# Test getting all essays
echo "6. Getting All Essays:"
curl -s "$BASE_URL/api/v1/essays" | jq '.' || echo "Failed to get essays"
echo

# Test creating POINTS_TO relationship if both concept and essay exist
if [[ -n "$CONCEPT_ID" && -n "$ESSAY_ID" ]]; then
  echo "7. Creating POINTS_TO relationship (Essay -> Concept):"
  POINTS_TO_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/points-to" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$ESSAY_ID\", \"to\": \"$CONCEPT_ID\"}")
  echo "Response: $POINTS_TO_RESPONSE"
  echo
fi

echo "✅ API testing complete!"