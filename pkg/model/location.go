package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Location represents a named location with a timezone
type Location struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Timezone    string    `json:"timezone"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateLocationRequest represents the request body for creating a location
type CreateLocationRequest struct {
	Name        string `json:"name"`
	Timezone    string `json:"timezone"`
	Description string `json:"description,omitempty"`
}

// UpdateLocationRequest represents the request body for updating a location
type UpdateLocationRequest struct {
	Timezone    string `json:"timezone,omitempty"`
	Description string `json:"description,omitempty"`
}

// LocationResponse represents a single location response
type LocationResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Timezone    string    `json:"timezone"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LocationListResponse represents a list of locations
type LocationListResponse struct {
	Locations []*LocationResponse `json:"locations"`
}

// LocationTimeResponse represents the current time for a location
type LocationTimeResponse struct {
	Location    string    `json:"location"`
	Timezone    string    `json:"timezone"`
	CurrentTime time.Time `json:"current_time"`
	UnixTime    int64     `json:"unix_time"`
	Formatted   string    `json:"formatted"`
}

// Validation errors
var (
	ErrEmptyName           = errors.New("location name cannot be empty")
	ErrNameTooLong         = errors.New("location name must be 100 characters or less")
	ErrInvalidNameFormat   = errors.New("location name must contain only alphanumeric characters, hyphens, and underscores")
	ErrEmptyTimezone       = errors.New("timezone cannot be empty")
	ErrInvalidTimezone     = errors.New("invalid IANA timezone")
	ErrDescriptionTooLong  = errors.New("description must be 500 characters or less")
)

// Regular expression for valid location names (alphanumeric, hyphens, underscores)
var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// NewLocation creates a new Location with the current timestamp
func NewLocation(name, timezone, description string) *Location {
	now := time.Now().UTC()
	return &Location{
		Name:        strings.ToLower(strings.TrimSpace(name)),
		Timezone:    strings.TrimSpace(timezone),
		Description: strings.TrimSpace(description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate validates all fields of a Location
func (l *Location) Validate() error {
	if err := ValidateName(l.Name); err != nil {
		return err
	}
	if err := ValidateTimezone(l.Timezone); err != nil {
		return err
	}
	if err := ValidateDescription(l.Description); err != nil {
		return err
	}
	return nil
}

// ValidateName validates a location name
func ValidateName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return ErrEmptyName
	}

	if len(name) > 100 {
		return ErrNameTooLong
	}

	if !nameRegex.MatchString(name) {
		return ErrInvalidNameFormat
	}

	return nil
}

// ValidateTimezone validates an IANA timezone string
func ValidateTimezone(timezone string) error {
	timezone = strings.TrimSpace(timezone)

	if timezone == "" {
		return ErrEmptyTimezone
	}

	// Try to load the timezone to verify it's valid
	_, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidTimezone, timezone)
	}

	return nil
}

// ValidateDescription validates a location description
func ValidateDescription(description string) error {
	description = strings.TrimSpace(description)

	if len(description) > 500 {
		return ErrDescriptionTooLong
	}

	return nil
}

// Validate validates a CreateLocationRequest
func (r *CreateLocationRequest) Validate() error {
	if err := ValidateName(r.Name); err != nil {
		return err
	}
	if err := ValidateTimezone(r.Timezone); err != nil {
		return err
	}
	if err := ValidateDescription(r.Description); err != nil {
		return err
	}
	return nil
}

// Normalize normalizes the fields of a CreateLocationRequest
func (r *CreateLocationRequest) Normalize() {
	r.Name = strings.ToLower(strings.TrimSpace(r.Name))
	r.Timezone = strings.TrimSpace(r.Timezone)
	r.Description = strings.TrimSpace(r.Description)
}

// Validate validates an UpdateLocationRequest
func (r *UpdateLocationRequest) Validate() error {
	// At least one field must be provided
	if r.Timezone == "" && r.Description == "" {
		return errors.New("at least one field must be provided for update")
	}

	// Validate timezone if provided
	if r.Timezone != "" {
		if err := ValidateTimezone(r.Timezone); err != nil {
			return err
		}
	}

	// Validate description
	if err := ValidateDescription(r.Description); err != nil {
		return err
	}

	return nil
}

// Normalize normalizes the fields of an UpdateLocationRequest
func (r *UpdateLocationRequest) Normalize() {
	r.Timezone = strings.TrimSpace(r.Timezone)
	r.Description = strings.TrimSpace(r.Description)
}

// ToResponse converts a Location to a LocationResponse
func (l *Location) ToResponse() *LocationResponse {
	return &LocationResponse{
		ID:          l.ID,
		Name:        l.Name,
		Timezone:    l.Timezone,
		Description: l.Description,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}

// ToLocationListResponse converts a slice of Locations to a LocationListResponse
func ToLocationListResponse(locations []*Location) *LocationListResponse {
	responses := make([]*LocationResponse, len(locations))
	for i, loc := range locations {
		responses[i] = loc.ToResponse()
	}
	return &LocationListResponse{
		Locations: responses,
	}
}
