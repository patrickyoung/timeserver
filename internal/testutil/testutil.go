package testutil

import (
	"net/http"
	"testing"
)

// MockLogger is a test logger that captures log calls for verification
type MockLogger struct {
	InfoCalls  []LogCall
	ErrorCalls []LogCall
}

// LogCall represents a single log method invocation
type LogCall struct {
	Msg  string
	Args []any
}

// Info implements logger.Logger
func (m *MockLogger) Info(msg string, args ...any) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Msg: msg, Args: args})
}

// Error implements logger.Logger
func (m *MockLogger) Error(msg string, args ...any) {
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Msg: msg, Args: args})
}

// Reset clears all captured log calls
func (m *MockLogger) Reset() {
	m.InfoCalls = nil
	m.ErrorCalls = nil
}

// AssertInfoCount verifies the number of Info calls
func (m *MockLogger) AssertInfoCount(t *testing.T, expected int) {
	t.Helper()
	if len(m.InfoCalls) != expected {
		t.Errorf("expected %d Info calls, got %d", expected, len(m.InfoCalls))
	}
}

// AssertErrorCount verifies the number of Error calls
func (m *MockLogger) AssertErrorCount(t *testing.T, expected int) {
	t.Helper()
	if len(m.ErrorCalls) != expected {
		t.Errorf("expected %d Error calls, got %d", expected, len(m.ErrorCalls))
	}
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
