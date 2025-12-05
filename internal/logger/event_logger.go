package logger

import (
	"encoding/json"
	"sync"
	"time"
)

type EventType string

const (
	EventTransactionReceived EventType = "transaction_received"
	EventTransactionSaved   EventType = "transaction_saved"
	EventKafkaSent         EventType = "kafka_sent"
	EventKafkaReceived     EventType = "kafka_received"
	EventRedisSaved        EventType = "redis_saved"
	EventAnalysisStarted   EventType = "analysis_started"
	EventAnalysisCompleted EventType = "analysis_completed"
	EventDBUpdated         EventType = "db_updated"
)

type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Service   string                 `json:"service"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Component string                 `json:"component"` // kafka, redis, sqlite, etc.
}

type EventLogger struct {
	events []Event
	mu     sync.RWMutex
	maxSize int
}

var globalLogger *EventLogger

func init() {
	globalLogger = NewEventLogger(1000) // Храним последние 1000 событий
}

func NewEventLogger(maxSize int) *EventLogger {
	return &EventLogger{
		events:  make([]Event, 0, maxSize),
		maxSize: maxSize,
	}
}

func LogEvent(eventType EventType, service string, component string, data map[string]interface{}) {
	globalLogger.LogEvent(eventType, service, component, data)
}

func (el *EventLogger) LogEvent(eventType EventType, service string, component string, data map[string]interface{}) {
	el.mu.Lock()
	defer el.mu.Unlock()

	event := Event{
		ID:        generateID(),
		Type:      eventType,
		Service:   service,
		Component: component,
		Timestamp: time.Now(),
		Data:      data,
	}

	el.events = append(el.events, event)
	
	// Ограничиваем размер
	if len(el.events) > el.maxSize {
		el.events = el.events[len(el.events)-el.maxSize:]
	}
}

func GetEvents(limit int) []Event {
	return globalLogger.GetEvents(limit)
}

func (el *EventLogger) GetEvents(limit int) []Event {
	el.mu.RLock()
	defer el.mu.RUnlock()

	if limit <= 0 || limit > len(el.events) {
		limit = len(el.events)
	}

	// Возвращаем последние события
	start := len(el.events) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Event, limit)
	copy(result, el.events[start:])
	return result
}

func GetStats() map[string]interface{} {
	return globalLogger.GetStats()
}

func (el *EventLogger) GetStats() map[string]interface{} {
	el.mu.RLock()
	defer el.mu.RUnlock()

	stats := make(map[string]interface{})
	componentStats := make(map[string]int)
	serviceStats := make(map[string]int)
	typeStats := make(map[string]int)

	for _, event := range el.events {
		componentStats[event.Component]++
		serviceStats[event.Service]++
		typeStats[string(event.Type)]++
	}

	stats["total_events"] = len(el.events)
	stats["components"] = componentStats
	stats["services"] = serviceStats
	stats["event_types"] = typeStats

	return stats
}

func generateID() string {
	return time.Now().Format("20060102150405.000000")
}

func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(&e),
	})
}

