#!/bin/bash
#
# Seed Example Locations
#
# This script populates the database with example locations for major cities
# and timezones around the world. Useful for demos and testing.
#
# Usage:
#   ./seed-locations.sh [API_URL]
#
# Examples:
#   ./seed-locations.sh                              # Default: http://localhost:8080
#   ./seed-locations.sh http://localhost:8080
#   ./seed-locations.sh https://timeservice.example.com
#

set -euo pipefail

# Configuration
API_URL=${1:-http://localhost:8080}
BASE_URL="$API_URL/api/locations"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if API is available
log_info "Checking API availability at $API_URL..."
if ! curl -f -s "$API_URL/health" > /dev/null; then
    log_error "API is not available at $API_URL"
    log_error "Make sure the server is running"
    exit 1
fi

log_info "API is available"

# Sample locations (name, timezone, description)
declare -A locations=(
    # Corporate locations
    ["headquarters"]="America/New_York|Company headquarters in NYC"
    ["west-coast-office"]="America/Los_Angeles|West Coast office in San Francisco"
    ["chicago-office"]="America/Chicago|Midwest office in Chicago"

    # International offices
    ["london-office"]="Europe/London|European headquarters in London"
    ["paris-office"]="Europe/Paris|Paris branch office"
    ["tokyo-office"]="Asia/Tokyo|Asia-Pacific headquarters in Tokyo"
    ["sydney-office"]="Australia/Sydney|Sydney branch office"
    ["singapore-office"]="Asia/Singapore|Southeast Asia office"
    ["mumbai-office"]="Asia/Kolkata|India office in Mumbai"

    # Data centers
    ["us-east-dc"]="America/New_York|US East data center"
    ["us-west-dc"]="America/Los_Angeles|US West data center"
    ["eu-central-dc"]="Europe/Frankfurt|EU Central data center"
    ["ap-northeast-dc"]="Asia/Tokyo|AP Northeast data center"

    # Support centers
    ["support-apac"]="Asia/Tokyo|24/7 APAC support center"
    ["support-emea"]="Europe/London|24/7 EMEA support center"
    ["support-americas"]="America/New_York|24/7 Americas support center"
)

# Track statistics
created_count=0
skipped_count=0
error_count=0

log_info "Seeding ${#locations[@]} example locations..."
echo ""

# Create each location
for name in "${!locations[@]}"; do
    IFS='|' read -r timezone description <<< "${locations[$name]}"

    # Build JSON payload
    json_payload=$(cat <<EOF
{
  "name": "$name",
  "timezone": "$timezone",
  "description": "$description"
}
EOF
)

    # Try to create the location
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL" \
        -H "Content-Type: application/json" \
        -d "$json_payload" 2>&1)

    # Extract HTTP status code (last line)
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | sed '$d')

    case $http_code in
        200|201)
            log_info "Created: $name ($timezone)"
            ((created_count++))
            ;;
        409)
            log_warn "Skipped: $name (already exists)"
            ((skipped_count++))
            ;;
        *)
            log_error "Failed: $name (HTTP $http_code)"
            ((error_count++))
            ;;
    esac
done

echo ""
log_info "Seeding complete!"
log_info "  Created: $created_count"
log_info "  Skipped: $skipped_count (already exist)"
log_info "  Errors:  $error_count"

# Show all locations
echo ""
log_info "Fetching all locations..."
curl -s "$BASE_URL" | grep -q "locations" && log_info "Successfully retrieved locations" || log_warn "Failed to retrieve locations"

echo ""
log_info "You can now query locations:"
echo "  curl $BASE_URL"
echo "  curl $BASE_URL/headquarters/time"
echo ""
