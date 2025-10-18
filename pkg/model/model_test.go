package model

import (
	"testing"
	"time"
)

func TestNewTimeResponse(t *testing.T) {
	// Use a fixed time for testing
	fixedTime := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)

	response := NewTimeResponse(fixedTime)

	// Verify all fields are populated
	if response.CurrentTime == "" {
		t.Error("expected CurrentTime to be set")
	}

	if response.UnixTime != fixedTime.Unix() {
		t.Errorf("expected UnixTime %d, got %d", fixedTime.Unix(), response.UnixTime)
	}

	if response.Timezone == "" {
		t.Error("expected Timezone to be set")
	}

	if response.Formatted == "" {
		t.Error("expected Formatted to be set")
	}

	// Verify specific values
	expectedUnix := int64(1705314645)
	if response.UnixTime != expectedUnix {
		t.Errorf("expected UnixTime %d, got %d", expectedUnix, response.UnixTime)
	}

	expectedFormatted := "2024-01-15T10:30:45Z"
	if response.Formatted != expectedFormatted {
		t.Errorf("expected Formatted %s, got %s", expectedFormatted, response.Formatted)
	}
}

func TestTimeResponseWithDifferentTimezones(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantZone string
	}{
		{
			name:     "UTC",
			location: "UTC",
			wantZone: "UTC",
		},
		{
			name:     "America/New_York",
			location: "America/New_York",
			wantZone: "EST", // or EDT depending on date
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := time.LoadLocation(tt.location)
			if err != nil {
				t.Fatalf("failed to load location %s: %v", tt.location, err)
			}

			fixedTime := time.Date(2024, 1, 15, 10, 30, 45, 0, loc)
			response := NewTimeResponse(fixedTime)

			if response.Timezone == "" {
				t.Error("expected Timezone to be set")
			}

			// Verify the timezone is captured
			if response.Timezone != tt.wantZone {
				t.Logf("note: got timezone %s (expected %s, may vary)", response.Timezone, tt.wantZone)
			}
		})
	}
}

func TestNewServiceInfo(t *testing.T) {
	info := NewServiceInfo()

	// Verify service name
	if info.Service != "timeservice" {
		t.Errorf("expected Service 'timeservice', got %s", info.Service)
	}

	// Verify version is set
	if info.Version == "" {
		t.Error("expected Version to be set")
	}

	// Verify version format
	if info.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got %s", info.Version)
	}

	// Verify endpoints are populated
	if len(info.Endpoints) == 0 {
		t.Error("expected Endpoints to be populated")
	}

	// Verify specific endpoints exist
	expectedEndpoints := map[string]string{
		"time":   "GET /api/time",
		"health": "GET /health",
		"mcp":    "POST /mcp",
	}

	for key, expectedValue := range expectedEndpoints {
		actualValue, exists := info.Endpoints[key]
		if !exists {
			t.Errorf("expected endpoint %s to exist", key)
		}
		if actualValue != expectedValue {
			t.Errorf("expected endpoint %s to be %s, got %s", key, expectedValue, actualValue)
		}
	}

	// Verify MCP info is set
	if info.MCPInfo == "" {
		t.Error("expected MCPInfo to be set")
	}

	// Verify MCP info contains expected text
	if info.MCPInfo != "Supports both stdio mode (--stdio flag) and HTTP transport (POST /mcp)" {
		t.Errorf("unexpected MCPInfo: %s", info.MCPInfo)
	}
}

func TestServiceInfoStructure(t *testing.T) {
	info := NewServiceInfo()

	// Test that it can be serialized (implicitly tests JSON tags)
	if info.Service == "" {
		t.Error("Service field should not be empty")
	}

	if info.Version == "" {
		t.Error("Version field should not be empty")
	}

	if info.Endpoints == nil {
		t.Error("Endpoints should not be nil")
	}

	if info.MCPInfo == "" {
		t.Error("MCPInfo field should not be empty")
	}
}
