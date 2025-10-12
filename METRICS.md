# Metrics Documentation

This document describes the Prometheus metrics exposed by the MQTT Home Automation system.

## Accessing Metrics

Metrics are exposed in Prometheus format at:

```
http://localhost:8080/metrics
```

You can scrape this endpoint with Prometheus or view it directly with curl:

```bash
curl http://localhost:8080/metrics
```

## Available Metrics

### Topic Processing Metrics

#### `automation_topics_processed_total`
**Type:** Counter
**Labels:**
- `strategy` - The strategy ID used to process the topic
- `topic_type` - The type of topic (e.g., "internal", "external", "system")

Total number of topics processed by the system. Increment each time a topic processes its inputs and executes its strategy.

**Example:**
```
automation_topics_processed_total{strategy="alias",topic_type="internal"} 150
automation_topics_processed_total{strategy="greater_than",topic_type="internal"} 42
```

#### `automation_topic_processing_duration_seconds`
**Type:** Histogram
**Labels:**
- `strategy` - The strategy ID
- `topic_type` - The type of topic

Time taken to process a topic (from input collection to emission). Helps identify slow strategies or processing bottlenecks.

**Buckets:** 1ms to ~1s (exponential)

**Example:**
```
automation_topic_processing_duration_seconds_bucket{strategy="alias",topic_type="internal",le="0.001"} 120
automation_topic_processing_duration_seconds_bucket{strategy="alias",topic_type="internal",le="0.002"} 145
```

#### `automation_topic_processing_errors_total`
**Type:** Counter
**Labels:**
- `strategy` - The strategy ID
- `error_type` - The type of error ("strategy_execution", "emit_events", etc.)

Total number of topic processing errors encountered.

### Database Metrics

#### `automation_database_queries_total`
**Type:** Counter
**Labels:**
- `operation` - The database operation (e.g., "save_topic_state", "load_topic_state")
- `mode` - Either "read" or "write"

Total number of database queries executed. Use this to understand database load and identify which operations are most frequent.

**Example:**
```
automation_database_queries_total{mode="write",operation="save_topic_state"} 1234
automation_database_queries_total{mode="write",operation="update_topic_last_value"} 1234
automation_database_queries_total{mode="read",operation="load_topic_state"} 456
```

#### `automation_database_query_duration_seconds`
**Type:** Histogram
**Labels:**
- `operation` - The database operation
- `mode` - Either "read" or "write"

Time taken for database queries. Helps identify slow queries and database performance issues.

**Buckets:** 0.1ms to ~400ms (exponential)

**Example:**
```
automation_database_query_duration_seconds_bucket{mode="write",operation="save_topic_state",le="0.001"} 1000
automation_database_query_duration_seconds_bucket{mode="write",operation="save_topic_state",le="0.002"} 1200
```

#### `automation_database_errors_total`
**Type:** Counter
**Labels:**
- `operation` - The database operation that failed

Total number of database errors encountered.

### MQTT Metrics

#### `automation_mqtt_messages_published_total`
**Type:** Counter
**Labels:**
- `topic` - The MQTT topic name

Total number of MQTT messages published by the system.

**Example:**
```
automation_mqtt_messages_published_total{topic="home/motion/living_room"} 567
automation_mqtt_messages_published_total{topic="home/lights/kitchen"} 234
```

#### `automation_mqtt_messages_received_total`
**Type:** Counter
**Labels:**
- `topic` - The MQTT topic name

Total number of MQTT messages received from the broker.

**Example:**
```
automation_mqtt_messages_received_total{topic="sensors/motion/garage"} 890
automation_mqtt_messages_received_total{topic="sensors/temperature/living_room"} 1234
```

#### `automation_mqtt_publish_duration_seconds`
**Type:** Histogram
**Labels:**
- `topic` - The MQTT topic name

Time taken to publish MQTT messages (includes serialization and network I/O).

**Buckets:** 1ms to ~1s (exponential)

#### `automation_mqtt_publish_errors_total`
**Type:** Counter
**Labels:**
- `topic` - The MQTT topic name

Total number of MQTT publish errors.

#### `automation_mqtt_connection_state`
**Type:** Gauge
**Labels:**
- `broker` - The MQTT broker address

Current MQTT connection state. Value is 1 when connected, 0 when disconnected.

**Example:**
```
automation_mqtt_connection_state{broker="mqtt://localhost:1883"} 1
```

### Strategy Execution Metrics

#### `automation_strategy_executions_total`
**Type:** Counter
**Labels:**
- `strategy_id` - The strategy ID
- `language` - The strategy language (e.g., "javascript")

Total number of strategy executions.

#### `automation_strategy_execution_duration_seconds`
**Type:** Histogram
**Labels:**
- `strategy_id` - The strategy ID
- `language` - The strategy language

Time taken to execute strategies. Helps identify slow strategies.

**Buckets:** 0.1ms to ~400ms (exponential)

#### `automation_strategy_execution_errors_total`
**Type:** Counter
**Labels:**
- `strategy_id` - The strategy ID
- `error_type` - The type of error

Total number of strategy execution errors.

### System Metrics

#### `automation_active_topics_total`
**Type:** Gauge
**Labels:**
- `topic_type` - The type of topic ("external", "internal", "system")

Current number of active topics in the system.

**Example:**
```
automation_active_topics_total{topic_type="external"} 25
automation_active_topics_total{topic_type="internal"} 50
automation_active_topics_total{topic_type="system"} 15
```

#### `automation_active_strategies_total`
**Type:** Gauge

Current number of registered strategies in the system.

### Topic Chain Metrics

#### `automation_topic_chain_depth`
**Type:** Histogram
**Labels:**
- `root_topic` - The root topic that started the chain

Depth of topic dependency chains (how many levels deep a topic chain goes).

**Buckets:** 1, 2, 3, 4, 5, 10, 15, 20

#### `automation_topic_chain_latency_seconds`
**Type:** Histogram
**Labels:**
- `root_topic` - The root topic that started the chain
- `depth` - The depth of the chain

End-to-end latency for topic chains (from root input to final output).

**Buckets:** 1ms to ~4s (exponential)

## Common Queries

### Prometheus Queries

**Database writes per second:**
```promql
rate(automation_database_queries_total{mode="write"}[1m])
```

**Topic processing rate by strategy:**
```promql
rate(automation_topics_processed_total[1m])
```

**99th percentile topic processing duration:**
```promql
histogram_quantile(0.99, rate(automation_topic_processing_duration_seconds_bucket[5m]))
```

**Database query 95th percentile latency:**
```promql
histogram_quantile(0.95, rate(automation_database_query_duration_seconds_bucket[5m]))
```

**MQTT publish error rate:**
```promql
rate(automation_mqtt_publish_errors_total[5m])
```

**Slowest strategies (by p95 execution time):**
```promql
topk(10, histogram_quantile(0.95, rate(automation_strategy_execution_duration_seconds_bucket[5m])))
```

**Most frequently processed topics:**
```promql
topk(10, rate(automation_topics_processed_total[5m]))
```

**Database write throughput:**
```promql
sum(rate(automation_database_queries_total{mode="write"}[1m]))
```

## Setting Up Prometheus

### Basic Prometheus Configuration

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'mqtt-automation'
    static_configs:
      - targets: ['localhost:8080']
```

### Running Prometheus

```bash
# Using Docker
docker run -p 9090:9090 -v ./prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus

# Access Prometheus UI
open http://localhost:9090
```

## Grafana Dashboard

You can create a Grafana dashboard to visualize these metrics. Key panels to include:

1. **Topic Processing Rate** - Graph of `automation_topics_processed_total` by strategy
2. **Database Load** - Graph of `automation_database_queries_total` split by mode (read/write)
3. **Topic Processing Latency** - Heatmap of `automation_topic_processing_duration_seconds`
4. **MQTT Traffic** - Graph of messages published/received per second
5. **Error Rates** - Single stat panels for各 error counters
6. **MQTT Connection Status** - Single stat showing `automation_mqtt_connection_state`
7. **Active Topics** - Gauge showing `automation_active_topics_total` by type

## Performance Monitoring

Use these metrics to identify performance bottlenecks:

1. **High topic processing latency:**
   - Check `automation_topic_processing_duration_seconds` histogram
   - Identify slow strategies
   - Look for sequential processing bottlenecks

2. **Database bottlenecks:**
   - Monitor `automation_database_queries_total` for excessive writes
   - Check `automation_database_query_duration_seconds` for slow queries
   - Consider batching if writes are high

3. **MQTT issues:**
   - Monitor `automation_mqtt_publish_errors_total` for connection problems
   - Check `automation_mqtt_connection_state` for disconnections
   - Review `automation_mqtt_publish_duration_seconds` for network latency

4. **Strategy performance:**
   - Identify slow strategies with `automation_strategy_execution_duration_seconds`
   - Check `automation_strategy_execution_errors_total` for failing strategies

## Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: mqtt_automation
    rules:
      - alert: HighDatabaseWriteRate
        expr: rate(automation_database_queries_total{mode="write"}[1m]) > 100
        for: 5m
        annotations:
          summary: "High database write rate detected"

      - alert: MQTTDisconnected
        expr: automation_mqtt_connection_state == 0
        for: 1m
        annotations:
          summary: "MQTT broker disconnected"

      - alert: HighErrorRate
        expr: rate(automation_topic_processing_errors_total[5m]) > 1
        for: 5m
        annotations:
          summary: "High topic processing error rate"

      - alert: SlowTopicProcessing
        expr: histogram_quantile(0.95, rate(automation_topic_processing_duration_seconds_bucket[5m])) > 0.5
        for: 10m
        annotations:
          summary: "Topic processing is slow (p95 > 500ms)"
```

## Troubleshooting

**Metrics not appearing:**
- Verify the server is running: `curl http://localhost:8080/metrics`
- Check Prometheus is configured to scrape the correct endpoint
- Ensure no firewall is blocking port 8080

**Missing labels:**
- Metrics only have labels after they've been recorded at least once
- Trigger some activity in the system to generate metrics

**High cardinality warnings:**
- The `topic` label on MQTT metrics can create high cardinality if you have many topics
- Consider using metric relabeling in Prometheus to aggregate or drop labels if needed
