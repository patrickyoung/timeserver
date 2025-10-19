# 3. Prometheus Instrumentation for HTTP and MCP Paths

Date: 2025-10-19

## Status

Accepted

## Context

Operating the time service in containers and Kubernetes requires visibility into request volume, latency, and MCP tool usage. Relying on log scraping is fragile and does not expose structured metrics that Prometheus and Grafana expect. Instrumentation must cover both HTTP endpoints and the MCP tool handlers introduced in ADR 0001, without forcing each handler to manage counters manually.

## Decision

Adopt Prometheus metrics as the observability backbone. A dedicated `pkg/metrics` package encapsulates the metric families we need (counters, histograms, gauges) and initializes them under the `timeservice` namespace. HTTP traffic is captured via a first-class middleware (`internal/middleware.Prometheus`) that tracks in-flight requests, latency, and payload sizes. MCP tools are wrapped with `internal/mcpserver.wrapWithMetrics` to record per-tool counts and durations. The `/metrics` endpoint is exposed on the main HTTP mux, enabling Prometheus scraping in Docker Compose and Kubernetes manifests. Configuration guardrails from ADR 0002 ensure metrics exposure shares the same listener and security posture.

## Consequences

- Operators gain standardized metrics for dashboards and alerting without additional code in handlers.
- Metrics incur a small runtime overhead and require Prometheus-compatible scraping infrastructure.
- Future endpoints must use the shared router/middleware stack to inherit instrumentation.
- The single-process design ties metrics availability to application uptime; a dedicated sidecar would be needed for out-of-process scraping.

## Related ADRs

- ADR 0001 – Dual Protocol Interface
- ADR 0002 – Configuration Guardrails for CORS and Startup Safety
- ADR 0004 – Hardened Container Packaging for Deployments
