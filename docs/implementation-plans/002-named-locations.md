# Named Locations Feature - Implementation Plan

This document outlines a phased implementation plan for adding named location management to the Time Service. Each phase is designed to be completed in a single work session with independent testing and can be committed separately.

**Feature Overview**: Allow users to store, manage, and query named locations with associated IANA timezones.

**Architecture Decision**: See [ADR 0006](adr/0006-sqlite-location-database.md) for database choice and rationale.

---

## Table of Contents

1. [Implementation Phases](#implementation-phases)
2. [Phase 0: Foundation (COMPLETED)](#phase-0-foundation-completed)
3. [Phase 1: Database Models and Repository](#phase-1-database-models-and-repository)
4. [Phase 2: Location HTTP API Endpoints](#phase-2-location-http-api-endpoints)
5. [Phase 3: MCP Tools for Location Management](#phase-3-mcp-tools-for-location-management)
6. [Phase 4: Database Configuration and Integration](#phase-4-database-configuration-and-integration)
7. [Phase 5: Container and Kubernetes Support](#phase-5-container-and-kubernetes-support)
8. [Phase 6: Documentation and Polish](#phase-6-documentation-and-polish)
9. [Testing Strategy](#testing-strategy)
10. [Rollback Plan](#rollback-plan)

---

## Implementation Phases

**Total Phases**: 6 (Phase 0 completed)
**Estimated Total Time**: 6-8 hours across 3-4 sessions
**Status**: Phase 0 complete, ready for Phase 1

---

## Phase 0: Foundation (COMPLETED ✅)

**Objective**: Research, architecture decision, and database infrastructure

**Completed Work**:
- ✅ ADR 0006 - SQLite vs Turso comparison and decision
- ✅ Database schema design (`db/migrations/001_create_locations.up.sql`)
- ✅ Migration rollback script (`db/migrations/001_create_locations.down.sql`)
- ✅ Database connection package (`pkg/db/db.go` - 295 lines)
  - Connection management with performance pragmas
  - WAL mode, cache configuration, busy timeout
  - Embedded migration system
  - Auto-migration on startup
- ✅ Dependency added: `modernc.org/sqlite v1.39.1`
- ✅ DESIGN.md updated with database architecture section

**Deliverables**:
- [x] ADR 0006 document
- [x] Migration files (up/down)
- [x] pkg/db package with connection management
- [x] DESIGN.md updates

**Commit Message**:
```
feat: add SQLite database foundation for named locations

- Add ADR 0006 documenting SQLite choice and rationale
- Create locations table migration with indexes
- Implement pkg/db with connection management and auto-migrations
- Configure WAL mode and performance optimizations
- Update DESIGN.md with database architecture

See ADR 0006 for detailed decision rationale.
```

---

## Phase 1: Database Models and Repository

**Objective**: Implement the data access layer with full CRUD operations

**Estimated Time**: 1.5-2 hours

**Prerequisites**: Phase 0 complete ✅

### 1.1 Location Model

**File**: `pkg/model/location.go`

**Tasks**:
- [ ] Create `Location` struct with JSON tags
- [ ] Add validation methods (`ValidateTimezone()`, `ValidateName()`)
- [ ] Create request/response DTOs:
  - `CreateLocationRequest`
  - `UpdateLocationRequest`
  - `LocationResponse`
  - `LocationListResponse`
- [ ] Add constructor functions

**Example**:
```go
type Location struct {
    ID          int64     `json:"id"`
    Name        string    `json:"name"`
    Timezone    string    `json:"timezone"`
    Description string    `json:"description,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**Validation**:
- Name: 1-100 characters, alphanumeric + hyphens/underscores
- Timezone: Valid IANA timezone (use `time.LoadLocation()`)
- Description: 0-500 characters (optional)

### 1.2 Location Repository

**File**: `internal/repository/location.go`

**Tasks**:
- [ ] Define `LocationRepository` interface
- [ ] Implement `sqliteLocationRepository` struct
- [ ] CRUD operations:
  - `Create(ctx, location)` - INSERT with conflict handling
  - `GetByName(ctx, name)` - SELECT with case-insensitive lookup
  - `Update(ctx, name, location)` - UPDATE with optimistic locking
  - `Delete(ctx, name)` - DELETE with existence check
  - `List(ctx)` - SELECT all, ordered by name
- [ ] Error handling:
  - `ErrLocationNotFound` - Custom error type
  - `ErrLocationExists` - Duplicate name conflict
  - `ErrInvalidTimezone` - Invalid IANA timezone
- [ ] Context cancellation support
- [ ] Prepared statement reuse (optional performance optimization)

**Interface**:
```go
type LocationRepository interface {
    Create(ctx context.Context, loc *model.Location) error
    GetByName(ctx context.Context, name string) (*model.Location, error)
    Update(ctx context.Context, name string, loc *model.Location) error
    Delete(ctx context.Context, name string) error
    List(ctx context.Context) ([]*model.Location, error)
}
```

### 1.3 Repository Tests

**File**: `internal/repository/location_test.go`

**Tasks**:
- [ ] Setup: In-memory SQLite database for tests
- [ ] Test `Create`:
  - Success case
  - Duplicate name conflict
  - Invalid timezone
  - Empty name
- [ ] Test `GetByName`:
  - Existing location
  - Non-existent location
  - Case-insensitive lookup
- [ ] Test `Update`:
  - Success case
  - Non-existent location
  - Change timezone
  - Invalid timezone
- [ ] Test `Delete`:
  - Success case
  - Non-existent location
  - Idempotency
- [ ] Test `List`:
  - Empty database
  - Multiple locations
  - Ordering (alphabetical by name)
- [ ] Test concurrent access (WAL mode)
- [ ] Test context cancellation

**Target Coverage**: >80%

### 1.4 Model Tests

**File**: `pkg/model/location_test.go`

**Tasks**:
- [ ] Test `ValidateTimezone` with various timezones
- [ ] Test `ValidateName` with edge cases
- [ ] Test JSON marshaling/unmarshaling
- [ ] Test request DTO validation

### Deliverables

- [ ] `pkg/model/location.go` (~100 lines)
- [ ] `pkg/model/location_test.go` (~150 lines)
- [ ] `internal/repository/location.go` (~250 lines)
- [ ] `internal/repository/location_test.go` (~400 lines)

### Testing Phase 1

```bash
# Run tests
go test ./pkg/model -v
go test ./internal/repository -v

# Check coverage
go test ./pkg/model -cover
go test ./internal/repository -cover

# Build verification
go build ./cmd/server
```

### Commit Message

```
feat: implement location model and repository layer

- Add Location model with validation
- Implement LocationRepository interface
- Create sqliteLocationRepository with full CRUD
- Add comprehensive tests for model and repository
- Handle edge cases (duplicates, invalid timezones, case-insensitive)

Coverage: model 95%, repository 85%
```

---

## Phase 2: Location HTTP API Endpoints

**Objective**: Expose location management via REST API

**Estimated Time**: 2-2.5 hours

**Prerequisites**: Phase 1 complete

### 2.1 Location Handler

**File**: `internal/handler/location.go`

**Tasks**:
- [ ] Create `LocationHandler` struct with repository dependency
- [ ] Implement HTTP handlers:
  - `CreateLocation` - POST /api/locations
  - `GetLocation` - GET /api/locations/{name}
  - `UpdateLocation` - PUT /api/locations/{name}
  - `DeleteLocation` - DELETE /api/locations/{name}
  - `ListLocations` - GET /api/locations
  - `GetLocationTime` - GET /api/locations/{name}/time
- [ ] Request validation and error responses
- [ ] JSON serialization/deserialization
- [ ] HTTP status codes:
  - 200 OK (GET, PUT success)
  - 201 Created (POST success)
  - 204 No Content (DELETE success)
  - 400 Bad Request (validation errors)
  - 404 Not Found (location doesn't exist)
  - 409 Conflict (duplicate name)
  - 500 Internal Server Error (database errors)
- [ ] Structured logging for all operations
- [ ] Extract claims from auth context (optional, if auth enabled)

**Example**:
```go
func (h *LocationHandler) CreateLocation(w http.ResponseWriter, r *http.Request) {
    var req model.CreateLocationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.errorJSON(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    loc := &model.Location{
        Name:        req.Name,
        Timezone:    req.Timezone,
        Description: req.Description,
    }

    if err := h.repo.Create(r.Context(), loc); err != nil {
        // Handle errors...
    }

    h.json(w, loc, http.StatusCreated)
}
```

### 2.2 Route Registration

**File**: `cmd/server/main.go` (update)

**Tasks**:
- [ ] Initialize database connection in main()
- [ ] Run migrations on startup
- [ ] Create location repository
- [ ] Create location handler
- [ ] Register routes in HTTP mux
- [ ] Apply auth middleware to protected routes (write operations)
- [ ] Add graceful database shutdown

**Routes**:
```go
// Location management endpoints
mux.HandleFunc("POST /api/locations", locationHandler.CreateLocation)
mux.HandleFunc("GET /api/locations", locationHandler.ListLocations)
mux.HandleFunc("GET /api/locations/{name}", locationHandler.GetLocation)
mux.HandleFunc("PUT /api/locations/{name}", locationHandler.UpdateLocation)
mux.HandleFunc("DELETE /api/locations/{name}", locationHandler.DeleteLocation)
mux.HandleFunc("GET /api/locations/{name}/time", locationHandler.GetLocationTime)
```

### 2.3 Handler Tests

**File**: `internal/handler/location_test.go`

**Tasks**:
- [ ] Mock repository implementation
- [ ] Test each HTTP handler:
  - Success cases with valid input
  - Validation errors (400)
  - Not found errors (404)
  - Conflict errors (409)
  - Repository errors (500)
- [ ] Test request/response JSON format
- [ ] Test URL parameter extraction
- [ ] Test auth integration (if enabled)
- [ ] Table-driven tests for comprehensive coverage

**Target Coverage**: >85%

### 2.4 Integration Test

**File**: `internal/handler/location_integration_test.go`

**Tasks**:
- [ ] End-to-end test with real SQLite database
- [ ] Test full CRUD workflow:
  1. Create location
  2. Get location
  3. List locations
  4. Update location
  5. Get time for location
  6. Delete location
- [ ] Test error scenarios
- [ ] Test concurrent requests

### Deliverables

- [ ] `internal/handler/location.go` (~400 lines)
- [ ] `internal/handler/location_test.go` (~500 lines)
- [ ] Updated `cmd/server/main.go` (database initialization)
- [ ] Optional: `internal/handler/location_integration_test.go` (~200 lines)

### Testing Phase 2

```bash
# Unit tests
go test ./internal/handler -v -run TestLocation

# Integration tests
go test ./internal/handler -v -run TestLocationIntegration

# Manual API testing
curl -X POST http://localhost:8080/api/locations \
  -H "Content-Type: application/json" \
  -d '{"name":"hq","timezone":"America/New_York","description":"Headquarters"}'

curl http://localhost:8080/api/locations

curl http://localhost:8080/api/locations/hq/time
```

### Commit Message

```
feat: add location management HTTP API

- Implement LocationHandler with full CRUD operations
- Add 6 new REST endpoints for location management
- Integrate with existing auth middleware for write protection
- Add comprehensive handler tests with mock repository
- Add integration tests with real database

Endpoints:
  POST   /api/locations - Create location
  GET    /api/locations - List all locations
  GET    /api/locations/{name} - Get location details
  PUT    /api/locations/{name} - Update location
  DELETE /api/locations/{name} - Delete location
  GET    /api/locations/{name}/time - Get time for location
```

---

## Phase 3: MCP Tools for Location Management

**Objective**: Expose location management via MCP tools

**Estimated Time**: 1.5-2 hours

**Prerequisites**: Phase 2 complete

### 3.1 MCP Tool Implementations

**File**: `internal/mcpserver/location_tools.go`

**Tasks**:
- [ ] Implement MCP tool handlers:
  - `handleAddLocation` - Add named location
  - `handleRemoveLocation` - Remove location
  - `handleUpdateLocation` - Update location
  - `handleListLocations` - List all locations
  - `handleGetLocationTime` - Get time for named location
- [ ] Parameter parsing and validation
- [ ] Error handling with user-friendly messages
- [ ] Structured logging
- [ ] Metrics wrapping (reuse existing `wrapWithMetrics`)

**Example**:
```go
func (s *MCPServer) handleAddLocation(args map[string]interface{}) (interface{}, error) {
    name := getStringParam(args, "name")
    timezone := getStringParam(args, "timezone")
    description := getStringParam(args, "description")

    loc := &model.Location{
        Name:        name,
        Timezone:    timezone,
        Description: description,
    }

    if err := s.locationRepo.Create(context.Background(), loc); err != nil {
        return nil, fmt.Errorf("failed to add location: %w", err)
    }

    return map[string]interface{}{
        "success": true,
        "location": loc,
    }, nil
}
```

### 3.2 Tool Registration

**File**: `internal/mcpserver/server.go` (update)

**Tasks**:
- [ ] Add location repository to MCPServer struct
- [ ] Register new tools in `NewServer()`:
  - `add_location`
  - `remove_location`
  - `update_location`
  - `list_locations`
  - `get_location_time`
- [ ] Define tool schemas with parameters
- [ ] Wrap handlers with metrics

**Tool Definitions**:
```go
server.AddTool(
    "add_location",
    "Add a named location with timezone",
    map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "name":        map[string]string{"type": "string", "description": "Location name"},
            "timezone":    map[string]string{"type": "string", "description": "IANA timezone"},
            "description": map[string]string{"type": "string", "description": "Optional description"},
        },
        "required": []string{"name", "timezone"},
    },
    wrapWithMetrics("add_location", s.handleAddLocation, metrics),
)
```

### 3.3 MCP Tool Tests

**File**: `internal/mcpserver/location_tools_test.go`

**Tasks**:
- [ ] Test each MCP tool with mock repository
- [ ] Test parameter validation
- [ ] Test success cases
- [ ] Test error cases (duplicate, not found, invalid timezone)
- [ ] Test response format

**Target Coverage**: >80%

### Deliverables

- [ ] `internal/mcpserver/location_tools.go` (~300 lines)
- [ ] Updated `internal/mcpserver/server.go` (tool registration)
- [ ] `internal/mcpserver/location_tools_test.go` (~400 lines)

### Testing Phase 3

```bash
# Unit tests
go test ./internal/mcpserver -v -run TestLocation

# Manual MCP testing (stdio mode)
echo '{"method":"tools/list"}' | go run cmd/server/main.go --stdio

echo '{"method":"tools/call","params":{"name":"add_location","arguments":{"name":"hq","timezone":"America/New_York"}}}' | go run cmd/server/main.go --stdio
```

### Commit Message

```
feat: add location management MCP tools

- Implement 5 new MCP tools for location management
- Add handleAddLocation, handleRemoveLocation, handleUpdateLocation
- Add handleListLocations, handleGetLocationTime
- Integrate with existing metrics wrapping
- Add comprehensive tests for all tools

MCP Tools:
  add_location - Add named location
  remove_location - Remove location
  update_location - Update location
  list_locations - List all locations
  get_location_time - Get time for location
```

---

## Phase 4: Database Configuration and Integration

**Objective**: Add database configuration and polish integration

**Estimated Time**: 1-1.5 hours

**Prerequisites**: Phase 3 complete

### 4.1 Database Configuration

**File**: `pkg/config/config.go` (update)

**Tasks**:
- [ ] Add database configuration fields to Config struct:
  - `DBPath` - Database file path
  - `DBMaxOpenConns` - Max open connections
  - `DBMaxIdleConns` - Max idle connections
  - `DBCacheSize` - Cache size in KB
  - `DBWalMode` - Enable WAL mode
- [ ] Add environment variable parsing
- [ ] Add validation rules
- [ ] Add defaults (from `db.DefaultConfig()`)

**Environment Variables**:
```bash
DB_PATH=data/timeservice.db
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CACHE_SIZE_KB=64000
DB_WAL_MODE=true
```

### 4.2 Database Metrics

**File**: `pkg/metrics/metrics.go` (update)

**Tasks**:
- [ ] Add database metric collectors:
  - `DBQueryDuration` - Histogram of query latencies
  - `DBQueriesTotal` - Counter of queries by operation
  - `DBConnectionsOpen` - Gauge of open connections
  - `DBConnectionsIdle` - Gauge of idle connections
  - `DBErrorsTotal` - Counter of database errors
- [ ] Export metrics for Prometheus scraping

**Metrics**:
```go
DBQueryDuration: promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Namespace: namespace,
        Name:      "db_query_duration_seconds",
        Help:      "Database query duration in seconds",
        Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
    },
    []string{"operation"},
),
```

### 4.3 Repository Metrics Integration

**File**: `internal/repository/location.go` (update)

**Tasks**:
- [ ] Add metrics to repository struct
- [ ] Instrument CRUD operations:
  - Record query duration
  - Count operations (create, get, update, delete, list)
  - Track errors
- [ ] Record connection pool stats periodically

### 4.4 Startup Integration

**File**: `cmd/server/main.go` (update)

**Tasks**:
- [ ] Load database config from environment
- [ ] Log database configuration on startup
- [ ] Initialize database with config
- [ ] Run migrations with logging
- [ ] Create repository with metrics
- [ ] Graceful database shutdown in signal handler
- [ ] Log database statistics on shutdown

### 4.5 Configuration Tests

**File**: `pkg/config/config_test.go` (update)

**Tasks**:
- [ ] Test database config defaults
- [ ] Test environment variable parsing
- [ ] Test validation (invalid paths, negative values)

### Deliverables

- [ ] Updated `pkg/config/config.go` (~50 lines added)
- [ ] Updated `pkg/metrics/metrics.go` (~80 lines added)
- [ ] Updated `internal/repository/location.go` (~50 lines added)
- [ ] Updated `cmd/server/main.go` (~60 lines added)
- [ ] Updated tests

### Testing Phase 4

```bash
# Test with custom config
DB_PATH=test.db DB_MAX_OPEN_CONNS=10 go run cmd/server/main.go

# Check metrics
curl http://localhost:8080/metrics | grep db_

# Test migrations
rm -f data/timeservice.db
go run cmd/server/main.go
# Should see migration logs

# Run all tests
go test ./... -cover
```

### Commit Message

```
feat: add database configuration and metrics

- Add database config to pkg/config with env var support
- Add database metrics (query duration, connections, errors)
- Instrument repository operations with metrics
- Integrate database initialization into server startup
- Add graceful database shutdown
- Update tests for database configuration

Environment variables:
  DB_PATH, DB_MAX_OPEN_CONNS, DB_MAX_IDLE_CONNS,
  DB_CACHE_SIZE_KB, DB_WAL_MODE
```

---

## Phase 5: Container and Kubernetes Support

**Objective**: Ensure database persistence in containerized environments

**Estimated Time**: 1-1.5 hours

**Prerequisites**: Phase 4 complete

### 5.1 Dockerfile Updates

**File**: `Dockerfile` (update)

**Tasks**:
- [ ] Create `/app/data` directory in final image
- [ ] Set ownership to appuser (UID 10001)
- [ ] Document volume mount point
- [ ] Add comment about database persistence

**Changes**:
```dockerfile
# Create data directory for database (mount volume here)
RUN mkdir -p /app/data && chown -R appuser:appuser /app/data

# Database file will be stored in /app/data (mount a volume for persistence)
VOLUME ["/app/data"]
```

### 5.2 Docker Compose Updates

**File**: `docker-compose.yml` (update)

**Tasks**:
- [ ] Add volume mount for database persistence
- [ ] Add database configuration environment variables
- [ ] Add comments about backup strategy
- [ ] Add volume definition

**Changes**:
```yaml
services:
  timeservice:
    volumes:
      - timeservice-data:/app/data
    environment:
      - DB_PATH=/app/data/timeservice.db
      - DB_MAX_OPEN_CONNS=25

volumes:
  timeservice-data:
    driver: local
```

### 5.3 Kubernetes Manifests

**File**: `k8s/deployment.yaml` (update)

**Tasks**:
- [ ] Change from Deployment to StatefulSet (for stable storage)
- [ ] Add PersistentVolumeClaim template
- [ ] Add volume mount for database
- [ ] Add database configuration environment variables
- [ ] Add init container for permissions (optional)
- [ ] Add database backup CronJob example (optional)

**StatefulSet**:
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: timeservice
spec:
  serviceName: timeservice
  replicas: 1  # Single instance for SQLite
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi
  template:
    spec:
      containers:
      - name: timeservice
        volumeMounts:
        - name: data
          mountPath: /app/data
        env:
        - name: DB_PATH
          value: "/app/data/timeservice.db"
```

### 5.4 Database Backup Script

**File**: `scripts/backup-db.sh` (new)

**Tasks**:
- [ ] Create backup script using SQLite VACUUM INTO
- [ ] Add timestamp to backup filename
- [ ] Add retention policy (keep last N backups)
- [ ] Make script executable

**Script**:
```bash
#!/bin/bash
DB_PATH=${1:-data/timeservice.db}
BACKUP_DIR=${2:-backups}
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"
sqlite3 "$DB_PATH" "VACUUM INTO '$BACKUP_DIR/timeservice_$TIMESTAMP.db'"
echo "Backup created: $BACKUP_DIR/timeservice_$TIMESTAMP.db"

# Keep only last 7 backups
ls -t "$BACKUP_DIR"/timeservice_*.db | tail -n +8 | xargs -r rm
```

### 5.5 Documentation

**File**: `k8s/README.md` (new)

**Tasks**:
- [ ] Document StatefulSet usage
- [ ] Explain PersistentVolume requirements
- [ ] Document backup/restore procedure
- [ ] Add scaling considerations (SQLite = single instance)

### Deliverables

- [ ] Updated `Dockerfile` (~10 lines added)
- [ ] Updated `docker-compose.yml` (~20 lines added)
- [ ] Updated `k8s/deployment.yaml` (convert to StatefulSet, ~100 lines)
- [ ] New `k8s/pvc.yaml` (PersistentVolumeClaim example)
- [ ] New `scripts/backup-db.sh` (~30 lines)
- [ ] New `k8s/README.md` (~100 lines)

### Testing Phase 5

```bash
# Test Docker Compose with persistence
docker-compose up -d
curl -X POST http://localhost:8080/api/locations \
  -d '{"name":"test","timezone":"UTC"}'
docker-compose down
docker-compose up -d
curl http://localhost:8080/api/locations
# Should see "test" location persisted

# Test database backup script
./scripts/backup-db.sh data/timeservice.db backups/
ls -lh backups/

# Test Kubernetes (minikube/kind)
kubectl apply -f k8s/
kubectl get statefulset timeservice
kubectl get pvc
```

### Commit Message

```
feat: add container persistence for database

- Update Dockerfile with data volume mount point
- Add volume configuration to docker-compose.yml
- Convert k8s Deployment to StatefulSet for stable storage
- Add PersistentVolumeClaim for database persistence
- Create database backup script
- Add k8s README with deployment instructions

Database now persists across container restarts.
Includes backup strategy and restore procedures.
```

---

## Phase 6: Documentation and Polish

**Objective**: Update documentation and add final polish

**Estimated Time**: 1-1.5 hours

**Prerequisites**: Phase 5 complete

### 6.1 README Updates

**File**: `README.md` (update)

**Tasks**:
- [ ] Add "Named Locations" section to Features
- [ ] Document new API endpoints
- [ ] Add MCP tools documentation
- [ ] Add database configuration section
- [ ] Add backup/restore examples
- [ ] Update Quick Start with location examples

**Example Section**:
```markdown
### Named Locations

Manage custom location names with associated timezones:

```bash
# Add a location
curl -X POST http://localhost:8080/api/locations \
  -H "Content-Type: application/json" \
  -d '{"name":"hq","timezone":"America/New_York","description":"Headquarters"}'

# Get time for location
curl http://localhost:8080/api/locations/hq/time
```
```

### 6.2 TESTING.md Updates

**File**: `docs/TESTING.md` (update)

**Tasks**:
- [ ] Update test coverage summary
- [ ] Add repository tests section
- [ ] Add location handler tests section
- [ ] Add location MCP tests section
- [ ] Document integration testing approach

### 6.3 DEVSECOPS.md Updates

**File**: `docs/DEVSECOPS.md` (update)

**Tasks**:
- [ ] Add database security considerations
- [ ] Document backup security (encryption at rest)
- [ ] Add SQL injection prevention (parameterized queries)
- [ ] Update dependency list with modernc.org/sqlite

### 6.4 Example Data

**File**: `scripts/seed-locations.sh` (new)

**Tasks**:
- [ ] Create script to seed example locations
- [ ] Add sample locations for major cities/timezones
- [ ] Make script idempotent (check existence)

**Example**:
```bash
#!/bin/bash
API_URL=${1:-http://localhost:8080}

locations=(
  '{"name":"hq","timezone":"America/New_York","description":"Headquarters"}'
  '{"name":"tokyo","timezone":"Asia/Tokyo","description":"Tokyo Office"}'
  '{"name":"london","timezone":"Europe/London","description":"London Office"}'
  '{"name":"sydney","timezone":"Australia/Sydney","description":"Sydney Office"}'
)

for loc in "${locations[@]}"; do
  curl -X POST "$API_URL/api/locations" \
    -H "Content-Type: application/json" \
    -d "$loc"
done
```

### 6.5 OpenAPI Specification

**File**: `docs/openapi.yaml` (new, optional)

**Tasks**:
- [ ] Create OpenAPI 3.0 spec for all endpoints
- [ ] Document request/response schemas
- [ ] Add examples
- [ ] Generate with Swagger UI link

### 6.6 Final Polish

**Tasks**:
- [ ] Run `gofmt` on all files
- [ ] Run `golangci-lint` and fix issues
- [ ] Update go.mod/go.sum
- [ ] Verify all tests pass
- [ ] Check test coverage (target >75%)
- [ ] Build Docker image and verify size
- [ ] Test end-to-end workflow

### Deliverables

- [ ] Updated `README.md` (~100 lines added)
- [ ] Updated `docs/TESTING.md` (~50 lines added)
- [ ] Updated `docs/DEVSECOPS.md` (~50 lines added)
- [ ] New `scripts/seed-locations.sh` (~30 lines)
- [ ] Optional: `docs/openapi.yaml` (~300 lines)

### Testing Phase 6

```bash
# Full test suite
make test-coverage

# Lint check
make lint

# Build and run
make build
./bin/server

# Seed example data
./scripts/seed-locations.sh

# Full integration test
curl http://localhost:8080/api/locations | jq
```

### Commit Message

```
docs: update documentation for location features

- Add Named Locations section to README
- Document all new API endpoints and MCP tools
- Add database configuration guide
- Update TESTING.md with repository and handler tests
- Update DEVSECOPS.md with database security notes
- Add seed script for example locations
- Add OpenAPI specification (optional)

Documentation is now complete for all location features.
```

---

## Testing Strategy

### Unit Testing

**Per-phase tests**:
- Phase 1: Model and repository tests (>80% coverage)
- Phase 2: Handler tests with mocks (>85% coverage)
- Phase 3: MCP tool tests with mocks (>80% coverage)
- Phase 4: Configuration tests

**Running tests**:
```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Testing

**End-to-end workflows**:
1. Start server with fresh database
2. Create multiple locations via API
3. Verify locations via API
4. Update location via API
5. Get time for location
6. Delete location
7. Verify via MCP tools
8. Test with authentication enabled

**Test script** (`scripts/integration-test.sh`):
```bash
#!/bin/bash
set -e

API_URL=http://localhost:8080

# Create location
curl -f -X POST $API_URL/api/locations \
  -H "Content-Type: application/json" \
  -d '{"name":"test","timezone":"UTC"}'

# Get location
curl -f $API_URL/api/locations/test | jq

# Get time for location
curl -f $API_URL/api/locations/test/time | jq

# List locations
curl -f $API_URL/api/locations | jq

# Delete location
curl -f -X DELETE $API_URL/api/locations/test

echo "✓ All integration tests passed"
```

### Performance Testing

**Database performance**:
- Benchmark repository operations
- Test with 1000+ locations
- Measure query latency
- Monitor connection pool

**Benchmark** (`internal/repository/location_bench_test.go`):
```go
func BenchmarkLocationCreate(b *testing.B) {
    repo := setupBenchRepo(b)
    for i := 0; i < b.N; i++ {
        loc := &model.Location{
            Name:     fmt.Sprintf("location-%d", i),
            Timezone: "UTC",
        }
        repo.Create(context.Background(), loc)
    }
}
```

### Manual Testing Checklist

- [ ] Server starts successfully with fresh database
- [ ] Migrations apply correctly
- [ ] Can create location via API
- [ ] Can create location via MCP tool
- [ ] Case-insensitive name lookup works
- [ ] Invalid timezone rejected
- [ ] Duplicate name rejected
- [ ] Update changes timezone
- [ ] Delete removes location
- [ ] List returns all locations
- [ ] Get time for location returns correct timezone
- [ ] Database persists across restarts
- [ ] Backup script works
- [ ] Metrics exported correctly
- [ ] Auth integration works (if enabled)
- [ ] Concurrent requests handled correctly

---

## Rollback Plan

### Per-Phase Rollback

Each phase is independently committable and can be rolled back:

**Phase 1 rollback**:
```bash
git revert <commit-hash>  # Removes model and repository
# No migration to rollback (not applied yet)
```

**Phase 2 rollback**:
```bash
git revert <commit-hash>  # Removes HTTP handlers
# Database still works, just no API access
```

**Phase 3 rollback**:
```bash
git revert <commit-hash>  # Removes MCP tools
# API still works, MCP tools removed
```

**Phase 4+ rollback**:
```bash
git revert <commit-hash>  # Removes config/metrics/docs
```

### Database Migration Rollback

If migrations cause issues:

```bash
# Manual rollback (if needed)
sqlite3 data/timeservice.db < db/migrations/001_create_locations.down.sql

# Or delete database and start fresh
rm -f data/timeservice.db
```

### Feature Flag (Optional)

Add feature flag to disable location features:

```go
// In config
LocationsEnabled bool  // Default: true

// In main.go
if cfg.LocationsEnabled {
    // Register location routes
}
```

Environment variable: `LOCATIONS_ENABLED=false`

---

## Success Criteria

### Phase Completion Criteria

Each phase is complete when:
- [ ] All code written and formatted
- [ ] All tests passing
- [ ] Test coverage meets target (>75%)
- [ ] No linter errors
- [ ] Documentation updated
- [ ] Manual testing complete
- [ ] Code reviewed (if team workflow)
- [ ] Committed with descriptive message

### Feature Completion Criteria

The entire feature is complete when:
- [ ] All 6 phases completed
- [ ] Full CRUD operations work via API and MCP
- [ ] Database persists correctly
- [ ] Backup/restore tested
- [ ] Container deployment works
- [ ] Kubernetes deployment works
- [ ] Documentation complete
- [ ] No known bugs
- [ ] Performance acceptable (<100ms query latency)
- [ ] Overall test coverage >75%

---

## Timeline Estimate

**Optimistic** (experienced developer, no blockers):
- Phase 1: 1.5 hours
- Phase 2: 2 hours
- Phase 3: 1.5 hours
- Phase 4: 1 hour
- Phase 5: 1 hour
- Phase 6: 1 hour
- **Total**: 8 hours (2 focused work sessions)

**Realistic** (includes breaks, debugging, reviews):
- Phase 1: 2 hours
- Phase 2: 2.5 hours
- Phase 3: 2 hours
- Phase 4: 1.5 hours
- Phase 5: 1.5 hours
- Phase 6: 1.5 hours
- **Total**: 11 hours (3-4 work sessions)

**Conservative** (learning, thorough testing, polish):
- Phase 1: 2.5 hours
- Phase 2: 3 hours
- Phase 3: 2.5 hours
- Phase 4: 2 hours
- Phase 5: 2 hours
- Phase 6: 2 hours
- **Total**: 14 hours (4-5 work sessions)

---

## Next Steps

**Current Status**: Phase 0 complete ✅

**Ready to begin**: Phase 1 - Database Models and Repository

**To start Phase 1**:
```bash
# Create model file
touch pkg/model/location.go
touch pkg/model/location_test.go

# Create repository files
mkdir -p internal/repository
touch internal/repository/location.go
touch internal/repository/location_test.go

# Run tests (should fail initially)
go test ./pkg/model ./internal/repository
```

**Recommended session plan**:
- Session 1: Phases 1 + 2 (models, repository, HTTP API)
- Session 2: Phases 3 + 4 (MCP tools, config, metrics)
- Session 3: Phases 5 + 6 (containers, documentation)

---

## Questions and Decisions

### Open Questions

1. **Authentication for MCP tools**: Inherit HTTP auth or separate mechanism?
   - **Decision**: Inherit HTTP endpoint permissions when auth enabled

2. **Location name uniqueness**: Case-sensitive or case-insensitive?
   - **Decision**: Case-insensitive (better UX, prevents "HQ" vs "hq" confusion)

3. **Database location in containers**: `/app/data` or `/data`?
   - **Decision**: `/app/data` (follows existing app structure)

4. **Scaling strategy**: Single instance or multiple replicas?
   - **Decision**: Single instance (SQLite limitation, document in ADR)

5. **Backup frequency**: Manual or automated CronJob?
   - **Decision**: Provide script, let operators decide (document both)

### Deferred Features

Features to consider for future phases:
- **Location groups/categories** (e.g., "offices", "datacenters")
- **Location metadata** (country, city, coordinates)
- **Bulk import/export** (CSV, JSON)
- **Location usage statistics** (most queried locations)
- **Time zone change notifications** (DST transitions)
- **Multi-database support** (PostgreSQL for scale)
- **Read replicas** (if using Turso/libSQL)
- **GraphQL API** (alternative to REST)

---

## Conclusion

This phased plan provides a structured approach to implementing the named locations feature while maintaining code quality, test coverage, and operational excellence.

Each phase is:
- ✅ Independently testable
- ✅ Committable as working code
- ✅ Reversible if needed
- ✅ Documented with clear deliverables

**Total estimated effort**: 8-14 hours across 3-5 sessions

**Current status**: Foundation complete, ready to begin Phase 1

**Next action**: Create model and repository (Phase 1)
