package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Authenticator handles OIDC authentication and JWT validation
type Authenticator struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   *Config
	logger   *slog.Logger
}

// Config holds auth configuration
type Config struct {
	// OIDC Provider configuration
	IssuerURL           string
	Audience            string
	SkipExpiryCheck     bool
	SkipClientIDCheck   bool
	SkipIssuerCheck     bool

	// Authorization requirements
	RequiredRoles       []string
	RequiredPermissions []string
	RequiredScopes      []string
}

// Claims represents the custom claims we extract from JWT tokens
type Claims struct {
	// Standard claims (from oidc.IDToken)
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	Expiry    int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`

	// Custom claims for authorization
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	Scope         string   `json:"scope"`

	// Additional custom claims (provider-specific)
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
}

// NewAuthenticator creates a new OIDC authenticator
func NewAuthenticator(ctx context.Context, cfg *Config, logger *slog.Logger) (*Authenticator, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("issuer URL is required")
	}

	// Create OIDC provider (auto-discovers JWKS endpoint and other metadata)
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Configure token verifier
	verifierConfig := &oidc.Config{
		ClientID:          cfg.Audience,
		SkipClientIDCheck: cfg.SkipClientIDCheck,
		SkipExpiryCheck:   cfg.SkipExpiryCheck,
		SkipIssuerCheck:   cfg.SkipIssuerCheck,
	}

	// Create ID token verifier
	verifier := provider.Verifier(verifierConfig)

	logger.Info("OIDC authenticator initialized",
		"issuer", cfg.IssuerURL,
		"audience", cfg.Audience,
		"skip_expiry_check", cfg.SkipExpiryCheck,
		"skip_client_id_check", cfg.SkipClientIDCheck,
	)

	// Warn if dangerous skip flags are enabled
	if cfg.SkipExpiryCheck || cfg.SkipClientIDCheck || cfg.SkipIssuerCheck {
		logger.Warn("SECURITY WARNING: Token validation checks are disabled",
			"skip_expiry_check", cfg.SkipExpiryCheck,
			"skip_client_id_check", cfg.SkipClientIDCheck,
			"skip_issuer_check", cfg.SkipIssuerCheck,
			"recommendation", "Only disable these in development environments",
		)
	}

	return &Authenticator{
		provider: provider,
		verifier: verifier,
		config:   cfg,
		logger:   logger,
	}, nil
}

// VerifyToken verifies a JWT token and extracts claims
func (a *Authenticator) VerifyToken(ctx context.Context, rawToken string) (*Claims, error) {
	// Verify the token signature and standard claims
	idToken, err := a.verifier.Verify(ctx, rawToken)
	if err != nil {
		a.logger.Debug("token verification failed", "error", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract custom claims
	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		a.logger.Debug("failed to extract claims", "error", err)
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	// Populate standard claims from IDToken
	claims.Subject = idToken.Subject
	claims.Issuer = idToken.Issuer
	claims.Expiry = idToken.Expiry.Unix()
	claims.IssuedAt = idToken.IssuedAt.Unix()

	a.logger.Debug("token verified successfully",
		"subject", claims.Subject,
		"email", claims.Email,
		"roles", claims.Roles,
	)

	return &claims, nil
}

// Authorize checks if the claims satisfy the authorization requirements
func (a *Authenticator) Authorize(claims *Claims) error {
	// Check required roles (user must have at least ONE of the required roles)
	if len(a.config.RequiredRoles) > 0 {
		if !hasAnyRole(claims.Roles, a.config.RequiredRoles) {
			if a.logger != nil {
				a.logger.Debug("authorization failed: missing required role",
					"user_roles", claims.Roles,
					"required_roles", a.config.RequiredRoles,
				)
			}
			return fmt.Errorf("missing required role: need one of %v", a.config.RequiredRoles)
		}
	}

	// Check required permissions (user must have ALL required permissions)
	if len(a.config.RequiredPermissions) > 0 {
		if !hasAllPermissions(claims.Permissions, a.config.RequiredPermissions) {
			if a.logger != nil {
				a.logger.Debug("authorization failed: missing required permission",
					"user_permissions", claims.Permissions,
					"required_permissions", a.config.RequiredPermissions,
				)
			}
			return fmt.Errorf("missing required permissions: need all of %v", a.config.RequiredPermissions)
		}
	}

	// Check required scopes (user must have at least ONE of the required scopes)
	if len(a.config.RequiredScopes) > 0 {
		userScopes := strings.Split(claims.Scope, " ")
		if !hasAnyScope(userScopes, a.config.RequiredScopes) {
			if a.logger != nil {
				a.logger.Debug("authorization failed: missing required scope",
					"user_scopes", userScopes,
					"required_scopes", a.config.RequiredScopes,
				)
			}
			return fmt.Errorf("missing required scope: need one of %v", a.config.RequiredScopes)
		}
	}

	if a.logger != nil {
		a.logger.Debug("authorization successful",
			"subject", claims.Subject,
			"email", claims.Email,
		)
	}

	return nil
}

// VerifyAndAuthorize is a convenience method that verifies the token and checks authorization in one step
func (a *Authenticator) VerifyAndAuthorize(ctx context.Context, rawToken string) (*Claims, error) {
	// Verify token and extract claims
	claims, err := a.VerifyToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if err := a.Authorize(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// OAuth2Config returns an OAuth2 config for this provider (for clients that need to obtain tokens)
func (a *Authenticator) OAuth2Config(clientID, clientSecret string, redirectURL string, scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     a.provider.Endpoint(),
		Scopes:       scopes,
	}
}

// Helper functions for authorization checks

// hasAnyRole checks if the user has at least one of the required roles
func hasAnyRole(userRoles, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true
	}
	roleSet := make(map[string]bool, len(userRoles))
	for _, role := range userRoles {
		roleSet[role] = true
	}
	for _, required := range requiredRoles {
		if roleSet[required] {
			return true
		}
	}
	return false
}

// hasAllPermissions checks if the user has all required permissions
func hasAllPermissions(userPermissions, requiredPermissions []string) bool {
	if len(requiredPermissions) == 0 {
		return true
	}
	permSet := make(map[string]bool, len(userPermissions))
	for _, perm := range userPermissions {
		permSet[perm] = true
	}
	for _, required := range requiredPermissions {
		if !permSet[required] {
			return false
		}
	}
	return true
}

// hasAnyScope checks if the user has at least one of the required scopes
func hasAnyScope(userScopes, requiredScopes []string) bool {
	if len(requiredScopes) == 0 {
		return true
	}
	scopeSet := make(map[string]bool, len(userScopes))
	for _, scope := range userScopes {
		scopeSet[scope] = true
	}
	for _, required := range requiredScopes {
		if scopeSet[required] {
			return true
		}
	}
	return false
}

// ExtractBearerToken extracts the token from an Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	// Expected format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("authorization header format must be 'Bearer {token}'")
	}

	if strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("authorization header must start with 'Bearer'")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("bearer token is empty")
	}

	return token, nil
}
