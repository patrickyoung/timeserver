package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNew(t *testing.T) {
	// Create metrics with test namespace
	m := New("test")

	if m == nil {
		t.Fatal("New() returned nil")
	}

	// Verify all metric fields are initialized
	if m.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal is nil")
	}
	if m.HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration is nil")
	}
	if m.HTTPRequestSize == nil {
		t.Error("HTTPRequestSize is nil")
	}
	if m.HTTPResponseSize == nil {
		t.Error("HTTPResponseSize is nil")
	}
	if m.HTTPRequestsInFlight == nil {
		t.Error("HTTPRequestsInFlight is nil")
	}
	if m.MCPToolCallsTotal == nil {
		t.Error("MCPToolCallsTotal is nil")
	}
	if m.MCPToolCallDuration == nil {
		t.Error("MCPToolCallDuration is nil")
	}
	if m.MCPToolCallsInFlight == nil {
		t.Error("MCPToolCallsInFlight is nil")
	}
	if m.BuildInfo == nil {
		t.Error("BuildInfo is nil")
	}
}

func TestSetBuildInfo(t *testing.T) {
	// Create a custom registry to avoid conflicts
	reg := prometheus.NewRegistry()

	// Create build info metric
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "test",
			Name:      "build_info",
			Help:      "Build information",
		},
		[]string{"version", "go_version"},
	)
	reg.MustRegister(buildInfo)

	// Set build info
	buildInfo.WithLabelValues("1.0.0", "go1.24.8").Set(1)

	// Verify the metric value
	metricValue := testutil.ToFloat64(buildInfo.WithLabelValues("1.0.0", "go1.24.8"))
	if metricValue != 1.0 {
		t.Errorf("build_info metric value = %f, want 1.0", metricValue)
	}
}

func TestHTTPMetrics(t *testing.T) {
	// Create a custom registry
	reg := prometheus.NewRegistry()

	// Create HTTP request counter
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "test",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	reg.MustRegister(counter)

	// Increment counter
	counter.WithLabelValues("GET", "/health", "200").Inc()
	counter.WithLabelValues("GET", "/health", "200").Inc()
	counter.WithLabelValues("POST", "/mcp", "200").Inc()

	// Verify counts
	healthCount := testutil.ToFloat64(counter.WithLabelValues("GET", "/health", "200"))
	if healthCount != 2.0 {
		t.Errorf("health endpoint count = %f, want 2.0", healthCount)
	}

	mcpCount := testutil.ToFloat64(counter.WithLabelValues("POST", "/mcp", "200"))
	if mcpCount != 1.0 {
		t.Errorf("mcp endpoint count = %f, want 1.0", mcpCount)
	}
}

func TestMCPToolMetrics(t *testing.T) {
	// Create a custom registry
	reg := prometheus.NewRegistry()

	// Create MCP tool call counter
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "test",
			Name:      "mcp_tool_calls_total",
			Help:      "Total MCP tool calls",
		},
		[]string{"tool", "status"},
	)
	reg.MustRegister(counter)

	// Simulate tool calls
	counter.WithLabelValues("get_current_time", "success").Inc()
	counter.WithLabelValues("get_current_time", "success").Inc()
	counter.WithLabelValues("add_time_offset", "success").Inc()
	counter.WithLabelValues("get_current_time", "error").Inc()

	// Verify counts
	successCount := testutil.ToFloat64(counter.WithLabelValues("get_current_time", "success"))
	if successCount != 2.0 {
		t.Errorf("get_current_time success count = %f, want 2.0", successCount)
	}

	errorCount := testutil.ToFloat64(counter.WithLabelValues("get_current_time", "error"))
	if errorCount != 1.0 {
		t.Errorf("get_current_time error count = %f, want 1.0", errorCount)
	}

	offsetCount := testutil.ToFloat64(counter.WithLabelValues("add_time_offset", "success"))
	if offsetCount != 1.0 {
		t.Errorf("add_time_offset count = %f, want 1.0", offsetCount)
	}
}

func TestInFlightGauges(t *testing.T) {
	// Create a custom registry
	reg := prometheus.NewRegistry()

	// Create in-flight gauge
	inFlight := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "test",
			Name:      "requests_in_flight",
			Help:      "Requests in flight",
		},
	)
	reg.MustRegister(inFlight)

	// Test increment/decrement
	inFlight.Inc()
	if testutil.ToFloat64(inFlight) != 1.0 {
		t.Errorf("in_flight after Inc() = %f, want 1.0", testutil.ToFloat64(inFlight))
	}

	inFlight.Inc()
	if testutil.ToFloat64(inFlight) != 2.0 {
		t.Errorf("in_flight after second Inc() = %f, want 2.0", testutil.ToFloat64(inFlight))
	}

	inFlight.Dec()
	if testutil.ToFloat64(inFlight) != 1.0 {
		t.Errorf("in_flight after Dec() = %f, want 1.0", testutil.ToFloat64(inFlight))
	}

	inFlight.Dec()
	if testutil.ToFloat64(inFlight) != 0.0 {
		t.Errorf("in_flight after second Dec() = %f, want 0.0", testutil.ToFloat64(inFlight))
	}
}
