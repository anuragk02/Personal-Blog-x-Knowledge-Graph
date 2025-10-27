#!/bin/bash

# Test script to verify relationship consolidation fix
BASE_URL="http://localhost:8080/api/v1"

echo "=== Testing Relationship Consolidation Fix ==="

# Reset consolidation first
echo "1. Resetting consolidation status..."
curl -s -X POST "$BASE_URL/consolidate/reset" | jq .

echo -e "\n2. Checking relationship status before consolidation..."
curl -s "$BASE_URL/debug/relationship-status" | jq '.total_relationships, .consolidated_relationships, .unconsolidated_relationships'

echo -e "\n3. Running consolidation..."
curl -s -X POST "$BASE_URL/consolidate" | jq .

echo -e "\n4. Checking relationship status after consolidation..."
curl -s "$BASE_URL/debug/relationship-status" | jq '.total_relationships, .consolidated_relationships, .unconsolidated_relationships'

echo -e "\n5. Showing detailed relationship status..."
curl -s "$BASE_URL/debug/relationship-status" | jq '.relationships[] | select(.consolidated == false)'

echo -e "\n=== Test Complete ==="