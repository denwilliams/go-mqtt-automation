# MQTT Home Automation System

A Go-based home automation system that processes hundreds of external MQTT events and enables users to create custom internal topics with configurable strategies for event processing and automation logic.

## Features

- **MQTT Integration**: Process 100+ external MQTT topics with robust connection management
- **Internal Topic System**: Create custom topics with configurable input mappings, friendly input names, and strategies
- **Strategy Engine**: Execute user-defined JavaScript code for automation logic with access to input names
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

## Key Features

### Input Names for Strategies

Topics can define friendly names for input topics, making strategy development more intuitive:

**Example**: Instead of accessing `context.inputs["teslamate/cars/1/battery_level"]` in your JavaScript strategy, you can access `context.inputs["Battery Level"]` by configuring input names in the topic settings.

**Benefits**:
- More readable strategy code
- Easier maintenance when MQTT topic paths change
- Better user experience in the web interface

**Usage in JavaScript strategies**:
```javascript
function process(context) {
  // Access by friendly name
  const batteryLevel = context.inputs["Battery Level"];
  const healthStatus = context.inputs["Tesla Health Status"];

  // View all available input names
  context.log("Available inputs:", Object.keys(context.inputNames));

  return batteryLevel > 50 && healthStatus;
}
```

### Strategy Output: Last Value Wins

Strategies can emit values using `context.emit(value)` or `return value`. If multiple values are emitted to the **same topic** (main or subtopic), **only the last value is kept**.

**Behavior**:
- Multiple `context.emit(value)` calls → only the last one is used
- `context.emit(value)` + `return value` → return value wins
- Multiple emits to the same subtopic → only the last one is kept
- Different subtopics → each keeps its last emitted value

**Examples**:
```javascript
// Only 300 is emitted to the main topic
function process(context) {
  context.emit(100);  // Overwritten
  context.emit(200);  // Overwritten
  context.emit(300);  // Final value
}

// Return wins over emit
function process(context) {
  context.emit(100);  // Overwritten by return
  return 200;         // Final value = 200
}

// Different subtopics each keep their last value
function process(context) {
  context.emit('/battery', 75);     // Subtopic /battery = 75
  context.emit('/status', 'good');  // Subtopic /status = 'good'
  context.emit(100);                // Main topic - overwritten
  return 200;                       // Main topic - final value = 200
}

// Multiple emits to same subtopic - last wins
function process(context) {
  context.emit('/battery', 50);     // Overwritten
  context.emit('/battery', 75);     // Overwritten
  context.emit('/battery', 90);     // Final /battery value = 90
  return 200;                       // Main topic = 200
}
```

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

## Monitoring & Metrics

For Prometheus metrics and monitoring, we recommend using an external MQTT-to-Prometheus exporter rather than building metrics into the core system. This keeps the automation system lightweight and focused.

**Recommended approach:**
- Use [mqtt-prometheus-exporter](https://github.com/torilabs/mqtt-prometheus-exporter) or similar to expose MQTT topics as Prometheus metrics
- Configure it to subscribe to your automation system's output topics
- This allows you to monitor any topic without modifying the core system

> **TODO**: Add a detailed guide on setting up MQTT-to-Prometheus monitoring for this system.

## Development

### CI/CD

This project uses GitHub Actions for continuous integration and deployment:

- **Tests**: Automatically run on every commit and pull request
- **Code Quality**: golangci-lint ensures code quality standards
- **Releases**: Create a GitHub release to automatically build binaries for:
  - Linux (AMD64, ARM64, ARM)
  - macOS (Intel, Apple Silicon)
  - Windows (AMD64)

### Building

```bash
# Development build
make build

# Build with version info
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/server

# Run tests
make test

# Run linter
golangci-lint run
```

### Creating a Release

1. Create a new tag: `git tag v1.0.0`
2. Push the tag: `git push origin v1.0.0`
3. Create a GitHub release from the tag
4. Binaries will be automatically built and attached to the release

See `TODO.md` for development progress and `PRD.md` for detailed specifications.

## License

MIT License