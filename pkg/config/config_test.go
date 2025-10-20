package config

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save current environment
	oldEnv := map[string]string{
		"PORT":                        os.Getenv("PORT"),
		"LOG_LEVEL":                   os.Getenv("LOG_LEVEL"),
		"ALLOWED_ORIGINS":             os.Getenv("ALLOWED_ORIGINS"),
		"ALLOW_CORS_WILDCARD_DEV":     os.Getenv("ALLOW_CORS_WILDCARD_DEV"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Test with defaults and dev escape hatch
	os.Unsetenv("PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("ALLOWED_ORIGINS")
	os.Setenv("ALLOW_CORS_WILDCARD_DEV", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with defaults and dev escape hatch failed: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}

	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("expected default log level Info, got %v", cfg.LogLevel)
	}

	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected wildcard allowed origins [*] with dev escape hatch, got %v", cfg.AllowedOrigins)
	}

	if cfg.ReadTimeout != 10*time.Second {
		t.Errorf("expected default read timeout 10s, got %v", cfg.ReadTimeout)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Save current environment
	oldEnv := map[string]string{
		"PORT":                os.Getenv("PORT"),
		"LOG_LEVEL":           os.Getenv("LOG_LEVEL"),
		"ALLOWED_ORIGINS":     os.Getenv("ALLOWED_ORIGINS"),
		"READ_TIMEOUT":        os.Getenv("READ_TIMEOUT"),
		"WRITE_TIMEOUT":       os.Getenv("WRITE_TIMEOUT"),
		"IDLE_TIMEOUT":        os.Getenv("IDLE_TIMEOUT"),
		"READ_HEADER_TIMEOUT": os.Getenv("READ_HEADER_TIMEOUT"),
		"SHUTDOWN_TIMEOUT":    os.Getenv("SHUTDOWN_TIMEOUT"),
		"MAX_HEADER_BYTES":    os.Getenv("MAX_HEADER_BYTES"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set custom values
	os.Setenv("PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("ALLOWED_ORIGINS", "https://example.com,https://app.example.com")
	os.Setenv("READ_TIMEOUT", "15s")
	os.Setenv("WRITE_TIMEOUT", "20s")
	os.Setenv("IDLE_TIMEOUT", "120s")
	os.Setenv("READ_HEADER_TIMEOUT", "3s")
	os.Setenv("SHUTDOWN_TIMEOUT", "30s")
	os.Setenv("MAX_HEADER_BYTES", "2097152") // 2MB

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with custom values failed: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}

	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("expected log level Debug, got %v", cfg.LogLevel)
	}

	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
	}

	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected read timeout 15s, got %v", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout != 20*time.Second {
		t.Errorf("expected write timeout 20s, got %v", cfg.WriteTimeout)
	}

	if cfg.IdleTimeout != 120*time.Second {
		t.Errorf("expected idle timeout 120s, got %v", cfg.IdleTimeout)
	}

	if cfg.ReadHeaderTimeout != 3*time.Second {
		t.Errorf("expected read header timeout 3s, got %v", cfg.ReadHeaderTimeout)
	}

	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("expected shutdown timeout 30s, got %v", cfg.ShutdownTimeout)
	}

	if cfg.MaxHeaderBytes != 2097152 {
		t.Errorf("expected max header bytes 2097152, got %d", cfg.MaxHeaderBytes)
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{
			name: "empty port",
			port: "",
			want: "PORT cannot be empty",
		},
		{
			name: "non-numeric port",
			port: "abc",
			want: "invalid PORT 'abc': must be a number",
		},
		{
			name: "port too low",
			port: "0",
			want: "invalid PORT 0: must be between 1 and 65535",
		},
		{
			name: "port too high",
			port: "65536",
			want: "invalid PORT 65536: must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:              tt.port,
				LogLevel:          slog.LevelInfo,
				AllowedOrigins:    []string{"*"},
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       60 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				MaxHeaderBytes:    1 << 20,
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error, got nil")
			} else if !contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestValidate_InvalidTimeouts(t *testing.T) {
	tests := []struct {
		name     string
		modifier func(*Config)
		want     string
	}{
		{
			name: "negative read timeout",
			modifier: func(c *Config) {
				c.ReadTimeout = -1 * time.Second
			},
			want: "READ_TIMEOUT must be positive",
		},
		{
			name: "zero write timeout",
			modifier: func(c *Config) {
				c.WriteTimeout = 0
			},
			want: "WRITE_TIMEOUT must be positive",
		},
		{
			name: "negative idle timeout",
			modifier: func(c *Config) {
				c.IdleTimeout = -5 * time.Second
			},
			want: "IDLE_TIMEOUT must be positive",
		},
		{
			name: "zero read header timeout",
			modifier: func(c *Config) {
				c.ReadHeaderTimeout = 0
			},
			want: "READ_HEADER_TIMEOUT must be positive",
		},
		{
			name: "negative shutdown timeout",
			modifier: func(c *Config) {
				c.ShutdownTimeout = -10 * time.Second
			},
			want: "SHUTDOWN_TIMEOUT must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:              "8080",
				LogLevel:          slog.LevelInfo,
				AllowedOrigins:    []string{"*"},
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       60 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				MaxHeaderBytes:    1 << 20,
			}

			tt.modifier(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error, got nil")
			} else if !contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestValidate_InvalidMaxHeaderBytes(t *testing.T) {
	tests := []struct {
		name           string
		maxHeaderBytes int
		want           string
	}{
		{
			name:           "zero max header bytes",
			maxHeaderBytes: 0,
			want:           "MAX_HEADER_BYTES must be positive",
		},
		{
			name:           "negative max header bytes",
			maxHeaderBytes: -1,
			want:           "MAX_HEADER_BYTES must be positive",
		},
		{
			name:           "max header bytes too large",
			maxHeaderBytes: 11 << 20, // 11MB
			want:           "MAX_HEADER_BYTES too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:              "8080",
				LogLevel:          slog.LevelInfo,
				AllowedOrigins:    []string{"*"},
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       60 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				MaxHeaderBytes:    tt.maxHeaderBytes,
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error, got nil")
			} else if !contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestLoad_MissingAllowedOrigins(t *testing.T) {
	// Save current environment
	oldEnv := map[string]string{
		"ALLOWED_ORIGINS":         os.Getenv("ALLOWED_ORIGINS"),
		"ALLOW_CORS_WILDCARD_DEV": os.Getenv("ALLOW_CORS_WILDCARD_DEV"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Unset both ALLOWED_ORIGINS and the dev escape hatch
	os.Unsetenv("ALLOWED_ORIGINS")
	os.Unsetenv("ALLOW_CORS_WILDCARD_DEV")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when ALLOWED_ORIGINS is not set and dev escape hatch is disabled")
	}

	expectedMsg := "ALLOWED_ORIGINS is required"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestValidate_MissingAllowedOrigins(t *testing.T) {
	cfg := &Config{
		Port:              "8080",
		LogLevel:          slog.LevelInfo,
		AllowedOrigins:    []string{}, // Empty - should fail validation
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should fail when AllowedOrigins is empty")
	}

	expectedMsg := "ALLOWED_ORIGINS is required"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo}, // defaults to info
		{"", slog.LevelInfo},         // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseAllowedOrigins(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"*", []string{"*"}},
		{"https://example.com", []string{"https://example.com"}},
		{"https://example.com,https://app.example.com", []string{"https://example.com", "https://app.example.com"}},
		{"https://example.com, https://app.example.com", []string{"https://example.com", "https://app.example.com"}}, // with spaces
		{"  https://example.com  ,  https://app.example.com  ", []string{"https://example.com", "https://app.example.com"}}, // extra spaces
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseAllowedOrigins(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseAllowedOrigins(%q) length = %d, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseAllowedOrigins(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

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

func TestLoad_DatabaseDefaults(t *testing.T) {
	// Save current environment
	oldEnv := map[string]string{
		"ALLOWED_ORIGINS":         os.Getenv("ALLOWED_ORIGINS"),
		"ALLOW_CORS_WILDCARD_DEV": os.Getenv("ALLOW_CORS_WILDCARD_DEV"),
		"DB_PATH":                 os.Getenv("DB_PATH"),
		"DB_MAX_OPEN_CONNS":       os.Getenv("DB_MAX_OPEN_CONNS"),
		"DB_MAX_IDLE_CONNS":       os.Getenv("DB_MAX_IDLE_CONNS"),
		"DB_CACHE_SIZE_KB":        os.Getenv("DB_CACHE_SIZE_KB"),
		"DB_WAL_MODE":             os.Getenv("DB_WAL_MODE"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set minimal required config
	os.Setenv("ALLOW_CORS_WILDCARD_DEV", "true")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("DB_MAX_OPEN_CONNS")
	os.Unsetenv("DB_MAX_IDLE_CONNS")
	os.Unsetenv("DB_CACHE_SIZE_KB")
	os.Unsetenv("DB_WAL_MODE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with database defaults failed: %v", err)
	}

	if cfg.DBPath != "data/timeservice.db" {
		t.Errorf("expected default DB_PATH data/timeservice.db, got %s", cfg.DBPath)
	}

	if cfg.DBMaxOpenConns != 25 {
		t.Errorf("expected default DB_MAX_OPEN_CONNS 25, got %d", cfg.DBMaxOpenConns)
	}

	if cfg.DBMaxIdleConns != 5 {
		t.Errorf("expected default DB_MAX_IDLE_CONNS 5, got %d", cfg.DBMaxIdleConns)
	}

	if cfg.DBCacheSize != 64000 {
		t.Errorf("expected default DB_CACHE_SIZE_KB 64000, got %d", cfg.DBCacheSize)
	}

	if !cfg.DBWalMode {
		t.Errorf("expected default DB_WAL_MODE true, got false")
	}
}

func TestLoad_DatabaseCustomValues(t *testing.T) {
	// Save current environment
	oldEnv := map[string]string{
		"ALLOWED_ORIGINS":         os.Getenv("ALLOWED_ORIGINS"),
		"ALLOW_CORS_WILDCARD_DEV": os.Getenv("ALLOW_CORS_WILDCARD_DEV"),
		"DB_PATH":                 os.Getenv("DB_PATH"),
		"DB_MAX_OPEN_CONNS":       os.Getenv("DB_MAX_OPEN_CONNS"),
		"DB_MAX_IDLE_CONNS":       os.Getenv("DB_MAX_IDLE_CONNS"),
		"DB_CACHE_SIZE_KB":        os.Getenv("DB_CACHE_SIZE_KB"),
		"DB_WAL_MODE":             os.Getenv("DB_WAL_MODE"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set custom database config
	os.Setenv("ALLOW_CORS_WILDCARD_DEV", "true")
	os.Setenv("DB_PATH", "/custom/path/db.sqlite")
	os.Setenv("DB_MAX_OPEN_CONNS", "50")
	os.Setenv("DB_MAX_IDLE_CONNS", "10")
	os.Setenv("DB_CACHE_SIZE_KB", "128000")
	os.Setenv("DB_WAL_MODE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with custom database config failed: %v", err)
	}

	if cfg.DBPath != "/custom/path/db.sqlite" {
		t.Errorf("expected DB_PATH /custom/path/db.sqlite, got %s", cfg.DBPath)
	}

	if cfg.DBMaxOpenConns != 50 {
		t.Errorf("expected DB_MAX_OPEN_CONNS 50, got %d", cfg.DBMaxOpenConns)
	}

	if cfg.DBMaxIdleConns != 10 {
		t.Errorf("expected DB_MAX_IDLE_CONNS 10, got %d", cfg.DBMaxIdleConns)
	}

	if cfg.DBCacheSize != 128000 {
		t.Errorf("expected DB_CACHE_SIZE_KB 128000, got %d", cfg.DBCacheSize)
	}

	if cfg.DBWalMode {
		t.Errorf("expected DB_WAL_MODE false, got true")
	}
}

func TestValidate_InvalidDatabaseConfig(t *testing.T) {
	tests := []struct {
		name     string
		modifier func(*Config)
		want     string
	}{
		{
			name: "empty DB_PATH",
			modifier: func(c *Config) {
				c.DBPath = ""
			},
			want: "DB_PATH cannot be empty",
		},
		{
			name: "zero DB_MAX_OPEN_CONNS",
			modifier: func(c *Config) {
				c.DBMaxOpenConns = 0
			},
			want: "DB_MAX_OPEN_CONNS must be positive",
		},
		{
			name: "negative DB_MAX_OPEN_CONNS",
			modifier: func(c *Config) {
				c.DBMaxOpenConns = -1
			},
			want: "DB_MAX_OPEN_CONNS must be positive",
		},
		{
			name: "zero DB_MAX_IDLE_CONNS",
			modifier: func(c *Config) {
				c.DBMaxIdleConns = 0
			},
			want: "DB_MAX_IDLE_CONNS must be positive",
		},
		{
			name: "negative DB_MAX_IDLE_CONNS",
			modifier: func(c *Config) {
				c.DBMaxIdleConns = -1
			},
			want: "DB_MAX_IDLE_CONNS must be positive",
		},
		{
			name: "DB_MAX_IDLE_CONNS exceeds DB_MAX_OPEN_CONNS",
			modifier: func(c *Config) {
				c.DBMaxOpenConns = 10
				c.DBMaxIdleConns = 20
			},
			want: "DB_MAX_IDLE_CONNS (20) cannot exceed DB_MAX_OPEN_CONNS (10)",
		},
		{
			name: "zero DB_CACHE_SIZE_KB",
			modifier: func(c *Config) {
				c.DBCacheSize = 0
			},
			want: "DB_CACHE_SIZE_KB must be positive",
		},
		{
			name: "negative DB_CACHE_SIZE_KB",
			modifier: func(c *Config) {
				c.DBCacheSize = -1
			},
			want: "DB_CACHE_SIZE_KB must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:              "8080",
				LogLevel:          slog.LevelInfo,
				AllowedOrigins:    []string{"*"},
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       60 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				MaxHeaderBytes:    1 << 20,
				DBPath:            "data/timeservice.db",
				DBMaxOpenConns:    25,
				DBMaxIdleConns:    5,
				DBCacheSize:       64000,
				DBWalMode:         true,
			}

			tt.modifier(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error, got nil")
			} else if !contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}
