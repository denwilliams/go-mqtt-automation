package topics

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
)

type InternalTopic struct {
	config  InternalTopicConfig
	manager *Manager
}

func NewInternalTopic(name string, inputs []string, strategyID string) *InternalTopic {
	return &InternalTopic{
		config: InternalTopicConfig{
			BaseTopicConfig: BaseTopicConfig{
				Name:        name,
				Type:        TopicTypeInternal,
				CreatedAt:   time.Now(),
				LastUpdated: time.Time{},
				Config:      make(map[string]interface{}),
			},
			Inputs:        inputs,
			StrategyID:    strategyID,
			EmitToMQTT:    false,
			NoOpUnchanged: false,
		},
	}
}

func (it *InternalTopic) Name() string {
	return it.config.Name
}

func (it *InternalTopic) Type() TopicType {
	return TopicTypeInternal
}

func (it *InternalTopic) LastValue() interface{} {
	return it.config.LastValue
}

func (it *InternalTopic) LastUpdated() time.Time {
	return it.config.LastUpdated
}

func (it *InternalTopic) SetManager(manager *Manager) {
	it.manager = manager
}

func (it *InternalTopic) GetInputs() []string {
	return it.config.Inputs
}

func (it *InternalTopic) GetStrategyID() string {
	return it.config.StrategyID
}

func (it *InternalTopic) ShouldEmitToMQTT() bool {
	return it.config.EmitToMQTT
}

func (it *InternalTopic) IsNoOpUnchanged() bool {
	return it.config.NoOpUnchanged
}

func (it *InternalTopic) Emit(value interface{}) error {
	previousValue := it.config.LastValue

	// Check if we should skip unchanged values
	if it.config.NoOpUnchanged && it.valuesEqual(value, previousValue) {
		return nil // Skip emission
	}

	it.config.LastValue = value
	it.config.LastUpdated = time.Now()

	if it.manager != nil {
		event := TopicEvent{
			TopicName:     it.config.Name,
			Value:         value,
			PreviousValue: previousValue,
			Timestamp:     it.config.LastUpdated,
			TriggerTopic:  it.config.Name,
		}

		// Emit to MQTT if configured
		if it.config.EmitToMQTT {
			if err := it.emitToMQTT(value); err != nil {
				return fmt.Errorf("failed to emit to MQTT: %w", err)
			}
		}

		if err := it.manager.NotifyTopicUpdate(event); err != nil {
			return fmt.Errorf("failed to notify topic update: %w", err)
		}

		// Save state to database
		if err := it.manager.SaveTopicState(it.config.Name, value); err != nil {
			return fmt.Errorf("failed to save topic state: %w", err)
		}
	}

	return nil
}

func (it *InternalTopic) ProcessInputs(triggerTopic string) error {
	if it.manager == nil {
		return fmt.Errorf("topic manager not set")
	}

	// Collect input values
	inputValues := make(map[string]interface{})
	for _, inputTopic := range it.config.Inputs {
		topic := it.manager.GetTopic(inputTopic)
		if topic != nil {
			inputValues[inputTopic] = topic.LastValue()
		} else {
			inputValues[inputTopic] = nil
		}
	}

	// Execute strategy
	emittedEvents, err := it.manager.ExecuteStrategy(it.config.StrategyID, inputValues, triggerTopic, it.config.LastValue)
	if err != nil {
		return fmt.Errorf("strategy execution failed: %w", err)
	}

	// Process all emitted events
	return it.processEmittedEvents(emittedEvents)
}

func (it *InternalTopic) emitToMQTT(value interface{}) error {
	if it.manager == nil || it.manager.mqttClient == nil {
		return fmt.Errorf("MQTT client not available")
	}

	// Serialize value to JSON
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Publish to MQTT
	return it.manager.mqttClient.Publish(it.config.Name, payload, false)
}

func (it *InternalTopic) valuesEqual(a, b interface{}) bool {
	// Use reflection for deep comparison
	return reflect.DeepEqual(a, b)
}

func (it *InternalTopic) GetConfig() InternalTopicConfig {
	return it.config
}

func (it *InternalTopic) UpdateConfig(config InternalTopicConfig) {
	it.config = config
}

func (it *InternalTopic) SetEmitToMQTT(emit bool) {
	it.config.EmitToMQTT = emit
}

func (it *InternalTopic) SetNoOpUnchanged(noop bool) {
	it.config.NoOpUnchanged = noop
}

func (it *InternalTopic) processEmittedEvents(events []strategy.EmitEvent) error {
	for _, event := range events {
		if event.Topic == "" {
			// Empty topic means main topic (this internal topic)
			if err := it.Emit(event.Value); err != nil {
				return fmt.Errorf("failed to emit to main topic: %w", err)
			}
		} else {
			// Handle subtopic emission
			if err := it.emitToSubtopic(event.Topic, event.Value); err != nil {
				return fmt.Errorf("failed to emit to subtopic %s: %w", event.Topic, err)
			}
		}
	}
	return nil
}

func (it *InternalTopic) emitToSubtopic(topicPath string, value interface{}) error {
	if it.manager == nil {
		return fmt.Errorf("manager not available")
	}

	// Determine the full topic name
	var fullTopicName string
	if strings.HasPrefix(topicPath, "/") {
		// Relative path - append to current topic name
		fullTopicName = it.config.Name + topicPath
	} else {
		// Absolute path - use as is
		fullTopicName = topicPath
	}

	// Create or update the subtopic
	return it.manager.createOrUpdateExternalTopic(fullTopicName, value)
}

func (it *InternalTopic) SetStrategyID(strategyID string) {
	it.config.StrategyID = strategyID
}

func (it *InternalTopic) AddInput(inputTopic string) {
	for _, existing := range it.config.Inputs {
		if existing == inputTopic {
			return // Already exists
		}
	}
	it.config.Inputs = append(it.config.Inputs, inputTopic)
}

func (it *InternalTopic) RemoveInput(inputTopic string) {
	for i, existing := range it.config.Inputs {
		if existing == inputTopic {
			it.config.Inputs = append(it.config.Inputs[:i], it.config.Inputs[i+1:]...)
			return
		}
	}
}
