// Package web provides HTTP handlers and web interface for the home automation system.
package web

import (
	"encoding/json"
	"net/http"
)

// API Endpoints
func (s *Server) handleAPITopics(w http.ResponseWriter, r *http.Request) {
	topics := s.topicManager.ListTopics()

	response := make(map[string]any)
	for name, topic := range topics {
		response[name] = map[string]any{
			"type":         topic.Type(),
			"last_value":   topic.LastValue(),
			"last_updated": topic.LastUpdated(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := s.strategyEngine.ListStrategies()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(strategies)
}

func (s *Server) handleAPISystemStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"mqtt_status":    s.getSystemStatus(),
		"topic_count":    s.topicManager.GetTopicCount(),
		"strategy_count": s.strategyEngine.GetStrategyCount(),
		"uptime":         "Unknown", // TODO: Implement uptime tracking
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

