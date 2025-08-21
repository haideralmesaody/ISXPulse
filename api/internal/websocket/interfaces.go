package websocket

import (
	"time"
)

// Connection defines the interface for WebSocket connections
// This allows for proper mocking in tests
type Connection interface {
	// WriteMessage writes a message with the given message type and payload
	WriteMessage(messageType int, data []byte) error
	
	// ReadMessage reads a message from the connection
	// Returns the message type and payload
	ReadMessage() (messageType int, p []byte, err error)
	
	// Close closes the connection
	Close() error
	
	// SetReadDeadline sets the read deadline on the connection
	SetReadDeadline(t time.Time) error
	
	// SetWriteDeadline sets the write deadline on the connection
	SetWriteDeadline(t time.Time) error
	
	// SetReadLimit sets the maximum size for a message read from the connection
	SetReadLimit(limit int64)
	
	// SetPongHandler sets the handler for pong messages
	SetPongHandler(h func(string) error)
	
	// SetPingHandler sets the handler for ping messages
	SetPingHandler(h func(string) error)
	
	// SetCloseHandler sets the handler for close messages
	SetCloseHandler(h func(code int, text string) error)
	
	// RemoteAddr returns the remote network address
	RemoteAddr() string
}

// HubInterface defines the interface for WebSocket hub
// This allows components to depend on an interface rather than concrete type
type HubInterface interface {
	// Register registers a new client
	Register(client *Client)
	
	// Unregister unregisters a client
	Unregister(client *Client)
	
	// Broadcast sends a message to all connected clients
	Broadcast(message []byte)
	
	// BroadcastToClient sends a message to a specific client
	BroadcastToClient(clientID string, message []byte) error
	
	// GetClientCount returns the number of connected clients
	GetClientCount() int
	
	// Start starts the hub's goroutines
	Start()
	
	// Stop stops the hub gracefully
	Stop()
	
	// Run runs the hub's main loop (called by Start)
	Run()
}

// MessageAdapterInterface defines the interface for message adapters
type MessageAdapterInterface interface {
	// BroadcastUpdate broadcasts an update message
	BroadcastUpdate(msgType, step, operationID string, data interface{})
	
	// SendProgress sends a progress update
	SendProgress(step string, progress int, message string)
	
	// SendComplete sends a completion message
	SendComplete(step string, message string, success bool)
	
	// SendError sends an error message
	SendError(step string, err error)
	
	// SendOutput sends output data
	SendOutput(level, message, step string)
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// RecordConnection records a new connection
	RecordConnection()
	
	// RecordDisconnection records a disconnection
	RecordDisconnection(duration time.Duration)
	
	// RecordMessage records message metrics
	RecordMessage(direction string, size int64, success bool)
	
	// RecordError records an error
	RecordError(errorType string)
	
	// RecordQueueDepth records the queue depth
	RecordQueueDepth(depth int64)
	
	// RecordDroppedMessage records a dropped message
	RecordDroppedMessage()
	
	// GetSnapshot returns a snapshot of current metrics
	GetSnapshot() map[string]interface{}
	
	// Reset resets all metrics
	Reset()
}