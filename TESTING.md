# Testing Strategy

This document describes the testing approach for the timeservice project.

## Test Coverage Summary

Overall test coverage: **60.8%** (excluding main.go and test utilities)

### Component Coverage
- **Handler**: 91.7% coverage
- **MCP Server**: 91.9% coverage
- **Middleware**: 100% coverage
- **Model**: 100% coverage

## Testing Philosophy

This project follows modern Go testing practices:

1. **Table-Driven Tests** - Using subtests with `t.Run()` for comprehensive test cases
2. **Dependency Injection** - All dependencies use interfaces for easy mocking
3. **Test Utilities** - Shared mocks and helpers in `internal/testutil`
4. **Standard Library** - Using `testing` and `net/http/httptest` packages
5. **Clear Test Names** - Descriptive test names that explain what's being tested

## Test Organization

```
.
├── internal/
│   ├── handler/
│   │   └── handler_test.go          # HTTP handler tests
│   ├── mcpserver/
│   │   └── server_test.go           # MCP tool handler tests
│   ├── middleware/
│   │   └── middleware_test.go       # Middleware chain tests
│   └── testutil/
│       └── testutil.go              # Mock implementations
└── pkg/
    └── model/
        └── model_test.go            # Model constructor tests
```

## Test Utilities

### MockLogger

Captures log calls for verification in tests:

```go
logger := &testutil.MockLogger{}
// ... run code that logs ...
logger.AssertInfoCount(t, 1)
logger.AssertErrorCount(t, 0)
```

### MockMCPServer

Test implementation of MCP HTTP server:

```go
mockServer := &testutil.MockMCPServer{
    ServeHTTPFunc: func(w http.ResponseWriter, r *http.Request) {
        // Custom behavior
    },
}
```

## Running Tests

### Using Make (Recommended)

```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Generate coverage report
make test-coverage

# Generate HTML coverage report
make test-coverage-html
```

### Using Go Command

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate coverage profile
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

## Test Categories

### Unit Tests

#### Handler Tests (`internal/handler/handler_test.go`)
- `TestGetTime` - Time endpoint returns correct response
- `TestHealth` - Health check endpoint
- `TestServiceInfo` - Service info endpoint (table-driven)
  - Root path returns service info
  - Non-root paths return 404
- `TestMCP` - MCP endpoint handling (table-driven)
  - Nil server error handling
  - Successful MCP requests
- `TestHandlerJSONHelpers` - JSON helper methods

#### Middleware Tests (`internal/middleware/middleware_test.go`)
- `TestLogger` - Request logging middleware
- `TestLoggerCapturesStatusCode` - Status code logging (table-driven)
- `TestRecover` - Panic recovery middleware
- `TestRecoverDoesNotInterceptNormalFlow` - Normal flow not affected
- `TestCORS` - CORS headers are set
- `TestCORSOptionsRequest` - OPTIONS request handling
- `TestChain` - Middleware chaining order
- `TestResponseWriter*` - Response writer wrapper tests

#### MCP Server Tests (`internal/mcpserver/server_test.go`)
- `TestNewServer` - Server initialization
- `TestHandleGetCurrentTime` - Time tool handler (table-driven)
  - Various format and timezone combinations
  - Invalid timezone handling
- `TestHandleGetCurrentTimeDefaults` - Default parameter handling
- `TestHandleAddTimeOffset` - Time offset tool (table-driven)
  - Add hours, minutes, both
  - Subtract time
  - Fractional hours
- `TestHandleAddTimeOffsetDefaults` - Default parameters
- `TestHandleAddTimeOffsetFormats` - Different output formats

#### Model Tests (`pkg/model/model_test.go`)
- `TestNewTimeResponse` - Time response construction
- `TestTimeResponseWithDifferentTimezones` - Timezone handling
- `TestNewServiceInfo` - Service info construction
- `TestServiceInfoStructure` - Service info structure validation

## Best Practices Demonstrated

### 1. Table-Driven Tests
```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {"case 1", "input1", "output1"},
    {"case 2", "input2", "output2"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test using tt.input and tt.expected
    })
}
```

### 2. Test Helpers with t.Helper()
```go
func (m *MockLogger) AssertInfoCount(t *testing.T, expected int) {
    t.Helper()  // Marks this as a helper function
    if len(m.InfoCalls) != expected {
        t.Errorf("expected %d Info calls, got %d", expected, len(m.InfoCalls))
    }
}
```

### 3. Subtests for Organization
```go
t.Run("json helper", func(t *testing.T) {
    // Test JSON helper
})
```

### 4. HTTP Testing with httptest
```go
req := httptest.NewRequest(http.MethodGet, "/api/time", nil)
w := httptest.NewRecorder()
handler.ServeHTTP(w, req)
```

### 5. Dependency Injection with Interfaces
All dependencies use interfaces defined in `pkg/logger` and `pkg/mcphttp`, making testing with mocks straightforward.

## Future Improvements

- [ ] Integration tests with real HTTP server
- [ ] Benchmark tests for performance-critical paths
- [ ] Property-based testing for complex logic
- [ ] E2E tests with MCP client
- [ ] Golden file testing for complex outputs
- [ ] Fuzz testing for input validation

## CI/CD Integration

The test suite is designed to be easily integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run Tests
  run: make test-coverage

- name: Check Coverage
  run: |
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$coverage < 80" | bc -l) )); then
      echo "Coverage is below 80%"
      exit 1
    fi
```
