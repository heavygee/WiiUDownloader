#!/bin/bash

echo "🧪 Simple WiiUDownloader API Tests"
echo "=================================="

API_BASE="http://localhost:11235/api"

# Test health endpoint
echo -n "Testing health endpoint... "
if curl -s "$API_BASE/health" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
fi

# Test OpenAPI spec
echo -n "Testing OpenAPI spec... "
if curl -s "$API_BASE/openapi.json" | jq -e '.openapi == "3.0.3"' >/dev/null 2>&1; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
fi

# Test list titles
echo -n "Testing list titles... "
if curl -s "$API_BASE/titles" | jq -e 'has("count") and has("titles")' >/dev/null 2>&1; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
fi

# Test platform filtering
echo -n "Testing platform filter (Wii U)... "
if curl -s "$API_BASE/titles?platform=wiiu" | jq -e 'has("count")' >/dev/null 2>&1; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
fi

# Test format filtering
echo -n "Testing format filter (CIA)... "
if curl -s "$API_BASE/titles?format=cia" | jq -e 'has("count")' >/dev/null 2>&1; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
fi

# Test combined filtering
echo -n "Testing combined filters... "
if curl -s "$API_BASE/titles?platform=3ds&format=cia&category=game" | jq -e 'has("count")' >/dev/null 2>&1; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
fi

# Test invalid parameters
echo -n "Testing invalid platform... "
if curl -s "$API_BASE/titles?platform=invalid" | jq -e '.error' >/dev/null 2>&1; then
    echo "✅ PASSED (correctly rejected)"
else
    echo "❌ FAILED (should have been rejected)"
fi

# Test title details (will fail for non-existent title, but tests endpoint)
echo -n "Testing title details endpoint... "
response=$(curl -s "$API_BASE/titles/00050000101C9500" 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "✅ PASSED (endpoint responded)"
else
    echo "❌ FAILED (endpoint not responding)"
fi

echo ""
echo "🎉 API tests completed!"
echo "API is running at: http://localhost:8080"
echo "OpenAPI spec: http://localhost:8080/api/openapi.json"
