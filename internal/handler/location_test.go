package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourorg/timeservice/internal/repository"
	"github.com/yourorg/timeservice/pkg/model"
)

// mockLocationRepository is a mock implementation of LocationRepository for testing
type mockLocationRepository struct {
	createFunc    func(ctx context.Context, loc *model.Location) error
	getByNameFunc func(ctx context.Context, name string) (*model.Location, error)
	updateFunc    func(ctx context.Context, name string, loc *model.Location) error
	deleteFunc    func(ctx context.Context, name string) error
	listFunc      func(ctx context.Context) ([]*model.Location, error)
}

func (m *mockLocationRepository) Create(ctx context.Context, loc *model.Location) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, loc)
	}
	return nil
}

func (m *mockLocationRepository) GetByName(ctx context.Context, name string) (*model.Location, error) {
	if m.getByNameFunc != nil {
		return m.getByNameFunc(ctx, name)
	}
	return nil, repository.ErrLocationNotFound
}

func (m *mockLocationRepository) Update(ctx context.Context, name string, loc *model.Location) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, name, loc)
	}
	return nil
}

func (m *mockLocationRepository) Delete(ctx context.Context, name string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, name)
	}
	return nil
}

func (m *mockLocationRepository) List(ctx context.Context) ([]*model.Location, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []*model.Location{}, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestCreateLocation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockCreateFunc func(ctx context.Context, loc *model.Location) error
		expectedStatus int
		expectedError  string
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful creation",
			requestBody: model.CreateLocationRequest{
				Name:        "hq",
				Timezone:    "America/New_York",
				Description: "Headquarters",
			},
			mockCreateFunc: func(ctx context.Context, loc *model.Location) error {
				loc.ID = 1
				return nil
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp model.LocationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 1 {
					t.Errorf("expected ID 1, got %d", resp.ID)
				}
				if resp.Name != "hq" {
					t.Errorf("expected name 'hq', got %s", resp.Name)
				}
			},
		},
		{
			name: "invalid JSON body",
			requestBody: "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError: "Invalid request body",
		},
		{
			name: "empty name validation",
			requestBody: model.CreateLocationRequest{
				Name:     "",
				Timezone: "America/New_York",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: "location name cannot be empty",
		},
		{
			name: "invalid timezone",
			requestBody: model.CreateLocationRequest{
				Name:     "test",
				Timezone: "Invalid/Timezone",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate location",
			requestBody: model.CreateLocationRequest{
				Name:     "existing",
				Timezone: "America/New_York",
			},
			mockCreateFunc: func(ctx context.Context, loc *model.Location) error {
				return repository.ErrLocationExists
			},
			expectedStatus: http.StatusConflict,
			expectedError: "Location already exists",
		},
		{
			name: "repository error",
			requestBody: model.CreateLocationRequest{
				Name:     "test",
				Timezone: "America/New_York",
			},
			mockCreateFunc: func(ctx context.Context, loc *model.Location) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockLocationRepository{
				createFunc: tt.mockCreateFunc,
			}
			handler := NewLocationHandler(mockRepo, newTestLogger())

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/locations", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.CreateLocation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp["error"] != tt.expectedError {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, errResp["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestGetLocation(t *testing.T) {
	tests := []struct {
		name              string
		pathName          string
		mockGetByNameFunc func(ctx context.Context, name string) (*model.Location, error)
		expectedStatus    int
		expectedError     string
		checkResponse     func(t *testing.T, body []byte)
	}{
		{
			name:     "successful get",
			pathName: "hq",
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				return &model.Location{
					ID:          1,
					Name:        "hq",
					Timezone:    "America/New_York",
					Description: "Headquarters",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp model.LocationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Name != "hq" {
					t.Errorf("expected name 'hq', got %s", resp.Name)
				}
			},
		},
		{
			name:     "location not found",
			pathName: "nonexistent",
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				return nil, repository.ErrLocationNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Location not found",
		},
		{
			name:     "repository error",
			pathName: "test",
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockLocationRepository{
				getByNameFunc: tt.mockGetByNameFunc,
			}
			handler := NewLocationHandler(mockRepo, newTestLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/locations/"+tt.pathName, nil)
			req.SetPathValue("name", tt.pathName)
			w := httptest.NewRecorder()

			handler.GetLocation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp["error"] != tt.expectedError {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, errResp["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestUpdateLocation(t *testing.T) {
	existingLocation := &model.Location{
		ID:          1,
		Name:        "hq",
		Timezone:    "America/New_York",
		Description: "Old description",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name              string
		pathName          string
		requestBody       interface{}
		mockGetByNameFunc func(ctx context.Context, name string) (*model.Location, error)
		mockUpdateFunc    func(ctx context.Context, name string, loc *model.Location) error
		expectedStatus    int
		expectedError     string
		checkResponse     func(t *testing.T, body []byte)
	}{
		{
			name:     "successful update",
			pathName: "hq",
			requestBody: model.UpdateLocationRequest{
				Timezone:    "America/Los_Angeles",
				Description: "New description",
			},
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				// Return a copy of existing location
				loc := *existingLocation
				return &loc, nil
			},
			mockUpdateFunc: func(ctx context.Context, name string, loc *model.Location) error {
				return nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp model.LocationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Timezone != "America/Los_Angeles" {
					t.Errorf("expected timezone 'America/Los_Angeles', got %s", resp.Timezone)
				}
				if resp.Description != "New description" {
					t.Errorf("expected description 'New description', got %s", resp.Description)
				}
			},
		},
		{
			name:     "invalid JSON body",
			pathName: "hq",
			requestBody: "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError: "Invalid request body",
		},
		{
			name:     "empty update request",
			pathName: "hq",
			requestBody: model.UpdateLocationRequest{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "location not found on get",
			pathName: "nonexistent",
			requestBody: model.UpdateLocationRequest{
				Timezone: "America/Chicago",
			},
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				return nil, repository.ErrLocationNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Location not found",
		},
		{
			name:     "invalid timezone in update",
			pathName: "hq",
			requestBody: model.UpdateLocationRequest{
				Timezone: "Invalid/Timezone",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockLocationRepository{
				getByNameFunc: tt.mockGetByNameFunc,
				updateFunc:    tt.mockUpdateFunc,
			}
			handler := NewLocationHandler(mockRepo, newTestLogger())

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPut, "/api/locations/"+tt.pathName, bytes.NewReader(body))
			req.SetPathValue("name", tt.pathName)
			w := httptest.NewRecorder()

			handler.UpdateLocation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp["error"] != tt.expectedError {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, errResp["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestDeleteLocation(t *testing.T) {
	tests := []struct {
		name           string
		pathName       string
		mockDeleteFunc func(ctx context.Context, name string) error
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "successful delete",
			pathName: "hq",
			mockDeleteFunc: func(ctx context.Context, name string) error {
				return nil
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:     "location not found",
			pathName: "nonexistent",
			mockDeleteFunc: func(ctx context.Context, name string) error {
				return repository.ErrLocationNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Location not found",
		},
		{
			name:     "repository error",
			pathName: "test",
			mockDeleteFunc: func(ctx context.Context, name string) error {
				return errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockLocationRepository{
				deleteFunc: tt.mockDeleteFunc,
			}
			handler := NewLocationHandler(mockRepo, newTestLogger())

			req := httptest.NewRequest(http.MethodDelete, "/api/locations/"+tt.pathName, nil)
			req.SetPathValue("name", tt.pathName)
			w := httptest.NewRecorder()

			handler.DeleteLocation(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp["error"] != tt.expectedError {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, errResp["error"])
				}
			}

			// For successful delete, check that response body is empty
			if tt.expectedStatus == http.StatusNoContent {
				if w.Body.Len() != 0 {
					t.Errorf("expected empty body, got %d bytes", w.Body.Len())
				}
			}
		})
	}
}

func TestListLocations(t *testing.T) {
	tests := []struct {
		name           string
		mockListFunc   func(ctx context.Context) ([]*model.Location, error)
		expectedStatus int
		expectedError  string
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful list with locations",
			mockListFunc: func(ctx context.Context) ([]*model.Location, error) {
				return []*model.Location{
					{
						ID:          1,
						Name:        "hq",
						Timezone:    "America/New_York",
						Description: "Headquarters",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
					{
						ID:          2,
						Name:        "west-office",
						Timezone:    "America/Los_Angeles",
						Description: "West Coast Office",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp model.LocationListResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Locations) != 2 {
					t.Errorf("expected 2 locations, got %d", len(resp.Locations))
				}
			},
		},
		{
			name: "empty list",
			mockListFunc: func(ctx context.Context) ([]*model.Location, error) {
				return []*model.Location{}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp model.LocationListResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Locations) != 0 {
					t.Errorf("expected 0 locations, got %d", len(resp.Locations))
				}
			},
		},
		{
			name: "repository error",
			mockListFunc: func(ctx context.Context) ([]*model.Location, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockLocationRepository{
				listFunc: tt.mockListFunc,
			}
			handler := NewLocationHandler(mockRepo, newTestLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/locations", nil)
			w := httptest.NewRecorder()

			handler.ListLocations(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp["error"] != tt.expectedError {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, errResp["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestGetLocationTime(t *testing.T) {
	tests := []struct {
		name              string
		pathName          string
		mockGetByNameFunc func(ctx context.Context, name string) (*model.Location, error)
		expectedStatus    int
		expectedError     string
		checkResponse     func(t *testing.T, body []byte)
	}{
		{
			name:     "successful get time",
			pathName: "hq",
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				return &model.Location{
					ID:          1,
					Name:        "hq",
					Timezone:    "America/New_York",
					Description: "Headquarters",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp model.LocationTimeResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Location != "hq" {
					t.Errorf("expected location 'hq', got %s", resp.Location)
				}
				if resp.Timezone != "America/New_York" {
					t.Errorf("expected timezone 'America/New_York', got %s", resp.Timezone)
				}
				if resp.Formatted == "" {
					t.Error("expected non-empty formatted time")
				}
			},
		},
		{
			name:     "location not found",
			pathName: "nonexistent",
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				return nil, repository.ErrLocationNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Location not found",
		},
		{
			name:     "invalid timezone in location",
			pathName: "bad",
			mockGetByNameFunc: func(ctx context.Context, name string) (*model.Location, error) {
				return &model.Location{
					ID:       1,
					Name:     "bad",
					Timezone: "Invalid/Timezone",
				}, nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Invalid timezone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockLocationRepository{
				getByNameFunc: tt.mockGetByNameFunc,
			}
			handler := NewLocationHandler(mockRepo, newTestLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/locations/"+tt.pathName+"/time", nil)
			req.SetPathValue("name", tt.pathName)
			w := httptest.NewRecorder()

			handler.GetLocationTime(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp["error"] != tt.expectedError {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, errResp["error"])
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}
