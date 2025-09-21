package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDatabase struct {
	db   *sql.DB
	path string
}

func NewSQLiteDatabase(dbPath string) (*SQLiteDatabase, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	sqliteDB := &SQLiteDatabase{
		db:   db,
		path: dbPath,
	}

	return sqliteDB, nil
}

func (s *SQLiteDatabase) Migrate() error {
	// Create a separate database connection for migrations to avoid connection interference
	migrationDB, err := sql.Open("sqlite3", s.path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to open migration database: %w", err)
	}
	defer migrationDB.Close()

	// Create sqlite3 driver instance
	driver, err := sqlite3.WithInstance(migrationDB, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite3 driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations/sqlite",
		"sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Migration helper methods removed - now handled by golang-migrate

func (s *SQLiteDatabase) Close() error {
	return s.db.Close()
}

// Topics
func (s *SQLiteDatabase) SaveTopic(config interface{}) error {
	switch t := config.(type) {
	case topics.BaseTopicConfig:
		return s.saveBaseTopic(t)
	case topics.InternalTopicConfig:
		return s.saveInternalTopic(t)
	case topics.SystemTopicConfig:
		return s.saveSystemTopic(t)
	default:
		return fmt.Errorf("unsupported topic config type: %T", config)
	}
}

func (s *SQLiteDatabase) saveBaseTopic(config topics.BaseTopicConfig) error {
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	valueJSON, err := json.Marshal(config.LastValue)
	if err != nil {
		return fmt.Errorf("failed to marshal last value: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO topics (name, type, last_value, last_updated, config, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		config.Name,
		string(config.Type),
		string(valueJSON),
		config.LastUpdated,
		string(configJSON),
		config.CreatedAt,
	)

	return err
}

func (s *SQLiteDatabase) saveInternalTopic(config topics.InternalTopicConfig) error {
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	valueJSON, err := json.Marshal(config.LastValue)
	if err != nil {
		return fmt.Errorf("failed to marshal last value: %w", err)
	}

	inputsJSON, err := json.Marshal(config.Inputs)
	if err != nil {
		return fmt.Errorf("failed to marshal inputs: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO topics (name, type, inputs, strategy_id, emit_to_mqtt, noop_unchanged, last_value, last_updated, config, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		config.Name,
		string(config.Type),
		string(inputsJSON),
		config.StrategyID,
		config.EmitToMQTT,
		config.NoOpUnchanged,
		string(valueJSON),
		config.LastUpdated,
		string(configJSON),
		config.CreatedAt,
	)

	return err
}

func (s *SQLiteDatabase) saveSystemTopic(config topics.SystemTopicConfig) error {
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	valueJSON, err := json.Marshal(config.LastValue)
	if err != nil {
		return fmt.Errorf("failed to marshal last value: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO topics (name, type, last_value, last_updated, config, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		config.Name,
		string(config.Type),
		string(valueJSON),
		config.LastUpdated,
		string(configJSON),
		config.CreatedAt,
	)

	return err
}

func (s *SQLiteDatabase) LoadTopic(name string) (interface{}, error) {
	query := `
		SELECT name, type, inputs, strategy_id, emit_to_mqtt, noop_unchanged, 
		       last_value, last_updated, config, created_at
		FROM topics WHERE name = ?
	`

	row := s.db.QueryRow(query, name)

	var topicName, topicType string
	var inputs, strategyID sql.NullString
	var emitToMQTT, noopUnchanged sql.NullBool
	var lastValue sql.NullString
	var config string
	var lastUpdated, createdAt time.Time

	err := row.Scan(&topicName, &topicType, &inputs, &strategyID,
		&emitToMQTT, &noopUnchanged, &lastValue, &lastUpdated, &config, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("topic not found: %s", name)
		}
		return nil, fmt.Errorf("failed to scan topic: %w", err)
	}

	return s.buildTopicConfig(topicName, topicType, inputs, strategyID,
		emitToMQTT, noopUnchanged, lastValue, lastUpdated, createdAt, config)
}

func (s *SQLiteDatabase) LoadAllTopics() ([]interface{}, error) {
	query := `
		SELECT name, type, inputs, strategy_id, emit_to_mqtt, noop_unchanged, 
		       last_value, last_updated, config, created_at
		FROM topics ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query topics: %w", err)
	}
	defer rows.Close()

	var topics []interface{}

	for rows.Next() {
		var topicName, topicType string
		var inputs, strategyID sql.NullString
		var emitToMQTT, noopUnchanged sql.NullBool
		var lastValue sql.NullString
		var config string
		var lastUpdated, createdAt time.Time

		err := rows.Scan(&topicName, &topicType, &inputs, &strategyID,
			&emitToMQTT, &noopUnchanged, &lastValue, &lastUpdated, &config, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan topic row: %w", err)
		}

		topicConfig, err := s.buildTopicConfig(topicName, topicType, inputs, strategyID,
			emitToMQTT, noopUnchanged, lastValue, lastUpdated, createdAt, config)
		if err != nil {
			return nil, err
		}

		topics = append(topics, topicConfig)
	}

	return topics, nil
}

func (s *SQLiteDatabase) buildTopicConfig(name, topicType string, inputs, strategyID sql.NullString,
	emitToMQTT, noopUnchanged sql.NullBool, lastValue sql.NullString, lastUpdated, createdAt time.Time,
	config string) (interface{}, error) {

	// Parse common fields
	var parsedLastValue interface{}
	if lastValue.Valid && lastValue.String != "" {
		if err := json.Unmarshal([]byte(lastValue.String), &parsedLastValue); err != nil {
			parsedLastValue = nil
		}
	} else {
		parsedLastValue = nil
	}

	var parsedConfig map[string]interface{}
	if err := json.Unmarshal([]byte(config), &parsedConfig); err != nil {
		parsedConfig = make(map[string]interface{})
	}

	baseConfig := topics.BaseTopicConfig{
		Name:        name,
		Type:        topics.TopicType(topicType),
		LastValue:   parsedLastValue,
		LastUpdated: lastUpdated,
		CreatedAt:   createdAt,
		Config:      parsedConfig,
	}

	switch topics.TopicType(topicType) {
	case topics.TopicTypeInternal:
		var parsedInputs []string
		if inputs.Valid {
			if err := json.Unmarshal([]byte(inputs.String), &parsedInputs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal inputs: %w", err)
			}
		}

		return topics.InternalTopicConfig{
			BaseTopicConfig: baseConfig,
			Inputs:          parsedInputs,
			StrategyID:      strategyID.String,
			EmitToMQTT:      emitToMQTT.Bool,
			NoOpUnchanged:   noopUnchanged.Bool,
		}, nil

	case topics.TopicTypeSystem:
		interval, _ := parsedConfig["interval"].(string)
		cron, _ := parsedConfig["cron"].(string)

		return topics.SystemTopicConfig{
			BaseTopicConfig: baseConfig,
			Interval:        interval,
			Cron:            cron,
		}, nil

	default:
		return baseConfig, nil
	}
}

func (s *SQLiteDatabase) DeleteTopic(name string) error {
	_, err := s.db.Exec("DELETE FROM topics WHERE name = ?", name)
	return err
}

// Strategies
func (s *SQLiteDatabase) SaveStrategy(strategy *strategy.Strategy) error {
	parametersJSON, err := json.Marshal(strategy.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO strategies (id, name, code, language, parameters, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		strategy.ID,
		strategy.Name,
		strategy.Code,
		strategy.Language,
		string(parametersJSON),
		strategy.CreatedAt,
		strategy.UpdatedAt,
	)

	return err
}

func (s *SQLiteDatabase) LoadStrategy(id string) (*strategy.Strategy, error) {
	query := `
		SELECT id, name, code, language, parameters, max_inputs, default_input_names, created_at, updated_at
		FROM strategies WHERE id = ?
	`

	row := s.db.QueryRow(query, id)

	var strat strategy.Strategy
	var parametersJSON string
	var maxInputs sql.NullInt64
	var defaultInputNamesJSON sql.NullString

	err := row.Scan(&strat.ID, &strat.Name, &strat.Code, &strat.Language,
		&parametersJSON, &maxInputs, &defaultInputNamesJSON, &strat.CreatedAt, &strat.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("strategy not found: %s", id)
		}
		return nil, fmt.Errorf("failed to scan strategy: %w", err)
	}

	if err := json.Unmarshal([]byte(parametersJSON), &strat.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	// Handle max_inputs
	if maxInputs.Valid {
		strat.MaxInputs = int(maxInputs.Int64)
	}

	// Handle default_input_names
	if defaultInputNamesJSON.Valid && defaultInputNamesJSON.String != "" {
		if err := json.Unmarshal([]byte(defaultInputNamesJSON.String), &strat.DefaultInputNames); err != nil {
			return nil, fmt.Errorf("failed to unmarshal default_input_names: %w", err)
		}
	}

	return &strat, nil
}

func (s *SQLiteDatabase) LoadAllStrategies() ([]*strategy.Strategy, error) {
	query := `
		SELECT id, name, code, language, parameters, max_inputs, default_input_names, created_at, updated_at
		FROM strategies ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query strategies: %w", err)
	}
	defer rows.Close()

	var strategies []*strategy.Strategy

	for rows.Next() {
		var strat strategy.Strategy
		var parametersJSON string
		var maxInputs sql.NullInt64
		var defaultInputNamesJSON sql.NullString

		err := rows.Scan(&strat.ID, &strat.Name, &strat.Code, &strat.Language,
			&parametersJSON, &maxInputs, &defaultInputNamesJSON, &strat.CreatedAt, &strat.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy row: %w", err)
		}

		if err := json.Unmarshal([]byte(parametersJSON), &strat.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		// Handle max_inputs
		if maxInputs.Valid {
			strat.MaxInputs = int(maxInputs.Int64)
		}

		// Handle default_input_names
		if defaultInputNamesJSON.Valid && defaultInputNamesJSON.String != "" {
			if err := json.Unmarshal([]byte(defaultInputNamesJSON.String), &strat.DefaultInputNames); err != nil {
				return nil, fmt.Errorf("failed to unmarshal default_input_names: %w", err)
			}
		}

		strategies = append(strategies, &strat)
	}

	return strategies, nil
}

func (s *SQLiteDatabase) DeleteStrategy(id string) error {
	_, err := s.db.Exec("DELETE FROM strategies WHERE id = ?", id)
	return err
}

// State
func (s *SQLiteDatabase) SaveState(key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO state (key, value, updated_at)
		VALUES (?, ?, ?)
	`

	_, err = s.db.Exec(query, key, string(valueJSON), time.Now())
	return err
}

func (s *SQLiteDatabase) LoadState(key string) (interface{}, error) {
	query := "SELECT value FROM state WHERE key = ?"

	var valueJSON string
	err := s.db.QueryRow(query, key).Scan(&valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("state key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to scan state: %w", err)
	}

	var value interface{}
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return value, nil
}

func (s *SQLiteDatabase) DeleteState(key string) error {
	_, err := s.db.Exec("DELETE FROM state WHERE key = ?", key)
	return err
}

// Execution logs
func (s *SQLiteDatabase) SaveExecutionLog(log ExecutionLog) error {
	inputJSON, err := json.Marshal(log.InputValues)
	if err != nil {
		return fmt.Errorf("failed to marshal input values: %w", err)
	}

	outputJSON, err := json.Marshal(log.OutputValues)
	if err != nil {
		return fmt.Errorf("failed to marshal output values: %w", err)
	}

	query := `
		INSERT INTO execution_log (topic_name, strategy_id, trigger_topic, input_values, 
		                          output_values, error_message, execution_time_ms, executed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		log.TopicName,
		log.StrategyID,
		log.TriggerTopic,
		string(inputJSON),
		string(outputJSON),
		log.ErrorMessage,
		log.ExecutionTimeMs,
		log.ExecutedAt,
	)

	return err
}

func (s *SQLiteDatabase) LoadExecutionLogs(topicName string, limit int) ([]ExecutionLog, error) {
	query := `
		SELECT id, topic_name, strategy_id, trigger_topic, input_values, 
		       output_values, error_message, execution_time_ms, executed_at
		FROM execution_log 
		WHERE topic_name = ? 
		ORDER BY executed_at DESC 
		LIMIT ?
	`

	rows, err := s.db.Query(query, topicName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution logs: %w", err)
	}
	defer rows.Close()

	var logs []ExecutionLog

	for rows.Next() {
		var log ExecutionLog
		var inputJSON, outputJSON string

		err := rows.Scan(&log.ID, &log.TopicName, &log.StrategyID, &log.TriggerTopic,
			&inputJSON, &outputJSON, &log.ErrorMessage, &log.ExecutionTimeMs, &log.ExecutedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution log row: %w", err)
		}

		if err := json.Unmarshal([]byte(inputJSON), &log.InputValues); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input values: %w", err)
		}

		if err := json.Unmarshal([]byte(outputJSON), &log.OutputValues); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output values: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, nil
}
