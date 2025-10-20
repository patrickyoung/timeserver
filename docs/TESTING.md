# Testing Strategy

This document describes the testing approach for the timeservice project.

> **📊 For detailed coverage analysis and testing metrics, see [TESTING_ANALYSIS.md](./TESTING_ANALYSIS.md)**

## Test Coverage Summary

Overall test coverage: **80%+** for key packages (excluding main.go and test utilities)

### Component Coverage
- **Model**: 100.0% coverage ✅ (includes location models)
- **Handler**: 84.8% coverage (includes location API integration tests)
- **MCP Server**: 83.7% coverage ✅ (includes location MCP tools, improved +13.5%)
- **Config**: 83.5% coverage ✅ (includes database configuration)
- **Repository**: 77.3% coverage (SQLite database layer)
- **Metrics**: 66.7% coverage (includes database metrics)
- **Auth**: 59.6% coverage
- **Middleware**: 41.5% coverage (includes auth middleware)

### Additional Test Coverage
- **Integration Tests**: 27/28 passing (96% pass rate) - See `scripts/integration-test.sh`
- **Benchmark Tests**: 7 comprehensive benchmarks - See `internal/repository/location_bench_test.go`

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
│   │   ├── handler_test.go                  # HTTP handler tests
│   │   └── location_integration_test.go     # Location API integration tests
│   ├── mcpserver/
│   │   ├── server_test.go                   # MCP tool handler tests
│   │   └── location_tools_test.go           # Location MCP tool tests
│   ├── middleware/
│   │   └── middleware_test.go               # Middleware chain tests
│   ├── repository/
│   │   └── location_test.go                 # Repository layer tests
│   └── testutil/
│       └── testutil.go                      # Mock implementations
└── pkg/
    ├── auth/
    │   └── auth_test.go                     # Auth package tests
    ├── config/
    │   └── config_test.go                   # Configuration tests
    ├── db/
    │   └── (integration tested via repository)
    ├── metrics/
    │   └── metrics_test.go                  # Metrics tests
    └── model/
        └── model_test.go                    # Model validation tests
```

## Test Utilities

### TestLogHandler

Captures log calls for verification in tests using slog.Handler interface:

```go
logger, logHandler := testutil.NewTestLogger()
// ... run code that logs ...
logHandler.AssertInfoCount(t, 1)
logHandler.AssertErrorCount(t, 0)
```

The `NewTestLogger()` function returns both a `*slog.Logger` (to pass to your code) and a `*TestLogHandler` (to make assertions on logged output).

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

#### Auth Tests (`pkg/auth/auth_test.go`)
- `TestExtractBearerToken` - Bearer token extraction (table-driven)
  - Valid tokens with various formats
  - Invalid tokens and error cases
- `TestHasAnyRole` - Role validation logic
- `TestHasAllPermissions` - Permission validation logic
- `TestHasAnyScope` - Scope validation logic
- `TestAuthorize` - Complete authorization flow (table-driven)
  - Valid cases with roles, permissions, scopes
  - Invalid cases with missing claims

#### Config Tests (`pkg/config/config_test.go`)
- `TestLoad` - Configuration loading from environment
- `TestValidate*` - Configuration validation (table-driven)
  - Port validation
  - Timeout validation
  - CORS validation
  - Auth configuration validation

#### Metrics Tests (`pkg/metrics/metrics_test.go`)
- `TestNew` - Metrics initialization
- `TestSetBuildInfo` - Build info metric
- `TestHTTPMetrics` - HTTP metric collectors
- `TestMCPToolMetrics` - MCP tool metric collectors
- `TestInFlightGauges` - In-flight request gauges

#### Model Tests (`pkg/model/model_test.go`)
- `TestNewTimeResponse` - Time response construction
- `TestTimeResponseWithDifferentTimezones` - Timezone handling
- `TestNewServiceInfo` - Service info construction
- `TestServiceInfoStructure` - Service info structure validation
- `TestValidateName` - Location name validation (table-driven)
- `TestValidateTimezone` - Timezone validation with IANA database
- `TestValidateDescription` - Description length validation
- `TestLocation_Validate` - Complete location validation
- `TestCreateLocationRequest_Validate` - Create request validation
- `TestUpdateLocationRequest_Validate` - Update request validation

#### Repository Tests (`internal/repository/location_test.go`)
- `TestCreate` - Location creation (table-driven)
  - Successful creation with ID assignment
  - Duplicate name handling (case-insensitive)
  - Validation errors
- `TestGetByName` - Location retrieval (table-driven)
  - Successful retrieval
  - Case-insensitive lookup
  - Not found handling
- `TestUpdate` - Location updates (table-driven)
  - Timezone updates
  - Description updates
  - Not found handling
- `TestDelete` - Location deletion (table-driven)
  - Successful deletion
  - Not found handling
- `TestList` - Location listing
  - Empty database
  - Multiple locations with sorting
  - Empty slice (not nil) for consistency
- `TestContextCancellation` - Context cancellation handling
- `TestConcurrentAccess` - Concurrent database operations
- `TestDescriptionHandling` - Empty and nil description handling

**Test Setup:** Uses in-memory SQLite database (`:memory:`) for fast, isolated tests with full schema migrations.

#### Location Handler Integration Tests (`internal/handler/location_integration_test.go`)
- `TestLocationIntegration_FullCRUDWorkflow` - Complete CRUD flow
  - Create → Read → Update → Delete
  - Uses temporary SQLite database
- `TestLocationIntegration_GetLocationTime` - Time for location
  - Validates timezone conversion
  - Format and timezone correctness
- `TestLocationIntegration_ErrorScenarios` - Error handling
  - Invalid JSON payloads
  - Missing required fields
  - Invalid timezones
  - Not found errors
  - Duplicate names
- `TestLocationIntegration_ListLocations` - List endpoint
  - Empty list handling
  - Multiple locations with ordering

**Test Setup:** Uses temporary on-disk SQLite database with full migrations for realistic integration testing.

#### Location MCP Tool Tests (`internal/mcpserver/location_tools_test.go`)
- `TestHandleAddLocation` - Add location tool (table-driven)
  - Successful creation
  - Missing parameters
  - Invalid timezones
  - Duplicate locations
  - Repository errors
- `TestHandleRemoveLocation` - Remove location tool (table-driven)
  - Successful deletion
  - Missing parameters
  - Not found handling
  - Repository errors
- `TestHandleUpdateLocation` - Update location tool (table-driven)
  - Timezone updates
  - Description updates
  - Missing parameters
  - Not found handling
  - Invalid timezones
- `TestHandleListLocations` - List locations tool (table-driven)
  - Multiple locations
  - Empty list handling
  - Repository errors
- `TestHandleGetLocationTime` - Get location time tool (table-driven)
  - Successful time retrieval
  - Missing parameters
  - Location not found
  - Repository errors

**Test Approach:** Uses mock repository for isolated unit testing of MCP tool handlers.

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
func (h *TestLogHandler) AssertInfoCount(t *testing.T, expected int) {
    t.Helper()  // Marks this as a helper function
    if len(h.InfoCalls) != expected {
        t.Errorf("expected %d Info calls, got %d", expected, len(h.InfoCalls))
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

### 5. Dependency Injection with Standard Interfaces
Dependencies use standard Go interfaces (e.g., `http.Handler`, `slog.Handler`), making testing with mocks straightforward without custom interface definitions.

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
