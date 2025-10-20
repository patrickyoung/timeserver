package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yourorg/timeservice/pkg/metrics"
	"github.com/yourorg/timeservice/pkg/model"
)

// Repository errors
var (
	ErrLocationNotFound = errors.New("location not found")
	ErrLocationExists   = errors.New("location already exists")
)

// LocationRepository defines the interface for location data access
type LocationRepository interface {
	Create(ctx context.Context, loc *model.Location) error
	GetByName(ctx context.Context, name string) (*model.Location, error)
	Update(ctx context.Context, name string, loc *model.Location) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]*model.Location, error)
}

// sqliteLocationRepository implements LocationRepository for SQLite
type sqliteLocationRepository struct {
	db      *sql.DB
	metrics *metrics.Metrics
}

// NewLocationRepository creates a new SQLite-backed location repository
func NewLocationRepository(db *sql.DB, m *metrics.Metrics) LocationRepository {
	return &sqliteLocationRepository{
		db:      db,
		metrics: m,
	}
}

// Create inserts a new location into the database
func (r *sqliteLocationRepository) Create(ctx context.Context, loc *model.Location) error {
	start := time.Now()
	operation := "create"

	// Validate the location
	if err := loc.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO locations (name, timezone, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		loc.Name,
		loc.Timezone,
		loc.Description,
		loc.CreatedAt,
		loc.UpdatedAt,
	)

	// Record metrics
	duration := time.Since(start).Seconds()
	r.metrics.DBQueryDuration.WithLabelValues(operation).Observe(duration)

	if err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()

		// Check for unique constraint violation (SQLITE_CONSTRAINT)
		if isSQLiteConstraintError(err) {
			return ErrLocationExists
		}
		return fmt.Errorf("failed to insert location: %w", err)
	}

	// Get the auto-generated ID
	id, err := result.LastInsertId()
	if err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return fmt.Errorf("failed to get insert id: %w", err)
	}

	r.metrics.DBQueriesTotal.WithLabelValues(operation, "success").Inc()
	loc.ID = id
	return nil
}

// GetByName retrieves a location by its name (case-insensitive)
func (r *sqliteLocationRepository) GetByName(ctx context.Context, name string) (*model.Location, error) {
	start := time.Now()
	operation := "get"

	query := `
		SELECT id, name, timezone, description, created_at, updated_at
		FROM locations
		WHERE name = ? COLLATE NOCASE
	`

	var loc model.Location
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&loc.ID,
		&loc.Name,
		&loc.Timezone,
		&loc.Description,
		&loc.CreatedAt,
		&loc.UpdatedAt,
	)

	// Record metrics
	duration := time.Since(start).Seconds()
	r.metrics.DBQueryDuration.WithLabelValues(operation).Observe(duration)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.metrics.DBQueriesTotal.WithLabelValues(operation, "not_found").Inc()
			return nil, ErrLocationNotFound
		}
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return nil, fmt.Errorf("failed to query location: %w", err)
	}

	r.metrics.DBQueriesTotal.WithLabelValues(operation, "success").Inc()
	return &loc, nil
}

// Update modifies an existing location
func (r *sqliteLocationRepository) Update(ctx context.Context, name string, loc *model.Location) error {
	start := time.Now()
	operation := "update"

	// Validate only the fields being updated
	if err := model.ValidateTimezone(loc.Timezone); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if err := model.ValidateDescription(loc.Description); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE locations
		SET timezone = ?, description = ?
		WHERE name = ? COLLATE NOCASE
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		loc.Timezone,
		loc.Description,
		name,
	)

	// Record metrics
	duration := time.Since(start).Seconds()
	r.metrics.DBQueryDuration.WithLabelValues(operation).Observe(duration)

	if err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return fmt.Errorf("failed to update location: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "not_found").Inc()
		return ErrLocationNotFound
	}

	r.metrics.DBQueriesTotal.WithLabelValues(operation, "success").Inc()
	return nil
}

// Delete removes a location by name
func (r *sqliteLocationRepository) Delete(ctx context.Context, name string) error {
	start := time.Now()
	operation := "delete"

	query := `
		DELETE FROM locations
		WHERE name = ? COLLATE NOCASE
	`

	result, err := r.db.ExecContext(ctx, query, name)

	// Record metrics
	duration := time.Since(start).Seconds()
	r.metrics.DBQueryDuration.WithLabelValues(operation).Observe(duration)

	if err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return fmt.Errorf("failed to delete location: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "not_found").Inc()
		return ErrLocationNotFound
	}

	r.metrics.DBQueriesTotal.WithLabelValues(operation, "success").Inc()
	return nil
}

// List retrieves all locations, ordered by name
func (r *sqliteLocationRepository) List(ctx context.Context) ([]*model.Location, error) {
	start := time.Now()
	operation := "list"

	query := `
		SELECT id, name, timezone, description, created_at, updated_at
		FROM locations
		ORDER BY name COLLATE NOCASE
	`

	rows, err := r.db.QueryContext(ctx, query)

	// Record query duration
	duration := time.Since(start).Seconds()
	r.metrics.DBQueryDuration.WithLabelValues(operation).Observe(duration)

	if err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return nil, fmt.Errorf("failed to query locations: %w", err)
	}
	defer rows.Close()

	var locations []*model.Location
	for rows.Next() {
		var loc model.Location
		err := rows.Scan(
			&loc.ID,
			&loc.Name,
			&loc.Timezone,
			&loc.Description,
			&loc.CreatedAt,
			&loc.UpdatedAt,
		)
		if err != nil {
			r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
			r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
			return nil, fmt.Errorf("failed to scan location: %w", err)
		}
		locations = append(locations, &loc)
	}

	if err := rows.Err(); err != nil {
		r.metrics.DBQueriesTotal.WithLabelValues(operation, "error").Inc()
		r.metrics.DBErrorsTotal.WithLabelValues(operation).Inc()
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Return empty slice instead of nil for consistency
	if locations == nil {
		locations = []*model.Location{}
	}

	r.metrics.DBQueriesTotal.WithLabelValues(operation, "success").Inc()
	return locations, nil
}

// isSQLiteConstraintError checks if an error is a SQLite constraint violation
// SQLite returns "UNIQUE constraint failed" for duplicate insertions
func isSQLiteConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// modernc.org/sqlite returns error strings containing "UNIQUE constraint"
	return contains(err.Error(), "UNIQUE constraint") ||
		contains(err.Error(), "constraint failed")
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			findSubstring(s, substr))
}

// findSubstring is a simple substring search
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
