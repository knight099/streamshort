#!/bin/bash

# Test script for Admin Get Episodes endpoint
# Usage: ./test_admin_episodes.sh [series_id]

SERIES_ID=${1:-"test-series-id"}
BASE_URL="http://localhost:8080"

echo "Testing Admin Get Episodes endpoint..."
echo "Series ID: $SERIES_ID"
echo ""

# Test 1: Get all episodes for a series
echo "1. Get all episodes for series:"
curl -X GET "$BASE_URL/api/admin/content/series/$SERIES_ID/episodes" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  | jq '.' 2>/dev/null || echo "Response received (install jq for pretty formatting)"

echo -e "\n"

# Test 2: Get episodes filtered by status
echo "2. Get episodes filtered by status (published):"
curl -X GET "$BASE_URL/api/admin/content/series/$SERIES_ID/episodes?status=published" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  | jq '.' 2>/dev/null || echo "Response received (install jq for pretty formatting)"

echo -e "\n"

# Test 3: Get episodes filtered by season
echo "3. Get episodes filtered by season (season 1):"
curl -X GET "$BASE_URL/api/admin/content/series/$SERIES_ID/episodes?season=1" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  | jq '.' 2>/dev/null || echo "Response received (install jq for pretty formatting)"

echo -e "\n"

# Test 4: Get episodes with multiple filters
echo "4. Get episodes with multiple filters (season 1, status published):"
curl -X GET "$BASE_URL/api/admin/content/series/$SERIES_ID/episodes?season=1&status=published" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  | jq '.' 2>/dev/null || echo "Response received (install jq for pretty formatting)"

echo -e "\n"
echo "Test completed!"
echo ""
echo "Note: Replace YOUR_ADMIN_TOKEN with a valid admin JWT token"
echo "Note: Replace $SERIES_ID with an actual series ID from your database"