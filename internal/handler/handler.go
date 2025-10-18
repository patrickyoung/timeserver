package handler

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/yourorg/timeservice/pkg/logger"
	"github.com/yourorg/timeservice/pkg/mcphttp"
	"github.com/yourorg/timeservice/pkg/model"
)

// Handler handles HTTP requests
type Handler struct {
	logger        logger.Logger
	mcpHTTPServer mcphttp.Server
}

// New creates a new handler with MCP support
// Accepts interfaces for logger and MCP server for dependency inversion
func New(log logger.Logger, mcpHTTPServer mcphttp.Server) *Handler {
	return &Handler{
		logger:        log,
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

// ServiceInfo returns information about the service and available endpoints
func (h *Handler) ServiceInfo(w http.ResponseWriter, r *http.Request) {
	// Only handle root path, not all unmatched paths
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	info := model.NewServiceInfo()
	h.json(w, http.StatusOK, info)
}

// MCP handles MCP protocol requests over HTTP
func (h *Handler) MCP(w http.ResponseWriter, r *http.Request) {
	// Check for nil interface or interface with nil value
	if h.mcpHTTPServer == nil || isNil(h.mcpHTTPServer) {
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

// isNil checks if an interface contains a nil value (handles typed nil)
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Ptr && v.IsNil()
}
