package web

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/config"
	"github.com/denwilliams/go-mqtt-automation/pkg/mqtt"
	"github.com/denwilliams/go-mqtt-automation/pkg/state"
	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
)

type Server struct {
	config         *config.Config
	topicManager   *topics.Manager
	strategyEngine *strategy.Engine
	stateManager   *state.Manager
	mqttClient     *mqtt.Client
	logger         *log.Logger
	server         *http.Server
	startTime      time.Time
}

func NewServer(cfg *config.Config, topicManager *topics.Manager, strategyEngine *strategy.Engine,
	stateManager *state.Manager, mqttClient *mqtt.Client, logger *log.Logger) (*Server, error) {

	if logger == nil {
		logger = log.Default()
	}

	server := &Server{
		config:         cfg,
		topicManager:   topicManager,
		strategyEngine: strategyEngine,
		stateManager:   stateManager,
		mqttClient:     mqttClient,
		logger:         logger,
		startTime:      time.Now(),
	}

	return server, nil
}

func (s *Server) Start() error {
	s.setupRoutes()

	address := s.config.GetAddress()
	s.logger.Printf("Starting web server on %s", address)

	s.server = &http.Server{
		Addr:    address,
		Handler: nil, // Uses DefaultServeMux
	}

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		s.logger.Println("Shutting down web server...")
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) setupRoutes() {
	// API v1 endpoints
	http.HandleFunc("/api/v1/dashboard", s.handleAPIDashboard)

	// Topics API
	http.HandleFunc("/api/v1/topics", s.handleAPIV1Topics)
	http.HandleFunc("/api/v1/topics/", s.handleAPITopicDetail)

	// Strategies API
	http.HandleFunc("/api/v1/strategies", s.handleAPIV1Strategies)
	http.HandleFunc("/api/v1/strategies/", s.handleAPIStrategyDetail)

	// System API
	http.HandleFunc("/api/v1/system", s.handleAPISystem)
	http.HandleFunc("/api/v1/system/info", s.handleAPISystemInfo)
	http.HandleFunc("/api/v1/system/stats", s.handleAPISystemStats)
	http.HandleFunc("/api/v1/system/activity", s.handleAPISystemActivity)

	// Legacy API endpoints for backwards compatibility
	http.HandleFunc("/api/topics", s.handleAPITopics)
	http.HandleFunc("/api/strategies", s.handleAPIStrategies)
	http.HandleFunc("/api/system/status", s.handleAPISystemStatus)

	// Static file serving for admin UI (catch-all handler)
	http.HandleFunc("/", s.handleStaticFiles)

	s.logger.Println("Web server routes configured")
}

func (s *Server) getSystemStatus() string {
	if s.mqttClient == nil {
		return "MQTT Client Not Configured"
	}

	switch s.mqttClient.GetState() {
	case mqtt.ConnectionStateConnected:
		return "Connected"
	case mqtt.ConnectionStateConnecting:
		return "Connecting"
	case mqtt.ConnectionStateReconnecting:
		return "Reconnecting"
	default:
		return "Disconnected"
	}
}
