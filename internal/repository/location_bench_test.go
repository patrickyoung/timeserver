package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/yourorg/timeservice/pkg/db"
	"github.com/yourorg/timeservice/pkg/metrics"
	"github.com/yourorg/timeservice/pkg/model"
)

// benchMetrics is a shared metrics instance for all benchmarks to avoid duplicate registration
var benchMetrics = metrics.New("bench_repository")

// setupBenchDB creates an in-memory database for benchmarking
func setupBenchDB(b *testing.B) *sql.DB {
	b.Helper()

	// Create a no-op logger for benchmarks to reduce overhead
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	cfg := &db.Config{
		Path:         ":memory:",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		BusyTimeout:  5000,
		ForeignKeys:  true,
		JournalMode:  "WAL",
		SyncMode:     "NORMAL",
		CacheSize:    -64000, // 64MB cache
		WalMode:      true,
	}

	database, err := db.Open(cfg, logger)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(database, logger); err != nil {
		database.Close()
		b.Fatalf("failed to run migrations: %v", err)
	}

	return database
}

// BenchmarkLocationCreate benchmarks location creation
func BenchmarkLocationCreate(b *testing.B) {
	database := setupBenchDB(b)
	defer database.Close()

	repo := NewLocationRepository(database, benchMetrics)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loc := model.NewLocation(
			fmt.Sprintf("bench-location-%d", i),
			"America/New_York",
			"Benchmark test location",
		)
		_ = repo.Create(ctx, loc)
	}
}

// BenchmarkLocationGet benchmarks location retrieval by name
func BenchmarkLocationGet(b *testing.B) {
	database := setupBenchDB(b)
	defer database.Close()

	// Use shared benchmark metrics
	repo := NewLocationRepository(database, benchMetrics)
	_ = "bench_get")
	repo := NewLocationRepository(database, m)
	ctx := context.Background()

	// Create some test locations
	for i := 0; i < 100; i++ {
		loc := model.NewLocation(
			fmt.Sprintf("test-location-%d", i),
			"America/New_York",
			"Test location",
		)
		repo.Create(ctx, loc)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("test-location-%d", i%100)
		_, _ = repo.GetByName(ctx, name)
	}
}

// BenchmarkLocationList benchmarks listing all locations
func BenchmarkLocationList(b *testing.B) {
	database := setupBenchDB(b)
	defer database.Close()

	// Use shared benchmark metrics
	repo := NewLocationRepository(database, benchMetrics)
	_ = "bench_list")
	repo := NewLocationRepository(database, m)
	ctx := context.Background()

	// Create test data with different sizes
	testSizes := []int{10, 100, 1000, 5000}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// Setup: create locations
			for i := 0; i < size; i++ {
				loc := model.NewLocation(
					fmt.Sprintf("list-location-%d-%d", size, i),
					"America/New_York",
					"List benchmark location",
				)
				repo.Create(ctx, loc)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = repo.List(ctx)
			}
		})
	}
}

// BenchmarkLocationUpdate benchmarks location updates
func BenchmarkLocationUpdate(b *testing.B) {
	database := setupBenchDB(b)
	defer database.Close()

	// Use shared benchmark metrics
	repo := NewLocationRepository(database, benchMetrics)
	_ = "bench_update")
	repo := NewLocationRepository(database, m)
	ctx := context.Background()

	// Create test locations
	for i := 0; i < 100; i++ {
		loc := model.NewLocation(
			fmt.Sprintf("update-location-%d", i),
			"America/New_York",
			"Original description",
		)
		repo.Create(ctx, loc)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("update-location-%d", i%100)
		// Create a location with updated values
		updatedLoc := model.NewLocation(
			name,
			"America/Los_Angeles",
			fmt.Sprintf("Updated description %d", i),
		)
		_ = repo.Update(ctx, name, updatedLoc)
	}
}

// BenchmarkLocationDelete benchmarks location deletion
func BenchmarkLocationDelete(b *testing.B) {
	database := setupBenchDB(b)
	defer database.Close()

	// Use shared benchmark metrics
	repo := NewLocationRepository(database, benchMetrics)
	_ = "bench_delete")
	repo := NewLocationRepository(database, m)
	ctx := context.Background()

	// Pre-create locations for deletion
	// We need more than b.N to avoid running out during benchmark
	numLocations := b.N * 10
	if numLocations < 10000 {
		numLocations = 10000
	}

	for i := 0; i < numLocations; i++ {
		loc := model.NewLocation(
			fmt.Sprintf("delete-location-%d", i),
			"America/New_York",
			"Delete benchmark location",
		)
		repo.Create(ctx, loc)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("delete-location-%d", i)
		_ = repo.Delete(ctx, name)
	}
}

// BenchmarkLocationConcurrent benchmarks concurrent operations
func BenchmarkLocationConcurrent(b *testing.B) {
	database := setupBenchDB(b)
	defer database.Close()

	// Use shared benchmark metrics
	repo := NewLocationRepository(database, benchMetrics)
	_ = "bench_concurrent")
	repo := NewLocationRepository(database, m)
	ctx := context.Background()

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		loc := model.NewLocation(
			fmt.Sprintf("concurrent-location-%d", i),
			"America/New_York",
			"Concurrent benchmark location",
		)
		repo.Create(ctx, loc)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Mix of operations
			switch i % 4 {
			case 0: // Create
				loc := model.NewLocation(
					fmt.Sprintf("concurrent-new-%d", i),
					"America/New_York",
					"New location",
				)
				repo.Create(ctx, loc)
			case 1: // Get
				name := fmt.Sprintf("concurrent-location-%d", i%100)
				repo.GetByName(ctx, name)
			case 2: // Update
				name := fmt.Sprintf("concurrent-location-%d", i%100)
				updatedLoc := model.NewLocation(
					name,
					"America/New_York",
					fmt.Sprintf("Updated %d", i),
				)
				repo.Update(ctx, name, updatedLoc)
			case 3: // List
				repo.List(ctx)
			}
			i++
		}
	})
}

// BenchmarkLocationFullCRUDCycle benchmarks a complete CRUD cycle
func BenchmarkLocationFullCRUDCycle(b *testing.B) {
	database := setupBenchDB(b)
	defer database.Close()

	// Use shared benchmark metrics
	repo := NewLocationRepository(database, benchMetrics)
	_ = "bench_crud_cycle")
	repo := NewLocationRepository(database, m)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("crud-location-%d", i)

		// Create
		loc := model.NewLocation(name, "America/New_York", "CRUD test")
		repo.Create(ctx, loc)

		// Read
		repo.GetByName(ctx, name)

		// Update
		updatedLoc := model.NewLocation(name, "America/Los_Angeles", "CRUD test")
		repo.Update(ctx, name, updatedLoc)

		// Read again
		repo.GetByName(ctx, name)

		// Delete
		repo.Delete(ctx, name)
	}
}
