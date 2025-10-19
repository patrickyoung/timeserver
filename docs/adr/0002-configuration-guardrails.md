# 2. Configuration Guardrails for CORS and Startup Safety

Date: 2025-10-19

## Status

Accepted

## Context

The HTTP entry points expose browser-facing JSON APIs. Allowing permissive CORS by default would be a security risk, yet developers need a fast way to run the service locally. We also rely on environment variables for runtime configuration because the binary must run in containers, Kubernetes, and stdio-driven developer workflows. Without explicit validation, misconfiguration could go unnoticed until runtime.

## Decision

Load configuration exclusively from environment variables (`pkg/config`). The loader enforces explicit `ALLOWED_ORIGINS` values and fails fast when they are omitted or malformed. A dedicated escape hatch (`ALLOW_CORS_WILDCARD_DEV=true`) allows wildcard CORS only for local development and requires an intentional opt-in. All other operational parameters (timeouts, log level, header limits) are parsed with sane defaults and validated before the server starts. Startup logging in `cmd/server/main.go` records the effective configuration to make deployments auditable (see ADR 0001 for how this feeds into dual transport modes).

## Consequences

- Production builds fail closed when CORS origins are missing, preventing accidental exposure.
- Local developers can still opt into wildcard CORS without editing code, but the log warns about the risk.
- Uniform environment-based configuration works across Docker, Kubernetes, and stdio execution.
- Future configuration sources (e.g., config files or flags) will need to wrap or extend the existing validation path to remain consistent.

## Related ADRs

- ADR 0001 – Dual Protocol Interface
- ADR 0003 – Prometheus Instrumentation for HTTP and MCP Paths
- ADR 0004 – Hardened Container Packaging for Deployments
