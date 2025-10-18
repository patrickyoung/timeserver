package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/yourorg/timeservice/internal/handler"
	"github.com/yourorg/timeservice/internal/mcpserver"
	"github.com/yourorg/timeservice/internal/middleware"
)

func main() {
	// Parse command line flags
	stdio := flag.Bool("stdio", false, "run in stdio mode for MCP communication")
	flag.Parse()

	// Setup logger
	// In stdio mode, logs must go to stderr, not stdout (stdout is for MCP protocol)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create MCP server
	mcpServer := mcpserver.NewServer(logger)

	// If stdio mode is requested, run MCP server via stdio and exit
	if *stdio {
		logger.Info("starting MCP server in stdio mode")
		if err := server.ServeStdio(mcpServer); err != nil {
			logger.Error("MCP stdio server error", "error", err)
			os.Exit(1)
		}
		return
	}

	// Otherwise run HTTP server with both REST endpoints and MCP support
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

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

	// Root endpoint with service info (handles all methods for backward compatibility)
	mux.HandleFunc("/", h.ServiceInfo)

	// Apply middleware
	handler := middleware.Chain(
		mux,
		middleware.Logger(logger),
		middleware.Recover(logger),
		middleware.CORS,
	)

	// Configure server
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server
	go func() {
		logger.Info("server starting",
			"addr", srv.Addr,
			"endpoints", map[string]string{
				"time":   "GET /api/time",
				"health": "GET /health",
				"mcp":    "POST /mcp",
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

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}
