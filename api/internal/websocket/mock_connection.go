package websocket

import (
	"errors"
	"sync"
	"time"
)

// MockConnection is a mock implementation of the Connection interface for testing
type MockConnection struct {
	mu sync.Mutex
	
	// WriteMessage behavior
	WriteMessageFunc func(messageType int, data []byte) error
	WrittenMessages  []MockMessage
	
	// ReadMessage behavior
	ReadMessageFunc func() (messageType int, p []byte, err error)
	ReadMessages    []MockMessage
	ReadIndex       int
	
	// Close behavior
	CloseFunc func() error
	Closed    bool
	
	// Deadline behavior
	ReadDeadline  time.Time
	WriteDeadline time.Time
	
	// Handlers
	PongHandler  func(string) error
	PingHandler  func(string) error
	CloseHandler func(code int, text string) error
	
	// Properties
	RemoteAddress string
	ReadLimit     int64
}

// MockMessage represents a message for mocking
type MockMessage struct {
	Type int
	Data []byte
	Err  error
}

// NewMockConnection creates a new mock connection
func NewMockConnection() *MockConnection {
	return &MockConnection{
		WrittenMessages: make([]MockMessage, 0),
		ReadMessages:    make([]MockMessage, 0),
		RemoteAddress:   "127.0.0.1:8080",
	}
}

// WriteMessage implements Connection.WriteMessage
func (m *MockConnection) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.Closed {
		return errors.New("connection closed")
	}
	
	if m.WriteMessageFunc != nil {
		return m.WriteMessageFunc(messageType, data)
	}
	
	// Default behavior: store the message
	m.WrittenMessages = append(m.WrittenMessages, MockMessage{
		Type: messageType,
		Data: data,
	})
	
	return nil
}

// ReadMessage implements Connection.ReadMessage
func (m *MockConnection) ReadMessage() (messageType int, p []byte, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.Closed {
		return 0, nil, errors.New("connection closed")
	}
	
	if m.ReadMessageFunc != nil {
		return m.ReadMessageFunc()
	}
	
	// Default behavior: return messages from ReadMessages slice
	if m.ReadIndex < len(m.ReadMessages) {
		msg := m.ReadMessages[m.ReadIndex]
		m.ReadIndex++
		return msg.Type, msg.Data, msg.Err
	}
	
	// No more messages
	return 0, nil, errors.New("no more messages")
}

// Close implements Connection.Close
func (m *MockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	
	m.Closed = true
	return nil
}

// SetReadDeadline implements Connection.SetReadDeadline
func (m *MockConnection) SetReadDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ReadDeadline = t
	return nil
}

// SetWriteDeadline implements Connection.SetWriteDeadline
func (m *MockConnection) SetWriteDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.WriteDeadline = t
	return nil
}

// SetReadLimit implements Connection.SetReadLimit
func (m *MockConnection) SetReadLimit(limit int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ReadLimit = limit
}

// SetPongHandler implements Connection.SetPongHandler
func (m *MockConnection) SetPongHandler(h func(string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.PongHandler = h
}

// SetPingHandler implements Connection.SetPingHandler
func (m *MockConnection) SetPingHandler(h func(string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.PingHandler = h
}

// SetCloseHandler implements Connection.SetCloseHandler
func (m *MockConnection) SetCloseHandler(h func(code int, text string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.CloseHandler = h
}

// RemoteAddr implements Connection.RemoteAddr
func (m *MockConnection) RemoteAddr() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.RemoteAddress
}

// Helper methods for testing

// AddReadMessage adds a message to be returned by ReadMessage
func (m *MockConnection) AddReadMessage(messageType int, data []byte, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ReadMessages = append(m.ReadMessages, MockMessage{
		Type: messageType,
		Data: data,
		Err:  err,
	})
}

// GetWrittenMessages returns all messages written to the connection
func (m *MockConnection) GetWrittenMessages() []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	result := make([]MockMessage, len(m.WrittenMessages))
	copy(result, m.WrittenMessages)
	return result
}

// Reset resets the mock connection state
func (m *MockConnection) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.WrittenMessages = make([]MockMessage, 0)
	m.ReadMessages = make([]MockMessage, 0)
	m.ReadIndex = 0
	m.Closed = false
	m.ReadDeadline = time.Time{}
	m.WriteDeadline = time.Time{}
}