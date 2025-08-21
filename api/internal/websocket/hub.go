package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"isxcli/internal/infrastructure"
)

// Legacy message type constants for backward compatibility
const (
	TypeConnection       = "connection"
	TypeProgress         = "progress"
	TypeOutput           = "output"
	TypeError            = "error"
	TypeDataUpdate       = "data_update"
	TypeOperationStatus  = "operation:status"
	TypePipelineProgress = "operation:progress"
	TypePipelineComplete = "operation:complete"
	TypeLog              = "log"
	SubtypeAll           = "all"
	ActionRefresh        = "refresh"
	
	// Message levels
	LevelInfo    = "info"
	LevelSuccess = "success"
	LevelWarning = "warning"
	LevelError   = "error"
)

// ErrorRecoveryHints provides user-friendly recovery suggestions
var ErrorRecoveryHints = map[string]string{
	"default": "Please try again or contact support",
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Logger instance
	logger *slog.Logger

	// Metrics
	totalConnections int64
	activConnections int64
	messagesSent     int64
	messagesReceived int64
	connectionErrors int64

	// Control
	quit        chan struct{}
	running     bool
	metricsQuit chan struct{}
}

// NewHub creates a new Hub instance with dependency injection
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = infrastructure.GetLogger()
	}
	logger = logger.With(slog.String("component", "websocket.hub"))

	hub := &Hub{
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		logger:      logger,
		quit:        make(chan struct{}),
		metricsQuit: make(chan struct{}),
	}

	return hub
}

// Start starts the hub's goroutines
func (h *Hub) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	// Start the main hub loop
	go h.Run()

	// Start metrics reporting
	go h.reportMetrics()
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case <-h.quit:
			h.logger.Info("Hub shutting down")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			count := len(h.clients)
			h.totalConnections++
			h.activConnections = int64(count)
			h.mu.Unlock()

			ctx := context.Background()
			if client.traceID != "" {
				ctx = infrastructure.WithTraceID(ctx, client.traceID)
			}

			h.logger.InfoContext(ctx, "Client registered",
				slog.Int("total_clients", count),
				slog.String("client_id", client.id),
				slog.String("remote_addr", client.remoteAddr))

			// Update metrics
			metrics := GetMetrics()
			metrics.RecordConnection()

			// Record OpenTelemetry metrics
			if otelMetrics := GetOTelMetrics(); otelMetrics != nil {
				otelMetrics.RecordConnection(ctx, client.id, client.remoteAddr)
				otelMetrics.RecordClientCount(ctx, int64(count))
			}

			// Send connection success message to the newly connected client
			connMsg := map[string]interface{}{
				"type": TypeConnection,
				"data": map[string]interface{}{
					"status":    "connected",
					"message":   "Connected to ISX WebSocket",
					"client_id": client.id,
				},
				"timestamp": time.Now().Format(time.RFC3339),
				"trace_id":  client.traceID,
			}

			jsonData, err := json.Marshal(connMsg)
			if err == nil {
				select {
				case client.send <- jsonData:
					h.logger.DebugContext(ctx, "Sent connection message to client",
						slog.String("client_id", client.id))
				default:
					h.logger.WarnContext(ctx, "Failed to send connection message - client buffer full",
						slog.String("client_id", client.id))
				}
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				count := len(h.clients)
				h.activConnections = int64(count)
				h.mu.Unlock()

				ctx := context.Background()
				if client.traceID != "" {
					ctx = infrastructure.WithTraceID(ctx, client.traceID)
				}

				h.logger.InfoContext(ctx, "Client unregistered",
					slog.Int("total_clients", count),
					slog.String("client_id", client.id),
					slog.Duration("connection_duration", time.Since(client.connectedAt)))

				// Update metrics
				metrics := GetMetrics()
				metrics.RecordDisconnection(time.Since(client.connectedAt))

				// Record OpenTelemetry metrics
				if otelMetrics := GetOTelMetrics(); otelMetrics != nil {
					otelMetrics.RecordDisconnection(ctx, client.id, time.Since(client.connectedAt), "normal")
					otelMetrics.RecordClientCount(ctx, int64(count))
				}
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			// Create a copy of clients to avoid holding lock during send
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			h.logger.Debug("Broadcasting message to clients",
				slog.Int("client_count", len(clients)),
				slog.Int("message_size", len(message)))

			successCount := 0
			failCount := 0

			// Send to all clients
			for _, client := range clients {
				select {
				case client.send <- message:
					successCount++
					h.messagesSent++
				default:
					failCount++
					// Client's send channel is full, close it
					h.mu.Lock()
					close(client.send)
					delete(h.clients, client)
					h.mu.Unlock()

					ctx := context.Background()
					if client.traceID != "" {
						ctx = infrastructure.WithTraceID(ctx, client.traceID)
					}
					h.logger.WarnContext(ctx, "Client send buffer full, disconnecting",
						slog.String("client_id", client.id))
				}
			}

			// Log full JSON (truncated) for troubleshooting at info level
			if h.logger != nil {
				preview := 512
				if len(message) < preview {
					preview = len(message)
				}
				h.logger.Info("WS broadcast payload",
					slog.String("payload_preview", string(message[:preview])),
					slog.Int("payload_size", len(message)))
			}

			if failCount > 0 {
				h.logger.Warn("Some clients failed to receive broadcast",
					slog.Int("success_count", successCount),
					slog.Int("fail_count", failCount))
			}

			// Record OpenTelemetry metrics for broadcast
			if otelMetrics := GetOTelMetrics(); otelMetrics != nil {
				ctx := context.Background()
				otelMetrics.RecordBroadcast(ctx, "broadcast", int64(len(clients)), int64(successCount), int64(failCount))
			}
		}
	}
}

// BroadcastUpdate sends a data update message to all connected clients
func (h *Hub) BroadcastUpdate(updateType, subtype, action string, data interface{}) {
	h.BroadcastUpdateWithTrace(updateType, subtype, action, data, "")
}

// BroadcastUpdateWithTrace sends a data update message with trace ID to all connected clients
func (h *Hub) BroadcastUpdateWithTrace(updateType, subtype, action string, data interface{}, traceID string) {
	// Simplified: Only handle operation:snapshot and essential events
	message := map[string]interface{}{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Add trace ID if provided
	if traceID != "" {
		message["trace_id"] = traceID
	}

	// For operation:snapshot, the data already contains all necessary information
	// For other events, preserve backward compatibility with minimal processing
	if updateType != "operation:snapshot" && updateType != "" {
		// Legacy support for non-operation events
		message["subtype"] = subtype
		message["action"] = action
	}

	h.broadcastJSON(message)
}

// broadcastJSON is a helper method to send JSON messages
func (h *Hub) broadcastJSON(message map[string]interface{}) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		ctx := context.Background()
		if traceID, ok := message["trace_id"].(string); ok && traceID != "" {
			ctx = infrastructure.WithTraceID(ctx, traceID)
		}
		h.logger.ErrorContext(ctx, "Error marshaling message",
			slog.String("error", err.Error()),
			slog.String("message_type", message["type"].(string)))
		return
	}

	h.broadcast <- jsonData

	h.mu.Lock()
	h.messagesSent++
	h.mu.Unlock()
}

// BroadcastProgress sends a progress update message
func (h *Hub) BroadcastProgress(step string, progress int, message string) {
	update := map[string]interface{}{
		"type": TypeProgress,
		"data": map[string]interface{}{
			"step":     step,
			"progress": progress,
			"message":  message,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("Error marshaling progress message", slog.String("error", err.Error()))
		return
	}

	h.broadcast <- jsonData
}

// BroadcastProgressWithDetails sends a detailed progress update
func (h *Hub) BroadcastProgressWithDetails(step string, current, total int, percentage float64, message, eta string, details map[string]interface{}) {
	update := map[string]interface{}{
		"type": TypeProgress,
		"data": map[string]interface{}{
			"step":       step,
			"current":    current,
			"total":      total,
			"percentage": percentage,
			"message":    message,
			"eta":        eta,
			"details":    details,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("Error marshaling detailed progress message", slog.String("error", err.Error()))
		return
	}

	h.broadcast <- jsonData
}

// BroadcastStatus sends a status update message
func (h *Hub) BroadcastStatus(status, message string) {
	h.BroadcastStatusWithTrace(status, message, "")
}

// BroadcastStatusWithTrace sends a status update message with trace ID
func (h *Hub) BroadcastStatusWithTrace(status, message, traceID string) {
	update := map[string]interface{}{
		"type": "status",
		"data": map[string]interface{}{
			"status":  status,
			"message": message,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if traceID != "" {
		update["trace_id"] = traceID
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		ctx := context.Background()
		if traceID != "" {
			ctx = infrastructure.WithTraceID(ctx, traceID)
		}
		h.logger.ErrorContext(ctx, "Error marshaling status message",
			slog.String("error", err.Error()))
		return
	}

	h.broadcast <- jsonData
}

// BroadcastOutput sends an output message
func (h *Hub) BroadcastOutput(message, level string) {
	update := map[string]interface{}{
		"type": TypeOutput,
		"data": map[string]interface{}{
			"message": message,
			"level":   level,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("Error marshaling output message", slog.String("error", err.Error()))
		return
	}

	h.broadcast <- jsonData
}

// BroadcastConnection sends a connection status message
func (h *Hub) BroadcastConnection(status string, licenseInfo interface{}) {
	message := map[string]interface{}{
		"type": TypeConnection,
		"data": map[string]interface{}{
			"status":  status,
			"message": "Connected to ISX CLI Web Interface",
			"license": licenseInfo,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Error marshaling connection message", slog.String("error", err.Error()))
		return
	}

	h.broadcast <- jsonData
}

// BroadcastError sends a structured error message
func (h *Hub) BroadcastError(code, message, details, step string, recoverable bool) {
	hint := ErrorRecoveryHints[code]
	if hint == "" {
		hint = "Please try again or contact support"
	}

	errorMsg := map[string]interface{}{
		"type": TypeError,
		"data": map[string]interface{}{
			"code":        code,
			"message":     message,
			"details":     details,
			"step":        step,
			"recoverable": recoverable,
			"hint":        hint,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(errorMsg)
	if err != nil {
		h.logger.Error("Error marshaling error message", slog.String("error", err.Error()))
		return
	}

	h.broadcast <- jsonData
}

// BroadcastRefresh sends a data refresh notification (for UI updates)
func (h *Hub) BroadcastRefresh(source string, components []string) {
	h.BroadcastUpdate(TypeDataUpdate, SubtypeAll, ActionRefresh, map[string]interface{}{
		"source":     source,
		"components": components,
	})
}

// BroadcastJSON sends a pre-formatted JSON message directly
func (h *Hub) BroadcastJSON(message map[string]interface{}) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Error marshaling JSON message", slog.String("error", err.Error()))
		return
	}

	h.broadcast <- jsonData
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Broadcast implements the services.WebSocketHub interface
func (h *Hub) Broadcast(messageType string, data interface{}) {
	h.BroadcastUpdate(messageType, "", "", data)
}

// Stop gracefully stops the hub
func (h *Hub) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.mu.Unlock()

	// Signal goroutines to stop
	close(h.quit)
	close(h.metricsQuit)

	// Close all client connections
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// reportMetrics periodically reports hub metrics
func (h *Hub) reportMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.metricsQuit:
			h.logger.Info("Metrics reporting shutting down")
			return

		case <-ticker.C:
			h.mu.RLock()
			activeClients := len(h.clients)
			h.mu.RUnlock()

			metrics := GetMetrics()
			metrics.RecordQueueDepth(int64(len(h.broadcast)))

			// Log current metrics
			h.logger.Info("WebSocket hub metrics",
				slog.Int("active_clients", activeClients),
				slog.Int64("total_connections", h.totalConnections),
				slog.Int64("messages_sent", h.messagesSent),
				slog.Int64("messages_received", h.messagesReceived),
				slog.Int("broadcast_queue", len(h.broadcast)),
			)
		}
	}
}

// GetHubMetrics returns current hub metrics
func (h *Hub) GetHubMetrics() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"active_clients":    len(h.clients),
		"total_connections": h.totalConnections,
		"messages_sent":     h.messagesSent,
		"messages_received": h.messagesReceived,
		"connection_errors": h.connectionErrors,
	}
}
