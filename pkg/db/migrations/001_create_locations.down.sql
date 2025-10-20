-- Rollback: Drop locations table and related objects
DROP TRIGGER IF EXISTS update_locations_updated_at;
DROP INDEX IF EXISTS idx_locations_timezone;
DROP INDEX IF EXISTS idx_locations_name;
DROP TABLE IF EXISTS locations;
