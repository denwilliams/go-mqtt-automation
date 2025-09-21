package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

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
	templates      *template.Template
	server         *http.Server
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
	}

	// Load templates
	if err := server.loadTemplates(); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) loadTemplates() error {
	// Try different possible template paths
	possiblePaths := []string{
		filepath.Join("web", "templates", "*.html"),
		filepath.Join(".", "web", "templates", "*.html"),
		filepath.Join("..", "web", "templates", "*.html"),
	}

	var templates *template.Template
	var err error

	for _, templatePath := range possiblePaths {
		s.logger.Printf("Attempting to load templates from: %s", templatePath)
		templates, err = template.New("").Funcs(template.FuncMap{
			"toJSON": func(v interface{}) string {
				if v == nil {
					return "null"
				}
				b, jsonErr := json.Marshal(v)
				if jsonErr != nil {
					return fmt.Sprintf("%v", v)
				}
				return string(b)
			},
			"truncate": func(s string, length int) string {
				if len(s) <= length {
					return s
				}
				return s[:length] + "..."
			},
		}).ParseGlob(templatePath)
		if err == nil && templates != nil {
			// Check if we actually loaded any templates
			if len(templates.Templates()) > 0 {
				s.logger.Printf("Successfully loaded %d templates from: %s", len(templates.Templates()), templatePath)
				s.templates = templates
				return nil
			}
		}
		s.logger.Printf("Failed to load templates from %s: %v", templatePath, err)
	}

	// Create a comprehensive fallback template set for development
	s.logger.Printf("Creating fallback template set")
	templates = template.New("base").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			if v == nil {
				return "null"
			}
			b, jsonErr := json.Marshal(v)
			if jsonErr != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
	})

	// Base template
	baseTemplate := `{{define "base"}}<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Home Automation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #333; color: white; padding: 1rem; margin-bottom: 2rem; }
        .nav a { color: white; text-decoration: none; margin-right: 1rem; }
        .card { background: white; padding: 1rem; margin: 1rem 0; border-radius: 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
        .alert { padding: 1rem; margin: 1rem 0; border-radius: 5px; }
        .alert-error { background: #f8d7da; color: #721c24; }
        .alert-success { background: #d4edda; color: #155724; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.5rem; border: 1px solid #ddd; text-align: left; }
        th { background: #f8f9fa; }
        .btn { padding: 0.5rem 1rem; background: #007bff; color: white; text-decoration: none; border-radius: 3px; }
        .dashboard-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin: 2rem 0; }
        .stat-card { background: white; padding: 2rem; text-align: center; border-radius: 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
        .stat-number { font-size: 2rem; font-weight: bold; color: #007bff; }
        .stat-label { color: #666; margin-top: 0.5rem; }
    </style>
</head>
<body>
    <div class="header">
        <div class="container">
            <h1>MQTT Home Automation</h1>
            <nav class="nav">
                <a href="/">Dashboard</a>
                <a href="/topics">Topics</a>
                <a href="/strategies">Strategies</a>
                <a href="/system">System</a>
                <a href="/logs">Logs</a>
            </nav>
        </div>
    </div>
    <div class="container">
        {{if .Error}}
        <div class="alert alert-error">{{.Error}}</div>
        {{end}}
        {{if .Success}}
        <div class="alert alert-success">{{.Success}}</div>
        {{end}}
        <div class="main-content">
            {{template "content" .}}
        </div>
    </div>
</body>
</html>{{end}}`

	_, err = templates.Parse(baseTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse base template: %w", err)
	}

	// Dashboard content template
	dashboardTemplate := `{{define "content"}}
<h2>Dashboard</h2>
<div class="dashboard-grid">
    <div class="stat-card">
        <div class="stat-number">{{if .Topics}}{{len .Topics}}{{else}}0{{end}}</div>
        <div class="stat-label">Total Topics</div>
    </div>
    <div class="stat-card">
        <div class="stat-number">{{if .StrategyCount}}{{.StrategyCount}}{{else}}0{{end}}</div>
        <div class="stat-label">Strategies</div>
    </div>
    <div class="stat-card">
        <div class="stat-number">{{if .SystemStatus}}{{.SystemStatus}}{{else}}Unknown{{end}}</div>
        <div class="stat-label">MQTT Status</div>
    </div>
</div>
<div class="card">
    <h3>System Information</h3>
    <p>Web UI is running with fallback templates. Check server logs for template loading issues.</p>
    <p><strong>Template Status:</strong> Using built-in fallback templates</p>
    <p><strong>Topics:</strong> {{if .Topics}}{{len .Topics}} configured{{else}}No topics configured{{end}}</p>
    <p><strong>Strategies:</strong> {{if .StrategyCount}}{{.StrategyCount}} loaded{{else}}No strategies loaded{{end}}</p>
</div>
{{end}}`

	_, err = templates.Parse(dashboardTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse dashboard template: %w", err)
	}

	// Topics content template
	topicsTemplate := `{{define "topics-content"}}
<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
    <h2>Topics</h2>
    <a href="/topics/new" class="btn">Create New Topic</a>
</div>
{{if .Topics}}
<div class="card">
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Last Value</th>
                <th>Last Updated</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range $name, $topic := .Topics}}
            <tr>
                <td><strong>{{$name}}</strong></td>
                <td>{{$topic.Type}}</td>
                <td>{{if $topic.LastValue}}{{$topic.LastValue}}{{else}}<em>null</em>{{end}}</td>
                <td>{{if not $topic.LastUpdated.IsZero}}{{$topic.LastUpdated.Format "2006-01-02 15:04:05"}}{{else}}<em>Never</em>{{end}}</td>
                <td>
                    {{if eq $topic.Type "internal"}}
                    <a href="/topics/edit/{{$name}}" class="btn">Edit</a>
                    {{else}}
                    <span>Read-only</span>
                    {{end}}
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{else}}
<div class="card">
    <p>No topics found. <a href="/topics/new">Create your first topic</a>.</p>
</div>
{{end}}
{{end}}`

	_, err = templates.Parse(topicsTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse topics template: %w", err)
	}

	// Strategies content template
	strategiesTemplate := `{{define "strategies-content"}}
<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
    <h2>Strategies</h2>
    <a href="/strategies/new" class="btn">Create New Strategy</a>
</div>
{{if .Strategies}}
<div class="card">
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>ID</th>
                <th>Language</th>
                <th>Created</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range $id, $strategy := .Strategies}}
            <tr>
                <td><strong>{{$strategy.Name}}</strong></td>
                <td><code>{{$strategy.ID}}</code></td>
                <td>{{$strategy.Language}}</td>
                <td>{{$strategy.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td>
                    <a href="/strategies/edit/{{$strategy.ID}}" class="btn">Edit</a>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{else}}
<div class="card">
    <p>No strategies found. <a href="/strategies/new">Create your first strategy</a>.</p>
</div>
{{end}}
{{end}}`

	_, err = templates.Parse(strategiesTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse strategies template: %w", err)
	}

	// Topic strategy selection content template
	topicStrategySelectTemplate := `{{define "topic-strategy-select-content"}}
<h2>Create New Topic - Select Strategy</h2>
<div class="card">
    <p>Choose the strategy that will process inputs for your new topic:</p>
    {{if .Strategies}}
    <div style="display: grid; gap: 15px; margin: 20px 0;">
        {{range .Strategies}}
        <div style="background: #f8f9fa; padding: 15px; border-radius: 5px; border-left: 4px solid #007bff;">
            <h4>{{.Name}}</h4>
            <p><strong>Language:</strong> {{.Language}}</p>
            <p><strong>Max Inputs:</strong> {{if gt .MaxInputs 0}}{{.MaxInputs}}{{else}}Unlimited{{end}}</p>
            {{if .DefaultInputNames}}
            <p><strong>Suggested Input Names:</strong> {{range $i, $name := .DefaultInputNames}}{{if $i}}, {{end}}"{{$name}}"{{end}}</p>
            {{end}}
            <div style="margin-bottom: 10px;">
                <small><strong>Code Preview:</strong></small>
                <pre style="font-size: 12px; max-height: 100px; overflow: hidden; background: #f1f1f1; padding: 8px; border-radius: 3px;">{{if gt (len .Code) 200}}{{slice .Code 0 200}}...{{else}}{{.Code}}{{end}}</pre>
            </div>
            <a href="/topics/new/{{.ID}}" class="btn">Use This Strategy</a>
        </div>
        {{end}}
    </div>
    {{else}}
    <p>No strategies available. <a href="/strategies/new" class="btn">Create a strategy first</a>.</p>
    {{end}}
    <div style="margin-top: 20px;">
        <a href="/topics" class="btn" style="background: #6c757d;">Cancel</a>
        <a href="/strategies/new" class="btn" style="margin-left: 10px;">Create New Strategy</a>
    </div>
</div>
{{end}}`

	_, err = templates.Parse(topicStrategySelectTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse topic strategy select template: %w", err)
	}

	// Topic edit content template
	topicEditTemplate := `{{define "topic-edit-content"}}
<h2>{{if .IsNew}}Create New Topic{{else}}Edit Topic{{end}}{{if .SelectedStrategy}} with {{.SelectedStrategy.Name}} Strategy{{end}}</h2>

<div class="card">
    <form method="POST" action="{{if .IsNew}}/topics/new{{else}}/topics/edit/{{.Topic.Name}}{{end}}">
        <div style="margin-bottom: 15px;">
            <label for="name"><strong>Topic Name:</strong></label>
            <input type="text" id="name" name="name" value="{{if .Topic}}{{.Topic.Name}}{{end}}"
                   {{if not .IsNew}}readonly{{end}}
                   style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;" required>
            <small>Use a descriptive name like 'bedroom_temperature_alert' or 'car_battery_status'</small>
        </div>

        {{if .SelectedStrategy}}
        <input type="hidden" name="strategy_id" value="{{.SelectedStrategy.ID}}">
        <div style="margin-bottom: 15px;">
            <label><strong>Strategy:</strong></label>
            <div style="background: #e7f3ff; padding: 10px; border-radius: 4px;">
                <strong>{{.SelectedStrategy.Name}}</strong> ({{.SelectedStrategy.Language}})
                {{if .SelectedStrategy.DefaultInputNames}}
                <br><small>Suggested inputs: {{range $i, $name := .SelectedStrategy.DefaultInputNames}}{{if $i}}, {{end}}"{{$name}}"{{end}}</small>
                {{end}}
            </div>
        </div>
        {{else if .Strategies}}
        <div style="margin-bottom: 15px;">
            <label for="strategy_id"><strong>Strategy:</strong></label>
            <select id="strategy_id" name="strategy_id" style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;" required>
                <option value="">Select a strategy...</option>
                {{range .Strategies}}
                <option value="{{.ID}}" {{if and $.Topic (eq $.Topic.GetStrategyID .ID)}}selected{{end}}>{{.Name}} ({{.Language}})</option>
                {{end}}
            </select>
        </div>
        {{end}}

        <div style="margin-bottom: 15px;">
            <label for="inputs"><strong>Input Topics:</strong></label>
            <textarea id="inputs" name="inputs" rows="4"
                      style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;"
                      placeholder="Enter one topic per line, e.g.:&#10;teslamate/cars/1/+&#10;sensibo/status/+&#10;home/temperature">{{if .Topic}}{{range .Topic.GetInputs}}{{.}}
{{end}}{{end}}</textarea>
            <small>Enter MQTT topic patterns, one per line. Use + for single-level wildcards, # for multi-level wildcards.</small>
        </div>

        <div style="margin-bottom: 15px;">
            <label>
                <input type="checkbox" name="emit_to_mqtt" {{if and .Topic .Topic.ShouldEmitToMQTT}}checked{{end}}>
                <strong>Emit to MQTT</strong>
            </label>
            <br><small>When checked, this topic's output will be published to MQTT</small>
        </div>

        <div style="margin-bottom: 15px;">
            <label>
                <input type="checkbox" name="noop_unchanged" {{if and .Topic .Topic.IsNoOpUnchanged}}checked{{end}}>
                <strong>Skip Unchanged Values</strong>
            </label>
            <br><small>When checked, the topic won't emit if the new value is the same as the previous value</small>
        </div>

        <div style="margin-top: 20px;">
            <button type="submit" class="btn">{{if .IsNew}}Create Topic{{else}}Update Topic{{end}}</button>
            <a href="/topics" class="btn" style="background: #6c757d; margin-left: 10px;">Cancel</a>
        </div>
    </form>
</div>
{{end}}`

	_, err = templates.Parse(topicEditTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse topic edit template: %w", err)
	}

	s.templates = templates
	s.logger.Printf("Fallback templates created successfully")
	return nil
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
	// Static files - try multiple paths
	s.setupStaticFiles()

	// Main pages
	http.HandleFunc("/", s.handleDashboard)
	http.HandleFunc("/topics", s.handleTopicsList)
	http.HandleFunc("/topics/new", s.handleTopicNew)
	http.HandleFunc("/topics/new/", s.handleTopicNew)
	http.HandleFunc("/topics/edit/", s.handleTopicEdit)
	http.HandleFunc("/topics/delete/", s.handleTopicDelete)
	http.HandleFunc("/strategies", s.handleStrategiesList)
	http.HandleFunc("/strategies/new", s.handleStrategyNew)
	http.HandleFunc("/strategies/edit/", s.handleStrategyEdit)
	http.HandleFunc("/strategies/delete/", s.handleStrategyDelete)
	http.HandleFunc("/system", s.handleSystemConfig)
	http.HandleFunc("/logs", s.handleLogs)

	// API endpoints
	http.HandleFunc("/api/topics", s.handleAPITopics)
	http.HandleFunc("/api/strategies", s.handleAPIStrategies)
	http.HandleFunc("/api/system/status", s.handleAPISystemStatus)

	// Debug endpoint
	http.HandleFunc("/debug", s.handleDebug)

	s.logger.Println("Web server routes configured")
}

func (s *Server) setupStaticFiles() {
	// Try different possible static file paths
	possibleStaticPaths := []string{
		"web/static/",
		"./web/static/",
		"../web/static/",
	}

	for _, staticPath := range possibleStaticPaths {
		if _, err := http.Dir(staticPath).Open("style.css"); err == nil {
			s.logger.Printf("Serving static files from: %s", staticPath)
			http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))
			return
		}
	}

	// If no static files found, create a minimal CSS handler
	s.logger.Printf("No static files found, creating minimal CSS handler")
	http.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		// CSS is already included in the fallback template, so just return empty
		w.WriteHeader(http.StatusOK)
	})
}

func (s *Server) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	// TODO: this is an AI generated mess. Needs some human love and direction.

	if s.templates == nil {
		s.logger.Printf("Templates not loaded when trying to render %s", templateName)
		http.Error(w, "Templates not loaded", http.StatusInternalServerError)
		return
	}

	s.logger.Printf("Attempting to render template: %s", templateName)


	// Determine the content template name based on the page template
	var contentTemplateName string
	switch templateName {
	case "dashboard.html":
		contentTemplateName = "dashboard-content"
	case "topics.html":
		contentTemplateName = "topics-content"
	case "strategies.html":
		contentTemplateName = "strategies-content"
	case "system.html":
		contentTemplateName = "system-content"
	case "logs.html":
		contentTemplateName = "logs-content"
	case "topic_edit.html":
		contentTemplateName = "topic-edit-content"
	case "strategy_edit.html":
		contentTemplateName = "strategy-edit-content"
	case "topic_strategy_select.html":
		contentTemplateName = "topic-strategy-select-content"
	default:
		contentTemplateName = "content" // fallback
	}

	// Create a custom base template that calls the specific content template
	baseTemplateWithContent := fmt.Sprintf(`{{define "base-with-content"}}{{template "base" .}}{{end}}
{{define "content"}}{{template "%s" .}}{{end}}`, contentTemplateName)

	// Parse the dynamic template
	tempTemplate, err := s.templates.Clone()
	if err != nil {
		s.logger.Printf("Failed to clone template: %v", err)
		s.renderFallbackForTemplate(templateName, w, data)
		return
	}

	_, err = tempTemplate.Parse(baseTemplateWithContent)
	if err != nil {
		s.logger.Printf("Failed to parse dynamic template: %v", err)
		s.renderFallbackForTemplate(templateName, w, data)
		return
	}

	// Execute the base template which now calls the correct content template
	if err := tempTemplate.ExecuteTemplate(w, "base", data); err != nil {
		s.logger.Printf("Failed to execute base template for %s: %v", templateName, err)
		s.renderFallbackForTemplate(templateName, w, data)
		return
	}
}

func (s *Server) renderFallbackForTemplate(templateName string, w http.ResponseWriter, data interface{}) {
	switch templateName {
	case "dashboard.html":
		s.renderFallbackDashboard(w, data)
	case "topics.html":
		s.renderFallbackTopics(w, data)
	case "strategies.html":
		s.renderFallbackStrategies(w, data)
	default:
		s.renderErrorPage(w, fmt.Sprintf("Template Error for %s", templateName))
	}
}

func (s *Server) renderFallbackDashboard(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "text/html")

	// Type assert the data to get the dashboard information
	dashData, ok := data.(DashboardData)
	if !ok {
		s.logger.Printf("Data is not DashboardData type: %T", data)
		dashData = DashboardData{
			PageData: PageData{Title: "Dashboard"},
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Home Automation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 20px; border-radius: 10px; }
        .header { background: #333; color: white; padding: 15px; margin: -20px -20px 20px -20px; border-radius: 10px 10px 0 0; }
        .nav a { color: white; text-decoration: none; margin-right: 15px; }
        .stats { display: flex; gap: 20px; margin: 20px 0; }
        .stat { background: #e7f3ff; padding: 15px; border-radius: 5px; text-align: center; flex: 1; }
        .stat-number { font-size: 24px; font-weight: bold; color: #0066cc; }
        .info { background: #f0f8ff; padding: 15px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>MQTT Home Automation Dashboard</h1>
            <nav class="nav">
                <a href="/">Dashboard</a>
                <a href="/topics">Topics</a>
                <a href="/strategies">Strategies</a>
                <a href="/system">System</a>
                <a href="/logs">Logs</a>
            </nav>
        </div>
        
        <div class="stats">
            <div class="stat">
                <div class="stat-number">%d</div>
                <div>Topics</div>
            </div>
            <div class="stat">
                <div class="stat-number">%d</div>
                <div>Strategies</div>
            </div>
            <div class="stat">
                <div class="stat-number">%s</div>
                <div>MQTT Status</div>
            </div>
        </div>
        
        <div class="info">
            <h3>System Status</h3>
            <p><strong>Web UI Status:</strong> Running with fallback rendering</p>
            <p><strong>Topics:</strong> %d configured</p>
            <p><strong>Strategies:</strong> %d loaded</p>
            <p><strong>MQTT:</strong> %s</p>
        </div>
        
        <div class="info">
            <p>The web UI is working but using simplified templates. Check server logs for more detailed information.</p>
            <p>Try navigating to different sections using the menu above.</p>
        </div>
    </div>
</body>
</html>`,
		dashData.Title,
		len(dashData.Topics),
		dashData.StrategyCount,
		dashData.SystemStatus,
		len(dashData.Topics),
		dashData.StrategyCount,
		dashData.SystemStatus)

	fmt.Fprint(w, html)
}

func (s *Server) renderFallbackTopics(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "text/html")

	// Type assert the data to get the topics information
	topicsData, ok := data.(TopicsListData)
	if !ok {
		s.logger.Printf("Data is not TopicsListData type: %T", data)
		topicsData = TopicsListData{
			PageData: PageData{Title: "Topics"},
			Topics:   make(map[string]topics.Topic),
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Home Automation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1000px; margin: 0 auto; background: white; padding: 20px; border-radius: 10px; }
        .header { background: #333; color: white; padding: 15px; margin: -20px -20px 20px -20px; border-radius: 10px 10px 0 0; }
        .nav a { color: white; text-decoration: none; margin-right: 15px; }
        .nav a:hover { text-decoration: underline; }
        .card { background: #f8f9fa; padding: 20px; margin: 20px 0; border-radius: 8px; border: 1px solid #dee2e6; }
        .topic-item { background: white; padding: 15px; margin: 10px 0; border-radius: 5px; border-left: 4px solid #007bff; }
        .topic-type { padding: 4px 8px; border-radius: 3px; font-size: 12px; color: white; }
        .topic-type-external { background: #28a745; }
        .topic-type-internal { background: #007bff; }
        .topic-type-system { background: #6c757d; }
        .btn { padding: 8px 16px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; display: inline-block; }
        .btn:hover { background: #0056b3; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>MQTT Home Automation</h1>
            <nav class="nav">
                <a href="/">Dashboard</a>
                <a href="/topics">Topics</a>
                <a href="/strategies">Strategies</a>
                <a href="/system">System</a>
                <a href="/logs">Logs</a>
            </nav>
        </div>
        
        <h2>Topics (%d)</h2>
        
        <div class="card">
            <p><strong>Note:</strong> Using fallback topic display. Check server logs for template issues.</p>
            <a href="/topics/new" class="btn">Add New Topic</a>
        </div>
        
        %s
    </div>
</body>
</html>`,
		topicsData.Title,
		len(topicsData.Topics),
		func() string {
			if len(topicsData.Topics) == 0 {
				return `<div class="card">
                    <p>No topics configured yet. <a href="/topics/new" class="btn">Create your first topic</a>.</p>
                </div>`
			}

			result := ""
			for name, topic := range topicsData.Topics {
				topicType := string(topic.Type())
				lastValue := "null"
				if topic.LastValue() != nil {
					if jsonBytes, err := json.Marshal(topic.LastValue()); err == nil {
						lastValue = string(jsonBytes)
					} else {
						lastValue = fmt.Sprintf("%v", topic.LastValue())
					}
					if len(lastValue) > 100 {
						lastValue = lastValue[:100] + "..."
					}
				}

				lastUpdated := "Never"
				if !topic.LastUpdated().IsZero() {
					lastUpdated = topic.LastUpdated().Format("2006-01-02 15:04:05")
				}

				result += fmt.Sprintf(`
                <div class="topic-item">
                    <h4>%s <span class="topic-type topic-type-%s">%s</span></h4>
                    <p><strong>Last Value:</strong> %s</p>
                    <p><strong>Last Updated:</strong> %s</p>
                    <div style="margin-top: 10px;">
                        <a href="/topics/edit/%s" class="btn" style="font-size: 12px; padding: 4px 8px;">Edit</a>
                    </div>
                </div>`, name, topicType, topicType, lastValue, lastUpdated, name)
			}
			return result
		}())

	fmt.Fprint(w, html)
}

func (s *Server) renderFallbackStrategies(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "text/html")

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Strategies - Home Automation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 20px; border-radius: 10px; }
        .header { background: #333; color: white; padding: 15px; margin: -20px -20px 20px -20px; border-radius: 10px 10px 0 0; }
        .nav a { color: white; text-decoration: none; margin-right: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>MQTT Home Automation</h1>
            <nav class="nav">
                <a href="/">Dashboard</a>
                <a href="/topics">Topics</a>
                <a href="/strategies">Strategies</a>
                <a href="/system">System</a>
                <a href="/logs">Logs</a>
            </nav>
        </div>
        
        <h2>Strategies</h2>
        <p>Strategies page is using fallback rendering. Check server logs for template issues.</p>
        <p><a href="/">← Back to Dashboard</a></p>
    </div>
</body>
</html>`

	fmt.Fprint(w, html)
}

func (s *Server) renderErrorPage(w http.ResponseWriter, errorMsg string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK) // Don't return 500, just show the error page

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Error - Home Automation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 20px; border-radius: 10px; }
        .error { background: #ffe6e6; padding: 15px; border-radius: 5px; margin: 20px 0; border-left: 5px solid #ff0000; }
        .nav a { margin-right: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>MQTT Home Automation</h1>
        <nav class="nav">
            <a href="/">Dashboard</a>
            <a href="/topics">Topics</a>
            <a href="/strategies">Strategies</a>
            <a href="/system">System</a>
            <a href="/logs">Logs</a>
        </nav>
        
        <div class="error">
            <h3>Template Error</h3>
            <p>%s</p>
        </div>
        
        <p>The web interface encountered an error rendering this page. The system is still operational.</p>
        <p>Try accessing other sections or check the server logs for more information.</p>
    </div>
</body>
</html>`, errorMsg)

	fmt.Fprint(w, html)
}

func (s *Server) handleDebug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// Get basic system info
	topics := make(map[string]topics.Topic)
	if s.topicManager != nil {
		topics = s.topicManager.ListTopics()
	}

	strategies := 0
	if s.strategyEngine != nil {
		strategies = s.strategyEngine.GetStrategyCount()
	}

	templateCount := 0
	templateList := ""
	if s.templates != nil {
		templateCount = len(s.templates.Templates())
		for i, t := range s.templates.Templates() {
			if i > 0 {
				templateList += ", "
			}
			templateList += t.Name()
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Debug Info - Home Automation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .info { background: #f0f8ff; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .success { background: #d4edda; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .warning { background: #fff3cd; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .error { background: #f8d7da; padding: 15px; margin: 10px 0; border-radius: 5px; }
        code { background: #f1f1f1; padding: 2px 4px; }
    </style>
</head>
<body>
    <h1>Debug Information</h1>
    
    <div class="info">
        <h3>Web Server Status</h3>
        <p><strong>Status:</strong> Running</p>
        <p><strong>Port:</strong> %d</p>
        <p><strong>Bind Address:</strong> %s</p>
        <p><a href="/">← Back to Dashboard</a></p>
    </div>
    
    <div class="info">
        <h3>Template System</h3>
        <p><strong>Templates Loaded:</strong> %d</p>
        <p><strong>Template Names:</strong> %s</p>
        <p><strong>Template Status:</strong> %s</p>
    </div>
    
    <div class="info">
        <h3>Data Management</h3>
        <p><strong>Topic Manager:</strong> %s</p>
        <p><strong>Strategy Engine:</strong> %s</p>
        <p><strong>State Manager:</strong> %s</p>
        <p><strong>MQTT Client:</strong> %s</p>
    </div>
    
    <div class="info">
        <h3>Current Data</h3>
        <p><strong>Topics:</strong> %d configured</p>
        <p><strong>Strategies:</strong> %d loaded</p>
        <p><strong>MQTT Status:</strong> %s</p>
    </div>
    
    <div class="info">
        <h3>Quick Tests</h3>
        <p><a href="/api/system/status">API System Status</a></p>
        <p><a href="/api/topics">API Topics</a></p>
        <p><a href="/api/strategies">API Strategies</a></p>
    </div>
</body>
</html>`,
		s.config.Web.Port,
		s.config.Web.Bind,
		templateCount,
		templateList,
		func() string {
			if templateCount > 0 {
				return "OK"
			}
			return "Using fallback templates"
		}(),
		func() string {
			if s.topicManager != nil {
				return "Initialized"
			}
			return "Not initialized"
		}(),
		func() string {
			if s.strategyEngine != nil {
				return "Initialized"
			}
			return "Not initialized"
		}(),
		func() string {
			if s.stateManager != nil {
				return "Initialized"
			}
			return "Not initialized"
		}(),
		func() string {
			if s.mqttClient != nil {
				return "Initialized"
			}
			return "Not initialized"
		}(),
		len(topics),
		strategies,
		s.getSystemStatus())

	fmt.Fprint(w, html)
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
