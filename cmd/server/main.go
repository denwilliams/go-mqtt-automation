package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/denwilliams/go-mqtt-automation/pkg/config"
	"github.com/denwilliams/go-mqtt-automation/pkg/mqtt"
	"github.com/denwilliams/go-mqtt-automation/pkg/state"
	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
	"github.com/denwilliams/go-mqtt-automation/pkg/web"
)

var (
	configPath  = flag.String("config", "config/config.yaml", "Path to configuration file")
	migrate     = flag.Bool("migrate", false, "Run database migrations and exit")
	showVersion = flag.Bool("version", false, "Show version and exit")

	// Build-time variables
	version   = "dev"
	buildDate = "unknown"
)

const (
	appName = "MQTT Home Automation"
)

type Application struct {
	config         *config.Config
	logger         *log.Logger
	stateManager   *state.Manager
	strategyEngine *strategy.Engine
	topicManager   *topics.Manager
	mqttClient     *mqtt.Client
	webServer      *web.Server
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func main() {
	flag.Parse()

	if *showVersion {
		log.Printf("%s version %s (built %s)", appName, version, buildDate)
		return
	}

	app, err := NewApplication(*configPath)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer app.Cleanup()

	if *migrate {
		log.Println("Running database migrations...")
		if err := app.stateManager.Close(); err != nil {
			log.Printf("Error closing state manager: %v", err)
		}
		log.Println("Migrations completed successfully")
		return
	}

	// Handle graceful shutdown
	app.setupSignalHandling()

	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for shutdown
	app.Wait()
	log.Println("Application shutdown complete")
}

func NewApplication(configPath string) (*Application, error) {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	// Setup logger
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	logger.Printf("Starting %s %s", appName, version)
	logger.Printf("Loaded configuration from: %s", configPath)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize components
	if err := app.initializeComponents(); err != nil {
		cancel()
		return nil, err
	}

	return app, nil
}

func (a *Application) initializeComponents() error {
	var err error

	// Initialize state manager
	a.logger.Println("Initializing state manager...")
	a.stateManager, err = state.NewManager(a.config.Database, a.logger)
	if err != nil {
		return err
	}

	// Initialize strategy engine
	a.logger.Println("Initializing strategy engine...")
	a.strategyEngine = strategy.NewEngine(a.logger)

	// Load strategies from database
	if loadErr := a.loadStrategies(); loadErr != nil {
		a.logger.Printf("Warning: Failed to load strategies: %v", loadErr)
	}

	// Initialize topic manager
	a.logger.Println("Initializing topic manager...")
	a.topicManager = topics.NewManager(a.logger)
	a.topicManager.SetStrategyExecutor(a.strategyEngine)
	a.topicManager.SetStateManager(a.stateManager)

	// Initialize MQTT client
	a.logger.Println("Initializing MQTT client...")
	a.mqttClient = mqtt.NewClient(a.config.MQTT, a.logger)
	a.topicManager.SetMQTTClient(a.mqttClient)
	a.mqttClient.SetTopicManager(a.topicManager)

	// Load topics from database
	if loadErr := a.loadTopics(); loadErr != nil {
		a.logger.Printf("Warning: Failed to load topics: %v", loadErr)
	}

	// Initialize system topics
	if initErr := a.topicManager.InitializeSystemTopics(a.config.SystemTopics); initErr != nil {
		return initErr
	}

	// Initialize web server
	a.logger.Println("Initializing web server...")
	a.webServer, err = web.NewServer(a.config, a.topicManager, a.strategyEngine, a.stateManager, a.mqttClient, a.logger)
	if err != nil {
		return err
	}

	a.logger.Println("All components initialized successfully")
	return nil
}

func (a *Application) loadStrategies() error {
	strategies, err := a.stateManager.LoadAllStrategies()
	if err != nil {
		return err
	}

	for _, strat := range strategies {
		if err := a.strategyEngine.AddStrategy(strat); err != nil {
			a.logger.Printf("Failed to load strategy %s: %v", strat.ID, err)
		} else {
			a.logger.Printf("Loaded strategy: %s", strat.Name)
		}
	}

	a.logger.Printf("Loaded %d strategies", len(strategies))
	return nil
}

func (a *Application) loadTopics() error {
	topicConfigs, err := a.stateManager.LoadAllTopicConfigs()
	if err != nil {
		return err
	}

	for _, config := range topicConfigs {
		switch cfg := config.(type) {
		case topics.InternalTopicConfig:
			_, err := a.topicManager.AddInternalTopic(cfg.Name, cfg.Inputs, cfg.InputNames, cfg.StrategyID, cfg.EmitToMQTT, cfg.NoOpUnchanged)
			if err != nil {
				a.logger.Printf("Failed to load internal topic %s: %v", cfg.Name, err)
			} else {
				a.logger.Printf("Loaded internal topic: %s", cfg.Name)
			}
		case topics.SystemTopicConfig:
			topic := a.topicManager.AddSystemTopic(cfg.Name, cfg.Config)
			if topic != nil {
				a.logger.Printf("Loaded system topic: %s", cfg.Name)
			}
		default:
			a.logger.Printf("Unknown topic config type for %v", config)
		}
	}

	a.logger.Printf("Loaded %d topic configurations", len(topicConfigs))
	return nil
}

func (a *Application) Start() error {
	a.logger.Println("Starting application components...")

	// Start MQTT client
	if err := a.mqttClient.Connect(); err != nil {
		a.logger.Printf("Failed to connect to MQTT broker: %v", err)
		// Don't fail startup if MQTT is not available
	} else {
		// Subscribe to MQTT messages
		a.wg.Add(1)
		go a.handleMQTTMessages()
	}

	// Start system topics
	if err := a.topicManager.StartSystemTopics(); err != nil {
		a.logger.Printf("Failed to start system topics: %v", err)
	}

	// Emit startup event
	a.emitSystemEvent("startup", map[string]interface{}{
		"version": version,
		"config":  a.config,
	})

	// Start web server
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.webServer.Start(); err != nil && err != http.ErrServerClosed {
			a.logger.Printf("Web server error: %v", err)
		}
	}()

	a.logger.Println("Application started successfully")
	return nil
}

func (a *Application) handleMQTTMessages() {
	defer a.wg.Done()

	// This would typically involve setting up MQTT message handlers
	// For now, we'll just log that the handler is running
	a.logger.Println("MQTT message handler started")

	// Wait for context cancellation
	<-a.ctx.Done()
	a.logger.Println("MQTT message handler stopped")
}

func (a *Application) emitSystemEvent(eventType string, data interface{}) {
	eventTopic := a.topicManager.GetSystemTopic("system/events/" + eventType)
	if eventTopic != nil {
		if err := eventTopic.EmitSystemEvent(eventType, data); err != nil {
			a.logger.Printf("Failed to emit system event %s: %v", eventType, err)
		}
	}
}

func (a *Application) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		a.logger.Printf("Received signal: %v", sig)
		a.logger.Println("Initiating graceful shutdown...")

		// Emit shutdown event
		a.emitSystemEvent("shutdown", map[string]interface{}{
			"signal": sig.String(),
		})

		a.cancel()
	}()
}

func (a *Application) Wait() {
	<-a.ctx.Done()
	a.logger.Println("Shutting down...")

	// Create shutdown timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop system topics
	a.topicManager.StopSystemTopics()

	// Shutdown web server
	if a.webServer != nil {
		if err := a.webServer.Shutdown(shutdownCtx); err != nil {
			a.logger.Printf("Error shutting down web server: %v", err)
		}
	}

	// Disconnect MQTT client
	if a.mqttClient != nil {
		a.mqttClient.Disconnect()
	}

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Println("All goroutines stopped")
	case <-shutdownCtx.Done():
		a.logger.Println("Shutdown timeout reached")
	}
}

func (a *Application) Cleanup() {
	if a.stateManager != nil {
		if err := a.stateManager.Close(); err != nil {
			a.logger.Printf("Error closing state manager: %v", err)
		}
	}
}
