package websocket

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOperationsEventIntegration tests that operations events are properly broadcast through WebSocket
func TestOperationsEventIntegration(t *testing.T) {
	logger := slog.Default()
	hub := NewHub(logger)
	
	// Start the hub
	hub.Start()
	defer hub.Stop()
	
	// Create a test client to receive messages
	testClient := &Client{
		id:          "test-client",
		send:        make(chan []byte, 256),
		traceID:     "test-trace-123",
		connectedAt: time.Now(),
		remoteAddr:  "test-addr",
	}
	
	// Register the client
	hub.Register(testClient)
	
	// Wait for registration to complete
	time.Sleep(100 * time.Millisecond)
	
	// Clear the connection message
	select {
	case <-testClient.send:
		// Connection message received and discarded
	case <-time.After(1 * time.Second):
		t.Fatal("Expected connection message")
	}
	
	tests := []struct {
		name           string
		eventType      string
		stepID         string
		status         string
		metadata       interface{}
		expectedType   string
		validateFunc   func(t *testing.T, msg map[string]interface{})
	}{
		{
			name:      "operation reset event",
			eventType: "operation:reset",
			stepID:    "op-123",
			status:    "initialized",
			metadata:  nil,
			expectedType: "operation:reset",
			validateFunc: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, "op-123", msg["operation_id"])
				assert.Equal(t, "initialized", msg["status"])
			},
		},
		{
			name:      "operation started event",
			eventType: "operation:started",
			stepID:    "op-456",
			status:    "running",
			metadata: map[string]interface{}{
				"operation_type": "full_pipeline",
				"steps_total":    5,
			},
			expectedType: "operation:start",
			validateFunc: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, "op-456", msg["operation_id"])
				assert.Equal(t, "running", msg["status"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "full_pipeline", data["operation_type"])
				assert.Equal(t, float64(5), data["steps_total"])
			},
		},
		{
			name:      "operation progress event",
			eventType: "operation:progress",
			stepID:    "step-789",
			status:    "active",
			metadata: map[string]interface{}{
				"operation_id": "op-456",
				"progress":     float64(50),
				"message":      "Processing data",
			},
			expectedType: "operation:progress",
			validateFunc: func(t *testing.T, msg map[string]interface{}) {
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "step-789", data["step_id"])
				assert.Equal(t, "active", data["status"])
				assert.Equal(t, "op-456", data["operation_id"])
				assert.Equal(t, float64(50), data["progress"])
				assert.Equal(t, "Processing data", data["message"])
			},
		},
		{
			name:      "operation completed event",
			eventType: "operation:completed",
			stepID:    "op-456",
			status:    "completed",
			metadata: map[string]interface{}{
				"duration": "5m30s",
				"results": map[string]interface{}{
					"processed": 100,
					"failed":    0,
				},
			},
			expectedType: "operation:complete",
			validateFunc: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, "op-456", msg["operation_id"])
				assert.Equal(t, "completed", msg["status"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "5m30s", data["duration"])
				results := data["results"].(map[string]interface{})
				assert.Equal(t, float64(100), results["processed"])
			},
		},
		{
			name:      "operation failed event",
			eventType: "operation:error",
			stepID:    "op-fail",
			status:    "failed",
			metadata: map[string]interface{}{
				"error":        "Connection timeout",
			},
			expectedType: "operation:failed",
			validateFunc: func(t *testing.T, msg map[string]interface{}) {
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "op-fail", data["operation_id"])
				assert.Equal(t, "failed", data["status"])
				assert.Equal(t, "Connection timeout", data["error"])
			},
		},
		{
			name:      "operation cancelled event",
			eventType: "operation:cancelled",
			stepID:    "op-cancel",
			status:    "cancelled",
			metadata:  nil,
			expectedType: "operation:cancelled",
			validateFunc: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, "op-cancel", msg["operation_id"])
				assert.Equal(t, "cancelled", msg["status"])
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send the event
			hub.BroadcastUpdate(tt.eventType, tt.stepID, tt.status, tt.metadata)
			
			// Wait for the message
			select {
			case msgBytes := <-testClient.send:
				var msg map[string]interface{}
				err := json.Unmarshal(msgBytes, &msg)
				require.NoError(t, err)
				
				// Verify message type
				assert.Equal(t, tt.expectedType, msg["type"])
				
				// Verify timestamp exists
				assert.NotEmpty(t, msg["timestamp"])
				
				// Run custom validation
				tt.validateFunc(t, msg)
				
			case <-time.After(1 * time.Second):
				t.Fatalf("Expected message for %s", tt.name)
			}
		})
	}
}

// TestOperationsAdapterIntegration tests the WebSocketOperationsAdapter
func TestOperationsAdapterIntegration(t *testing.T) {
	logger := slog.Default()
	hub := NewHub(logger)
	
	// Start the hub
	hub.Start()
	defer hub.Stop()
	
	// Create adapter (simulating what operations service does)
	adapter := &operationsWebSocketAdapter{hub: hub}
	
	// Create a test client
	testClient := &Client{
		id:          "adapter-test-client",
		send:        make(chan []byte, 256),
		traceID:     "adapter-trace-123",
		connectedAt: time.Now(),
		remoteAddr:  "test-addr",
	}
	
	hub.Register(testClient)
	time.Sleep(100 * time.Millisecond)
	
	// Clear connection message
	<-testClient.send
	
	// Test adapter broadcasts
	testCases := []struct {
		name         string
		eventType    string
		stepID       string
		status       string
		metadata     interface{}
		expectedType string
	}{
		{
			name:         "adapter operation start",
			eventType:    "operation:started",
			stepID:       "op-adapter-1",
			status:       "running",
			metadata:     nil,
			expectedType: "operation:start",
		},
		{
			name:      "adapter step progress",
			eventType: "operation:progress",
			stepID:    "step-adapter-1",
			status:    "active",
			metadata: map[string]interface{}{
				"operation_id": "op-adapter-1",
				"progress":     75.5,
			},
			expectedType: "operation:progress",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use adapter to broadcast
			adapter.BroadcastUpdate(tc.eventType, tc.stepID, tc.status, tc.metadata)
			
			// Verify message received
			select {
			case msgBytes := <-testClient.send:
				var msg map[string]interface{}
				err := json.Unmarshal(msgBytes, &msg)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedType, msg["type"])
			case <-time.After(1 * time.Second):
				t.Fatal("Expected message from adapter")
			}
		})
	}
}

// operationsWebSocketAdapter is a test adapter that mimics WebSocketOperationsAdapter
type operationsWebSocketAdapter struct {
	hub *Hub
}

func (a *operationsWebSocketAdapter) BroadcastUpdate(eventType, stepID, status string, metadata interface{}) {
	a.hub.BroadcastUpdate(eventType, stepID, status, metadata)
}