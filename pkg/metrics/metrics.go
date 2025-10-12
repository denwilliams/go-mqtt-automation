package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Topic processing metrics
	TopicsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_topics_processed_total",
			Help: "Total number of topics processed",
		},
		[]string{"strategy", "topic_type"},
	)

	TopicProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "automation_topic_processing_duration_seconds",
			Help:    "Time taken to process a topic",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"strategy", "topic_type"},
	)

	TopicProcessingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_topic_processing_errors_total",
			Help: "Total number of topic processing errors",
		},
		[]string{"strategy", "error_type"},
	)

	// Database metrics
	DatabaseQueries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "mode"}, // mode: read, write
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "automation_database_query_duration_seconds",
			Help:    "Time taken for database queries",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // 0.1ms to ~400ms
		},
		[]string{"operation", "mode"},
	)

	DatabaseErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_database_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation"},
	)

	// MQTT metrics
	MQTTMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_mqtt_messages_published_total",
			Help: "Total number of MQTT messages published",
		},
		[]string{"topic"},
	)

	MQTTMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_mqtt_messages_received_total",
			Help: "Total number of MQTT messages received",
		},
		[]string{"topic"},
	)

	MQTTPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "automation_mqtt_publish_duration_seconds",
			Help:    "Time taken to publish MQTT messages",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"topic"},
	)

	MQTTPublishErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_mqtt_publish_errors_total",
			Help: "Total number of MQTT publish errors",
		},
		[]string{"topic"},
	)

	MQTTConnectionState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "automation_mqtt_connection_state",
			Help: "MQTT connection state (1=connected, 0=disconnected)",
		},
		[]string{"broker"},
	)

	// Strategy execution metrics
	StrategyExecutions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_strategy_executions_total",
			Help: "Total number of strategy executions",
		},
		[]string{"strategy_id", "language"},
	)

	StrategyExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "automation_strategy_execution_duration_seconds",
			Help:    "Time taken to execute strategies",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // 0.1ms to ~400ms
		},
		[]string{"strategy_id", "language"},
	)

	StrategyExecutionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "automation_strategy_execution_errors_total",
			Help: "Total number of strategy execution errors",
		},
		[]string{"strategy_id", "error_type"},
	)

	// System metrics
	ActiveTopics = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "automation_active_topics_total",
			Help: "Current number of active topics",
		},
		[]string{"topic_type"}, // external, internal, system
	)

	ActiveStrategies = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "automation_active_strategies_total",
			Help: "Current number of registered strategies",
		},
	)

	// Topic chain metrics
	TopicChainDepth = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "automation_topic_chain_depth",
			Help:    "Depth of topic dependency chains",
			Buckets: []float64{1, 2, 3, 4, 5, 10, 15, 20},
		},
		[]string{"root_topic"},
	)

	TopicChainLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "automation_topic_chain_latency_seconds",
			Help:    "End-to-end latency for topic chains",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s
		},
		[]string{"root_topic", "depth"},
	)
)

// Helper functions for common metric operations

// RecordTopicProcessed records a topic processing event
func RecordTopicProcessed(strategy, topicType string, duration float64) {
	TopicsProcessed.WithLabelValues(strategy, topicType).Inc()
	TopicProcessingDuration.WithLabelValues(strategy, topicType).Observe(duration)
}

// RecordTopicProcessingError records a topic processing error
func RecordTopicProcessingError(strategy, errorType string) {
	TopicProcessingErrors.WithLabelValues(strategy, errorType).Inc()
}

// RecordDatabaseQuery records a database query
func RecordDatabaseQuery(operation, mode string, duration float64) {
	DatabaseQueries.WithLabelValues(operation, mode).Inc()
	DatabaseQueryDuration.WithLabelValues(operation, mode).Observe(duration)
}

// RecordDatabaseError records a database error
func RecordDatabaseError(operation string) {
	DatabaseErrors.WithLabelValues(operation).Inc()
}

// RecordMQTTPublish records an MQTT publish event
func RecordMQTTPublish(topic string, duration float64) {
	MQTTMessagesPublished.WithLabelValues(topic).Inc()
	MQTTPublishDuration.WithLabelValues(topic).Observe(duration)
}

// RecordMQTTReceive records an MQTT receive event
func RecordMQTTReceive(topic string) {
	MQTTMessagesReceived.WithLabelValues(topic).Inc()
}

// RecordMQTTPublishError records an MQTT publish error
func RecordMQTTPublishError(topic string) {
	MQTTPublishErrors.WithLabelValues(topic).Inc()
}

// SetMQTTConnectionState sets the MQTT connection state
func SetMQTTConnectionState(broker string, connected bool) {
	state := 0.0
	if connected {
		state = 1.0
	}
	MQTTConnectionState.WithLabelValues(broker).Set(state)
}

// RecordStrategyExecution records a strategy execution
func RecordStrategyExecution(strategyID, language string, duration float64) {
	StrategyExecutions.WithLabelValues(strategyID, language).Inc()
	StrategyExecutionDuration.WithLabelValues(strategyID, language).Observe(duration)
}

// RecordStrategyError records a strategy execution error
func RecordStrategyError(strategyID, errorType string) {
	StrategyExecutionErrors.WithLabelValues(strategyID, errorType).Inc()
}

// SetActiveTopics sets the current number of active topics
func SetActiveTopics(topicType string, count int) {
	ActiveTopics.WithLabelValues(topicType).Set(float64(count))
}

// SetActiveStrategies sets the current number of active strategies
func SetActiveStrategies(count int) {
	ActiveStrategies.Set(float64(count))
}

// RecordTopicChain records metrics for a topic chain
func RecordTopicChain(rootTopic string, depth int, latency float64) {
	TopicChainDepth.WithLabelValues(rootTopic).Observe(float64(depth))
	TopicChainLatency.WithLabelValues(rootTopic, string(rune(depth+'0'))).Observe(latency)
}
