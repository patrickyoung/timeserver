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
│   ├── handler/         # HTTP handlers (time, health, service info, MCP proxy)
│   ├── middleware/      # Logging, recovery, auth, CORS enforcement, Prometheus instrumentation
│   ├── mcpserver/       # MCP tool definitions, execution, metrics hooks
│   └── testutil/        # Shared testing helpers and mocks
└── pkg/
    ├── auth/            # OIDC authentication, JWT verification, claims authorization
    ├── config/          # Environment-driven configuration and validation
    ├── metrics/         # Prometheus collectors, build info instrumentation
    ├── model/           # Response models shared by HTTP and MCP paths
    └── version/         # Canonical service name and version constants
```

- **ADRs** in `docs/adr/` document major architectural decisions (dual interface, configuration guardrails, observability, container hardening, authentication).
- **Supporting manifests** (`Dockerfile`, `docker-compose.yml`, `k8s/`) reflect deployment patterns aligned with the Go code.

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
  - Timeouts, header limits, and port ranges are validated with descriptive error messages.
- `cmd/server/main.go` logs effective configuration at startup to assist operators.
- Logging level defaults to `info` but can be set via `LOG_LEVEL`; stdio mode uses a lightweight configuration path.

---

## 6. Observability
### 6.1 Structured Logging
- Built on Go’s `slog` with JSON output to stderr.
- Request logging includes method, path, status, duration (ms), and remote address.
- MCP tool handlers emit structured events for successful and error cases.

### 6.2 Metrics (Prometheus)
- `pkg/metrics` defines all collectors: HTTP counters/histograms/gauges, MCP tool metrics, auth metrics, and a build info gauge.
- **Auth metrics** (ADR 0005): Track authentication attempts (by path and status), auth duration, and token verification results.
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
- Security design (`docs/SECURITY.md`)
- Testing strategy (`docs/TESTING.md`)
