package testutil

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
)

// TestLogHandler is a slog.Handler that captures log calls for testing
type TestLogHandler struct {
	InfoCalls  []LogCall
	ErrorCalls []LogCall
}

// LogCall represents a single log method invocation
type LogCall struct {
	Msg   string
	Level slog.Level
	Attrs []slog.Attr
}

// Enabled always returns true for testing
func (h *TestLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle captures log records
func (h *TestLogHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	call := LogCall{
		Msg:   r.Message,
		Level: r.Level,
		Attrs: attrs,
	}

	if r.Level == slog.LevelError {
		h.ErrorCalls = append(h.ErrorCalls, call)
	} else {
		h.InfoCalls = append(h.InfoCalls, call)
	}

	return nil
}

// WithAttrs returns a new handler with additional attributes
func (h *TestLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For testing, we can ignore attrs and return the same handler
	return h
}

// WithGroup returns a new handler with a group
func (h *TestLogHandler) WithGroup(name string) slog.Handler {
	// For testing, we can ignore groups and return the same handler
	return h
}

// Reset clears all captured log calls
func (h *TestLogHandler) Reset() {
	h.InfoCalls = nil
	h.ErrorCalls = nil
}

// AssertInfoCount verifies the number of Info calls
func (h *TestLogHandler) AssertInfoCount(t *testing.T, expected int) {
	t.Helper()
	if len(h.InfoCalls) != expected {
		t.Errorf("expected %d Info calls, got %d", expected, len(h.InfoCalls))
	}
}

// AssertErrorCount verifies the number of Error calls
func (h *TestLogHandler) AssertErrorCount(t *testing.T, expected int) {
	t.Helper()
	if len(h.ErrorCalls) != expected {
		t.Errorf("expected %d Error calls, got %d", expected, len(h.ErrorCalls))
	}
}

// NewTestLogger creates a logger with a TestLogHandler for testing
func NewTestLogger() (*slog.Logger, *TestLogHandler) {
	handler := &TestLogHandler{}
	return slog.New(handler), handler
}

// MockMCPServer is a test MCP HTTP server
type MockMCPServer struct {
	ServeHTTPFunc func(http.ResponseWriter, *http.Request)
}

// ServeHTTP implements http.Handler
func (m *MockMCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle nil receiver (shouldn't happen in practice, but tests may pass nil)
	if m == nil {
		http.Error(w, "MockMCPServer is nil", http.StatusInternalServerError)
		return
	}

	if m.ServeHTTPFunc != nil {
		m.ServeHTTPFunc(w, r)
	} else {
		// Default behavior if no func is set
		w.WriteHeader(http.StatusOK)
	}
}
