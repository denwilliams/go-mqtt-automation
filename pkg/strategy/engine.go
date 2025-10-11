package strategy

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Engine struct {
	strategies map[string]*Strategy
	executors  map[string]LanguageExecutor
	logger     *log.Logger
	mutex      sync.RWMutex
}

func NewEngine(logger *log.Logger) *Engine {
	if logger == nil {
		logger = log.Default()
	}

	engine := &Engine{
		strategies: make(map[string]*Strategy),
		executors:  make(map[string]LanguageExecutor),
		logger:     logger,
	}

	// Register default executors
	engine.RegisterExecutor("javascript", NewJavaScriptExecutor())

	return engine
}

func (e *Engine) RegisterExecutor(language string, executor LanguageExecutor) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.executors[language] = executor
	e.logger.Printf("Registered strategy executor for language: %s", language)
}

func (e *Engine) AddStrategy(strategy *Strategy) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Validate the strategy
	if err := e.validateStrategy(strategy); err != nil {
		return fmt.Errorf("strategy validation failed: %w", err)
	}

	// Update timestamps
	now := time.Now()
	if strategy.CreatedAt.IsZero() {
		strategy.CreatedAt = now
	}
	strategy.UpdatedAt = now

	e.strategies[strategy.ID] = strategy
	e.logger.Printf("Added strategy: %s (%s)", strategy.Name, strategy.ID)

	return nil
}

func (e *Engine) RemoveStrategy(strategyID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, exists := e.strategies[strategyID]; !exists {
		return fmt.Errorf("strategy %s not found", strategyID)
	}

	delete(e.strategies, strategyID)
	e.logger.Printf("Removed strategy: %s", strategyID)

	return nil
}

func (e *Engine) GetStrategy(strategyID string) (*Strategy, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	strategy, exists := e.strategies[strategyID]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", strategyID)
	}

	return strategy, nil
}

func (e *Engine) ListStrategies() map[string]*Strategy {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	result := make(map[string]*Strategy)
	for id, strategy := range e.strategies {
		// Create a copy to avoid race conditions
		strategyCopy := *strategy
		result[id] = &strategyCopy
	}

	return result
}

func (e *Engine) ExecuteStrategy(strategyID string, inputs map[string]interface{}, inputNames map[string]string, triggerTopic string, lastOutput interface{}, topicParameters map[string]interface{}) ([]EmitEvent, error) {
	e.mutex.RLock()
	strategy, exists := e.strategies[strategyID]
	if !exists {
		e.mutex.RUnlock()
		return nil, fmt.Errorf("strategy %s not found", strategyID)
	}

	executor, executorExists := e.executors[strategy.Language]
	if !executorExists {
		e.mutex.RUnlock()
		return nil, fmt.Errorf("no executor found for language %s", strategy.Language)
	}
	e.mutex.RUnlock()

	// Merge parameters: topic parameters override strategy defaults
	mergedParameters := make(map[string]interface{})
	// Start with strategy defaults
	for k, v := range strategy.Parameters {
		mergedParameters[k] = v
	}
	// Override with topic-specific parameters
	for k, v := range topicParameters {
		mergedParameters[k] = v
	}

	// Ensure lastOutput is always an object (never nil)
	if lastOutput == nil {
		lastOutput = map[string]interface{}{}
	}

	// Get the triggering value
	var triggeringValue interface{}
	if triggerTopic != "" {
		triggeringValue = inputs[triggerTopic]
	}

	// Create execution context
	context := ExecutionContext{
		InputValues:     inputs,
		InputNames:      inputNames,
		TriggeringTopic: triggerTopic,
		TriggeringValue: triggeringValue,
		LastOutputs:     lastOutput,
		Parameters:      mergedParameters,
		TopicName:       "", // This would be set by the topic manager
	}

	e.logger.Printf("Executing strategy %s (%s) triggered by %s", strategy.Name, strategyID, triggerTopic)

	// Execute the strategy
	result := executor.Execute(strategy, context)

	// Log execution details
	if result.Error != nil {
		e.logger.Printf("Strategy execution failed: %v", result.Error)
	} else {
		e.logger.Printf("Strategy executed successfully in %v", result.ExecutionTime)
	}

	// Log any messages from the strategy
	for _, msg := range result.LogMessages {
		e.logger.Printf("Strategy log: %s", msg)
	}

	// Handle emitted events (this would typically be handled by the topic manager)
	if len(result.EmittedEvents) > 0 {
		e.logger.Printf("Strategy emitted %d events", len(result.EmittedEvents))
		for _, event := range result.EmittedEvents {
			e.logger.Printf("Emitted event: %s = %v", event.Topic, event.Value)
		}
	}

	if result.Error != nil {
		return nil, result.Error
	}

	// Prepare events to return
	// Strategy: For each topic (main or subtopic), keep only the LAST emit
	// This handles both context.emit(value) and return value
	eventMap := make(map[string]EmitEvent) // topic path -> last event

	// Process emitted events (from context.emit calls)
	for _, event := range result.EmittedEvents {
		// Replace any previous emit to the same topic
		eventMap[event.Topic] = event
	}

	// If function returned a value, it overrides any previous main topic emit
	if result.Result != nil {
		eventMap[""] = EmitEvent{
			Topic: "",
			Value: result.Result,
		}
	}

	// Convert map to slice
	events := make([]EmitEvent, 0, len(eventMap))
	for _, event := range eventMap {
		events = append(events, event)
	}

	return events, nil
}

func (e *Engine) ValidateStrategy(strategy *Strategy) error {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return e.validateStrategy(strategy)
}

func (e *Engine) validateStrategy(strategy *Strategy) error {
	// Check required fields
	if strategy.ID == "" {
		return fmt.Errorf("strategy ID is required")
	}
	if strategy.Name == "" {
		return fmt.Errorf("strategy name is required")
	}
	if strategy.Code == "" {
		return fmt.Errorf("strategy code is required")
	}
	if strategy.Language == "" {
		strategy.Language = "javascript" // Default language
	}

	// Check if executor exists for the language
	executor, exists := e.executors[strategy.Language]
	if !exists {
		return fmt.Errorf("no executor available for language %s", strategy.Language)
	}

	// Validate the code using the executor
	if err := executor.Validate(strategy.Code); err != nil {
		return fmt.Errorf("code validation failed: %w", err)
	}

	// Initialize parameters if nil
	if strategy.Parameters == nil {
		strategy.Parameters = make(map[string]interface{})
	}

	return nil
}

func (e *Engine) UpdateStrategy(strategy *Strategy) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, exists := e.strategies[strategy.ID]; !exists {
		return fmt.Errorf("strategy %s not found", strategy.ID)
	}

	// Validate the updated strategy
	if err := e.validateStrategy(strategy); err != nil {
		return fmt.Errorf("strategy validation failed: %w", err)
	}

	// Update timestamp
	strategy.UpdatedAt = time.Now()

	e.strategies[strategy.ID] = strategy
	e.logger.Printf("Updated strategy: %s (%s)", strategy.Name, strategy.ID)

	return nil
}

func (e *Engine) GetSupportedLanguages() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	languages := make([]string, 0, len(e.executors))
	for lang := range e.executors {
		languages = append(languages, lang)
	}

	return languages
}

func (e *Engine) GetStrategyCount() int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return len(e.strategies)
}

// ReloadStrategyFromDatabase loads a strategy from the database and updates the in-memory version
func (e *Engine) ReloadStrategyFromDatabase(strategyID string, strategy *Strategy) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Validate the strategy
	if err := e.validateStrategy(strategy); err != nil {
		return fmt.Errorf("strategy validation failed: %w", err)
	}

	// Update timestamp
	strategy.UpdatedAt = time.Now()

	// Update the in-memory strategy
	e.strategies[strategy.ID] = strategy
	e.logger.Printf("Reloaded strategy from database: %s (%s)", strategy.Name, strategy.ID)

	return nil
}
