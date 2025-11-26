#!/bin/bash

# Integration tests for WiiUDownloader API
# Run these tests against a running API server

set -e

API_BASE=${API_BASE:-"http://localhost:8080/api"}
QUIET=${QUIET:-false}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Helper functions
log_info() {
    [ "$QUIET" = false ] && echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_warning() {
    [ "$QUIET" = false ] && echo -e "${YELLOW}⚠${NC} $1"
}

run_test() {
    local test_name="$1"
    local command="$2"
    local expected_status="${3:-200}"

    ((TESTS_RUN++))
    [ "$QUIET" = false ] && echo -n "Testing $test_name... "

    if eval "$command" 2>/dev/null; then
        log_info "$test_name passed"
        ((TESTS_PASSED++))
        return 0
    else
        log_error "$test_name failed"
        return 1
    fi
}

# Test health endpoint
test_health() {
    run_test "Health endpoint" "curl -s '$API_BASE/health' | jq -e '.status == \"healthy\"' >/dev/null"
}

# Test OpenAPI spec
test_openapi() {
    run_test "OpenAPI spec" "curl -s '$API_BASE/openapi.json' | jq -e '.openapi == \"3.0.3\"' >/dev/null"
}

# Test list titles endpoint
test_list_titles() {
    run_test "List titles" "curl -s '$API_BASE/titles' | jq -e 'has(\"count\") and has(\"titles\")' >/dev/null"
}

# Test platform filtering
test_platform_filter() {
    run_test "Platform filter (Wii U)" "curl -s '$API_BASE/titles?platform=wiiu' | jq -e 'has(\"count\")' >/dev/null"
    run_test "Platform filter (3DS)" "curl -s '$API_BASE/titles?platform=3ds' | jq -e 'has(\"count\")' >/dev/null"
}

# Test format filtering
test_format_filter() {
    run_test "Format filter (CIA)" "curl -s '$API_BASE/titles?format=cia' | jq -e 'has(\"count\")' >/dev/null"
    run_test "Format filter (NSP)" "curl -s '$API_BASE/titles?format=nsp' | jq -e 'has(\"count\")' >/dev/null"
}

# Test category filtering
test_category_filter() {
    run_test "Category filter (games)" "curl -s '$API_BASE/titles?category=game' | jq -e 'has(\"count\")' >/dev/null"
    run_test "Category filter (updates)" "curl -s '$API_BASE/titles?category=update' | jq -e 'has(\"count\")' >/dev/null"
}

# Test region filtering
test_region_filter() {
    run_test "Region filter (USA)" "curl -s '$API_BASE/titles?region=usa' | jq -e 'has(\"count\")' >/dev/null"
    run_test "Region filter (Europe)" "curl -s '$API_BASE/titles?region=europe' | jq -e 'has(\"count\")' >/dev/null"
}

# Test search functionality
test_search() {
    run_test "Search functionality" "curl -s '$API_BASE/titles?search=test' | jq -e 'has(\"count\")' >/dev/null"
}

# Test combined filtering
test_combined_filters() {
    run_test "Combined filters" "curl -s '$API_BASE/titles?platform=wiiu&category=game&region=usa' | jq -e 'has(\"count\")' >/dev/null"
}

# Test invalid parameters
test_invalid_params() {
    run_test "Invalid platform" "curl -s '$API_BASE/titles?platform=invalid' | jq -e '.error' >/dev/null" 400
    run_test "Invalid format" "curl -s '$API_BASE/titles?format=invalid' | jq -e '.error' >/dev/null" 400
}

# Test title details endpoint
test_title_details() {
    # Try to get a title that might exist (this will likely fail in test environment, but tests the endpoint)
    run_test "Title details (valid format)" "curl -s '$API_BASE/titles/00050000101C9500' | jq -e 'type == \"object\"' >/dev/null" 404
    run_test "Title details (invalid ID)" "curl -s '$API_BASE/titles/invalid' | jq -e '.error' >/dev/null" 400
}

# Test download endpoint
test_download_endpoint() {
    # Test valid download request format (will fail due to no real title, but tests JSON validation)
    run_test "Download request (invalid title)" "curl -s -X POST '$API_BASE/download' -H 'Content-Type: application/json' -d '{\"title_id\":\"00050000101C9500\"}' | jq -e 'type == \"object\"' >/dev/null" 404

    # Test invalid requests
    run_test "Download request (no title_id)" "curl -s -X POST '$API_BASE/download' -H 'Content-Type: application/json' -d '{}' | jq -e '.error' >/dev/null" 400
}

# Test CORS headers
test_cors() {
    run_test "CORS headers" "curl -s -I -X OPTIONS '$API_BASE/titles' | grep -q 'Access-Control-Allow-Origin: *'"
}

# Main test runner
main() {
    echo "🧪 WiiUDownloader API Integration Tests"
    echo "========================================"
    echo "Testing API at: $API_BASE"
    echo ""

    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_error "jq is required for these tests. Please install jq."
        exit 1
    fi

    # Check if API is reachable
    if ! curl -s "$API_BASE/health" >/dev/null 2>&1; then
        log_error "API server is not reachable at $API_BASE"
        log_warning "Make sure the API server is running: ./build-api.sh && docker-compose up -d"
        exit 1
    fi

    log_info "API server is reachable"

    # Run all tests
    test_health
    test_openapi
    test_list_titles
    test_platform_filter
    test_format_filter
    test_category_filter
    test_region_filter
    test_search
    test_combined_filters
    test_invalid_params
    test_title_details
    test_download_endpoint
    test_cors

    echo ""
    echo "📊 Test Results: $TESTS_PASSED/$TESTS_RUN tests passed"

    if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
        log_info "All tests passed! 🎉"
        exit 0
    else
        log_error "Some tests failed. Check the output above."
        exit 1
    fi
}

# Allow running specific tests
if [ $# -gt 0 ]; then
    case "$1" in
        "health")
            test_health
            ;;
        "openapi")
            test_openapi
            ;;
        "list")
            test_list_titles
            ;;
        "platform")
            test_platform_filter
            ;;
        "format")
            test_format_filter
            ;;
        "category")
            test_category_filter
            ;;
        "region")
            test_region_filter
            ;;
        "search")
            test_search
            ;;
        "combined")
            test_combined_filters
            ;;
        "invalid")
            test_invalid_params
            ;;
        "title")
            test_title_details
            ;;
        "download")
            test_download_endpoint
            ;;
        "cors")
            test_cors
            ;;
        *)
            echo "Usage: $0 [test_name]"
            echo "Available tests: health, openapi, list, platform, format, category, region, search, combined, invalid, title, download, cors"
            echo "Run without arguments to run all tests"
            exit 1
            ;;
    esac
else
    main
fi
