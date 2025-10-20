# 6. SQLite for Named Location Storage

Date: 2025-10-19

## Status

Accepted

## Context

The Time Service needs to support named locations for time queries, allowing users to:
- Store custom location names mapped to IANA timezones
- Add, update, remove, and list locations
- Query current time by location name (e.g., "headquarters", "office-tokyo")

This requires persistent storage for location data. Key requirements:
1. **Simplicity**: Minimal operational complexity, no separate database server
2. **Performance**: Fast reads and writes for location lookups
3. **Reliability**: ACID guarantees, crash recovery
4. **Portability**: Works in development, containers, and Kubernetes
5. **Low Dependencies**: Aligns with our minimal external dependencies philosophy

Database options considered:
- **PostgreSQL/MySQL**: Full-featured RDBMS, but requires separate server, complex operations
- **Embedded key-value stores** (BoltDB, BadgerDB): Simple but limited query capabilities
- **SQLite**: Embedded SQL database, zero-config, ACID compliant
- **Turso/libSQL**: SQLite fork with distributed features (replication, edge replicas)

## Decision

Use **SQLite** with **modernc.org/sqlite** (pure Go driver) for location storage.

### Database: SQLite

SQLite is the optimal choice because:
- ✅ **Zero configuration**: Single file database, no server process
- ✅ **ACID compliant**: Reliable transactions and crash recovery
- ✅ **Battle-tested**: Used in billions of devices, mature codebase
- ✅ **Excellent performance**: Optimized for read-heavy workloads with proper tuning
- ✅ **Simple operations**: Backup is file copy, migrations are straightforward
- ✅ **Portable**: Works identically in dev, Docker, Kubernetes
- ✅ **Small footprint**: ~600KB compiled size
- ✅ **SQL support**: Full relational queries, indexes, constraints

**Why not Turso/libSQL?**
- Turso's strengths are distributed features (replication, embedded replicas)
- Our use case is single-node with simple CRUD operations
- Additional complexity not justified for location storage
- libSQL is a fork, while SQLite is the stable, canonical implementation
- Turso is rewriting in Rust (future uncertainty)

### Go Driver: modernc.org/sqlite

**modernc.org/sqlite** (pure Go) chosen over **mattn/go-sqlite3** (CGo):

| Feature | modernc.org/sqlite | mattn/go-sqlite3 |
|---------|-------------------|------------------|
| Implementation | Pure Go (translated C) | CGo wrapper |
| Cross-compilation | ✅ Easy (no C toolchain) | ❌ Requires CGo, C compiler |
| Build speed | ✅ Fast | ❌ Slower (C compilation) |
| Static binaries | ✅ True static linking | ⚠️ Requires CGo_ENABLED=0 workarounds |
| Performance | ✅ Comparable (~5-10% overhead) | ✅ Native C speed |
| Maintenance | ✅ Active, modern | ✅ Active, mature |
| Compatibility | ✅ SQLite 3.x compatible | ✅ SQLite 3.x compatible |

**Decision rationale**:
- **Simplicity**: Pure Go eliminates CGo dependencies and C toolchain requirements
- **Build simplicity**: Faster builds, easier cross-compilation (already using pure Go time service)
- **Docker builds**: No need for gcc/musl in builder image
- **Minimal overhead**: ~5-10% performance difference negligible for our workload
- **Philosophy alignment**: Matches our commitment to simplicity and minimal dependencies

Performance overhead is acceptable because:
- Location lookups are not in hot path (cached results recommended)
- Read-heavy workload with small dataset (hundreds to thousands of locations)
- Simplicity benefits outweigh marginal performance difference

### Schema Design

```sql
CREATE TABLE locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    timezone TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_locations_name ON locations(name COLLATE NOCASE);
CREATE INDEX idx_locations_timezone ON locations(timezone);
```

**Design decisions**:
- `name`: Case-insensitive unique constraint for user-friendly lookups
- `timezone`: IANA timezone string (validated on insert/update)
- `description`: Optional metadata for UI/documentation
- Timestamps for auditing
- Indexes on lookup columns

### Performance Optimizations

SQLite will be configured for optimal performance:

```go
// Connection string with performance pragmas
db, err := sql.Open("sqlite", "file:data/timeservice.db?"+
    "_pragma=journal_mode(WAL)&"+          // Write-Ahead Logging for concurrency
    "_pragma=synchronous(NORMAL)&"+        // Faster commits, still safe
    "_pragma=cache_size(-64000)&"+         // 64MB cache
    "_pragma=busy_timeout(5000)&"+         // 5s busy timeout
    "_pragma=foreign_keys(ON)&"+           // Enforce FK constraints
    "_pragma=temp_store(MEMORY)")          // Temp tables in memory
```

**Optimizations**:
- **WAL mode**: Concurrent readers and writer, better performance
- **Synchronous NORMAL**: Balance between speed and safety (safe for WAL mode)
- **Cache size**: 64MB for hot data (adjustable via config)
- **Busy timeout**: Prevents immediate lock errors under contention
- **Connection pooling**: Limit concurrent connections to avoid lock contention

### Migration Strategy

Use simple SQL migration files in `pkg/db/migrations/` (embedded via go:embed):
```
pkg/db/migrations/001_create_locations.up.sql
pkg/db/migrations/001_create_locations.down.sql
```

Migrations applied automatically on startup via `pkg/db/migrate.go`.

### API Design

**HTTP Endpoints**:
- `POST /api/locations` - Create location
- `GET /api/locations` - List all locations
- `GET /api/locations/{name}` - Get location details
- `PUT /api/locations/{name}` - Update location
- `DELETE /api/locations/{name}` - Delete location
- `GET /api/locations/{name}/time` - Get current time for location

**MCP Tools**:
- `add_location(name, timezone, description)` - Add named location
- `remove_location(name)` - Remove location
- `update_location(name, timezone, description)` - Update location
- `list_locations()` - List all locations
- `get_location_time(name, format)` - Get time for named location

### Authorization

Protected endpoints (write operations):
- `POST /api/locations` - Requires `locations:write` permission or `admin` role
- `PUT /api/locations/{name}` - Requires `locations:write` permission
- `DELETE /api/locations/{name}` - Requires `locations:write` permission

Public endpoints (read operations):
- `GET /api/locations` - Public or requires `locations:read` permission (configurable)
- `GET /api/locations/{name}` - Public or requires `locations:read` permission
- `GET /api/locations/{name}/time` - Public or requires `time:read` permission

MCP tools inherit HTTP endpoint permissions when auth is enabled.

## Consequences

### Positive

- **Simple operations**: Backup = copy file, restore = copy file back
- **No separate service**: Database embedded in application binary
- **Fast development**: Standard SQL, excellent tooling (sqlite3 CLI)
- **Portable**: Identical behavior across dev/test/prod environments
- **ACID guarantees**: Reliable transactions, no data corruption on crashes
- **Pure Go benefits**: Easy cross-compilation, faster builds, smaller images
- **Low resource usage**: Minimal memory and CPU overhead
- **Proven reliability**: SQLite is one of the most tested software libraries

### Negative

- **Single-node only**: No built-in replication (acceptable for our use case)
- **Write concurrency**: Limited to one writer at a time (mitigated by WAL mode)
- **File system dependency**: Performance depends on underlying filesystem
- **Pure Go overhead**: ~5-10% slower than CGo driver (negligible for our workload)

### Operational Impact

**Development**:
- `data/` directory created automatically for database file
- Migrations run on startup
- Easy to reset: delete database file and restart

**Docker**:
- Database file stored in volume mount: `-v ./data:/app/data`
- Volume ensures data persistence across container restarts
- Backup = volume snapshot

**Kubernetes**:
- PersistentVolumeClaim for database storage
- StatefulSet for stable network identity and persistent storage
- Volume snapshots for backup/restore

**Backup Strategy**:
```bash
# Online backup (SQLite VACUUM INTO or backup API)
sqlite3 data/timeservice.db "VACUUM INTO 'backup.db'"

# Offline backup (file copy)
cp data/timeservice.db backup/timeservice-$(date +%Y%m%d).db
```

### Migration Path

If distributed database becomes necessary in the future:
1. Export locations to SQL/CSV
2. Import to PostgreSQL/CockroachDB/TiDB
3. Update repository implementation (same interface)
4. No API changes required (abstraction via repository layer)

Turso/libSQL remains an option if edge replication is needed, as libSQL is SQLite-compatible.

## Alternatives Considered

### Turso/libSQL
- **Pros**: Distributed features, edge replicas, replication
- **Cons**: More complex, fork of SQLite, rewriting in Rust, overkill for single-node use case
- **Verdict**: Excellent for edge/multi-region, unnecessary for our requirements

### mattn/go-sqlite3 (CGo driver)
- **Pros**: Native C performance, widely used
- **Cons**: Requires CGo, slower builds, cross-compilation complexity
- **Verdict**: Performance benefit minimal, simplicity cost too high

### PostgreSQL
- **Pros**: Full-featured, excellent replication, mature ecosystem
- **Cons**: Separate server, operational complexity, overkill for simple location storage
- **Verdict**: Great for large-scale systems, violates simplicity principle here

### BoltDB/BadgerDB
- **Pros**: Pure Go, embedded, fast key-value access
- **Cons**: Limited query capabilities (no SQL), manual indexing, less mature
- **Verdict**: Good for simple key-value, insufficient for relational queries

## References

- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [SQLite Performance Tuning](https://www.sqlite.org/speed.html)
- [modernc.org/sqlite GitHub](https://gitlab.com/cznic/sqlite)
- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
- [High Performance SQLite](https://phiresky.github.io/blog/2020/sqlite-performance-tuning/)

## Related ADRs

- ADR 0002 – Configuration Guardrails (database path validation)
- ADR 0003 – Prometheus Instrumentation (database metrics)
- ADR 0005 – OAuth2/OIDC Authorization (location write permissions)
