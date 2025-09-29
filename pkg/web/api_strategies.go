package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
)

// Strategy API handlers

func (s *Server) handleAPIV1Strategies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleAPIStrategiesList(w, r)
	case "POST":
		s.handleAPIStrategiesCreate(w, r)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
	}
}

func (s *Server) handleAPIStrategiesList(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	languageFilter := r.URL.Query().Get("language")

	// Get all strategies from database (already ordered by name)
	allStrategies, err := s.stateManager.LoadAllStrategies()
	if err != nil {
		s.logger.Printf("Failed to load strategies from database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load strategies", nil)
		return
	}

	// Convert to slice for filtering and pagination
	strategyList := make([]StrategySummary, 0, len(allStrategies))
	for _, strat := range allStrategies {
		// Apply language filter if specified
		if languageFilter != "" && strat.Language != languageFilter {
			continue
		}

		summary := StrategySummary{
			ID:                strat.ID,
			Name:              strat.Name,
			Language:          strat.Language,
			CreatedAt:         strat.CreatedAt,
			UpdatedAt:         strat.UpdatedAt,
			MaxInputs:         strat.MaxInputs,
			DefaultInputNames: strat.DefaultInputNames,
		}
		strategyList = append(strategyList, summary)
	}

	total := len(strategyList)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		strategyList = []StrategySummary{}
	} else {
		if end > total {
			end = total
		}
		strategyList = strategyList[start:end]
	}

	response := StrategyListResponse{
		Strategies: strategyList,
		Pagination: PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: total,
			Pages: calculatePages(total, limit),
		},
	}

	writeAPIResponse(w, response)
}

func (s *Server) handleAPIStrategiesCreate(w http.ResponseWriter, r *http.Request) {
	var req StrategyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body", nil)
		return
	}

	// Validate required fields
	if req.ID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Strategy ID is required", nil)
		return
	}
	if req.Name == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Strategy name is required", nil)
		return
	}
	if req.Code == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Strategy code is required", nil)
		return
	}

	// Set defaults
	if req.Language == "" {
		req.Language = "javascript"
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]interface{})
	}

	// Create strategy
	strat := &strategy.Strategy{
		ID:                req.ID,
		Name:              req.Name,
		Code:              req.Code,
		Language:          req.Language,
		Parameters:        req.Parameters,
		MaxInputs:         req.MaxInputs,
		DefaultInputNames: req.DefaultInputNames,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Save to database first
	if err := s.stateManager.SaveStrategy(strat); err != nil {
		s.logger.Printf("Failed to save strategy to database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save strategy", nil)
		return
	}

	// Add to in-memory engine
	if err := s.strategyEngine.AddStrategy(strat); err != nil {
		s.logger.Printf("Failed to create strategy in memory: %v", err)
		// Try to reload from database instead
		if reloadErr := s.strategyEngine.ReloadStrategyFromDatabase(strat.ID, strat); reloadErr != nil {
			s.logger.Printf("Failed to reload strategy from database: %v", reloadErr)
			writeAPIError(w, http.StatusInternalServerError, "STRATEGY_LOAD_ERROR", "Strategy saved but failed to load in memory", nil)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	writeAPIResponse(w, map[string]string{"message": "Strategy created successfully"})
}

// Strategy detail endpoint
func (s *Server) handleAPIStrategyDetail(w http.ResponseWriter, r *http.Request) {
	// Extract strategy ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/strategies/")
	parts := strings.Split(path, "/")
	strategyID := parts[0]

	if strategyID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Strategy ID required", nil)
		return
	}

	// Handle sub-paths like /test
	if len(parts) > 1 && parts[1] == "test" {
		if r.Method == "POST" {
			s.handleAPIStrategyTest(w, r, strategyID)
		} else {
			writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		}
		return
	}

	switch r.Method {
	case "GET":
		s.handleAPIStrategyGet(w, r, strategyID)
	case "PUT":
		s.handleAPIStrategyUpdate(w, r, strategyID)
	case "DELETE":
		s.handleAPIStrategyDelete(w, r, strategyID)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
	}
}

func (s *Server) handleAPIStrategyGet(w http.ResponseWriter, r *http.Request, strategyID string) {
	// Load from database (source of truth)
	strat, err := s.stateManager.LoadStrategy(strategyID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "Strategy not found", nil)
		return
	}

	detail := StrategyDetail{
		ID:                strat.ID,
		Name:              strat.Name,
		Code:              strat.Code,
		Language:          strat.Language,
		Parameters:        strat.Parameters,
		MaxInputs:         strat.MaxInputs,
		DefaultInputNames: strat.DefaultInputNames,
		CreatedAt:         strat.CreatedAt,
		UpdatedAt:         strat.UpdatedAt,
	}

	writeAPIResponse(w, detail)
}

func (s *Server) handleAPIStrategyUpdate(w http.ResponseWriter, r *http.Request, strategyID string) {
	var req StrategyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body", nil)
		return
	}

	// Get existing strategy from database
	existingStrategy, err := s.stateManager.LoadStrategy(strategyID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "Strategy not found", nil)
		return
	}

	// Update strategy fields
	strat := &strategy.Strategy{
		ID:                strategyID,
		Name:              req.Name,
		Code:              req.Code,
		Language:          req.Language,
		Parameters:        req.Parameters,
		MaxInputs:         req.MaxInputs,
		DefaultInputNames: req.DefaultInputNames,
		CreatedAt:         existingStrategy.CreatedAt, // Keep original creation time
		UpdatedAt:         time.Now(),
	}

	// Set defaults
	if strat.Language == "" {
		strat.Language = "javascript"
	}
	if strat.Parameters == nil {
		strat.Parameters = make(map[string]interface{})
	}

	// Save to database first
	if err := s.stateManager.SaveStrategy(strat); err != nil {
		s.logger.Printf("Failed to save strategy to database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save strategy", nil)
		return
	}

	// Reload in-memory version
	if err := s.strategyEngine.ReloadStrategyFromDatabase(strategyID, strat); err != nil {
		s.logger.Printf("Failed to reload strategy from database: %v", err)
	}

	writeAPIResponse(w, map[string]string{"message": "Strategy updated successfully"})
}

func (s *Server) handleAPIStrategyDelete(w http.ResponseWriter, r *http.Request, strategyID string) {
	// Delete from database first
	if err := s.stateManager.DeleteStrategy(strategyID); err != nil {
		s.logger.Printf("Failed to delete strategy from database: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete strategy", nil)
		return
	}

	// Remove from memory
	if err := s.strategyEngine.RemoveStrategy(strategyID); err != nil {
		s.logger.Printf("Failed to remove strategy from memory: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// Strategy test structures
type StrategyTestRequest struct {
	Inputs     map[string]interface{} `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type StrategyTestResponse struct {
	Result          interface{}          `json:"result"`
	LogMessages     []string             `json:"log_messages"`
	EmittedEvents   []strategy.EmitEvent `json:"emitted_events"`
	ExecutionTimeMS int64                `json:"execution_time_ms"`
	Error           string               `json:"error,omitempty"`
}

func (s *Server) handleAPIStrategyTest(w http.ResponseWriter, r *http.Request, strategyID string) {
	var req StrategyTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body", nil)
		return
	}

	// Get strategy from in-memory engine (for testing)
	strat, err := s.strategyEngine.GetStrategy(strategyID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "Strategy not found", nil)
		return
	}

	// Override parameters if provided in request (for testing)
	_ = strat.Parameters // Using the strategy's default parameters

	// Execute strategy
	events, err := s.strategyEngine.ExecuteStrategy(strategyID, req.Inputs, nil, "test", nil)

	response := StrategyTestResponse{
		EmittedEvents: events,
	}

	if err != nil {
		response.Error = err.Error()
		writeAPIError(w, http.StatusBadRequest, "STRATEGY_EXECUTION_ERROR", "Strategy execution failed", response)
		return
	}

	// For successful execution, we'd need to modify the strategy engine to return more details
	// For now, just return the basic response
	if len(events) > 0 {
		// Return the main result (empty topic) if it exists
		for _, event := range events {
			if event.Topic == "" {
				response.Result = event.Value
				break
			}
		}
	}

	writeAPIResponse(w, response)
}
