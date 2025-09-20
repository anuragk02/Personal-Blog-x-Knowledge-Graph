#!/bin/bash

BASE_URL="http://localhost:8080"

echo "🔍 Testing Jnana Yoga Knowledge API (Complete Schema)..."
echo "=================================================="
echo

# Test health
echo "1. Health Check:"
curl -s "$BASE_URL/health" | jq '.' || echo "Health check failed"
echo

# Store IDs for relationship testing
CONCEPT_ID=""
ESSAY_ID=""
CLAIM_ID=""
SOURCE_ID=""
QUESTION_ID=""

# ====================
# NODE CREATION TESTS
# ====================

echo "🧠 CONCEPT CRUD:"
echo "2. Creating a Concept:"
CONCEPT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/concepts" \
  -H "Content-Type: application/json" \
  -d '{"name": "System Dynamics", "summary": "Study of complex systems", "mastery_level": 7}')
echo "Response: $CONCEPT_RESPONSE"
echo

if [[ $CONCEPT_RESPONSE == *"id"* ]]; then
  CONCEPT_ID=$(echo $CONCEPT_RESPONSE | jq -r '.id')
  echo "3. Getting Concept by ID ($CONCEPT_ID):"
  curl -s "$BASE_URL/api/v1/concepts/$CONCEPT_ID" | jq '.' || echo "Failed to get concept"
  echo
fi

echo "📄 ESSAY CRUD:"
echo "4. Creating an Essay:"
ESSAY_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/essays" \
  -H "Content-Type: application/json" \
  -d '{"title": "Understanding Complex Systems", "content": "This essay explores how system dynamics can help us understand complex adaptive systems..."}')
echo "Response: $ESSAY_RESPONSE"
echo

if [[ $ESSAY_RESPONSE == *"id"* ]]; then
  ESSAY_ID=$(echo $ESSAY_RESPONSE | jq -r '.id')
  echo "5. Getting Essay by ID ($ESSAY_ID):"
  curl -s "$BASE_URL/api/v1/essays/$ESSAY_ID" | jq '.' || echo "Failed to get essay"
  echo
fi

echo "💡 CLAIM CRUD:"
echo "6. Creating a Claim:"
CLAIM_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/claims" \
  -H "Content-Type: application/json" \
  -d '{"text": "Complex systems exhibit emergent properties", "confidence_score": 8, "is_verified": false}')
echo "Response: $CLAIM_RESPONSE"
echo

if [[ $CLAIM_RESPONSE == *"id"* ]]; then
  CLAIM_ID=$(echo $CLAIM_RESPONSE | jq -r '.id')
  echo "7. Getting Claim by ID ($CLAIM_ID):"
  curl -s "$BASE_URL/api/v1/claims/$CLAIM_ID" | jq '.' || echo "Failed to get claim"
  echo
fi

echo "📚 SOURCE CRUD:"
echo "8. Creating a Source:"
SOURCE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/sources" \
  -H "Content-Type: application/json" \
  -d '{"title": "Thinking in Systems", "author": "Donella Meadows", "type": "Book", "url": "https://example.com/thinking-in-systems"}')
echo "Response: $SOURCE_RESPONSE"
echo

if [[ $SOURCE_RESPONSE == *"id"* ]]; then
  SOURCE_ID=$(echo $SOURCE_RESPONSE | jq -r '.id')
  echo "9. Getting Source by ID ($SOURCE_ID):"
  curl -s "$BASE_URL/api/v1/sources/$SOURCE_ID" | jq '.' || echo "Failed to get source"
  echo
fi

echo "❓ QUESTION CRUD:"
echo "10. Creating a Question:"
QUESTION_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/questions" \
  -H "Content-Type: application/json" \
  -d '{"text": "How do emergent properties arise in complex systems?", "priority": 8, "status": "open"}')
echo "Response: $QUESTION_RESPONSE"
echo

if [[ $QUESTION_RESPONSE == *"id"* ]]; then
  QUESTION_ID=$(echo $QUESTION_RESPONSE | jq -r '.id')
  echo "11. Getting Question by ID ($QUESTION_ID):"
  curl -s "$BASE_URL/api/v1/questions/$QUESTION_ID" | jq '.' || echo "Failed to get question"
  echo
fi

# ====================
# RELATIONSHIP TESTS
# ====================

echo "🔗 RELATIONSHIP CREATION:"

# Test DEFINES relationship (Concept -> Concept)
if [[ -n "$CONCEPT_ID" ]]; then
  echo "12. Creating DEFINES relationship:"
  curl -s -X POST "$BASE_URL/api/v1/defines" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$CONCEPT_ID\", \"to\": \"$CONCEPT_ID\", \"definition\": \"Self-referential definition\"}" | jq '.'
  echo
fi

# Test INFLUENCES relationship (Concept -> Concept)
if [[ -n "$CONCEPT_ID" ]]; then
  echo "13. Creating INFLUENCES relationship:"
  curl -s -X POST "$BASE_URL/api/v1/influences" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$CONCEPT_ID\", \"to\": \"$CONCEPT_ID\", \"strength\": 7}" | jq '.'
  echo
fi

# Test SUPPORTS relationship (Claim -> Claim or Source -> Claim)
if [[ -n "$SOURCE_ID" && -n "$CLAIM_ID" ]]; then
  echo "14. Creating SUPPORTS relationship:"
  curl -s -X POST "$BASE_URL/api/v1/supports" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$SOURCE_ID\", \"to\": \"$CLAIM_ID\", \"strength\": 8}" | jq '.'
  echo
fi

# Test CONTRADICTS relationship (Claim -> Claim)
if [[ -n "$CLAIM_ID" ]]; then
  echo "15. Creating CONTRADICTS relationship:"
  curl -s -X POST "$BASE_URL/api/v1/contradicts" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$CLAIM_ID\", \"to\": \"$CLAIM_ID\", \"strength\": 3}" | jq '.'
  echo
fi

# Test DERIVED_FROM relationship (Concept -> Source)
if [[ -n "$CONCEPT_ID" && -n "$SOURCE_ID" ]]; then
  echo "16. Creating DERIVED_FROM relationship:"
  curl -s -X POST "$BASE_URL/api/v1/derived-from" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$CONCEPT_ID\", \"to\": \"$SOURCE_ID\"}" | jq '.'
  echo
fi

# Test RAISES relationship (Question -> Concept)
if [[ -n "$QUESTION_ID" && -n "$CONCEPT_ID" ]]; then
  echo "17. Creating RAISES relationship:"
  curl -s -X POST "$BASE_URL/api/v1/raises" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$QUESTION_ID\", \"to\": \"$CONCEPT_ID\"}" | jq '.'
  echo
fi

# Test POINTS_TO relationship (Essay -> any node)
if [[ -n "$ESSAY_ID" && -n "$CONCEPT_ID" ]]; then
  echo "18. Creating POINTS_TO relationship:"
  curl -s -X POST "$BASE_URL/api/v1/points-to" \
    -H "Content-Type: application/json" \
    -d "{\"from\": \"$ESSAY_ID\", \"to\": \"$CONCEPT_ID\"}" | jq '.'
  echo
fi

# ====================
# COLLECTION TESTS
# ====================

echo "📋 COLLECTION RETRIEVAL:"

echo "19. Getting All Concepts:"
curl -s "$BASE_URL/api/v1/concepts" | jq '. | length' | xargs echo "Found concepts:"
echo

echo "20. Getting All Essays:"
curl -s "$BASE_URL/api/v1/essays" | jq '. | length' | xargs echo "Found essays:"
echo

echo "21. Getting All Claims:"
curl -s "$BASE_URL/api/v1/claims" | jq '. | length' | xargs echo "Found claims:"
echo

echo "22. Getting All Sources:"
curl -s "$BASE_URL/api/v1/sources" | jq '. | length' | xargs echo "Found sources:"
echo

echo "23. Getting All Questions:"
curl -s "$BASE_URL/api/v1/questions" | jq '. | length' | xargs echo "Found questions:"
echo

# ====================
# UPDATE TESTS
# ====================

echo "🔄 UPDATE OPERATIONS:"

# Update Concept
if [[ -n "$CONCEPT_ID" ]]; then
  echo "24. Updating Concept ($CONCEPT_ID):"
  curl -s -X PUT "$BASE_URL/api/v1/concepts/$CONCEPT_ID" \
    -H "Content-Type: application/json" \
    -d '{"name": "Advanced System Dynamics", "summary": "Updated study of complex systems with advanced patterns", "mastery_level": 9}' | jq '.'
  echo
fi

# Update Essay
if [[ -n "$ESSAY_ID" ]]; then
  echo "25. Updating Essay ($ESSAY_ID):"
  curl -s -X PUT "$BASE_URL/api/v1/essays/$ESSAY_ID" \
    -H "Content-Type: application/json" \
    -d '{"title": "Deep Understanding of Complex Systems", "content": "This updated essay provides deeper insights into how system dynamics can help us understand complex adaptive systems and their emergent properties..."}' | jq '.'
  echo
fi

# Update Claim
if [[ -n "$CLAIM_ID" ]]; then
  echo "26. Updating Claim ($CLAIM_ID):"
  curl -s -X PUT "$BASE_URL/api/v1/claims/$CLAIM_ID" \
    -H "Content-Type: application/json" \
    -d '{"text": "Complex systems exhibit emergent properties that cannot be predicted from individual components", "confidence_score": 9, "is_verified": true}' | jq '.'
  echo
fi

# Update Source
if [[ -n "$SOURCE_ID" ]]; then
  echo "27. Updating Source ($SOURCE_ID):"
  curl -s -X PUT "$BASE_URL/api/v1/sources/$SOURCE_ID" \
    -H "Content-Type: application/json" \
    -d '{"title": "Thinking in Systems: A Primer", "author": "Donella H. Meadows", "type": "Book", "url": "https://updated-url.com/thinking-in-systems"}' | jq '.'
  echo
fi

# Update Question
if [[ -n "$QUESTION_ID" ]]; then
  echo "28. Updating Question ($QUESTION_ID):"
  curl -s -X PUT "$BASE_URL/api/v1/questions/$QUESTION_ID" \
    -H "Content-Type: application/json" \
    -d '{"text": "How do emergent properties arise in complex systems and what are their implications?", "priority": 10, "status": "in-progress"}' | jq '.'
  echo
fi

# ====================
# DELETE TESTS
# ====================

echo "🗑️ DELETE OPERATIONS:"

# Test deleting (we'll delete some nodes to test the functionality)
# Create a temporary concept to delete
echo "29. Creating temporary concept for deletion test:"
TEMP_CONCEPT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/concepts" \
  -H "Content-Type: application/json" \
  -d '{"name": "Temporary Concept", "summary": "This will be deleted", "mastery_level": 1}')
TEMP_CONCEPT_ID=$(echo $TEMP_CONCEPT_RESPONSE | jq -r '.id')
echo "Created temporary concept: $TEMP_CONCEPT_ID"
echo

echo "30. Deleting temporary concept ($TEMP_CONCEPT_ID):"
curl -s -X DELETE "$BASE_URL/api/v1/concepts/$TEMP_CONCEPT_ID" | jq '.'
echo

echo "31. Attempting to get deleted concept (should return 404):"
curl -s "$BASE_URL/api/v1/concepts/$TEMP_CONCEPT_ID" | jq '.'
echo

echo "✅ Complete Schema API Testing Finished!"
echo "=================================================="
echo "Created nodes:"
echo "  - Concept ID: $CONCEPT_ID"
echo "  - Essay ID: $ESSAY_ID" 
echo "  - Claim ID: $CLAIM_ID"
echo "  - Source ID: $SOURCE_ID"
echo "  - Question ID: $QUESTION_ID"
echo "=================================================="