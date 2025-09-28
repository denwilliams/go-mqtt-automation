# Admin API Design for MQTT Home Automation System

## Base Path: `/api/v1`

## Authentication
- For now, we'll implement without authentication (internal admin interface)
- Future: JWT tokens or session-based auth

## Content Type
- All requests: `application/json`
- All responses: `application/json`

## API Endpoints

### Dashboard / Overview
```
GET /api/v1/dashboard
Response: {
  "system": {
    "uptime": "2h 30m",
    "version": "1.0.0",
    "status": "healthy"
  },
  "stats": {
    "topics": {
      "external": 15,
      "internal": 8,
      "system": 5,
      "total": 28
    },
    "strategies": {
      "total": 12,
      "active": 10,
      "failed": 2
    },
    "mqtt": {
      "connected": true,
      "messages_processed": 1245,
      "last_message": "2023-12-01T10:30:00Z"
    }
  }
}
```

### Topics API

#### List all topics
```
GET /api/v1/topics?type=internal|external|system&page=1&limit=50
Response: {
  "topics": [
    {
      "name": "lights/living_room",
      "type": "internal",
      "last_value": true,
      "last_updated": "2023-12-01T10:30:00Z",
      "inputs": ["motion/living_room", "schedule/evening"],
      "strategy_id": "lighting-automation",
      "emit_to_mqtt": true
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 28,
    "pages": 1
  }
}
```

#### Get single topic
```
GET /api/v1/topics/{name}
Response: {
  "name": "lights/living_room",
  "type": "internal",
  "last_value": true,
  "last_updated": "2023-12-01T10:30:00Z",
  "created_at": "2023-11-15T09:00:00Z",
  "inputs": ["motion/living_room", "schedule/evening"],
  "input_names": {
    "motion/living_room": "Motion Sensor",
    "schedule/evening": "Evening Schedule"
  },
  "strategy_id": "lighting-automation",
  "emit_to_mqtt": true,
  "noop_unchanged": true,
  "config": {}
}
```

#### Create topic
```
POST /api/v1/topics
Body: {
  "name": "lights/bedroom",
  "type": "internal",
  "inputs": ["motion/bedroom", "schedule/bedtime"],
  "input_names": {
    "motion/bedroom": "Bedroom Motion",
    "schedule/bedtime": "Bedtime Schedule"
  },
  "strategy_id": "lighting-automation",
  "emit_to_mqtt": true,
  "noop_unchanged": false
}
Response: 201 Created
```

#### Update topic
```
PUT /api/v1/topics/{name}
Body: { /* same as create */ }
Response: 200 OK
```

#### Delete topic
```
DELETE /api/v1/topics/{name}
Response: 204 No Content
```

#### Get topic history/logs
```
GET /api/v1/topics/{name}/history?limit=100
Response: {
  "executions": [
    {
      "timestamp": "2023-12-01T10:30:00Z",
      "trigger_topic": "motion/living_room",
      "input_values": {
        "motion/living_room": true,
        "schedule/evening": false
      },
      "output_value": true,
      "execution_time_ms": 45,
      "error": null
    }
  ]
}
```

### Strategies API

#### List strategies
```
GET /api/v1/strategies?page=1&limit=50
Response: {
  "strategies": [
    {
      "id": "lighting-automation",
      "name": "Smart Lighting",
      "language": "javascript",
      "created_at": "2023-11-15T09:00:00Z",
      "updated_at": "2023-11-20T14:30:00Z",
      "max_inputs": 5,
      "default_input_names": ["motion", "schedule", "manual_override"]
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 12,
    "pages": 1
  }
}
```

#### Get single strategy
```
GET /api/v1/strategies/{id}
Response: {
  "id": "lighting-automation",
  "name": "Smart Lighting",
  "code": "function process(context) { ... }",
  "language": "javascript",
  "parameters": {
    "default_brightness": 80,
    "motion_timeout": 300
  },
  "max_inputs": 5,
  "default_input_names": ["motion", "schedule", "manual_override"],
  "created_at": "2023-11-15T09:00:00Z",
  "updated_at": "2023-11-20T14:30:00Z"
}
```

#### Create strategy
```
POST /api/v1/strategies
Body: {
  "id": "new-automation",
  "name": "New Automation",
  "code": "function process(context) { return true; }",
  "language": "javascript",
  "parameters": {},
  "max_inputs": 3,
  "default_input_names": ["input1", "input2"]
}
Response: 201 Created
```

#### Update strategy
```
PUT /api/v1/strategies/{id}
Body: { /* same as create */ }
Response: 200 OK
```

#### Delete strategy
```
DELETE /api/v1/strategies/{id}
Response: 204 No Content
```

#### Test strategy
```
POST /api/v1/strategies/{id}/test
Body: {
  "inputs": {
    "motion": true,
    "schedule": false
  },
  "parameters": {
    "brightness": 75
  }
}
Response: {
  "result": true,
  "log_messages": ["Motion detected", "Lights turned on"],
  "emitted_events": [
    {
      "topic": "/brightness",
      "value": 75
    }
  ],
  "execution_time_ms": 12,
  "error": null
}
```

### System API

#### System info
```
GET /api/v1/system/info
Response: {
  "version": "1.0.0",
  "uptime": "2h 30m 45s",
  "build_date": "2023-12-01T08:00:00Z",
  "go_version": "go1.21.0",
  "database_type": "sqlite",
  "mqtt_connected": true
}
```

#### System stats
```
GET /api/v1/system/stats
Response: {
  "topics": {
    "external": 15,
    "internal": 8,
    "system": 5
  },
  "strategies": {
    "total": 12,
    "languages": {
      "javascript": 12
    }
  },
  "mqtt": {
    "messages_processed": 1245,
    "last_message_time": "2023-12-01T10:30:00Z",
    "connection_uptime": "2h 25m"
  },
  "database": {
    "size_mb": 2.5,
    "connections": 1
  }
}
```

#### Recent activity
```
GET /api/v1/system/activity?limit=20
Response: {
  "activities": [
    {
      "timestamp": "2023-12-01T10:30:00Z",
      "type": "topic_execution",
      "topic": "lights/living_room",
      "message": "Topic executed successfully",
      "level": "info"
    },
    {
      "timestamp": "2023-12-01T10:29:45Z",
      "type": "mqtt_message",
      "topic": "motion/living_room",
      "message": "MQTT message received",
      "level": "debug"
    }
  ]
}
```

### Real-time API (WebSocket)

#### WebSocket endpoint for live updates
```
WS /api/v1/ws
Messages:
{
  "type": "topic_update",
  "data": {
    "topic": "motion/living_room",
    "value": true,
    "timestamp": "2023-12-01T10:30:00Z"
  }
}

{
  "type": "strategy_execution",
  "data": {
    "topic": "lights/living_room",
    "strategy": "lighting-automation",
    "result": true,
    "execution_time_ms": 45
  }
}

{
  "type": "system_event",
  "data": {
    "event": "mqtt_connected",
    "timestamp": "2023-12-01T10:30:00Z"
  }
}
```

## Error Responses

Standard HTTP error codes with JSON responses:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid topic name format",
    "details": {
      "field": "name",
      "reason": "Topic name must not contain spaces"
    }
  }
}
```

Error Codes:
- `VALIDATION_ERROR` - Input validation failed
- `NOT_FOUND` - Resource not found
- `DUPLICATE_RESOURCE` - Resource already exists
- `STRATEGY_EXECUTION_ERROR` - Strategy failed to execute
- `DATABASE_ERROR` - Database operation failed
- `MQTT_ERROR` - MQTT-related error

## Rate Limiting
- 100 requests per minute per IP for GET requests
- 20 requests per minute per IP for POST/PUT/DELETE requests

## Response Formats

### Success Response
```json
{
  "success": true,
  "data": { /* response data */ }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message"
  }
}
```