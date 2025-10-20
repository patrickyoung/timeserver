package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourorg/timeservice/internal/handler"
	"github.com/yourorg/timeservice/internal/mcpserver"
	"github.com/yourorg/timeservice/internal/middleware"
	"github.com/yourorg/timeservice/internal/repository"
	"github.com/yourorg/timeservice/pkg/auth"
	"github.com/yourorg/timeservice/pkg/config"
	"github.com/yourorg/timeservice/pkg/db"
	"github.com/yourorg/timeservice/pkg/metrics"
	"github.com/yourorg/timeservice/pkg/version"
)

func main() {
	// Parse command line flags
	stdio := flag.Bool("stdio", false, "run in stdio mode for MCP communication")
	flag.Parse()

	// If stdio mode is requested, run with minimal config (no CORS needed)
	if *stdio {
		// Minimal logger for stdio mode (logs to stderr)
		logLevel := config.ParseLogLevelFromEnv()
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: logLevel,
		}))

		logger.Info("starting MCP server in stdio mode")

		// Initialize database for stdio mode
		dbConfig := db.DefaultConfig()
		database, err := db.Open(dbConfig, logger)
		if err != nil {
			logger.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer database.Close()

		// Run migrations
		if err := db.Migrate(database, logger); err != nil {
			logger.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}

		// Initialize metrics for stdio mode (minimal, for database tracking)
		metricsCollector := metrics.New("timeservice")

		// Check feature flag for stdio mode
		locationsEnabled := os.Getenv("FEATURE_LOCATIONS_ENABLED") != "false" // Default: true

		// Initialize location repository with metrics (conditionally)
		var locationRepo repository.LocationRepository
		if locationsEnabled {
			locationRepo = repository.NewLocationRepository(database, metricsCollector)
			logger.Info("location features enabled in stdio mode")
		} else {
			logger.Info("location features disabled in stdio mode via FEATURE_LOCATIONS_ENABLED=false")
		}

		// Create MCP server with metrics and location repository
		mcpServer := mcpserver.NewServerWithMetrics(logger, metricsCollector, locationRepo)

		if err := server.ServeStdio(mcpServer); err != nil {
			logger.Error("MCP stdio server error", "error", err)
			os.Exit(1)
		}
		return
	}

	// HTTP mode: Load full configuration with CORS validation
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logger with configured log level
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	// Log configuration on startup (helps with debugging deployment issues)
	logger.Info("configuration loaded",
		"port", cfg.Port,
		"log_level", cfg.LogLevel.String(),
		"allowed_origins", cfg.AllowedOrigins,
		"read_timeout", cfg.ReadTimeout,
		"write_timeout", cfg.WriteTimeout,
		"idle_timeout", cfg.IdleTimeout,
		"auth_enabled", cfg.AuthEnabled,
		"oidc_issuer", cfg.OIDCIssuerURL,
		"db_path", cfg.DBPath,
		"db_max_open_conns", cfg.DBMaxOpenConns,
		"db_max_idle_conns", cfg.DBMaxIdleConns,
		"db_cache_size_kb", cfg.DBCacheSize,
		"db_wal_mode", cfg.DBWalMode,
		"locations_enabled", cfg.LocationsEnabled,
	)

	// Warn if wildcard CORS is configured (security risk)
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			logger.Warn("wildcard CORS (*) is enabled - this is INSECURE for production",
				"recommendation", "set explicit origins in ALLOWED_ORIGINS",
				"dev_only", "use ALLOW_CORS_WILDCARD_DEV=true only in development",
			)
			break
		}
	}

	// Initialize authenticator if auth is enabled
	var authenticator *auth.Authenticator
	if cfg.AuthEnabled {
		// Build required roles/permissions/scopes from config
		var requiredRoles []string
		if cfg.AuthRequiredRole != "" {
			requiredRoles = []string{cfg.AuthRequiredRole}
		}
		var requiredPermissions []string
		if cfg.AuthRequiredPermission != "" {
			requiredPermissions = []string{cfg.AuthRequiredPermission}
		}
		var requiredScopes []string
		if cfg.AuthRequiredScope != "" {
			requiredScopes = []string{cfg.AuthRequiredScope}
		}

		authConfig := &auth.Config{
			IssuerURL:           cfg.OIDCIssuerURL,
			Audience:            cfg.OIDCAudience,
			SkipExpiryCheck:     cfg.OIDCSkipExpiryCheck,
			SkipClientIDCheck:   cfg.OIDCSkipClientIDCheck,
			SkipIssuerCheck:     cfg.OIDCSkipIssuerCheck,
			RequiredRoles:       requiredRoles,
			RequiredPermissions: requiredPermissions,
			RequiredScopes:      requiredScopes,
		}

		var err error
		authenticator, err = auth.NewAuthenticator(context.Background(), authConfig, logger)
		if err != nil {
			logger.Error("failed to initialize authenticator", "error", err)
			os.Exit(1)
		}

		logger.Info("authentication enabled",
			"issuer", cfg.OIDCIssuerURL,
			"audience", cfg.OIDCAudience,
			"public_paths", cfg.AuthPublicPaths,
			"required_role", cfg.AuthRequiredRole,
			"required_permission", cfg.AuthRequiredPermission,
			"required_scope", cfg.AuthRequiredScope,
		)
	} else {
		logger.Info("authentication disabled - all endpoints are unprotected",
			"recommendation", "enable auth in production with AUTH_ENABLED=true",
		)
	}

	// Initialize database with configuration from config
	dbConfig := &db.Config{
		Path:         cfg.DBPath,
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxIdleConns,
		CacheSize:    -cfg.DBCacheSize, // Convert KB to negative pages for SQLite
		BusyTimeout:  5000,             // 5 seconds
		WalMode:      cfg.DBWalMode,
		SyncMode:     "NORMAL",
		ForeignKeys:  true,
		JournalMode:  "WAL",
	}
	if !cfg.DBWalMode {
		dbConfig.JournalMode = "DELETE"
	}

	database, err := db.Open(dbConfig, logger)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	// Run migrations
	if err := db.Migrate(database, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		database.Close()
		os.Exit(1)
	}

	// Initialize Prometheus metrics
	metricsCollector := metrics.New("timeservice")
	metricsCollector.SetBuildInfo(version.Version, runtime.Version())

	// Initialize repositories with metrics (conditionally based on feature flags)
	var locationRepo repository.LocationRepository
	if cfg.LocationsEnabled {
		locationRepo = repository.NewLocationRepository(database, metricsCollector)
		logger.Info("location repository initialized")
	} else {
		logger.Info("location repository disabled via feature flag")
	}

	// Start goroutine to periodically update database connection pool metrics
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := database.Stats()
			metricsCollector.DBConnectionsOpen.Set(float64(stats.OpenConnections))
			metricsCollector.DBConnectionsIdle.Set(float64(stats.Idle))
		}
	}()

	// Create MCP server with metrics and location repository
	// locationRepo will be nil if feature is disabled, causing MCP tools to be skipped
	mcpServer := mcpserver.NewServerWithMetrics(logger, metricsCollector, locationRepo)

	// Otherwise run HTTP server with both REST endpoints and MCP support

	// Create StreamableHTTPServer for MCP over HTTP
	mcpHTTPServer := server.NewStreamableHTTPServer(mcpServer)

	// Create HTTP handler - only needs the StreamableHTTPServer, not the full MCPServer
	h := handler.New(logger, mcpHTTPServer)

	// Setup router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", h.Health)

	// Time endpoint
	mux.HandleFunc("GET /api/time", h.GetTime)

	// Location management endpoints (feature-flagged)
	if cfg.LocationsEnabled {
		locationHandler := handler.NewLocationHandler(locationRepo, logger)
		mux.HandleFunc("POST /api/locations", locationHandler.CreateLocation)
		mux.HandleFunc("GET /api/locations", locationHandler.ListLocations)
		mux.HandleFunc("GET /api/locations/{name}", locationHandler.GetLocation)
		mux.HandleFunc("PUT /api/locations/{name}", locationHandler.UpdateLocation)
		mux.HandleFunc("DELETE /api/locations/{name}", locationHandler.DeleteLocation)
		mux.HandleFunc("GET /api/locations/{name}/time", locationHandler.GetLocationTime)
		logger.Info("location features enabled", "endpoints", 6)
	} else {
		logger.Info("location features disabled via FEATURE_LOCATIONS_ENABLED=false")
	}

	// MCP endpoint (HTTP transport) - POST only for JSON-RPC
	mux.HandleFunc("POST /mcp", h.MCP)

	// Prometheus metrics endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	// Root endpoint with service info (handles all methods for backward compatibility)
	mux.HandleFunc("/", h.ServiceInfo)

	// Apply middleware with configured CORS origins and auth
	// Note: Prometheus middleware comes first to capture all request metrics
	// Auth middleware comes after logging/recovery but before CORS to ensure security
	handler := middleware.Chain(
		mux,
		middleware.Prometheus(metricsCollector),
		middleware.Logger(logger),
		middleware.Recover(logger),
		middleware.Auth(&middleware.AuthConfig{
			Enabled:       cfg.AuthEnabled,
			Authenticator: authenticator,
			PublicPaths:   cfg.AuthPublicPaths,
			Logger:        logger,
			Metrics:       metricsCollector,
		}),
		middleware.CORSWithOrigins(cfg.AllowedOrigins),
	)

	// Configure server with timeouts from config
	// Use net.JoinHostPort to properly handle IPv6 addresses (e.g., [::1]:8080)
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server
	go func() {
		logger.Info("server starting",
			"addr", srv.Addr,
			"endpoints", map[string]string{
				"time":    "GET /api/time",
				"health":  "GET /health",
				"mcp":     "POST /mcp",
				"metrics": "GET /metrics",
			},
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	logger.Info("shutting down server...")

	// Shutdown with configured timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	// Log database statistics before closing
	stats := database.Stats()
	logger.Info("database statistics",
		"max_open_connections", stats.MaxOpenConnections,
		"open_connections", stats.OpenConnections,
		"in_use", stats.InUse,
		"idle", stats.Idle,
		"wait_count", stats.WaitCount,
		"wait_duration", stats.WaitDuration,
		"max_idle_closed", stats.MaxIdleClosed,
		"max_idle_time_closed", stats.MaxIdleTimeClosed,
		"max_lifetime_closed", stats.MaxLifetimeClosed,
	)

	// Close database connection
	if err := db.Close(database, logger); err != nil {
		logger.Error("database close error", "error", err)
	}

	logger.Info("server stopped gracefully")
}
