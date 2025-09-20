// Package web provides HTTP handlers and web interface for the home automation system.
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
)

// Dashboard
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Safely get topics with null checks
	var topicsMap map[string]topics.Topic
	if s.topicManager != nil {
		topicsMap = s.topicManager.ListTopics()
	}
	if topicsMap == nil {
		topicsMap = make(map[string]topics.Topic)
	}

	// Safely get topic counts
	var topicCounts map[topics.TopicType]int
	if s.topicManager != nil {
		topicCounts = s.topicManager.GetTopicCount()
	}
	if topicCounts == nil {
		topicCounts = make(map[topics.TopicType]int)
	}

	// Safely get strategy count
	var strategyCount int
	if s.strategyEngine != nil {
		strategyCount = s.strategyEngine.GetStrategyCount()
	}

	// Safely get system status
	systemStatus := s.getSystemStatus()
	if systemStatus == "" {
		systemStatus = "Unknown"
	}

	data := DashboardData{
		PageData: PageData{
			Title: "Home Automation Dashboard",
		},
		Topics:        topicsMap,
		TopicCounts:   topicCounts,
		StrategyCount: strategyCount,
		SystemStatus:  systemStatus,
		RecentLogs:    []string{"System started", "Web UI loaded"}, // TODO: Implement real logs
	}

	// Log the data for debugging
	if s.logger != nil {
		s.logger.Printf("Dashboard data: Topics=%d, Strategies=%d, Status=%s",
			len(data.Topics), data.StrategyCount, data.SystemStatus)
	}

	s.renderTemplate(w, "dashboard.html", data)
}

// Topics
func (s *Server) handleTopicsList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	allTopics := s.topicManager.ListTopics()

	// Apply filter if specified
	filteredTopics := make(map[string]topics.Topic)
	if filter != "" {
		for name, topic := range allTopics {
			if string(topic.Type()) == filter {
				filteredTopics[name] = topic
			}
		}
	} else {
		filteredTopics = allTopics
	}

	data := TopicsListData{
		PageData: PageData{
			Title: "Topics",
		},
		Topics:      filteredTopics,
		TopicFilter: filter,
	}

	s.renderTemplate(w, "topics.html", data)
}

func (s *Server) handleTopicNew(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		strategies := s.strategyEngine.ListStrategies()
		strategyList := make([]strategy.Strategy, 0, len(strategies))
		for _, strategy := range strategies {
			strategyList = append(strategyList, *strategy)
		}

		data := TopicEditData{
			PageData: PageData{
				Title: "Create New Topic",
			},
			Topic:      nil,
			Strategies: strategyList,
			IsNew:      true,
		}

		s.renderTemplate(w, "topic_edit.html", data)
		return
	}

	if r.Method == "POST" {
		s.handleTopicCreate(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleTopicEdit(w http.ResponseWriter, r *http.Request) {
	// Extract topic name from URL
	path := strings.TrimPrefix(r.URL.Path, "/topics/edit/")
	topicName := strings.TrimSuffix(path, "/")

	if topicName == "" {
		http.Error(w, "Topic name required", http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		topic := s.topicManager.GetTopic(topicName)
		if topic == nil {
			http.Error(w, "Topic not found", http.StatusNotFound)
			return
		}

		strategies := s.strategyEngine.ListStrategies()
		strategyList := make([]strategy.Strategy, 0, len(strategies))
		for _, strategy := range strategies {
			strategyList = append(strategyList, *strategy)
		}

		data := TopicEditData{
			PageData: PageData{
				Title: fmt.Sprintf("Edit Topic: %s", topicName),
			},
			Topic:      topic,
			Strategies: strategyList,
			IsNew:      false,
		}

		s.renderTemplate(w, "topic_edit.html", data)
		return
	}

	if r.Method == "POST" {
		s.handleTopicUpdate(w, r, topicName)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleTopicCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	inputs := strings.Split(r.FormValue("inputs"), "\n")
	strategyID := r.FormValue("strategy_id")
	emitToMQTT := r.FormValue("emit_to_mqtt") == "on"
	noOpUnchanged := r.FormValue("noop_unchanged") == "on"

	// Clean up inputs
	cleanInputs := make([]string, 0)
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input != "" {
			cleanInputs = append(cleanInputs, input)
		}
	}

	_, err := s.topicManager.AddInternalTopic(name, cleanInputs, strategyID)
	if err != nil {
		s.logger.Printf("Failed to create topic: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create topic: %v", err), http.StatusBadRequest)
		return
	}

	// Update topic settings
	topic := s.topicManager.GetInternalTopic(name)
	if topic != nil {
		topic.SetEmitToMQTT(emitToMQTT)
		topic.SetNoOpUnchanged(noOpUnchanged)

		// Save to database
		config := topic.GetConfig()
		if err := s.stateManager.SaveTopicConfig(config); err != nil {
			s.logger.Printf("Failed to save topic config: %v", err)
		}
	}

	http.Redirect(w, r, "/topics", http.StatusSeeOther)
}

func (s *Server) handleTopicUpdate(w http.ResponseWriter, r *http.Request, topicName string) {
	// Similar to handleTopicCreate but for updates
	// Implementation would update existing topic configuration
	http.Redirect(w, r, "/topics", http.StatusSeeOther)
}

func (s *Server) handleTopicDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract topic name from URL
	path := strings.TrimPrefix(r.URL.Path, "/topics/delete/")
	topicName := strings.TrimSuffix(path, "/")

	if topicName == "" {
		http.Error(w, "Topic name required", http.StatusBadRequest)
		return
	}

	if err := s.topicManager.RemoveTopic(topicName); err != nil {
		s.logger.Printf("Failed to delete topic: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete topic: %v", err), http.StatusBadRequest)
		return
	}

	// Delete from database
	if err := s.stateManager.DeleteTopicConfig(topicName); err != nil {
		s.logger.Printf("Failed to delete topic from database: %v", err)
	}

	http.Redirect(w, r, "/topics", http.StatusSeeOther)
}

// Strategies
func (s *Server) handleStrategiesList(w http.ResponseWriter, r *http.Request) {
	data := StrategiesListData{
		PageData: PageData{
			Title: "Strategies",
		},
		Strategies: s.strategyEngine.ListStrategies(),
	}

	s.renderTemplate(w, "strategies.html", data)
}

func (s *Server) handleStrategyNew(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := StrategyEditData{
			PageData: PageData{
				Title: "Create New Strategy",
			},
			Strategy: &strategy.Strategy{
				Language:   "javascript",
				Parameters: make(map[string]interface{}),
			},
			IsNew: true,
		}

		s.renderTemplate(w, "strategy_edit.html", data)
		return
	}

	if r.Method == "POST" {
		s.handleStrategyCreate(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleStrategyEdit(w http.ResponseWriter, r *http.Request) {
	// Extract strategy ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/strategies/edit/")
	strategyID := strings.TrimSuffix(path, "/")

	if strategyID == "" {
		http.Error(w, "Strategy ID required", http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		strategy, err := s.strategyEngine.GetStrategy(strategyID)
		if err != nil {
			http.Error(w, "Strategy not found", http.StatusNotFound)
			return
		}

		data := StrategyEditData{
			PageData: PageData{
				Title: fmt.Sprintf("Edit Strategy: %s", strategy.Name),
			},
			Strategy: strategy,
			IsNew:    false,
		}

		s.renderTemplate(w, "strategy_edit.html", data)
		return
	}

	if r.Method == "POST" {
		s.handleStrategyUpdate(w, r, strategyID)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleStrategyCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	strat := &strategy.Strategy{
		ID:         fmt.Sprintf("strategy_%d", time.Now().Unix()),
		Name:       r.FormValue("name"),
		Code:       r.FormValue("code"),
		Language:   r.FormValue("language"),
		Parameters: make(map[string]interface{}),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Parse parameters JSON if provided
	if params := r.FormValue("parameters"); params != "" {
		if err := json.Unmarshal([]byte(params), &strat.Parameters); err != nil {
			http.Error(w, "Invalid parameters JSON", http.StatusBadRequest)
			return
		}
	}

	if err := s.strategyEngine.AddStrategy(strat); err != nil {
		s.logger.Printf("Failed to create strategy: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create strategy: %v", err), http.StatusBadRequest)
		return
	}

	// Save to database
	if err := s.stateManager.SaveStrategy(strat); err != nil {
		s.logger.Printf("Failed to save strategy to database: %v", err)
	}

	http.Redirect(w, r, "/strategies", http.StatusSeeOther)
}

func (s *Server) handleStrategyUpdate(w http.ResponseWriter, r *http.Request, strategyID string) {
	// Similar to handleStrategyCreate but for updates
	// Implementation would update existing strategy
	http.Redirect(w, r, "/strategies", http.StatusSeeOther)
}

func (s *Server) handleStrategyDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract strategy ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/strategies/delete/")
	strategyID := strings.TrimSuffix(path, "/")

	if strategyID == "" {
		http.Error(w, "Strategy ID required", http.StatusBadRequest)
		return
	}

	if err := s.strategyEngine.RemoveStrategy(strategyID); err != nil {
		s.logger.Printf("Failed to delete strategy: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete strategy: %v", err), http.StatusBadRequest)
		return
	}

	// Delete from database
	if err := s.stateManager.DeleteStrategy(strategyID); err != nil {
		s.logger.Printf("Failed to delete strategy from database: %v", err)
	}

	http.Redirect(w, r, "/strategies", http.StatusSeeOther)
}

// System
func (s *Server) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	data := SystemConfigData{
		PageData: PageData{
			Title: "System Configuration",
		},
		MQTTBroker:    s.config.MQTT.Broker,
		MQTTConnected: s.mqttClient.IsConnected(),
		DatabaseType:  s.config.Database.Type,
		DatabasePath:  s.config.Database.Connection,
		WebPort:       s.config.Web.Port,
		LogLevel:      s.config.Logging.Level,
	}

	s.renderTemplate(w, "system.html", data)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	topicName := r.URL.Query().Get("topic")
	maxEntriesStr := r.URL.Query().Get("max")

	maxEntries := 100
	if maxEntriesStr != "" {
		if parsed, err := strconv.Atoi(maxEntriesStr); err == nil && parsed > 0 {
			maxEntries = parsed
		}
	}

	// For now, return placeholder logs
	// In a real implementation, this would fetch from the execution logs
	logs := []string{
		"System started successfully",
		"MQTT client connected",
		"Strategy engine initialized",
		"Web server started on port " + strconv.Itoa(s.config.Web.Port),
	}

	data := LogsData{
		PageData: PageData{
			Title: "System Logs",
		},
		Logs:       logs,
		TopicName:  topicName,
		MaxEntries: maxEntries,
	}

	s.renderTemplate(w, "logs.html", data)
}

// API Endpoints
func (s *Server) handleAPITopics(w http.ResponseWriter, r *http.Request) {
	topics := s.topicManager.ListTopics()

	response := make(map[string]interface{})
	for name, topic := range topics {
		response[name] = map[string]interface{}{
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
	status := map[string]interface{}{
		"mqtt_status":    s.getSystemStatus(),
		"topic_count":    s.topicManager.GetTopicCount(),
		"strategy_count": s.strategyEngine.GetStrategyCount(),
		"uptime":         "Unknown", // TODO: Implement uptime tracking
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
