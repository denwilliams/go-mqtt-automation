package strategy

import (
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// Mock executor for testing
type mockExecutor struct {
	validateFunc func(code string) error
	executeFunc  func(strategy *Strategy, context ExecutionContext) ExecutionResult
}

func (m *mockExecutor) Validate(code string) error {
	if m.validateFunc != nil {
		return m.validateFunc(code)
	}
	if strings.Contains(code, "INVALID") {
		return errors.New("invalid code")
	}
	return nil
}

func (m *mockExecutor) Execute(strategy *Strategy, context ExecutionContext) ExecutionResult {
	if m.executeFunc != nil {
		return m.executeFunc(strategy, context)
	}
	return ExecutionResult{
		Result:        "mock result",
		ExecutionTime: time.Millisecond * 10,
	}
}

func TestNewEngine(t *testing.T) {
	engine := NewEngine(nil)

	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}

	if engine.strategies == nil {
		t.Error("strategies map not initialized")
	}

	if engine.executors == nil {
		t.Error("executors map not initialized")
	}

	if engine.logger == nil {
		t.Error("logger not initialized")
	}

	// Should have JavaScript executor registered by default
	if len(engine.executors) == 0 {
		t.Error("no default executors registered")
	}

	if _, exists := engine.executors["javascript"]; !exists {
		t.Error("JavaScript executor not registered by default")
	}
}

func TestNewEngineWithCustomLogger(t *testing.T) {
	customLogger := log.New(os.Stderr, "TEST: ", log.LstdFlags)
	engine := NewEngine(customLogger)

	if engine.logger != customLogger {
		t.Error("custom logger not set")
	}
}

func TestRegisterExecutor(t *testing.T) {
	engine := NewEngine(nil)
	mockExec := &mockExecutor{}

	engine.RegisterExecutor("test-lang", mockExec)

	if len(engine.executors) < 2 { // javascript + test-lang
		t.Error("executor not registered")
	}

	if executor, exists := engine.executors["test-lang"]; !exists || executor != mockExec {
		t.Error("executor not properly registered")
	}
}

func TestAddStrategy(t *testing.T) {
	engine := NewEngine(nil)
	mockExec := &mockExecutor{}
	engine.RegisterExecutor("test-lang", mockExec)

	strategy := &Strategy{
		ID:       "test-strategy",
		Name:     "Test Strategy",
		Code:     "test code",
		Language: "test-lang",
	}

	err := engine.AddStrategy(strategy)
	if err != nil {
		t.Fatalf("AddStrategy() failed: %v", err)
	}

	if len(engine.strategies) != 1 {
		t.Error("strategy not added to engine")
	}

	if !strategy.CreatedAt.After(time.Time{}) {
		t.Error("CreatedAt not set")
	}

	if !strategy.UpdatedAt.After(time.Time{}) {
		t.Error("UpdatedAt not set")
	}

	if strategy.Parameters == nil {
		t.Error("Parameters not initialized")
	}
}

func TestAddStrategyValidation(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		strategy *Strategy
		wantErr  bool
	}{
		{
			name: "missing ID",
			strategy: &Strategy{
				Name:     "Test",
				Code:     "test",
				Language: "javascript",
			},
			wantErr: true,
		},
		{
			name: "missing Name",
			strategy: &Strategy{
				ID:       "test",
				Code:     "test",
				Language: "javascript",
			},
			wantErr: true,
		},
		{
			name: "missing Code",
			strategy: &Strategy{
				ID:       "test",
				Name:     "Test",
				Language: "javascript",
			},
			wantErr: true,
		},
		{
			name: "invalid language",
			strategy: &Strategy{
				ID:       "test",
				Name:     "Test",
				Code:     "test",
				Language: "nonexistent",
			},
			wantErr: true,
		},
		{
			name: "invalid code",
			strategy: &Strategy{
				ID:       "test",
				Name:     "Test",
				Code:     "INVALID code",
				Language: "javascript",
			},
			wantErr: true,
		},
		{
			name: "valid strategy",
			strategy: &Strategy{
				ID:       "test",
				Name:     "Test",
				Code:     "function process() { return 'test'; }",
				Language: "javascript",
			},
			wantErr: false,
		},
		{
			name: "default language",
			strategy: &Strategy{
				ID:   "test2",
				Name: "Test2",
				Code: "function process() { return 'test'; }",
				// Language omitted - should default to javascript
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.AddStrategy(tt.strategy)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddStrategy() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.strategy.Language == "" {
				// Should have defaulted to javascript
				if tt.strategy.Language != "javascript" {
					t.Error("Language not defaulted to javascript")
				}
			}
		})
	}
}

func TestRemoveStrategy(t *testing.T) {
	engine := NewEngine(nil)

	strategy := &Strategy{
		ID:       "test-strategy",
		Name:     "Test Strategy",
		Code:     "function process() { return 'test'; }",
		Language: "javascript",
	}

	// Add strategy first
	err := engine.AddStrategy(strategy)
	if err != nil {
		t.Fatalf("Failed to add strategy: %v", err)
	}

	// Remove strategy
	err = engine.RemoveStrategy("test-strategy")
	if err != nil {
		t.Fatalf("RemoveStrategy() failed: %v", err)
	}

	if len(engine.strategies) != 0 {
		t.Error("strategy not removed")
	}

	// Try to remove non-existent strategy
	err = engine.RemoveStrategy("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent strategy")
	}
}

func TestGetStrategy(t *testing.T) {
	engine := NewEngine(nil)

	strategy := &Strategy{
		ID:       "test-strategy",
		Name:     "Test Strategy",
		Code:     "function process() { return 'test'; }",
		Language: "javascript",
	}

	// Add strategy first
	err := engine.AddStrategy(strategy)
	if err != nil {
		t.Fatalf("Failed to add strategy: %v", err)
	}

	// Get existing strategy
	retrieved, err := engine.GetStrategy("test-strategy")
	if err != nil {
		t.Fatalf("GetStrategy() failed: %v", err)
	}

	if retrieved.ID != strategy.ID {
		t.Error("retrieved strategy has wrong ID")
	}

	// Try to get non-existent strategy
	_, err = engine.GetStrategy("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent strategy")
	}
}

func TestListStrategies(t *testing.T) {
	engine := NewEngine(nil)

	strategies := []*Strategy{
		{
			ID:       "strategy1",
			Name:     "Strategy 1",
			Code:     "function process() { return 'test1'; }",
			Language: "javascript",
		},
		{
			ID:       "strategy2",
			Name:     "Strategy 2",
			Code:     "function process() { return 'test2'; }",
			Language: "javascript",
		},
	}

	// Add strategies
	for _, s := range strategies {
		err := engine.AddStrategy(s)
		if err != nil {
			t.Fatalf("Failed to add strategy %s: %v", s.ID, err)
		}
	}

	// List strategies
	listed := engine.ListStrategies()

	if len(listed) != 2 {
		t.Errorf("expected 2 strategies, got %d", len(listed))
	}

	for _, expected := range strategies {
		if found, exists := listed[expected.ID]; !exists {
			t.Errorf("strategy %s not found in list", expected.ID)
		} else if found.Name != expected.Name {
			t.Errorf("strategy %s has wrong name", expected.ID)
		}
	}

	// Verify we get copies, not references
	original := strategies[0]
	copy := listed[original.ID]
	copy.Name = "Modified"

	if original.Name == "Modified" {
		t.Error("ListStrategies() returned reference instead of copy")
	}
}

func TestExecuteStrategy(t *testing.T) {
	engine := NewEngine(nil)

	// Create mock executor that returns predictable results
	mockExec := &mockExecutor{
		executeFunc: func(strategy *Strategy, context ExecutionContext) ExecutionResult {
			return ExecutionResult{
				Result:        map[string]interface{}{"strategy_id": strategy.ID, "input_count": len(context.InputValues)},
				ExecutionTime: time.Millisecond * 5,
				LogMessages:   []string{"test log message"},
			}
		},
	}
	engine.RegisterExecutor("mock", mockExec)

	strategy := &Strategy{
		ID:       "test-strategy",
		Name:     "Test Strategy",
		Code:     "test code",
		Language: "mock",
		Parameters: map[string]interface{}{
			"test_param": "test_value",
		},
	}

	err := engine.AddStrategy(strategy)
	if err != nil {
		t.Fatalf("Failed to add strategy: %v", err)
	}

	inputs := map[string]interface{}{
		"topic1": 25.5,
		"topic2": true,
	}

	result, err := engine.ExecuteStrategy("test-strategy", inputs, "topic1", nil)
	if err != nil {
		t.Fatalf("ExecuteStrategy() failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	if resultMap["strategy_id"] != "test-strategy" {
		t.Error("strategy ID not passed correctly")
	}

	if resultMap["input_count"] != 2 {
		t.Error("input count not passed correctly")
	}

	// Test with non-existent strategy
	_, err = engine.ExecuteStrategy("nonexistent", inputs, "topic1", nil)
	if err == nil {
		t.Error("expected error for non-existent strategy")
	}
}

func TestExecuteStrategyWithError(t *testing.T) {
	engine := NewEngine(nil)

	// Mock executor that returns an error
	mockExec := &mockExecutor{
		executeFunc: func(strategy *Strategy, context ExecutionContext) ExecutionResult {
			return ExecutionResult{
				Error:         errors.New("execution failed"),
				ExecutionTime: time.Millisecond * 2,
			}
		},
	}
	engine.RegisterExecutor("mock", mockExec)

	strategy := &Strategy{
		ID:       "error-strategy",
		Name:     "Error Strategy",
		Code:     "error code",
		Language: "mock",
	}

	err := engine.AddStrategy(strategy)
	if err != nil {
		t.Fatalf("Failed to add strategy: %v", err)
	}

	_, err = engine.ExecuteStrategy("error-strategy", map[string]interface{}{}, "topic1", nil)
	if err == nil {
		t.Error("expected error from strategy execution")
	}
}

func TestUpdateStrategy(t *testing.T) {
	engine := NewEngine(nil)

	strategy := &Strategy{
		ID:       "test-strategy",
		Name:     "Original Name",
		Code:     "function process() { return 'original'; }",
		Language: "javascript",
	}

	// Add original strategy
	err := engine.AddStrategy(strategy)
	if err != nil {
		t.Fatalf("Failed to add strategy: %v", err)
	}

	originalUpdateTime := strategy.UpdatedAt
	time.Sleep(time.Millisecond * 10) // Ensure time difference

	// Update strategy
	strategy.Name = "Updated Name"
	strategy.Code = "function process() { return 'updated'; }"

	err = engine.UpdateStrategy(strategy)
	if err != nil {
		t.Fatalf("UpdateStrategy() failed: %v", err)
	}

	// Verify update
	retrieved, err := engine.GetStrategy("test-strategy")
	if err != nil {
		t.Fatalf("Failed to get updated strategy: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Error("strategy name not updated")
	}

	if !retrieved.UpdatedAt.After(originalUpdateTime) {
		t.Error("UpdatedAt not updated")
	}

	// Test updating non-existent strategy
	nonExistent := &Strategy{
		ID:       "nonexistent",
		Name:     "Test",
		Code:     "function process() { return 'test'; }",
		Language: "javascript",
	}

	err = engine.UpdateStrategy(nonExistent)
	if err == nil {
		t.Error("expected error for non-existent strategy")
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	engine := NewEngine(nil)

	// Should have javascript by default
	languages := engine.GetSupportedLanguages()
	if len(languages) < 1 {
		t.Error("no supported languages found")
	}

	found := false
	for _, lang := range languages {
		if lang == "javascript" {
			found = true
			break
		}
	}
	if !found {
		t.Error("javascript not in supported languages")
	}

	// Add another language
	mockExec := &mockExecutor{}
	engine.RegisterExecutor("test-lang", mockExec)

	languages = engine.GetSupportedLanguages()
	if len(languages) < 2 {
		t.Error("new language not included in supported languages")
	}
}

func TestGetStrategyCount(t *testing.T) {
	engine := NewEngine(nil)

	if engine.GetStrategyCount() != 0 {
		t.Error("expected 0 strategies initially")
	}

	strategy1 := &Strategy{
		ID:       "strategy1",
		Name:     "Strategy 1",
		Code:     "function process() { return 'test1'; }",
		Language: "javascript",
	}

	strategy2 := &Strategy{
		ID:       "strategy2",
		Name:     "Strategy 2",
		Code:     "function process() { return 'test2'; }",
		Language: "javascript",
	}

	// Add strategies
	engine.AddStrategy(strategy1)
	if engine.GetStrategyCount() != 1 {
		t.Error("expected 1 strategy after adding first")
	}

	engine.AddStrategy(strategy2)
	if engine.GetStrategyCount() != 2 {
		t.Error("expected 2 strategies after adding second")
	}

	// Remove one
	engine.RemoveStrategy("strategy1")
	if engine.GetStrategyCount() != 1 {
		t.Error("expected 1 strategy after removing one")
	}
}

func TestValidateStrategy(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		strategy *Strategy
		wantErr  bool
	}{
		{
			name: "valid strategy",
			strategy: &Strategy{
				ID:       "test",
				Name:     "Test",
				Code:     "function process() { return 'test'; }",
				Language: "javascript",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			strategy: &Strategy{
				Name:     "Test",
				Code:     "function process() { return 'test'; }",
				Language: "javascript",
			},
			wantErr: true,
		},
		{
			name: "empty code",
			strategy: &Strategy{
				ID:       "test",
				Name:     "Test",
				Code:     "",
				Language: "javascript",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateStrategy(tt.strategy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStrategy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test concurrent access to ensure thread safety
func TestConcurrentAccess(t *testing.T) {
	engine := NewEngine(nil)

	// Add initial strategy
	strategy := &Strategy{
		ID:       "concurrent-test",
		Name:     "Concurrent Test",
		Code:     "function process() { return 'test'; }",
		Language: "javascript",
	}

	err := engine.AddStrategy(strategy)
	if err != nil {
		t.Fatalf("Failed to add initial strategy: %v", err)
	}

	// Run concurrent operations
	done := make(chan bool, 3)

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			engine.ListStrategies()
			engine.GetStrategy("concurrent-test")
			engine.GetStrategyCount()
		}
		done <- true
	}()

	// Concurrent execution
	go func() {
		inputs := map[string]interface{}{"test": 1}
		for i := 0; i < 50; i++ {
			engine.ExecuteStrategy("concurrent-test", inputs, "test", nil)
		}
		done <- true
	}()

	// Concurrent updates
	go func() {
		for i := 0; i < 10; i++ {
			newStrategy := &Strategy{
				ID:       "concurrent-test",
				Name:     "Updated Test",
				Code:     "function process() { return 'updated'; }",
				Language: "javascript",
			}
			engine.UpdateStrategy(newStrategy)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify engine is still functional
	if engine.GetStrategyCount() != 1 {
		t.Error("engine corrupted after concurrent access")
	}
}
