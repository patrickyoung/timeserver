package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestListTools(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := NewServer(logger)

	reqBody := Request{
		Method: "tools/list",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.HandleHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("expected tools to be an array")
	}

	if len(tools) == 0 {
		t.Error("expected at least one tool to be registered")
	}
}

func TestCallTool_GetCurrentTime(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := NewServer(logger)

	reqBody := Request{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name": "get_current_time",
			"arguments": map[string]interface{}{
				"format":   "unix",
				"timezone": "UTC",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.HandleHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("expected no error, got: %s", response.Error.Message)
	}
}

func TestCallTool_AddTimeOffset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := NewServer(logger)

	reqBody := Request{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name": "add_time_offset",
			"arguments": map[string]interface{}{
				"hours":   2.0,
				"minutes": 30.0,
				"format":  "iso8601",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.HandleHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("expected no error, got: %s", response.Error.Message)
	}
}

func TestRegisterTool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := NewServer(logger)

	initialCount := len(server.tools)

	// Register a custom tool
	server.RegisterTool(Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "test result", nil
		},
	})

	if len(server.tools) != initialCount+1 {
		t.Errorf("expected %d tools, got %d", initialCount+1, len(server.tools))
	}

	if _, exists := server.tools["test_tool"]; !exists {
		t.Error("expected test_tool to be registered")
	}
}

func TestCallTool_InvalidToolName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := NewServer(logger)

	reqBody := Request{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name": "nonexistent_tool",
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.HandleHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestInvalidMethod(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := NewServer(logger)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	w := httptest.NewRecorder()

	server.HandleHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
