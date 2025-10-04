package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type PostgreSQLDatabase struct {
	db  *sql.DB
	dsn string
}

func NewPostgreSQLDatabase(dsn string) (*PostgreSQLDatabase, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set reasonable connection limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	pgDB := &PostgreSQLDatabase{
		db:  db,
		dsn: dsn,
	}

	return pgDB, nil
}

func (p *PostgreSQLDatabase) Migrate() error {
	// Create a separate database connection for migrations to avoid connection interference
	migrationDB, err := sql.Open("postgres", p.dsn)
	if err != nil {
		return fmt.Errorf("failed to open migration database: %w", err)
	}
	defer migrationDB.Close()

	// Create postgres driver instance
	driver, err := postgres.WithInstance(migrationDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations/postgres",
		"postgres", driver)
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

func (p *PostgreSQLDatabase) Close() error {
	return p.db.Close()
}

// Topics
func (p *PostgreSQLDatabase) SaveTopic(config interface{}) error {
	switch t := config.(type) {
	case topics.BaseTopicConfig:
		return p.saveBaseTopic(t)
	case topics.InternalTopicConfig:
		return p.saveInternalTopic(t)
	case topics.SystemTopicConfig:
		return p.saveSystemTopic(t)
	default:
		return fmt.Errorf("unsupported topic config type: %T", config)
	}
}

func (p *PostgreSQLDatabase) saveBaseTopic(config topics.BaseTopicConfig) error {
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	var lastValueJSON sql.NullString
	if config.LastValue != nil {
		lastValueBytes, marshalErr := json.Marshal(config.LastValue)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal last value: %w", marshalErr)
		}
		lastValueJSON = sql.NullString{String: string(lastValueBytes), Valid: true}
	}

	query := `
		INSERT INTO topics (name, type, last_value, last_updated, config, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name) 
		DO UPDATE SET 
			type = EXCLUDED.type,
			last_value = EXCLUDED.last_value,
			last_updated = EXCLUDED.last_updated,
			config = EXCLUDED.config
	`

	_, err = p.db.Exec(query, config.Name, config.Type, lastValueJSON, config.LastUpdated, string(configJSON), config.CreatedAt)
	return err
}

func (p *PostgreSQLDatabase) saveInternalTopic(config topics.InternalTopicConfig) error {
	// First save the base topic data
	if err := p.saveBaseTopic(config.BaseTopicConfig); err != nil {
		return err
	}

	inputsJSON, err := json.Marshal(config.Inputs)
	if err != nil {
		return fmt.Errorf("failed to marshal inputs: %w", err)
	}

	inputNamesJSON, err := json.Marshal(config.InputNames)
	if err != nil {
		return fmt.Errorf("failed to marshal input names: %w", err)
	}

	parametersJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		UPDATE topics
		SET inputs = $1, input_names = $2, strategy_id = $3, parameters = $4, emit_to_mqtt = $5, noop_unchanged = $6
		WHERE name = $7
	`

	_, err = p.db.Exec(query, string(inputsJSON), string(inputNamesJSON), config.StrategyID, string(parametersJSON), config.EmitToMQTT, config.NoOpUnchanged, config.Name)
	return err
}

func (p *PostgreSQLDatabase) saveSystemTopic(config topics.SystemTopicConfig) error {
	return p.saveBaseTopic(config.BaseTopicConfig)
}

func (p *PostgreSQLDatabase) LoadTopic(name string) (interface{}, error) {
	query := `
		SELECT name, type, inputs, input_names, strategy_id, parameters, emit_to_mqtt, noop_unchanged,
		       last_value, last_updated, created_at, config
		FROM topics
		WHERE name = $1
	`

	var topicName, topicType string
	var inputs, inputNames, strategyID, parameters sql.NullString
	var emitToMQTT, noopUnchanged sql.NullBool
	var lastValue sql.NullString
	var lastUpdated, createdAt time.Time
	var config string

	err := p.db.QueryRow(query, name).Scan(
		&topicName, &topicType, &inputs, &inputNames, &strategyID, &parameters,
		&emitToMQTT, &noopUnchanged, &lastValue, &lastUpdated, &createdAt, &config,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load topic: %w", err)
	}

	return p.buildTopicConfig(topicName, topicType, inputs, inputNames, strategyID, parameters,
		emitToMQTT, noopUnchanged, lastValue, lastUpdated, createdAt, config)
}

func (p *PostgreSQLDatabase) LoadAllTopics() ([]interface{}, error) {
	query := `
		SELECT name, type, inputs, input_names, strategy_id, parameters, emit_to_mqtt, noop_unchanged,
		       last_value, last_updated, created_at, config
		FROM topics
		ORDER BY name
	`

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query topics: %w", err)
	}
	defer rows.Close()

	var topics []interface{}
	for rows.Next() {
		var topicName, topicType string
		var inputs, inputNames, strategyID, parameters sql.NullString
		var emitToMQTT, noopUnchanged sql.NullBool
		var lastValue sql.NullString
		var lastUpdated, createdAt time.Time
		var config string

		err := rows.Scan(
			&topicName, &topicType, &inputs, &inputNames, &strategyID, &parameters,
			&emitToMQTT, &noopUnchanged, &lastValue, &lastUpdated, &createdAt, &config,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}

		topicConfig, err := p.buildTopicConfig(topicName, topicType, inputs, inputNames, strategyID, parameters,
			emitToMQTT, noopUnchanged, lastValue, lastUpdated, createdAt, config)
		if err != nil {
			return nil, fmt.Errorf("failed to build topic config for %s: %w", topicName, err)
		}

		topics = append(topics, topicConfig)
	}

	return topics, rows.Err()
}

func (p *PostgreSQLDatabase) buildTopicConfig(name, topicType string, inputs, inputNames, strategyID, parameters sql.NullString,
	emitToMQTT, noopUnchanged sql.NullBool, lastValue sql.NullString, lastUpdated, createdAt time.Time,
	config string) (interface{}, error) {

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

	switch topicType {
	case "internal":
		var parsedInputs []string
		if inputs.Valid && inputs.String != "" {
			if err := json.Unmarshal([]byte(inputs.String), &parsedInputs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal inputs: %w", err)
			}
		}

		var parsedInputNames map[string]string
		if inputNames.Valid && inputNames.String != "" {
			if err := json.Unmarshal([]byte(inputNames.String), &parsedInputNames); err != nil {
				return nil, fmt.Errorf("failed to unmarshal input names: %w", err)
			}
		}

		var parsedParameters map[string]interface{}
		if parameters.Valid && parameters.String != "" {
			if err := json.Unmarshal([]byte(parameters.String), &parsedParameters); err != nil {
				return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
			}
		}

		return topics.InternalTopicConfig{
			BaseTopicConfig: baseConfig,
			Inputs:          parsedInputs,
			InputNames:      parsedInputNames,
			StrategyID:      strategyID.String,
			Parameters:      parsedParameters,
			EmitToMQTT:      emitToMQTT.Bool,
			NoOpUnchanged:   noopUnchanged.Bool,
		}, nil

	case "system":
		return topics.SystemTopicConfig{
			BaseTopicConfig: baseConfig,
		}, nil

	case "external":
		return baseConfig, nil

	default:
		return nil, fmt.Errorf("unknown topic type: %s", topicType)
	}
}

func (p *PostgreSQLDatabase) DeleteTopic(name string) error {
	query := "DELETE FROM topics WHERE name = $1"
	_, err := p.db.Exec(query, name)
	return err
}

// Strategies
func (p *PostgreSQLDatabase) SaveStrategy(strategy *strategy.Strategy) error {
	parametersJSON, err := json.Marshal(strategy.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO strategies (id, name, description, code, language, parameters, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id)
		DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			code = EXCLUDED.code,
			language = EXCLUDED.language,
			parameters = EXCLUDED.parameters,
			updated_at = EXCLUDED.updated_at
	`

	_, err = p.db.Exec(query, strategy.ID, strategy.Name, strategy.Description, strategy.Code, strategy.Language,
		string(parametersJSON), strategy.CreatedAt, strategy.UpdatedAt)
	return err
}

func (p *PostgreSQLDatabase) LoadStrategy(id string) (*strategy.Strategy, error) {
	query := `
		SELECT id, name, description, code, language, parameters, max_inputs, default_input_names, created_at, updated_at
		FROM strategies
		WHERE id = $1
	`

	var strat strategy.Strategy
	var parametersJSON string
	var maxInputs sql.NullInt64
	var defaultInputNamesJSON sql.NullString

	err := p.db.QueryRow(query, id).Scan(
		&strat.ID, &strat.Name, &strat.Description, &strat.Code, &strat.Language,
		&parametersJSON, &maxInputs, &defaultInputNamesJSON, &strat.CreatedAt, &strat.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load strategy: %w", err)
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

func (p *PostgreSQLDatabase) LoadAllStrategies() ([]*strategy.Strategy, error) {
	query := `
		SELECT id, name, description, code, language, parameters, max_inputs, default_input_names, created_at, updated_at
		FROM strategies
		ORDER BY name
	`

	rows, err := p.db.Query(query)
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

		err := rows.Scan(
			&strat.ID, &strat.Name, &strat.Description, &strat.Code, &strat.Language,
			&parametersJSON, &maxInputs, &defaultInputNamesJSON, &strat.CreatedAt, &strat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy: %w", err)
		}

		if err := json.Unmarshal([]byte(parametersJSON), &strat.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters for strategy %s: %w", strat.ID, err)
		}

		// Handle max_inputs
		if maxInputs.Valid {
			strat.MaxInputs = int(maxInputs.Int64)
		}

		// Handle default_input_names
		if defaultInputNamesJSON.Valid && defaultInputNamesJSON.String != "" {
			if err := json.Unmarshal([]byte(defaultInputNamesJSON.String), &strat.DefaultInputNames); err != nil {
				return nil, fmt.Errorf("failed to unmarshal default_input_names for strategy %s: %w", strat.ID, err)
			}
		}

		strategies = append(strategies, &strat)
	}

	return strategies, rows.Err()
}

func (p *PostgreSQLDatabase) DeleteStrategy(id string) error {
	query := "DELETE FROM strategies WHERE id = $1"
	_, err := p.db.Exec(query, id)
	return err
}

// State
func (p *PostgreSQLDatabase) SaveState(key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	query := `
		INSERT INTO state (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) 
		DO UPDATE SET 
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`

	_, err = p.db.Exec(query, key, string(valueJSON), time.Now())
	return err
}

func (p *PostgreSQLDatabase) LoadState(key string) (interface{}, error) {
	query := "SELECT value FROM state WHERE key = $1"
	var valueJSON string

	err := p.db.QueryRow(query, key).Scan(&valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	var value interface{}
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return value, nil
}

func (p *PostgreSQLDatabase) DeleteState(key string) error {
	query := "DELETE FROM state WHERE key = $1"
	_, err := p.db.Exec(query, key)
	return err
}

// Execution logs
func (p *PostgreSQLDatabase) SaveExecutionLog(log ExecutionLog) error {
	inputValuesJSON, err := json.Marshal(log.InputValues)
	if err != nil {
		return fmt.Errorf("failed to marshal input values: %w", err)
	}

	outputValuesJSON, err := json.Marshal(log.OutputValues)
	if err != nil {
		return fmt.Errorf("failed to marshal output values: %w", err)
	}

	query := `
		INSERT INTO execution_log 
		(topic_name, strategy_id, trigger_topic, input_values, output_values, error_message, execution_time_ms, executed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = p.db.Exec(query, log.TopicName, log.StrategyID, log.TriggerTopic,
		string(inputValuesJSON), string(outputValuesJSON), log.ErrorMessage,
		log.ExecutionTimeMs, log.ExecutedAt)
	return err
}

func (p *PostgreSQLDatabase) LoadExecutionLogs(topicName string, limit int) ([]ExecutionLog, error) {
	query := `
		SELECT id, topic_name, strategy_id, trigger_topic, input_values, output_values,
		       error_message, execution_time_ms, executed_at
		FROM execution_log
		WHERE topic_name = $1
		ORDER BY executed_at DESC
		LIMIT $2
	`

	rows, err := p.db.Query(query, topicName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution logs: %w", err)
	}
	defer rows.Close()

	var logs []ExecutionLog
	for rows.Next() {
		var log ExecutionLog
		var inputValuesJSON, outputValuesJSON string

		err := rows.Scan(
			&log.ID, &log.TopicName, &log.StrategyID, &log.TriggerTopic,
			&inputValuesJSON, &outputValuesJSON, &log.ErrorMessage,
			&log.ExecutionTimeMs, &log.ExecutedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution log: %w", err)
		}

		if err := json.Unmarshal([]byte(inputValuesJSON), &log.InputValues); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input values: %w", err)
		}

		if err := json.Unmarshal([]byte(outputValuesJSON), &log.OutputValues); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output values: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}
