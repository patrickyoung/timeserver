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

	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourorg/timeservice/internal/handler"
	"github.com/yourorg/timeservice/internal/mcpserver"
	"github.com/yourorg/timeservice/internal/middleware"
	"github.com/yourorg/timeservice/pkg/config"
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

		// Create basic MCP server without metrics (stdio mode doesn't need HTTP metrics)
		mcpServer := mcpserver.NewServer(logger)

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

	// Initialize Prometheus metrics
	metricsCollector := metrics.New("timeservice")
	metricsCollector.SetBuildInfo(version.Version, runtime.Version())

	// Create MCP server with metrics
	mcpServer := mcpserver.NewServerWithMetrics(logger, metricsCollector)

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

	// MCP endpoint (HTTP transport) - POST only for JSON-RPC
	mux.HandleFunc("POST /mcp", h.MCP)

	// Prometheus metrics endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	// Root endpoint with service info (handles all methods for backward compatibility)
	mux.HandleFunc("/", h.ServiceInfo)

	// Apply middleware with configured CORS origins
	// Note: Prometheus middleware comes first to capture all request metrics
	handler := middleware.Chain(
		mux,
		middleware.Prometheus(metricsCollector),
		middleware.Logger(logger),
		middleware.Recover(logger),
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

	logger.Info("server stopped gracefully")
}
