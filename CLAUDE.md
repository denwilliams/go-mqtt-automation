# Claude Development Notes

This file contains important development notes and patterns for future Claude sessions working on this MQTT Home Automation project.

## Database Migrations

### Template-Based Migration System

This project uses a template-based system to generate database-agnostic migrations for SQLite, PostgreSQL, and MySQL.

**DO NOT** create direct `.sql` migration files. Instead:

1. **Create templates**: Add new migrations as `.sql.template` files in `db/migrations/`
2. **Use placeholders**: Replace database-specific syntax with template variables:
   - `{{.TextType}}` - TEXT for all databases
   - `{{.IntType}}` - INTEGER/INT
   - `{{.BoolType}}` - BOOLEAN
   - `{{.TimestampType}}` - TIMESTAMP
   - `{{.CurrentTimestamp}}` - CURRENT_TIMESTAMP
   - `{{.AutoIncrementType}}` - INTEGER/SERIAL/INT
   - `{{.AutoIncrementSuffix}}` - " AUTOINCREMENT"/""/", AUTO_INCREMENT"

3. **Generate migrations**: Run `make migrations` to create database-specific files

### Example Template Usage

```sql
-- In migration template file
CREATE TABLE example (
    id {{.AutoIncrementType}} PRIMARY KEY{{.AutoIncrementSuffix}},
    name {{.TextType}} NOT NULL,
    count {{.IntType}} DEFAULT 0,
    created_at {{.TimestampType}} DEFAULT {{.CurrentTimestamp}}
);
```

**Generates:**
- SQLite: `id INTEGER PRIMARY KEY AUTOINCREMENT`
- PostgreSQL: `id SERIAL PRIMARY KEY`
- MySQL: `id INT PRIMARY KEY AUTO_INCREMENT`

### Migration Workflow

```bash
# Clean and regenerate all database-specific migrations
make migrations-clean && make migrations

# View generated files
ls db/migrations/*/
```

### Files Created

- `pkg/migration/template.go` - Template processing logic
- `cmd/migrate-gen/main.go` - CLI tool for generation
- `db/migrations/*.sql.template` - Template source files
- `db/migrations/{sqlite,postgres,mysql}/` - Generated database-specific files

## Build System

### Architecture-Specific Builds

The project builds different binaries for different deployment scenarios:

- **Linux AMD64** (CGO enabled): Full SQLite support for x86_64 servers
- **Linux ARM64** (CGO disabled): Pure Go for Raspberry Pi 4 and ARM servers
- **macOS ARM64/AMD64** (CGO disabled): Development on Apple Silicon/Intel Macs

### CGO Requirements

- **SQLite with CGO**: Use `go-sqlite3` driver (better performance)
- **SQLite without CGO**: Use `modernc.org/sqlite` pure Go driver
- **PostgreSQL/MySQL**: No CGO required

## Deployment

### Systemd Service

- Service file: `systemd/automation.service`
- Installation script: `scripts/install.sh`
- User: `automation` (non-privileged)
- Directories:
  - `/opt/automation` - Binary and config
  - `/var/lib/automation` - Database files
  - `/var/log/automation` - Log files

### Installation

```bash
# Production installation
sudo ./scripts/install.sh

# Start service
sudo systemctl enable automation
sudo systemctl start automation
```

## Development Commands

```bash
# Build and test
make build
make test

# Database operations
make migrations          # Generate database-specific migrations
make migrations-clean    # Clean generated files
make db-reset           # Reset database
make db-status          # Check database status

# Development server
make dev                # Run with development config
make run                # Build and run
```

## Code Patterns

### Strategy Development

All strategies use JavaScript execution environment with context object:
- `context.inputs` - Input topic values (keyed by topic name or input name)
- `context.inputNames` - Mapping of topic paths to friendly names
- `context.emit(value)` - Emit to main topic
- `context.emit('/subtopic', value)` - Emit to derived topic
- `context.log(message)` - Log message
- `context.parameters` - Strategy parameters

#### Input Names Feature

Topics can define friendly names for input topics, making strategy development more intuitive:

**Database Configuration:**
```json
{
  "teslamate/cars/1/battery_level": "Battery Level",
  "teslamate/cars/1/healthy": "Tesla Health Status"
}
```

**Strategy Access:**
```javascript
function process(context) {
  // Access by friendly name (if input names are configured)
  const batteryLevel = context.inputs["Battery Level"];

  // Access by original topic path (always available)
  const healthStatus = context.inputs["teslamate/cars/1/healthy"];

  // View all input name mappings
  context.log("Input names available:", context.inputNames);

  return batteryLevel > 50;
}
```

**Note:** Input names are passed through the entire execution pipeline from database → topic manager → strategy engine → JavaScript context.

### Template System

When working with web templates, use proper naming conventions:
- Template files: `web/templates/*.html`
- Content blocks: `{{define "page-name-content"}}`
- Avoid generic names like `content` to prevent recursion

## Important Notes

1. **Never commit database files** (`automation.db`, `*.db-shm`, `*.db-wal`)
2. **Always run migrations** through the template system
3. **Test on target architecture** before deploying to Raspberry Pi
4. **Use proper CGO settings** for each build target
5. **Follow systemd security practices** (non-root user, restricted paths)

## Troubleshooting

### CGO Issues
```
Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work
```
**Solution**: Use CGO-enabled build or switch to pure Go SQLite driver

### PostgreSQL Migration Issues
```
syntax error at or near "AUTOINCREMENT"
```
**Solution**: Use template system instead of SQLite-specific syntax

### Template Recursion
```
exceeded maximum template depth (100000)
```
**Solution**: Use unique template names, avoid generic `content` blocks

### Input Names Not Available in Strategies
```
Strategy log: Input Names: {}
```
**Problem**: Input names are saved to database but not accessible in strategy execution context

**Root Cause**: The `AddInternalTopic` method signature was missing the `inputNames` parameter, causing input names to be lost during topic loading from database

**Solution**: Updated the entire pipeline:
1. `AddInternalTopic(name, inputs, inputNames, strategyID)` - Added inputNames parameter
2. `main.go` topic loading - Pass `cfg.InputNames` to `AddInternalTopic`
3. `api.go` topic creation - Pass `req.InputNames` to `AddInternalTopic`

**Files Modified:**
- `pkg/topics/manager.go:84` - Updated method signature and implementation
- `cmd/server/main.go:195` - Pass input names when loading topics
- `pkg/web/api.go:465` - Pass input names when creating topics via API