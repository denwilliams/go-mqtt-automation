package topics

import (
	"encoding/json"
	"time"
)

type TopicType string

const (
	TopicTypeExternal TopicType = "external"
	TopicTypeInternal TopicType = "internal"
	TopicTypeSystem   TopicType = "system"
)

type Topic interface {
	Name() string
	Type() TopicType
	LastValue() interface{}
	LastUpdated() time.Time
	Emit(value interface{}) error
	SetManager(manager *Manager)
}

type BaseTopicConfig struct {
	Name        string                 `json:"name" db:"name"`
	Type        TopicType              `json:"type" db:"type"`
	LastValue   interface{}            `json:"last_value" db:"last_value"`
	LastUpdated time.Time              `json:"last_updated" db:"last_updated"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	Config      map[string]interface{} `json:"config" db:"config"`
}

type InternalTopicConfig struct {
	BaseTopicConfig
	Inputs        []string               `json:"inputs" db:"inputs"`
	InputNames    map[string]string      `json:"input_names,omitempty" db:"input_names"`
	StrategyID    string                 `json:"strategy_id" db:"strategy_id"`
	Parameters    map[string]interface{} `json:"parameters,omitempty" db:"parameters"`
	EmitToMQTT    bool                   `json:"emit_to_mqtt" db:"emit_to_mqtt"`
	NoOpUnchanged bool                   `json:"noop_unchanged" db:"noop_unchanged"`
}

type SystemTopicConfig struct {
	BaseTopicConfig
	Interval string `json:"interval,omitempty"`
	Cron     string `json:"cron,omitempty"`
}

type TopicEvent struct {
	TopicName     string
	Value         interface{}
	PreviousValue interface{}
	Timestamp     time.Time
	TriggerTopic  string
}

func (btc *BaseTopicConfig) MarshalConfig() (string, error) {
	data, err := json.Marshal(btc.Config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (btc *BaseTopicConfig) UnmarshalConfig(data string) error {
	if data == "" {
		btc.Config = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(data), &btc.Config)
}

func (itc *InternalTopicConfig) MarshalInputs() (string, error) {
	if len(itc.Inputs) == 0 {
		return "", nil
	}
	data, err := json.Marshal(itc.Inputs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (itc *InternalTopicConfig) UnmarshalInputs(data string) error {
	if data == "" {
		itc.Inputs = []string{}
		return nil
	}
	return json.Unmarshal([]byte(data), &itc.Inputs)
}
