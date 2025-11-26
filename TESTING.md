# WiiUDownloader Testing

This document covers the testing setup for the WiiUDownloader API.

## Test Types

### Unit Tests
Located in `cmd/wiiu-api/main_test.go`, these test individual functions and handlers without requiring a running server.

```bash
# Run unit tests
cd cmd/wiiu-api
go test -v
```

### Integration Tests
The `test-api.sh` script tests the API endpoints against a running server.

```bash
# Run all integration tests
./test-api.sh

# Run specific test
./test-api.sh health
./test-api.sh platform
./test-api.sh format

# Run quietly (less verbose output)
QUIET=true ./test-api.sh

# Test different API endpoint
API_BASE=http://localhost:8081/api ./test-api.sh
```

## Running Tests

### Prerequisites
- Go 1.24+ (for unit tests)
- `jq` (for JSON parsing in integration tests)
- Running API server (for integration tests)

### Quick Test Setup

```bash
# 1. Build and start the API
./build-api.sh
docker-compose up -d

# 2. Wait for API to be ready
sleep 10

# 3. Run unit tests
cd cmd/wiiu-api && go test -v

# 4. Run integration tests
cd ../..
./test-api.sh

# 5. Stop the API
docker-compose down
```

## Test Coverage

### Unit Tests Cover:
- ✅ Health endpoint responses
- ✅ Platform detection from Title IDs
- ✅ Format detection from Title IDs
- ✅ API endpoint status codes
- ✅ JSON response validation
- ✅ CORS headers
- ✅ OpenAPI spec serving

### Integration Tests Cover:
- ✅ All API endpoints functionality
- ✅ Parameter validation (valid/invalid)
- ✅ Filtering by platform, format, category, region
- ✅ Search functionality
- ✅ Combined filtering
- ✅ CORS headers
- ✅ Error responses

## Test Structure

```
cmd/wiiu-api/
├── main.go           # API server
├── main_test.go      # Unit tests
└── ...

test-api.sh           # Integration tests
TESTING.md           # This file
```

## CI/CD Integration

The tests can be integrated into CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Run unit tests
  run: cd cmd/wiiu-api && go test -v

- name: Run integration tests
  run: |
    ./build-api.sh
    docker-compose up -d
    sleep 30
    ./test-api.sh
    docker-compose down
```

## Manual Testing

### Using curl

```bash
# Health check
curl http://localhost:8080/health

# List titles
curl "http://localhost:8080/api/titles?platform=wiiu&format=content"

# Get title details
curl http://localhost:8080/api/titles/00050000101C9500

# Start download
curl -X POST http://localhost:8080/api/download \
  -H "Content-Type: application/json" \
  -d '{"title_id": "00050000101C9500", "decrypt": true}'

# Check download status
curl http://localhost:8080/api/download/{job_id}
```

### Using Postman/Insomnia

1. Import the OpenAPI spec: `http://localhost:8080/api/openapi.json`
2. All endpoints and parameters will be auto-configured
3. Test individual endpoints with different parameters

## Test Data

The tests use the real title database from `db.go`. If you need to regenerate test data:

```bash
python3 grabTitles.py
```

## Troubleshooting

### Tests Fail with "API server not reachable"
- Ensure the API server is running: `docker-compose up -d`
- Wait 30 seconds for the server to fully start
- Check the API health: `curl http://localhost:8080/health`

### Tests Fail with "jq command not found"
- Install jq: `sudo apt install jq` (Ubuntu/Debian)
- Or use: `brew install jq` (macOS)

### Unit Tests Fail
- Ensure you're in the correct directory: `cd cmd/wiiu-api`
- Check Go version: `go version` (should be 1.24+)
- Clean test cache: `go clean -testcache`

## Adding New Tests

### Unit Tests
Add new test functions to `cmd/wiiu-api/main_test.go`:

```go
func TestNewFeature(t *testing.T) {
    // Test logic here
}
```

### Integration Tests
Add new test functions to `test-api.sh`:

```bash
test_new_feature() {
    run_test "New feature" "curl -s 'http://localhost:8080/api/new-endpoint' | jq -e '.success == true' >/dev/null"
}
```

Call the new function from `main()`.
