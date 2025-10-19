# Time Service Security Design

This document defines the authentication and authorization strategy for the Time Service. It outlines how we secure our HTTP APIs and MCP server using industry-standard OAuth2/OIDC with JWT-based claims authorization, while maintaining our commitment to simple, secure-by-default design.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Security Objectives](#2-security-objectives)
3. [Authentication Strategy](#3-authentication-strategy)
4. [Authorization Model](#4-authorization-model)
5. [Technical Architecture](#5-technical-architecture)
6. [Provider Options](#6-provider-options)
7. [JWT Token Structure](#7-jwt-token-structure)
8. [Claims-Based Access Control](#8-claims-based-access-control)
9. [Implementation Approach](#9-implementation-approach)
10. [Configuration](#10-configuration)
11. [Security Considerations](#11-security-considerations)
12. [Testing Strategy](#12-testing-strategy)

---

## 1. Overview

Modern teams secure microservices using **OAuth2 + OpenID Connect (OIDC)** with **JWT tokens** for stateless, scalable authentication and authorization. This approach:

- Delegates authentication to external Identity and Access Management (IAM) providers
- Uses cryptographically signed JWT tokens that services can validate independently
- Enables fine-grained, claims-based authorization without database lookups
- Scales horizontally without shared session state
- Follows industry best practices and security standards

---

## 2. Security Objectives

1. **Zero Trust**: Authenticate and authorize every request to protected endpoints
2. **Defense in Depth**: Multiple layers of security validation
3. **Principle of Least Privilege**: Grant minimum necessary access based on claims
4. **Secure by Default**: Require explicit configuration; fail closed on errors
5. **Simple Design**: Leverage proven standards and libraries over custom crypto
6. **Operational Excellence**: Observable, auditable, and debuggable security events

---

## 3. Authentication Strategy

### 3.1 OAuth2 + OpenID Connect

We use **OAuth2** for authorization framework and **OpenID Connect (OIDC)** for authentication:

- **OAuth2**: Industry-standard protocol for delegated authorization (RFC 6749)
- **OIDC**: Identity layer on top of OAuth2, providing standardized user authentication
- **JWT**: JSON Web Tokens (RFC 7519) carry cryptographically signed claims about the authenticated user

### 3.2 Token Flow

```
┌─────────┐                                    ┌──────────────┐
│ Client  │                                    │ Auth Provider│
│ (e.g.,  │                                    │ (Auth0,Okta, │
│  App)   │                                    │  Keycloak)   │
└────┬────┘                                    └──────┬───────┘
     │                                                │
     │  1. Request login/token                        │
     ├───────────────────────────────────────────────>│
     │                                                │
     │  2. Authenticate user                          │
     │     (credentials, MFA, SSO, etc.)              │
     │                                                │
     │  3. Return JWT Access Token                    │
     │     (signed with provider's private key)       │
     │<───────────────────────────────────────────────┤
     │                                                │
┌────┴────┐                                    ┌──────┴───────┐
│ Client  │                                    │ Auth Provider│
└────┬────┘                                    └──────────────┘
     │
     │  4. Request to Time Service API
     │     Authorization: Bearer <JWT>
     │
     v
┌──────────────────┐
│  Time Service    │
│  ┌────────────┐  │
│  │ Middleware │  │  5. Extract JWT from Authorization header
│  └─────┬──────┘  │
│        │         │  6. Verify JWT signature using provider's public key
│        │         │     (fetched from JWKS endpoint)
│        │         │
│        │         │  7. Validate claims:
│        │         │     - Expiration (exp)
│        │         │     - Audience (aud)
│        │         │     - Issuer (iss)
│        │         │     - Not before (nbf)
│        │         │
│        │         │  8. Extract custom claims
│        │         │     (roles, permissions, scopes)
│        v         │
│  ┌────────────┐  │  9. Authorize based on claims
│  │  Handler   │  │     - Check required roles/permissions
│  └────────────┘  │     - Grant or deny access
└──────────────────┘
```

### 3.3 Why External Auth Providers?

Implementing authentication securely is **complex and error-prone**:

- Secure password storage (bcrypt, argon2)
- Protection against timing attacks, rainbow tables, credential stuffing
- Multi-factor authentication (MFA)
- Account recovery, password reset flows
- Compliance (GDPR, SOC2, HIPAA)
- Regular security audits and patches

**Delegating to specialized IAM providers** is the modern best practice:
- ✅ Battle-tested security implementations
- ✅ Automatic security updates and patches
- ✅ Built-in MFA, SSO, social login
- ✅ Compliance certifications
- ✅ Reduced attack surface for our service

---

## 4. Authorization Model

### 4.1 Claims-Based Authorization

Authorization decisions are based on **claims** embedded in the JWT token:

- **Claims**: Key-value assertions about the user (e.g., `{"roles": ["admin", "user"]}`)
- **Stateless**: All authorization data is in the token; no database lookups needed
- **Fine-Grained**: Can express complex permissions (roles, scopes, custom attributes)

### 4.2 Standard Claims

- `sub` (Subject): Unique user identifier
- `iss` (Issuer): Auth provider URL
- `aud` (Audience): Intended recipient (our service)
- `exp` (Expiration): Token expiration timestamp
- `iat` (Issued At): Token creation timestamp
- `nbf` (Not Before): Token not valid before this timestamp

### 4.3 Custom Claims

Application-specific claims for authorization:

- `roles`: Array of role names (e.g., `["admin", "time-reader"]`)
- `permissions`: Array of granular permissions (e.g., `["time:read", "time:offset"]`)
- `scope`: OAuth2 scopes (e.g., `"openid profile email time:read"`)

### 4.4 Access Control Rules

Each endpoint/tool can require specific claims:

| Endpoint/Tool        | Required Claim Example              |
|----------------------|-------------------------------------|
| GET /api/time        | `roles` contains `"time-reader"`    |
| POST /mcp            | `roles` contains `"mcp-user"`       |
| get_current_time     | `permissions` contains `"time:read"`|
| add_time_offset      | `permissions` contains `"time:offset"`|
| GET /metrics         | Public or `roles` contains `"ops"`  |
| GET /health          | Public (no auth)                    |

---

## 5. Technical Architecture

### 5.1 Components

```
┌─────────────────────────────────────────────────────────────┐
│                        Time Service                         │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Middleware Stack                        │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  1. Prometheus (metrics)                       │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  2. Logger (structured logging)                │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  3. Recover (panic recovery)                   │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  4. Auth (NEW - JWT validation & authz)        │  │  │
│  │  │     - Extract Bearer token from header         │  │  │
│  │  │     - Verify JWT signature (JWKS)              │  │  │
│  │  │     - Validate standard claims                 │  │  │
│  │  │     - Check required roles/permissions         │  │  │
│  │  │     - Add user context to request              │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  5. CORS (origin validation)                   │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              HTTP Handlers                           │  │
│  │  - GET /api/time       (protected)                   │  │
│  │  - POST /mcp           (protected)                   │  │
│  │  - GET /health         (public)                      │  │
│  │  - GET /metrics        (configurable)                │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              MCP Server (over HTTP & stdio)          │  │
│  │  Tools are authorized separately based on claims     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 Auth Middleware Flow

```go
// Pseudocode for auth middleware
func Auth(requiredClaims Claims) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract token from Authorization header
            token := extractBearerToken(r.Header.Get("Authorization"))
            if token == "" {
                http.Error(w, "Unauthorized", 401)
                return
            }

            // 2. Verify JWT signature and validate claims
            idToken, err := verifier.Verify(r.Context(), token)
            if err != nil {
                http.Error(w, "Invalid token", 401)
                return
            }

            // 3. Extract custom claims
            var claims CustomClaims
            if err := idToken.Claims(&claims); err != nil {
                http.Error(w, "Invalid claims", 401)
                return
            }

            // 4. Check authorization (roles, permissions)
            if !hasRequiredClaims(claims, requiredClaims) {
                http.Error(w, "Forbidden", 403)
                return
            }

            // 5. Add user context to request for handlers
            ctx := context.WithValue(r.Context(), userKey, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### 5.3 Library: coreos/go-oidc

We use **github.com/coreos/go-oidc/v3** for OIDC client implementation:

- ✅ **Most popular**: 1,141 importers, actively maintained (latest: Oct 2025)
- ✅ **Standard-compliant**: Official Go OIDC implementation
- ✅ **Auto JWKS refresh**: Automatically fetches and caches provider's public keys
- ✅ **Token verification**: Validates signatures, expiration, audience, issuer
- ✅ **Claims extraction**: Unmarshals JWT claims into Go structs

---

## 6. Provider Options

We support multiple OIDC-compliant providers. Teams choose based on their needs:

### 6.1 Comparison Matrix

| Provider  | Type          | Hosting        | Cost         | Best For                          |
|-----------|---------------|----------------|--------------|-----------------------------------|
| **Auth0** | SaaS          | Fully managed  | Paid (free tier) | Quick setup, managed service  |
| **Okta**  | SaaS          | Fully managed  | Paid (free tier) | Enterprise SSO, large orgs    |
| **Azure Entra ID** | SaaS  | Microsoft-managed | Free/Paid tiers | Microsoft ecosystem, enterprise identity |
| **Keycloak** | Open Source | Self-hosted    | Free (hosting costs) | Full control, cost-sensitive |
| **AWS Cognito** | SaaS     | AWS-managed    | Pay-per-use  | AWS-native apps               |
| **Google Identity** | SaaS | Google-managed | Free/Paid    | Google Workspace integration  |

### 6.2 Recommended Default: Keycloak

For **simple, open-source, self-hosted** solution aligned with our philosophy:

- ✅ **Free and open-source**: No licensing costs
- ✅ **Full control**: Self-hosted on-premises or cloud
- ✅ **Feature-rich**: Supports OIDC, SAML, social login, MFA
- ✅ **Highly customizable**: Extend with custom themes, plugins
- ✅ **Active community**: Well-maintained, extensive documentation

**For production SaaS convenience**, Auth0 or Okta are excellent choices.

### 6.3 Provider Configuration

The service is **provider-agnostic** via OIDC standard:

```bash
# Environment variables (provider-agnostic)
OIDC_ISSUER_URL="https://your-provider.example.com"
OIDC_AUDIENCE="timeservice"
OIDC_JWKS_URL="https://your-provider.example.com/.well-known/jwks.json"  # Auto-discovered
```

Examples for specific providers:

```bash
# Auth0
OIDC_ISSUER_URL="https://your-tenant.auth0.com/"
OIDC_AUDIENCE="https://timeservice.example.com"

# Okta
OIDC_ISSUER_URL="https://your-org.okta.com/oauth2/default"
OIDC_AUDIENCE="api://timeservice"

# Azure Entra ID (formerly Azure AD)
OIDC_ISSUER_URL="https://login.microsoftonline.com/{tenant-id}/v2.0"
OIDC_AUDIENCE="api://timeservice"  # Or your Application ID URI

# Keycloak
OIDC_ISSUER_URL="https://keycloak.example.com/realms/myrealm"
OIDC_AUDIENCE="timeservice"

# AWS Cognito
OIDC_ISSUER_URL="https://cognito-idp.{region}.amazonaws.com/{user-pool-id}"
OIDC_AUDIENCE="{client-id}"

# Google
OIDC_ISSUER_URL="https://accounts.google.com"
OIDC_AUDIENCE="your-client-id.apps.googleusercontent.com"
```

### 6.4 Azure Entra ID Setup Guide

Azure Entra ID (formerly Azure Active Directory) is Microsoft's cloud-based identity and access management service. It's ideal for organizations using Azure, Microsoft 365, or other Microsoft services.

#### Prerequisites
- Azure subscription (free tier available)
- Azure Entra ID tenant

#### Setup Steps

1. **Register Application in Azure Portal**
   - Navigate to Azure Portal → Azure Entra ID → App registrations
   - Click "New registration"
   - Name: `timeservice`
   - Supported account types: Choose based on your needs (single tenant, multi-tenant, etc.)
   - Redirect URI: Not needed for API authentication
   - Click "Register"

2. **Configure API Permissions (Optional)**
   - Go to "API permissions" in your app registration
   - Add permissions your app needs (e.g., `User.Read` for user profile)
   - Grant admin consent if required

3. **Expose an API**
   - Go to "Expose an API"
   - Click "Set" next to Application ID URI
   - Set to `api://timeservice` (or your preferred URI)
   - Click "Add a scope"
     - Scope name: `time.read`
     - Who can consent: Admins and users
     - Display name: "Read time data"
     - Description: "Allows reading time information"
   - Click "Add scope"

4. **Configure App Roles (for role-based access)**
   - Go to "App roles"
   - Click "Create app role"
     - Display name: `Time Reader`
     - Allowed member types: Users/Groups
     - Value: `time-reader` (this appears in the `roles` claim)
     - Description: "Can read time information"
   - Repeat for other roles (e.g., `mcp-user`, `admin`)

5. **Note Configuration Values**
   - Application (client) ID: Used as `OIDC_AUDIENCE` (or use Application ID URI)
   - Directory (tenant) ID: Used in `OIDC_ISSUER_URL`
   - Issuer URL format: `https://login.microsoftonline.com/{tenant-id}/v2.0`

#### Environment Configuration

```bash
# Azure Entra ID Configuration
AUTH_ENABLED=true
OIDC_ISSUER_URL="https://login.microsoftonline.com/12345678-1234-1234-1234-123456789abc/v2.0"
OIDC_AUDIENCE="api://timeservice"  # Your Application ID URI
AUTH_PUBLIC_PATHS="/health,/"
AUTH_REQUIRED_ROLE="time-reader"  # Matches app role value
ALLOWED_ORIGINS="https://app.example.com"
```

#### Azure-Specific Claims

Azure Entra ID includes these claims in JWT tokens:

**Standard claims:**
- `iss`: Issuer (e.g., `https://login.microsoftonline.com/{tenant}/v2.0`)
- `sub`: Subject (user object ID)
- `aud`: Audience (your application ID or API URI)
- `exp`: Expiration timestamp
- `iat`: Issued at timestamp
- `tid`: Tenant ID

**Custom claims:**
- `roles`: Array of app roles assigned to the user (e.g., `["time-reader", "admin"]`)
- `groups`: Array of Azure AD group IDs (if configured)
- `scp` or `scope`: Delegated permissions scopes
- `preferred_username`: User's email or UPN
- `name`: User's display name
- `oid`: Object ID (unique user identifier)

#### Example: Obtaining a Token (for testing)

Using Azure CLI:
```bash
# Login to Azure
az login

# Get access token for your API
az account get-access-token \
  --resource api://timeservice \
  --query accessToken -o tsv
```

Using curl (client credentials flow):
```bash
curl -X POST \
  https://login.microsoftonline.com/{tenant-id}/oauth2/v2.0/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id={client-id}" \
  -d "client_secret={client-secret}" \
  -d "scope=api://timeservice/.default" \
  -d "grant_type=client_credentials"
```

#### Assigning Roles to Users

1. Go to Azure Portal → Enterprise applications
2. Find your application (`timeservice`)
3. Go to "Users and groups"
4. Click "Add user/group"
5. Select user and assign role (e.g., `Time Reader`)
6. Tokens for this user will now include `"roles": ["time-reader"]`

#### Conditional Access (Optional)

Azure Entra ID supports Conditional Access policies for advanced security:
- Require MFA for API access
- Restrict access by location, device compliance, or risk level
- Block/allow specific IP ranges
- Enforce terms of use

Configure in: Azure Portal → Azure Entra ID → Security → Conditional Access

#### Testing

```bash
# Set your token
TOKEN="eyJ0eXAi..."

# Test authenticated request
curl http://localhost:8080/api/time \
  -H "Authorization: Bearer $TOKEN"

# Should return time data if token is valid and has required role
```

#### Common Azure Entra Issues

**Issue: Token validation fails**
- Ensure tenant ID in issuer URL is correct
- Verify `OIDC_AUDIENCE` matches your Application ID URI or client ID
- Check that app roles are assigned to the user

**Issue: Missing roles claim**
- Ensure app roles are defined in app registration
- Verify roles are assigned to users in Enterprise applications
- Check token by decoding at jwt.ms

**Issue: "Invalid issuer" error**
- Use `/v2.0` endpoint for modern apps
- Format: `https://login.microsoftonline.com/{tenant-id}/v2.0`
- For multi-tenant: Use `https://login.microsoftonline.com/common/v2.0`

---

## 7. JWT Token Structure

### 7.1 Example JWT Token

After decoding, a JWT has three parts: header, payload (claims), signature.

**Header:**
```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "key-id-123"
}
```

**Payload (Claims):**
```json
{
  "iss": "https://auth.example.com",
  "sub": "auth0|507f1f77bcf86cd799439011",
  "aud": "timeservice",
  "exp": 1735689600,
  "iat": 1735603200,
  "nbf": 1735603200,
  "scope": "openid profile email time:read",
  "roles": ["time-reader", "mcp-user"],
  "permissions": ["time:read", "time:offset"],
  "email": "user@example.com",
  "email_verified": true
}
```

**Signature:** (validates that token hasn't been tampered with)

### 7.2 Signature Verification

- Tokens are signed using **asymmetric cryptography** (RS256, ES256)
- Auth provider signs with **private key** (kept secret)
- Our service verifies with **public key** (fetched from JWKS endpoint)
- This allows **stateless verification** without calling the auth provider on every request

---

## 8. Claims-Based Access Control

### 8.1 Role-Based Access Control (RBAC)

Simple approach: check if user has required role.

```go
type AuthRequirements struct {
    RequireAuth bool
    RequiredRoles []string  // User must have at least one of these roles
}

// Example: GET /api/time requires "time-reader" role
authMiddleware := middleware.Auth(AuthRequirements{
    RequireAuth: true,
    RequiredRoles: []string{"time-reader", "admin"},
})
```

### 8.2 Permission-Based Access Control

More granular: check specific permissions.

```go
type AuthRequirements struct {
    RequireAuth bool
    RequiredPermissions []string  // User must have ALL of these permissions
}

// Example: MCP add_time_offset requires "time:offset" permission
authMiddleware := middleware.Auth(AuthRequirements{
    RequireAuth: true,
    RequiredPermissions: []string{"time:offset"},
})
```

### 8.3 Scope-Based Access Control

OAuth2 scopes for coarse-grained access.

```go
type AuthRequirements struct {
    RequireAuth bool
    RequiredScopes []string  // User must have at least one of these scopes
}

// Example: All time APIs require "time:read" scope
authMiddleware := middleware.Auth(AuthRequirements{
    RequireAuth: true,
    RequiredScopes: []string{"time:read"},
})
```

### 8.4 Combined Requirements

Mix and match for complex rules:

```go
authMiddleware := middleware.Auth(AuthRequirements{
    RequireAuth: true,
    RequiredRoles: []string{"admin"},
    RequiredScopes: []string{"time:admin"},
})
```

---

## 9. Implementation Approach

### 9.1 Phase 1: Core Auth Middleware

1. Add `pkg/auth` package with OIDC provider setup
2. Implement JWT verification using `coreos/go-oidc/v3`
3. Create `internal/middleware.Auth` middleware
4. Add claims extraction and validation logic

### 9.2 Phase 2: Protect HTTP Endpoints

1. Apply auth middleware to protected routes
2. Configure per-endpoint authorization requirements
3. Add user context to request for handler access
4. Update tests with mock OIDC provider

### 9.3 Phase 3: Protect MCP Server

1. Add auth validation layer for MCP requests
2. Check claims before executing tool handlers
3. Return appropriate errors for unauthorized access
4. Ensure stdio mode handles auth appropriately (or disables if local)

### 9.4 Phase 4: Configuration & Observability

1. Add auth config to `pkg/config` with validation
2. Add auth metrics to Prometheus (auth attempts, failures)
3. Add structured logging for auth events
4. Update ADR and docs

---

## 10. Configuration

### 10.1 Environment Variables

```bash
# OIDC Provider Configuration
OIDC_ENABLED=true                           # Enable/disable auth (default: false for backward compat)
OIDC_ISSUER_URL=https://auth.example.com    # OIDC issuer URL (required if enabled)
OIDC_AUDIENCE=timeservice                   # Expected audience claim (required if enabled)
OIDC_SKIP_EXPIRY_CHECK=false                # Skip token expiration check (DANGEROUS - dev only)
OIDC_SKIP_CLIENT_ID_CHECK=false             # Skip audience validation (DANGEROUS - dev only)

# Endpoint Protection Configuration
AUTH_PUBLIC_PATHS=/health,/                 # Comma-separated list of public paths (no auth)
AUTH_PROTECTED_PATHS=/api/*,/mcp            # Paths requiring authentication

# MCP-Specific Configuration
AUTH_MCP_ENABLED=true                       # Require auth for MCP over HTTP (default: true if OIDC enabled)
AUTH_MCP_STDIO_ENABLED=false                # Require auth for stdio mode (default: false - local trusted)

# Claims Requirements (examples - can be per-endpoint)
AUTH_REQUIRED_ROLE=time-reader              # Global required role
AUTH_REQUIRED_PERMISSION=                   # Global required permission
AUTH_REQUIRED_SCOPE=time:read               # Global required scope
```

### 10.2 Configuration Validation

Following ADR 0002 (Configuration Guardrails), we enforce:

- `OIDC_ISSUER_URL` is required if `OIDC_ENABLED=true`
- `OIDC_AUDIENCE` is required if `OIDC_ENABLED=true`
- Issuer URL must be valid HTTPS URL (warn if HTTP in dev)
- Fail-fast on startup if configuration is invalid
- Log warning if `OIDC_SKIP_*` flags are enabled (security risk)

### 10.3 Default Security Posture

**Secure by default:**

- Authentication is **opt-in** via `OIDC_ENABLED=true` (backward compatibility)
- Once enabled, **all endpoints are protected** except those in `AUTH_PUBLIC_PATHS`
- No wildcard defaults - explicit configuration required
- Stdio mode does NOT require auth by default (local use case)

---

## 11. Security Considerations

### 11.1 Token Lifetime and Rotation

- **Short-lived access tokens**: 15-60 minutes expiration
- **Long-lived refresh tokens**: Days/weeks for obtaining new access tokens
- Service validates expiration on every request
- Clients must implement token refresh flow

### 11.2 Token Revocation

JWT tokens are **stateless**, making immediate revocation difficult:

**Mitigation strategies:**
1. **Short expiration**: Limits exposure window
2. **Token versioning**: Include `jti` (JWT ID) claim, maintain blocklist for critical revocations
3. **Provider-side revocation**: Some providers offer introspection endpoints to check token validity

### 11.3 JWKS Caching and Rotation

- Provider's public keys are fetched from JWKS endpoint
- `coreos/go-oidc` automatically caches and refreshes keys
- Service gracefully handles key rotation (tokens signed with old or new keys)

### 11.4 Token Storage (Client-Side)

**Not our responsibility**, but best practices to recommend:

- Store access tokens in **memory only** (not localStorage)
- Use **httpOnly, secure cookies** for web apps if applicable
- Never log or expose tokens in client-side code

### 11.5 Rate Limiting

Add rate limiting middleware to prevent:
- Brute force token guessing
- DoS via auth endpoint flooding

### 11.6 Secure Token Transmission

- **Always use HTTPS** in production
- Tokens transmitted in `Authorization: Bearer <token>` header
- Never in URL query parameters (logged in access logs)

### 11.7 Principle of Least Privilege

- Grant users **minimum required permissions**
- Regularly audit user roles and permissions
- Implement time-limited elevated access if needed

---

## 12. Testing Strategy

### 12.1 Unit Tests

- Mock OIDC provider using test JWKS server
- Test JWT signature verification (valid, expired, wrong audience)
- Test claims extraction and validation
- Test authorization logic (roles, permissions, scopes)
- Test error handling (missing token, invalid token, insufficient permissions)

### 12.2 Integration Tests

- Test end-to-end flow with real OIDC provider (Keycloak in Docker)
- Test token refresh flows
- Test key rotation handling
- Test concurrent requests with different tokens

### 12.3 Security Tests

- Test expired token rejection
- Test wrong audience rejection
- Test tampered token signature rejection
- Test missing claims handling
- Test unauthorized access returns 401/403

### 12.4 Load Tests

- Verify JWKS caching performance
- Test concurrent auth under load
- Measure auth middleware latency impact

---

## Summary

This security design implements **OAuth2/OIDC with JWT-based claims authorization** using industry best practices:

✅ **Stateless, scalable** authentication and authorization
✅ **Provider-agnostic** (Auth0, Okta, Keycloak, etc.)
✅ **Simple implementation** using `coreos/go-oidc/v3`
✅ **Claims-based access control** for fine-grained permissions
✅ **Secure by default** with explicit configuration requirements
✅ **Observable** with metrics and structured logging
✅ **Testable** with comprehensive test coverage

This approach aligns with modern microservices security while honoring our commitment to **simple, secure-by-default design**.

---

## References

- [OAuth 2.0 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [JSON Web Token (JWT) RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519)
- [coreos/go-oidc GitHub](https://github.com/coreos/go-oidc)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
