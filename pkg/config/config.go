package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all operational configuration
type Config struct {
	// Server configuration
	Port string
	Host string

	// Logging configuration
	LogLevel slog.Level

	// CORS configuration
	AllowedOrigins []string

	// Timeout configuration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration

	// Resource limits
	MaxHeaderBytes int

	// Auth configuration
	AuthEnabled            bool
	OIDCIssuerURL          string
	OIDCAudience           string
	OIDCSkipExpiryCheck    bool
	OIDCSkipClientIDCheck  bool
	OIDCSkipIssuerCheck    bool
	AuthPublicPaths        []string
	AuthRequiredRole       string
	AuthRequiredPermission string
	AuthRequiredScope      string

	// Database configuration
	DBPath         string
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBCacheSize    int // In KB (will be converted to negative pages for SQLite)
	DBWalMode      bool
}

// Load loads configuration from environment variables with validation
func Load() (*Config, error) {
	// Get ALLOWED_ORIGINS - NO DEFAULT for security
	// User must explicitly configure CORS policy
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")

	// Dev-only escape hatch: allow wildcard ONLY if explicitly enabled
	// This prevents accidental wildcard CORS in production
	if allowedOrigins == "" {
		if getEnv("ALLOW_CORS_WILDCARD_DEV", "") == "true" {
			allowedOrigins = "*"
		}
		// Otherwise leave empty - validation will fail
	}

	cfg := &Config{
		// Set defaults
		Port:              getEnv("PORT", "8080"),
		Host:              getEnv("HOST", ""),
		LogLevel:          parseLogLevel(getEnv("LOG_LEVEL", "info")),
		AllowedOrigins:    parseAllowedOrigins(allowedOrigins),
		ReadTimeout:       parseDuration(getEnv("READ_TIMEOUT", "10s"), 10*time.Second),
		WriteTimeout:      parseDuration(getEnv("WRITE_TIMEOUT", "10s"), 10*time.Second),
		IdleTimeout:       parseDuration(getEnv("IDLE_TIMEOUT", "60s"), 60*time.Second),
		ReadHeaderTimeout: parseDuration(getEnv("READ_HEADER_TIMEOUT", "5s"), 5*time.Second),
		ShutdownTimeout:   parseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"), 10*time.Second),
		MaxHeaderBytes:    parseInt(getEnv("MAX_HEADER_BYTES", "1048576"), 1<<20), // 1MB default

		// Auth configuration
		AuthEnabled:            parseBool(getEnv("AUTH_ENABLED", "false")),
		OIDCIssuerURL:          getEnv("OIDC_ISSUER_URL", ""),
		OIDCAudience:           getEnv("OIDC_AUDIENCE", ""),
		OIDCSkipExpiryCheck:    parseBool(getEnv("OIDC_SKIP_EXPIRY_CHECK", "false")),
		OIDCSkipClientIDCheck:  parseBool(getEnv("OIDC_SKIP_CLIENT_ID_CHECK", "false")),
		OIDCSkipIssuerCheck:    parseBool(getEnv("OIDC_SKIP_ISSUER_CHECK", "false")),
		AuthPublicPaths:        parseCommaSeparatedList(getEnv("AUTH_PUBLIC_PATHS", "/health,/,/metrics")),
		AuthRequiredRole:       getEnv("AUTH_REQUIRED_ROLE", ""),
		AuthRequiredPermission: getEnv("AUTH_REQUIRED_PERMISSION", ""),
		AuthRequiredScope:      getEnv("AUTH_REQUIRED_SCOPE", ""),

		// Database configuration (defaults from db.DefaultConfig())
		DBPath:         getEnv("DB_PATH", "data/timeservice.db"),
		DBMaxOpenConns: parseInt(getEnv("DB_MAX_OPEN_CONNS", "25"), 25),
		DBMaxIdleConns: parseInt(getEnv("DB_MAX_IDLE_CONNS", "5"), 5),
		DBCacheSize:    parseInt(getEnv("DB_CACHE_SIZE_KB", "64000"), 64000),
		DBWalMode:      parseBool(getEnv("DB_WAL_MODE", "true")),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate port
	if c.Port == "" {
		return fmt.Errorf("PORT cannot be empty")
	}
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return fmt.Errorf("invalid PORT '%s': must be a number", c.Port)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid PORT %d: must be between 1 and 65535", port)
	}

	// Validate timeouts
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("READ_TIMEOUT must be positive, got %v", c.ReadTimeout)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("WRITE_TIMEOUT must be positive, got %v", c.WriteTimeout)
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("IDLE_TIMEOUT must be positive, got %v", c.IdleTimeout)
	}
	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("READ_HEADER_TIMEOUT must be positive, got %v", c.ReadHeaderTimeout)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive, got %v", c.ShutdownTimeout)
	}

	// Validate max header bytes
	if c.MaxHeaderBytes <= 0 {
		return fmt.Errorf("MAX_HEADER_BYTES must be positive, got %d", c.MaxHeaderBytes)
	}
	if c.MaxHeaderBytes > 10<<20 { // 10MB max
		return fmt.Errorf("MAX_HEADER_BYTES too large: %d (max 10MB)", c.MaxHeaderBytes)
	}

	// Validate allowed origins
	if len(c.AllowedOrigins) == 0 {
		return fmt.Errorf("ALLOWED_ORIGINS is required. Set explicit origins (e.g., ALLOWED_ORIGINS=\"https://example.com\") or use ALLOW_CORS_WILDCARD_DEV=true for development ONLY. Wildcard CORS (*) is a security vulnerability in production")
	}

	// Warn if using wildcard (but allow it since user explicitly configured it)
	for _, origin := range c.AllowedOrigins {
		if origin == "*" {
			// This is logged as a warning by the caller - wildcard is dangerous
			break
		}
	}

	// Validate auth configuration if enabled
	if c.AuthEnabled {
		if c.OIDCIssuerURL == "" {
			return fmt.Errorf("OIDC_ISSUER_URL is required when AUTH_ENABLED=true")
		}
		if c.OIDCAudience == "" {
			return fmt.Errorf("OIDC_AUDIENCE is required when AUTH_ENABLED=true")
		}
		// Validate issuer URL format
		if !strings.HasPrefix(c.OIDCIssuerURL, "https://") && !strings.HasPrefix(c.OIDCIssuerURL, "http://") {
			return fmt.Errorf("OIDC_ISSUER_URL must be a valid HTTP(S) URL, got: %s", c.OIDCIssuerURL)
		}
		// Warn if using HTTP in production
		if strings.HasPrefix(c.OIDCIssuerURL, "http://") && getEnv("ALLOW_HTTP_OIDC_DEV", "") != "true" {
			return fmt.Errorf("OIDC_ISSUER_URL uses HTTP (insecure). Set ALLOW_HTTP_OIDC_DEV=true for development ONLY")
		}
	}

	// Validate database configuration
	if c.DBPath == "" {
		return fmt.Errorf("DB_PATH cannot be empty")
	}
	if c.DBMaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be positive, got %d", c.DBMaxOpenConns)
	}
	if c.DBMaxIdleConns <= 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be positive, got %d", c.DBMaxIdleConns)
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)", c.DBMaxIdleConns, c.DBMaxOpenConns)
	}
	if c.DBCacheSize <= 0 {
		return fmt.Errorf("DB_CACHE_SIZE_KB must be positive, got %d", c.DBCacheSize)
	}

	return nil
}

// String returns a string representation of the config (safe for logging)
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port:%s, Host:%s, LogLevel:%s, AllowedOrigins:%v, "+
		"ReadTimeout:%v, WriteTimeout:%v, IdleTimeout:%v, ReadHeaderTimeout:%v, "+
		"ShutdownTimeout:%v, MaxHeaderBytes:%d, DBPath:%s, DBMaxOpenConns:%d, "+
		"DBMaxIdleConns:%d, DBCacheSize:%dKB, DBWalMode:%v}",
		c.Port, c.Host, c.LogLevel, c.AllowedOrigins,
		c.ReadTimeout, c.WriteTimeout, c.IdleTimeout, c.ReadHeaderTimeout,
		c.ShutdownTimeout, c.MaxHeaderBytes, c.DBPath, c.DBMaxOpenConns,
		c.DBMaxIdleConns, c.DBCacheSize, c.DBWalMode)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		// Invalid level defaults to info
		return slog.LevelInfo
	}
}

func parseAllowedOrigins(origins string) []string {
	// Split by comma and trim whitespace
	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseDuration(value string, defaultDuration time.Duration) time.Duration {
	if value == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultDuration
	}
	return duration
}

func parseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

func parseBool(value string) bool {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return b
}

func parseCommaSeparatedList(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ParseLogLevelFromEnv parses log level from environment without full config loading
// This is useful for stdio mode which doesn't need CORS configuration
func ParseLogLevelFromEnv() slog.Level {
	return parseLogLevel(getEnv("LOG_LEVEL", "info"))
}
