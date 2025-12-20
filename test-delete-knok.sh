#!/bin/bash

# Test script for deleting a knok via the admin API
# Usage: ./test-delete-knok.sh <knok-id> <admin-api-key>

KNOK_ID="${1:-370daa08-8b0f-4878-9f3d-d3a945b6dcc4}"
API_KEY="${2:-your-admin-api-key}"
API_URL="${3:-http://localhost:8080}"

echo "Testing DELETE endpoint for knok: $KNOK_ID"
echo "Using API URL: $API_URL"
echo ""

# Make the DELETE request
curl -v -X DELETE \
  "$API_URL/api/v1/admin/knoks/$KNOK_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json"

echo ""
echo ""
echo "Done!"
