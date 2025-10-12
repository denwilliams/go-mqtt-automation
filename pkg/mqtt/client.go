// Package mqtt provides MQTT client functionality for the home automation system.
package mqtt

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/config"
	"github.com/denwilliams/go-mqtt-automation/pkg/metrics"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	config         config.MQTTConfig
	client         mqtt.Client
	handlers       map[string]EventHandler
	state          ConnectionState
	stateMutex     sync.RWMutex
	logger         *log.Logger
	stopChan       chan bool
	reconnectDelay time.Duration
	topicManager   TopicManager
}

type TopicManager interface {
	HandleMQTTMessage(event Event) error
}

func NewClient(cfg config.MQTTConfig, logger *log.Logger) *Client {
	if logger == nil {
		logger = log.Default()
	}

	return &Client{
		config:         cfg,
		handlers:       make(map[string]EventHandler),
		state:          ConnectionStateClosed,
		logger:         logger,
		stopChan:       make(chan bool),
		reconnectDelay: 5 * time.Second,
	}
}

func (c *Client) SetTopicManager(manager TopicManager) {
	c.topicManager = manager
}

func (c *Client) Connect() error {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	if c.state == ConnectionStateConnected {
		return nil
	}

	c.state = ConnectionStateConnecting
	c.logger.Printf("Connecting to MQTT broker: %s", c.config.Broker)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.config.Broker)
	opts.SetClientID(c.config.ClientID)

	if c.config.Username != "" {
		opts.SetUsername(c.config.Username)
	}
	if c.config.Password != "" {
		opts.SetPassword(c.config.Password)
	}

	opts.SetAutoReconnect(false) // We handle reconnection manually
	opts.SetCleanSession(true)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	opts.SetConnectionLostHandler(c.onConnectionLost)
	opts.SetOnConnectHandler(c.onConnect)

	opts.SetDefaultPublishHandler(c.onMessage)

	c.client = mqtt.NewClient(opts)

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		c.state = ConnectionStateClosed
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	c.state = ConnectionStateConnected
	c.logger.Println("Successfully connected to MQTT broker")

	// Update connection metrics
	metrics.SetMQTTConnectionState(c.config.Broker, true)

	// Subscribe to configured topics (async to prevent blocking)
	go func() {
		for _, topic := range c.config.Topics {
			if err := c.Subscribe(topic, c.handleTopicMessage); err != nil {
				c.logger.Printf("Failed to subscribe to topic %s: %v", topic, err)
			}
		}
	}()

	return nil
}

func (c *Client) Disconnect() {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	if c.state == ConnectionStateClosed {
		return
	}

	c.logger.Println("Disconnecting from MQTT broker")

	close(c.stopChan)

	if c.client != nil {
		c.client.Disconnect(250)
	}

	c.state = ConnectionStateClosed

	// Update connection metrics
	metrics.SetMQTTConnectionState(c.config.Broker, false)

	c.logger.Println("Disconnected from MQTT broker")
}

func (c *Client) Subscribe(topic string, handler EventHandler) error {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()

	if c.state != ConnectionStateConnected {
		return fmt.Errorf("not connected to MQTT broker")
	}

	c.handlers[topic] = handler

	token := c.client.Subscribe(topic, 0, nil)
	token.Wait()

	if token.Error() != nil {
		delete(c.handlers, topic)
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
	}

	c.logger.Printf("Subscribed to topic: %s", topic)
	return nil
}

func (c *Client) Unsubscribe(topic string) error {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()

	if c.state != ConnectionStateConnected {
		return fmt.Errorf("not connected to MQTT broker")
	}

	delete(c.handlers, topic)

	token := c.client.Unsubscribe(topic)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from topic %s: %w", topic, token.Error())
	}

	c.logger.Printf("Unsubscribed from topic: %s", topic)
	return nil
}

func (c *Client) Publish(topic string, payload []byte, retain bool) error {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()

	if c.state != ConnectionStateConnected {
		return fmt.Errorf("not connected to MQTT broker")
	}

	token := c.client.Publish(topic, 0, retain, payload)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topic, token.Error())
	}

	c.logger.Printf("Published to topic: %s (%d bytes)", topic, len(payload))
	return nil
}

func (c *Client) IsConnected() bool {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state == ConnectionStateConnected
}

func (c *Client) GetState() ConnectionState {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state
}

func (c *Client) onConnect(client mqtt.Client) {
	c.logger.Println("MQTT client connected")
}

func (c *Client) onConnectionLost(client mqtt.Client, err error) {
	c.stateMutex.Lock()
	c.state = ConnectionStateReconnecting
	c.stateMutex.Unlock()

	c.logger.Printf("Connection lost: %v", err)
	c.logger.Println("Starting reconnection attempts...")

	go c.reconnect()
}

func (c *Client) reconnect() {
	for {
		select {
		case <-c.stopChan:
			return
		case <-time.After(c.reconnectDelay):
			c.logger.Println("Attempting to reconnect...")

			if err := c.Connect(); err != nil {
				c.logger.Printf("Reconnection failed: %v", err)
				// Exponential backoff with max delay
				c.reconnectDelay *= 2
				if c.reconnectDelay > time.Minute*5 {
					c.reconnectDelay = time.Minute * 5
				}
			} else {
				c.logger.Println("Successfully reconnected")
				c.reconnectDelay = 5 * time.Second // Reset delay
				return
			}
		}
	}
}

func (c *Client) onMessage(client mqtt.Client, msg mqtt.Message) {
	event := Event{
		Topic:     msg.Topic(),
		Payload:   msg.Payload(),
		Timestamp: time.Now(),
	}

	// Find matching handler
	for pattern, handler := range c.handlers {
		if c.topicMatches(pattern, msg.Topic()) {
			if err := handler(event); err != nil {
				c.logger.Printf("Error handling message for topic %s: %v", msg.Topic(), err)
			}
			break
		}
	}
}

func (c *Client) handleTopicMessage(event Event) error {
	// Record MQTT message received
	metrics.RecordMQTTReceive(event.Topic)

	c.logger.Printf("Received message on topic %s: %s", event.Topic, string(event.Payload))

	// Notify topic manager if available
	if c.topicManager != nil {
		if err := c.topicManager.HandleMQTTMessage(event); err != nil {
			c.logger.Printf("Error handling MQTT message for topic %s: %v", event.Topic, err)
			return err
		}
	}

	return nil
}

// topicMatches checks if a topic matches a pattern with MQTT wildcards
// Supports:
// + (single-level wildcard): matches exactly one level
// # (multi-level wildcard): matches zero or more levels (only at end)
func (c *Client) topicMatches(pattern, topic string) bool {
	return TopicMatches(pattern, topic)
}

// TopicMatches checks if a topic matches a pattern with MQTT wildcards
// This is a standalone function that can be used throughout the codebase
func TopicMatches(pattern, topic string) bool {
	// Exact match
	if pattern == topic {
		return true
	}

	// Split into segments
	patternSegments := strings.Split(pattern, "/")
	topicSegments := strings.Split(topic, "/")

	return matchSegments(patternSegments, topicSegments)
}

func matchSegments(patternSegments, topicSegments []string) bool {
	patternLen := len(patternSegments)
	topicLen := len(topicSegments)

	// Handle empty patterns or topics
	if patternLen == 0 || topicLen == 0 {
		return patternLen == 0 && topicLen == 0
	}

	// Handle multi-level wildcard (#) - must be last segment
	if patternLen > 0 && patternSegments[patternLen-1] == "#" {
		// # matches zero or more levels
		if patternLen == 1 {
			return true // # matches everything
		}
		// Match all segments before the #
		if topicLen < patternLen-1 {
			return false
		}
		for i := 0; i < patternLen-1; i++ {
			if patternSegments[i] != "+" && patternSegments[i] != topicSegments[i] {
				return false
			}
		}
		return true
	}

	// For patterns without #, lengths must match
	if patternLen != topicLen {
		return false
	}

	// Match each segment
	for i := 0; i < patternLen; i++ {
		if patternSegments[i] != "+" && patternSegments[i] != topicSegments[i] {
			return false
		}
	}

	return true
}
