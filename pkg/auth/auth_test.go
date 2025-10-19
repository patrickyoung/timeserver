package auth

import (
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantToken   string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:    false,
		},
		{
			name:       "valid bearer token with extra spaces",
			authHeader: "Bearer  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9  ",
			wantToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:    false,
		},
		{
			name:       "case insensitive bearer",
			authHeader: "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:    false,
		},
		{
			name:        "empty header",
			authHeader:  "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "missing bearer prefix",
			authHeader:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:     true,
			errContains: "format must be",
		},
		{
			name:        "wrong prefix",
			authHeader:  "Basic dXNlcjpwYXNz",
			wantErr:     true,
			errContains: "must start with 'Bearer'",
		},
		{
			name:        "bearer without token",
			authHeader:  "Bearer",
			wantErr:     true,
			errContains: "format must be",
		},
		{
			name:        "bearer with empty token",
			authHeader:  "Bearer  ",
			wantErr:     true,
			errContains: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, err := ExtractBearerToken(tt.authHeader)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractBearerToken() expected error but got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ExtractBearerToken() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractBearerToken() unexpected error = %v", err)
				return
			}

			if gotToken != tt.wantToken {
				t.Errorf("ExtractBearerToken() = %v, want %v", gotToken, tt.wantToken)
			}
		})
	}
}

func TestHasAnyRole(t *testing.T) {
	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		want          bool
	}{
		{
			name:          "user has required role",
			userRoles:     []string{"admin", "user"},
			requiredRoles: []string{"admin"},
			want:          true,
		},
		{
			name:          "user has one of multiple required roles",
			userRoles:     []string{"user", "editor"},
			requiredRoles: []string{"admin", "editor", "viewer"},
			want:          true,
		},
		{
			name:          "user missing all required roles",
			userRoles:     []string{"user"},
			requiredRoles: []string{"admin", "superuser"},
			want:          false,
		},
		{
			name:          "empty required roles (allow all)",
			userRoles:     []string{"user"},
			requiredRoles: []string{},
			want:          true,
		},
		{
			name:          "empty user roles",
			userRoles:     []string{},
			requiredRoles: []string{"admin"},
			want:          false,
		},
		{
			name:          "both empty",
			userRoles:     []string{},
			requiredRoles: []string{},
			want:          true,
		},
		{
			name:          "case sensitive check",
			userRoles:     []string{"Admin"},
			requiredRoles: []string{"admin"},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAnyRole(tt.userRoles, tt.requiredRoles); got != tt.want {
				t.Errorf("hasAnyRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasAllPermissions(t *testing.T) {
	tests := []struct {
		name                string
		userPermissions     []string
		requiredPermissions []string
		want                bool
	}{
		{
			name:                "user has all required permissions",
			userPermissions:     []string{"read", "write", "delete"},
			requiredPermissions: []string{"read", "write"},
			want:                true,
		},
		{
			name:                "user missing one permission",
			userPermissions:     []string{"read"},
			requiredPermissions: []string{"read", "write"},
			want:                false,
		},
		{
			name:                "user has exact permissions",
			userPermissions:     []string{"read", "write"},
			requiredPermissions: []string{"read", "write"},
			want:                true,
		},
		{
			name:                "empty required permissions (allow all)",
			userPermissions:     []string{"read"},
			requiredPermissions: []string{},
			want:                true,
		},
		{
			name:                "empty user permissions",
			userPermissions:     []string{},
			requiredPermissions: []string{"read"},
			want:                false,
		},
		{
			name:                "both empty",
			userPermissions:     []string{},
			requiredPermissions: []string{},
			want:                true,
		},
		{
			name:                "case sensitive check",
			userPermissions:     []string{"Read"},
			requiredPermissions: []string{"read"},
			want:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAllPermissions(tt.userPermissions, tt.requiredPermissions); got != tt.want {
				t.Errorf("hasAllPermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasAnyScope(t *testing.T) {
	tests := []struct {
		name           string
		userScopes     []string
		requiredScopes []string
		want           bool
	}{
		{
			name:           "user has required scope",
			userScopes:     []string{"openid", "profile", "email"},
			requiredScopes: []string{"email"},
			want:           true,
		},
		{
			name:           "user has one of multiple required scopes",
			userScopes:     []string{"openid", "profile"},
			requiredScopes: []string{"email", "profile", "address"},
			want:           true,
		},
		{
			name:           "user missing all required scopes",
			userScopes:     []string{"openid"},
			requiredScopes: []string{"email", "profile"},
			want:           false,
		},
		{
			name:           "empty required scopes (allow all)",
			userScopes:     []string{"openid"},
			requiredScopes: []string{},
			want:           true,
		},
		{
			name:           "empty user scopes",
			userScopes:     []string{},
			requiredScopes: []string{"openid"},
			want:           false,
		},
		{
			name:           "both empty",
			userScopes:     []string{},
			requiredScopes: []string{},
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAnyScope(tt.userScopes, tt.requiredScopes); got != tt.want {
				t.Errorf("hasAnyScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthorize(t *testing.T) {
	tests := []struct {
		name        string
		claims      *Claims
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid - user has required role",
			claims: &Claims{
				Roles: []string{"admin", "user"},
			},
			config: &Config{
				RequiredRoles: []string{"admin"},
			},
			wantErr: false,
		},
		{
			name: "valid - user has required permission",
			claims: &Claims{
				Permissions: []string{"time:read", "time:write"},
			},
			config: &Config{
				RequiredPermissions: []string{"time:read"},
			},
			wantErr: false,
		},
		{
			name: "valid - user has required scope",
			claims: &Claims{
				Scope: "openid profile email",
			},
			config: &Config{
				RequiredScopes: []string{"email"},
			},
			wantErr: false,
		},
		{
			name: "valid - all checks pass",
			claims: &Claims{
				Roles:       []string{"admin"},
				Permissions: []string{"time:read"},
				Scope:       "openid email",
			},
			config: &Config{
				RequiredRoles:       []string{"admin"},
				RequiredPermissions: []string{"time:read"},
				RequiredScopes:      []string{"email"},
			},
			wantErr: false,
		},
		{
			name: "invalid - missing required role",
			claims: &Claims{
				Roles: []string{"user"},
			},
			config: &Config{
				RequiredRoles: []string{"admin"},
			},
			wantErr:     true,
			errContains: "missing required role",
		},
		{
			name: "invalid - missing required permission",
			claims: &Claims{
				Permissions: []string{"time:read"},
			},
			config: &Config{
				RequiredPermissions: []string{"time:read", "time:write"},
			},
			wantErr:     true,
			errContains: "missing required permissions",
		},
		{
			name: "invalid - missing required scope",
			claims: &Claims{
				Scope: "openid profile",
			},
			config: &Config{
				RequiredScopes: []string{"email"},
			},
			wantErr:     true,
			errContains: "missing required scope",
		},
		{
			name: "valid - no requirements",
			claims: &Claims{
				Roles: []string{},
			},
			config: &Config{
				RequiredRoles:       []string{},
				RequiredPermissions: []string{},
				RequiredScopes:      []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal authenticator (we're only testing the Authorize method)
			auth := &Authenticator{
				config: tt.config,
				logger: nil, // Use nil logger for tests
			}

			err := auth.Authorize(tt.claims)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Authorize() expected error but got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Authorize() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Authorize() unexpected error = %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
