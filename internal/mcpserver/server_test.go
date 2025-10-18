package mcpserver

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yourorg/timeservice/internal/testutil"
)

func TestNewServer(t *testing.T) {
	logger := &testutil.MockLogger{}

	server := NewServer(logger)

	if server == nil {
		t.Fatal("expected server to be created")
	}

	// Verify initialization was logged
	logger.AssertInfoCount(t, 1)

	if len(logger.InfoCalls) > 0 {
		logCall := logger.InfoCalls[0]
		if logCall.Msg != "MCP server initialized" {
			t.Errorf("expected log message 'MCP server initialized', got %s", logCall.Msg)
		}
	}
}

func TestHandleGetCurrentTime(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		timezone    string
		shouldError bool
	}{
		{
			name:        "default format UTC",
			format:      "iso8601",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "unix format",
			format:      "unix",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "unixmilli format",
			format:      "unixmilli",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "rfc3339 format",
			format:      "rfc3339",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "America/New_York timezone",
			format:      "iso8601",
			timezone:    "America/New_York",
			shouldError: false,
		},
		{
			name:        "invalid timezone",
			format:      "iso8601",
			timezone:    "Invalid/Timezone",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &testutil.MockLogger{}
			ctx := context.Background()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"format":   tt.format,
						"timezone": tt.timezone,
					},
				},
			}

			result, err := handleGetCurrentTime(ctx, request, logger)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			if tt.shouldError {
				// For errors, we expect an error result
				if !result.IsError {
					t.Error("expected result to be an error")
				}
				// Verify error was logged
				logger.AssertErrorCount(t, 1)
			} else {
				// For success, verify result contains text
				if result.IsError {
					t.Errorf("expected successful result, got error: %v", result.Content)
				}

				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}

				// Verify success was logged
				logger.AssertInfoCount(t, 1)
			}
		})
	}
}

func TestHandleGetCurrentTimeDefaults(t *testing.T) {
	logger := &testutil.MockLogger{}
	ctx := context.Background()

	// Request with no arguments should use defaults
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleGetCurrentTime(ctx, request, logger)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if result.IsError {
		t.Errorf("expected successful result with defaults, got error: %v", result.Content)
	}
}

func TestHandleAddTimeOffset(t *testing.T) {
	tests := []struct {
		name    string
		hours   float64
		minutes float64
		format  string
	}{
		{
			name:    "add hours",
			hours:   2,
			minutes: 0,
			format:  "iso8601",
		},
		{
			name:    "add minutes",
			hours:   0,
			minutes: 30,
			format:  "unix",
		},
		{
			name:    "add both",
			hours:   1,
			minutes: 15,
			format:  "rfc3339",
		},
		{
			name:    "subtract hours",
			hours:   -2,
			minutes: 0,
			format:  "iso8601",
		},
		{
			name:    "fractional hours",
			hours:   1.5,
			minutes: 0,
			format:  "unixmilli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &testutil.MockLogger{}
			ctx := context.Background()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"hours":   tt.hours,
						"minutes": tt.minutes,
						"format":  tt.format,
					},
				},
			}

			result, err := handleAddTimeOffset(ctx, request, logger)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			if result.IsError {
				t.Errorf("expected successful result, got error: %v", result.Content)
			}

			if len(result.Content) == 0 {
				t.Error("expected result content to be non-empty")
			}

			// Verify logging
			logger.AssertInfoCount(t, 1)
		})
	}
}

func TestHandleAddTimeOffsetDefaults(t *testing.T) {
	logger := &testutil.MockLogger{}
	ctx := context.Background()

	// Request with no arguments should use defaults (0 hours, 0 minutes, iso8601)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleAddTimeOffset(ctx, request, logger)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if result.IsError {
		t.Errorf("expected successful result with defaults, got error: %v", result.Content)
	}
}

func TestHandleAddTimeOffsetFormats(t *testing.T) {
	formats := []string{"iso8601", "unix", "unixmilli", "rfc3339", "custom"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			logger := &testutil.MockLogger{}
			ctx := context.Background()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"hours":  1,
						"format": format,
					},
				},
			}

			result, err := handleAddTimeOffset(ctx, request, logger)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			// All formats should work (custom is treated as a Go time format string)
			if result.IsError {
				t.Errorf("expected successful result for format %s, got error: %v", format, result.Content)
			}
		})
	}
}
