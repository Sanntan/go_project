package logger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventLogger(t *testing.T) {
	logger := NewEventLogger(100)
	require.NotNil(t, logger)
	assert.Equal(t, 100, logger.maxSize)
	assert.NotNil(t, logger.events)
	assert.Equal(t, 0, len(logger.events))
}

func TestEventLogger_LogEvent(t *testing.T) {
	logger := NewEventLogger(100)

	data := map[string]interface{}{
		"transaction_id": "TXN-001",
		"amount":         100000.0,
	}

	logger.LogEvent(EventTransactionReceived, "ingestion-service", "kafka", data)

	assert.Len(t, logger.events, 1)
	event := logger.events[0]
	assert.Equal(t, EventTransactionReceived, event.Type)
	assert.Equal(t, "ingestion-service", event.Service)
	assert.Equal(t, "kafka", event.Component)
	assert.Equal(t, data, event.Data)
	assert.NotEmpty(t, event.ID)
	assert.False(t, event.Timestamp.IsZero())
}

func TestEventLogger_LogEvent_MultipleEvents(t *testing.T) {
	logger := NewEventLogger(100)

	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"index": i,
		}
		logger.LogEvent(EventTransactionReceived, "test-service", "test", data)
	}

	assert.Len(t, logger.events, 5)
}

func TestEventLogger_LogEvent_MaxSize(t *testing.T) {
	logger := NewEventLogger(3)

	// Добавляем больше событий, чем maxSize
	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"index": i,
		}
		logger.LogEvent(EventTransactionReceived, "test-service", "test", data)
	}

	// Должно остаться только последние 3 события
	assert.Len(t, logger.events, 3)
	
	// Проверяем, что остались последние события
	assert.Equal(t, 2, logger.events[0].Data["index"])
	assert.Equal(t, 3, logger.events[1].Data["index"])
	assert.Equal(t, 4, logger.events[2].Data["index"])
}

func TestEventLogger_GetEvents(t *testing.T) {
	logger := NewEventLogger(100)

	// Добавляем события
	for i := 0; i < 10; i++ {
		data := map[string]interface{}{
			"index": i,
		}
		logger.LogEvent(EventTransactionReceived, "test-service", "test", data)
	}

	// Получаем все события
	events := logger.GetEvents(0)
	assert.Len(t, events, 10)

	// Получаем ограниченное количество
	events = logger.GetEvents(5)
	assert.Len(t, events, 5)

	// Проверяем, что возвращаются последние события
	assert.Equal(t, 5, events[0].Data["index"])
	assert.Equal(t, 9, events[4].Data["index"])
}

func TestEventLogger_GetEvents_MoreThanAvailable(t *testing.T) {
	logger := NewEventLogger(100)

	// Добавляем 3 события
	for i := 0; i < 3; i++ {
		data := map[string]interface{}{
			"index": i,
		}
		logger.LogEvent(EventTransactionReceived, "test-service", "test", data)
	}

	// Запрашиваем больше, чем есть
	events := logger.GetEvents(10)
	assert.Len(t, events, 3)
}

func TestEventLogger_GetStats(t *testing.T) {
	logger := NewEventLogger(100)

	// Добавляем разные события
	logger.LogEvent(EventTransactionReceived, "service1", "component1", map[string]interface{}{})
	logger.LogEvent(EventTransactionSaved, "service1", "component2", map[string]interface{}{})
	logger.LogEvent(EventTransactionReceived, "service2", "component1", map[string]interface{}{})

	stats := logger.GetStats()
	require.NotNil(t, stats)

	assert.Equal(t, 3, stats["total_events"])

	// Проверяем статистику по компонентам
	components, ok := stats["components"].(map[string]int)
	require.True(t, ok)
	assert.Equal(t, 2, components["component1"])
	assert.Equal(t, 1, components["component2"])

	// Проверяем статистику по сервисам
	services, ok := stats["services"].(map[string]int)
	require.True(t, ok)
	assert.Equal(t, 2, services["service1"])
	assert.Equal(t, 1, services["service2"])

	// Проверяем статистику по типам событий
	eventTypes, ok := stats["event_types"].(map[string]int)
	require.True(t, ok)
	assert.Equal(t, 2, eventTypes[string(EventTransactionReceived)])
	assert.Equal(t, 1, eventTypes[string(EventTransactionSaved)])
}

func TestLogEvent_Global(t *testing.T) {
	// Используем глобальный логгер
	data := map[string]interface{}{
		"test": "value",
	}

	LogEvent(EventTransactionReceived, "test-service", "test-component", data)

	events := GetEvents(1)
	require.Len(t, events, 1)
	assert.Equal(t, EventTransactionReceived, events[0].Type)
	assert.Equal(t, "test-service", events[0].Service)
	assert.Equal(t, "test-component", events[0].Component)
}

func TestGetEvents_Global(t *testing.T) {
	// Очищаем глобальный логгер, добавляя много событий
	for i := 0; i < 100; i++ {
		LogEvent(EventTransactionReceived, "test", "test", map[string]interface{}{})
	}

	events := GetEvents(5)
	assert.Len(t, events, 5)
}

func TestGetStats_Global(t *testing.T) {
	stats := GetStats()
	require.NotNil(t, stats)
	assert.Contains(t, stats, "total_events")
	assert.Contains(t, stats, "components")
	assert.Contains(t, stats, "services")
	assert.Contains(t, stats, "event_types")
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	
	// ID должны быть разными (если генерируются с разницей во времени)
	// Но могут быть одинаковыми, если генерируются в одну и ту же миллисекунду
	// Поэтому просто проверяем формат
	assert.Len(t, id1, 21) // Формат: 20060102150405.000000
}

func TestEvent_MarshalJSON(t *testing.T) {
	event := Event{
		ID:        "test-id",
		Type:      EventTransactionReceived,
		Service:   "test-service",
		Component: "test-component",
		Timestamp: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Data:      map[string]interface{}{"key": "value"},
	}

	jsonData, err := event.MarshalJSON()
	require.NoError(t, err)
	require.NotEmpty(t, jsonData)

	// Проверяем, что JSON содержит timestamp в формате RFC3339
	assert.Contains(t, string(jsonData), "2024-01-15T14:30:00Z")
}

func TestEventLogger_ConcurrentAccess(t *testing.T) {
	logger := NewEventLogger(1000)

	// Запускаем несколько горутин, которые одновременно пишут события
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			for j := 0; j < 10; j++ {
				data := map[string]interface{}{
					"goroutine": index,
					"event":     j,
				}
				logger.LogEvent(EventTransactionReceived, "test", "test", data)
			}
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем, что все события записаны
	events := logger.GetEvents(0)
	assert.Len(t, events, 100)
}

func TestEventLogger_DifferentEventTypes(t *testing.T) {
	logger := NewEventLogger(100)

	eventTypes := []EventType{
		EventTransactionReceived,
		EventTransactionSaved,
		EventKafkaSent,
		EventKafkaReceived,
		EventAnalysisStarted,
		EventAnalysisCompleted,
		EventDBUpdated,
	}

	for _, eventType := range eventTypes {
		logger.LogEvent(eventType, "test-service", "test-component", map[string]interface{}{})
	}

	events := logger.GetEvents(0)
	assert.Len(t, events, len(eventTypes))

	// Проверяем, что все типы событий присутствуют
	stats := logger.GetStats()
	eventTypesStats, ok := stats["event_types"].(map[string]int)
	require.True(t, ok)

	for _, eventType := range eventTypes {
		assert.Equal(t, 1, eventTypesStats[string(eventType)])
	}
}

