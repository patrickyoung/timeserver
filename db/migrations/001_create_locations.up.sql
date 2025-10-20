-- Create locations table for named timezone storage
CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    timezone TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast name lookups (case-insensitive)
CREATE INDEX IF NOT EXISTS idx_locations_name ON locations(name COLLATE NOCASE);

-- Index for timezone filtering/reporting
CREATE INDEX IF NOT EXISTS idx_locations_timezone ON locations(timezone);

-- Trigger to automatically update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_locations_updated_at
AFTER UPDATE ON locations
FOR EACH ROW
BEGIN
    UPDATE locations SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;
