# Testing Guide - MQTT Home Automation System

This guide helps you test the automation system with realistic scenarios and strategies.

## 🚀 Quick Test Setup

### Step 1: Start the System
```bash
# Option A: Docker (recommended)
make docker-run

# Option B: Local development
make dev
```

### Step 2: Install Python MQTT Library
```bash
pip install paho-mqtt
```

### Step 3: Run the MQTT Simulator
```bash
# Start publishing test data
python scripts/test-mqtt-simulator.py

# Or with custom settings
python scripts/test-mqtt-simulator.py --host localhost --port 1883 --duration 300
```

### Step 4: Access Web Interface
Open http://localhost:8080 and watch the dashboard come alive!

## 🧪 Available Test Strategies

### 1. **Value Monitor** (`test-value-monitor`)
**Purpose**: Basic logging and passthrough
- **Input**: Any single topic
- **Output**: Structured data with input topic, value, and timestamps
- **Use Case**: General monitoring and debugging

### 2. **Temperature Converter** (`test-temp-converter`)  
**Purpose**: Convert Celsius to Fahrenheit
- **Input**: `sensors/temperature/celsius`
- **Output**: Both Celsius and Fahrenheit values
- **Use Case**: Unit conversion and data transformation

### 3. **Threshold Detector** (`test-threshold-detector`)
**Purpose**: Alert when values exceed configurable thresholds
- **Input**: Any numeric sensor
- **Parameters**: `threshold` (default: 25), `name` (sensor description)
- **Output**: Value, threshold status, and alerts
- **Use Case**: Temperature alerts, pressure monitoring, etc.

### 4. **Average Calculator** (`test-average-calculator`)
**Purpose**: Calculate statistics across multiple sensors
- **Input**: Multiple numeric topics
- **Output**: Average, min, max, count, and source list
- **Use Case**: Multi-room temperature averaging, load balancing

### 5. **Motion Aggregator** (`test-motion-aggregator`)
**Purpose**: Combine motion sensors with timeout logic
- **Input**: Multiple motion sensor topics
- **Parameters**: `timeout` (minutes, default: 5)
- **Output**: Motion status, active zones, timeout handling
- **Use Case**: Security systems, occupancy detection

### 6. **State Toggle** (`test-state-toggle`)
**Purpose**: Toggle switch with persistent state
- **Input**: Any trigger topic
- **Output**: Current state, previous state, change detection
- **Use Case**: Light switches, mode toggles, on/off controls

### 7. **Smart Light Controller** (`test-smart-light`)
**Purpose**: Intelligent lighting based on motion, light level, and time
- **Input**: Motion sensor, light level sensor
- **Parameters**: `dark_threshold`, `night_start`, `night_end`
- **Output**: Light commands with brightness and reasoning
- **Use Case**: Automatic lighting systems, energy saving

## 🎯 Test Scenarios

### Scenario 1: Temperature Monitoring
1. **Watch Dashboard**: Monitor `sensors/temperature/*` topics
2. **View Processing**: Check `monitoring/temperature-status` for logged data
3. **See Conversion**: Watch `converted/temperature-fahrenheit` for C→F conversion
4. **Test Alerts**: Observe `alerts/high-temperature` when temp > 25°C

### Scenario 2: Motion Detection
1. **Multiple Sensors**: Monitor motion in living-room, kitchen, bedroom
2. **Aggregation**: Check `security/motion-status` for combined status
3. **Timeout Logic**: Wait 5 minutes after motion stops
4. **Smart Lights**: See automatic light control in `automation/smart-lights`

### Scenario 3: Multi-Sensor Averaging
1. **Individual Sensors**: Watch `test/temp-sensor-1`, `test/temp-sensor-2`, `test/temp-sensor-3`
2. **Average Calculation**: Monitor `test/sensor-average` for statistics
3. **House Average**: Check `calculated/average-temperature` for room averaging

### Scenario 4: Toggle Controls
1. **Button Press**: The simulator occasionally publishes to `test/toggle-button`
2. **State Change**: Watch `controls/toggle-switch` maintain persistent state
3. **Output Events**: See emitted events in `controls/toggle-output`

## 📊 Simulator Data Patterns

The MQTT simulator creates realistic test data:

### Temperature Sensors
- **Base Temperature**: ~22°C with gradual drift
- **Room Variations**: Kitchen +1°C, Outdoor ±4°C
- **Random Fluctuations**: ±0.5°C per cycle
- **Topics**: `sensors/temperature/{location}`

### Humidity Sensors  
- **Base Humidity**: ~45% with ±10% variation
- **Realistic Range**: 20-80%
- **Topics**: `sensors/humidity/{location}`

### Motion Sensors
- **Day Pattern**: 30% activity probability (6AM-10PM)
- **Night Pattern**: 10% activity probability
- **Automatic Timeout**: Sensors turn off after detection
- **Topics**: `sensors/motion/{location}`

### Light Levels
- **Day Cycle**: 70±20 lux (6AM-8PM)
- **Night Cycle**: 10±5 lux (10PM-6AM)
- **Dawn/Dusk**: 40±15 lux (transition periods)
- **Topics**: `sensors/light`

## 🔧 Manual Testing

### Test Individual Strategies
1. **Go to Strategies**: http://localhost:8080/strategies
2. **Create New Strategy**: Try the examples below
3. **Create Topics**: Link strategies to input topics
4. **Publish Test Data**: Use MQTT client or simulator

### Simple Test Strategy Example
```javascript
function process(context) {
    const value = context.inputs[context.triggeringTopic];
    context.log(`Received: ${value}`);
    
    return {
        original: value,
        doubled: value * 2,
        processed_at: context.getISO()
    };
}
```

### MQTT Manual Testing
```bash
# Install mosquitto-clients
apt-get install mosquitto-clients  # Ubuntu/Debian
brew install mosquitto             # macOS

# Publish test data
mosquitto_pub -h localhost -t "sensors/temperature/test" -m "25.5"
mosquitto_pub -h localhost -t "sensors/motion/test" -m "true"

# Subscribe to outputs  
mosquitto_sub -h localhost -t "monitoring/#"
mosquitto_sub -h localhost -t "alerts/#"
```

## 📈 Monitoring and Debugging

### Web Interface Monitoring
- **Dashboard**: Real-time topic values and system status
- **Topics Page**: All active topics with last values and timestamps  
- **Strategies Page**: Strategy code and execution status
- **Logs Page**: System logs and strategy execution logs

### Key Metrics to Watch
- **Topic Count**: Should increase as simulator runs
- **MQTT Status**: Should show "Connected"
- **Strategy Executions**: Check logs for processing messages
- **Value Changes**: Topics should update every 5 seconds

### Common Issues
1. **No Data**: Check MQTT connection and simulator
2. **Strategy Errors**: Check JavaScript syntax and logic
3. **Missing Topics**: Verify input topic names match simulator
4. **No Processing**: Ensure topics have valid strategy assignments

## 🎮 Interactive Testing

### Live Strategy Development
1. **Create Topic**: Start with simple passthrough strategy
2. **Test with Simulator**: Watch real-time processing
3. **Modify Strategy**: Update logic and see immediate effects
4. **Add Complexity**: Gradually build sophisticated automation

### Strategy Testing Workflow
```bash
# 1. Start system and simulator
make docker-run
python scripts/test-mqtt-simulator.py

# 2. Open web interface
open http://localhost:8080

# 3. Create and test strategies iteratively
# 4. Monitor logs and outputs
# 5. Refine and improve
```

## 🏆 Advanced Testing

### Custom Scenarios
Modify the simulator script to create specific test cases:
- **Sensor Failures**: Publish `null` or invalid values
- **Rapid Changes**: Reduce sleep intervals for stress testing  
- **Edge Cases**: Test boundary conditions and error handling
- **Load Testing**: Increase topic count and frequency

### Integration Testing  
- **MQTT Reconnection**: Stop/start broker during operation
- **Database Recovery**: Restart system and verify state persistence
- **Strategy Updates**: Modify strategies during operation
- **Resource Usage**: Monitor memory and CPU under load

This testing framework provides comprehensive validation of the automation system's capabilities and helps you develop robust, reliable automation strategies.