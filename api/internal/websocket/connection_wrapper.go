package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionWrapper wraps a gorilla/websocket connection to implement our Connection interface
type ConnectionWrapper struct {
	conn *websocket.Conn
}

// NewConnectionWrapper creates a new connection wrapper
func NewConnectionWrapper(conn *websocket.Conn) Connection {
	return &ConnectionWrapper{conn: conn}
}

// WriteMessage writes a message with the given message type and payload
func (c *ConnectionWrapper) WriteMessage(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

// ReadMessage reads a message from the connection
func (c *ConnectionWrapper) ReadMessage() (messageType int, p []byte, err error) {
	return c.conn.ReadMessage()
}

// Close closes the connection
func (c *ConnectionWrapper) Close() error {
	return c.conn.Close()
}

// SetReadDeadline sets the read deadline on the connection
func (c *ConnectionWrapper) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the connection
func (c *ConnectionWrapper) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// SetReadLimit sets the maximum size for a message read from the connection
func (c *ConnectionWrapper) SetReadLimit(limit int64) {
	c.conn.SetReadLimit(limit)
}

// SetPongHandler sets the handler for pong messages
func (c *ConnectionWrapper) SetPongHandler(h func(string) error) {
	c.conn.SetPongHandler(h)
}

// SetPingHandler sets the handler for ping messages
func (c *ConnectionWrapper) SetPingHandler(h func(string) error) {
	c.conn.SetPingHandler(h)
}

// SetCloseHandler sets the handler for close messages
func (c *ConnectionWrapper) SetCloseHandler(h func(code int, text string) error) {
	c.conn.SetCloseHandler(h)
}

// RemoteAddr returns the remote network address
func (c *ConnectionWrapper) RemoteAddr() string {
	if c.conn.RemoteAddr() != nil {
		return c.conn.RemoteAddr().String()
	}
	return ""
}