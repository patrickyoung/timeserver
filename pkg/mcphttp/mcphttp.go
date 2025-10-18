package mcphttp

import "net/http"

// Server defines the interface for MCP HTTP server
// This abstraction allows the handler to depend on behavior rather than concrete implementation
type Server interface {
	// ServeHTTP handles HTTP requests for MCP protocol
	http.Handler
}
