package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// MCP tool metrics
	MCPToolCallsTotal    *prometheus.CounterVec
	MCPToolCallDuration  *prometheus.HistogramVec
	MCPToolCallsInFlight prometheus.Gauge

	// Application metrics
	BuildInfo *prometheus.GaugeVec
}

// New creates a new Metrics instance with all metrics registered
func New(namespace string) *Metrics {
	m := &Metrics{
		// HTTP request counter by method, path, and status code
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),

		// HTTP request duration histogram
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
			},
			[]string{"method", "path"},
		),

		// HTTP request size histogram
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100, 1000, 10000, ..., 100000000
			},
			[]string{"method", "path"},
		),

		// HTTP response size histogram
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path"},
		),

		// HTTP requests currently in flight
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),

		// MCP tool call counter by tool name and status
		MCPToolCallsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "mcp_tool_calls_total",
				Help:      "Total number of MCP tool calls",
			},
			[]string{"tool", "status"},
		),

		// MCP tool call duration histogram
		MCPToolCallDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "mcp_tool_call_duration_seconds",
				Help:      "MCP tool call duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"tool"},
		),

		// MCP tool calls currently in flight
		MCPToolCallsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "mcp_tool_calls_in_flight",
				Help:      "Number of MCP tool calls currently being processed",
			},
		),

		// Build info metric (always 1, labeled with version info)
		BuildInfo: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "build_info",
				Help:      "Build information",
			},
			[]string{"version", "go_version"},
		),
	}

	return m
}

// SetBuildInfo sets the build info metric
func (m *Metrics) SetBuildInfo(version, goVersion string) {
	m.BuildInfo.WithLabelValues(version, goVersion).Set(1)
}
