package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/yourorg/timeservice/pkg/model"
)

// Handler handles HTTP requests
type Handler struct {
	logger        *slog.Logger
	mcpHTTPServer *server.StreamableHTTPServer
}

// New creates a new handler with MCP support
func New(logger *slog.Logger, mcpServer *server.MCPServer) *Handler {
	// Create StreamableHTTPServer for handling MCP over HTTP
	mcpHTTPServer := server.NewStreamableHTTPServer(mcpServer)

	return &Handler{
		logger:        logger,
		mcpHTTPServer: mcpHTTPServer,
	}
}

// GetTime returns the current server time
func (h *Handler) GetTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	response := model.NewTimeResponse(now)

	h.logger.Info("time request",
		"remote_addr", r.RemoteAddr,
		"time", response.Formatted,
	)

	h.json(w, http.StatusOK, response)
}

// Health returns a health check response
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.json(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// MCP handles MCP protocol requests over HTTP
func (h *Handler) MCP(w http.ResponseWriter, r *http.Request) {
	if h.mcpHTTPServer == nil {
		h.error(w, http.StatusInternalServerError, "MCP server not initialized")
		return
	}

	// Delegate to the StreamableHTTPServer
	h.mcpHTTPServer.ServeHTTP(w, r)
}

// json sends a JSON response
func (h *Handler) json(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("json encode error", "error", err)
	}
}

// error sends an error JSON response
func (h *Handler) error(w http.ResponseWriter, status int, message string) {
	h.json(w, status, map[string]string{"error": message})
}
