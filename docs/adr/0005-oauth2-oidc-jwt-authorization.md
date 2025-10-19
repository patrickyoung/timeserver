# 5. OAuth2/OIDC JWT-Based Authentication and Authorization

Date: 2025-10-19

## Status

Accepted

## Context

The Time Service currently lacks authentication and authorization for its HTTP APIs and MCP server endpoints. Modern microservices require robust security to:

1. **Authenticate** requests to verify the caller's identity
2. **Authorize** requests based on the caller's permissions (roles, scopes, claims)
3. **Audit** access for compliance and security monitoring
4. **Scale horizontally** without shared session state

Implementing secure authentication is complex and error-prone, requiring defense against common attacks (credential stuffing, timing attacks, token tampering), secure password storage, MFA support, and ongoing security patches. Teams building internal services should delegate authentication to specialized Identity and Access Management (IAM) providers rather than implementing custom solutions.

The defacto standard for modern microservices security is **OAuth2 + OpenID Connect (OIDC)** with **JWT tokens** for stateless authentication and **claims-based authorization**. This approach is used by major cloud providers (AWS, GCP, Azure), SaaS platforms, and security-conscious organizations worldwide.

We need a security design that:
- Protects both HTTP APIs (`/api/time`) and MCP server endpoints (`/mcp`, MCP tools)
- Supports fine-grained access control (restrict specific tools/endpoints to users with certain claims)
- Works with external OIDC providers (Auth0, Okta, Keycloak, AWS Cognito, Google)
- Maintains stateless, horizontally scalable architecture
- Aligns with our "simple design" philosophy (leverage standards over custom crypto)
- Follows "secure by default" principles from ADR 0002

## Decision

Adopt **OAuth2/OIDC with JWT-based claims authorization** as the authentication and authorization mechanism for the Time Service.

### Architecture Components

1. **External OIDC Provider**: Handles user authentication, issues cryptographically signed JWT access tokens containing user identity and claims (roles, permissions, scopes)

2. **Auth Middleware** (`internal/middleware.Auth`):
   - Intercepts HTTP requests before they reach handlers
   - Extracts JWT from `Authorization: Bearer <token>` header
   - Verifies JWT signature using provider's public key (from JWKS endpoint)
   - Validates standard claims (expiration, audience, issuer)
   - Extracts custom claims (roles, permissions, scopes)
   - Checks authorization requirements for the endpoint
   - Returns 401 Unauthorized or 403 Forbidden on failure
   - Adds authenticated user context to request for handlers

3. **OIDC Client Library**: Use `github.com/coreos/go-oidc/v3` (latest: Oct 2025, 1,141 importers)
   - Standard-compliant OIDC implementation
   - Automatic JWKS fetching and caching
   - JWT signature verification
   - Claims extraction into Go structs

4. **Claims-Based Authorization**: Access control based on JWT claims
   - **Roles**: User role membership (e.g., `["admin", "time-reader"]`)
   - **Permissions**: Granular capabilities (e.g., `["time:read", "time:offset"]`)
   - **Scopes**: OAuth2 scopes (e.g., `"time:read time:write"`)
   - Each endpoint/tool specifies required claims; middleware enforces

5. **Configuration** (`pkg/config`):
   - `OIDC_ENABLED`: Opt-in flag (default: false for backward compatibility)
   - `OIDC_ISSUER_URL`: Auth provider URL (required if enabled)
   - `OIDC_AUDIENCE`: Expected audience claim (required if enabled)
   - `AUTH_PUBLIC_PATHS`: Comma-separated list of public endpoints (e.g., `/health,/`)
   - Validation enforces HTTPS issuer in production, fails fast on invalid config

6. **Provider-Agnostic Design**: Support any OIDC-compliant provider
   - Recommended default: **Keycloak** (open-source, self-hosted, free)
   - Production SaaS options: **Auth0**, **Okta** (fully managed, enterprise features)
   - Cloud-native: **AWS Cognito**, **Google Identity Platform**

### Security Model

- **Stateless**: All authorization data in JWT; no database lookups or shared session state
- **Asymmetric Crypto**: Tokens signed with provider's private key, verified with public key
- **Short-lived Tokens**: 15-60 minute expiration limits exposure window
- **Secure by Default**: All endpoints protected except explicit `AUTH_PUBLIC_PATHS`
- **Defense in Depth**: Multiple validation layers (signature, expiration, audience, claims)

### Middleware Integration

Auth middleware inserts into existing middleware chain (from `cmd/server/main.go`):

```go
handler := middleware.Chain(
    mux,
    middleware.Prometheus(metricsCollector),  // First: capture all requests
    middleware.Logger(logger),                 // Log before auth
    middleware.Recover(logger),                // Catch panics before auth
    middleware.Auth(authConfig),               // NEW: Auth before business logic
    middleware.CORSWithOrigins(cfg.AllowedOrigins),
)
```

### Endpoint Protection Examples

```go
// Public endpoints (no auth required)
AUTH_PUBLIC_PATHS=/health,/,/metrics

// Protected endpoints with role requirements
// GET /api/time - requires "time-reader" role
// POST /mcp - requires "mcp-user" role
// MCP get_current_time tool - requires "time:read" permission
// MCP add_time_offset tool - requires "time:offset" permission
```

### MCP Server Security

- **HTTP MCP (`POST /mcp`)**: Protected by auth middleware like other HTTP endpoints
- **Stdio MCP (`--stdio`)**: Auth disabled by default (local trusted use case); optionally enable via `AUTH_MCP_STDIO_ENABLED=true`
- **Per-Tool Authorization**: MCP request handler validates claims before delegating to tool functions

## Consequences

### Positive

- **Industry Standard**: OAuth2/OIDC is battle-tested, widely adopted, with extensive tooling and documentation
- **Delegated Authentication**: Offload complex auth logic to specialized IAM providers with automatic security updates
- **Stateless Scalability**: No shared state; services scale horizontally without session affinity
- **Fine-Grained Control**: Claims-based authz supports complex permission models (RBAC, ABAC, scopes)
- **Provider Flexibility**: Swap providers (Auth0 → Keycloak) without code changes
- **Simple Implementation**: `coreos/go-oidc` handles complexity; our middleware is ~200 lines
- **Observable**: Auth events logged (successes, failures); Prometheus metrics for auth attempts, latencies, errors
- **Testable**: Mock OIDC providers in tests; comprehensive coverage for auth flows
- **Backward Compatible**: Auth is opt-in (`OIDC_ENABLED=false` by default)

### Negative

- **External Dependency**: Requires auth provider availability (mitigated by short-lived token caching)
- **JWT Size**: Large claims increase header size (typically <2KB, within limits)
- **Revocation Latency**: Stateless tokens can't be instantly revoked (mitigated by short expiration)
- **Configuration Complexity**: Teams must set up and configure OIDC provider
- **Clock Skew Sensitivity**: JWT expiration depends on clock sync (NTP recommended)

### Operational Impact

- **Setup Required**: Teams must deploy/configure OIDC provider before enabling auth
- **Token Management**: Clients must implement OAuth2 flows (client credentials, authorization code)
- **Monitoring**: New metrics for auth failures, token validation errors, latency
- **Incident Response**: Auth failures logged for security auditing

### Migration Path

1. **Phase 1**: Implement auth middleware with `OIDC_ENABLED=false` (default - no impact)
2. **Phase 2**: Teams opt-in by configuring provider and setting `OIDC_ENABLED=true`
3. **Phase 3**: Future: make auth required (remove opt-out) for production deployments

## Related ADRs

- ADR 0001 – Dual Protocol Interface (MCP over HTTP and stdio)
- ADR 0002 – Configuration Guardrails for CORS and Startup Safety (similar validation approach)
- ADR 0003 – Prometheus Instrumentation (auth metrics added to existing collectors)

## References

- [OAuth 2.0 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [JSON Web Token (JWT) RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519)
- [coreos/go-oidc v3 Documentation](https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc)
- [Security Design Document](../SECURITY.md) - Comprehensive implementation guide
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
