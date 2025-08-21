package websocket

import (
	"bytes"
	"context"
	"log/slog"
	"time"

	"isxcli/internal/infrastructure"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub *Hub

	// The websocket connection
	conn Connection

	// Buffered channel of outbound messages
	send chan []byte
	
	// Client metadata
	id           string
	traceID      string
	remoteAddr   string
	connectedAt  time.Time
	
	// Logger
	logger      *slog.Logger
	
	// Metrics
	messagesSent     int64
	messagesReceived int64
	bytessSent       int64
	bytesReceived    int64
}

// NewClient creates a new Client with dependency injection
func NewClient(hub *Hub, conn *websocket.Conn, logger *slog.Logger) *Client {
	if logger == nil {
		logger = infrastructure.GetLogger()
	}
	
	id := uuid.New().String()
	logger = logger.With(
		slog.String("component", "websocket.client"),
		slog.String("client_id", id),
	)
	
	// Wrap the websocket.Conn in our interface
	wrappedConn := NewConnectionWrapper(conn)
	
	return &Client{
		hub:         hub,
		conn:        wrappedConn,
		send:        make(chan []byte, 256),
		id:          id,
		remoteAddr:  wrappedConn.RemoteAddr(),
		connectedAt: time.Now(),
		logger:      logger,
	}
}

// NewClientWithConnection creates a new Client with a custom connection (for testing)
func NewClientWithConnection(hub *Hub, conn Connection, logger *slog.Logger) *Client {
	if logger == nil {
		logger = infrastructure.GetLogger()
	}
	
	id := uuid.New().String()
	logger = logger.With(
		slog.String("component", "websocket.client"),
		slog.String("client_id", id),
	)
	
	return &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		id:          id,
		remoteAddr:  conn.RemoteAddr(),
		connectedAt: time.Now(),
		logger:      logger,
	}
}

// NewClientWithTrace creates a new Client with trace ID and dependency injection
func NewClientWithTrace(hub *Hub, conn *websocket.Conn, traceID string, logger *slog.Logger) *Client {
	client := NewClient(hub, conn, logger)
	client.traceID = traceID
	client.logger = client.logger.With(slog.String("trace_id", traceID))
	return client
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		ctx := context.Background()
		if c.traceID != "" {
			ctx = infrastructure.WithTraceID(ctx, c.traceID)
		}
		c.logger.InfoContext(ctx, "WebSocket client disconnected (readPump)",
			slog.Duration("connection_duration", time.Since(c.connectedAt)),
			slog.Int64("messages_received", c.messagesReceived),
			slog.Int64("bytes_received", c.bytesReceived))
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ctx := context.Background()
				if c.traceID != "" {
					ctx = infrastructure.WithTraceID(ctx, c.traceID)
				}
				c.logger.ErrorContext(ctx, "Unexpected WebSocket close error",
					slog.String("error", err.Error()))
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		
		c.messagesReceived++
		c.bytesReceived += int64(len(message))
		
		// Record OpenTelemetry metrics for received message
		if otelMetrics := GetOTelMetrics(); otelMetrics != nil {
			ctx := context.Background()
			if c.traceID != "" {
				ctx = infrastructure.WithTraceID(ctx, c.traceID)
			}
			otelMetrics.RecordMessageReceived(ctx, "client_message", c.id, int64(len(message)))
		}
		
		// Handle heartbeat messages from JavaScript client
		if string(message) == `{"type":"heartbeat"}` {
			// Heartbeat received, connection is alive
			// The pong handler already updates the read deadline
			c.logger.Debug("Heartbeat received")
			continue
		}
		
		// For now, we don't process other incoming messages from clients
		// but this is where we would handle client commands if needed
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		
		ctx := context.Background()
		if c.traceID != "" {
			ctx = infrastructure.WithTraceID(ctx, c.traceID)
		}
		c.logger.InfoContext(ctx, "WebSocket write pump stopped",
			slog.Int64("messages_sent", c.messagesSent),
			slog.Int64("bytes_sent", c.bytessSent))
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message as a complete WebSocket frame
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				ctx := context.Background()
				if c.traceID != "" {
					ctx = infrastructure.WithTraceID(ctx, c.traceID)
				}
				c.logger.ErrorContext(ctx, "Error writing message to WebSocket",
					slog.String("error", err.Error()))
				return
			}
			
			c.messagesSent++
			c.bytessSent += int64(len(message))
			
			// Record OpenTelemetry metrics for sent message
			if otelMetrics := GetOTelMetrics(); otelMetrics != nil {
				ctx := context.Background()
				if c.traceID != "" {
					ctx = infrastructure.WithTraceID(ctx, c.traceID)
				}
				otelMetrics.RecordMessageSent(ctx, "server_message", c.id, int64(len(message)))
			}

			// Send any queued messages as separate WebSocket frames
			n := len(c.send)
			for i := 0; i < n; i++ {
				select {
				case msg := <-c.send:
					c.conn.SetWriteDeadline(time.Now().Add(writeWait))
					if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
						ctx := context.Background()
						if c.traceID != "" {
							ctx = infrastructure.WithTraceID(ctx, c.traceID)
						}
						c.logger.ErrorContext(ctx, "Error writing queued message to WebSocket",
							slog.String("error", err.Error()))
						return
					}
					c.messagesSent++
					c.bytessSent += int64(len(msg))
					
					// Record OpenTelemetry metrics for queued message
					if otelMetrics := GetOTelMetrics(); otelMetrics != nil {
						ctx := context.Background()
						if c.traceID != "" {
							ctx = infrastructure.WithTraceID(ctx, c.traceID)
						}
						otelMetrics.RecordMessageSent(ctx, "server_message_queued", c.id, int64(len(msg)))
					}
				default:
					// Channel was empty
				}
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				ctx := context.Background()
				if c.traceID != "" {
					ctx = infrastructure.WithTraceID(ctx, c.traceID)
				}
				c.logger.DebugContext(ctx, "Failed to send ping message",
					slog.String("error", err.Error()))
				return
			}
		}
	}
}

// ServeWS handles websocket requests from the peer
func ServeWS(hub *Hub, conn *websocket.Conn) {
	client := NewClient(hub, conn, nil) // nil logger will be replaced with default in NewClient
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines
	go client.WritePump()
	go client.ReadPump()
}