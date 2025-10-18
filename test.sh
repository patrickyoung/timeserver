#!/bin/bash

# Time Service Test Script
# This script demonstrates how to interact with the time service API and MCP endpoints

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "========================================"
echo "Time Service API Test"
echo "========================================"
echo ""

# Test 1: Root endpoint
echo "1. Testing root endpoint..."
curl -s "$BASE_URL/" | jq .
echo ""

# Test 2: Health check
echo "2. Testing health endpoint..."
curl -s "$BASE_URL/health" | jq .
echo ""

# Test 3: Get current time
echo "3. Testing time endpoint..."
curl -s "$BASE_URL/api/time" | jq .
echo ""

echo "========================================"
echo "MCP Server Tests"
echo "========================================"
echo ""

# Test 4: List MCP tools
echo "4. Listing available MCP tools..."
curl -s -X POST "$BASE_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/list"
  }' | jq .
echo ""

# Test 5: Get current time via MCP (ISO8601)
echo "5. Getting current time via MCP (ISO8601, UTC)..."
curl -s -X POST "$BASE_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "get_current_time",
      "arguments": {
        "format": "iso8601",
        "timezone": "UTC"
      }
    }
  }' | jq .
echo ""

# Test 6: Get current time via MCP (Unix timestamp)
echo "6. Getting current time via MCP (Unix timestamp)..."
curl -s -X POST "$BASE_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "get_current_time",
      "arguments": {
        "format": "unix"
      }
    }
  }' | jq .
echo ""

# Test 7: Get current time in New York timezone
echo "7. Getting current time in America/New_York..."
curl -s -X POST "$BASE_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "get_current_time",
      "arguments": {
        "format": "rfc3339",
        "timezone": "America/New_York"
      }
    }
  }' | jq .
echo ""

# Test 8: Add time offset (add 2 hours)
echo "8. Adding 2 hours to current time..."
curl -s -X POST "$BASE_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "add_time_offset",
      "arguments": {
        "hours": 2,
        "minutes": 0,
        "format": "rfc3339"
      }
    }
  }' | jq .
echo ""

# Test 9: Subtract time offset (subtract 30 minutes)
echo "9. Subtracting 30 minutes from current time..."
curl -s -X POST "$BASE_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "add_time_offset",
      "arguments": {
        "hours": 0,
        "minutes": -30,
        "format": "iso8601"
      }
    }
  }' | jq .
echo ""

echo "========================================"
echo "All tests completed!"
echo "========================================"
