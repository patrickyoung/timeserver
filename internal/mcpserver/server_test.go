package mcpserver

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yourorg/timeservice/internal/testutil"
)

func TestNewServer(t *testing.T) {
	logger, logHandler := testutil.NewTestLogger()

	server := NewServer(logger)

	if server == nil {
		t.Fatal("expected server to be created")
	}

	// Verify initialization was logged
	logHandler.AssertInfoCount(t, 1)

	if len(logHandler.InfoCalls) > 0 {
		logCall := logHandler.InfoCalls[0]
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
			logger, logHandler := testutil.NewTestLogger()
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
				logHandler.AssertErrorCount(t, 1)
			} else {
				// For success, verify result contains text
				if result.IsError {
					t.Errorf("expected successful result, got error: %v", result.Content)
				}

				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}

				// Verify success was logged
				logHandler.AssertInfoCount(t, 1)
			}
		})
	}
}

func TestHandleGetCurrentTimeDefaults(t *testing.T) {
	logger, _ := testutil.NewTestLogger()
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

func TestHandleGetCurrentTimeCustomFormats(t *testing.T) {
	tests := []struct {
		name         string
		customFormat string
	}{
		{
			name:         "date only format",
			customFormat: "2006-01-02",
		},
		{
			name:         "time only format",
			customFormat: "15:04:05",
		},
		{
			name:         "datetime with space",
			customFormat: "2006-01-02 15:04",
		},
		{
			name:         "custom readable format",
			customFormat: "Mon Jan 2 15:04:05 2006",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logHandler := testutil.NewTestLogger()
			ctx := context.Background()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"format":   tt.customFormat,
						"timezone": "UTC",
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

			if result.IsError {
				t.Errorf("expected successful result for custom format %q, got error: %v", tt.customFormat, result.Content)
			}

			if len(result.Content) == 0 {
				t.Error("expected result content to be non-empty")
			}

			// Verify success was logged
			logHandler.AssertInfoCount(t, 1)
		})
	}
}

func TestHandleAddTimeOffset(t *testing.T) {
	tests := []struct {
		name        string
		hours       float64
		minutes     float64
		format      string
		timezone    string
		shouldError bool
	}{
		{
			name:        "add hours UTC",
			hours:       2,
			minutes:     0,
			format:      "iso8601",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "add minutes",
			hours:       0,
			minutes:     30,
			format:      "unix",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "add both",
			hours:       1,
			minutes:     15,
			format:      "rfc3339",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "subtract hours",
			hours:       -2,
			minutes:     0,
			format:      "iso8601",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "fractional hours",
			hours:       1.5,
			minutes:     0,
			format:      "unixmilli",
			timezone:    "UTC",
			shouldError: false,
		},
		{
			name:        "America/New_York timezone",
			hours:       1,
			minutes:     0,
			format:      "iso8601",
			timezone:    "America/New_York",
			shouldError: false,
		},
		{
			name:        "Asia/Tokyo timezone",
			hours:       2,
			minutes:     30,
			format:      "rfc3339",
			timezone:    "Asia/Tokyo",
			shouldError: false,
		},
		{
			name:        "invalid timezone",
			hours:       1,
			minutes:     0,
			format:      "iso8601",
			timezone:    "Invalid/Timezone",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logHandler := testutil.NewTestLogger()
			ctx := context.Background()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"hours":    tt.hours,
						"minutes":  tt.minutes,
						"format":   tt.format,
						"timezone": tt.timezone,
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

			if tt.shouldError {
				// For errors, we expect an error result
				if !result.IsError {
					t.Error("expected result to be an error")
				}
				// Verify error was logged
				logHandler.AssertErrorCount(t, 1)
			} else {
				if result.IsError {
					t.Errorf("expected successful result, got error: %v", result.Content)
				}

				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}

				// Verify logging
				logHandler.AssertInfoCount(t, 1)
			}
		})
	}
}

func TestHandleAddTimeOffsetDefaults(t *testing.T) {
	logger, logHandler := testutil.NewTestLogger()
	ctx := context.Background()

	// Request with no arguments should use defaults (0 hours, 0 minutes, iso8601, UTC)
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

	// Verify that UTC was logged as the timezone
	logHandler.AssertInfoCount(t, 1)
	if len(logHandler.InfoCalls) > 0 {
		logCall := logHandler.InfoCalls[0]
		foundTimezone := false
		for _, attr := range logCall.Attrs {
			if attr.Key == "timezone" {
				foundTimezone = true
				if attr.Value.String() != "UTC" {
					t.Errorf("expected default timezone to be UTC, got %s", attr.Value.String())
				}
				break
			}
		}
		if !foundTimezone {
			t.Error("expected timezone to be logged")
		}
	}
}

func TestHandleAddTimeOffsetFormats(t *testing.T) {
	formats := []string{"iso8601", "unix", "unixmilli", "rfc3339", "custom"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger()
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

func TestHandleAddTimeOffsetCustomFormats(t *testing.T) {
	tests := []struct {
		name         string
		customFormat string
	}{
		{
			name:         "date only format",
			customFormat: "2006-01-02",
		},
		{
			name:         "time only format",
			customFormat: "15:04:05",
		},
		{
			name:         "datetime with space",
			customFormat: "2006-01-02 15:04",
		},
		{
			name:         "custom readable format",
			customFormat: "Mon Jan 2 15:04:05 2006",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logHandler := testutil.NewTestLogger()
			ctx := context.Background()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"hours":  1,
						"format": tt.customFormat,
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
				t.Errorf("expected successful result for custom format %q, got error: %v", tt.customFormat, result.Content)
			}

			if len(result.Content) == 0 {
				t.Error("expected result content to be non-empty")
			}

			// Verify success was logged
			logHandler.AssertInfoCount(t, 1)
		})
	}
}
