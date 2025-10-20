#!/bin/bash
#
# Database Backup Script for SQLite
#
# This script creates a backup of the SQLite database using VACUUM INTO,
# which creates a clean, optimized copy of the database.
#
# Usage:
#   ./backup-db.sh [DB_PATH] [BACKUP_DIR]
#
# Examples:
#   ./backup-db.sh data/timeservice.db backups/
#   ./backup-db.sh /app/data/timeservice.db /backups/
#

set -euo pipefail

# Configuration
DB_PATH=${1:-data/timeservice.db}
BACKUP_DIR=${2:-backups}
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/timeservice_$TIMESTAMP.db"
RETENTION_DAYS=${RETENTION_DAYS:-7}  # Keep backups for 7 days by default

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if sqlite3 is installed
if ! command -v sqlite3 &> /dev/null; then
    log_error "sqlite3 command not found. Please install SQLite."
    exit 1
fi

# Check if database file exists
if [ ! -f "$DB_PATH" ]; then
    log_error "Database file not found: $DB_PATH"
    exit 1
fi

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

log_info "Starting database backup..."
log_info "Source: $DB_PATH"
log_info "Destination: $BACKUP_FILE"

# Perform backup using VACUUM INTO (creates optimized copy)
# This is safer than simply copying the file, as it ensures consistency
if sqlite3 "$DB_PATH" "VACUUM INTO '$BACKUP_FILE'"; then
    log_info "Backup created successfully: $BACKUP_FILE"

    # Get backup file size
    BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    log_info "Backup size: $BACKUP_SIZE"
else
    log_error "Backup failed!"
    exit 1
fi

# Clean up old backups (keep only last N days based on RETENTION_DAYS)
log_info "Cleaning up old backups (keeping last $RETENTION_DAYS days)..."
DELETED_COUNT=0

# Find and delete backups older than retention period
while IFS= read -r old_backup; do
    rm -f "$old_backup"
    ((DELETED_COUNT++))
    log_info "Deleted old backup: $(basename "$old_backup")"
done < <(find "$BACKUP_DIR" -name "timeservice_*.db" -type f -mtime +"$RETENTION_DAYS")

if [ "$DELETED_COUNT" -eq 0 ]; then
    log_info "No old backups to delete"
else
    log_info "Deleted $DELETED_COUNT old backup(s)"
fi

# List remaining backups
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "timeservice_*.db" -type f | wc -l)
log_info "Total backups: $BACKUP_COUNT"

log_info "Backup completed successfully!"
