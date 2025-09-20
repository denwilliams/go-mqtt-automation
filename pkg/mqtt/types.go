package mqtt

import (
	"time"
)

type Event struct {
	Topic     string
	Payload   []byte
	Timestamp time.Time
}

type EventHandler func(event Event) error

type ConnectionState int

const (
	ConnectionStateClosed ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateConnected
	ConnectionStateReconnecting
)
