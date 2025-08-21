package websocket

import (
	"sync"
	"time"
)

// Metrics tracks WebSocket performance metrics
type Metrics struct {
	mu sync.RWMutex
	
	// Connection metrics
	TotalConnections    int64
	ActiveConnections   int64
	FailedConnections   int64
	MaxConcurrent       int64
	AvgConnectionTime   time.Duration
	
	// Message metrics
	MessagesSent        int64
	MessagesReceived    int64
	BytesSent           int64
	BytesReceived       int64
	MessageErrors       int64
	
	// Performance metrics
	AvgMessageSize      int64
	AvgQueueDepth       int64
	MaxQueueDepth       int64
	DroppedMessages     int64
	
	// Error metrics by type
	ErrorsByType        map[string]int64
	
	// Time-based metrics
	LastReset           time.Time
	connectionTimes     []time.Duration
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		ErrorsByType:    make(map[string]int64),
		LastReset:       time.Now(),
		connectionTimes: make([]time.Duration, 0, 100),
	}
}

// RecordConnection records a new connection
func (m *Metrics) RecordConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.TotalConnections++
	m.ActiveConnections++
	
	if m.ActiveConnections > m.MaxConcurrent {
		m.MaxConcurrent = m.ActiveConnections
	}
}

// RecordDisconnection records a disconnection
func (m *Metrics) RecordDisconnection(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ActiveConnections--
	
	// Update average connection time
	m.connectionTimes = append(m.connectionTimes, duration)
	if len(m.connectionTimes) > 100 {
		m.connectionTimes = m.connectionTimes[1:] // Keep last 100
	}
	
	var total time.Duration
	for _, d := range m.connectionTimes {
		total += d
	}
	if len(m.connectionTimes) > 0 {
		m.AvgConnectionTime = total / time.Duration(len(m.connectionTimes))
	}
}

// RecordMessage records message metrics
func (m *Metrics) RecordMessage(direction string, size int64, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	switch direction {
	case "sent":
		m.MessagesSent++
		m.BytesSent += size
	case "received":
		m.MessagesReceived++
		m.BytesReceived += size
	}
	
	if !success {
		m.MessageErrors++
	}
	
	// Update average message size
	totalMessages := m.MessagesSent + m.MessagesReceived
	if totalMessages > 0 {
		m.AvgMessageSize = (m.BytesSent + m.BytesReceived) / totalMessages
	}
}

// RecordError records an error by type
func (m *Metrics) RecordError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ErrorsByType[errorType]++
}

// RecordQueueDepth records the current queue depth
func (m *Metrics) RecordQueueDepth(depth int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if depth > m.MaxQueueDepth {
		m.MaxQueueDepth = depth
	}
	
	// Simple moving average for queue depth
	if m.AvgQueueDepth == 0 {
		m.AvgQueueDepth = depth
	} else {
		m.AvgQueueDepth = (m.AvgQueueDepth*9 + depth) / 10
	}
}

// RecordDroppedMessage records a dropped message
func (m *Metrics) RecordDroppedMessage() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.DroppedMessages++
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	errorCounts := make(map[string]int64)
	for k, v := range m.ErrorsByType {
		errorCounts[k] = v
	}
	
	return map[string]interface{}{
		"connections": map[string]interface{}{
			"total":              m.TotalConnections,
			"active":             m.ActiveConnections,
			"failed":             m.FailedConnections,
			"max_concurrent":     m.MaxConcurrent,
			"avg_duration_ms":    m.AvgConnectionTime.Milliseconds(),
		},
		"messages": map[string]interface{}{
			"sent":               m.MessagesSent,
			"received":           m.MessagesReceived,
			"bytes_sent":         m.BytesSent,
			"bytes_received":     m.BytesReceived,
			"errors":             m.MessageErrors,
			"avg_size":           m.AvgMessageSize,
			"dropped":            m.DroppedMessages,
		},
		"performance": map[string]interface{}{
			"avg_queue_depth":    m.AvgQueueDepth,
			"max_queue_depth":    m.MaxQueueDepth,
		},
		"errors": errorCounts,
		"uptime_seconds": time.Since(m.LastReset).Seconds(),
	}
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.TotalConnections = 0
	m.ActiveConnections = 0
	m.FailedConnections = 0
	m.MaxConcurrent = 0
	m.AvgConnectionTime = 0
	m.MessagesSent = 0
	m.MessagesReceived = 0
	m.BytesSent = 0
	m.BytesReceived = 0
	m.MessageErrors = 0
	m.AvgMessageSize = 0
	m.AvgQueueDepth = 0
	m.MaxQueueDepth = 0
	m.DroppedMessages = 0
	m.ErrorsByType = make(map[string]int64)
	m.LastReset = time.Now()
	m.connectionTimes = make([]time.Duration, 0, 100)
}

// Global metrics instance
var globalMetrics = NewMetrics()

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	return globalMetrics
}