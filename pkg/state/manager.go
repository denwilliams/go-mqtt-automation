package state

import (
	"fmt"
	"log"

	"github.com/denwilliams/go-mqtt-automation/pkg/config"
	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
)

type Manager struct {
	db     Database
	logger *log.Logger
}

func NewManager(cfg config.DatabaseConfig, logger *log.Logger) (*Manager, error) {
	if logger == nil {
		logger = log.Default()
	}

	var db Database
	var err error

	switch cfg.Type {
	case "sqlite":
		db, err = NewSQLiteDatabase(cfg.Connection)
	case "postgres", "postgresql":
		db, err = NewPostgreSQLDatabase(cfg.Connection)
	default:
		err = fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	manager := &Manager{
		db:     db,
		logger: logger,
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	logger.Printf("State manager initialized with %s database", cfg.Type)
	return manager, nil
}

func (m *Manager) Close() error {
	return m.db.Close()
}

// Topic State Management
func (m *Manager) SaveTopicState(topicName string, value interface{}) error {
	// Save to state table (for legacy compatibility)
	key := fmt.Sprintf("topic:%s", topicName)
	if err := m.db.SaveState(key, value); err != nil {
		m.logger.Printf("Failed to save topic state for %s: %v", topicName, err)
		return err
	}

	// Also update the last_value column in topics table
	if err := m.db.UpdateTopicLastValue(topicName, value); err != nil {
		m.logger.Printf("Failed to update topic last value for %s: %v", topicName, err)
		// Don't return error here - state table update succeeded
	}

	return nil
}

func (m *Manager) LoadTopicState(topicName string) (interface{}, error) {
	key := fmt.Sprintf("topic:%s", topicName)
	value, err := m.db.LoadState(key)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// Topic Configuration Management
func (m *Manager) SaveTopicConfig(config interface{}) error {
	if err := m.db.SaveTopic(config); err != nil {
		m.logger.Printf("Failed to save topic config: %v", err)
		return err
	}
	return nil
}

func (m *Manager) LoadTopicConfig(name string) (interface{}, error) {
	return m.db.LoadTopic(name)
}

func (m *Manager) LoadAllTopicConfigs() ([]interface{}, error) {
	return m.db.LoadAllTopics()
}

func (m *Manager) DeleteTopicConfig(name string) error {
	if err := m.db.DeleteTopic(name); err != nil {
		m.logger.Printf("Failed to delete topic config %s: %v", name, err)
		return err
	}
	return nil
}

// Strategy Management
func (m *Manager) SaveStrategy(strategy *strategy.Strategy) error {
	if err := m.db.SaveStrategy(strategy); err != nil {
		m.logger.Printf("Failed to save strategy %s: %v", strategy.ID, err)
		return err
	}
	m.logger.Printf("Saved strategy: %s (%s)", strategy.Name, strategy.ID)
	return nil
}

func (m *Manager) LoadStrategy(id string) (*strategy.Strategy, error) {
	return m.db.LoadStrategy(id)
}

func (m *Manager) LoadAllStrategies() ([]*strategy.Strategy, error) {
	return m.db.LoadAllStrategies()
}

func (m *Manager) DeleteStrategy(id string) error {
	if err := m.db.DeleteStrategy(id); err != nil {
		m.logger.Printf("Failed to delete strategy %s: %v", id, err)
		return err
	}
	m.logger.Printf("Deleted strategy: %s", id)
	return nil
}

// General State Management
func (m *Manager) SaveState(key string, value interface{}) error {
	if err := m.db.SaveState(key, value); err != nil {
		m.logger.Printf("Failed to save state %s: %v", key, err)
		return err
	}
	return nil
}

func (m *Manager) LoadState(key string) (interface{}, error) {
	return m.db.LoadState(key)
}

func (m *Manager) DeleteState(key string) error {
	if err := m.db.DeleteState(key); err != nil {
		m.logger.Printf("Failed to delete state %s: %v", key, err)
		return err
	}
	return nil
}

// Execution Log Management
func (m *Manager) SaveExecutionLog(log ExecutionLog) error {
	if err := m.db.SaveExecutionLog(log); err != nil {
		m.logger.Printf("Failed to save execution log: %v", err)
		return err
	}
	return nil
}

func (m *Manager) LoadExecutionLogs(topicName string, limit int) ([]ExecutionLog, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}
	return m.db.LoadExecutionLogs(topicName, limit)
}

// System Recovery
func (m *Manager) RestoreTopicStates() (map[string]interface{}, error) {
	m.logger.Println("Restoring topic states from database...")

	// This is a simplified approach - in a real system you might want to
	// implement a more sophisticated state recovery mechanism
	states := make(map[string]interface{})

	// For now, we'll just return an empty map as individual topics
	// will load their states as needed

	m.logger.Println("Topic states restored")
	return states, nil
}

// Database maintenance
func (m *Manager) CleanupOldLogs(days int) error {
	// This would implement cleanup of old execution logs
	// For now, just log the action
	m.logger.Printf("Would cleanup execution logs older than %d days", days)
	return nil
}

func (m *Manager) GetDatabaseStats() map[string]interface{} {
	// This would return database statistics
	// For now, return basic info
	return map[string]interface{}{
		"type": "sqlite",
		"path": m.db.(*SQLiteDatabase).path,
	}
}
