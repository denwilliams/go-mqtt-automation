package topics

import (
	"encoding/json"
	"fmt"
	"time"
)

type ExternalTopic struct {
	config  BaseTopicConfig
	manager *Manager
}

func NewExternalTopic(name string) *ExternalTopic {
	return &ExternalTopic{
		config: BaseTopicConfig{
			Name:        name,
			Type:        TopicTypeExternal,
			CreatedAt:   time.Now(),
			LastUpdated: time.Time{},
			Config:      make(map[string]interface{}),
		},
	}
}

func (et *ExternalTopic) Name() string {
	return et.config.Name
}

func (et *ExternalTopic) Type() TopicType {
	return TopicTypeExternal
}

func (et *ExternalTopic) LastValue() interface{} {
	return et.config.LastValue
}

func (et *ExternalTopic) LastUpdated() time.Time {
	return et.config.LastUpdated
}

func (et *ExternalTopic) SetManager(manager *Manager) {
	et.manager = manager
}

func (et *ExternalTopic) Emit(value interface{}) error {
	previousValue := et.config.LastValue
	et.config.LastValue = value
	et.config.LastUpdated = time.Now()

	if et.manager != nil {
		event := TopicEvent{
			TopicName:     et.config.Name,
			Value:         value,
			PreviousValue: previousValue,
			Timestamp:     et.config.LastUpdated,
			TriggerTopic:  et.config.Name,
		}

		if err := et.manager.NotifyTopicUpdate(event); err != nil {
			return fmt.Errorf("failed to notify topic update: %w", err)
		}

		// Save state to database
		if err := et.manager.SaveTopicState(et.config.Name, value); err != nil {
			return fmt.Errorf("failed to save topic state: %w", err)
		}
	}

	return nil
}

func (et *ExternalTopic) UpdateFromMQTT(payload []byte) error {
	// Try to parse as JSON first, fall back to string
	var value interface{}
	if err := json.Unmarshal(payload, &value); err != nil {
		// If JSON parsing fails, treat as string
		value = string(payload)
	}

	return et.Emit(value)
}

func (et *ExternalTopic) GetConfig() BaseTopicConfig {
	return et.config
}

func (et *ExternalTopic) UpdateConfig(config BaseTopicConfig) {
	et.config = config
}
