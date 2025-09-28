package topics

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/config"
	"github.com/denwilliams/go-mqtt-automation/pkg/mqtt"
	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
)

type StrategyExecutor interface {
	ExecuteStrategy(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) ([]strategy.EmitEvent, error)
	GetStrategy(strategyID string) (*strategy.Strategy, error)
}

type StateManager interface {
	SaveTopicState(topicName string, value interface{}) error
	LoadTopicState(topicName string) (interface{}, error)
	LoadTopicConfig(topicName string) (interface{}, error)
}

type Manager struct {
	topics           map[string]Topic
	externalTopics   map[string]*ExternalTopic
	internalTopics   map[string]*InternalTopic
	systemTopics     map[string]*SystemTopic
	strategyExecutor StrategyExecutor
	stateManager     StateManager
	mqttClient       *mqtt.Client
	logger           *log.Logger
	mutex            sync.RWMutex
}

func NewManager(logger *log.Logger) *Manager {
	if logger == nil {
		logger = log.Default()
	}

	return &Manager{
		topics:         make(map[string]Topic),
		externalTopics: make(map[string]*ExternalTopic),
		internalTopics: make(map[string]*InternalTopic),
		systemTopics:   make(map[string]*SystemTopic),
		logger:         logger,
	}
}

func (m *Manager) SetStrategyExecutor(executor StrategyExecutor) {
	m.strategyExecutor = executor
}

func (m *Manager) SetStateManager(stateManager StateManager) {
	m.stateManager = stateManager
}

func (m *Manager) SetMQTTClient(client *mqtt.Client) {
	m.mqttClient = client
}

func (m *Manager) AddExternalTopic(name string) *ExternalTopic {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
	}()

	if topic, exists := m.externalTopics[name]; exists {
		return topic
	}

	topic := NewExternalTopic(name)
	topic.SetManager(m)

	m.externalTopics[name] = topic
	m.topics[name] = topic

	m.logger.Printf("Added external topic: %s", name)
	return topic
}

func (m *Manager) AddInternalTopic(name string, inputs []string, strategyID string) (*InternalTopic, error) {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
	}()

	if _, exists := m.topics[name]; exists {
		return nil, fmt.Errorf("topic %s already exists", name)
	}

	// Validate max inputs if strategy executor is available
	if m.strategyExecutor != nil {
		if strategy, err := m.strategyExecutor.GetStrategy(strategyID); err == nil {
			// Only validate if MaxInputs is set (non-zero), 0 or NULL means unlimited
			if strategy.MaxInputs > 0 && len(inputs) > strategy.MaxInputs {
				return nil, fmt.Errorf("strategy %s allows maximum %d inputs, but %d inputs provided", strategyID, strategy.MaxInputs, len(inputs))
			}
		}
	}

	topic := NewInternalTopic(name, inputs, strategyID)
	topic.SetManager(m)

	m.internalTopics[name] = topic
	m.topics[name] = topic

	m.logger.Printf("Added internal topic: %s with inputs: %v", name, inputs)
	return topic, nil
}

func (m *Manager) AddSystemTopic(name string, config map[string]interface{}) *SystemTopic {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
	}()

	if topic, exists := m.systemTopics[name]; exists {
		return topic
	}

	topic := NewSystemTopic(name, config)
	topic.SetManager(m)

	m.systemTopics[name] = topic
	m.topics[name] = topic

	m.logger.Printf("Added system topic: %s", name)
	return topic
}

func (m *Manager) RemoveTopic(name string) error {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
	}()

	topic, exists := m.topics[name]
	if !exists {
		return fmt.Errorf("topic %s not found", name)
	}

	// Stop system topics if running
	if systemTopic, ok := topic.(*SystemTopic); ok {
		systemTopic.Stop()
		delete(m.systemTopics, name)
	} else if _, ok := topic.(*ExternalTopic); ok {
		delete(m.externalTopics, name)
	} else if _, ok := topic.(*InternalTopic); ok {
		delete(m.internalTopics, name)
	}

	delete(m.topics, name)
	m.logger.Printf("Removed topic: %s", name)
	return nil
}

func (m *Manager) GetTopic(name string) Topic {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	return m.topics[name]
}

func (m *Manager) GetExternalTopic(name string) *ExternalTopic {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	return m.externalTopics[name]
}

func (m *Manager) GetInternalTopic(name string) *InternalTopic {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	return m.internalTopics[name]
}

func (m *Manager) GetSystemTopic(name string) *SystemTopic {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	return m.systemTopics[name]
}

func (m *Manager) ListTopics() map[string]Topic {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	result := make(map[string]Topic)
	for name, topic := range m.topics {
		result[name] = topic
	}
	return result
}

func (m *Manager) ListTopicsByType(topicType TopicType) map[string]Topic {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	result := make(map[string]Topic)
	for name, topic := range m.topics {
		if topic.Type() == topicType {
			result[name] = topic
		}
	}
	return result
}

func (m *Manager) NotifyTopicUpdate(event TopicEvent) error {
	// Find all internal topics that depend on this topic (including wildcard matches)
	m.mutex.RLock()

	// Only log MQTT inputs (external topics) and internal topic outputs
	// Skip noisy system topics like tickers, schedulers, etc.
	// Note: We call shouldLogTopicUpdate while holding the lock to avoid double-locking
	shouldLog := m.shouldLogTopicUpdateUnsafe(event.TopicName)

	dependents := make([]*InternalTopic, 0)
	for _, internalTopic := range m.internalTopics {
		for _, input := range internalTopic.GetInputs() {
			// Check for exact match or wildcard match
			if input == event.TopicName || mqtt.TopicMatches(input, event.TopicName) {
				dependents = append(dependents, internalTopic)
				break
			}
		}
	}
	m.mutex.RUnlock()

	// Log the topic update if needed (do this after releasing the lock)
	if shouldLog {
		m.logger.Printf("Topic update: %s = %v", event.TopicName, event.Value)
	}

	// Process dependent topics
	for _, dependent := range dependents {
		if err := dependent.ProcessInputs(event.TopicName); err != nil {
			m.logger.Printf("Error processing inputs for topic %s: %v", dependent.Name(), err)
		}
	}

	return nil
}

func (m *Manager) ExecuteStrategy(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) ([]strategy.EmitEvent, error) {
	if m.strategyExecutor == nil {
		return nil, fmt.Errorf("strategy executor not configured")
	}

	return m.strategyExecutor.ExecuteStrategy(strategyID, inputs, triggerTopic, lastOutput)
}

func (m *Manager) SaveTopicState(topicName string, value interface{}) error {
	if m.stateManager == nil {
		return nil // No state manager configured
	}

	return m.stateManager.SaveTopicState(topicName, value)
}

func (m *Manager) LoadTopicState(topicName string) (interface{}, error) {
	if m.stateManager == nil {
		return nil, fmt.Errorf("state manager not configured")
	}

	return m.stateManager.LoadTopicState(topicName)
}

func (m *Manager) InitializeSystemTopics(cfg config.SystemTopicsConfig) error {
	systemTopics := CreateDefaultSystemTopics(cfg)

	for _, topic := range systemTopics {
		m.AddSystemTopic(topic.Name(), topic.config.Config)

		// Start ticker topics
		if topic.config.Interval != "" {
			if err := topic.Start(); err != nil {
				m.logger.Printf("Failed to start system topic %s: %v", topic.Name(), err)
			}
		}
	}

	return nil
}

func (m *Manager) StartSystemTopics() error {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	for _, topic := range m.systemTopics {
		if topic.config.Interval != "" && !topic.IsRunning() {
			if err := topic.Start(); err != nil {
				m.logger.Printf("Failed to start system topic %s: %v", topic.Name(), err)
			}
		}
	}

	return nil
}

func (m *Manager) StopSystemTopics() {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	for _, topic := range m.systemTopics {
		if topic.IsRunning() {
			topic.Stop()
		}
	}
}

func (m *Manager) HandleMQTTMessage(event mqtt.Event) error {
	// Find or create external topic
	topic := m.GetExternalTopic(event.Topic)
	if topic == nil {
		topic = m.AddExternalTopic(event.Topic)
	}

	// Update topic with MQTT payload
	return topic.UpdateFromMQTT(event.Payload)
}

func (m *Manager) GetTopicCount() map[TopicType]int {
	m.mutex.RLock()
	defer func() {
		m.mutex.RUnlock()
	}()

	counts := map[TopicType]int{
		TopicTypeExternal: len(m.externalTopics),
		TopicTypeInternal: len(m.internalTopics),
		TopicTypeSystem:   len(m.systemTopics),
	}

	return counts
}

// shouldLogTopicUpdateUnsafe determines if a topic update should be logged (assumes lock is already held)
func (m *Manager) shouldLogTopicUpdateUnsafe(topicName string) bool {
	if _, exists := m.internalTopics[topicName]; exists {
		return true
	}

	// Always log external topics (these are MQTT inputs from sensors, devices, etc.)
	if _, exists := m.externalTopics[topicName]; exists {
		return true
	}

	// For system topics, only log important events, skip noisy ones
	if _, exists := m.systemTopics[topicName]; exists {
		// Log important system events, but skip heartbeat
		if strings.HasPrefix(topicName, "system/events/") {
			result := topicName != "system/events/heartbeat"
			return result
		}

		// Skip noisy system topics like tickers
		if strings.HasPrefix(topicName, "system/ticker/") {
			return false
		}

		if strings.HasPrefix(topicName, "system/scheduler/") {
			return false
		}

		// Log other system topics by default (for now)
		return true
	}

	// Default to not logging unknown topics
	return false
}

// createOrUpdateDerivedTopic creates or updates a derived internal topic (from strategy emissions)
func (m *Manager) createOrUpdateDerivedTopic(topicName string, value interface{}) error {
	m.mutex.Lock()

	// Check if topic already exists as an internal topic
	if existingTopic, exists := m.internalTopics[topicName]; exists {
		// Update existing derived internal topic directly
		previousValue := existingTopic.config.LastValue
		existingTopic.config.LastValue = value
		existingTopic.config.LastUpdated = time.Now()

		// Save state to database
		if err := m.SaveTopicState(topicName, value); err != nil {
			m.mutex.Unlock()
			return fmt.Errorf("failed to save topic state: %w", err)
		}

		// Release lock before triggering other topics to prevent deadlock
		m.mutex.Unlock()

		// Notify other topics that depend on this derived topic
		event := TopicEvent{
			TopicName:     topicName,
			Value:         value,
			PreviousValue: previousValue,
			Timestamp:     time.Now(),
			TriggerTopic:  topicName,
		}

		if err := m.NotifyTopicUpdate(event); err != nil {
			return fmt.Errorf("failed to notify topic update: %w", err)
		}

		return nil
	}

	// Continue with topic creation (lock is still held)
	// Note: We manually unlock before NotifyTopicUpdate to prevent deadlock

	// Check if topic exists in the main topics map
	if _, exists := m.topics[topicName]; exists {
		// Topic exists but not as internal - this shouldn't happen for derived topics
		return fmt.Errorf("topic %s already exists as a different type", topicName)
	}

	// Create new derived internal topic (read-only, no strategy)
	newTopic := &InternalTopic{
		config: InternalTopicConfig{
			BaseTopicConfig: BaseTopicConfig{
				Name:        topicName,
				Type:        TopicTypeInternal,
				LastValue:   value,
				LastUpdated: time.Now(),
				CreatedAt:   time.Now(),
			},
			Inputs:        []string{}, // Derived topics have no inputs
			StrategyID:    "",         // Derived topics have no strategy
			EmitToMQTT:    false,      // Default to not emitting derived topics to MQTT
			NoOpUnchanged: false,
		},
		manager: m,
	}

	m.internalTopics[topicName] = newTopic
	m.topics[topicName] = newTopic

	// Save state to database
	if err := m.SaveTopicState(topicName, value); err != nil {
		return fmt.Errorf("failed to save topic state for %s: %w", topicName, err)
	}

	m.logger.Printf("Created new derived internal topic: %s", topicName)

	// Release lock before triggering other topics to prevent deadlock
	m.mutex.Unlock()

	// Notify other topics that might depend on this new derived topic
	event := TopicEvent{
		TopicName:     topicName,
		Value:         value,
		PreviousValue: nil,
		Timestamp:     time.Now(),
		TriggerTopic:  topicName,
	}

	if err := m.NotifyTopicUpdate(event); err != nil {
		return fmt.Errorf("failed to notify topic update: %w", err)
	}

	return nil
}

// ReloadTopicFromDatabase loads a topic configuration from the database and updates the in-memory version
func (m *Manager) ReloadTopicFromDatabase(topicName string) error {
	if m.stateManager == nil {
		return fmt.Errorf("state manager not configured")
	}

	// Load topic config from database
	configInterface, err := m.stateManager.LoadTopicConfig(topicName)
	if err != nil {
		return fmt.Errorf("failed to load topic config from database: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Handle different topic types
	switch cfg := configInterface.(type) {
	case InternalTopicConfig:
		// Update existing internal topic or create new one
		if existingTopic, exists := m.internalTopics[topicName]; exists {
			// Update existing topic
			existingTopic.UpdateConfig(cfg)
			m.logger.Printf("Reloaded internal topic from database: %s", topicName)
		} else {
			// Create new internal topic
			newTopic := &InternalTopic{
				config:  cfg,
				manager: m,
			}
			m.internalTopics[topicName] = newTopic
			m.topics[topicName] = newTopic
			m.logger.Printf("Created new internal topic from database: %s", topicName)
		}

	case BaseTopicConfig:
		// Handle external topics (they use BaseTopicConfig)
		if cfg.Type == TopicTypeExternal {
			if existingTopic, exists := m.externalTopics[topicName]; exists {
				// Update existing topic
				existingTopic.UpdateConfig(cfg)
				m.logger.Printf("Reloaded external topic from database: %s", topicName)
			} else {
				// Create new external topic
				newTopic := NewExternalTopic(topicName)
				newTopic.SetManager(m)
				newTopic.UpdateConfig(cfg)
				m.externalTopics[topicName] = newTopic
				m.topics[topicName] = newTopic
				m.logger.Printf("Created new external topic from database: %s", topicName)
			}
		}

	case SystemTopicConfig:
		// Update existing system topic or create new one
		if existingTopic, exists := m.systemTopics[topicName]; exists {
			// Stop existing topic before updating
			if existingTopic.IsRunning() {
				existingTopic.Stop()
			}
			// Update existing topic
			existingTopic.UpdateConfig(cfg)
			// Restart if it has an interval
			if cfg.Interval != "" {
				if startErr := existingTopic.Start(); startErr != nil {
					m.logger.Printf("Failed to restart system topic %s: %v", topicName, startErr)
				}
			}
			m.logger.Printf("Reloaded system topic from database: %s", topicName)
		} else {
			// Create new system topic
			newTopic := NewSystemTopic(topicName, cfg.Config)
			newTopic.SetManager(m)
			m.systemTopics[topicName] = newTopic
			m.topics[topicName] = newTopic
			// Start if it has an interval
			if cfg.Interval != "" {
				if startErr := newTopic.Start(); startErr != nil {
					m.logger.Printf("Failed to start new system topic %s: %v", topicName, startErr)
				}
			}
			m.logger.Printf("Created new system topic from database: %s", topicName)
		}

	default:
		return fmt.Errorf("unknown topic config type: %T", configInterface)
	}

	return nil
}
