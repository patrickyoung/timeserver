package mcpserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yourorg/timeservice/internal/repository"
	"github.com/yourorg/timeservice/internal/testutil"
	"github.com/yourorg/timeservice/pkg/model"
)

// mockLocationRepository is a mock implementation for testing
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

func TestHandleAddLocation(t *testing.T) {
	tests := []struct {
		name         string
		arguments    map[string]interface{}
		mockCreate   func(ctx context.Context, loc *model.Location) error
		shouldError  bool
		errorMessage string
	}{
		{
			name: "successful add",
			arguments: map[string]interface{}{
				"name":        "hq",
				"timezone":    "America/New_York",
				"description": "Headquarters",
			},
			mockCreate: func(ctx context.Context, loc *model.Location) error {
				loc.ID = 1
				return nil
			},
			shouldError: false,
		},
		{
			name: "missing name parameter",
			arguments: map[string]interface{}{
				"timezone": "America/New_York",
			},
			shouldError:  true,
			errorMessage: "Parameter 'name' is required",
		},
		{
			name: "missing timezone parameter",
			arguments: map[string]interface{}{
				"name": "hq",
			},
			shouldError:  true,
			errorMessage: "Parameter 'timezone' is required",
		},
		{
			name: "invalid timezone",
			arguments: map[string]interface{}{
				"name":     "test",
				"timezone": "Invalid/Timezone",
			},
			mockCreate: func(ctx context.Context, loc *model.Location) error {
				return nil
			},
			shouldError:  true,
			errorMessage: "Validation failed",
		},
		{
			name: "duplicate location",
			arguments: map[string]interface{}{
				"name":     "existing",
				"timezone": "America/New_York",
			},
			mockCreate: func(ctx context.Context, loc *model.Location) error {
				return repository.ErrLocationExists
			},
			shouldError:  true,
			errorMessage: "already exists",
		},
		{
			name: "repository error",
			arguments: map[string]interface{}{
				"name":     "test",
				"timezone": "America/New_York",
			},
			mockCreate: func(ctx context.Context, loc *model.Location) error {
				return errors.New("database error")
			},
			shouldError:  true,
			errorMessage: "Failed to add location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger()
			ctx := context.Background()

			mockRepo := &mockLocationRepository{
				createFunc: tt.mockCreate,
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.arguments,
				},
			}

			result, err := handleAddLocation(ctx, request, logger, mockRepo)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			if tt.shouldError {
				if !result.IsError {
					t.Error("expected error result, got success")
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", result.Content)
				}
				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}
			}
		})
	}
}

func TestHandleRemoveLocation(t *testing.T) {
	tests := []struct {
		name         string
		arguments    map[string]interface{}
		mockDelete   func(ctx context.Context, name string) error
		shouldError  bool
		errorMessage string
	}{
		{
			name: "successful remove",
			arguments: map[string]interface{}{
				"name": "hq",
			},
			mockDelete: func(ctx context.Context, name string) error {
				return nil
			},
			shouldError: false,
		},
		{
			name:         "missing name parameter",
			arguments:    map[string]interface{}{},
			shouldError:  true,
			errorMessage: "Parameter 'name' is required",
		},
		{
			name: "location not found",
			arguments: map[string]interface{}{
				"name": "nonexistent",
			},
			mockDelete: func(ctx context.Context, name string) error {
				return repository.ErrLocationNotFound
			},
			shouldError:  true,
			errorMessage: "not found",
		},
		{
			name: "repository error",
			arguments: map[string]interface{}{
				"name": "test",
			},
			mockDelete: func(ctx context.Context, name string) error {
				return errors.New("database error")
			},
			shouldError:  true,
			errorMessage: "Failed to remove location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger()
			ctx := context.Background()

			mockRepo := &mockLocationRepository{
				deleteFunc: tt.mockDelete,
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.arguments,
				},
			}

			result, err := handleRemoveLocation(ctx, request, logger, mockRepo)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			if tt.shouldError {
				if !result.IsError {
					t.Error("expected error result, got success")
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", result.Content)
				}
				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}
			}
		})
	}
}

func TestHandleUpdateLocation(t *testing.T) {
	existingLocation := &model.Location{
		ID:          1,
		Name:        "hq",
		Timezone:    "America/New_York",
		Description: "Old description",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name         string
		arguments    map[string]interface{}
		mockGetByName func(ctx context.Context, name string) (*model.Location, error)
		mockUpdate   func(ctx context.Context, name string, loc *model.Location) error
		shouldError  bool
		errorMessage string
	}{
		{
			name: "successful update timezone",
			arguments: map[string]interface{}{
				"name":     "hq",
				"timezone": "America/Chicago",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				loc := *existingLocation
				return &loc, nil
			},
			mockUpdate: func(ctx context.Context, name string, loc *model.Location) error {
				return nil
			},
			shouldError: false,
		},
		{
			name: "successful update description",
			arguments: map[string]interface{}{
				"name":        "hq",
				"description": "New description",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				loc := *existingLocation
				return &loc, nil
			},
			mockUpdate: func(ctx context.Context, name string, loc *model.Location) error {
				return nil
			},
			shouldError: false,
		},
		{
			name: "missing name parameter",
			arguments: map[string]interface{}{
				"timezone": "America/Chicago",
			},
			shouldError:  true,
			errorMessage: "Parameter 'name' is required",
		},
		{
			name: "no fields to update",
			arguments: map[string]interface{}{
				"name": "hq",
			},
			shouldError:  true,
			errorMessage: "At least one of 'timezone' or 'description' must be provided",
		},
		{
			name: "location not found",
			arguments: map[string]interface{}{
				"name":     "nonexistent",
				"timezone": "America/Chicago",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				return nil, repository.ErrLocationNotFound
			},
			shouldError:  true,
			errorMessage: "not found",
		},
		{
			name: "invalid timezone",
			arguments: map[string]interface{}{
				"name":     "hq",
				"timezone": "Invalid/Timezone",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				loc := *existingLocation
				return &loc, nil
			},
			shouldError:  true,
			errorMessage: "Invalid timezone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger()
			ctx := context.Background()

			mockRepo := &mockLocationRepository{
				getByNameFunc: tt.mockGetByName,
				updateFunc:    tt.mockUpdate,
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.arguments,
				},
			}

			result, err := handleUpdateLocation(ctx, request, logger, mockRepo)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			if tt.shouldError {
				if !result.IsError {
					t.Error("expected error result, got success")
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", result.Content)
				}
				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}
			}
		})
	}
}

func TestHandleListLocations(t *testing.T) {
	tests := []struct {
		name        string
		mockList    func(ctx context.Context) ([]*model.Location, error)
		shouldError bool
		expectCount int
	}{
		{
			name: "successful list with locations",
			mockList: func(ctx context.Context) ([]*model.Location, error) {
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
						Name:        "west",
						Timezone:    "America/Los_Angeles",
						Description: "West Coast",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				}, nil
			},
			shouldError: false,
			expectCount: 2,
		},
		{
			name: "empty list",
			mockList: func(ctx context.Context) ([]*model.Location, error) {
				return []*model.Location{}, nil
			},
			shouldError: false,
			expectCount: 0,
		},
		{
			name: "repository error",
			mockList: func(ctx context.Context) ([]*model.Location, error) {
				return nil, errors.New("database error")
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger()
			ctx := context.Background()

			mockRepo := &mockLocationRepository{
				listFunc: tt.mockList,
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{},
				},
			}

			result, err := handleListLocations(ctx, request, logger, mockRepo)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			if tt.shouldError {
				if !result.IsError {
					t.Error("expected error result, got success")
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", result.Content)
				}
				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}
			}
		})
	}
}

func TestHandleGetLocationTime(t *testing.T) {
	tests := []struct {
		name          string
		arguments     map[string]interface{}
		mockGetByName func(ctx context.Context, name string) (*model.Location, error)
		shouldError   bool
		errorMessage  string
	}{
		{
			name: "successful get time",
			arguments: map[string]interface{}{
				"name": "hq",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				return &model.Location{
					ID:          1,
					Name:        "hq",
					Timezone:    "America/New_York",
					Description: "Headquarters",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			shouldError: false,
		},
		{
			name: "successful get time with format",
			arguments: map[string]interface{}{
				"name":   "hq",
				"format": "unix",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				return &model.Location{
					ID:          1,
					Name:        "hq",
					Timezone:    "America/New_York",
					Description: "Headquarters",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			shouldError: false,
		},
		{
			name: "missing name parameter",
			arguments: map[string]interface{}{
				"format": "rfc3339",
			},
			shouldError:  true,
			errorMessage: "Parameter 'name' is required",
		},
		{
			name: "location not found",
			arguments: map[string]interface{}{
				"name": "nonexistent",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				return nil, repository.ErrLocationNotFound
			},
			shouldError:  true,
			errorMessage: "not found",
		},
		{
			name: "invalid timezone in location",
			arguments: map[string]interface{}{
				"name": "bad",
			},
			mockGetByName: func(ctx context.Context, name string) (*model.Location, error) {
				return &model.Location{
					ID:       1,
					Name:     "bad",
					Timezone: "Invalid/Timezone",
				}, nil
			},
			shouldError:  true,
			errorMessage: "Invalid timezone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger()
			ctx := context.Background()

			mockRepo := &mockLocationRepository{
				getByNameFunc: tt.mockGetByName,
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.arguments,
				},
			}

			result, err := handleGetLocationTime(ctx, request, logger, mockRepo)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result to be non-nil")
			}

			if tt.shouldError {
				if !result.IsError {
					t.Error("expected error result, got success")
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", result.Content)
				}
				if len(result.Content) == 0 {
					t.Error("expected result content to be non-empty")
				}
			}
		})
	}
}

