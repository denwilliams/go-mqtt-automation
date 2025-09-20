# MQTT Home Automation System - PRD & Software Design Specification

## 1. Product Requirements Document (PRD)

### 1.1 Product Overview
A Go-based home automation system that processes hundreds of external MQTT events and enables users to create custom internal topics with configurable strategies for event processing and automation logic.

### 1.2 Core Requirements

#### 1.2.1 MQTT Integration
- **External Topics**: Receive and process 100+ external MQTT topic events
- **Internal Topic Publishing**: Optionally emit processed results back to MQTT
- **Connection Management**: Robust MQTT client with reconnection and error handling
- **Topic Wildcards**: Support MQTT wildcard subscriptions where applicable

#### 1.2.2 Internal Topic System
- **Topic Creation**: Users can create internal topics with custom names
- **Input Mapping**: Each internal topic can have 1+ input topics (external or internal)
- **Strategy Assignment**: Each internal topic must have an attached strategy
- **Output Control**: Topics can emit to their named path or multiple sub-topics
- **No-op Configuration**: Optional setting to suppress output when values are unchanged

#### 1.2.3 Strategy Engine
- **Strategy Execution**: Run custom code when input topics emit
- **Input Parameters**: Receive last values of all inputs, triggering topic name, and last outputs
- **Multiple Outputs**: Support default output and multiple sub-topic outputs
- **Database Storage**: Strategies must be persistable to database
- **Runtime Execution**: Safe execution environment for user-defined code

#### 1.2.4 System Topics
- **Ticker**: Configurable time-based triggers
- **Scheduler**: Cron-like scheduling system
- **System Events**: Boot, shutdown, error events
- **Extensible Framework**: Easy addition of new system topic types

#### 1.2.5 State Persistence
- **State Recovery**: Full system state restoration after restart
- **Topic Values**: Last known values for all topics
- **Strategy State**: Internal strategy state preservation
- **Configuration**: All topic and strategy configurations

#### 1.2.6 Web UI
- **Plain HTML Forms**: Minimal JavaScript, maximum compatibility
- **Topic Management**: Create, edit, delete internal topics
- **Strategy Management**: Create, edit, assign strategies
- **System Monitoring**: View topic values and system status
- **Configuration**: System settings and MQTT connection setup

### 1.3 User Stories

#### 1.3.1 System Administrator
- As an admin, I want to configure MQTT connection settings
- As an admin, I want to view system health and diagnostics
- As an admin, I want to backup and restore system configuration

#### 1.3.2 Automation Developer
- As a developer, I want to create internal topics that process sensor data
- As a developer, I want to write custom strategies in a simple scripting language
- As a developer, I want to test strategies before deployment
- As a developer, I want to view logs and debug strategy execution

#### 1.3.3 End User
- As a user, I want to see current values of all topics
- As a user, I want to enable/disable automations
- As a user, I want to receive notifications when automations trigger

## 2. Software Design Specification

### 2.1 System Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web UI        │    │  MQTT Broker    │    │   Database      │
│  (HTTP Server)  │    │   (External)    │    │  (SQLite/PG)    │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │              ┌───────▼───────┐              │
          │              │  MQTT Client  │              │
          │              └───────┬───────┘              │
          │                      │                      │
          └──────────────────────▼──────────────────────┘
                          ┌─────────────────┐
                          │   Core Engine   │
                          │                 │
                          │ ┌─────────────┐ │
                          │ │Topic Manager│ │
                          │ └─────────────┘ │
                          │ ┌─────────────┐ │
                          │ │Strategy Eng.│ │
                          │ └─────────────┘ │
                          │ ┌─────────────┐ │
                          │ │State Manager│ │
                          │ └─────────────┘ │
                          └─────────────────┘
```

### 2.2 Core Components

#### 2.2.1 MQTT Client (`pkg/mqtt/`)
```go
type MQTTClient struct {
    client   mqtt.Client
    topics   map[string]chan<- Event
    config   MQTTConfig
}

type Event struct {
    Topic     string
    Payload   []byte
    Timestamp time.Time
}
```

#### 2.2.2 Topic Manager (`pkg/topics/`)
```go
type TopicManager struct {
    topics    map[string]*Topic
    externals map[string]*ExternalTopic
    internals map[string]*InternalTopic
    systemTopics map[string]*SystemTopic
}

type Topic interface {
    Name() string
    LastValue() interface{}
    Emit(value interface{}) error
}

type InternalTopic struct {
    name       string
    inputs     []string
    strategy   *Strategy
    lastValue  interface{}
    emitToMQTT bool
    noOpUnchanged bool
}
```

#### 2.2.3 Strategy Engine (`pkg/strategy/`)
```go
type StrategyEngine struct {
    strategies map[string]*Strategy
    executor   *Executor
}

type Strategy struct {
    ID          string
    Name        string
    Code        string
    Language    string // "javascript", "lua", "go-template"
    Parameters  map[string]interface{}
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type ExecutionContext struct {
    InputValues    map[string]interface{}
    TriggeringTopic string
    LastOutputs    map[string]interface{}
    Parameters     map[string]interface{}
}
```

#### 2.2.4 State Manager (`pkg/state/`)
```go
type StateManager struct {
    db     Database
    cache  map[string]interface{}
}

type Database interface {
    SaveTopic(topic *InternalTopic) error
    LoadTopic(name string) (*InternalTopic, error)
    SaveStrategy(strategy *Strategy) error
    LoadStrategy(id string) (*Strategy, error)
    SaveState(key string, value interface{}) error
    LoadState(key string) (interface{}, error)
}
```

### 2.3 Database Schema

#### 2.3.1 Topics Table
```sql
CREATE TABLE topics (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL, -- 'internal', 'external', 'system'
    inputs TEXT, -- JSON array of input topic names
    strategy_id TEXT,
    emit_to_mqtt BOOLEAN DEFAULT false,
    noop_unchanged BOOLEAN DEFAULT false,
    last_value TEXT, -- JSON serialized value
    last_updated TIMESTAMP,
    config TEXT, -- JSON configuration
    FOREIGN KEY (strategy_id) REFERENCES strategies(id)
);
```

#### 2.3.2 Strategies Table
```sql
CREATE TABLE strategies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    language TEXT DEFAULT 'javascript',
    parameters TEXT, -- JSON
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 2.3.3 State Table
```sql
CREATE TABLE state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL, -- JSON serialized
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 2.4 Strategy Execution Environment

#### 2.4.1 JavaScript Engine (Primary)
- Use `goja` library for JavaScript execution
- Sandboxed environment with limited API access
- Built-in functions: `log()`, `emit()`, `getTime()`, `parseJSON()`, `stringify()`

```javascript
// Example strategy code
function process(context) {
    const temp = context.inputs['sensor/temperature'];
    const humidity = context.inputs['sensor/humidity'];
    
    if (temp > 25 && humidity < 40) {
        context.emit('hvac/cooling', { enabled: true, target: 22 });
        context.log('Cooling activated due to high temp and low humidity');
    }
    
    return {
        'comfort_index': calculateComfort(temp, humidity),
        'recommendation': temp > 25 ? 'cool' : 'maintain'
    };
}
```

#### 2.4.2 Alternative: Go Templates
- For simple logic and string manipulation
- Safe, limited functionality
- Good for formatting outputs

### 2.5 System Topics

#### 2.5.1 Ticker Topics
- `system/ticker/1s`, `system/ticker/5s`, `system/ticker/1m`, etc.
- Emit timestamps at configured intervals

#### 2.5.2 Scheduler Topics
- `system/scheduler/[name]`
- Cron-based scheduling
- User-configurable schedules

#### 2.5.3 System Event Topics
- `system/events/startup`
- `system/events/shutdown`
- `system/events/error`

### 2.6 Web UI Structure

#### 2.6.1 Pages and Forms
```
/                    - Dashboard (topic status overview)
/topics              - List all topics
/topics/new          - Create new internal topic
/topics/edit/{name}  - Edit topic configuration
/strategies          - List all strategies
/strategies/new      - Create new strategy
/strategies/edit/{id} - Edit strategy code
/system              - System configuration
/logs                - View execution logs
```

#### 2.6.2 HTML Form Examples
```html
<!-- Topic Creation Form -->
<form method="POST" action="/topics/create">
    <label>Topic Name: <input name="name" required></label>
    <label>Input Topics: <textarea name="inputs" placeholder="sensor/temp&#10;sensor/humidity"></textarea></label>
    <label>Strategy: <select name="strategy_id">...</select></label>
    <label><input type="checkbox" name="emit_to_mqtt"> Emit to MQTT</label>
    <label><input type="checkbox" name="noop_unchanged"> Skip unchanged values</label>
    <button type="submit">Create Topic</button>
</form>
```

### 2.7 Directory Structure
```
mqtt-automation/
├── cmd/
│   └── server/
│       └── main.go
├── pkg/
│   ├── mqtt/
│   │   ├── client.go
│   │   └── config.go
│   ├── topics/
│   │   ├── manager.go
│   │   ├── internal.go
│   │   ├── external.go
│   │   └── system.go
│   ├── strategy/
│   │   ├── engine.go
│   │   ├── executor.go
│   │   └── javascript.go
│   ├── state/
│   │   ├── manager.go
│   │   ├── database.go
│   │   └── sqlite.go
│   └── web/
│       ├── server.go
│       ├── handlers.go
│       └── templates/
├── web/
│   ├── static/
│   │   └── style.css
│   └── templates/
│       ├── base.html
│       ├── dashboard.html
│       ├── topics.html
│       └── strategies.html
├── migrations/
├── config/
│   └── config.yaml
└── docker/
    └── Dockerfile
```

### 2.8 Configuration Management

#### 2.8.1 Configuration File (`config/config.yaml`)
```yaml
mqtt:
  broker: "mqtt://localhost:1883"
  client_id: "home-automation"
  username: ""
  password: ""
  topics:
    - "sensors/+"
    - "devices/+"

database:
  type: "sqlite" # or "postgres"
  connection: "./automation.db"

web:
  port: 8080
  bind: "0.0.0.0"

logging:
  level: "info"
  file: "./automation.log"

system_topics:
  ticker_intervals:
    - "1s"
    - "5s"
    - "30s"
    - "1m"
    - "5m"
```

## 3. Additional Considerations

### 3.1 Security Considerations
- **Strategy Sandboxing**: Ensure user strategies cannot access system resources
- **Input Validation**: Validate all user inputs in web forms
- **MQTT Authentication**: Support MQTT username/password and certificates
- **Access Control**: Consider adding user authentication for the web UI

### 3.2 Performance Considerations
- **Event Queuing**: Implement buffered channels for high-frequency events
- **Strategy Caching**: Cache compiled strategies for better performance
- **Database Indexing**: Proper indexing on frequently queried columns
- **Memory Management**: Monitor memory usage with many active topics

### 3.3 Reliability Considerations
- **MQTT Reconnection**: Automatic reconnection with exponential backoff
- **Error Recovery**: Graceful handling of strategy execution errors
- **Health Checks**: Implement health check endpoints
- **Graceful Shutdown**: Proper cleanup on system shutdown

### 3.4 Monitoring and Debugging
- **Execution Logs**: Detailed logging of strategy executions
- **Metrics Collection**: Topic trigger counts, execution times
- **Error Tracking**: Strategy errors and system issues
- **Debug Mode**: Verbose logging for development

### 3.5 Extensibility Features
- **Plugin System**: Architecture for adding new strategy languages
- **Custom System Topics**: Framework for adding new system topic types
- **API Endpoints**: REST API for external integrations
- **Webhook Support**: HTTP callbacks for external notifications

### 3.6 Deployment Considerations
- **Docker Support**: Containerized deployment
- **Service Management**: Systemd service files
- **Backup Strategy**: Database and configuration backup procedures
- **Update Mechanism**: Safe system updates without losing state

### 3.7 Testing Strategy
- **Unit Tests**: All core components
- **Integration Tests**: MQTT and database interactions
- **Strategy Testing**: Test harness for strategy development
- **Load Testing**: Performance under high message volumes

### 3.8 Documentation Requirements
- **User Manual**: Web UI usage guide
- **Strategy Development Guide**: How to write effective strategies
- **API Documentation**: For external integrations
- **Deployment Guide**: Installation and configuration instructions

### 3.9 Migration and Import/Export
- **Configuration Export**: Backup all topics and strategies
- **Configuration Import**: Restore from backup
- **Migration Tools**: Upgrade between versions
- **Legacy System Integration**: Import from other automation systems