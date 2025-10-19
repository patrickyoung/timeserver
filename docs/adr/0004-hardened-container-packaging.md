# 4. Hardened Container Packaging for Deployments

Date: 2025-10-19

## Status

Accepted

## Context

The service is deployed via Docker Compose and Kubernetes. We need small images, fast cold starts, and a secure runtime posture (non-root, read-only filesystem) to meet organizational policies. The binary must also expose a lightweight health probe so orchestrators can differentiate between transient failures and ready states. Supporting both stdio and HTTP modes from ADR 0001 means the image should ship a single server binary plus ancillary tooling without bloating the final artifact.

## Decision

Use a multi-stage Docker build (`Dockerfile`) that compiles static Linux/amd64 binaries for the server and a dedicated healthcheck probe. The final image is based on Alpine, adds only CA certificates and timezone data, and runs as an unprivileged user (`appuser`). The container filesystem is read-only, with tmpfs mounts declared in `docker-compose.yml` and Kubernetes manifests. Security options drop Linux capabilities, disable privilege escalation, and enforce RuntimeDefault seccomp profiles. Health checks rely on the tiny `/app/healthcheck` binary, enabling consistent readiness and liveness probes across Compose and Kubernetes. Deployment specs (`k8s/deployment.yaml`) pin image tags, set resource limits, and annotate pods for Prometheus scraping (per ADR 0003). Configuration via environment variables (ADR 0002) keeps the image immutable.

## Consequences

- Small, reproducible images (<10 MB) ship quickly and reduce attack surface.
- Non-root execution and read-only filesystems satisfy container hardening requirements but necessitate tmpfs mounts for writable paths.
- Any new runtime dependency must be added to the builder stage or provided via sidecar/volume mounts.
- Cross-compilation targets other than Linux/amd64 will require additional build stages or tooling.

## Related ADRs

- ADR 0001 – Dual Protocol Interface
- ADR 0002 – Configuration Guardrails for CORS and Startup Safety
- ADR 0003 – Prometheus Instrumentation for HTTP and MCP Paths
