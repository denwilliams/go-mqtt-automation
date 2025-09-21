-- Initial schema and data for MQTT Home Automation System
-- Template for multiple database types

-- ============================================================================
-- SCHEMA CREATION
-- ============================================================================

-- Topics table: stores all topic configurations
CREATE TABLE IF NOT EXISTS topics (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('internal', 'external', 'system')),
    inputs TEXT, -- JSON array of input topic names
    input_names TEXT, -- JSON object mapping input names to topic paths
    strategy_id TEXT,
    emit_to_mqtt BOOLEAN DEFAULT false,
    noop_unchanged BOOLEAN DEFAULT false,
    last_value TEXT, -- JSON serialized value
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    config TEXT, -- JSON configuration
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (strategy_id) REFERENCES strategies(id)
);

-- Strategies table: stores JavaScript code and configuration
CREATE TABLE IF NOT EXISTS strategies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    language TEXT DEFAULT 'javascript' CHECK (language IN ('javascript', 'lua', 'go-template')),
    parameters TEXT, -- JSON
    max_inputs INTEGER DEFAULT NULL, -- NULL means unlimited
    default_input_names TEXT, -- JSON array of default input names
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- State table: stores arbitrary key-value state data
CREATE TABLE IF NOT EXISTS state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL, -- JSON serialized
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Execution log table: stores strategy execution history
CREATE TABLE IF NOT EXISTS execution_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    topic_name TEXT NOT NULL,
    strategy_id TEXT,
    trigger_topic TEXT,
    input_values TEXT, -- JSON
    output_values TEXT, -- JSON
    error_message TEXT,
    execution_time_ms INTEGER,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (topic_name) REFERENCES topics(name),
    FOREIGN KEY (strategy_id) REFERENCES strategies(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_topics_type ON topics(type);
CREATE INDEX IF NOT EXISTS idx_topics_updated ON topics(last_updated);
CREATE INDEX IF NOT EXISTS idx_strategies_name ON strategies(name);
CREATE INDEX IF NOT EXISTS idx_execution_log_topic ON execution_log(topic_name);
CREATE INDEX IF NOT EXISTS idx_execution_log_executed ON execution_log(executed_at);
CREATE INDEX IF NOT EXISTS idx_state_updated ON state(updated_at);

-- ============================================================================
-- SYSTEM TOPICS
-- ============================================================================

-- System event topics
INSERT INTO topics (name, type, emit_to_mqtt, config) VALUES
('system/events/startup', 'system', false, '{"description": "System startup event"}'),
('system/events/shutdown', 'system', false, '{"description": "System shutdown event"}'),
('system/events/error', 'system', false, '{"description": "System error event"}'),
('system/events/heartbeat', 'system', true, '{"description": "System heartbeat", "interval": "30s"}');

-- Default ticker topics
INSERT INTO topics (name, type, emit_to_mqtt, config) VALUES
('system/ticker/1s', 'system', false, '{"description": "1 second ticker", "interval": "1s"}'),
('system/ticker/5s', 'system', false, '{"description": "5 second ticker", "interval": "5s"}'),
('system/ticker/30s', 'system', false, '{"description": "30 second ticker", "interval": "30s"}'),
('system/ticker/1m', 'system', false, '{"description": "1 minute ticker", "interval": "1m"}'),
('system/ticker/5m', 'system', false, '{"description": "5 minute ticker", "interval": "5m"}'),
('system/ticker/15m', 'system', false, '{"description": "15 minute ticker", "interval": "15m"}'),
('system/ticker/30m', 'system', false, '{"description": "30 minute ticker", "interval": "30m"}'),
('system/ticker/1h', 'system', false, '{"description": "1 hour ticker", "interval": "1h"}');

-- Default scheduler topics (examples)
INSERT INTO topics (name, type, emit_to_mqtt, config) VALUES
('system/scheduler/daily-morning', 'system', false, '{"description": "Daily morning trigger", "cron": "0 8 * * *"}'),
('system/scheduler/daily-evening', 'system', false, '{"description": "Daily evening trigger", "cron": "0 20 * * *"}'),
('system/scheduler/weekly-maintenance', 'system', false, '{"description": "Weekly maintenance", "cron": "0 2 * * 0"}');

-- ============================================================================
-- BUILT-IN STRATEGIES
-- ============================================================================

-- Alias strategy - echoes value from one or more inputs
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'alias',
    'Alias',
    'function process(context) {
    // Emit the first non-null input value
    const keys = Object.keys(context.inputs);
    for (const key of keys) {
        const value = context.inputs[key];
        if (value !== null && value !== undefined) {
            context.emit(value);
            return;
        }
    }
    context.emit(null);
}',
    'javascript',
    '{}',
    NULL,
    NULL
);

-- Boolean strategy - true if truthy, false otherwise
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'bool',
    'Boolean Conversion',
    'function process(context) {
    const value = context.inputs.value;
    context.emit(Boolean(value));
}',
    'javascript',
    '{}',
    1,
    '["value"]'
);

-- Not strategy - logical NOT operation
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'not',
    'Logical NOT',
    'function process(context) {
    const value = context.inputs.value;
    context.emit(!Boolean(value));
}',
    'javascript',
    '{}',
    1,
    '["value"]'
);

-- Add strategy - sum all numeric inputs
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'add',
    'Addition',
    'function process(context) {
    let sum = 0;
    let hasNumeric = false;

    for (const key of Object.keys(context.inputs)) {
        const value = context.inputs[key];
        const num = Number(value);
        if (!isNaN(num)) {
            sum += num;
            hasNumeric = true;
        }
    }

    context.emit(hasNumeric ? sum : null);
}',
    'javascript',
    '{}',
    NULL,
    NULL
);

-- Subtract strategy - subtract second input from first
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'subtract',
    'Subtraction',
    'function process(context) {
    const minuend = Number(context.inputs.minuend);
    const subtrahend = Number(context.inputs.subtrahend);

    if (isNaN(minuend) || isNaN(subtrahend)) {
        context.emit(null);
        return;
    }

    context.emit(minuend - subtrahend);
}',
    'javascript',
    '{}',
    2,
    '["minuend", "subtrahend"]'
);

-- Multiply strategy - multiply all numeric inputs
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'multiply',
    'Multiplication',
    'function process(context) {
    let product = 1;
    let hasNumeric = false;

    for (const key of Object.keys(context.inputs)) {
        const value = context.inputs[key];
        const num = Number(value);
        if (!isNaN(num)) {
            product *= num;
            hasNumeric = true;
        }
    }

    context.emit(hasNumeric ? product : null);
}',
    'javascript',
    '{}',
    NULL,
    NULL
);

-- And strategy - true if all inputs are truthy
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'and',
    'Logical AND',
    'function process(context) {
    const keys = Object.keys(context.inputs);
    if (keys.length === 0) {
        context.emit(true);
        return;
    }

    for (const key of keys) {
        if (!Boolean(context.inputs[key])) {
            context.emit(false);
            return;
        }
    }

    context.emit(true);
}',
    'javascript',
    '{}',
    NULL,
    NULL
);

-- Or strategy - true if any input is truthy
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'or',
    'Logical OR',
    'function process(context) {
    const keys = Object.keys(context.inputs);
    if (keys.length === 0) {
        context.emit(false);
        return;
    }

    for (const key of keys) {
        if (Boolean(context.inputs[key])) {
            context.emit(true);
            return;
        }
    }

    context.emit(false);
}',
    'javascript',
    '{}',
    NULL,
    NULL
);

-- Pick strategy - returns the value of a named field from the input object
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'pick',
    'Pick Field',
    'function process(context) {
    const input = context.inputs.object;
    const fieldName = context.parameters.field;

    if (!fieldName) {
        context.log("Warning: No field parameter specified");
        context.emit(null);
        return;
    }

    if (typeof input === "object" && input !== null) {
        context.emit(input[fieldName] !== undefined ? input[fieldName] : null);
    } else {
        context.emit(null);
    }
}',
    'javascript',
    '{"field": ""}',
    1,
    '["object"]'
);

-- Inside strategy - true if value is between min and max (inclusive)
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'inside',
    'Inside Range',
    'function process(context) {
    const value = Number(context.inputs.value);
    const min = Number(context.parameters.min || 0);
    const max = Number(context.parameters.max || 100);

    if (isNaN(value)) {
        context.emit(false);
        return;
    }

    context.emit(value >= min && value <= max);
}',
    'javascript',
    '{"min": 0, "max": 100}',
    1,
    '["value"]'
);

-- Outside strategy - true if value is outside min and max range
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'outside',
    'Outside Range',
    'function process(context) {
    const value = Number(context.inputs.value);
    const min = Number(context.parameters.min || 0);
    const max = Number(context.parameters.max || 100);

    if (isNaN(value)) {
        context.emit(false);
        return;
    }

    context.emit(value < min || value > max);
}',
    'javascript',
    '{"min": 0, "max": 100}',
    1,
    '["value"]'
);

-- Toggle strategy - flip the value each event
INSERT INTO strategies (id, name, code, language, parameters, max_inputs, default_input_names) VALUES
(
    'toggle',
    'Toggle',
    'function process(context) {
    // Get the last output value, default to false
    const lastValue = context.lastOutputs.value || false;

    // Flip the boolean value
    const newValue = !Boolean(lastValue);

    context.emit({ value: newValue });
}',
    'javascript',
    '{}',
    NULL,
    NULL
);