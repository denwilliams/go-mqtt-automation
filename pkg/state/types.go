package state

import (
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
)

type Database interface {
	// Topics
	SaveTopic(config interface{}) error
	LoadTopic(name string) (interface{}, error)
	LoadAllTopics() ([]interface{}, error)
	DeleteTopic(name string) error

	// Strategies
	SaveStrategy(strategy *strategy.Strategy) error
	LoadStrategy(id string) (*strategy.Strategy, error)
	LoadAllStrategies() ([]*strategy.Strategy, error)
	DeleteStrategy(id string) error

	// State
	SaveState(key string, value interface{}) error
	LoadState(key string) (interface{}, error)
	DeleteState(key string) error

	// Execution logs
	SaveExecutionLog(log ExecutionLog) error
	LoadExecutionLogs(topicName string, limit int) ([]ExecutionLog, error)

	// Maintenance
	Close() error
	Migrate() error
}

type ExecutionLog struct {
	ID              int                    `db:"id"`
	TopicName       string                 `db:"topic_name"`
	StrategyID      string                 `db:"strategy_id"`
	TriggerTopic    string                 `db:"trigger_topic"`
	InputValues     map[string]interface{} `db:"input_values"`
	OutputValues    interface{}            `db:"output_values"`
	ErrorMessage    string                 `db:"error_message"`
	ExecutionTimeMs int64                  `db:"execution_time_ms"`
	ExecutedAt      time.Time              `db:"executed_at"`
}

type TopicState struct {
	Name      string      `db:"name"`
	Value     interface{} `db:"value"`
	UpdatedAt time.Time   `db:"updated_at"`
}
