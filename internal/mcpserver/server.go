package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yourorg/timeservice/internal/repository"
	"github.com/yourorg/timeservice/pkg/metrics"
	"github.com/yourorg/timeservice/pkg/version"
)

// NewServer creates and configures a new MCP server with time-related tools
func NewServer(log *slog.Logger, locationRepo repository.LocationRepository) *server.MCPServer {
	// Create server with capabilities and options
	mcpServer := server.NewMCPServer(
		version.ServiceName,
		version.Version,
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Register get_current_time tool
	getCurrentTimeTool := mcp.NewTool("get_current_time",
		mcp.WithDescription("Get the current server time in various formats and timezones"),
		mcp.WithString("format",
			mcp.Description("Time format: iso8601, unix, unixmilli, rfc3339, or custom Go format (e.g., '2006-01-02 15:04')"),
		),
		mcp.WithString("timezone",
			mcp.Description("IANA timezone (e.g., America/New_York, UTC, Europe/London). Defaults to UTC"),
		),
	)

	mcpServer.AddTool(getCurrentTimeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetCurrentTime(ctx, request, log)
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
			mcp.Description("Output format: iso8601, unix, unixmilli, rfc3339, or custom Go format (e.g., '2006-01-02 15:04')"),
		),
		mcp.WithString("timezone",
			mcp.Description("IANA timezone (e.g., America/New_York, UTC, Europe/London). Defaults to UTC"),
		),
	)

	mcpServer.AddTool(addTimeOffsetTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleAddTimeOffset(ctx, request, log)
	})

	// Register location management tools
	addLocationTool := mcp.NewTool("add_location",
		mcp.WithDescription("Add a named location with timezone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name (alphanumeric, hyphens, and underscores only)"),
		),
		mcp.WithString("timezone",
			mcp.Required(),
			mcp.Description("IANA timezone (e.g., America/New_York, Europe/London, Asia/Tokyo)"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description of the location"),
		),
	)

	mcpServer.AddTool(addLocationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleAddLocation(ctx, request, log, locationRepo)
	})

	removeLocationTool := mcp.NewTool("remove_location",
		mcp.WithDescription("Remove a named location"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name to remove"),
		),
	)

	mcpServer.AddTool(removeLocationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRemoveLocation(ctx, request, log, locationRepo)
	})

	updateLocationTool := mcp.NewTool("update_location",
		mcp.WithDescription("Update a location's timezone or description"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name to update"),
		),
		mcp.WithString("timezone",
			mcp.Description("New IANA timezone (optional)"),
		),
		mcp.WithString("description",
			mcp.Description("New description (optional)"),
		),
	)

	mcpServer.AddTool(updateLocationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateLocation(ctx, request, log, locationRepo)
	})

	listLocationsTool := mcp.NewTool("list_locations",
		mcp.WithDescription("List all saved locations"),
	)

	mcpServer.AddTool(listLocationsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListLocations(ctx, request, log, locationRepo)
	})

	getLocationTimeTool := mcp.NewTool("get_location_time",
		mcp.WithDescription("Get the current time for a named location"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name"),
		),
		mcp.WithString("format",
			mcp.Description("Time format: rfc3339, iso8601, unix, unixmilli, or custom Go format (default: rfc3339)"),
		),
	)

	mcpServer.AddTool(getLocationTimeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetLocationTime(ctx, request, log, locationRepo)
	})

	log.Info("MCP server initialized",
		"name", version.ServiceName,
		"version", version.Version,
		"tools", []string{"get_current_time", "add_time_offset", "add_location", "remove_location", "update_location", "list_locations", "get_location_time"},
	)

	return mcpServer
}

// NewServerWithMetrics creates and configures a new MCP server with metrics tracking
func NewServerWithMetrics(log *slog.Logger, m *metrics.Metrics, locationRepo repository.LocationRepository) *server.MCPServer {
	// Create server with capabilities and options
	mcpServer := server.NewMCPServer(
		version.ServiceName,
		version.Version,
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Register get_current_time tool with metrics
	getCurrentTimeTool := mcp.NewTool("get_current_time",
		mcp.WithDescription("Get the current server time in various formats and timezones"),
		mcp.WithString("format",
			mcp.Description("Time format: iso8601, unix, unixmilli, rfc3339, or custom Go format (e.g., '2006-01-02 15:04')"),
		),
		mcp.WithString("timezone",
			mcp.Description("IANA timezone (e.g., America/New_York, UTC, Europe/London). Defaults to UTC"),
		),
	)

	mcpServer.AddTool(getCurrentTimeTool, wrapWithMetrics("get_current_time", m, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetCurrentTime(ctx, request, log)
	}))

	// Register add_time_offset tool with metrics
	addTimeOffsetTool := mcp.NewTool("add_time_offset",
		mcp.WithDescription("Add a time offset (hours and/or minutes) to the current time"),
		mcp.WithNumber("hours",
			mcp.Description("Hours to add (can be negative for subtraction)"),
		),
		mcp.WithNumber("minutes",
			mcp.Description("Minutes to add (can be negative for subtraction)"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: iso8601, unix, unixmilli, rfc3339, or custom Go format (e.g., '2006-01-02 15:04')"),
		),
		mcp.WithString("timezone",
			mcp.Description("IANA timezone (e.g., America/New_York, UTC, Europe/London). Defaults to UTC"),
		),
	)

	mcpServer.AddTool(addTimeOffsetTool, wrapWithMetrics("add_time_offset", m, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleAddTimeOffset(ctx, request, log)
	}))

	// Register add_location tool
	addLocationTool := mcp.NewTool("add_location",
		mcp.WithDescription("Add a named location with timezone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name (alphanumeric, hyphens, and underscores only)"),
		),
		mcp.WithString("timezone",
			mcp.Required(),
			mcp.Description("IANA timezone (e.g., America/New_York, Europe/London, Asia/Tokyo)"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description of the location"),
		),
	)

	mcpServer.AddTool(addLocationTool, wrapWithMetrics("add_location", m, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleAddLocation(ctx, request, log, locationRepo)
	}))

	// Register remove_location tool
	removeLocationTool := mcp.NewTool("remove_location",
		mcp.WithDescription("Remove a named location"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name to remove"),
		),
	)

	mcpServer.AddTool(removeLocationTool, wrapWithMetrics("remove_location", m, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRemoveLocation(ctx, request, log, locationRepo)
	}))

	// Register update_location tool
	updateLocationTool := mcp.NewTool("update_location",
		mcp.WithDescription("Update a location's timezone or description"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name to update"),
		),
		mcp.WithString("timezone",
			mcp.Description("New IANA timezone (optional)"),
		),
		mcp.WithString("description",
			mcp.Description("New description (optional)"),
		),
	)

	mcpServer.AddTool(updateLocationTool, wrapWithMetrics("update_location", m, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateLocation(ctx, request, log, locationRepo)
	}))

	// Register list_locations tool
	listLocationsTool := mcp.NewTool("list_locations",
		mcp.WithDescription("List all saved locations"),
	)

	mcpServer.AddTool(listLocationsTool, wrapWithMetrics("list_locations", m, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListLocations(ctx, request, log, locationRepo)
	}))

	// Register get_location_time tool
	getLocationTimeTool := mcp.NewTool("get_location_time",
		mcp.WithDescription("Get the current time for a named location"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Location name"),
		),
		mcp.WithString("format",
			mcp.Description("Time format: rfc3339, iso8601, unix, unixmilli, or custom Go format (default: rfc3339)"),
		),
	)

	mcpServer.AddTool(getLocationTimeTool, wrapWithMetrics("get_location_time", m, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetLocationTime(ctx, request, log, locationRepo)
	}))

	log.Info("MCP server initialized",
		"name", version.ServiceName,
		"version", version.Version,
		"tools", []string{"get_current_time", "add_time_offset", "add_location", "remove_location", "update_location", "list_locations", "get_location_time"},
	)

	return mcpServer
}

// wrapWithMetrics wraps a tool handler with metrics tracking
func wrapWithMetrics(toolName string, m *metrics.Metrics, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		// Track in-flight tool calls
		m.MCPToolCallsInFlight.Inc()
		defer m.MCPToolCallsInFlight.Dec()

		// Execute the tool handler
		result, err := handler(ctx, request)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := "success"
		if err != nil || (result != nil && result.IsError) {
			status = "error"
		}

		m.MCPToolCallsTotal.WithLabelValues(toolName, status).Inc()
		m.MCPToolCallDuration.WithLabelValues(toolName).Observe(duration)

		return result, err
	}
}

// handleGetCurrentTime handles the get_current_time tool
func handleGetCurrentTime(ctx context.Context, request mcp.CallToolRequest, log *slog.Logger) (*mcp.CallToolResult, error) {
	// Extract arguments using helper methods with defaults
	format := request.GetString("format", "iso8601")
	tzName := request.GetString("timezone", "UTC")

	// Load the timezone
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		log.Error("invalid timezone", "timezone", tzName, "error", err)
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

	log.Info("get_current_time executed",
		"format", format,
		"timezone", tzName,
		"result", result,
	)

	return mcp.NewToolResultText(result), nil
}

// handleAddTimeOffset handles the add_time_offset tool
func handleAddTimeOffset(ctx context.Context, request mcp.CallToolRequest, log *slog.Logger) (*mcp.CallToolResult, error) {
	// Extract arguments - GetFloat returns float64
	hours := request.GetFloat("hours", 0)
	minutes := request.GetFloat("minutes", 0)
	format := request.GetString("format", "iso8601")
	tzName := request.GetString("timezone", "UTC")

	// Load the timezone
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		log.Error("invalid timezone", "timezone", tzName, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Invalid timezone '%s': %v", tzName, err)), nil
	}

	// Calculate the offset duration
	// Multiply in float64 space before converting to Duration to handle fractional hours/minutes
	offset := time.Duration(hours*float64(time.Hour)) + time.Duration(minutes*float64(time.Minute))

	// Get current time in the specified timezone and add offset
	result := time.Now().In(loc).Add(offset)

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

	log.Info("add_time_offset executed",
		"hours", hours,
		"minutes", minutes,
		"format", format,
		"timezone", tzName,
		"result", timeStr,
	)

	return mcp.NewToolResultText(timeStr), nil
}
