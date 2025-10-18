package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourorg/timeservice/internal/testutil"
	"github.com/yourorg/timeservice/pkg/model"
)

func TestGetTime(t *testing.T) {
	logger := &testutil.MockLogger{}
	mcpServer := &testutil.MockMCPServer{}
	h := New(logger, mcpServer)

	req := httptest.NewRequest(http.MethodGet, "/api/time", nil)
	w := httptest.NewRecorder()

	h.GetTime(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Verify response structure
	var response model.TimeResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.CurrentTime == "" {
		t.Error("expected CurrentTime to be set")
	}

	if response.UnixTime == 0 {
		t.Error("expected UnixTime to be non-zero")
	}

	if response.Timezone == "" {
		t.Error("expected Timezone to be set")
	}

	// Verify logging occurred
	logger.AssertInfoCount(t, 1)
}

func TestHealth(t *testing.T) {
	logger := &testutil.MockLogger{}
	mcpServer := &testutil.MockMCPServer{}
	h := New(logger, mcpServer)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	status, ok := response["status"].(string)
	if !ok || status != "healthy" {
		t.Errorf("expected status 'healthy', got %v", response["status"])
	}

	if _, ok := response["time"]; !ok {
		t.Error("expected time field in response")
	}
}

func TestServiceInfo(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		shouldDecode   bool
	}{
		{
			name:           "root path",
			path:           "/",
			expectedStatus: http.StatusOK,
			shouldDecode:   true,
		},
		{
			name:           "non-root path",
			path:           "/some/other/path",
			expectedStatus: http.StatusNotFound,
			shouldDecode:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &testutil.MockLogger{}
			mcpServer := &testutil.MockMCPServer{}
			h := New(logger, mcpServer)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			h.ServiceInfo(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.shouldDecode {
				var response model.ServiceInfo
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response.Service != "timeservice" {
					t.Errorf("expected service 'timeservice', got %s", response.Service)
				}

				if response.Version == "" {
					t.Error("expected version to be set")
				}

				if len(response.Endpoints) == 0 {
					t.Error("expected endpoints to be populated")
				}
			}
		})
	}
}

func TestMCP(t *testing.T) {
	tests := []struct {
		name           string
		mcpServer      *testutil.MockMCPServer
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "nil server",
			mcpServer:      nil,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"MCP server not initialized"}`,
		},
		{
			name: "server returns OK",
			mcpServer: &testutil.MockMCPServer{
				ServeHTTPFunc: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"result":"success"}`))
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"result":"success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &testutil.MockLogger{}
			h := New(logger, tt.mcpServer)

			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			w := httptest.NewRecorder()

			h.MCP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Decode and compare as JSON to avoid newline issues
			var gotBody, expectedBody map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &gotBody); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expectedBody), &expectedBody); err != nil {
				t.Fatalf("failed to decode expected body: %v", err)
			}

			// Compare relevant fields
			for key, expectedVal := range expectedBody {
				if gotVal, ok := gotBody[key]; !ok {
					t.Errorf("missing key %q in response", key)
				} else if gotVal != expectedVal {
					t.Errorf("for key %q: expected %v, got %v", key, expectedVal, gotVal)
				}
			}
		})
	}
}

func TestHandlerJSONHelpers(t *testing.T) {
	t.Run("json helper", func(t *testing.T) {
		logger := &testutil.MockLogger{}
		h := New(logger, nil)

		w := httptest.NewRecorder()
		data := map[string]string{"key": "value"}

		h.json(w, http.StatusCreated, data)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		}

		if ct := w.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var result map[string]string
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if result["key"] != "value" {
			t.Errorf("expected value, got %s", result["key"])
		}
	})

	t.Run("error helper", func(t *testing.T) {
		logger := &testutil.MockLogger{}
		h := New(logger, nil)

		w := httptest.NewRecorder()

		h.error(w, http.StatusBadRequest, "test error")

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var result map[string]string
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if result["error"] != "test error" {
			t.Errorf("expected 'test error', got %s", result["error"])
		}
	})
}
