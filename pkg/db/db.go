package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds database configuration
type Config struct {
	Path         string
	MaxOpenConns int
	MaxIdleConns int
	CacheSize    int    // In KB, negative for pages
	BusyTimeout  int    // In milliseconds
	WalMode      bool   // Use Write-Ahead Logging
	SyncMode     string // OFF, NORMAL, FULL, EXTRA
	ForeignKeys  bool
	JournalMode  string // DELETE, TRUNCATE, PERSIST, MEMORY, WAL, OFF
}

// DefaultConfig returns sensible defaults for production
func DefaultConfig() *Config {
	return &Config{
		Path:         "data/timeservice.db",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		CacheSize:    -64000, // 64MB cache (negative = pages)
		BusyTimeout:  5000,   // 5 seconds
		WalMode:      true,
		SyncMode:     "NORMAL", // Safe for WAL mode, faster than FULL
		ForeignKeys:  true,
		JournalMode:  "WAL",
	}
}

// Open opens a connection to the SQLite database with performance optimizations
func Open(cfg *Config, logger *slog.Logger) (*sql.DB, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Build connection string with pragmas
	connStr := buildConnectionString(cfg)

	logger.Info("opening database",
		"path", cfg.Path,
		"wal_mode", cfg.WalMode,
		"cache_size_kb", cfg.CacheSize/-1,
		"busy_timeout_ms", cfg.BusyTimeout,
	)

	// Open database connection
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		"max_open_conns", cfg.MaxOpenConns,
		"max_idle_conns", cfg.MaxIdleConns,
	)

	return db, nil
}

// buildConnectionString constructs the SQLite connection string with pragmas
func buildConnectionString(cfg *Config) string {
	var pragmas []string

	// Journal mode (WAL recommended for concurrency)
	pragmas = append(pragmas, fmt.Sprintf("_pragma=journal_mode(%s)", cfg.JournalMode))

	// Synchronous mode
	pragmas = append(pragmas, fmt.Sprintf("_pragma=synchronous(%s)", cfg.SyncMode))

	// Cache size (negative = pages, positive = KB)
	pragmas = append(pragmas, fmt.Sprintf("_pragma=cache_size(%d)", cfg.CacheSize))

	// Busy timeout
	pragmas = append(pragmas, fmt.Sprintf("_pragma=busy_timeout(%d)", cfg.BusyTimeout))

	// Foreign keys
	if cfg.ForeignKeys {
		pragmas = append(pragmas, "_pragma=foreign_keys(ON)")
	}

	// Temp store in memory for performance
	pragmas = append(pragmas, "_pragma=temp_store(MEMORY)")

	// Build connection string
	// Format: file:path?pragma1&pragma2&...
	connStr := fmt.Sprintf("file:%s?%s", cfg.Path, strings.Join(pragmas, "&"))
	return connStr
}

// Migrate runs all pending migrations
func Migrate(db *sql.DB, logger *slog.Logger) error {
	logger.Info("running database migrations")

	// Create migrations tracking table
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get available migrations
	available, err := getAvailableMigrations()
	if err != nil {
		return fmt.Errorf("failed to get available migrations: %w", err)
	}

	// Apply pending migrations
	pending := getPendingMigrations(available, applied)
	if len(pending) == 0 {
		logger.Info("no pending migrations")
		return nil
	}

	logger.Info("applying migrations", "count", len(pending))

	for _, name := range pending {
		if err := applyMigration(db, name, logger); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", name, err)
		}
	}

	logger.Info("migrations completed successfully", "applied", len(pending))
	return nil
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// getAppliedMigrations returns the list of already applied migration versions
func getAppliedMigrations(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applied []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied = append(applied, version)
	}

	return applied, rows.Err()
}

// getAvailableMigrations returns all available migration files (*.up.sql)
func getAvailableMigrations() ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var migrations []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			// Extract version (e.g., "001" from "001_create_locations.up.sql")
			version := strings.TrimSuffix(name, ".up.sql")
			migrations = append(migrations, version)
		}
	}

	sort.Strings(migrations)
	return migrations, nil
}

// getPendingMigrations returns migrations that haven't been applied yet
func getPendingMigrations(available, applied []string) []string {
	appliedSet := make(map[string]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	var pending []string
	for _, v := range available {
		if !appliedSet[v] {
			pending = append(pending, v)
		}
	}

	return pending
}

// applyMigration applies a single migration
func applyMigration(db *sql.DB, version string, logger *slog.Logger) error {
	logger.Info("applying migration", "version", version)

	// Read migration file
	filename := fmt.Sprintf("migrations/%s.up.sql", version)
	content, err := migrationsFS.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration in a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration as applied
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	logger.Info("migration applied successfully", "version", version)
	return nil
}

// Close closes the database connection gracefully
func Close(db *sql.DB, logger *slog.Logger) error {
	logger.Info("closing database connection")
	return db.Close()
}
