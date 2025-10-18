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

### Run Locally

#### HTTP Server Mode (for remote access)

```bash
# Download dependencies
make deps

# Run the server
make run
```

The server will start on port 8080 (or the port specified in the `PORT` environment variable).

#### Stdio Mode (for Claude Code / MCP clients)

```bash
# Run in stdio mode for MCP communication
go run cmd/server/main.go --stdio
```

This mode communicates via stdin/stdout using JSON-RPC, which is required for Claude Code and other local MCP clients.

### Build Binary

```bash
make build
./bin/server
```

### Run with Docker

```bash
make docker
docker run -p 8080:8080 timeservice:latest
```

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
  "mcp_methods": [
    "tools/list",
    "tools/call"
  ]
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

The service can be configured through environment variables:

- `PORT`: Server port (default: `8080`)

Example:
```bash
PORT=3000 make run
```

## Development

### Project Structure

```
timeservice/
├── cmd/server/          # Application entry point
│   └── main.go
├── internal/            # Private application code
│   ├── handler/         # HTTP handlers
│   ├── mcpserver/      # MCP server implementation (using mcp-go SDK)
│   └── middleware/     # HTTP middleware
├── pkg/                # Public packages
│   └── model/          # Data models
├── bin/                # Compiled binaries
├── mcp-config.json     # Example MCP configuration for Claude Desktop
├── run-mcp.sh          # Helper script to run in stdio mode
├── Makefile            # Build commands
├── Dockerfile          # Container image
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

## License

MIT License
