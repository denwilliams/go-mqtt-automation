package topics

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/denwilliams/go-mqtt-automation/pkg/config"
	"github.com/denwilliams/go-mqtt-automation/pkg/mqtt"
	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
)

type StrategyExecutor interface {
	ExecuteStrategy(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) ([]strategy.EmitEvent, error)
}

type StateManager interface {
	SaveTopicState(topicName string, value interface{}) error
	LoadTopicState(topicName string) (interface{}, error)
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
	defer m.mutex.Unlock()

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
	defer m.mutex.Unlock()

	if _, exists := m.topics[name]; exists {
		return nil, fmt.Errorf("topic %s already exists", name)
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
	defer m.mutex.Unlock()

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
	defer m.mutex.Unlock()

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
	defer m.mutex.RUnlock()

	return m.topics[name]
}

func (m *Manager) GetExternalTopic(name string) *ExternalTopic {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.externalTopics[name]
}

func (m *Manager) GetInternalTopic(name string) *InternalTopic {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.internalTopics[name]
}

func (m *Manager) GetSystemTopic(name string) *SystemTopic {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.systemTopics[name]
}

func (m *Manager) ListTopics() map[string]Topic {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]Topic)
	for name, topic := range m.topics {
		result[name] = topic
	}
	return result
}

func (m *Manager) ListTopicsByType(topicType TopicType) map[string]Topic {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]Topic)
	for name, topic := range m.topics {
		if topic.Type() == topicType {
			result[name] = topic
		}
	}
	return result
}

func (m *Manager) NotifyTopicUpdate(event TopicEvent) error {
	// Only log MQTT inputs (external topics) and internal topic outputs
	// Skip noisy system topics like tickers, schedulers, etc.
	if m.shouldLogTopicUpdate(event.TopicName) {
		m.logger.Printf("Topic update: %s = %v", event.TopicName, event.Value)
	}

	// Find all internal topics that depend on this topic
	m.mutex.RLock()
	dependents := make([]*InternalTopic, 0)
	for _, internalTopic := range m.internalTopics {
		for _, input := range internalTopic.GetInputs() {
			if input == event.TopicName {
				dependents = append(dependents, internalTopic)
				break
			}
		}
	}
	m.mutex.RUnlock()

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
	defer m.mutex.RUnlock()

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
	defer m.mutex.RUnlock()

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
	defer m.mutex.RUnlock()

	counts := map[TopicType]int{
		TopicTypeExternal: len(m.externalTopics),
		TopicTypeInternal: len(m.internalTopics),
		TopicTypeSystem:   len(m.systemTopics),
	}

	return counts
}

// shouldLogTopicUpdate determines if a topic update should be logged
func (m *Manager) shouldLogTopicUpdate(topicName string) bool {
	// Always log internal topics (these are the automation logic outputs)
	m.mutex.RLock()
	defer m.mutex.RUnlock()

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
			return topicName != "system/events/heartbeat"
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

// createOrUpdateExternalTopic creates or updates an external topic with the given value
func (m *Manager) createOrUpdateExternalTopic(topicName string, value interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if topic already exists
	if existingTopic, exists := m.externalTopics[topicName]; exists {
		// Update existing external topic
		return existingTopic.Emit(value)
	}

	// Create new external topic
	newTopic := NewExternalTopic(topicName)
	newTopic.SetManager(m)
	m.externalTopics[topicName] = newTopic

	// Emit the initial value
	if err := newTopic.Emit(value); err != nil {
		return fmt.Errorf("failed to emit to new external topic %s: %w", topicName, err)
	}

	m.logger.Printf("Created new external topic: %s", topicName)
	return nil
}
