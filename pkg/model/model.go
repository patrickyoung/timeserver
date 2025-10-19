package model

import (
	"time"

	"github.com/yourorg/timeservice/pkg/version"
)

// TimeResponse represents a time response with various formats
type TimeResponse struct {
	CurrentTime string `json:"current_time"`
	UnixTime    int64  `json:"unix_time"`
	Timezone    string `json:"timezone"`
	Formatted   string `json:"formatted"`
}

// NewTimeResponse creates a new TimeResponse from a time.Time
func NewTimeResponse(t time.Time) *TimeResponse {
	zone, _ := t.Zone()
	return &TimeResponse{
		CurrentTime: t.Format(time.RFC3339Nano),
		UnixTime:    t.Unix(),
		Timezone:    zone,
		Formatted:   t.Format(time.RFC3339),
	}
}

// ServiceInfo represents the service information response
type ServiceInfo struct {
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Endpoints map[string]string `json:"endpoints"`
	MCPInfo   string            `json:"mcp_info"`
}

// NewServiceInfo creates a new ServiceInfo with default values
func NewServiceInfo() *ServiceInfo {
	return &ServiceInfo{
		Service: version.ServiceName,
		Version: version.Version,
		Endpoints: map[string]string{
			"time":    "GET /api/time",
			"health":  "GET /health",
			"mcp":     "POST /mcp",
			"metrics": "GET /metrics",
		},
		MCPInfo: "Supports both stdio mode (--stdio flag) and HTTP transport (POST /mcp)",
	}
}
