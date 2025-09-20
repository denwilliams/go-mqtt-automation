package topics

import (
	"fmt"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/config"
)

type SystemTopic struct {
	config     SystemTopicConfig
	manager    *Manager
	ticker     *time.Ticker
	stopChan   chan bool
	isRunning  bool
}

func NewSystemTopic(name string, config map[string]interface{}) *SystemTopic {
	st := &SystemTopic{
		config: SystemTopicConfig{
			BaseTopicConfig: BaseTopicConfig{
				Name:        name,
				Type:        TopicTypeSystem,
				CreatedAt:   time.Now(),
				LastUpdated: time.Time{},
				Config:      config,
			},
		},
		stopChan:  make(chan bool),
		isRunning: false,
	}

	// Extract interval or cron from config
	if interval, ok := config["interval"].(string); ok {
		st.config.Interval = interval
	}
	if cron, ok := config["cron"].(string); ok {
		st.config.Cron = cron
	}

	return st
}

func (st *SystemTopic) Name() string {
	return st.config.Name
}

func (st *SystemTopic) Type() TopicType {
	return TopicTypeSystem
}

func (st *SystemTopic) LastValue() interface{} {
	return st.config.LastValue
}

func (st *SystemTopic) LastUpdated() time.Time {
	return st.config.LastUpdated
}

func (st *SystemTopic) SetManager(manager *Manager) {
	st.manager = manager
}

func (st *SystemTopic) Emit(value interface{}) error {
	previousValue := st.config.LastValue
	st.config.LastValue = value
	st.config.LastUpdated = time.Now()

	if st.manager != nil {
		event := TopicEvent{
			TopicName:     st.config.Name,
			Value:         value,
			PreviousValue: previousValue,
			Timestamp:     st.config.LastUpdated,
			TriggerTopic:  st.config.Name,
		}

		if err := st.manager.NotifyTopicUpdate(event); err != nil {
			return fmt.Errorf("failed to notify topic update: %w", err)
		}

		// Save state to database
		if err := st.manager.SaveTopicState(st.config.Name, value); err != nil {
			return fmt.Errorf("failed to save topic state: %w", err)
		}
	}

	return nil
}

func (st *SystemTopic) Start() error {
	if st.isRunning {
		return nil
	}

	if st.config.Interval != "" {
		duration, err := time.ParseDuration(st.config.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval duration: %w", err)
		}

		st.ticker = time.NewTicker(duration)
		st.isRunning = true

		go st.runTicker()
	} else if st.config.Cron != "" {
		// TODO: Implement cron scheduling
		return fmt.Errorf("cron scheduling not yet implemented")
	}

	return nil
}

func (st *SystemTopic) Stop() {
	if !st.isRunning {
		return
	}

	close(st.stopChan)
	if st.ticker != nil {
		st.ticker.Stop()
		st.ticker = nil
	}
	st.isRunning = false
}

func (st *SystemTopic) IsRunning() bool {
	return st.isRunning
}

func (st *SystemTopic) runTicker() {
	for {
		select {
		case <-st.stopChan:
			return
		case t := <-st.ticker.C:
			value := map[string]interface{}{
				"timestamp": t.Unix(),
				"iso_time":  t.Format(time.RFC3339),
				"topic":     st.config.Name,
			}
			
			if err := st.Emit(value); err != nil {
				// Log error but continue running
				if st.manager != nil && st.manager.logger != nil {
					st.manager.logger.Printf("Error emitting system topic %s: %v", st.config.Name, err)
				}
			}
		}
	}
}

func (st *SystemTopic) GetConfig() SystemTopicConfig {
	return st.config
}

func (st *SystemTopic) UpdateConfig(config SystemTopicConfig) {
	wasRunning := st.isRunning
	
	if wasRunning {
		st.Stop()
	}
	
	st.config = config
	
	if wasRunning {
		_ = st.Start() // Ignore error on restart
	}
}

// CreateDefaultSystemTopics creates the standard system topics
func CreateDefaultSystemTopics(cfg config.SystemTopicsConfig) []*SystemTopic {
	var topics []*SystemTopic

	// Create ticker topics
	for _, interval := range cfg.TickerIntervals {
		name := fmt.Sprintf("system/ticker/%s", interval)
		config := map[string]interface{}{
			"interval":    interval,
			"description": fmt.Sprintf("%s ticker", interval),
		}
		topics = append(topics, NewSystemTopic(name, config))
	}

	// Create event topics
	eventTopics := []struct {
		name        string
		description string
	}{
		{"system/events/startup", "System startup event"},
		{"system/events/shutdown", "System shutdown event"},
		{"system/events/error", "System error event"},
		{"system/events/heartbeat", "System heartbeat"},
	}

	for _, et := range eventTopics {
		config := map[string]interface{}{
			"description": et.description,
		}
		topics = append(topics, NewSystemTopic(et.name, config))
	}

	return topics
}

// EmitSystemEvent is a helper to emit system events
func (st *SystemTopic) EmitSystemEvent(eventType string, data interface{}) error {
	event := map[string]interface{}{
		"event_type": eventType,
		"timestamp":  time.Now().Unix(),
		"iso_time":   time.Now().Format(time.RFC3339),
		"data":       data,
	}
	
	return st.Emit(event)
}