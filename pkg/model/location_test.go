package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError error
	}{
		{"valid simple name", "headquarters", nil},
		{"valid with hyphens", "us-east-1", nil},
		{"valid with underscores", "tokyo_office", nil},
		{"valid mixed", "office-01_main", nil},
		{"valid uppercase", "NYC", nil},
		{"empty name", "", ErrEmptyName},
		{"whitespace only", "   ", ErrEmptyName},
		{"too long", strings.Repeat("a", 101), ErrNameTooLong},
		{"invalid spaces", "new york", ErrInvalidNameFormat},
		{"invalid special chars", "office@work", ErrInvalidNameFormat},
		{"invalid dots", "office.main", ErrInvalidNameFormat},
		{"single char", "a", nil},
		{"exactly 100 chars", strings.Repeat("a", 100), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if err != tt.wantError {
				t.Errorf("ValidateName(%q) = %v, want %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid UTC", "UTC", false},
		{"valid America/New_York", "America/New_York", false},
		{"valid Europe/London", "Europe/London", false},
		{"valid Asia/Tokyo", "Asia/Tokyo", false},
		{"valid with whitespace", "  America/Chicago  ", false},
		{"empty timezone", "", true},
		{"whitespace only", "   ", true},
		{"invalid timezone", "Invalid/Zone", true},
		{"invalid format", "New York", true},
		{"partially valid", "America/Invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimezone(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTimezone(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError error
	}{
		{"valid empty", "", nil},
		{"valid short", "Company HQ", nil},
		{"valid long", strings.Repeat("a", 500), nil},
		{"too long", strings.Repeat("a", 501), ErrDescriptionTooLong},
		{"with whitespace", "  Valid description  ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.input)
			if err != tt.wantError {
				t.Errorf("ValidateDescription() = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestNewLocation(t *testing.T) {
	name := "HeadQuarters"
	timezone := "America/New_York"
	description := "  Company HQ in NYC  "

	before := time.Now().UTC()
	loc := NewLocation(name, timezone, description)
	after := time.Now().UTC()

	// Check normalization
	if loc.Name != "headquarters" {
		t.Errorf("Name not normalized: got %q, want %q", loc.Name, "headquarters")
	}

	if loc.Timezone != "America/New_York" {
		t.Errorf("Timezone = %q, want %q", loc.Timezone, "America/New_York")
	}

	if loc.Description != "Company HQ in NYC" {
		t.Errorf("Description not trimmed: got %q, want %q", loc.Description, "Company HQ in NYC")
	}

	// Check timestamps
	if loc.CreatedAt.Before(before) || loc.CreatedAt.After(after) {
		t.Errorf("CreatedAt not in expected range")
	}

	if loc.UpdatedAt.Before(before) || loc.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt not in expected range")
	}

	if !loc.CreatedAt.Equal(loc.UpdatedAt) {
		t.Errorf("CreatedAt and UpdatedAt should be equal for new location")
	}

	// ID should be 0 (not set)
	if loc.ID != 0 {
		t.Errorf("ID should be 0 for new location, got %d", loc.ID)
	}
}

func TestLocation_Validate(t *testing.T) {
	tests := []struct {
		name      string
		location  *Location
		wantError bool
	}{
		{
			name: "valid location",
			location: &Location{
				Name:        "headquarters",
				Timezone:    "America/New_York",
				Description: "Company HQ",
			},
			wantError: false,
		},
		{
			name: "invalid name",
			location: &Location{
				Name:     "",
				Timezone: "UTC",
			},
			wantError: true,
		},
		{
			name: "invalid timezone",
			location: &Location{
				Name:     "office",
				Timezone: "Invalid/Zone",
			},
			wantError: true,
		},
		{
			name: "description too long",
			location: &Location{
				Name:        "office",
				Timezone:    "UTC",
				Description: strings.Repeat("a", 501),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.location.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Location.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCreateLocationRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   *CreateLocationRequest
		wantError bool
	}{
		{
			name: "valid request",
			request: &CreateLocationRequest{
				Name:        "headquarters",
				Timezone:    "America/New_York",
				Description: "Company HQ",
			},
			wantError: false,
		},
		{
			name: "valid without description",
			request: &CreateLocationRequest{
				Name:     "office",
				Timezone: "UTC",
			},
			wantError: false,
		},
		{
			name: "empty name",
			request: &CreateLocationRequest{
				Name:     "",
				Timezone: "UTC",
			},
			wantError: true,
		},
		{
			name: "invalid timezone",
			request: &CreateLocationRequest{
				Name:     "office",
				Timezone: "Invalid",
			},
			wantError: true,
		},
		{
			name: "description too long",
			request: &CreateLocationRequest{
				Name:        "office",
				Timezone:    "UTC",
				Description: strings.Repeat("a", 501),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("CreateLocationRequest.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCreateLocationRequest_Normalize(t *testing.T) {
	req := &CreateLocationRequest{
		Name:        "  HeadQuarters  ",
		Timezone:    "  America/New_York  ",
		Description: "  Company HQ  ",
	}

	req.Normalize()

	if req.Name != "headquarters" {
		t.Errorf("Name = %q, want %q", req.Name, "headquarters")
	}

	if req.Timezone != "America/New_York" {
		t.Errorf("Timezone = %q, want %q", req.Timezone, "America/New_York")
	}

	if req.Description != "Company HQ" {
		t.Errorf("Description = %q, want %q", req.Description, "Company HQ")
	}
}

func TestUpdateLocationRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   *UpdateLocationRequest
		wantError bool
	}{
		{
			name: "valid with timezone",
			request: &UpdateLocationRequest{
				Timezone: "Europe/London",
			},
			wantError: false,
		},
		{
			name: "valid with description",
			request: &UpdateLocationRequest{
				Description: "Updated description",
			},
			wantError: false,
		},
		{
			name: "valid with both",
			request: &UpdateLocationRequest{
				Timezone:    "Asia/Tokyo",
				Description: "Tokyo office",
			},
			wantError: false,
		},
		{
			name:      "empty request",
			request:   &UpdateLocationRequest{},
			wantError: true,
		},
		{
			name: "invalid timezone",
			request: &UpdateLocationRequest{
				Timezone: "Invalid/Zone",
			},
			wantError: true,
		},
		{
			name: "description too long",
			request: &UpdateLocationRequest{
				Description: strings.Repeat("a", 501),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("UpdateLocationRequest.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestUpdateLocationRequest_Normalize(t *testing.T) {
	req := &UpdateLocationRequest{
		Timezone:    "  Europe/London  ",
		Description: "  London office  ",
	}

	req.Normalize()

	if req.Timezone != "Europe/London" {
		t.Errorf("Timezone = %q, want %q", req.Timezone, "Europe/London")
	}

	if req.Description != "London office" {
		t.Errorf("Description = %q, want %q", req.Description, "London office")
	}
}

func TestLocation_ToResponse(t *testing.T) {
	now := time.Now().UTC()
	loc := &Location{
		ID:          1,
		Name:        "headquarters",
		Timezone:    "America/New_York",
		Description: "Company HQ",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := loc.ToResponse()

	if resp.ID != loc.ID {
		t.Errorf("ID = %d, want %d", resp.ID, loc.ID)
	}
	if resp.Name != loc.Name {
		t.Errorf("Name = %q, want %q", resp.Name, loc.Name)
	}
	if resp.Timezone != loc.Timezone {
		t.Errorf("Timezone = %q, want %q", resp.Timezone, loc.Timezone)
	}
	if resp.Description != loc.Description {
		t.Errorf("Description = %q, want %q", resp.Description, loc.Description)
	}
	if !resp.CreatedAt.Equal(loc.CreatedAt) {
		t.Errorf("CreatedAt mismatch")
	}
	if !resp.UpdatedAt.Equal(loc.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch")
	}
}

func TestToLocationListResponse(t *testing.T) {
	now := time.Now().UTC()
	locations := []*Location{
		{
			ID:        1,
			Name:      "headquarters",
			Timezone:  "America/New_York",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        2,
			Name:      "tokyo-office",
			Timezone:  "Asia/Tokyo",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	resp := ToLocationListResponse(locations)

	if len(resp.Locations) != len(locations) {
		t.Errorf("len(Locations) = %d, want %d", len(resp.Locations), len(locations))
	}

	for i, locResp := range resp.Locations {
		if locResp.ID != locations[i].ID {
			t.Errorf("Location[%d].ID = %d, want %d", i, locResp.ID, locations[i].ID)
		}
		if locResp.Name != locations[i].Name {
			t.Errorf("Location[%d].Name = %q, want %q", i, locResp.Name, locations[i].Name)
		}
	}
}

func TestToLocationListResponse_Empty(t *testing.T) {
	resp := ToLocationListResponse([]*Location{})

	if len(resp.Locations) != 0 {
		t.Errorf("Expected empty list, got %d items", len(resp.Locations))
	}
}

func TestLocation_JSON(t *testing.T) {
	now := time.Date(2025, 10, 19, 12, 0, 0, 0, time.UTC)
	loc := &Location{
		ID:          1,
		Name:        "headquarters",
		Timezone:    "America/New_York",
		Description: "Company HQ",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Marshal
	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded Location
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Compare
	if decoded.ID != loc.ID {
		t.Errorf("ID = %d, want %d", decoded.ID, loc.ID)
	}
	if decoded.Name != loc.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, loc.Name)
	}
	if decoded.Timezone != loc.Timezone {
		t.Errorf("Timezone = %q, want %q", decoded.Timezone, loc.Timezone)
	}
	if decoded.Description != loc.Description {
		t.Errorf("Description = %q, want %q", decoded.Description, loc.Description)
	}
}

func TestLocationResponse_JSON_OmitEmpty(t *testing.T) {
	resp := &LocationResponse{
		ID:        1,
		Name:      "office",
		Timezone:  "UTC",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		// Description intentionally omitted
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Check that description field is not in JSON when empty
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := decoded["description"]; exists {
		t.Error("Expected 'description' to be omitted when empty")
	}
}
