package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourorg/timeservice/internal/repository"
	"github.com/yourorg/timeservice/pkg/model"
)

// LocationHandler handles location-related HTTP requests
type LocationHandler struct {
	repo   repository.LocationRepository
	logger *slog.Logger
}

// NewLocationHandler creates a new location handler
func NewLocationHandler(repo repository.LocationRepository, logger *slog.Logger) *LocationHandler {
	return &LocationHandler{
		repo:   repo,
		logger: logger,
	}
}

// CreateLocation handles POST /api/locations
func (h *LocationHandler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var req model.CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		h.errorJSON(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Normalize and validate request
	req.Normalize()
	if err := req.Validate(); err != nil {
		h.logger.Warn("validation failed", "error", err)
		h.errorJSON(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create location model
	loc := model.NewLocation(req.Name, req.Timezone, req.Description)

	// Create in repository
	if err := h.repo.Create(r.Context(), loc); err != nil {
		if errors.Is(err, repository.ErrLocationExists) {
			h.logger.Warn("location already exists", "name", req.Name)
			h.errorJSON(w, "Location already exists", http.StatusConflict)
			return
		}
		h.logger.Error("failed to create location", "error", err)
		h.errorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("location created",
		"name", loc.Name,
		"timezone", loc.Timezone,
		"id", loc.ID,
	)

	h.json(w, loc.ToResponse(), http.StatusCreated)
}

// GetLocation handles GET /api/locations/{name}
func (h *LocationHandler) GetLocation(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.errorJSON(w, "Location name is required", http.StatusBadRequest)
		return
	}

	loc, err := h.repo.GetByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			h.logger.Debug("location not found", "name", name)
			h.errorJSON(w, "Location not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get location", "error", err, "name", name)
		h.errorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("location retrieved", "name", name)
	h.json(w, loc.ToResponse(), http.StatusOK)
}

// UpdateLocation handles PUT /api/locations/{name}
func (h *LocationHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.errorJSON(w, "Location name is required", http.StatusBadRequest)
		return
	}

	var req model.UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		h.errorJSON(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Normalize and validate request
	req.Normalize()
	if err := req.Validate(); err != nil {
		h.logger.Warn("validation failed", "error", err)
		h.errorJSON(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing location first
	existing, err := h.repo.GetByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			h.logger.Debug("location not found", "name", name)
			h.errorJSON(w, "Location not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get location", "error", err, "name", name)
		h.errorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update only provided fields
	if req.Timezone != "" {
		existing.Timezone = req.Timezone
	}
	// Always update description (even if empty string to allow clearing)
	existing.Description = req.Description
	existing.UpdatedAt = time.Now().UTC()

	// Update in repository
	if err := h.repo.Update(r.Context(), name, existing); err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			h.logger.Debug("location not found", "name", name)
			h.errorJSON(w, "Location not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to update location", "error", err, "name", name)
		h.errorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("location updated",
		"name", name,
		"timezone", existing.Timezone,
	)

	h.json(w, existing.ToResponse(), http.StatusOK)
}

// DeleteLocation handles DELETE /api/locations/{name}
func (h *LocationHandler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.errorJSON(w, "Location name is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), name); err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			h.logger.Debug("location not found", "name", name)
			h.errorJSON(w, "Location not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete location", "error", err, "name", name)
		h.errorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("location deleted", "name", name)
	w.WriteHeader(http.StatusNoContent)
}

// ListLocations handles GET /api/locations
func (h *LocationHandler) ListLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := h.repo.List(r.Context())
	if err != nil {
		h.logger.Error("failed to list locations", "error", err)
		h.errorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("locations listed", "count", len(locations))
	h.json(w, model.ToLocationListResponse(locations), http.StatusOK)
}

// GetLocationTime handles GET /api/locations/{name}/time
func (h *LocationHandler) GetLocationTime(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.errorJSON(w, "Location name is required", http.StatusBadRequest)
		return
	}

	loc, err := h.repo.GetByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			h.logger.Debug("location not found", "name", name)
			h.errorJSON(w, "Location not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get location", "error", err, "name", name)
		h.errorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Load the timezone
	tz, err := time.LoadLocation(loc.Timezone)
	if err != nil {
		h.logger.Error("failed to load timezone", "error", err, "timezone", loc.Timezone)
		h.errorJSON(w, "Invalid timezone", http.StatusInternalServerError)
		return
	}

	// Get current time in the location's timezone
	now := time.Now().In(tz)

	response := &model.LocationTimeResponse{
		Location:    loc.Name,
		Timezone:    loc.Timezone,
		CurrentTime: now,
		UnixTime:    now.Unix(),
		Formatted:   now.Format(time.RFC3339),
	}

	h.logger.Debug("location time retrieved",
		"name", name,
		"timezone", loc.Timezone,
		"time", response.Formatted,
	)

	h.json(w, response, http.StatusOK)
}

// json sends a JSON response
func (h *LocationHandler) json(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("json encode error", "error", err)
	}
}

// errorJSON sends an error JSON response
func (h *LocationHandler) errorJSON(w http.ResponseWriter, message string, status int) {
	h.json(w, map[string]string{"error": message}, status)
}
