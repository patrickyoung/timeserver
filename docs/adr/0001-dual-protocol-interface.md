# 1. Dual Protocol Interface

Date: 2025-10-19

## Status

Accepted

## Context

The time service must serve two very different types of clients. Traditional HTTP consumers expect REST endpoints for time, health, and service discovery. At the same time, Model Context Protocol (MCP) clients expect a JSON-RPC interface that can operate over stdio or HTTP streaming. Supporting these transports with separate binaries would duplicate logic and drift over time.

## Decision

Run a single Go process that embeds both the REST router and the MCP tool server. The binary accepts a `--stdio` flag to switch between a stdio-only MCP mode and the HTTP mode that exposes REST, MCP over HTTP, and Prometheus metrics on the same listener (`cmd/server/main.go`). The MCP capability surface is implemented once in `internal/mcpserver`, and the HTTP handler delegates to the shared MCP server when the `/mcp` endpoint is invoked. This keeps REST responses (`internal/handler`) and MCP tool behavior consistent by sharing the same model and logging primitives.

## Consequences

- Operational simplicity: one artifact supports both deployment modes, and switching to stdio requires only a flag.
- Shared domain logic: enhancements to MCP tools or time formatting automatically benefit both REST and MCP transports.
- Cross-cutting concerns (logging, metrics, graceful shutdown) are centralized; failure in one transport impacts the single process.
- When we need to scale transports independently, we may need to refactor into separate services or binaries.

## Related ADRs

- ADR 0002 – Configuration Guardrails for CORS and Startup Safety
- ADR 0003 – Prometheus Instrumentation for HTTP and MCP Paths
- ADR 0004 – Hardened Container Packaging for Deployments
