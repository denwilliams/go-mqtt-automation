# MQTT Home Automation System Architecture

## Overview

This is a Go-based MQTT home automation system that processes sensor data, executes JavaScript strategies, and manages internal/external topics. The system is designed to be scalable, concurrent, and extensible for various home automation scenarios.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    MQTT Home Automation System                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │    Web      │    │   MQTT      │    │   Topic     │         │
│  │   Server    │    │   Client    │    │  Manager    │         │
│  │             │    │             │    │             │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│         │                   │                   │              │
│         └───────────────────┼───────────────────┘              │
│                             │                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Strategy   │    │    State    │    │   Config    │         │
│  │   Engine    │    │  Manager    │    │  Manager    │         │
│  │             │    │             │    │             │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
                   ┌─────────────────────────┐
                   │      SQLite Database    │
                   │                         │
                   │  • Topic States         │
                   │  • Strategy Configs     │
                   │  • Execution Logs       │
                   └─────────────────────────┘
```

## Core Components

### 1. Topic Manager (`pkg/topics/`)

The heart of the system that manages three types of topics:

#### Topic Types

**External Topics** (`external.go`)
- Represent MQTT sensor inputs (e.g., `teslamate/cars/1/battery_level`)
- Created automatically when MQTT messages arrive
- Read-only from the system perspective
- Source of truth for physical device states

**Internal Topics** (`internal.go`)
- Represent computed/derived values
- Execute JavaScript strategies when input topics change
- Can emit to subtopics (creating derived internal topics)
- Support wildcard input patterns (e.g., `teslamate/cars/1/+`)

**System Topics** (`system.go`)
- Built-in system functionality
- Tickers (1s, 5s, 30s, 1m, 5m, 15m, 30m, 1h)
- Schedulers (daily-morning, daily-evening, weekly-maintenance)
- Events (startup, shutdown, error, heartbeat)

#### Topic Relationships

```
External Topic (MQTT) → Internal Topic (Strategy) → Derived Topics
     │                        │                          │
     ▼                        ▼                          ▼
sensor/temperature    home/avg_temp              home/avg_temp/status
     │                        │                          │
  (Raw data)           (Computed value)            (Status indicator)
```

### 2. Strategy Engine (`pkg/strategy/`)

Executes JavaScript code to process topic inputs and generate outputs.

#### Strategy Execution Flow

1. **Trigger**: External/System topic updates
2. **Input Collection**: Gather values from input topics (supports wildcards)
3. **Strategy Execution**: Run JavaScript with inputs
4. **Event Emission**: Process emitted events to update topics

#### JavaScript Context

```javascript
// Available in strategy context
context = {
    emit: function(topic, value),  // Emit to topic (relative or absolute)
    log: function(message),        // Log message
    // Input values available as variables
}

// Example strategy
if (battery_level < 20) {
    context.emit('/low_battery', true);
    context.log('Low battery warning!');
}
```

### 3. MQTT Client (`pkg/mqtt/`)

Handles all MQTT communication with robust reconnection logic.

#### Features
- Auto-reconnection with exponential backoff
- Wildcard subscription support (`+`, `#`)
- Topic pattern matching
- Message routing to Topic Manager

#### Message Flow
```
MQTT Broker → MQTT Client → Topic Manager → Strategy Engine → Topic Updates
```

### 4. Web Server (`pkg/web/`)

Provides REST API and web interface for system management.

#### API Endpoints
- `GET /api/v1/topics` - List all topics with filtering and pagination
- `GET /api/v1/strategies` - List all available strategies
- `GET /api/v1/dashboard` - System health and statistics
- `POST /api/v1/topics` - Create new internal topics
- `POST /api/v1/strategies` - Create new strategies
- Full RESTful CRUD operations for topics and strategies

See [README.md](README.md#api-reference) for complete API documentation.

#### Web Interface
- Dashboard with system overview
- Topic browser and editor
- Strategy creation and management
- System logs viewer

### 5. State Manager (`pkg/state/`)

Persistent storage for topic states and configurations.

#### Storage Types
- **SQLite** (`sqlite.go`) - Default, file-based
- **PostgreSQL** (`postgres.go`) - Production scale

#### Schema
```sql
-- Topic states (current values)
state(key, value, updated_at)

-- Topic configurations
topics(name, type, inputs, strategy_id, ...)

-- Strategy definitions
strategies(id, name, code, language, ...)

-- Execution logs
execution_log(strategy_id, inputs, outputs, duration, ...)
```

## Data Flow

### 1. MQTT Message Processing

```
1. MQTT message arrives → teslamate/cars/1/battery_level: 80
2. MQTT Client receives and routes to Topic Manager
3. Topic Manager creates/updates External Topic
4. Topic Manager finds dependent Internal Topics
5. Internal Topic executes strategy with new input
6. Strategy emits results to derived topics
7. State Manager persists all changes
```

### 2. Strategy Execution

```
1. Input topic changes (External/System/Internal)
2. Topic Manager identifies dependent Internal Topics
3. For each dependent topic:
   a. Collect input values (including wildcards)
   b. Execute strategy with JavaScript engine
   c. Process emitted events
   d. Update derived topics
   e. Save state to database
```

### 3. Derived Topic Creation

When an Internal Topic strategy emits to a subtopic:

```
Internal Topic: car
Strategy emits to: /range → Creates car/range (Internal, derived)
Strategy emits to: /healthy → Creates car/healthy (Internal, derived)
```

These derived topics:
- Are classified as Internal (not External)
- Have no inputs or strategies (read-only)
- Are controlled by their parent topic
- Can optionally emit to MQTT

## Concurrency Model

### Thread Safety
- **Topic Manager**: RWMutex protects topic maps
- **MQTT Client**: RWMutex protects connection state
- **Strategy Engine**: Goroutine pool for parallel execution
- **State Manager**: Database handles concurrent access

### Event Processing
```
MQTT Message → Topic Update → Strategy Execution (async) → Derived Updates
     │              │                    │                      │
  Single thread   Protected by      Goroutine pool         State persistence
                    mutex
```

## Configuration

### System Configuration (`config/config.yaml`)
```yaml
MQTT:
  Broker: "mqtt://user:pass@broker:1883"
  Topics: ["zigbee2mqtt/+", "teslamate/cars/1/+"]

Database:
  Type: "sqlite"
  Connection: "./automation.db"

Web:
  Port: 8080
  Bind: "0.0.0.0"

SystemTopics:
  TickerIntervals: ["1s", "5s", "30s", "1m", "5m", "15m", "30m", "1h"]
```

### Topic Configuration (Database)
```sql
-- Internal topic with strategy
INSERT INTO topics (name, type, inputs, strategy_id) VALUES
('car', 'internal', '["teslamate/cars/1/+"]', 'tesla_car');

-- Strategy definition
INSERT INTO strategies (id, name, code, language) VALUES
('tesla_car', 'Tesla Car', 'if (battery_level < 20) { ... }', 'javascript');
```

## Key Design Patterns

### 1. Event-Driven Architecture
- Topics emit events when values change
- Strategies react to input changes
- Asynchronous processing prevents blocking

### 2. Plugin Architecture
- Strategies are pluggable JavaScript modules
- New functionality added via strategy creation
- No core code changes needed for new features

### 3. Separation of Concerns
- **Topic Manager**: Topic lifecycle and routing
- **Strategy Engine**: Business logic execution
- **MQTT Client**: Communication transport
- **State Manager**: Data persistence
- **Web Server**: User interface

### 4. Wildcard Pattern Matching
- Input topics support MQTT wildcards (`+`, `#`)
- Enables processing of dynamic topic hierarchies
- Single strategy can handle multiple similar devices

## Scalability Considerations

### Performance
- Concurrent strategy execution via goroutine pools
- Efficient topic lookup using Go maps
- Minimal memory allocation in hot paths

### Storage
- SQLite for single-instance deployments
- PostgreSQL for multi-instance/production
- Database migrations for schema evolution

### Extensibility
- New topic types can be added
- Additional strategy languages supported
- Custom state managers for different backends

## Security

### Network Security
- MQTT authentication via username/password
- TLS support for MQTT connections
- Web server authentication (configurable)

### Code Security
- JavaScript execution in sandboxed environment
- Input validation for strategy creation
- No direct file system access from strategies

## Monitoring and Observability

### Logging
- Structured logging with levels
- Strategy execution logs with timing
- MQTT connection status
- Web server access logs

### Metrics
- Topic count by type
- Strategy execution frequency
- MQTT message rates
- System health indicators

### Health Checks
- MQTT connection status
- Database connectivity
- Strategy execution success rates
- Memory and CPU usage

## Development Workflow

### Adding New Features

1. **New Topic Type**: Implement Topic interface
2. **New Strategy Language**: Implement Executor interface
3. **New State Backend**: Implement StateManager interface
4. **New Web Features**: Add routes and handlers

### Testing Strategy
- Unit tests for core components
- Integration tests for MQTT flows
- End-to-end tests for web interface
- Load testing for concurrent strategy execution

This architecture provides a robust, scalable foundation for home automation while maintaining simplicity and extensibility.