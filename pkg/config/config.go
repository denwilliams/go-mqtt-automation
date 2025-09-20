package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MQTT         MQTTConfig         `yaml:"mqtt"`
	Database     DatabaseConfig     `yaml:"database"`
	Web          WebConfig          `yaml:"web"`
	Logging      LoggingConfig      `yaml:"logging"`
	SystemTopics SystemTopicsConfig `yaml:"system_topics"`
}

type MQTTConfig struct {
	Broker   string   `yaml:"broker"`
	ClientID string   `yaml:"client_id"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Topics   []string `yaml:"topics"`
}

type DatabaseConfig struct {
	Type       string `yaml:"type"`
	Connection string `yaml:"connection"`
}

type WebConfig struct {
	Port int    `yaml:"port"`
	Bind string `yaml:"bind"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type SystemTopicsConfig struct {
	TickerIntervals []string `yaml:"ticker_intervals"`
}

func Load(configPath string) (*Config, error) {
	// Set default config path if not provided
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	config.setDefaults()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func (c *Config) setDefaults() {
	// MQTT defaults
	if c.MQTT.ClientID == "" {
		c.MQTT.ClientID = "home-automation"
	}
	if c.MQTT.Broker == "" {
		c.MQTT.Broker = "mqtt://localhost:1883"
	}

	// Database defaults
	if c.Database.Type == "" {
		c.Database.Type = "sqlite"
	}
	if c.Database.Connection == "" {
		// Use test database if running in test mode
		if isTestMode() {
			c.Database.Connection = "./test.db"
		} else {
			c.Database.Connection = "./automation.db"
		}
	}

	// Web defaults
	if c.Web.Port == 0 {
		c.Web.Port = 8080
	}
	if c.Web.Bind == "" {
		c.Web.Bind = "0.0.0.0"
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.File == "" {
		c.Logging.File = "./automation.log"
	}

	// System topics defaults
	if len(c.SystemTopics.TickerIntervals) == 0 {
		c.SystemTopics.TickerIntervals = []string{"1s", "5s", "30s", "1m", "5m"}
	}
}

func (c *Config) validate() error {
	// Validate MQTT broker URL
	if c.MQTT.Broker == "" {
		return fmt.Errorf("MQTT broker URL is required")
	}

	// Validate database type
	if c.Database.Type != "sqlite" && c.Database.Type != "postgres" {
		return fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}

	// Validate web port
	if c.Web.Port < 1 || c.Web.Port > 65535 {
		return fmt.Errorf("invalid web port: %d", c.Web.Port)
	}

	// Validate logging level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", c.Logging.Level)
	}

	// Validate ticker intervals
	for _, interval := range c.SystemTopics.TickerIntervals {
		if _, err := time.ParseDuration(interval); err != nil {
			return fmt.Errorf("invalid ticker interval: %s", interval)
		}
	}

	return nil
}

func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Web.Bind, c.Web.Port)
}

// isTestMode detects if we're running in test mode
func isTestMode() bool {
	// Check if the executable name contains ".test" (indicates test binary)
	if exe, err := os.Executable(); err == nil {
		return strings.Contains(exe, ".test")
	}

	// Check if we're being called from 'go test'
	for _, arg := range os.Args {
		if strings.Contains(arg, "test") || strings.HasSuffix(arg, ".test") {
			return true
		}
	}

	// Check TEST environment variable
	return os.Getenv("TEST") == "1"
}
