package web

import (
	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
)

type PageData struct {
	Title   string
	Error   string
	Success string
}

type DashboardData struct {
	PageData
	Topics        map[string]topics.Topic
	TopicCounts   map[topics.TopicType]int
	StrategyCount int
	SystemStatus  string
	RecentLogs    []string
}

type TopicsListData struct {
	PageData
	Topics      map[string]topics.Topic
	TopicFilter string
}

type TopicEditData struct {
	PageData
	Topic      interface{}
	Strategies []strategy.Strategy
	IsNew      bool
}

type StrategiesListData struct {
	PageData
	Strategies map[string]*strategy.Strategy
}

type StrategyEditData struct {
	PageData
	Strategy *strategy.Strategy
	IsNew    bool
}

type SystemConfigData struct {
	PageData
	MQTTBroker    string
	MQTTConnected bool
	DatabaseType  string
	DatabasePath  string
	WebPort       int
	LogLevel      string
}

type LogsData struct {
	PageData
	Logs       []string
	TopicName  string
	MaxEntries int
}
