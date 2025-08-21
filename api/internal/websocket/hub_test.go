package websocket

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHub tests hub creation
func TestNewHub(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.logger)
	assert.NotNil(t, hub.quit)
	assert.NotNil(t, hub.metricsQuit)
	assert.Equal(t, 0, len(hub.clients))
	assert.False(t, hub.running)
}

// TestHubStartStop tests starting and stopping the hub
func TestHubStartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)

	// Start the hub
	hub.Start()
	assert.True(t, hub.running)

	// Starting again should be idempotent
	hub.Start()
	assert.True(t, hub.running)

	// Wait a bit to ensure goroutines are running
	time.Sleep(10 * time.Millisecond)

	// Stop the hub
	hub.Stop()
	assert.False(t, hub.running)

	// Stopping again should be idempotent
	hub.Stop()
	assert.False(t, hub.running)
}

// TestHubClientRegistration tests client registration and unregistration
func TestHubClientRegistration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create a test client
	client := &Client{
		id:          "test-client-1",
		hub:         hub,
		send:        make(chan []byte, 256),
		traceID:     "test-trace-1",
		connectedAt: time.Now(),
		remoteAddr:  "127.0.0.1:8080",
	}

	// Register the client
	hub.Register(client)

	// Wait for registration to complete
	time.Sleep(50 * time.Millisecond)

	// Check client count
	assert.Equal(t, 1, hub.ClientCount())

	// Client should receive connection message
	select {
	case msg := <-client.send:
		var connMsg map[string]interface{}
		err := json.Unmarshal(msg, &connMsg)
		require.NoError(t, err)
		assert.Equal(t, TypeConnection, connMsg["type"])
		data := connMsg["data"].(map[string]interface{})
		assert.Equal(t, "connected", data["status"])
		assert.Equal(t, "test-client-1", data["client_id"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for connection message")
	}

	// Unregister the client
	hub.unregister <- client

	// Wait for unregistration to complete
	time.Sleep(50 * time.Millisecond)

	// Check client count
	assert.Equal(t, 0, hub.ClientCount())
}

// TestHubBroadcast tests message broadcasting to multiple clients
func TestHubBroadcast(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create multiple test clients
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &Client{
			id:          fmt.Sprintf("test-client-%d", i),
			hub:         hub,
			send:        make(chan []byte, 256),
			connectedAt: time.Now(),
			remoteAddr:  fmt.Sprintf("127.0.0.1:808%d", i),
		}
		hub.Register(clients[i])
	}

	// Wait for registrations to complete
	time.Sleep(100 * time.Millisecond)

	// Clear connection messages
	for _, client := range clients {
		<-client.send
	}

	// Broadcast a message
	testMsg := map[string]interface{}{
		"type": "test",
		"data": "broadcast test",
	}
	jsonData, _ := json.Marshal(testMsg)
	hub.broadcast <- jsonData

	// All clients should receive the message
	var wg sync.WaitGroup
	wg.Add(3)
	for i, client := range clients {
		go func(idx int, c *Client) {
			defer wg.Done()
			select {
			case msg := <-c.send:
				assert.Equal(t, jsonData, msg)
			case <-time.After(1 * time.Second):
				t.Errorf("client %d: timeout waiting for broadcast", idx)
			}
		}(i, client)
	}
	wg.Wait()
}

// TestHubBroadcastMethods tests various broadcast helper methods
func TestHubBroadcastMethods(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create a test client
	client := &Client{
		id:          "test-client",
		hub:         hub,
		send:        make(chan []byte, 256),
		connectedAt: time.Now(),
		remoteAddr:  "127.0.0.1:8080",
	}
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)
	<-client.send // Clear connection message

	tests := []struct {
		name        string
		broadcast   func()
		checkMsg    func(t *testing.T, msg map[string]interface{})
	}{
		{
			name: "BroadcastOutput",
			broadcast: func() {
				hub.BroadcastOutput("Test output message", LevelInfo)
			},
			checkMsg: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, TypeOutput, msg["type"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "Test output message", data["message"])
				assert.Equal(t, LevelInfo, data["level"])
			},
		},
		{
			name: "BroadcastProgress",
			broadcast: func() {
				hub.BroadcastProgress("processing", 50, "Processing data")
			},
			checkMsg: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, TypeProgress, msg["type"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "processing", data["step"])
				assert.Equal(t, float64(50), data["progress"])
				assert.Equal(t, "Processing data", data["message"])
			},
		},
		{
			name: "BroadcastStatus",
			broadcast: func() {
				hub.BroadcastStatus("active", "System is active")
			},
			checkMsg: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, "status", msg["type"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "active", data["status"])
				assert.Equal(t, "System is active", data["message"])
			},
		},
		{
			name: "BroadcastError",
			broadcast: func() {
				hub.BroadcastError("ERR_1001", "Connection timeout", "Failed to connect", "connection", true)
			},
			checkMsg: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, TypeError, msg["type"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "ERR_1001", data["code"])
				assert.Equal(t, "Connection timeout", data["message"])
				assert.Equal(t, "Failed to connect", data["details"])
				assert.Equal(t, "connection", data["step"])
				assert.Equal(t, true, data["recoverable"])
				assert.NotEmpty(t, data["hint"])
			},
		},
		{
			name: "BroadcastRefresh",
			broadcast: func() {
				hub.BroadcastRefresh("test-source", []string{"component1", "component2"})
			},
			checkMsg: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, TypeDataUpdate, msg["type"])
				assert.Equal(t, SubtypeAll, msg["subtype"])
				assert.Equal(t, ActionRefresh, msg["action"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "test-source", data["source"])
				components := data["components"].([]interface{})
				assert.Equal(t, 2, len(components))
			},
		},
		{
			name: "BroadcastConnection",
			broadcast: func() {
				hub.BroadcastConnection("connected", map[string]interface{}{"licensed": true})
			},
			checkMsg: func(t *testing.T, msg map[string]interface{}) {
				assert.Equal(t, TypeConnection, msg["type"])
				data := msg["data"].(map[string]interface{})
				assert.Equal(t, "connected", data["status"])
				license := data["license"].(map[string]interface{})
				assert.Equal(t, true, license["licensed"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Broadcast the message
			tt.broadcast()

			// Check the received message
			select {
			case msgBytes := <-client.send:
				var msg map[string]interface{}
				err := json.Unmarshal(msgBytes, &msg)
				require.NoError(t, err)
				tt.checkMsg(t, msg)
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for broadcast message")
			}
		})
	}
}

// TestHubOperationEvents tests operation-specific event broadcasting
func TestHubOperationEvents(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create a test client
	client := &Client{
		id:          "test-client",
		hub:         hub,
		send:        make(chan []byte, 256),
		connectedAt: time.Now(),
		remoteAddr:  "127.0.0.1:8080",
	}
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)
	<-client.send // Clear connection message

	tests := []struct {
		name      string
		eventType string
		subtype   string
		action    string
		data      interface{}
		checkType string
	}{
		{
			name:      "operation:started",
			eventType: "operation:started",
			subtype:   "op-123",
			action:    "running",
			data:      map[string]interface{}{"step": "init"},
			checkType: "operation:start",
		},
		{
			name:      "operation:progress",
			eventType: "operation:progress",
			subtype:   "step-456",
			action:    "processing",
			data:      map[string]interface{}{"progress": 75},
			checkType: "operation:progress",
		},
		{
			name:      "operation:completed",
			eventType: "operation:completed",
			subtype:   "op-789",
			action:    "success",
			data:      map[string]interface{}{"duration": "5s"},
			checkType: "operation:complete",
		},
		{
			name:      "operation:failed",
			eventType: "operation:failed",
			subtype:   "op-999",
			action:    "error",
			data:      map[string]interface{}{"error": "timeout"},
			checkType: "operation:failed",
		},
		{
			name:      "operation:cancelled",
			eventType: "operation:cancelled",
			subtype:   "op-111",
			action:    "cancelled",
			data:      nil,
			checkType: "operation:cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Broadcast the operation event
			hub.BroadcastUpdate(tt.eventType, tt.subtype, tt.action, tt.data)

			// Check the received message
			select {
			case msgBytes := <-client.send:
				var msg map[string]interface{}
				err := json.Unmarshal(msgBytes, &msg)
				require.NoError(t, err)
				assert.Equal(t, tt.checkType, msg["type"])
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for operation event")
			}
		})
	}
}

// TestHubMetrics tests hub metrics collection
func TestHubMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create and register clients
	for i := 0; i < 2; i++ {
		client := &Client{
			id:          fmt.Sprintf("client-%d", i),
			hub:         hub,
			send:        make(chan []byte, 256),
			connectedAt: time.Now(),
			remoteAddr:  fmt.Sprintf("127.0.0.1:808%d", i),
		}
		hub.Register(client)
	}

	// Wait for registrations
	time.Sleep(100 * time.Millisecond)

	// Send some messages
	for i := 0; i < 5; i++ {
		hub.broadcast <- []byte(fmt.Sprintf("test message %d", i))
	}

	// Wait for messages to be processed
	time.Sleep(100 * time.Millisecond)

	// Get metrics
	metrics := hub.GetHubMetrics()

	assert.Equal(t, 2, metrics["active_clients"])
	assert.Equal(t, int64(2), metrics["total_connections"])
	assert.True(t, metrics["messages_sent"].(int64) > 0)
}

// TestHubClientDisconnectOnFullBuffer tests that clients are disconnected when their buffer is full
func TestHubClientDisconnectOnFullBuffer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create a client with a very small buffer
	client := &Client{
		id:          "test-client",
		hub:         hub,
		send:        make(chan []byte, 1), // Very small buffer
		connectedAt: time.Now(),
		remoteAddr:  "127.0.0.1:8080",
	}
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	// Initial client count
	assert.Equal(t, 1, hub.ClientCount())

	// Send multiple messages to overflow the buffer
	for i := 0; i < 10; i++ {
		hub.broadcast <- []byte(fmt.Sprintf("message %d", i))
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Client should be disconnected due to full buffer
	assert.Equal(t, 0, hub.ClientCount())
}

// TestHubConcurrentAccess tests concurrent access to hub
func TestHubConcurrentAccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	var wg sync.WaitGroup
	clientCount := 10
	messageCount := 5

	// Concurrently register clients
	wg.Add(clientCount)
	for i := 0; i < clientCount; i++ {
		go func(idx int) {
			defer wg.Done()
			client := &Client{
				id:          fmt.Sprintf("client-%d", idx),
				hub:         hub,
				send:        make(chan []byte, 256),
				connectedAt: time.Now(),
				remoteAddr:  fmt.Sprintf("127.0.0.1:80%02d", idx),
			}
			hub.Register(client)
		}(i)
	}
	wg.Wait()

	// Wait for registrations
	time.Sleep(100 * time.Millisecond)

	// Check client count
	assert.Equal(t, clientCount, hub.ClientCount())

	// Concurrently broadcast messages
	wg.Add(messageCount)
	for i := 0; i < messageCount; i++ {
		go func(idx int) {
			defer wg.Done()
			hub.BroadcastOutput(fmt.Sprintf("Concurrent message %d", idx), LevelInfo)
		}(i)
	}
	wg.Wait()

	// Concurrently get metrics
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			_ = hub.GetHubMetrics()
			_ = hub.ClientCount()
		}()
	}
	wg.Wait()
}

// TestHubWithNilLogger tests hub creation with nil logger
func TestHubWithNilLogger(t *testing.T) {
	hub := NewHub(nil)
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.logger)
}

// TestHubBroadcastWithTrace tests broadcasting with trace IDs
func TestHubBroadcastWithTrace(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create a test client
	client := &Client{
		id:          "test-client",
		hub:         hub,
		send:        make(chan []byte, 256),
		connectedAt: time.Now(),
		remoteAddr:  "127.0.0.1:8080",
	}
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)
	<-client.send // Clear connection message

	// Test BroadcastUpdateWithTrace
	hub.BroadcastUpdateWithTrace("test_type", "test_sub", "test_action", map[string]interface{}{"key": "value"}, "trace-123")

	select {
	case msgBytes := <-client.send:
		var msg map[string]interface{}
		err := json.Unmarshal(msgBytes, &msg)
		require.NoError(t, err)
		assert.Equal(t, "trace-123", msg["trace_id"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message with trace")
	}

	// Test BroadcastStatusWithTrace
	hub.BroadcastStatusWithTrace("active", "System active", "trace-456")

	select {
	case msgBytes := <-client.send:
		var msg map[string]interface{}
		err := json.Unmarshal(msgBytes, &msg)
		require.NoError(t, err)
		assert.Equal(t, "trace-456", msg["trace_id"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for status message with trace")
	}
}

// BenchmarkHubBroadcast benchmarks message broadcasting
func BenchmarkHubBroadcast(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create multiple clients
	clientCount := 100
	for i := 0; i < clientCount; i++ {
		client := &Client{
			id:          fmt.Sprintf("bench-client-%d", i),
			hub:         hub,
			send:        make(chan []byte, 256),
			connectedAt: time.Now(),
			remoteAddr:  fmt.Sprintf("127.0.0.1:8%03d", i),
		}
		hub.Register(client)
	}

	// Wait for registrations
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastOutput(fmt.Sprintf("Benchmark message %d", i), LevelInfo)
	}
}

// BenchmarkHubConcurrentBroadcast benchmarks concurrent broadcasting
func BenchmarkHubConcurrentBroadcast(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	hub := NewHub(logger)
	hub.Start()
	defer hub.Stop()

	// Create multiple clients
	clientCount := 50
	for i := 0; i < clientCount; i++ {
		client := &Client{
			id:          fmt.Sprintf("bench-client-%d", i),
			hub:         hub,
			send:        make(chan []byte, 256),
			connectedAt: time.Now(),
			remoteAddr:  fmt.Sprintf("127.0.0.1:8%03d", i),
		}
		hub.Register(client)
	}

	// Wait for registrations
	time.Sleep(100 * time.Millisecond)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			hub.BroadcastOutput(fmt.Sprintf("Concurrent benchmark message %d", i), LevelInfo)
			i++
		}
	})
}

