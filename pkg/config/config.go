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
}

// Load loads configuration from environment variables with validation
func Load() (*Config, error) {
	cfg := &Config{
		// Set defaults
		Port:              getEnv("PORT", "8080"),
		Host:              getEnv("HOST", ""),
		LogLevel:          parseLogLevel(getEnv("LOG_LEVEL", "info")),
		AllowedOrigins:    parseAllowedOrigins(getEnv("ALLOWED_ORIGINS", "*")),
		ReadTimeout:       parseDuration(getEnv("READ_TIMEOUT", "10s"), 10*time.Second),
		WriteTimeout:      parseDuration(getEnv("WRITE_TIMEOUT", "10s"), 10*time.Second),
		IdleTimeout:       parseDuration(getEnv("IDLE_TIMEOUT", "60s"), 60*time.Second),
		ReadHeaderTimeout: parseDuration(getEnv("READ_HEADER_TIMEOUT", "5s"), 5*time.Second),
		ShutdownTimeout:   parseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"), 10*time.Second),
		MaxHeaderBytes:    parseInt(getEnv("MAX_HEADER_BYTES", "1048576"), 1<<20), // 1MB default
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
		return fmt.Errorf("ALLOWED_ORIGINS cannot be empty")
	}

	return nil
}

// String returns a string representation of the config (safe for logging)
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port:%s, Host:%s, LogLevel:%s, AllowedOrigins:%v, "+
		"ReadTimeout:%v, WriteTimeout:%v, IdleTimeout:%v, ReadHeaderTimeout:%v, "+
		"ShutdownTimeout:%v, MaxHeaderBytes:%d}",
		c.Port, c.Host, c.LogLevel, c.AllowedOrigins,
		c.ReadTimeout, c.WriteTimeout, c.IdleTimeout, c.ReadHeaderTimeout,
		c.ShutdownTimeout, c.MaxHeaderBytes)
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
