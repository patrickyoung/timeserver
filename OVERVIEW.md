# Time Service - Project Overview

## What You've Got

A production-ready Go web service that provides server time through both REST API and Model Context Protocol (MCP) endpoints.

## Key Features

✅ **REST API** - Simple `/api/time` endpoint returning current server time  
✅ **MCP Server** - Full Model Context Protocol implementation over HTTP  
✅ **Two MCP Tools**:
   - `get_current_time` - Get time in any format/timezone
   - `add_time_offset` - Add/subtract time from current time  
✅ **Production Ready** - Structured logging, graceful shutdown, middleware stack  
✅ **Tiny Docker Image** - Multi-stage build producing <10MB images  
✅ **Well Tested** - Comprehensive test suite included  

## Quick Start (3 Steps)

### 1. Run the Service

**Option A: Local Development**
```bash
cd timeservice
go run cmd/server/main.go
```

**Option B: With Docker**
```bash
cd timeservice
docker-compose up --build
```

### 2. Test the REST API

```bash
# Get current time
curl http://localhost:8080/api/time
```

### 3. Test the MCP Server

```bash
# List available tools
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"method": "tools/list"}'

# Get current time via MCP
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "get_current_time",
      "arguments": {"format": "unix"}
    }
  }'
```

## Project Structure

```
timeservice/
├── cmd/server/main.go       # Application entry point
├── internal/
│   ├── handler/             # HTTP handlers (time endpoint)
│   ├── mcp/                 # MCP server implementation
│   └── middleware/          # Logging, recovery, CORS
├── pkg/model/               # Data models (TimeResponse)
├── Dockerfile               # Multi-stage build
├── docker-compose.yml       # Easy deployment
├── Makefile                 # Build commands
├── test.sh                  # Automated test script
└── README.md               # Full documentation
```

## Available Endpoints

| Method | Path        | Description                    |
|--------|-------------|--------------------------------|
| GET    | `/`         | Service information           |
| GET    | `/health`   | Health check                  |
| GET    | `/api/time` | Get current server time       |
| POST   | `/mcp`      | MCP server (tools/list, tools/call) |

## MCP Tools

### 1. get_current_time

Get the current time in any format and timezone.

**Arguments:**
- `format` (string): `iso8601`, `unix`, `rfc3339`, or custom Go format
- `timezone` (string): IANA timezone like `America/New_York`, `UTC`, `Asia/Tokyo`

**Example:**
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

### 2. add_time_offset

Add or subtract time from the current moment.

**Arguments:**
- `hours` (number): Hours to add (negative to subtract)
- `minutes` (number): Minutes to add (negative to subtract)
- `format` (string): Output format

**Example:**
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "add_time_offset",
      "arguments": {
        "hours": 2.5,
        "minutes": 30,
        "format": "iso8601"
      }
    }
  }'
```

## Development Commands

```bash
make run      # Run the server
make build    # Build binary
make test     # Run tests
make docker   # Build Docker image
make fmt      # Format code
make lint     # Lint code
make clean    # Clean build artifacts
```

## Running Tests

### Unit Tests
```bash
make test
```

### Integration Tests (requires running server)
```bash
# Terminal 1: Start the server
make run

# Terminal 2: Run test script
./test.sh
```

## Configuration

Set via environment variables:

```bash
# Change port
PORT=3000 make run

# Using docker-compose
PORT=3000 docker-compose up
```

## What Makes This Special

This isn't just a toy example - it follows production Go patterns:

- **Idiomatic Go** - Uses stdlib routing (Go 1.22+), structured logging with slog
- **Clean Architecture** - Handler → Service → Store separation (simplified for this example)
- **Middleware Chain** - Composable middleware for logging, recovery, CORS
- **Graceful Shutdown** - Proper cleanup on SIGINT/SIGTERM
- **Context Propagation** - Request context flows through all layers
- **Comprehensive Tests** - Unit tests with table-driven patterns
- **Minimal Docker** - Scratch-based image under 10MB
- **Production Logging** - Structured JSON logs with all relevant context

## Next Steps

1. **Add Authentication** - Implement JWT auth for protected endpoints
2. **Add Database** - Store time records with SQLite/Turso
3. **Add More Tools** - Expand MCP with timezone conversion, scheduling, etc.
4. **Add Metrics** - Prometheus metrics endpoint
5. **Add Tracing** - OpenTelemetry integration

## Architecture Pattern Used

This service follows the **go-webservice** skill patterns:

```
Request → Middleware (logging/recovery) 
        → Router (stdlib http.ServeMux)
        → Handler (validates input, formats output)
        → Service (business logic) [future]
        → Store (data access) [future]
```

For this simple service, we collapsed some layers since there's no complex business logic or data storage needed.

## Learn More

- Full documentation: See `README.md`
- MCP Protocol: https://modelcontextprotocol.io/
- Go HTTP patterns: `internal/handler/handler.go`, `internal/middleware/middleware.go`
- Testing patterns: `internal/mcp/server_test.go`

---

**Ready to deploy?** Just run `docker-compose up -d` and you're live! 🚀
