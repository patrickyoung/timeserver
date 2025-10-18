package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates and configures a new MCP server with time-related tools
func NewServer(logger *slog.Logger) *server.MCPServer {
	// Create server with capabilities and options
	mcpServer := server.NewMCPServer(
		"timeservice",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Register get_current_time tool
	getCurrentTimeTool := mcp.NewTool("get_current_time",
		mcp.WithDescription("Get the current server time in various formats and timezones"),
		mcp.WithString("format",
			mcp.Description("Time format: iso8601, unix, unixmilli, rfc3339, or custom Go format"),
			mcp.Enum("iso8601", "unix", "unixmilli", "rfc3339"),
		),
		mcp.WithString("timezone",
			mcp.Description("IANA timezone (e.g., America/New_York, UTC, Europe/London). Defaults to UTC"),
		),
	)

	mcpServer.AddTool(getCurrentTimeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetCurrentTime(ctx, request, logger)
	})

	// Register add_time_offset tool
	addTimeOffsetTool := mcp.NewTool("add_time_offset",
		mcp.WithDescription("Add a time offset (hours and/or minutes) to the current time"),
		mcp.WithNumber("hours",
			mcp.Description("Hours to add (can be negative for subtraction)"),
		),
		mcp.WithNumber("minutes",
			mcp.Description("Minutes to add (can be negative for subtraction)"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: iso8601, unix, unixmilli, or rfc3339"),
			mcp.Enum("iso8601", "unix", "unixmilli", "rfc3339"),
		),
	)

	mcpServer.AddTool(addTimeOffsetTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleAddTimeOffset(ctx, request, logger)
	})

	logger.Info("MCP server initialized",
		"name", "timeservice",
		"version", "1.0.0",
		"tools", []string{"get_current_time", "add_time_offset"},
	)

	return mcpServer
}

// handleGetCurrentTime handles the get_current_time tool
func handleGetCurrentTime(ctx context.Context, request mcp.CallToolRequest, logger *slog.Logger) (*mcp.CallToolResult, error) {
	// Extract arguments using helper methods with defaults
	format := request.GetString("format", "iso8601")
	tzName := request.GetString("timezone", "UTC")

	// Load the timezone
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		logger.Error("invalid timezone", "timezone", tzName, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Invalid timezone '%s': %v", tzName, err)), nil
	}

	// Get current time in the specified timezone
	now := time.Now().In(loc)
	var result string

	// Format the time based on the requested format
	switch format {
	case "iso8601", "rfc3339":
		result = now.Format(time.RFC3339)
	case "unix":
		result = fmt.Sprintf("%d", now.Unix())
	case "unixmilli":
		result = fmt.Sprintf("%d", now.UnixMilli())
	default:
		// Treat as custom Go format
		result = now.Format(format)
	}

	logger.Info("get_current_time executed",
		"format", format,
		"timezone", tzName,
		"result", result,
	)

	return mcp.NewToolResultText(result), nil
}

// handleAddTimeOffset handles the add_time_offset tool
func handleAddTimeOffset(ctx context.Context, request mcp.CallToolRequest, logger *slog.Logger) (*mcp.CallToolResult, error) {
	// Extract arguments - GetFloat returns float64
	hours := request.GetFloat("hours", 0)
	minutes := request.GetFloat("minutes", 0)
	format := request.GetString("format", "iso8601")

	// Calculate the offset duration
	offset := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute

	// Get current time and add offset
	result := time.Now().Add(offset)

	// Format the result
	var timeStr string
	switch format {
	case "iso8601", "rfc3339":
		timeStr = result.Format(time.RFC3339)
	case "unix":
		timeStr = fmt.Sprintf("%d", result.Unix())
	case "unixmilli":
		timeStr = fmt.Sprintf("%d", result.UnixMilli())
	default:
		timeStr = result.Format(format)
	}

	logger.Info("add_time_offset executed",
		"hours", hours,
		"minutes", minutes,
		"format", format,
		"result", timeStr,
	)

	return mcp.NewToolResultText(timeStr), nil
}
