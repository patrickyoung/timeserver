# Time Service Technical Design Blueprint

This document captures the technical design patterns, conventions, and operational practices embodied in the Time Service project. It is intended as a reusable blueprint for future Go services that expose both REST APIs and Model Context Protocol (MCP) tooling while maintaining strong DevSecOps hygiene.

---

## 1. Design Objectives
- **Single artifact, dual interfaces**: Serve both HTTP API clients and MCP clients from one deployable binary.
- **Secure-by-default posture**: Enforce strict configuration validation, least-privilege runtime, and hardened containers.
- **Operational excellence**: Provide first-class observability (structured logs, Prometheus metrics), health checks, and graceful shutdown.
- **Developer velocity**: Offer a clear project layout, reproducible builds, and high-quality automated tests.

---

## 2. Architecture Overview
### 2.1 Process Modes
- **HTTP mode (default)**: Runs a full HTTP server exposing REST endpoints, MCP over HTTP, and the Prometheus `/metrics` endpoint. See `cmd/server/main.go`.
- **STDIO mode (`--stdio`)**: Launches the same MCP server over standard input/output for local MCP clients (e.g., editors) without HTTP or CORS concerns.

### 2.2 Package Topology
```
.
├── cmd/
│   ├── server/          # Entry point managing mode selection, HTTP mux, graceful shutdown
│   └── healthcheck/     # Tiny probe binary for container health checks
├── internal/
│   ├── handler/         # HTTP handlers (time, health, service info, MCP proxy, locations)
│   ├── middleware/      # Logging, recovery, auth, CORS enforcement, Prometheus instrumentation
│   ├── mcpserver/       # MCP tool definitions, execution, metrics hooks
│   ├── repository/      # Data access layer (location CRUD operations)
│   └── testutil/        # Shared testing helpers and mocks
└── pkg/
    ├── auth/            # OIDC authentication, JWT verification, claims authorization
    ├── config/          # Environment-driven configuration and validation
    ├── db/              # Database connection management, migration engine
    │   └── migrations/  # SQL migration files (*.up.sql, *.down.sql)
    ├── metrics/         # Prometheus collectors, build info instrumentation
    ├── model/           # Response models shared by HTTP and MCP paths
    └── version/         # Canonical service name and version constants
```

- **ADRs** in `docs/adr/` document major architectural decisions (dual interface, configuration guardrails, observability, container hardening, authentication, database choice).
- **Supporting manifests** (`Dockerfile`, `docker-compose.yml`, `k8s/`) reflect deployment patterns aligned with the Go code.
- **Migrations** in `pkg/db/migrations/` define schema changes applied automatically on startup (embedded via go:embed).

---

## 3. Request Handling and Routing
- HTTP routing uses the standard library `http.ServeMux` with method-aware patterns (`mux.HandleFunc("GET /api/time", ...)`).
- `internal/handler` centralizes JSON responses, ensuring consistent logging and error handling across endpoints.
- Middleware is composed through `internal/middleware.Chain`, providing:
  - Prometheus instrumentation (first in chain to capture all requests).
  - Structured logging (`slog`) with duration and status code.
  - Panic recovery producing 500 responses instead of crashes.
  - **Authentication and Authorization** (ADR 0005): JWT validation, claims extraction, and role/permission enforcement.
  - Configurable CORS with explicit allow-list enforcement.
- MCP over HTTP routes requests to `github.com/mark3labs/mcp-go/server.NewStreamableHTTPServer`, sharing tool implementations with stdio mode.

---

## 4. MCP Tooling Design
- Tools are registered in `internal/mcpserver` using the MCP Go SDK.
- Each tool function (`handleGetCurrentTime`, `handleAddTimeOffset`) encapsulates domain logic, logging, and validation.
- Metrics wrapping (`wrapWithMetrics`) captures per-tool success/error counts and durations.
- Tool metadata (descriptions, parameters) is declared alongside handler registration for discoverability.
- Stdio mode leverages the same server instance via `server.ServeStdio`, ensuring parity between transports.

---

## 5. Configuration Strategy
- `pkg/config.Load` reads environment variables, applies defaults, and validates invariants before the server starts.
- Critical safeguards:
  - `ALLOWED_ORIGINS` is mandatory; wildcard CORS requires `ALLOW_CORS_WILDCARD_DEV=true`.
  - **Authentication configuration** (ADR 0005): `OIDC_ISSUER_URL` and `OIDC_AUDIENCE` required when `AUTH_ENABLED=true`.
  - **Database configuration** (ADR 0006): `DB_PATH` defaults to `data/timeservice.db`, configurable for different environments.
  - Timeouts, header limits, and port ranges are validated with descriptive error messages.
- `cmd/server/main.go` logs effective configuration at startup to assist operators.
- Logging level defaults to `info` but can be set via `LOG_LEVEL`; stdio mode uses a lightweight configuration path.

---

## 5a. Data Persistence Layer

### 5a.1 Database Architecture (ADR 0006)

The service uses **SQLite** with **modernc.org/sqlite** (pure Go driver) for named location storage:

```
┌──────────────────────────────────────────────────┐
│              Application Layer                   │
│  (Handlers, MCP Tools)                          │
└─────────────────┬────────────────────────────────┘
                  │
┌─────────────────▼────────────────────────────────┐
│           Repository Layer                       │
│  (internal/repository/location.go)              │
│  - Add, Update, Delete, Get, List               │
│  - Business logic validation                     │
└─────────────────┬────────────────────────────────┘
                  │
┌─────────────────▼────────────────────────────────┐
│          Database Connection Pool                │
│  (pkg/db/db.go)                                 │
│  - Connection management                         │
│  - Performance tuning (WAL, cache, pragmas)     │
│  - Migration engine                              │
└─────────────────┬────────────────────────────────┘
                  │
┌─────────────────▼────────────────────────────────┐
│              SQLite Database                     │
│  (data/timeservice.db)                          │
│  - locations table                               │
│  - schema_migrations table                       │
└──────────────────────────────────────────────────┘
```

**Key characteristics**:
- **Pure Go**: No CGo dependencies, simplified builds and cross-compilation
- **Embedded**: Single-file database, no separate server process
- **Performance optimized**: WAL mode, 64MB cache, connection pooling
- **ACID compliant**: Reliable transactions and crash recovery
- **Auto-migrations**: Schema changes applied on startup from `pkg/db/migrations/`

### 5a.2 Schema Design

```sql
CREATE TABLE locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,  -- Case-insensitive
    timezone TEXT NOT NULL,                      -- IANA timezone
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_locations_name ON locations(name COLLATE NOCASE);
CREATE INDEX idx_locations_timezone ON locations(timezone);
```

**Design rationale**:
- Case-insensitive name for user-friendly lookups
- IANA timezone string validated on insert/update
- Automatic timestamp management via trigger
- Indexes for fast lookups

### 5a.3 Repository Pattern

`internal/repository/location.go` implements the data access layer:

```go
type LocationRepository interface {
    Create(ctx context.Context, loc *model.Location) error
    GetByName(ctx context.Context, name string) (*model.Location, error)
    Update(ctx context.Context, name string, loc *model.Location) error
    Delete(ctx context.Context, name string) error
    List(ctx context.Context) ([]*model.Location, error)
}
```

**Benefits**:
- Abstraction over data access (testable with mocks)
- Centralized validation and error handling
- Transaction management
- Metrics instrumentation
- Easy to swap implementations if migrating to different database

### 5a.4 Migration Management

Migrations are embedded in the binary using go:embed (`pkg/db/db.go`):
- `pkg/db/migrations/*.up.sql` - Schema changes
- `pkg/db/migrations/*.down.sql` - Rollback scripts
- `schema_migrations` table tracks applied versions
- Automatic application on startup
- Transactional (all or nothing)

Migration versioning: `001_create_locations.up.sql`, `002_add_indexes.up.sql`, etc.

### 5a.5 Location Management Features

**Capabilities**:
1. **Add locations**: Store custom location names with IANA timezones
2. **Update locations**: Change timezone or description
3. **Delete locations**: Remove named locations
4. **List locations**: Retrieve all configured locations
5. **Get time by location**: Query current time using location name

**Example locations**:
```json
{
  "name": "headquarters",
  "timezone": "America/New_York",
  "description": "Company HQ in NYC"
},
{
  "name": "tokyo-office",
  "timezone": "Asia/Tokyo",
  "description": "Tokyo branch office"
}
```

**API Endpoints** (new):
- `POST /api/locations` - Create location (requires `locations:write` permission)
- `GET /api/locations` - List all locations
- `GET /api/locations/{name}` - Get specific location
- `PUT /api/locations/{name}` - Update location (requires `locations:write`)
- `DELETE /api/locations/{name}` - Delete location (requires `locations:write`)
- `GET /api/locations/{name}/time` - Get current time for location

**MCP Tools** (new):
- `add_location(name, timezone, description)` - Add named location
- `remove_location(name)` - Remove location
- `update_location(name, timezone, description)` - Update location
- `list_locations()` - List all locations
- `get_location_time(name, format)` - Get time for named location

---

## 6. Observability
### 6.1 Structured Logging
- Built on Go’s `slog` with JSON output to stderr.
- Request logging includes method, path, status, duration (ms), and remote address.
- MCP tool handlers emit structured events for successful and error cases.

### 6.2 Metrics (Prometheus)
- `pkg/metrics` defines all collectors: HTTP counters/histograms/gauges, MCP tool metrics, auth metrics, database metrics, and a build info gauge.
- **Auth metrics** (ADR 0005): Track authentication attempts (by path and status), auth duration, and token verification results.
- **Database metrics** (ADR 0006): Track query duration, connection pool stats, transaction counts, and error rates.
- Middleware and tool wrappers ensure instrumentation is automatic for every handler.
- `/metrics` endpoint is exposed by default and integrated with container/Kubernetes manifests.

### 6.3 Health Checks
- `/health` endpoint returns JSON status.
- Dedicated `cmd/healthcheck` probe binary performs an HTTP GET and exits with appropriate status codes.

---

## 7. Security and DevSecOps Posture
- **Configuration Guardrails**: Fail-fast on missing CORS configuration (ADR 0002).
- **Authentication & Authorization** (ADR 0005):
  - OAuth2/OIDC with JWT-based claims authorization.
  - Provider-agnostic design (Auth0, Okta, Azure Entra ID, Keycloak, etc.).
  - Stateless token verification using public keys from JWKS endpoint.
  - Claims-based access control (roles, permissions, scopes).
  - Configurable public/protected paths with secure-by-default posture.
- **CORS Enforcement**: Middleware checks origins per request and handles preflight (`OPTIONS`) requests securely.
- **Graceful Shutdown**: Signal handling ensures in-flight requests complete within configured timeouts.
- **Container Hardening** (ADR 0004):
  - Multi-stage build produces static binaries.
  - Final image runs as non-root on Alpine, with CA certs and tzdata only.
  - Read-only root filesystem; tmpfs mounts declared in Compose/Kubernetes manifests.
  - Linux capabilities dropped; no-privilege escalation; RuntimeDefault seccomp.
- **Infrastructure as Code**: Docker Compose and Kubernetes manifests pin versions/tags, configure probes, and set resource limits.

---

## 8. Deployment Patterns
### 8.1 Docker
- `Makefile` target `make docker` builds versioned and `latest` tags.
- Containers expose port 8080 and rely on environment variables for runtime configuration.
- Health checks use the probe binary (`HEALTHCHECK` instruction).

### 8.2 Docker Compose
- Hardened defaults: read-only filesystem, dropped capabilities, tmpfs for `/tmp`, resource limits, Prometheus labels.

### 8.3 Kubernetes
- Deployment manifests enforce Pod/Container security contexts and service account usage.
- Readiness/liveness probes execute `/app/healthcheck`.
- ServiceMonitor integrates with Prometheus Operator setups.

---

## 9. Development Workflow
- **Tooling**: `Makefile` orchestrates dependency download, formatting, linting, testing, and image builds.
- **Formatting & Linting**: `go fmt` and optional `golangci-lint`.
- **Testing**: Unit tests across handlers, middleware, MCP server, and models (see `docs/TESTING.md` for detailed coverage).
- **Versioning**: `pkg/version` provides a single source of truth for versions, used in logs and metrics.

---

## 10. Testing Strategy
- Table-driven tests with subtests across packages.
- `internal/testutil` provides a `TestLogHandler` and `MockMCPServer` to capture logs and simulate HTTP handlers.
- Coverage focuses on behavior-rich components; main packages are tested via integration-style tests where feasible.

---

## 11. Extensibility Guidelines
- **Adding HTTP Endpoints**:
  1. Implement handler method using shared JSON helpers.
  2. Register route in `cmd/server/main.go` through the mux.
  3. Ensure middleware chain instrumentation remains intact.
  4. Update models and tests accordingly.
- **Adding MCP Tools**:
  1. Define the tool schema and parameters in `internal/mcpserver`.
  2. Implement handler function with logging and validation.
  3. Wrap with `wrapWithMetrics` to inherit observability.
  4. Extend tests for success and failure scenarios.
- **New Configuration Values**:
  1. Add fields to `pkg/config.Config`.
  2. Parse and validate in `Load`.
  3. Expose the value through startup logs and, if relevant, metrics.
- **Metrics Extensions**: Prefer augmenting `pkg/metrics` to keep namespace consistency and reduce duplicate registration.

---

## 12. Operational Playbook
- **Startup**: Confirm logs show configuration values, auth status, and CORS warnings when wildcard origins are used.
- **Monitoring**: Use Prometheus metrics for dashboards and alerts:
  - HTTP: `timeservice_http_requests_total`, `timeservice_http_request_duration_seconds`
  - MCP: `timeservice_mcp_tool_calls_total`, `timeservice_mcp_tool_call_duration_seconds`
  - Auth: `timeservice_auth_attempts_total`, `timeservice_auth_duration_seconds`, `timeservice_auth_tokens_verified_total`
- **Graceful Shutdown**: On SIGTERM/SIGINT, the server honors `ShutdownTimeout` and drains listeners gracefully.
- **Incident Response**: Structured logs simplify correlation; metrics highlight error rates and latency spikes, especially across MCP tools and authentication failures.

---

## 13. Future Enhancements
- Introduce distributed tracing (OpenTelemetry) to complement structured logging.
- Expand configuration sources (e.g., file-based overrides) while reusing validation logic.
- Provide Helm charts or Terraform modules to codify infrastructure deployment patterns.
- Offer CLI scaffolding to bootstrap new services following this layout.

---

## 14. References
- ADR 0001 – Dual Protocol Interface (`docs/adr/0001-dual-protocol-interface.md`)
- ADR 0002 – Configuration Guardrails for CORS and Startup Safety (`docs/adr/0002-configuration-guardrails.md`)
- ADR 0003 – Prometheus Instrumentation for HTTP and MCP Paths (`docs/adr/0003-prometheus-instrumentation.md`)
- ADR 0004 – Hardened Container Packaging for Deployments (`docs/adr/0004-hardened-container-packaging.md`)
- ADR 0005 – OAuth2/OIDC JWT-Based Authentication and Authorization (`docs/adr/0005-oauth2-oidc-jwt-authorization.md`)
- ADR 0006 – SQLite for Named Location Storage (`docs/adr/0006-sqlite-location-database.md`)
- Security design (`docs/SECURITY.md`)
- DevSecOps review (`docs/DEVSECOPS.md`)
- Testing strategy (`docs/TESTING.md`)
- Implementation plans (`docs/implementation-plans/`)
  - Plan 001: Foundation and Authentication (`001-foundation-and-auth.md`)
  - Plan 002: Named Locations (`002-named-locations.md`)
