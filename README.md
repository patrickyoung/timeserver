# Time Service

A simple Go web service that provides the current server time through both a standard REST API and a Model Context Protocol (MCP) server interface over HTTP.

## Features

- **REST API**: Simple endpoint to get current server time
- **MCP Server**: Model Context Protocol server with time-related tools
- **Structured Logging**: JSON-formatted logs with slog
- **Graceful Shutdown**: Proper cleanup on termination signals
- **Middleware Stack**: Logging, recovery, and CORS support
- **Minimal Docker Image**: Multi-stage build producing <10MB images

## Quick Start

**IMPORTANT**: The server requires CORS configuration to start. For local development, use the `ALLOW_CORS_WILDCARD_DEV=true` environment variable.

### Run Locally

#### HTTP Server Mode (for remote access)

```bash
# Download dependencies
make deps

# Run the server (development mode with wildcard CORS)
ALLOW_CORS_WILDCARD_DEV=true make run
```

The server will start on port 8080 (or the port specified in the `PORT` environment variable).

**For production**, always set explicit allowed origins:
```bash
ALLOWED_ORIGINS="https://example.com,https://app.example.com" make run
```

#### Stdio Mode (for Claude Code / MCP clients)

```bash
# Run in stdio mode for MCP communication
# Note: stdio mode doesn't require CORS configuration
go run cmd/server/main.go --stdio
```

This mode communicates via stdin/stdout using JSON-RPC, which is required for Claude Code and other local MCP clients.

### Build Binary

```bash
make build

# Run with development CORS (local only)
ALLOW_CORS_WILDCARD_DEV=true ./bin/server

# Or with explicit origins (production)
ALLOWED_ORIGINS="https://example.com" ./bin/server
```

### Run with Docker

```bash
# Build image (creates both v1.0.0 and latest tags)
make docker

# Run with versioned tag (recommended)
docker run -p 8080:8080 -e ALLOW_CORS_WILDCARD_DEV=true timeservice:v1.0.0

# Or run with latest tag (local dev only)
docker run -p 8080:8080 -e ALLOW_CORS_WILDCARD_DEV=true timeservice:latest
```

**Production Note:** Always use versioned tags (`v1.0.0`) or image digests (`@sha256:...`) for production deployments to ensure deterministic, reproducible deployments. The `latest` tag is mutable and should only be used for local development.

### Run with Docker Compose (Hardened)

The project includes a hardened docker-compose.yml with security best practices:

```bash
docker-compose up
```

This configuration includes:
- Read-only root filesystem
- Dropped capabilities (ALL)
- No new privileges
- Resource limits
- Non-root user execution
- Tmpfs for writable directories

## API Endpoints

### 1. Root Endpoint

Get service information:

```bash
curl http://localhost:8080/
```

Response:
```json
{
  "service": "timeservice",
  "version": "1.0.0",
  "endpoints": {
    "time": "GET /api/time",
    "mcp": "POST /mcp",
    "health": "GET /health"
  },
  "mcp_info": "Supports both stdio mode (--stdio flag) and HTTP transport (POST /mcp)"
}
```

### 2. Time Endpoint

Get the current server time:

```bash
curl http://localhost:8080/api/time
```

Response:
```json
{
  "current_time": "2025-10-17T15:30:45.123456Z",
  "unix_time": 1729180245,
  "timezone": "UTC",
  "formatted": "2025-10-17T15:30:45Z"
}
```

### 3. Health Endpoint

Check service health:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "time": "2025-10-17T15:30:45Z"
}
```

## MCP Server Endpoint

The service includes a Model Context Protocol (MCP) server that exposes time-related tools for AI agents and other clients.

### List Available Tools

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/list"
  }'
```

Response:
```json
{
  "result": {
    "tools": [
      {
        "name": "get_current_time",
        "description": "Get the current server time in various formats",
        "inputSchema": {
          "type": "object",
          "properties": {
            "format": {
              "type": "string",
              "description": "Time format (iso8601, unix, rfc3339, or custom Go format)",
              "default": "iso8601"
            },
            "timezone": {
              "type": "string",
              "description": "IANA timezone (e.g., America/New_York, UTC)",
              "default": "UTC"
            }
          }
        }
      },
      {
        "name": "add_time_offset",
        "description": "Add a time offset to the current time",
        "inputSchema": {
          "type": "object",
          "properties": {
            "hours": {
              "type": "number",
              "description": "Hours to add (can be negative)",
              "default": 0
            },
            "minutes": {
              "type": "number",
              "description": "Minutes to add (can be negative)",
              "default": 0
            },
            "format": {
              "type": "string",
              "description": "Output format",
              "default": "iso8601"
            }
          }
        }
      }
    ]
  }
}
```

### Call a Tool: Get Current Time

Get the current time in ISO8601 format (UTC):

```bash
curl -X POST http://localhost:8080/mcp \
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
  }'
```

Get the current time in a specific timezone:

```bash
curl -X POST http://localhost:8080/mcp \
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
  }'
```

Get the current Unix timestamp:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "get_current_time",
      "arguments": {
        "format": "unix"
      }
    }
  }'
```

### Call a Tool: Add Time Offset

Add 3 hours to the current time:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "add_time_offset",
      "arguments": {
        "hours": 3,
        "minutes": 0,
        "format": "rfc3339"
      }
    }
  }'
```

Subtract 30 minutes from the current time:

```bash
curl -X POST http://localhost:8080/mcp \
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
  }'
```

## Configuration

The service can be configured through environment variables. All configuration is validated at startup, and the server will fail to start if invalid values are provided.

### Server Configuration

| Variable | Default | Description | Valid Values |
|----------|---------|-------------|--------------|
| `PORT` | `8080` | HTTP server port | `1-65535` |
| `HOST` | `` (all interfaces) | Bind address | Any valid IP or hostname |

### Logging Configuration

| Variable | Default | Description | Valid Values |
|----------|---------|-------------|--------------|
| `LOG_LEVEL` | `info` | Logging level | `debug`, `info`, `warn`, `warning`, `error` |

### CORS Configuration

**SECURITY-CRITICAL**: CORS configuration is required for the server to start.

| Variable | Default | Description | Valid Values |
|----------|---------|-------------|--------------|
| `ALLOWED_ORIGINS` | **REQUIRED** | Allowed CORS origins (comma-separated) | Explicit origins like `https://example.com,https://app.example.com` |
| `ALLOW_CORS_WILDCARD_DEV` | (none) | Dev-only escape hatch to allow wildcard CORS | `true` to enable `*` origin (DEVELOPMENT ONLY) |

**Security Notes**:
- **ALLOWED_ORIGINS is REQUIRED**: The server will fail to start if ALLOWED_ORIGINS is not set, preventing accidental wildcard CORS in production.
- **No wildcard default**: There is no default value. You must explicitly configure allowed origins.
- **Production**: Always use explicit origins (e.g., `ALLOWED_ORIGINS="https://example.com,https://app.example.com"`).
- **Development only**: Use `ALLOW_CORS_WILDCARD_DEV=true` to enable wildcard CORS (`*`) for local development. This is a conscious opt-in that prevents accidental production exposure.
- **Why this matters**: Wildcard CORS (`*`) allows any website to make authenticated requests to your API, potentially stealing cookies, session tokens, and user data. This is a critical security vulnerability.

**Example - Production (CORRECT):**
```bash
ALLOWED_ORIGINS="https://example.com,https://app.example.com" ./bin/server
```

**Example - Development (USE WITH CAUTION):**
```bash
ALLOW_CORS_WILDCARD_DEV=true ./bin/server
```

**What happens without configuration:**
```bash
$ ./bin/server
Configuration error: invalid configuration: ALLOWED_ORIGINS is required. Set explicit origins (e.g., ALLOWED_ORIGINS="https://example.com") or use ALLOW_CORS_WILDCARD_DEV=true for development ONLY. Wildcard CORS (*) is a security vulnerability in production
```

### Timeout Configuration

All timeout values use Go duration format (e.g., `10s`, `1m`, `500ms`).

| Variable | Default | Description | Valid Values |
|----------|---------|-------------|--------------|
| `READ_TIMEOUT` | `10s` | Maximum duration for reading request | Positive duration |
| `WRITE_TIMEOUT` | `10s` | Maximum duration for writing response | Positive duration |
| `IDLE_TIMEOUT` | `60s` | Maximum idle time between requests | Positive duration |
| `READ_HEADER_TIMEOUT` | `5s` | Maximum duration for reading request headers | Positive duration |
| `SHUTDOWN_TIMEOUT` | `10s` | Maximum duration for graceful shutdown | Positive duration |

### Resource Limits

| Variable | Default | Description | Valid Values |
|----------|---------|-------------|--------------|
| `MAX_HEADER_BYTES` | `1048576` (1MB) | Maximum size of request headers | `1-10485760` (1 byte - 10MB) |

### Configuration Examples

**Basic Production Configuration:**
```bash
PORT=8080 \
LOG_LEVEL=info \
ALLOWED_ORIGINS="https://example.com,https://app.example.com" \
READ_TIMEOUT=15s \
WRITE_TIMEOUT=15s \
make run
```

**Development Configuration with Debug Logging:**
```bash
PORT=3000 \
LOG_LEVEL=debug \
ALLOW_CORS_WILDCARD_DEV=true \
make run
```

**High-Performance Configuration:**
```bash
PORT=8080 \
ALLOWED_ORIGINS="https://api.example.com,https://app.example.com" \
READ_TIMEOUT=5s \
WRITE_TIMEOUT=5s \
IDLE_TIMEOUT=30s \
MAX_HEADER_BYTES=524288 \
make run
```

**Docker Compose Configuration:**
```yaml
environment:
  - PORT=8080
  - LOG_LEVEL=info
  - ALLOWED_ORIGINS=https://example.com,https://app.example.com
  - READ_TIMEOUT=15s
  - WRITE_TIMEOUT=15s
```

### Configuration Validation

The server validates all configuration on startup and will exit with an error message if any values are invalid:

```bash
# Missing ALLOWED_ORIGINS example
$ ./bin/server
Configuration error: invalid configuration: ALLOWED_ORIGINS is required. Set explicit origins (e.g., ALLOWED_ORIGINS="https://example.com") or use ALLOW_CORS_WILDCARD_DEV=true for development ONLY. Wildcard CORS (*) is a security vulnerability in production

# Invalid port example
$ PORT=999999 ALLOWED_ORIGINS="https://example.com" ./bin/server
Configuration error: invalid configuration: invalid PORT 999999: must be between 1 and 65535

# Invalid timeout example
$ READ_TIMEOUT=-5s ALLOWED_ORIGINS="https://example.com" ./bin/server
Configuration error: invalid configuration: READ_TIMEOUT must be positive, got -5s
```

Configuration values are logged at startup (at INFO level) for debugging deployment issues:

```json
{
  "time": "2025-10-18T09:22:14Z",
  "level": "INFO",
  "msg": "configuration loaded",
  "port": "8080",
  "log_level": "INFO",
  "allowed_origins": ["https://example.com","https://app.example.com"],
  "read_timeout": 10000000000,
  "write_timeout": 10000000000,
  "idle_timeout": 60000000000
}
```

If wildcard CORS is detected (via `ALLOW_CORS_WILDCARD_DEV=true`), a warning will be logged:

```json
{
  "time": "2025-10-18T09:22:14Z",
  "level": "WARN",
  "msg": "wildcard CORS (*) is enabled - this is INSECURE for production",
  "recommendation": "set explicit origins in ALLOWED_ORIGINS",
  "dev_only": "use ALLOW_CORS_WILDCARD_DEV=true only in development"
}
```

## Development

### Project Structure

```
timeservice/
├── cmd/                 # Command-line applications
│   ├── server/          # Main server application
│   │   └── main.go
│   └── healthcheck/     # Healthcheck utility
│       └── main.go
├── internal/            # Private application code
│   ├── handler/         # HTTP handlers
│   ├── mcpserver/       # MCP server implementation (using mcp-go SDK)
│   ├── middleware/      # HTTP middleware (CORS, logging, metrics, recovery)
│   └── testutil/        # Testing utilities
├── pkg/                 # Public packages
│   ├── config/          # Configuration management
│   ├── metrics/         # Prometheus metrics
│   ├── model/           # Data models
│   └── version/         # Version information
├── k8s/                 # Kubernetes deployment manifests
│   ├── deployment.yaml  # K8s deployment with ServiceMonitor
│   └── prometheus.yml   # Sample Prometheus configuration
├── docs/                # Documentation
│   └── TESTING.md       # Testing guide
├── bin/                 # Compiled binaries (gitignored)
│   ├── server           # Main server binary
│   └── healthcheck      # Healthcheck binary
├── run-mcp.sh           # Helper script to run in stdio mode
├── Makefile             # Build commands
├── Dockerfile           # Multi-stage container image
├── docker-compose.yml   # Docker Compose configuration
└── README.md
```

### Available Make Commands

```bash
make help     # Show available commands
make build    # Build binary
make run      # Run server
make test     # Run tests
make fmt      # Format code
make lint     # Lint code
make clean    # Remove build artifacts
make deps     # Download dependencies
make docker   # Build Docker image
```

### Pre-commit Hooks

The project includes pre-commit hooks to enforce code quality and prevent committing binaries or coverage files.

#### Traditional Git Hooks

A pre-commit hook is automatically installed in `.git/hooks/pre-commit` that prevents committing:
- Binaries (`.exe`, `.dll`, `.so`, `.dylib`)
- Build artifacts in `bin/` directory
- Test binaries (`.test`)
- Coverage files (`.out`, `.coverprofile`)

The hook runs automatically on every commit. If it detects forbidden files, it will:
1. Block the commit
2. Display which files matched forbidden patterns
3. Provide instructions on how to fix the issue

#### Modern Pre-commit Framework (Optional)

For teams using the [pre-commit framework](https://pre-commit.com/), a `.pre-commit-config.yaml` is provided with additional checks:

**Setup:**
```bash
# Install pre-commit (if not already installed)
pip install pre-commit

# Install the git hooks
pre-commit install

# Run hooks manually on all files
pre-commit run --all-files
```

**Included Checks:**
- File size limits (max 500KB)
- Merge conflict detection
- YAML syntax validation
- Go formatting (`go fmt`)
- Go vetting (`go vet`)
- Go imports organization
- Go mod tidy
- Go build verification
- Go test execution
- Binary and coverage file prevention

#### .gitignore

The `.gitignore` file prevents accidentally adding:
- Build artifacts (`bin/`, `*.exe`, etc.)
- Test binaries (`*.test`)
- Coverage files (`*.out`, `coverage.html`)
- IDE files (`.idea/`, `.vscode/`)
- Environment files (`.env*`)
- OS files (`.DS_Store`)
- Temporary files (`tmp/`, `*.tmp`)

All developers should ensure their local builds respect these ignore rules.

## Architecture

This service follows the idiomatic Go web service patterns:

- **Separation of Concerns**: Handler → Service → Store layers (simplified for this example)
- **Structured Logging**: Using `log/slog` for structured, JSON-formatted logs
- **Middleware Chain**: Composable middleware for cross-cutting concerns
- **Graceful Shutdown**: Proper cleanup on SIGINT/SIGTERM
- **Context Propagation**: Request context passed through all layers
- **Minimal Dependencies**: Relies primarily on Go standard library

## Using with Claude Desktop

To use this MCP server with Claude Desktop, add the following configuration to your Claude Desktop MCP settings file:

**macOS/Linux**: `~/.config/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "timeservice": {
      "command": "/full/path/to/time-server/bin/server",
      "args": ["--stdio"],
      "description": "Time service providing current time and time offset calculations"
    }
  }
}
```

Replace `/full/path/to/time-server` with the actual absolute path to this project directory.

After adding the configuration:
1. Build the server: `make build`
2. Restart Claude Desktop
3. The timeservice tools will be available to Claude

You can verify it's working by asking Claude: "What time is it in Tokyo right now?"

### Available MCP Tools

- `get_current_time` - Get current server time in various formats and timezones
  - Parameters: `format` (iso8601, unix, unixmilli, rfc3339), `timezone` (IANA timezone name)
- `add_time_offset` - Add hours/minutes offset to current time
  - Parameters: `hours` (number), `minutes` (number), `format` (output format)

## MCP Protocol

The Model Context Protocol (MCP) is a protocol that allows AI models to interact with tools and resources. This service implements an MCP server using the [mcp-go SDK](https://github.com/mark3labs/mcp-go) in two modes:

- **Stdio mode** (for Claude Desktop and local MCP clients): JSON-RPC over stdin/stdout
- **HTTP mode** (for remote access): JSON-RPC over HTTP POST using StreamableHTTPServer

### MCP Methods

- `tools/list`: List all available tools
- `tools/call`: Call a specific tool with arguments

### MCP Response Format

Successful response:
```json
{
  "result": { ... }
}
```

Error response:
```json
{
  "error": {
    "code": 400,
    "message": "error description"
  }
}
```

## Testing

Run the test suite:

```bash
make test
```

Run tests with race detector:

```bash
make test-race
```

Generate coverage report:

```bash
make test-coverage
```

Generate HTML coverage report:

```bash
make test-coverage-html
# Open coverage.html in your browser
```

## CI/CD Pipeline

This project includes a comprehensive CI/CD pipeline using GitHub Actions that runs on every push and pull request.

### GitHub Actions Workflow

The CI pipeline (`.github/workflows/ci.yml`) includes:

#### Test Job
- **Multi-version testing**: Tests against Go 1.22, 1.23, and 1.24
- **Code formatting**: Ensures code is formatted with `go fmt`
- **Static analysis**: Runs `go vet` to catch common mistakes
- **Unit tests**: Executes all tests with verbose output
- **Race detection**: Runs tests with the race detector enabled
- **Coverage reporting**: Generates and uploads coverage reports
- **Codecov integration**: Optional upload to Codecov for tracking coverage over time

#### Lint Job
- **golangci-lint**: Runs comprehensive linting with multiple linters enabled
- **Timeout**: 5-minute timeout for linting
- **Parallel execution**: Runs in parallel with tests

#### Build Job
- **Binary compilation**: Builds the server binary
- **Artifact upload**: Uploads binary as GitHub artifact (7-day retention)
- **Size reporting**: Reports binary size

#### Docker Job
- **Image build**: Builds Docker image using BuildKit
- **Cache optimization**: Uses GitHub Actions cache for faster builds
- **Image testing**: Validates the built image
- **Size reporting**: Reports final image size

#### Security Job
- **Gosec scanner**: Security-focused Go linter
- **Trivy scanner**: Vulnerability scanner for dependencies and code
- **SARIF upload**: Uploads security findings to GitHub Security tab

### Local CI Simulation

Run all CI checks locally before pushing:

```bash
make ci-local
```

This runs:
1. `make deps` - Download and verify dependencies
2. `make fmt` - Format code
3. `make vet` - Run go vet
4. `make lint` - Run golangci-lint
5. `make test-race` - Run tests with race detector
6. `make test-coverage` - Generate coverage report

### Linting Configuration

The project uses golangci-lint with a comprehensive configuration (`.golangci.yml`) that includes:

- **Error checking**: errcheck, gosec
- **Code quality**: gosimple, staticcheck, unused
- **Style**: gofmt, goimports, revive
- **Performance**: gocritic with performance checks
- **Security**: gosec with security-focused checks
- **Best practices**: bodyclose, nilerr, unconvert

Install golangci-lint:

```bash
# Linux/macOS
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or using Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Run linting locally:

```bash
make lint
```

### Coverage Artifacts

Coverage reports are automatically:
- Generated for each Go version tested
- Uploaded as GitHub Actions artifacts (30-day retention)
- Available for download from the Actions tab
- Optionally uploaded to Codecov for tracking trends

### CI Badge

Add the CI status badge to your README (update repository URL):

```markdown
[![CI](https://github.com/yourorg/timeservice/workflows/CI/badge.svg)](https://github.com/yourorg/timeservice/actions)
```

## Container Security Hardening

The Docker image has been hardened following security best practices:

### Dockerfile Security Features

1. **Pinned Base Images**
   - Uses specific patch versions: `golang:1.24.8-alpine3.21` and `alpine:3.21.0`
   - Ensures reproducible builds and prevents supply chain attacks

2. **Minimal Attack Surface**
   - Multi-stage build reduces final image size to ~16MB
   - Only includes necessary runtime dependencies (ca-certificates, tzdata)
   - No shell or unnecessary binaries in final image

3. **Non-Root User**
   - Creates dedicated user `appuser` (UID 10001) and group `appgroup` (GID 10001)
   - All processes run as non-root by default
   - Application files owned by non-root user

4. **Timezone Data**
   - Installed via `apk add --no-cache tzdata` in builder
   - Copied from `/usr/share/zoneinfo` instead of Go's embedded zoneinfo
   - Supports all IANA timezones without embedding in binary

5. **Build Security Flags**
   - `-trimpath`: Removes absolute paths from binary
   - `-w -s`: Strips debugging information
   - `-extldflags "-static"`: Creates static binary (no dynamic dependencies)
   - `go mod verify`: Ensures dependencies haven't been tampered with

6. **Health Check**
   - Built-in Docker HEALTHCHECK directive
   - Validates container is functioning correctly

### Runtime Security (Docker/Kubernetes)

The included `docker-compose.yml` and `k8s/deployment.yaml` demonstrate runtime hardening:

**Docker Compose Features:**
```yaml
# Read-only root filesystem
read_only: true

# Drop all capabilities
cap_drop: - ALL

# Prevent privilege escalation
security_opt:
  - no-new-privileges:true

# Resource limits
deploy:
  resources:
    limits:
      cpus: '0.5'
      memory: 128M
```

**Kubernetes Security Context:**
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 10001
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

### Container Image Scanning

The CI pipeline includes three scanning tools:

1. **Trivy** (Aqua Security)
   - Scans for OS and application vulnerabilities
   - Checks for misconfigurations
   - Results uploaded to GitHub Security tab

2. **Grype** (Anchore)
   - Multi-source vulnerability database
   - Catches CVEs across different sources
   - SARIF format for GitHub integration

3. **Docker Scout**
   - Official Docker vulnerability scanner
   - Integrated with Docker Hub CVE database
   - Provides remediation advice

All scan results are available in:
- GitHub Actions workflow logs
- GitHub Security → Code scanning alerts
- Downloadable SARIF artifacts

### Running with Full Security

**Docker Run:**
```bash
docker run -d \
  --name timeservice \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges:true \
  --tmpfs /tmp:noexec,nosuid,size=10M \
  -p 8080:8080 \
  -e ALLOW_CORS_WILDCARD_DEV=true \
  timeservice:v1.0.0
```

**Docker Compose:**
```bash
docker-compose up -d
```

**Kubernetes:**
```bash
# Build and tag image for production
docker build -t timeservice:v1.0.0 .

# Push to your container registry (update with your registry)
# docker tag timeservice:v1.0.0 your-registry.com/timeservice:v1.0.0
# docker push your-registry.com/timeservice:v1.0.0

# Deploy to Kubernetes
kubectl apply -f k8s/deployment.yaml
```

**Note:** The Kubernetes deployment uses `image: timeservice:v1.0.0` for deterministic deployments. Update `k8s/deployment.yaml` with your registry URL and credentials if deploying to a real cluster.

### Security Verification

Verify the container runs as non-root:
```bash
docker run --rm timeservice:v1.0.0 id
# Expected: uid=10001(appuser) gid=10001(appgroup)
```

Check image vulnerabilities:
```bash
docker scout cves timeservice:v1.0.0
# Or
trivy image timeservice:v1.0.0
```

## Prometheus Observability

This service exposes Prometheus metrics for comprehensive observability and monitoring.

### Metrics Endpoint

The `/metrics` endpoint exposes Prometheus-formatted metrics:

```bash
curl http://localhost:8080/metrics
```

### Available Metrics

#### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `timeservice_http_requests_total` | Counter | `method`, `path`, `status` | Total number of HTTP requests |
| `timeservice_http_request_duration_seconds` | Histogram | `method`, `path` | HTTP request duration in seconds |
| `timeservice_http_request_size_bytes` | Histogram | `method`, `path` | HTTP request size in bytes |
| `timeservice_http_response_size_bytes` | Histogram | `method`, `path` | HTTP response size in bytes |
| `timeservice_http_requests_in_flight` | Gauge | - | Number of HTTP requests currently being processed |

#### MCP Tool Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `timeservice_mcp_tool_calls_total` | Counter | `tool`, `status` | Total number of MCP tool calls |
| `timeservice_mcp_tool_call_duration_seconds` | Histogram | `tool` | MCP tool call duration in seconds |
| `timeservice_mcp_tool_calls_in_flight` | Gauge | - | Number of MCP tool calls currently being processed |

#### Application Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `timeservice_build_info` | Gauge | `version`, `go_version` | Build information (always 1) |

#### Standard Go Metrics

The service also exposes standard Go runtime metrics:
- `go_goroutines` - Number of goroutines
- `go_memstats_*` - Memory statistics
- `go_gc_*` - Garbage collection statistics
- `process_*` - Process statistics (CPU, memory, file descriptors)

### Prometheus Configuration

#### Docker Compose

The `docker-compose.yml` includes labels for Prometheus service discovery:

```yaml
labels:
  - "prometheus.scrape=true"
  - "prometheus.port=8080"
  - "prometheus.path=/metrics"
```

#### Kubernetes

The Kubernetes deployment includes pod annotations for automatic Prometheus scraping:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

A `ServiceMonitor` resource is also provided for Prometheus Operator:

```bash
kubectl apply -f k8s/deployment.yaml
```

#### Standalone Prometheus

Example Prometheus configuration (`k8s/prometheus.yml`):

```yaml
scrape_configs:
  - job_name: 'timeservice'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Grafana Dashboards

#### Example Queries

**Request Rate (requests/second):**
```promql
rate(timeservice_http_requests_total[5m])
```

**Request Duration (p95):**
```promql
histogram_quantile(0.95, rate(timeservice_http_request_duration_seconds_bucket[5m]))
```

**Error Rate:**
```promql
rate(timeservice_http_requests_total{status=~"5.."}[5m])
/ rate(timeservice_http_requests_total[5m])
```

**MCP Tool Success Rate:**
```promql
rate(timeservice_mcp_tool_calls_total{status="success"}[5m])
/ rate(timeservice_mcp_tool_calls_total[5m])
```

**In-Flight Requests:**
```promql
timeservice_http_requests_in_flight
```

#### Creating a Dashboard

1. Import the metrics endpoint into Grafana datasource
2. Create panels using the queries above
3. Set up alerts for:
   - High error rates (> 5%)
   - High latency (p95 > 1s)
   - Service down (no metrics scraped)

### Monitoring Best Practices

#### Alerts

Recommended alerts:

**High Error Rate:**
```yaml
- alert: HighErrorRate
  expr: |
    rate(timeservice_http_requests_total{status=~"5.."}[5m])
    / rate(timeservice_http_requests_total[5m]) > 0.05
  for: 5m
  annotations:
    summary: "High error rate detected"
    description: "Error rate is {{ $value | humanizePercentage }}"
```

**High Latency:**
```yaml
- alert: HighLatency
  expr: |
    histogram_quantile(0.95,
      rate(timeservice_http_request_duration_seconds_bucket[5m])
    ) > 1.0
  for: 5m
  annotations:
    summary: "High latency detected"
    description: "P95 latency is {{ $value }}s"
```

**Service Down:**
```yaml
- alert: ServiceDown
  expr: up{job="timeservice"} == 0
  for: 1m
  annotations:
    summary: "Timeservice is down"
    description: "Service has been down for more than 1 minute"
```

#### Recording Rules

Pre-compute common queries:

```yaml
groups:
  - name: timeservice
    interval: 30s
    rules:
      - record: timeservice:http_requests:rate5m
        expr: rate(timeservice_http_requests_total[5m])

      - record: timeservice:http_request_duration:p95
        expr: histogram_quantile(0.95, rate(timeservice_http_request_duration_seconds_bucket[5m]))

      - record: timeservice:http_error_rate:rate5m
        expr: |
          rate(timeservice_http_requests_total{status=~"5.."}[5m])
          / rate(timeservice_http_requests_total[5m])
```

### Testing Metrics

Generate test traffic:

```bash
# Start server
ALLOWED_ORIGINS="*" ./bin/server

# Generate requests
for i in {1..100}; do
  curl -s http://localhost:8080/health > /dev/null
  curl -s http://localhost:8080/api/time > /dev/null
done

# View metrics
curl http://localhost:8080/metrics | grep timeservice
```

### Metrics Architecture

The metrics implementation follows Prometheus best practices:

1. **Automatic Instrumentation**: HTTP middleware automatically tracks all requests
2. **Tool-Level Tracking**: MCP tool calls are wrapped with metrics collection
3. **Cardinality Control**: Labels are carefully chosen to prevent metric explosion
4. **Namespace**: All metrics use `timeservice` namespace to avoid conflicts
5. **Standard Buckets**: Histograms use Prometheus default buckets for broad coverage

## License

MIT License
