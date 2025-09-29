package topics

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
)

// Mock strategy executor for testing
type mockStrategyExecutor struct {
	executeFunc func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error)
}

func (m *mockStrategyExecutor) ExecuteStrategy(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) ([]strategy.EmitEvent, error) {
	if m.executeFunc != nil {
		if result, err := m.executeFunc(strategyID, inputs, triggerTopic, lastOutput); err != nil {
			return nil, err
		} else {
			// Handle special test cases that emit derived topics
			switch strategyID {
			case "parent-strategy":
				// Emit derived topic
				return []strategy.EmitEvent{
					{Topic: "", Value: result},        // Main topic output
					{Topic: "/battery", Value: "75%"}, // Derived topic (relative path)
				}, nil
			case "emitter-strategy":
				// Emit multiple derived topics
				if resultMap, ok := result.(map[string]interface{}); ok {
					events := []strategy.EmitEvent{{Topic: "", Value: result}} // Main topic
					for key, value := range resultMap {
						events = append(events, strategy.EmitEvent{Topic: "/" + key, Value: value})
					}
					return events, nil
				}
			case "source-strategy":
				// Emit to a specific derived topic
				return []strategy.EmitEvent{
					{Topic: "", Value: result},        // Main topic output
					{Topic: "/output", Value: result}, // Derived topic (relative path)
				}, nil
			}
			// Default: just return main topic
			return []strategy.EmitEvent{{Topic: "", Value: result}}, nil
		}
	}
	return []strategy.EmitEvent{{Topic: "", Value: "mock result"}}, nil
}

func (m *mockStrategyExecutor) GetStrategy(strategyID string) (*strategy.Strategy, error) {
	// Return a mock strategy for testing
	return &strategy.Strategy{
		ID:       strategyID,
		Name:     "Mock Strategy",
		Code:     "mock code",
		Language: "javascript",
	}, nil
}

// Mock state manager for testing
type mockStateManager struct {
	saveFunc       func(topicName string, value interface{}) error
	loadFunc       func(topicName string) (interface{}, error)
	loadConfigFunc func(topicName string) (interface{}, error)
}

func (m *mockStateManager) SaveTopicState(topicName string, value interface{}) error {
	if m.saveFunc != nil {
		return m.saveFunc(topicName, value)
	}
	return nil
}

func (m *mockStateManager) LoadTopicState(topicName string) (interface{}, error) {
	if m.loadFunc != nil {
		return m.loadFunc(topicName)
	}
	return nil, nil
}

func (m *mockStateManager) LoadTopicConfig(topicName string) (interface{}, error) {
	if m.loadConfigFunc != nil {
		return m.loadConfigFunc(topicName)
	}
	return nil, nil
}

func TestNewManager(t *testing.T) {
	manager := NewManager(nil)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.topics == nil {
		t.Error("topics map not initialized")
	}

	if manager.externalTopics == nil {
		t.Error("externalTopics map not initialized")
	}

	if manager.internalTopics == nil {
		t.Error("internalTopics map not initialized")
	}

	if manager.systemTopics == nil {
		t.Error("systemTopics map not initialized")
	}

	if manager.logger == nil {
		t.Error("logger not initialized")
	}
}

func TestNewManagerWithCustomLogger(t *testing.T) {
	customLogger := log.New(os.Stderr, "TEST: ", log.LstdFlags)
	manager := NewManager(customLogger)

	if manager.logger != customLogger {
		t.Error("custom logger not set")
	}
}

func TestAddExternalTopic(t *testing.T) {
	manager := NewManager(nil)

	topic := manager.AddExternalTopic("sensors/temperature")

	if topic == nil {
		t.Fatal("AddExternalTopic() returned nil")
	}

	if topic.Name() != "sensors/temperature" {
		t.Errorf("topic name = %q, want 'sensors/temperature'", topic.Name())
	}

	if topic.Type() != TopicTypeExternal {
		t.Errorf("topic type = %v, want %v", topic.Type(), TopicTypeExternal)
	}

	// Check it's in the manager's maps
	if len(manager.externalTopics) != 1 {
		t.Error("external topic not added to externalTopics map")
	}

	if len(manager.topics) != 1 {
		t.Error("external topic not added to topics map")
	}

	// Adding same topic should return existing one
	topic2 := manager.AddExternalTopic("sensors/temperature")
	if topic2 != topic {
		t.Error("AddExternalTopic() should return existing topic for same name")
	}
}

func TestAddInternalTopic(t *testing.T) {
	manager := NewManager(nil)

	inputs := []string{"sensors/temp1", "sensors/temp2"}
	topic, err := manager.AddInternalTopic("calculated/average", inputs, nil, "test-strategy", false, false)

	if err != nil {
		t.Fatalf("AddInternalTopic() failed: %v", err)
	}

	if topic == nil {
		t.Fatal("AddInternalTopic() returned nil")
	}

	if topic.Name() != "calculated/average" {
		t.Errorf("topic name = %q, want 'calculated/average'", topic.Name())
	}

	if topic.Type() != TopicTypeInternal {
		t.Errorf("topic type = %v, want %v", topic.Type(), TopicTypeInternal)
	}

	if len(topic.GetInputs()) != 2 {
		t.Errorf("input count = %d, want 2", len(topic.GetInputs()))
	}

	if topic.GetStrategyID() != "test-strategy" {
		t.Errorf("strategy ID = %q, want 'test-strategy'", topic.GetStrategyID())
	}

	// Check it's in the manager's maps
	if len(manager.internalTopics) != 1 {
		t.Error("internal topic not added to internalTopics map")
	}

	if len(manager.topics) != 1 {
		t.Error("internal topic not added to topics map")
	}

	// Try to add duplicate
	_, err = manager.AddInternalTopic("calculated/average", inputs, nil, "other-strategy", false, false)
	if err == nil {
		t.Error("expected error when adding duplicate internal topic")
	}
}

func TestAddSystemTopic(t *testing.T) {
	manager := NewManager(nil)

	config := map[string]interface{}{
		"interval":    "5s",
		"description": "Test ticker",
	}

	topic := manager.AddSystemTopic("system/ticker/5s", config)

	if topic == nil {
		t.Fatal("AddSystemTopic() returned nil")
	}

	if topic.Name() != "system/ticker/5s" {
		t.Errorf("topic name = %q, want 'system/ticker/5s'", topic.Name())
	}

	if topic.Type() != TopicTypeSystem {
		t.Errorf("topic type = %v, want %v", topic.Type(), TopicTypeSystem)
	}

	// Check it's in the manager's maps
	if len(manager.systemTopics) != 1 {
		t.Error("system topic not added to systemTopics map")
	}

	if len(manager.topics) != 1 {
		t.Error("system topic not added to topics map")
	}

	// Adding same topic should return existing one
	topic2 := manager.AddSystemTopic("system/ticker/5s", config)
	if topic2 != topic {
		t.Error("AddSystemTopic() should return existing topic for same name")
	}
}

func TestRemoveTopic(t *testing.T) {
	manager := NewManager(nil)

	// Add topics of different types
	extTopic := manager.AddExternalTopic("sensors/temp")
	intTopic, _ := manager.AddInternalTopic("calc/avg", []string{"input"}, nil, "strategy", false, false)
	sysTopic := manager.AddSystemTopic("system/test", map[string]interface{}{})

	if len(manager.topics) != 3 {
		t.Fatalf("expected 3 topics, got %d", len(manager.topics))
	}

	// Remove external topic
	err := manager.RemoveTopic(extTopic.Name())
	if err != nil {
		t.Errorf("RemoveTopic() failed for external topic: %v", err)
	}

	if len(manager.externalTopics) != 0 {
		t.Error("external topic not removed from externalTopics map")
	}

	// Remove internal topic
	err = manager.RemoveTopic(intTopic.Name())
	if err != nil {
		t.Errorf("RemoveTopic() failed for internal topic: %v", err)
	}

	if len(manager.internalTopics) != 0 {
		t.Error("internal topic not removed from internalTopics map")
	}

	// Remove system topic
	err = manager.RemoveTopic(sysTopic.Name())
	if err != nil {
		t.Errorf("RemoveTopic() failed for system topic: %v", err)
	}

	if len(manager.systemTopics) != 0 {
		t.Error("system topic not removed from systemTopics map")
	}

	if len(manager.topics) != 0 {
		t.Error("topics not removed from main topics map")
	}

	// Try to remove non-existent topic
	err = manager.RemoveTopic("nonexistent")
	if err == nil {
		t.Error("expected error when removing non-existent topic")
	}
}

func TestGetTopic(t *testing.T) {
	manager := NewManager(nil)

	// Add test topic
	added := manager.AddExternalTopic("test/topic")

	// Get existing topic
	retrieved := manager.GetTopic("test/topic")
	if retrieved != added {
		t.Error("GetTopic() returned different instance")
	}

	// Get non-existent topic
	notFound := manager.GetTopic("nonexistent")
	if notFound != nil {
		t.Error("GetTopic() should return nil for non-existent topic")
	}
}

func TestListTopics(t *testing.T) {
	manager := NewManager(nil)

	// Add topics of different types
	manager.AddExternalTopic("sensors/temp")
	manager.AddInternalTopic("calc/avg", []string{"input"}, nil, "strategy", false, false)
	manager.AddSystemTopic("system/test", map[string]interface{}{})

	topics := manager.ListTopics()

	if len(topics) != 3 {
		t.Errorf("ListTopics() returned %d topics, want 3", len(topics))
	}

	// Verify all types are present
	typeCount := make(map[TopicType]int)
	for _, topic := range topics {
		typeCount[topic.Type()]++
	}

	if typeCount[TopicTypeExternal] != 1 {
		t.Errorf("expected 1 external topic, got %d", typeCount[TopicTypeExternal])
	}

	if typeCount[TopicTypeInternal] != 1 {
		t.Errorf("expected 1 internal topic, got %d", typeCount[TopicTypeInternal])
	}

	if typeCount[TopicTypeSystem] != 1 {
		t.Errorf("expected 1 system topic, got %d", typeCount[TopicTypeSystem])
	}
}

func TestListTopicsByType(t *testing.T) {
	manager := NewManager(nil)

	// Add multiple topics of different types
	manager.AddExternalTopic("sensors/temp1")
	manager.AddExternalTopic("sensors/temp2")
	manager.AddInternalTopic("calc/avg", []string{"input"}, nil, "strategy", false, false)
	manager.AddSystemTopic("system/test", map[string]interface{}{})

	// List external topics only
	externalTopics := manager.ListTopicsByType(TopicTypeExternal)
	if len(externalTopics) != 2 {
		t.Errorf("ListTopicsByType(External) returned %d topics, want 2", len(externalTopics))
	}

	// List internal topics only
	internalTopics := manager.ListTopicsByType(TopicTypeInternal)
	if len(internalTopics) != 1 {
		t.Errorf("ListTopicsByType(Internal) returned %d topics, want 1", len(internalTopics))
	}

	// List system topics only
	systemTopics := manager.ListTopicsByType(TopicTypeSystem)
	if len(systemTopics) != 1 {
		t.Errorf("ListTopicsByType(System) returned %d topics, want 1", len(systemTopics))
	}
}

func TestNotifyTopicUpdate(t *testing.T) {
	manager := NewManager(nil)

	// Set up mock strategy executor
	executionCalled := false
	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			executionCalled = true
			if strategyID != "test-strategy" {
				t.Errorf("strategy ID = %q, want 'test-strategy'", strategyID)
			}
			if triggerTopic != "sensors/temp" {
				t.Errorf("trigger topic = %q, want 'sensors/temp'", triggerTopic)
			}
			if len(inputs) != 1 {
				t.Errorf("inputs count = %d, want 1", len(inputs))
			}
			if inputs["sensors/temp"] != 25.5 {
				t.Errorf("input value = %v, want 25.5", inputs["sensors/temp"])
			}
			return "strategy result", nil
		},
	}
	manager.SetStrategyExecutor(mockExec)

	// Add external topic (source) and set its value
	extTopic := manager.AddExternalTopic("sensors/temp")
	extTopic.Emit(25.5) // Set the value first

	// Add internal topic that depends on the external topic
	_, err := manager.AddInternalTopic("processed/temp", []string{"sensors/temp"}, nil, "test-strategy", false, false)
	if err != nil {
		t.Fatalf("Failed to add internal topic: %v", err)
	}

	// Simulate topic update
	event := TopicEvent{
		TopicName:     "sensors/temp",
		Value:         25.5,
		PreviousValue: nil,
		Timestamp:     time.Now(),
		TriggerTopic:  "sensors/temp",
	}

	err = manager.NotifyTopicUpdate(event)
	if err != nil {
		t.Errorf("NotifyTopicUpdate() failed: %v", err)
	}

	if !executionCalled {
		t.Error("strategy execution was not triggered")
	}
}

func TestGetTopicCount(t *testing.T) {
	manager := NewManager(nil)

	counts := manager.GetTopicCount()
	if counts[TopicTypeExternal] != 0 || counts[TopicTypeInternal] != 0 || counts[TopicTypeSystem] != 0 {
		t.Error("expected all counts to be 0 initially")
	}

	// Add topics
	manager.AddExternalTopic("sensors/temp1")
	manager.AddExternalTopic("sensors/temp2")
	manager.AddInternalTopic("calc/avg", []string{"input"}, nil, "strategy", false, false)
	manager.AddSystemTopic("system/test", map[string]interface{}{})

	counts = manager.GetTopicCount()

	if counts[TopicTypeExternal] != 2 {
		t.Errorf("external count = %d, want 2", counts[TopicTypeExternal])
	}

	if counts[TopicTypeInternal] != 1 {
		t.Errorf("internal count = %d, want 1", counts[TopicTypeInternal])
	}

	if counts[TopicTypeSystem] != 1 {
		t.Errorf("system count = %d, want 1", counts[TopicTypeSystem])
	}
}

func TestSetStrategyExecutor(t *testing.T) {
	manager := NewManager(nil)
	mockExec := &mockStrategyExecutor{}

	manager.SetStrategyExecutor(mockExec)

	if manager.strategyExecutor == nil {
		t.Error("strategy executor not set")
	}
}

func TestSetStateManager(t *testing.T) {
	manager := NewManager(nil)
	mockState := &mockStateManager{}

	manager.SetStateManager(mockState)

	if manager.stateManager != mockState {
		t.Error("state manager not set")
	}
}

func TestExecuteStrategy(t *testing.T) {
	manager := NewManager(nil)

	// Test without strategy executor
	_, err := manager.ExecuteStrategy("test", map[string]interface{}{}, "topic", nil)
	if err == nil {
		t.Error("expected error when no strategy executor is set")
	}

	// Test with strategy executor
	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			return "test result", nil
		},
	}
	manager.SetStrategyExecutor(mockExec)

	emittedEvents, err := manager.ExecuteStrategy("test-strategy", map[string]interface{}{"input": 1}, "trigger", "last")
	if err != nil {
		t.Errorf("ExecuteStrategy() failed: %v", err)
	}

	if len(emittedEvents) != 1 {
		t.Fatalf("Expected 1 emitted event, got %d", len(emittedEvents))
	}

	if emittedEvents[0].Value != "test result" {
		t.Errorf("emitted value = %q, want 'test result'", emittedEvents[0].Value)
	}
}

func TestSaveTopicState(t *testing.T) {
	manager := NewManager(nil)

	// Test without state manager (should not error)
	err := manager.SaveTopicState("test/topic", "value")
	if err != nil {
		t.Errorf("SaveTopicState() failed without state manager: %v", err)
	}

	// Test with state manager
	saveCalled := false
	mockState := &mockStateManager{
		saveFunc: func(topicName string, value interface{}) error {
			saveCalled = true
			if topicName != "test/topic" {
				t.Errorf("topic name = %q, want 'test/topic'", topicName)
			}
			if value != "test value" {
				t.Errorf("value = %q, want 'test value'", value)
			}
			return nil
		},
	}
	manager.SetStateManager(mockState)

	err = manager.SaveTopicState("test/topic", "test value")
	if err != nil {
		t.Errorf("SaveTopicState() failed: %v", err)
	}

	if !saveCalled {
		t.Error("state manager save was not called")
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager(nil)

	// Run concurrent operations
	done := make(chan bool, 3)

	// Concurrent topic additions
	go func() {
		for i := 0; i < 50; i++ {
			manager.AddExternalTopic(fmt.Sprintf("topic%d", i))
		}
		done <- true
	}()

	// Concurrent topic listings
	go func() {
		for i := 0; i < 100; i++ {
			manager.ListTopics()
			manager.GetTopicCount()
		}
		done <- true
	}()

	// Concurrent topic updates
	go func() {
		for i := 0; i < 50; i++ {
			event := TopicEvent{
				TopicName: "topic0",
				Value:     i,
				Timestamp: time.Now(),
			}
			manager.NotifyTopicUpdate(event)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify manager is still functional
	topics := manager.ListTopics()
	if len(topics) < 1 {
		t.Error("manager corrupted after concurrent access")
	}
}

// TestInternalTopicTriggering tests that internal topics can trigger other internal topics
func TestInternalTopicTriggering(t *testing.T) {
	manager := NewManager(log.New(os.Stdout, "test: ", log.LstdFlags))

	// Track execution calls to verify triggering
	var parentExecuted, childExecuted bool
	var parentResult, childResult interface{}

	// Mock strategy executor that tracks execution
	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			switch strategyID {
			case "parent-strategy":
				parentExecuted = true
				parentResult = "parent-output"
				return parentResult, nil
			case "child-strategy":
				childExecuted = true
				childResult = fmt.Sprintf("child processed: %v", inputs)
				return childResult, nil
			default:
				return nil, fmt.Errorf("unknown strategy: %s", strategyID)
			}
		},
	}

	manager.SetStrategyExecutor(mockExec)
	manager.SetStateManager(&mockStateManager{})

	// Create parent topic that emits to subtopic
	_, err := manager.AddInternalTopic("tesla/mycar", []string{"sensors/battery_level"}, nil, "parent-strategy", false, false)
	if err != nil {
		t.Fatalf("Failed to create parent topic: %v", err)
	}

	// Create child topic that depends on parent's subtopic
	_, err = manager.AddInternalTopic("tesla/mycar/battery/alerts", []string{"tesla/mycar/battery"}, nil, "child-strategy", false, false)
	if err != nil {
		t.Fatalf("Failed to create child topic: %v", err)
	}

	// Create external topic to trigger the chain
	batteryTopic := manager.AddExternalTopic("sensors/battery_level")
	err = batteryTopic.Emit(25.0)
	if err != nil {
		t.Fatalf("Failed to emit to external topic: %v", err)
	}

	// Give some time for async processing
	time.Sleep(10 * time.Millisecond)

	// Verify parent was executed
	if !parentExecuted {
		t.Error("Parent topic was not executed")
	}

	// Verify child was executed (triggered by parent's subtopic emission)
	if !childExecuted {
		t.Error("Child topic was not executed - internal topic triggering failed")
	}

	// Verify the derived topic was created
	derivedTopic := manager.GetTopic("tesla/mycar/battery")
	if derivedTopic == nil {
		t.Error("Derived topic tesla/mycar/battery was not created")
	} else {
		if derivedTopic.LastValue() != "75%" {
			t.Errorf("Derived topic value = %v, want '75%%'", derivedTopic.LastValue())
		}
	}
}

// TestDerivedTopicCreationAndTriggering tests that derived topics are created and trigger other topics
func TestDerivedTopicCreationAndTriggering(t *testing.T) {
	manager := NewManager(log.New(os.Stdout, "test: ", log.LstdFlags))

	// Track which topics were executed and when
	var executionOrder []string
	var results = make(map[string]interface{})

	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			executionOrder = append(executionOrder, strategyID)

			switch strategyID {
			case "emitter-strategy":
				// Strategy that emits to multiple subtopics
				results[strategyID] = map[string]interface{}{
					"battery": 75,
					"speed":   55.5,
				}
				return results[strategyID], nil
			case "battery-monitor":
				results[strategyID] = fmt.Sprintf("Battery level: %v", inputs)
				return results[strategyID], nil
			case "speed-monitor":
				results[strategyID] = fmt.Sprintf("Speed: %v", inputs)
				return results[strategyID], nil
			default:
				return nil, fmt.Errorf("unknown strategy: %s", strategyID)
			}
		},
	}

	manager.SetStrategyExecutor(mockExec)
	manager.SetStateManager(&mockStateManager{})

	// Create main topic that emits to multiple subtopics
	_, err := manager.AddInternalTopic("vehicle/status", []string{"sensors/raw_data"}, nil, "emitter-strategy", false, false)
	if err != nil {
		t.Fatalf("Failed to create main topic: %v", err)
	}

	// Create topics that depend on the derived subtopics
	_, err = manager.AddInternalTopic("vehicle/battery/alerts", []string{"vehicle/status/battery"}, nil, "battery-monitor", false, false)
	if err != nil {
		t.Fatalf("Failed to create battery monitor: %v", err)
	}

	_, err = manager.AddInternalTopic("vehicle/speed/alerts", []string{"vehicle/status/speed"}, nil, "speed-monitor", false, false)
	if err != nil {
		t.Fatalf("Failed to create speed monitor: %v", err)
	}

	// Trigger the chain
	dataTopic := manager.AddExternalTopic("sensors/raw_data")
	err = dataTopic.Emit("test-data")
	if err != nil {
		t.Fatalf("Failed to emit to trigger topic: %v", err)
	}

	// Give time for async processing
	time.Sleep(20 * time.Millisecond)

	// Verify execution order
	expectedStrategies := []string{"emitter-strategy", "battery-monitor", "speed-monitor"}
	if len(executionOrder) != len(expectedStrategies) {
		t.Errorf("Expected %d strategy executions, got %d: %v", len(expectedStrategies), len(executionOrder), executionOrder)
	}

	// Verify main strategy executed first
	if len(executionOrder) > 0 && executionOrder[0] != "emitter-strategy" {
		t.Errorf("Expected emitter-strategy to execute first, got %s", executionOrder[0])
	}

	// Verify derived topics exist and have correct values
	batteryTopic := manager.GetTopic("vehicle/status/battery")
	speedTopic := manager.GetTopic("vehicle/status/speed")

	if batteryTopic == nil {
		t.Error("Battery derived topic was not created")
	} else if batteryTopic.LastValue() != 75 {
		t.Errorf("Battery topic value = %v, want 75", batteryTopic.LastValue())
	}

	if speedTopic == nil {
		t.Error("Speed derived topic was not created")
	} else if speedTopic.LastValue() != 55.5 {
		t.Errorf("Speed topic value = %v, want 55.5", speedTopic.LastValue())
	}

	// Verify dependent topics were triggered
	batteryAlerts := manager.GetTopic("vehicle/battery/alerts")
	speedAlerts := manager.GetTopic("vehicle/speed/alerts")

	if batteryAlerts == nil || batteryAlerts.LastValue() == nil {
		t.Error("Battery alerts topic was not triggered")
	}

	if speedAlerts == nil || speedAlerts.LastValue() == nil {
		t.Error("Speed alerts topic was not triggered")
	}
}

// TestDerivedTopicUpdateTriggering tests that updates to derived topics trigger dependent topics
func TestDerivedTopicUpdateTriggering(t *testing.T) {
	manager := NewManager(log.New(os.Stdout, "test: ", log.LstdFlags))

	var executionCount int
	var lastInput interface{}

	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			switch strategyID {
			case "source-strategy":
				return fmt.Sprintf("output-%d", executionCount), nil
			case "dependent-strategy":
				executionCount++
				lastInput = inputs
				return fmt.Sprintf("processed-%d", executionCount), nil
			}
			return nil, fmt.Errorf("unknown strategy: %s", strategyID)
		},
	}

	manager.SetStrategyExecutor(mockExec)
	manager.SetStateManager(&mockStateManager{})

	// Create source topic that emits to subtopic
	_, err := manager.AddInternalTopic("source", []string{"trigger"}, nil, "source-strategy", false, false)
	if err != nil {
		t.Fatalf("Failed to create source topic: %v", err)
	}

	// Create dependent topic
	_, err = manager.AddInternalTopic("dependent", []string{"source/output"}, nil, "dependent-strategy", false, false)
	if err != nil {
		t.Fatalf("Failed to create dependent topic: %v", err)
	}

	// First trigger - create external topic and trigger update
	triggerTopic := manager.AddExternalTopic("trigger")
	err = triggerTopic.Emit("first")
	if err != nil {
		t.Fatalf("Failed to trigger first time: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if executionCount != 1 {
		t.Errorf("Expected 1 execution after first trigger, got %d", executionCount)
	}

	// Second trigger - should cause another execution
	err = triggerTopic.Emit("second")
	if err != nil {
		t.Fatalf("Failed to trigger second time: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if executionCount != 2 {
		t.Errorf("Expected 2 executions after second trigger, got %d", executionCount)
	}

	// Verify the derived topic was updated
	derivedTopic := manager.GetTopic("source/output")
	if derivedTopic == nil {
		t.Error("Derived topic was not created")
	} else if derivedTopic.LastValue() != "output-1" {
		t.Errorf("Derived topic value = %v, want 'output-1'", derivedTopic.LastValue())
	}

	// Verify dependent topic was triggered with correct input
	dependentTopic := manager.GetTopic("dependent")
	if dependentTopic == nil {
		t.Error("Dependent topic was not created")
	} else if dependentTopic.LastValue() != "processed-2" {
		t.Errorf("Dependent topic value = %v, want 'processed-2'", dependentTopic.LastValue())
	}

	// Verify the dependent strategy received correct input
	if lastInput == nil {
		t.Error("Dependent strategy never received input")
	} else if inputMap, ok := lastInput.(map[string]interface{}); !ok {
		t.Error("Last input is not a map")
	} else if inputMap["source/output"] != "output-1" {
		t.Errorf("Last input source/output = %v, want 'output-1'", inputMap["source/output"])
	}
}
