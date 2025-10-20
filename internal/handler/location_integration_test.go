package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/yourorg/timeservice/internal/repository"
	"github.com/yourorg/timeservice/pkg/db"
	"github.com/yourorg/timeservice/pkg/metrics"
	"github.com/yourorg/timeservice/pkg/model"
)

// testMetrics is a shared metrics instance for all tests to avoid duplicate registration
var testMetrics = metrics.New("test_handler")

// setupTestDB creates a temporary SQLite database for testing
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "timeservice-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Configure test database
	cfg := &db.Config{
		Path:         filepath.Join(tmpDir, "test.db"),
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		CacheSize:    -2000,
		BusyTimeout:  5000,
		WalMode:      false, // Disable WAL for testing
		SyncMode:     "OFF", // Faster for tests
		ForeignKeys:  true,
		JournalMode:  "MEMORY",
	}

	// Open database
	database, err := db.Open(cfg, newTestLogger())
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open test database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(database, newTestLogger()); err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func TestLocationIntegration_FullCRUDWorkflow(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository with shared test metrics
	repo := repository.NewLocationRepository(database, testMetrics)
	handler := NewLocationHandler(repo, newTestLogger())

	// Test 1: Create a location
	t.Run("Create location", func(t *testing.T) {
		reqBody := model.CreateLocationRequest{
			Name:        "hq",
			Timezone:    "America/New_York",
			Description: "Headquarters",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/locations", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.CreateLocation(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		}

		var resp model.LocationResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if resp.Name != "hq" {
			t.Errorf("expected name 'hq', got %s", resp.Name)
		}
	})

	// Test 2: Get the created location
	t.Run("Get location", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/hq", nil)
		req.SetPathValue("name", "hq")
		w := httptest.NewRecorder()

		handler.GetLocation(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp model.LocationResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "hq" {
			t.Errorf("expected name 'hq', got %s", resp.Name)
		}
		if resp.Timezone != "America/New_York" {
			t.Errorf("expected timezone 'America/New_York', got %s", resp.Timezone)
		}
	})

	// Test 3: Create another location
	t.Run("Create second location", func(t *testing.T) {
		reqBody := model.CreateLocationRequest{
			Name:        "west-office",
			Timezone:    "America/Los_Angeles",
			Description: "West Coast Office",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/locations", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.CreateLocation(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		}
	})

	// Test 4: List all locations
	t.Run("List locations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations", nil)
		w := httptest.NewRecorder()

		handler.ListLocations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp model.LocationListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Locations) != 2 {
			t.Errorf("expected 2 locations, got %d", len(resp.Locations))
		}
	})

	// Test 5: Update a location
	t.Run("Update location", func(t *testing.T) {
		reqBody := model.UpdateLocationRequest{
			Timezone:    "America/Chicago",
			Description: "Central Headquarters",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/locations/hq", bytes.NewReader(body))
		req.SetPathValue("name", "hq")
		w := httptest.NewRecorder()

		handler.UpdateLocation(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp model.LocationResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Timezone != "America/Chicago" {
			t.Errorf("expected timezone 'America/Chicago', got %s", resp.Timezone)
		}
		if resp.Description != "Central Headquarters" {
			t.Errorf("expected description 'Central Headquarters', got %s", resp.Description)
		}
	})

	// Test 6: Get time for a location
	t.Run("Get location time", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/hq/time", nil)
		req.SetPathValue("name", "hq")
		w := httptest.NewRecorder()

		handler.GetLocationTime(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp model.LocationTimeResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Location != "hq" {
			t.Errorf("expected location 'hq', got %s", resp.Location)
		}
		if resp.Timezone != "America/Chicago" {
			t.Errorf("expected timezone 'America/Chicago', got %s", resp.Timezone)
		}
		if resp.Formatted == "" {
			t.Error("expected non-empty formatted time")
		}
	})

	// Test 7: Delete a location
	t.Run("Delete location", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/locations/west-office", nil)
		req.SetPathValue("name", "west-office")
		w := httptest.NewRecorder()

		handler.DeleteLocation(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	// Test 8: Verify deletion
	t.Run("Verify deleted location is gone", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/west-office", nil)
		req.SetPathValue("name", "west-office")
		w := httptest.NewRecorder()

		handler.GetLocation(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Test 9: Verify list now has only 1 location
	t.Run("List locations after delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations", nil)
		w := httptest.NewRecorder()

		handler.ListLocations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp model.LocationListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Locations) != 1 {
			t.Errorf("expected 1 location, got %d", len(resp.Locations))
		}
	})
}

func TestLocationIntegration_ErrorScenarios(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository with shared test metrics
	repo := repository.NewLocationRepository(database, testMetrics)
	handler := NewLocationHandler(repo, newTestLogger())

	// Test 1: Duplicate location name
	t.Run("Duplicate location name", func(t *testing.T) {
		// Create first location
		reqBody := model.CreateLocationRequest{
			Name:     "duplicate",
			Timezone: "America/New_York",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/locations", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.CreateLocation(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("first create: expected status %d, got %d", http.StatusCreated, w.Code)
		}

		// Try to create duplicate
		body, _ = json.Marshal(reqBody)
		req = httptest.NewRequest(http.MethodPost, "/api/locations", bytes.NewReader(body))
		w = httptest.NewRecorder()
		handler.CreateLocation(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("duplicate create: expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})

	// Test 2: Update non-existent location
	t.Run("Update non-existent location", func(t *testing.T) {
		reqBody := model.UpdateLocationRequest{
			Timezone: "America/Chicago",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/locations/nonexistent", bytes.NewReader(body))
		req.SetPathValue("name", "nonexistent")
		w := httptest.NewRecorder()

		handler.UpdateLocation(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Test 3: Delete non-existent location
	t.Run("Delete non-existent location", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/locations/nonexistent", nil)
		req.SetPathValue("name", "nonexistent")
		w := httptest.NewRecorder()

		handler.DeleteLocation(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Test 4: Get non-existent location time
	t.Run("Get time for non-existent location", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/nonexistent/time", nil)
		req.SetPathValue("name", "nonexistent")
		w := httptest.NewRecorder()

		handler.GetLocationTime(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestLocationIntegration_ConcurrentRequests(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository with shared test metrics
	repo := repository.NewLocationRepository(database, testMetrics)
	handler := NewLocationHandler(repo, newTestLogger())

	// Create a location first
	ctx := context.Background()
	loc := model.NewLocation("concurrent-test", "America/New_York", "Test location")
	if err := repo.Create(ctx, loc); err != nil {
		t.Fatalf("failed to create test location: %v", err)
	}

	// Test concurrent reads
	t.Run("Concurrent reads", func(t *testing.T) {
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodGet, "/api/locations/concurrent-test", nil)
				req.SetPathValue("name", "concurrent-test")
				w := httptest.NewRecorder()

				handler.GetLocation(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestLocationIntegration_CaseInsensitivity(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository with shared test metrics
	repo := repository.NewLocationRepository(database, testMetrics)
	handler := NewLocationHandler(repo, newTestLogger())

	// Create location with lowercase name
	t.Run("Create location with lowercase name", func(t *testing.T) {
		reqBody := model.CreateLocationRequest{
			Name:     "testloc",
			Timezone: "America/New_York",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/locations", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.CreateLocation(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		}
	})

	// Try to retrieve with different case
	t.Run("Get location with different case", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/locations/TESTLOC", nil)
		req.SetPathValue("name", "TESTLOC")
		w := httptest.NewRecorder()

		handler.GetLocation(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp model.LocationResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "testloc" {
			t.Errorf("expected name 'testloc', got %s", resp.Name)
		}
	})

	// Try to create duplicate with different case
	t.Run("Create duplicate with different case", func(t *testing.T) {
		reqBody := model.CreateLocationRequest{
			Name:     "TESTLOC",
			Timezone: "America/Los_Angeles",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/locations", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.CreateLocation(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})
}
