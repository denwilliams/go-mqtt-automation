# MQTT Home Automation System

A Go-based home automation system that processes hundreds of external MQTT events and enables users to create custom internal topics with configurable strategies for event processing and automation logic.

## Features

- **MQTT Integration**: Process 100+ external MQTT topics with robust connection management
- **Internal Topic System**: Create custom topics with configurable input mappings and strategies
- **Strategy Engine**: Execute user-defined JavaScript code for automation logic
- **System Topics**: Time-based triggers, schedulers, and system events
- **State Persistence**: Full system state recovery after restart with SQLite or PostgreSQL
- **Web UI**: Simple HTML forms for topic and strategy management

## Quick Start

1. **Install dependencies**:
   ```bash
   go mod download
   ```

2. **Configure the system**:
   ```bash
   cp config/config.example.yaml config/config.yaml
   # Edit config/config.yaml with your MQTT broker settings
   ```

3. **Run the application**:
   ```bash
   go run cmd/server/main.go
   ```

4. **Access the web interface**: http://localhost:8080

## Architecture

The system consists of several core components:

- **MQTT Client**: Handles external MQTT connections and message routing
- **Topic Manager**: Manages internal, external, and system topics
- **Strategy Engine**: Executes JavaScript strategies in a sandboxed environment
- **State Manager**: Provides persistence and state recovery
- **Web UI**: Simple HTML interface for configuration and monitoring

## Configuration

The system supports both SQLite (default) and PostgreSQL databases:

### SQLite (Default)
```bash
cp config/config.example.yaml config/config.yaml
# Edit MQTT broker settings
```

### PostgreSQL
```bash
cp config/config.postgres.example.yaml config/config.yaml
# Set up PostgreSQL database and edit connection settings
```

See configuration files for options including:
- MQTT broker settings  
- Database configuration (SQLite/PostgreSQL)
- Web server settings
- System topic intervals

For detailed database setup instructions, see [DATABASE.md](DATABASE.md).

## Development

See `TODO.md` for development progress and `PRD.md` for detailed specifications.

## License

MIT License