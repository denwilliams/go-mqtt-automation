package strategy

import (
	"time"
)

type Strategy struct {
	ID                string                 `json:"id" db:"id"`
	Name              string                 `json:"name" db:"name"`
	Description       string                 `json:"description" db:"description"`
	Code              string                 `json:"code" db:"code"`
	Language          string                 `json:"language" db:"language"`
	Builtin           bool                   `json:"builtin" db:"builtin"`
	Parameters        map[string]interface{} `json:"parameters" db:"parameters"`
	MaxInputs         int                    `json:"max_inputs" db:"max_inputs"`
	DefaultInputNames []string               `json:"default_input_names" db:"default_input_names"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

type ExecutionContext struct {
	InputValues     map[string]interface{} `json:"input_values"`
	InputNames      map[string]string      `json:"input_names,omitempty"`
	TriggeringTopic string                 `json:"triggering_topic"`
	TriggeringValue interface{}            `json:"triggering_value"`
	LastOutputs     interface{}            `json:"last_outputs"`
	Parameters      map[string]interface{} `json:"parameters"`
	TopicName       string                 `json:"topic_name"`
}

type ExecutionResult struct {
	Result        interface{}   `json:"result"`
	Error         error         `json:"error,omitempty"`
	LogMessages   []string      `json:"log_messages,omitempty"`
	EmittedEvents []EmitEvent   `json:"emitted_events,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
}

type EmitEvent struct {
	Topic string      `json:"topic"`
	Value interface{} `json:"value"`
}

type LanguageExecutor interface {
	Execute(strategy *Strategy, context ExecutionContext) ExecutionResult
	Validate(code string) error
}
