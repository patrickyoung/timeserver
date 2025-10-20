package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yourorg/timeservice/pkg/db"
	"github.com/yourorg/timeservice/pkg/model"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	// Use in-memory database for tests
	cfg := &db.Config{
		Path:         ":memory:",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		CacheSize:    -2000,     // Small cache for tests
		BusyTimeout:  5000,
		WalMode:      false,     // WAL mode not available for :memory:
		SyncMode:     "NORMAL",
		ForeignKeys:  true,
		JournalMode:  "MEMORY",  // Use memory journal for in-memory DB
	}

	database, err := db.Open(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(database, logger); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return database
}

func TestCreate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		loc := model.NewLocation("headquarters", "America/New_York", "Company HQ")

		err := repo.Create(ctx, loc)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Verify ID was set
		if loc.ID == 0 {
			t.Error("Expected ID to be set after Create()")
		}

		// Verify it can be retrieved
		retrieved, err := repo.GetByName(ctx, "headquarters")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}

		if retrieved.Name != "headquarters" {
			t.Errorf("Name = %q, want %q", retrieved.Name, "headquarters")
		}
		if retrieved.Timezone != "America/New_York" {
			t.Errorf("Timezone = %q, want %q", retrieved.Timezone, "America/New_York")
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		loc1 := model.NewLocation("office", "UTC", "First")
		loc2 := model.NewLocation("office", "Europe/London", "Second")

		if err := repo.Create(ctx, loc1); err != nil {
			t.Fatalf("First Create() error = %v", err)
		}

		err := repo.Create(ctx, loc2)
		if err != ErrLocationExists {
			t.Errorf("Create() error = %v, want %v", err, ErrLocationExists)
		}
	})

	t.Run("case insensitive duplicate", func(t *testing.T) {
		loc1 := model.NewLocation("london", "Europe/London", "London office")
		loc2 := model.NewLocation("LONDON", "Europe/London", "London office caps")

		if err := repo.Create(ctx, loc1); err != nil {
			t.Fatalf("First Create() error = %v", err)
		}

		err := repo.Create(ctx, loc2)
		if err != ErrLocationExists {
			t.Errorf("Create() error = %v, want %v", err, ErrLocationExists)
		}
	})

	t.Run("invalid timezone", func(t *testing.T) {
		loc := &model.Location{
			Name:      "invalid",
			Timezone:  "Invalid/Zone",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		err := repo.Create(ctx, loc)
		if err == nil {
			t.Error("Expected error for invalid timezone")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		loc := &model.Location{
			Name:      "",
			Timezone:  "UTC",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		err := repo.Create(ctx, loc)
		if err == nil {
			t.Error("Expected error for empty name")
		}
	})
}

func TestGetByName(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)
	ctx := context.Background()

	// Setup: Create a test location
	loc := model.NewLocation("tokyo-office", "Asia/Tokyo", "Tokyo branch")
	if err := repo.Create(ctx, loc); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("existing location", func(t *testing.T) {
		retrieved, err := repo.GetByName(ctx, "tokyo-office")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}

		if retrieved.Name != "tokyo-office" {
			t.Errorf("Name = %q, want %q", retrieved.Name, "tokyo-office")
		}
		if retrieved.Timezone != "Asia/Tokyo" {
			t.Errorf("Timezone = %q, want %q", retrieved.Timezone, "Asia/Tokyo")
		}
		if retrieved.Description != "Tokyo branch" {
			t.Errorf("Description = %q, want %q", retrieved.Description, "Tokyo branch")
		}
		if retrieved.ID == 0 {
			t.Error("Expected ID to be set")
		}
	})

	t.Run("case insensitive lookup", func(t *testing.T) {
		tests := []string{"tokyo-office", "TOKYO-OFFICE", "Tokyo-Office", "ToKyO-OfFiCe"}

		for _, name := range tests {
			retrieved, err := repo.GetByName(ctx, name)
			if err != nil {
				t.Errorf("GetByName(%q) error = %v", name, err)
				continue
			}
			if retrieved.Name != "tokyo-office" {
				t.Errorf("GetByName(%q) returned name %q, want %q", name, retrieved.Name, "tokyo-office")
			}
		}
	})

	t.Run("non-existent location", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "nonexistent")
		if err != ErrLocationNotFound {
			t.Errorf("GetByName() error = %v, want %v", err, ErrLocationNotFound)
		}
	})
}

func TestUpdate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)
	ctx := context.Background()

	// Setup: Create a test location
	original := model.NewLocation("updateme", "UTC", "Original description")
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		// Small sleep to ensure the trigger updates with a different timestamp
		time.Sleep(10 * time.Millisecond)

		updated := &model.Location{
			Timezone:    "Europe/Paris",
			Description: "Updated description",
		}

		err := repo.Update(ctx, "updateme", updated)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		// Verify the update
		retrieved, err := repo.GetByName(ctx, "updateme")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}

		if retrieved.Timezone != "Europe/Paris" {
			t.Errorf("Timezone = %q, want %q", retrieved.Timezone, "Europe/Paris")
		}
		if retrieved.Description != "Updated description" {
			t.Errorf("Description = %q, want %q", retrieved.Description, "Updated description")
		}
		// Note: UpdatedAt is managed by the database trigger
	})

	t.Run("case insensitive update", func(t *testing.T) {
		updated := &model.Location{
			Timezone:    "America/New_York",
			Description: "Updated via uppercase",
		}

		err := repo.Update(ctx, "UPDATEME", updated)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		retrieved, err := repo.GetByName(ctx, "updateme")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}

		if retrieved.Timezone != "America/New_York" {
			t.Errorf("Timezone = %q, want %q", retrieved.Timezone, "America/New_York")
		}
	})

	t.Run("non-existent location", func(t *testing.T) {
		updated := &model.Location{
			Timezone: "UTC",
		}

		err := repo.Update(ctx, "nonexistent", updated)
		if err != ErrLocationNotFound {
			t.Errorf("Update() error = %v, want %v", err, ErrLocationNotFound)
		}
	})

	t.Run("invalid timezone", func(t *testing.T) {
		updated := &model.Location{
			Timezone: "Invalid/Zone",
		}

		err := repo.Update(ctx, "updateme", updated)
		if err == nil {
			t.Error("Expected error for invalid timezone")
		}
	})
}

func TestDelete(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		// Create a location to delete
		loc := model.NewLocation("deleteme", "UTC", "Will be deleted")
		if err := repo.Create(ctx, loc); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		// Delete it
		err := repo.Delete(ctx, "deleteme")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify it's gone
		_, err = repo.GetByName(ctx, "deleteme")
		if err != ErrLocationNotFound {
			t.Errorf("After delete, GetByName() error = %v, want %v", err, ErrLocationNotFound)
		}
	})

	t.Run("case insensitive delete", func(t *testing.T) {
		// Create a location
		loc := model.NewLocation("deleteme2", "UTC", "Case test")
		if err := repo.Create(ctx, loc); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		// Delete using different case
		err := repo.Delete(ctx, "DELETEME2")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify it's gone
		_, err = repo.GetByName(ctx, "deleteme2")
		if err != ErrLocationNotFound {
			t.Errorf("After delete, GetByName() error = %v, want %v", err, ErrLocationNotFound)
		}
	})

	t.Run("non-existent location", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if err != ErrLocationNotFound {
			t.Errorf("Delete() error = %v, want %v", err, ErrLocationNotFound)
		}
	})

	t.Run("idempotency", func(t *testing.T) {
		// Create and delete
		loc := model.NewLocation("idempotent", "UTC", "Test")
		if err := repo.Create(ctx, loc); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		if err := repo.Delete(ctx, "idempotent"); err != nil {
			t.Fatalf("First delete failed: %v", err)
		}

		// Second delete should fail
		err := repo.Delete(ctx, "idempotent")
		if err != ErrLocationNotFound {
			t.Errorf("Second Delete() error = %v, want %v", err, ErrLocationNotFound)
		}
	})
}

func TestList(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)
	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		locations, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if locations == nil {
			t.Error("Expected empty slice, got nil")
		}
		if len(locations) != 0 {
			t.Errorf("Expected empty list, got %d items", len(locations))
		}
	})

	t.Run("multiple locations", func(t *testing.T) {
		// Create test locations (in non-alphabetical order)
		testLocations := []struct {
			name     string
			timezone string
			desc     string
		}{
			{"zebra", "UTC", "Last alphabetically"},
			{"alpha", "America/New_York", "First alphabetically"},
			{"delta", "Europe/London", "Middle"},
		}

		for _, tc := range testLocations {
			loc := model.NewLocation(tc.name, tc.timezone, tc.desc)
			if err := repo.Create(ctx, loc); err != nil {
				t.Fatalf("Failed to create location %s: %v", tc.name, err)
			}
		}

		// List all locations
		locations, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(locations) != 3 {
			t.Fatalf("Expected 3 locations, got %d", len(locations))
		}

		// Verify alphabetical ordering
		expectedOrder := []string{"alpha", "delta", "zebra"}
		for i, loc := range locations {
			if loc.Name != expectedOrder[i] {
				t.Errorf("Location[%d].Name = %q, want %q", i, loc.Name, expectedOrder[i])
			}
		}

		// Verify all fields are populated
		for _, loc := range locations {
			if loc.ID == 0 {
				t.Errorf("Location %s has ID 0", loc.Name)
			}
			if loc.Timezone == "" {
				t.Errorf("Location %s has empty timezone", loc.Name)
			}
			if loc.CreatedAt.IsZero() {
				t.Errorf("Location %s has zero CreatedAt", loc.Name)
			}
			if loc.UpdatedAt.IsZero() {
				t.Errorf("Location %s has zero UpdatedAt", loc.Name)
			}
		}
	})

	t.Run("after delete", func(t *testing.T) {
		// Delete one location
		if err := repo.Delete(ctx, "delta"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// List should now have 2 items
		locations, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(locations) != 2 {
			t.Fatalf("Expected 2 locations after delete, got %d", len(locations))
		}

		// Verify the right ones remain
		expectedNames := map[string]bool{"alpha": true, "zebra": true}
		for _, loc := range locations {
			if !expectedNames[loc.Name] {
				t.Errorf("Unexpected location %s in list", loc.Name)
			}
		}
	})
}

func TestContextCancellation(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)

	t.Run("create with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		loc := model.NewLocation("cancelled", "UTC", "Test")
		err := repo.Create(ctx, loc)
		if err == nil {
			t.Error("Expected error when context is cancelled")
		}
	})

	t.Run("list with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := repo.List(ctx)
		if err == nil {
			t.Error("Expected error when context is cancelled")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)
	ctx := context.Background()

	// Create multiple goroutines that insert different locations
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			loc := model.NewLocation(
				strings.ToLower(string(rune('a'+n))),
				"UTC",
				"Concurrent test",
			)
			if err := repo.Create(ctx, loc); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify all locations were created
	locations, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(locations) != 10 {
		t.Errorf("Expected 10 locations, got %d", len(locations))
	}
}

func TestDescriptionHandling(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewLocationRepository(database)
	ctx := context.Background()

	t.Run("empty description", func(t *testing.T) {
		loc := model.NewLocation("nodesc", "UTC", "")
		if err := repo.Create(ctx, loc); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		retrieved, err := repo.GetByName(ctx, "nodesc")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}

		if retrieved.Description != "" {
			t.Errorf("Description = %q, want empty string", retrieved.Description)
		}
	})

	t.Run("long description", func(t *testing.T) {
		longDesc := strings.Repeat("a", 500) // Maximum allowed
		loc := model.NewLocation("longdesc", "UTC", longDesc)
		if err := repo.Create(ctx, loc); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		retrieved, err := repo.GetByName(ctx, "longdesc")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}

		if len(retrieved.Description) != 500 {
			t.Errorf("Description length = %d, want 500", len(retrieved.Description))
		}
	})
}
