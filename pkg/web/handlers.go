// Package web provides HTTP handlers and web interface for the home automation system.
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
)

// Pre-compiled regex for strategy ID validation
var strategyIDPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

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
		// Extract strategy ID from URL path
		path := strings.TrimPrefix(r.URL.Path, "/topics/new")
		path = strings.TrimPrefix(path, "/")
		strategyID := strings.TrimSuffix(path, "/")

		// If no strategy ID provided, show strategy selection page
		if strategyID == "" {
			strategies := s.strategyEngine.ListStrategies()
			strategyList := make([]strategy.Strategy, 0, len(strategies))
			for _, strategy := range strategies {
				strategyList = append(strategyList, *strategy)
			}

			data := struct {
				Title      string
				Error      string
				Success    string
				Strategies []strategy.Strategy
			}{
				Title:      "Create New Topic - Select Strategy",
				Error:      "",
				Success:    "",
				Strategies: strategyList,
			}

			s.renderTemplate(w, "topic_strategy_select.html", data)
			return
		}

		// Strategy ID provided, show topic creation form with pre-selected strategy
		strategies := s.strategyEngine.ListStrategies()
		strategyList := make([]strategy.Strategy, 0, len(strategies))
		for _, strategy := range strategies {
			strategyList = append(strategyList, *strategy)
		}

		// Verify the strategy exists
		selectedStrategy, err := s.strategyEngine.GetStrategy(strategyID)
		if err != nil {
			http.Error(w, "Strategy not found", http.StatusNotFound)
			return
		}

		data := TopicEditData{
			PageData: PageData{
				Title: "Create New Topic",
			},
			Topic:            nil,
			Strategies:       strategyList,
			SelectedStrategy: selectedStrategy,
			IsNew:            true,
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
		// Load topic from database to ensure we have the latest data
		configInterface, err := s.stateManager.LoadTopicConfig(topicName)
		if err != nil {
			s.logger.Printf("Failed to load topic config from database: %v", err)
			http.Error(w, "Failed to load topic config", http.StatusInternalServerError)
			return
		}
		if configInterface == nil {
			http.Error(w, "Topic not found", http.StatusNotFound)
			return
		}

		// Get the in-memory topic and sync it with database data
		topic := s.topicManager.GetTopic(topicName)
		if topic == nil {
			http.Error(w, "Topic not found in memory", http.StatusNotFound)
			return
		}

		// If it's an internal topic, update the in-memory config with database data
		if internalTopic := s.topicManager.GetInternalTopic(topicName); internalTopic != nil {
			if dbConfig, ok := configInterface.(topics.InternalTopicConfig); ok {
				internalTopic.UpdateConfig(dbConfig)
			}
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
	inputNamesJSON := r.FormValue("input_names")

	// Clean up inputs
	cleanInputs := make([]string, 0)
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input != "" {
			cleanInputs = append(cleanInputs, input)
		}
	}

	// Parse input names JSON
	var inputNames map[string]string
	if inputNamesJSON != "" && inputNamesJSON != "{}" {
		if err := json.Unmarshal([]byte(inputNamesJSON), &inputNames); err != nil {
			http.Error(w, "Invalid input names JSON", http.StatusBadRequest)
			return
		}

		// Validate that all input names keys exist in cleanInputs
		for topicPath := range inputNames {
			found := false
			for _, input := range cleanInputs {
				if input == topicPath {
					found = true
					break
				}
			}
			if !found {
				http.Error(w, fmt.Sprintf("Input name key '%s' does not match any input topic", topicPath), http.StatusBadRequest)
				return
			}
		}
	}

	// Create topic config first
	config := topics.InternalTopicConfig{
		BaseTopicConfig: topics.BaseTopicConfig{
			Name:        name,
			Type:        topics.TopicTypeInternal,
			CreatedAt:   time.Now(),
			LastUpdated: time.Now(),
			Config:      make(map[string]interface{}),
		},
		Inputs:        cleanInputs,
		InputNames:    inputNames,
		StrategyID:    strategyID,
		EmitToMQTT:    emitToMQTT,
		NoOpUnchanged: noOpUnchanged,
	}

	// Save to database FIRST (source of truth)
	if err := s.stateManager.SaveTopicConfig(config); err != nil {
		s.logger.Printf("Failed to save topic to database: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save topic to database: %v", err), http.StatusInternalServerError)
		return
	}

	// THEN create in-memory version by loading from database
	_, err := s.topicManager.AddInternalTopic(name, cleanInputs, strategyID)
	if err != nil {
		s.logger.Printf("Failed to create topic in memory: %v", err)
		// Try to reload from database instead
		if reloadErr := s.topicManager.ReloadTopicFromDatabase(name); reloadErr != nil {
			s.logger.Printf("Failed to reload topic from database: %v", reloadErr)
			http.Error(w, fmt.Sprintf("Topic saved to database but failed to load in memory: %v", reloadErr), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/topics", http.StatusSeeOther)
}

func (s *Server) handleTopicUpdate(w http.ResponseWriter, r *http.Request, topicName string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	inputs := strings.Split(r.FormValue("inputs"), "\n")
	strategyID := r.FormValue("strategy_id")
	emitToMQTT := r.FormValue("emit_to_mqtt") == "on"
	noOpUnchanged := r.FormValue("noop_unchanged") == "on"
	inputNamesJSON := r.FormValue("input_names")

	// Clean up inputs
	cleanInputs := make([]string, 0)
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input != "" {
			cleanInputs = append(cleanInputs, input)
		}
	}

	// Parse input names JSON
	var inputNames map[string]string
	if inputNamesJSON != "" && inputNamesJSON != "{}" {
		if err := json.Unmarshal([]byte(inputNamesJSON), &inputNames); err != nil {
			http.Error(w, fmt.Sprintf("Invalid input names JSON: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Get the existing topic
	topic := s.topicManager.GetInternalTopic(topicName)
	if topic == nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	// Get existing config and update only the changed fields
	config := topic.GetConfig()
	config.Inputs = cleanInputs
	config.InputNames = inputNames
	config.StrategyID = strategyID
	config.EmitToMQTT = emitToMQTT
	config.NoOpUnchanged = noOpUnchanged

	// Ensure the Type field is properly set
	config.Type = topics.TopicTypeInternal

	// Save to database FIRST (source of truth)
	if err := s.stateManager.SaveTopicConfig(config); err != nil {
		s.logger.Printf("Failed to save topic to database: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save topic to database: %v", err), http.StatusInternalServerError)
		return
	}

	// THEN reload in-memory version from database to sync
	if err := s.topicManager.ReloadTopicFromDatabase(topicName); err != nil {
		s.logger.Printf("Failed to reload topic from database: %v", err)
		// Continue anyway - database save succeeded, so this is just a warning
	}

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
		// Load strategy from database to ensure we have the latest data
		strategy, err := s.stateManager.LoadStrategy(strategyID)
		if err != nil {
			s.logger.Printf("Failed to load strategy from database: %v", err)
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

	// Get custom ID or generate one
	customID := strings.TrimSpace(r.FormValue("id"))
	var strategyID string

	if customID != "" {
		// Validate custom ID format
		if !isValidStrategyID(customID) {
			http.Error(w, "Invalid strategy ID. Use only lowercase letters, numbers, and underscores.", http.StatusBadRequest)
			return
		}

		// Check if ID already exists
		if _, err := s.strategyEngine.GetStrategy(customID); err == nil {
			http.Error(w, "Strategy ID already exists. Please choose a different ID.", http.StatusBadRequest)
			return
		}

		strategyID = customID
	} else {
		// Generate fallback ID
		strategyID = fmt.Sprintf("strategy_%d", time.Now().Unix())
	}

	strat := &strategy.Strategy{
		ID:         strategyID,
		Name:       r.FormValue("name"),
		Code:       r.FormValue("code"),
		Language:   r.FormValue("language"),
		Parameters: make(map[string]interface{}),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Parse max inputs
	if maxInputsStr := r.FormValue("max_inputs"); maxInputsStr != "" {
		if maxInputs, err := strconv.Atoi(maxInputsStr); err == nil {
			strat.MaxInputs = maxInputs
		}
	}

	// Parse default input names JSON if provided
	if defaultInputNamesJSON := r.FormValue("default_input_names"); defaultInputNamesJSON != "" {
		defaultInputNamesJSON = strings.TrimSpace(defaultInputNamesJSON)
		if defaultInputNamesJSON != "[]" && defaultInputNamesJSON != "null" {
			if err := json.Unmarshal([]byte(defaultInputNamesJSON), &strat.DefaultInputNames); err != nil {
				http.Error(w, "Invalid default input names JSON", http.StatusBadRequest)
				return
			}
		}
	}

	// Parse parameters JSON if provided
	if params := r.FormValue("parameters"); params != "" {
		if err := json.Unmarshal([]byte(params), &strat.Parameters); err != nil {
			http.Error(w, "Invalid parameters JSON", http.StatusBadRequest)
			return
		}
	}

	// Save to database FIRST (source of truth)
	if err := s.stateManager.SaveStrategy(strat); err != nil {
		s.logger.Printf("Failed to save strategy to database: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save strategy to database: %v", err), http.StatusInternalServerError)
		return
	}

	// THEN add to in-memory engine
	if err := s.strategyEngine.AddStrategy(strat); err != nil {
		s.logger.Printf("Failed to create strategy in memory: %v", err)
		// Try to reload from database instead
		if reloadErr := s.strategyEngine.ReloadStrategyFromDatabase(strat.ID, strat); reloadErr != nil {
			s.logger.Printf("Failed to reload strategy from database: %v", reloadErr)
			http.Error(w, fmt.Sprintf("Strategy saved to database but failed to load in memory: %v", reloadErr), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/strategies", http.StatusSeeOther)
}

func (s *Server) handleStrategyUpdate(w http.ResponseWriter, r *http.Request, strategyID string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get existing strategy from database (source of truth)
	existingStrategy, err := s.stateManager.LoadStrategy(strategyID)
	if err != nil {
		s.logger.Printf("Failed to load strategy from database: %v", err)
		http.Error(w, "Strategy not found", http.StatusNotFound)
		return
	}

	// Update strategy fields
	strat := &strategy.Strategy{
		ID:        strategyID,
		Name:      r.FormValue("name"),
		Code:      r.FormValue("code"),
		Language:  r.FormValue("language"),
		CreatedAt: existingStrategy.CreatedAt, // Keep original creation time
		UpdatedAt: time.Now(),
	}

	// Parse max inputs
	if maxInputsStr := r.FormValue("max_inputs"); maxInputsStr != "" {
		if maxInputs, err := strconv.Atoi(maxInputsStr); err == nil {
			strat.MaxInputs = maxInputs
		}
	}

	// Parse default input names JSON if provided
	if defaultInputNamesJSON := r.FormValue("default_input_names"); defaultInputNamesJSON != "" {
		defaultInputNamesJSON = strings.TrimSpace(defaultInputNamesJSON)
		if defaultInputNamesJSON != "[]" && defaultInputNamesJSON != "null" {
			if err := json.Unmarshal([]byte(defaultInputNamesJSON), &strat.DefaultInputNames); err != nil {
				http.Error(w, "Invalid default input names JSON", http.StatusBadRequest)
				return
			}
		}
	}

	// Parse parameters JSON if provided
	if params := r.FormValue("parameters"); params != "" {
		if err := json.Unmarshal([]byte(params), &strat.Parameters); err != nil {
			http.Error(w, "Invalid parameters JSON", http.StatusBadRequest)
			return
		}
	} else {
		strat.Parameters = make(map[string]interface{})
	}

	// Save to database FIRST (source of truth)
	if err := s.stateManager.SaveStrategy(strat); err != nil {
		s.logger.Printf("Failed to save strategy to database: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save strategy to database: %v", err), http.StatusInternalServerError)
		return
	}

	// THEN reload in-memory version from database to sync
	if err := s.strategyEngine.ReloadStrategyFromDatabase(strategyID, strat); err != nil {
		s.logger.Printf("Failed to reload strategy from database: %v", err)
		// Continue anyway - database save succeeded, so this is just a warning
	}

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

// isValidStrategyID validates that a strategy ID follows the required pattern
func isValidStrategyID(id string) bool {
	if id == "" {
		return false
	}

	// Use pre-compiled regex pattern
	return strategyIDPattern.MatchString(id)
}
