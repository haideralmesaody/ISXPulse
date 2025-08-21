package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.TotalConnections)
	assert.Equal(t, int64(0), metrics.ActiveConnections)
	assert.Equal(t, int64(0), metrics.MessagesSent)
	assert.Equal(t, int64(0), metrics.MessagesReceived)
}

func TestMetrics_RecordConnection(t *testing.T) {
	metrics := NewMetrics()
	
	// Record a connection
	metrics.RecordConnection()
	
	assert.Equal(t, int64(1), metrics.TotalConnections)
	assert.Equal(t, int64(1), metrics.ActiveConnections)
}

func TestMetrics_RecordDisconnection(t *testing.T) {
	metrics := NewMetrics()
	
	// First record a connection
	metrics.RecordConnection()
	assert.Equal(t, int64(1), metrics.ActiveConnections)
	
	// Then record disconnection
	duration := 5 * time.Minute
	metrics.RecordDisconnection(duration)
	
	assert.Equal(t, int64(0), metrics.ActiveConnections)
}

func TestMetrics_RecordMessage(t *testing.T) {
	metrics := NewMetrics()
	
	// Record sent message
	metrics.RecordMessage("sent", 256, true)
	
	assert.Equal(t, int64(1), metrics.MessagesSent)
	assert.Equal(t, int64(256), metrics.BytesSent)
	
	// Record received message
	metrics.RecordMessage("received", 128, true)
	
	assert.Equal(t, int64(1), metrics.MessagesReceived)
	assert.Equal(t, int64(128), metrics.BytesReceived)
	
	// Record failed message
	metrics.RecordMessage("sent", 64, false)
	assert.Equal(t, int64(1), metrics.MessageErrors)
}

func TestMetrics_RecordError(t *testing.T) {
	metrics := NewMetrics()
	
	// Record different error types
	metrics.RecordError("connection")
	metrics.RecordError("message")
	metrics.RecordError("connection")
	
	// Check error counts
	metrics.mu.RLock()
	connErrors := metrics.ErrorsByType["connection"]
	msgErrors := metrics.ErrorsByType["message"]
	metrics.mu.RUnlock()
	
	assert.Equal(t, int64(2), connErrors)
	assert.Equal(t, int64(1), msgErrors)
}

func TestMetrics_RecordQueueDepth(t *testing.T) {
	metrics := NewMetrics()
	
	// Record queue depths
	metrics.RecordQueueDepth(10)
	metrics.RecordQueueDepth(15)
	metrics.RecordQueueDepth(5)
	
	// Check that max was recorded
	assert.Equal(t, int64(15), metrics.MaxQueueDepth)
}

func TestMetrics_RecordDroppedMessage(t *testing.T) {
	metrics := NewMetrics()
	
	// Record dropped messages
	metrics.RecordDroppedMessage()
	metrics.RecordDroppedMessage()
	metrics.RecordDroppedMessage()
	
	assert.Equal(t, int64(3), metrics.DroppedMessages)
}

func TestMetrics_GetSnapshot(t *testing.T) {
	metrics := NewMetrics()
	
	// Setup some metrics
	metrics.RecordConnection()
	metrics.RecordConnection()
	metrics.RecordDisconnection(1 * time.Minute)
	
	metrics.RecordMessage("sent", 100, true)
	metrics.RecordMessage("sent", 200, true)
	metrics.RecordMessage("received", 50, true)
	
	metrics.RecordError("connection")
	metrics.RecordDroppedMessage()
	
	// Get snapshot
	snapshot := metrics.GetSnapshot()
	
	// Access nested structure
	connections := snapshot["connections"].(map[string]interface{})
	messages := snapshot["messages"].(map[string]interface{})
	
	assert.Equal(t, int64(1), connections["active"])
	assert.Equal(t, int64(2), connections["total"])
	assert.Equal(t, int64(2), messages["sent"])
	assert.Equal(t, int64(1), messages["received"])
	assert.Equal(t, int64(300), messages["bytes_sent"])
	assert.Equal(t, int64(50), messages["bytes_received"])
	assert.Equal(t, int64(1), messages["dropped"])
	assert.NotNil(t, snapshot["errors"])
	assert.NotZero(t, snapshot["uptime_seconds"])
}

func TestMetrics_Reset(t *testing.T) {
	metrics := NewMetrics()
	
	// Setup metrics
	metrics.RecordConnection()
	metrics.RecordMessage("sent", 100, true)
	metrics.RecordError("test")
	metrics.RecordQueueDepth(10)
	metrics.RecordDroppedMessage()
	
	// Verify metrics are set
	assert.Greater(t, metrics.ActiveConnections, int64(0))
	assert.Greater(t, metrics.MessagesSent, int64(0))
	
	// Reset
	metrics.Reset()
	
	// Verify all metrics are reset
	assert.Equal(t, int64(0), metrics.TotalConnections)
	assert.Equal(t, int64(0), metrics.ActiveConnections)
	assert.Equal(t, int64(0), metrics.MessagesSent)
	assert.Equal(t, int64(0), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.BytesSent)
	assert.Equal(t, int64(0), metrics.BytesReceived)
	assert.Equal(t, int64(0), metrics.DroppedMessages)
	assert.Equal(t, int64(0), metrics.MessageErrors)
	assert.Equal(t, int64(0), metrics.MaxQueueDepth)
	
	// Check map is empty
	metrics.mu.RLock()
	assert.Empty(t, metrics.ErrorsByType)
	metrics.mu.RUnlock()
}

func TestGetMetrics(t *testing.T) {
	// Test singleton pattern
	metrics1 := GetMetrics()
	metrics2 := GetMetrics()
	
	assert.NotNil(t, metrics1)
	assert.Same(t, metrics1, metrics2) // Should be same instance
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewMetrics()
	
	// Test concurrent access to metrics
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100
	
	// Concurrent connections
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				metrics.RecordConnection()
			}
		}(i)
	}
	
	// Concurrent messages
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				metrics.RecordMessage("sent", 100, true)
				metrics.RecordMessage("received", 50, true)
			}
		}(i)
	}
	
	// Concurrent errors
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				metrics.RecordError("test")
				metrics.RecordDroppedMessage()
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify results
	expectedConnections := int64(numGoroutines * numOperations)
	expectedMessages := int64(numGoroutines * numOperations)
	expectedErrors := int64(numGoroutines * numOperations)
	
	assert.Equal(t, expectedConnections, metrics.ActiveConnections)
	assert.Equal(t, expectedConnections, metrics.TotalConnections)
	assert.Equal(t, expectedMessages, metrics.MessagesSent)
	assert.Equal(t, expectedMessages, metrics.MessagesReceived)
	assert.Equal(t, expectedErrors, metrics.DroppedMessages)
}

func TestMetrics_EdgeCases(t *testing.T) {
	metrics := NewMetrics()
	
	// Test invalid direction
	metrics.RecordMessage("invalid", 100, true)
	// Should not crash, but won't increment sent/received
	assert.Equal(t, int64(0), metrics.MessagesSent)
	assert.Equal(t, int64(0), metrics.MessagesReceived)
	
	// Test empty error type
	metrics.RecordError("")
	// Should still record
	metrics.mu.RLock()
	emptyErrors := metrics.ErrorsByType[""]
	metrics.mu.RUnlock()
	assert.Equal(t, int64(1), emptyErrors)
	
	// Test negative duration (shouldn't happen but test resilience)
	metrics.RecordConnection()
	metrics.RecordDisconnection(-1 * time.Second)
	assert.Equal(t, int64(0), metrics.ActiveConnections) // Should still decrement
}

