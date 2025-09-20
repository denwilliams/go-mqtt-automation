# Database Configuration

The MQTT Home Automation system supports both SQLite and PostgreSQL databases.

## SQLite (Default)

SQLite is the default database option and requires no additional setup. The database file will be created automatically.

### Configuration

```yaml
database:
  type: sqlite
  connection: "automation.db"  # Path to database file
```

### Benefits
- No additional software required
- Simple setup
- Good for single-node deployments
- Automatic file-based backups

## PostgreSQL

PostgreSQL provides better performance, concurrent access, and advanced features for production deployments.

### Prerequisites

1. **Install PostgreSQL** (version 12 or later recommended)
2. **Create database and user:**

```sql
-- Connect as postgres superuser
CREATE USER automation WITH PASSWORD 'your_secure_password';
CREATE DATABASE automation OWNER automation;
GRANT ALL PRIVILEGES ON DATABASE automation TO automation;

-- Connect to the automation database
\c automation automation

-- Grant schema permissions (PostgreSQL 15+)
GRANT ALL ON SCHEMA public TO automation;
```

### Configuration

```yaml
database:
  type: postgres
  connection: "host=localhost port=5432 user=automation password=your_password dbname=automation sslmode=require"
```

#### Connection String Formats

**Host-based format:**
```
host=localhost port=5432 user=automation password=your_password dbname=automation sslmode=require
```

**URL format:**
```
postgres://automation:your_password@localhost:5432/automation?sslmode=require
```

**Cloud database examples:**

*Amazon RDS:*
```
host=your-instance.region.rds.amazonaws.com port=5432 user=automation password=your_password dbname=automation sslmode=require
```

*Google Cloud SQL:*
```
host=/cloudsql/project:region:instance user=automation password=your_password dbname=automation sslmode=disable
```

### SSL Configuration

For production deployments, always use SSL:

- `sslmode=require` - Requires SSL but doesn't verify certificates
- `sslmode=verify-ca` - Requires SSL and verifies certificate authority
- `sslmode=verify-full` - Requires SSL and verifies certificate and hostname

### Performance Tuning

For better performance with PostgreSQL:

```sql
-- Recommended settings for automation workloads
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;

-- Reload configuration
SELECT pg_reload_conf();
```

### Backup and Maintenance

**Automated backups:**
```bash
# Create backup
pg_dump -h localhost -U automation automation > backup.sql

# Restore backup
psql -h localhost -U automation automation < backup.sql
```

**Maintenance tasks:**
```sql
-- Vacuum and analyze tables (run weekly)
VACUUM ANALYZE;

-- Check database size
SELECT pg_size_pretty(pg_database_size('automation'));

-- Monitor active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'automation';
```

## Migration Between Databases

### SQLite to PostgreSQL

1. **Export data from SQLite:**
```bash
sqlite3 automation.db .dump > sqlite_export.sql
```

2. **Set up PostgreSQL** as described above

3. **Convert and import data:**
   - Update the configuration to use PostgreSQL
   - Start the application (migrations will run automatically)
   - Use custom migration scripts if needed

### PostgreSQL to SQLite

1. **Export data from PostgreSQL:**
```bash
pg_dump -h localhost -U automation --inserts automation > postgres_export.sql
```

2. **Update configuration** to use SQLite
3. **Start application** (new SQLite database will be created)

## Docker Deployment

### PostgreSQL with Docker

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: automation
      POSTGRES_USER: automation  
      POSTGRES_PASSWORD: your_secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    
  automation:
    image: your-automation-image
    depends_on:
      - postgres
    environment:
      DB_TYPE: postgres
      DB_CONNECTION: "host=postgres port=5432 user=automation password=your_secure_password dbname=automation sslmode=disable"

volumes:
  postgres_data:
```

## Troubleshooting

### Common Issues

**Connection refused:**
- Check PostgreSQL is running: `systemctl status postgresql`
- Verify port is open: `netstat -ln | grep 5432`
- Check pg_hba.conf for authentication settings

**Permission denied:**
- Ensure user has correct database permissions
- Check password is correct
- Verify SSL settings match server configuration

**Migration failures:**
- Check logs for specific SQL errors
- Ensure user has CREATE/ALTER privileges
- Verify PostgreSQL version compatibility (12+)

### Debugging Connection Issues

Enable connection logging in PostgreSQL:
```sql
ALTER SYSTEM SET log_connections = on;
ALTER SYSTEM SET log_disconnections = on;
ALTER SYSTEM SET log_statement = 'all';
SELECT pg_reload_conf();
```

Check logs in:
- Ubuntu/Debian: `/var/log/postgresql/`
- CentOS/RHEL: `/var/lib/pgsql/data/log/`
- macOS (Homebrew): `/usr/local/var/log/`