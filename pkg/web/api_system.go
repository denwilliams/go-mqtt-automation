package web

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
)

// System API handlers

func (s *Server) handleAPISystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get MQTT connection status
	mqttConnected := false
	if s.mqttClient != nil {
		mqttConnected = s.mqttClient.IsConnected()
	}

	response := SystemInfoResponse{
		Version:       "1.0.0",             // TODO: Get from build info
		Uptime:        "0m",                // TODO: Calculate actual uptime
		BuildDate:     "unknown",           // TODO: Get from build info
		GoVersion:     runtime.Version(),
		DatabaseType:  "sqlite",            // TODO: Get from config
		MQTTConnected: mqttConnected,
	}

	writeAPIResponse(w, response)
}

// System stats structures
type SystemStatsResponse struct {
	Topics     TopicStatsDetail     `json:"topics"`
	Strategies StrategyStatsDetail  `json:"strategies"`
	MQTT       MQTTStatsDetail      `json:"mqtt"`
	Database   DatabaseStatsDetail  `json:"database"`
}

type TopicStatsDetail struct {
	External int `json:"external"`
	Internal int `json:"internal"`
	System   int `json:"system"`
}

type StrategyStatsDetail struct {
	Total     int                    `json:"total"`
	Languages map[string]int         `json:"languages"`
}

type MQTTStatsDetail struct {
	MessagesProcessed   int64     `json:"messages_processed"`
	LastMessageTime     time.Time `json:"last_message_time"`
	ConnectionUptime    string    `json:"connection_uptime"`
}

type DatabaseStatsDetail struct {
	SizeMB      float64 `json:"size_mb"`
	Connections int     `json:"connections"`
}

func (s *Server) handleAPISystemStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get topic counts
	topicCounts := s.topicManager.GetTopicCount()

	// Get strategy stats
	allStrategies := s.strategyEngine.ListStrategies()
	languages := make(map[string]int)
	for _, strat := range allStrategies {
		languages[strat.Language]++
	}

	response := SystemStatsResponse{
		Topics: TopicStatsDetail{
			External: topicCounts[topics.TopicTypeExternal],
			Internal: topicCounts[topics.TopicTypeInternal],
			System:   topicCounts[topics.TopicTypeSystem],
		},
		Strategies: StrategyStatsDetail{
			Total:     len(allStrategies),
			Languages: languages,
		},
		MQTT: MQTTStatsDetail{
			MessagesProcessed: 0,           // TODO: Track messages
			LastMessageTime:   time.Now(),  // TODO: Track last message
			ConnectionUptime:  "0m",        // TODO: Track connection uptime
		},
		Database: DatabaseStatsDetail{
			SizeMB:      0.0, // TODO: Calculate database size
			Connections: 1,   // TODO: Track actual connections
		},
	}

	writeAPIResponse(w, response)
}

// Activity structures
type ActivityResponse struct {
	Activities []ActivityItem `json:"activities"`
}

type ActivityItem struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Topic     string    `json:"topic,omitempty"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
}

func (s *Server) handleAPISystemActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// For now, return empty activities
	// TODO: Implement actual activity logging and retrieval
	response := ActivityResponse{
		Activities: []ActivityItem{
			{
				Timestamp: time.Now().Add(-5 * time.Minute),
				Type:      "system_start",
				Message:   "System started successfully",
				Level:     "info",
			},
			{
				Timestamp: time.Now().Add(-2 * time.Minute),
				Type:      "mqtt_connect",
				Message:   "Connected to MQTT broker",
				Level:     "info",
			},
		},
	}

	writeAPIResponse(w, response)
}

// Combined system endpoint structure
type CombinedSystemResponse struct {
	Info interface{}    `json:"info"`
	Logs []ActivityItem `json:"logs"`
}

// handleAPISystem combines system info, stats, and activity for the admin UI
func (s *Server) handleAPISystem(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get MQTT connection status
	mqttConnected := false
	if s.mqttClient != nil {
		mqttConnected = s.mqttClient.IsConnected()
	}

	// Get topic counts
	topicCounts := s.topicManager.GetTopicCount()

	// Get strategy stats
	allStrategies := s.strategyEngine.ListStrategies()

	// Get current PID
	pid := os.Getpid()

	// Create extended system info that matches React component interface
	extendedInfo := map[string]interface{}{
		"system": map[string]interface{}{
			"version":      "1.0.0",             // TODO: Get from build info
			"uptime":       "0m",                // TODO: Calculate actual uptime
			"status":       "healthy",
			"pid":          pid,
			"memory_usage": "0 MB", // TODO: Calculate actual memory usage
			"goroutines":   runtime.NumGoroutine(),
		},
		"database": map[string]interface{}{
			"type":             "sqlite", // TODO: Get from config
			"status":           "connected",
			"total_topics":     topicCounts[topics.TopicTypeExternal] + topicCounts[topics.TopicTypeInternal] + topicCounts[topics.TopicTypeSystem],
			"total_strategies": len(allStrategies),
		},
		"mqtt": map[string]interface{}{
			"broker_url":         "localhost:1883", // TODO: Get from config
			"connected":          mqttConnected,
			"messages_processed": 0,                // TODO: Track actual messages
			"subscriptions":      0,                // TODO: Track actual subscriptions
		},
		"performance": map[string]interface{}{
			"cpu_usage":    "0%",    // TODO: Calculate actual CPU usage
			"memory_usage": "0 MB",  // TODO: Calculate actual memory usage
			"disk_usage":   "0 GB",  // TODO: Calculate actual disk usage
			"network_io": map[string]interface{}{
				"bytes_sent":     0, // TODO: Track actual network I/O
				"bytes_received": 0,
			},
		},
	}

	// Mock logs for now
	logs := []ActivityItem{
		{
			Timestamp: time.Now().Add(-5 * time.Minute),
			Type:      "info",
			Message:   "System started successfully",
			Level:     "info",
		},
		{
			Timestamp: time.Now().Add(-2 * time.Minute),
			Type:      "info",
			Message:   "Connected to MQTT broker",
			Level:     "info",
		},
		{
			Timestamp: time.Now().Add(-1 * time.Minute),
			Type:      "info",
			Message:   "All strategies loaded successfully",
			Level:     "info",
		},
	}

	response := CombinedSystemResponse{
		Info: extendedInfo,
		Logs: logs,
	}

	writeAPIResponse(w, response)
}