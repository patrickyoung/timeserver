package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourorg/timeservice/internal/testutil"
)

func TestLogger(t *testing.T) {
	logger, logHandler := testutil.NewTestLogger()

	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify logging occurred
	logHandler.AssertInfoCount(t, 1)

	if len(logHandler.InfoCalls) > 0 {
		logCall := logHandler.InfoCalls[0]
		if logCall.Msg != "request" {
			t.Errorf("expected log message 'request', got %s", logCall.Msg)
		}

		// Verify log contains expected fields (method, path, status, duration_ms, ip)
		if len(logCall.Attrs) < 5 { // 5 fields
			t.Errorf("expected at least 5 log attributes, got %d", len(logCall.Attrs))
		}
	}
}

func TestLoggerCapturesStatusCode(t *testing.T) {
	tests := []struct {
		name           string
		handlerStatus  int
		expectedStatus int
	}{
		{
			name:           "200 OK",
			handlerStatus:  http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "404 Not Found",
			handlerStatus:  http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "500 Internal Server Error",
			handlerStatus:  http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger()

			handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	logger, logHandler := testutil.NewTestLogger()

	handler := Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic
	handler.ServeHTTP(w, req)

	// Verify error was logged
	logHandler.AssertErrorCount(t, 1)

	if len(logHandler.ErrorCalls) > 0 {
		logCall := logHandler.ErrorCalls[0]
		if logCall.Msg != "panic recovered" {
			t.Errorf("expected log message 'panic recovered', got %s", logCall.Msg)
		}
	}

	// Verify response status
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestRecoverDoesNotInterceptNormalFlow(t *testing.T) {
	logger, logHandler := testutil.NewTestLogger()

	handler := Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify no errors were logged
	logHandler.AssertErrorCount(t, 0)

	// Verify normal response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got %s", w.Body.String())
	}
}

func TestCORS(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %s", origin)
	}

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, PUT, DELETE, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods to be set correctly, got %s", methods)
	}

	if headers := w.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type, Authorization" {
		t.Errorf("expected Access-Control-Allow-Headers to be set correctly, got %s", headers)
	}
}

func TestCORSOptionsRequest(t *testing.T) {
	handlerCalled := false
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify OPTIONS returns 204
	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify next handler was NOT called
	if handlerCalled {
		t.Error("expected handler not to be called for OPTIONS request")
	}

	// Verify CORS headers are still set
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %s", origin)
	}
}

func TestChain(t *testing.T) {
	// Track middleware execution order
	var order []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "before-1")
			next.ServeHTTP(w, r)
			order = append(order, "after-1")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "before-2")
			next.ServeHTTP(w, r)
			order = append(order, "after-2")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(handler, middleware1, middleware2)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	// Verify execution order
	expected := []string{"before-1", "before-2", "handler", "after-2", "after-1"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}

	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("at position %d: expected %s, got %s", i, exp, order[i])
		}
	}
}

func TestResponseWriterCapturesStatus(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected statusCode %d, got %d", http.StatusNotFound, rw.statusCode)
	}
}

func TestResponseWriterDefaultStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// Write without calling WriteHeader explicitly
	rw.Write([]byte("test"))

	// Should default to 200
	if rw.statusCode != http.StatusOK {
		t.Errorf("expected statusCode %d, got %d", http.StatusOK, rw.statusCode)
	}
}

func TestResponseWriterDoesNotWriteHeaderTwice(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusNotFound) // Should be ignored

	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected statusCode %d, got %d", http.StatusCreated, rw.statusCode)
	}
}

func TestComputeApproximateRequestSize(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		contentLength  int64
		expectPositive bool
		expectMinSize  int
	}{
		{
			name:           "GET request without body (ContentLength -1)",
			method:         "GET",
			path:           "/api/time",
			headers:        map[string]string{"User-Agent": "test"},
			contentLength:  -1, // Sentinel value for unknown/not set
			expectPositive: true,
			expectMinSize:  10, // At least method + path
		},
		{
			name:           "POST request with body",
			method:         "POST",
			path:           "/mcp",
			headers:        map[string]string{"Content-Type": "application/json"},
			contentLength:  100,
			expectPositive: true,
			expectMinSize:  100, // Should include body size
		},
		{
			name:           "POST request with zero ContentLength",
			method:         "POST",
			path:           "/api/test",
			headers:        map[string]string{},
			contentLength:  0,
			expectPositive: true,
			expectMinSize:  10,
		},
		{
			name:           "Request with large body",
			method:         "PUT",
			path:           "/upload",
			headers:        map[string]string{"Content-Type": "application/octet-stream"},
			contentLength:  1048576, // 1MB
			expectPositive: true,
			expectMinSize:  1048576,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.ContentLength = tt.contentLength

			size := computeApproximateRequestSize(req)

			if tt.expectPositive && size < 0 {
				t.Errorf("expected positive size, got %d (ContentLength was %d)", size, tt.contentLength)
			}

			if size < tt.expectMinSize {
				t.Errorf("expected size >= %d, got %d", tt.expectMinSize, size)
			}
		})
	}
}
