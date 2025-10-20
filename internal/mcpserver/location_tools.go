package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yourorg/timeservice/internal/repository"
	"github.com/yourorg/timeservice/pkg/model"
)

// handleAddLocation handles the add_location tool
func handleAddLocation(ctx context.Context, request mcp.CallToolRequest, log *slog.Logger, repo repository.LocationRepository) (*mcp.CallToolResult, error) {
	// Extract arguments with validation
	name := request.GetString("name", "")
	if name == "" {
		log.Warn("add_location: missing required parameter", "parameter", "name")
		return mcp.NewToolResultError("Parameter 'name' is required"), nil
	}

	timezone := request.GetString("timezone", "")
	if timezone == "" {
		log.Warn("add_location: missing required parameter", "parameter", "timezone")
		return mcp.NewToolResultError("Parameter 'timezone' is required"), nil
	}

	description := request.GetString("description", "")

	// Create location model
	loc := model.NewLocation(name, timezone, description)

	// Validate
	if err := loc.Validate(); err != nil {
		log.Warn("add_location: validation failed",
			"name", name,
			"timezone", timezone,
			"error", err,
		)
		return mcp.NewToolResultError(fmt.Sprintf("Validation failed: %v", err)), nil
	}

	// Create in repository
	if err := repo.Create(ctx, loc); err != nil {
		if errors.Is(err, repository.ErrLocationExists) {
			log.Warn("add_location: location already exists", "name", name)
			return mcp.NewToolResultError(fmt.Sprintf("Location '%s' already exists", name)), nil
		}
		log.Error("add_location: failed to create location",
			"name", name,
			"error", err,
		)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add location: %v", err)), nil
	}

	log.Info("add_location executed",
		"name", loc.Name,
		"timezone", loc.Timezone,
		"id", loc.ID,
	)

	// Format response
	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Location '%s' added successfully", loc.Name),
		"location": map[string]interface{}{
			"id":          loc.ID,
			"name":        loc.Name,
			"timezone":    loc.Timezone,
			"description": loc.Description,
			"created_at":  loc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":  loc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Error("add_location: failed to marshal response", "error", err)
		return mcp.NewToolResultError("Failed to format response"), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// handleRemoveLocation handles the remove_location tool
func handleRemoveLocation(ctx context.Context, request mcp.CallToolRequest, log *slog.Logger, repo repository.LocationRepository) (*mcp.CallToolResult, error) {
	// Extract arguments
	name := request.GetString("name", "")
	if name == "" {
		log.Warn("remove_location: missing required parameter", "parameter", "name")
		return mcp.NewToolResultError("Parameter 'name' is required"), nil
	}

	// Delete from repository
	if err := repo.Delete(ctx, name); err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			log.Warn("remove_location: location not found", "name", name)
			return mcp.NewToolResultError(fmt.Sprintf("Location '%s' not found", name)), nil
		}
		log.Error("remove_location: failed to delete location",
			"name", name,
			"error", err,
		)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove location: %v", err)), nil
	}

	log.Info("remove_location executed", "name", name)

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Location '%s' removed successfully", name),
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Error("remove_location: failed to marshal response", "error", err)
		return mcp.NewToolResultError("Failed to format response"), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// handleUpdateLocation handles the update_location tool
func handleUpdateLocation(ctx context.Context, request mcp.CallToolRequest, log *slog.Logger, repo repository.LocationRepository) (*mcp.CallToolResult, error) {
	// Extract arguments
	name := request.GetString("name", "")
	if name == "" {
		log.Warn("update_location: missing required parameter", "parameter", "name")
		return mcp.NewToolResultError("Parameter 'name' is required"), nil
	}

	timezone := request.GetString("timezone", "")
	description := request.GetString("description", "")

	// At least one field must be provided
	if timezone == "" && description == "" {
		log.Warn("update_location: no fields to update", "name", name)
		return mcp.NewToolResultError("At least one of 'timezone' or 'description' must be provided"), nil
	}

	// Get existing location
	existing, err := repo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			log.Warn("update_location: location not found", "name", name)
			return mcp.NewToolResultError(fmt.Sprintf("Location '%s' not found", name)), nil
		}
		log.Error("update_location: failed to get location",
			"name", name,
			"error", err,
		)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get location: %v", err)), nil
	}

	// Update fields
	if timezone != "" {
		// Validate timezone
		if err := model.ValidateTimezone(timezone); err != nil {
			log.Warn("update_location: invalid timezone",
				"name", name,
				"timezone", timezone,
				"error", err,
			)
			return mcp.NewToolResultError(fmt.Sprintf("Invalid timezone: %v", err)), nil
		}
		existing.Timezone = timezone
	}
	existing.Description = description

	// Update in repository
	if err := repo.Update(ctx, name, existing); err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			log.Warn("update_location: location not found", "name", name)
			return mcp.NewToolResultError(fmt.Sprintf("Location '%s' not found", name)), nil
		}
		log.Error("update_location: failed to update location",
			"name", name,
			"error", err,
		)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update location: %v", err)), nil
	}

	log.Info("update_location executed",
		"name", name,
		"timezone", existing.Timezone,
	)

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Location '%s' updated successfully", name),
		"location": map[string]interface{}{
			"id":          existing.ID,
			"name":        existing.Name,
			"timezone":    existing.Timezone,
			"description": existing.Description,
			"created_at":  existing.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":  existing.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Error("update_location: failed to marshal response", "error", err)
		return mcp.NewToolResultError("Failed to format response"), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// handleListLocations handles the list_locations tool
func handleListLocations(ctx context.Context, request mcp.CallToolRequest, log *slog.Logger, repo repository.LocationRepository) (*mcp.CallToolResult, error) {
	// List all locations
	locations, err := repo.List(ctx)
	if err != nil {
		log.Error("list_locations: failed to list locations", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list locations: %v", err)), nil
	}

	log.Info("list_locations executed", "count", len(locations))

	// Format response
	locationList := make([]map[string]interface{}, len(locations))
	for i, loc := range locations {
		locationList[i] = map[string]interface{}{
			"id":          loc.ID,
			"name":        loc.Name,
			"timezone":    loc.Timezone,
			"description": loc.Description,
			"created_at":  loc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":  loc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	response := map[string]interface{}{
		"success":   true,
		"count":     len(locations),
		"locations": locationList,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Error("list_locations: failed to marshal response", "error", err)
		return mcp.NewToolResultError("Failed to format response"), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// handleGetLocationTime handles the get_location_time tool
func handleGetLocationTime(ctx context.Context, request mcp.CallToolRequest, log *slog.Logger, repo repository.LocationRepository) (*mcp.CallToolResult, error) {
	// Extract arguments
	name := request.GetString("name", "")
	if name == "" {
		log.Warn("get_location_time: missing required parameter", "parameter", "name")
		return mcp.NewToolResultError("Parameter 'name' is required"), nil
	}

	format := request.GetString("format", "rfc3339")

	// Get location from repository
	loc, err := repo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			log.Warn("get_location_time: location not found", "name", name)
			return mcp.NewToolResultError(fmt.Sprintf("Location '%s' not found", name)), nil
		}
		log.Error("get_location_time: failed to get location",
			"name", name,
			"error", err,
		)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get location: %v", err)), nil
	}

	// Load timezone
	tz, err := time.LoadLocation(loc.Timezone)
	if err != nil {
		log.Error("get_location_time: failed to load timezone",
			"name", name,
			"timezone", loc.Timezone,
			"error", err,
		)
		return mcp.NewToolResultError(fmt.Sprintf("Invalid timezone '%s': %v", loc.Timezone, err)), nil
	}

	// Get current time in location's timezone
	now := time.Now().In(tz)

	// Format the time
	var formatted string
	switch format {
	case "rfc3339", "iso8601":
		formatted = now.Format("2006-01-02T15:04:05Z07:00")
	case "unix":
		formatted = fmt.Sprintf("%d", now.Unix())
	case "unixmilli":
		formatted = fmt.Sprintf("%d", now.UnixMilli())
	default:
		// Custom format
		formatted = now.Format(format)
	}

	log.Info("get_location_time executed",
		"name", name,
		"timezone", loc.Timezone,
		"format", format,
		"time", formatted,
	)

	response := map[string]interface{}{
		"success":      true,
		"location":     loc.Name,
		"timezone":     loc.Timezone,
		"current_time": now.Format("2006-01-02T15:04:05Z07:00"),
		"unix_time":    now.Unix(),
		"formatted":    formatted,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Error("get_location_time: failed to marshal response", "error", err)
		return mcp.NewToolResultError("Failed to format response"), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}
