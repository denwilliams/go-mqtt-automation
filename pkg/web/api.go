package web

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
)

// API Response structures
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type PaginationResponse struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// Dashboard structures
type DashboardResponse struct {
	System DashboardSystem `json:"system"`
	Stats  DashboardStats  `json:"stats"`
}

type DashboardSystem struct {
	Uptime  string `json:"uptime"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type DashboardStats struct {
	Topics     TopicStats    `json:"topics"`
	Strategies StrategyStats `json:"strategies"`
	MQTT       MQTTStats     `json:"mqtt"`
}

type TopicStats struct {
	External int `json:"external"`
	Internal int `json:"internal"`
	System   int `json:"system"`
	Total    int `json:"total"`
}

type StrategyStats struct {
	Total  int `json:"total"`
	Active int `json:"active"`
	Failed int `json:"failed"`
}

type MQTTStats struct {
	Connected         bool      `json:"connected"`
	MessagesProcessed int64     `json:"messages_processed"`
	LastMessage       time.Time `json:"last_message"`
}

// Topic structures
type TopicListResponse struct {
	Topics     []TopicSummary     `json:"topics"`
	Pagination PaginationResponse `json:"pagination"`
}

type TopicSummary struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	LastValue   interface{} `json:"last_value"`
	LastUpdated time.Time   `json:"last_updated"`
	Inputs      []string    `json:"inputs,omitempty"`
	StrategyID  string      `json:"strategy_id,omitempty"`
	EmitToMQTT  bool        `json:"emit_to_mqtt,omitempty"`
}

type TopicDetail struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	LastValue     interface{}            `json:"last_value"`
	LastUpdated   time.Time              `json:"last_updated"`
	CreatedAt     time.Time              `json:"created_at"`
	Inputs        []string               `json:"inputs,omitempty"`
	InputNames    map[string]string      `json:"input_names,omitempty"`
	StrategyID    string                 `json:"strategy_id,omitempty"`
	EmitToMQTT    bool                   `json:"emit_to_mqtt,omitempty"`
	NoOpUnchanged bool                   `json:"noop_unchanged,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

type TopicCreateRequest struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Inputs        []string          `json:"inputs,omitempty"`
	InputNames    map[string]string `json:"input_names,omitempty"`
	StrategyID    string            `json:"strategy_id,omitempty"`
	EmitToMQTT    bool              `json:"emit_to_mqtt,omitempty"`
	NoOpUnchanged bool              `json:"noop_unchanged,omitempty"`
}

// Strategy structures
type StrategyListResponse struct {
	Strategies []StrategySummary  `json:"strategies"`
	Pagination PaginationResponse `json:"pagination"`
}

type StrategySummary struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Language          string    `json:"language"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	MaxInputs         int       `json:"max_inputs"`
	DefaultInputNames []string  `json:"default_input_names"`
}

type StrategyDetail struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Code              string                 `json:"code"`
	Language          string                 `json:"language"`
	Parameters        map[string]interface{} `json:"parameters"`
	MaxInputs         int                    `json:"max_inputs"`
	DefaultInputNames []string               `json:"default_input_names"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type StrategyCreateRequest struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Code              string                 `json:"code"`
	Language          string                 `json:"language"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	MaxInputs         int                    `json:"max_inputs,omitempty"`
	DefaultInputNames []string               `json:"default_input_names,omitempty"`
}

// System structures
type SystemInfoResponse struct {
	Version       string `json:"version"`
	Uptime        string `json:"uptime"`
	BuildDate     string `json:"build_date"`
	GoVersion     string `json:"go_version"`
	DatabaseType  string `json:"database_type"`
	MQTTConnected bool   `json:"mqtt_connected"`
}

// Helper functions
func writeAPIResponse(w http.ResponseWriter, data interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	writeJSONResponse(w, status, response)
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func parsePagination(r *http.Request) (page, limit int) {
	page = 1
	limit = 50

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	return page, limit
}

func calculatePages(total, limit int) int {
	if limit <= 0 {
		return 1
	}
	return (total + limit - 1) / limit
}

// API Handlers

// Dashboard endpoint
func (s *Server) handleAPIDashboard(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight requests
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

	// Get topic counts
	topicCounts := s.topicManager.GetTopicCount()
	total := topicCounts[topics.TopicTypeExternal] + topicCounts[topics.TopicTypeInternal] + topicCounts[topics.TopicTypeSystem]

	// Get strategy count
	strategyCount := s.strategyEngine.GetStrategyCount()

	// Get MQTT status
	mqttConnected := false
	if s.mqttClient != nil {
		mqttConnected = s.mqttClient.IsConnected()
	}

	response := DashboardResponse{
		System: DashboardSystem{
			Uptime:  "0m",    // TODO: Calculate actual uptime
			Version: "1.0.0", // TODO: Get from build info
			Status:  "healthy",
		},
		Stats: DashboardStats{
			Topics: TopicStats{
				External: topicCounts[topics.TopicTypeExternal],
				Internal: topicCounts[topics.TopicTypeInternal],
				System:   topicCounts[topics.TopicTypeSystem],
				Total:    total,
			},
			Strategies: StrategyStats{
				Total:  strategyCount,
				Active: strategyCount, // TODO: Track active vs failed
				Failed: 0,
			},
			MQTT: MQTTStats{
				Connected:         mqttConnected,
				MessagesProcessed: 0, // TODO: Track messages
				LastMessage:       time.Now(),
			},
		},
	}

	writeAPIResponse(w, response)
}

// Topics API
func (s *Server) handleAPIV1Topics(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleAPITopicsList(w, r)
	case "POST":
		s.handleAPITopicsCreate(w, r)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
	}
}

func (s *Server) handleAPITopicsList(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	topicType := r.URL.Query().Get("type")

	// Get all topics from database (already ordered by name)
	allTopicConfigs, err := s.stateManager.LoadAllTopicConfigs()
	if err != nil {
		s.logger.Printf("Failed to load topics from database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load topics", nil)
		return
	}

	// Get child topics and external topics from in-memory topic manager
	childTopics := s.topicManager.GetChildTopics()
	externalTopics := s.topicManager.GetExternalTopics()

	// Convert to slice for filtering and pagination
	// Allocate space for database topics, child topics, and external topics
	topicList := make([]TopicSummary, 0, len(allTopicConfigs)+len(childTopics)+len(externalTopics))

	// Process database topics
	for _, config := range allTopicConfigs {
		// Extract common fields based on config type
		var summary TopicSummary

		switch cfg := config.(type) {
		case topics.BaseTopicConfig:
			summary = TopicSummary{
				Name:        cfg.Name,
				Type:        string(cfg.Type),
				LastValue:   cfg.LastValue,
				LastUpdated: cfg.LastUpdated,
			}
		case topics.InternalTopicConfig:
			summary = TopicSummary{
				Name:        cfg.Name,
				Type:        string(cfg.Type),
				LastValue:   cfg.LastValue,
				LastUpdated: cfg.LastUpdated,
				Inputs:      cfg.Inputs,
				StrategyID:  cfg.StrategyID,
				EmitToMQTT:  cfg.EmitToMQTT,
			}
		case topics.SystemTopicConfig:
			summary = TopicSummary{
				Name:        cfg.Name,
				Type:        string(cfg.Type),
				LastValue:   cfg.LastValue,
				LastUpdated: cfg.LastUpdated,
			}
		default:
			s.logger.Printf("Unknown topic config type: %T", cfg)
			continue
		}

		// Apply type filter if specified
		if topicType != "" && summary.Type != topicType {
			continue
		}

		topicList = append(topicList, summary)
	}

	// Add child topics from in-memory topic manager
	for _, childConfig := range childTopics {
		summary := TopicSummary{
			Name:        childConfig.Name,
			Type:        string(childConfig.Type),
			LastValue:   childConfig.LastValue,
			LastUpdated: childConfig.LastUpdated,
			Inputs:      childConfig.Inputs,
			StrategyID:  childConfig.StrategyID,
			EmitToMQTT:  childConfig.EmitToMQTT,
		}

		// Apply type filter if specified
		if topicType != "" && summary.Type != topicType {
			continue
		}

		topicList = append(topicList, summary)
	}

	// Add external topics from in-memory topic manager
	for _, externalConfig := range externalTopics {
		summary := TopicSummary{
			Name:        externalConfig.Name,
			Type:        string(externalConfig.Type),
			LastValue:   externalConfig.LastValue,
			LastUpdated: externalConfig.LastUpdated,
		}

		// Apply type filter if specified
		if topicType != "" && summary.Type != topicType {
			continue
		}

		topicList = append(topicList, summary)
	}

	// Sort topics by name since we merged from multiple sources
	sort.Slice(topicList, func(i, j int) bool {
		return topicList[i].Name < topicList[j].Name
	})

	total := len(topicList)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		topicList = []TopicSummary{}
	} else {
		if end > total {
			end = total
		}
		topicList = topicList[start:end]
	}

	response := TopicListResponse{
		Topics: topicList,
		Pagination: PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: total,
			Pages: calculatePages(total, limit),
		},
	}

	writeAPIResponse(w, response)
}

func (s *Server) handleAPITopicsCreate(w http.ResponseWriter, r *http.Request) {
	var req TopicCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Topic name is required", nil)
		return
	}

	// Only support internal topics for creation via API
	if req.Type != "internal" {
		writeAPIError(w, http.StatusBadRequest, "INVALID_TYPE", "Only internal topics can be created via API", nil)
		return
	}

	// Create the topic config
	config := topics.InternalTopicConfig{
		BaseTopicConfig: topics.BaseTopicConfig{
			Name:        req.Name,
			Type:        topics.TopicTypeInternal,
			CreatedAt:   time.Now(),
			LastUpdated: time.Now(),
			Config:      make(map[string]interface{}),
		},
		Inputs:        req.Inputs,
		InputNames:    req.InputNames,
		StrategyID:    req.StrategyID,
		EmitToMQTT:    req.EmitToMQTT,
		NoOpUnchanged: req.NoOpUnchanged,
	}

	// Save to database first
	if err := s.stateManager.SaveTopicConfig(config); err != nil {
		s.logger.Printf("Failed to save topic to database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save topic", nil)
		return
	}

	// Create in-memory version
	_, err := s.topicManager.AddInternalTopic(req.Name, req.Inputs, req.StrategyID)
	if err != nil {
		s.logger.Printf("Failed to create topic in memory: %v", err)
		// Try to reload from database instead
		if reloadErr := s.topicManager.ReloadTopicFromDatabase(req.Name); reloadErr != nil {
			s.logger.Printf("Failed to reload topic from database: %v", reloadErr)
			writeAPIError(w, http.StatusInternalServerError, "TOPIC_LOAD_ERROR", "Topic saved but failed to load in memory", nil)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	writeAPIResponse(w, map[string]string{"message": "Topic created successfully"})
}

// Topic detail endpoint
func (s *Server) handleAPITopicDetail(w http.ResponseWriter, r *http.Request) {
	// Extract topic name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/topics/")
	topicName := strings.TrimSuffix(path, "/")

	if topicName == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Topic name required", nil)
		return
	}

	switch r.Method {
	case "GET":
		s.handleAPITopicGet(w, r, topicName)
	case "PUT":
		s.handleAPITopicUpdate(w, r, topicName)
	case "DELETE":
		s.handleAPITopicDelete(w, r, topicName)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
	}
}

func (s *Server) handleAPITopicGet(w http.ResponseWriter, r *http.Request, topicName string) {
	// Load from database (source of truth)
	configInterface, err := s.stateManager.LoadTopicConfig(topicName)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "Topic not found", nil)
		return
	}

	// Get the in-memory topic for additional info
	topic := s.topicManager.GetTopic(topicName)
	if topic == nil {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "Topic not found in memory", nil)
		return
	}

	var detail TopicDetail
	detail.Name = topic.Name()
	detail.Type = string(topic.Type())
	detail.LastValue = topic.LastValue()
	detail.LastUpdated = topic.LastUpdated()

	// Handle different topic types
	switch cfg := configInterface.(type) {
	case topics.InternalTopicConfig:
		detail.CreatedAt = cfg.CreatedAt
		detail.Inputs = cfg.Inputs
		detail.InputNames = cfg.InputNames
		detail.StrategyID = cfg.StrategyID
		detail.EmitToMQTT = cfg.EmitToMQTT
		detail.NoOpUnchanged = cfg.NoOpUnchanged
		detail.Config = cfg.Config
	case topics.BaseTopicConfig:
		detail.CreatedAt = cfg.CreatedAt
		detail.Config = cfg.Config
	case topics.SystemTopicConfig:
		detail.CreatedAt = cfg.CreatedAt
		detail.Config = cfg.Config
	}

	writeAPIResponse(w, detail)
}

func (s *Server) handleAPITopicUpdate(w http.ResponseWriter, r *http.Request, topicName string) {
	var req TopicCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body", nil)
		return
	}

	// Get existing topic
	topic := s.topicManager.GetInternalTopic(topicName)
	if topic == nil {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "Topic not found", nil)
		return
	}

	// Update config
	config := topic.GetConfig()
	config.Inputs = req.Inputs
	config.InputNames = req.InputNames
	config.StrategyID = req.StrategyID
	config.EmitToMQTT = req.EmitToMQTT
	config.NoOpUnchanged = req.NoOpUnchanged
	config.Type = topics.TopicTypeInternal

	// Save to database first
	if err := s.stateManager.SaveTopicConfig(config); err != nil {
		s.logger.Printf("Failed to save topic to database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save topic", nil)
		return
	}

	// Reload in-memory version
	if err := s.topicManager.ReloadTopicFromDatabase(topicName); err != nil {
		s.logger.Printf("Failed to reload topic from database: %v", err)
	}

	writeAPIResponse(w, map[string]string{"message": "Topic updated successfully"})
}

func (s *Server) handleAPITopicDelete(w http.ResponseWriter, r *http.Request, topicName string) {
	// Delete from database first
	if err := s.stateManager.DeleteTopicConfig(topicName); err != nil {
		s.logger.Printf("Failed to delete topic from database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete topic", nil)
		return
	}

	// Remove from memory
	if err := s.topicManager.RemoveTopic(topicName); err != nil {
		s.logger.Printf("Failed to remove topic from memory: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
