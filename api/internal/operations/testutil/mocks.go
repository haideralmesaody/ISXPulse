package testutil

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"isxcli/internal/operations"
)

// MockStage is a configurable mock implementation of the step interface
type MockStage struct {
	IDValue          string
	NameValue        string
	DependenciesValue []string
	
	// Configurable functions
	ExecuteFunc  func(ctx context.Context, state *operations.OperationState) error
	ValidateFunc func(state *operations.OperationState) error
	
	// Call tracking
	mu               sync.Mutex
	ExecuteCalls     int
	ExecuteArgs      []ExecuteCall
	ValidateCalls    int
	ValidateArgs     []ValidateCall
}

// ExecuteCall tracks arguments passed to Execute
type ExecuteCall struct {
	Ctx   context.Context
	State *operations.OperationState
	Time  time.Time
}

// ValidateCall tracks arguments passed to Validate
type ValidateCall struct {
	State *operations.OperationState
	Time  time.Time
}

// ID returns the step ID
func (m *MockStage) ID() string {
	return m.IDValue
}

// Name returns the step name
func (m *MockStage) Name() string {
	return m.NameValue
}

// GetDependencies returns the step dependencies
func (m *MockStage) GetDependencies() []string {
	if m.DependenciesValue == nil {
		return []string{}
	}
	return m.DependenciesValue
}

// Execute runs the mock execute function
func (m *MockStage) Execute(ctx context.Context, state *operations.OperationState) error {
	m.mu.Lock()
	m.ExecuteCalls++
	m.ExecuteArgs = append(m.ExecuteArgs, ExecuteCall{
		Ctx:   ctx,
		State: state,
		Time:  time.Now(),
	})
	m.mu.Unlock()
	
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, state)
	}
	return nil
}

// Validate runs the mock validate function
func (m *MockStage) Validate(state *operations.OperationState) error {
	m.mu.Lock()
	m.ValidateCalls++
	m.ValidateArgs = append(m.ValidateArgs, ValidateCall{
		State: state,
		Time:  time.Now(),
	})
	m.mu.Unlock()
	
	if m.ValidateFunc != nil {
		return m.ValidateFunc(state)
	}
	return nil
}

// GetExecuteCalls returns the number of Execute calls
func (m *MockStage) GetExecuteCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ExecuteCalls
}

// GetValidateCalls returns the number of Validate calls
func (m *MockStage) GetValidateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ValidateCalls
}

// RequiredInputs returns empty requirements by default
func (m *MockStage) RequiredInputs() []operations.DataRequirement {
	return []operations.DataRequirement{}
}

// ProducedOutputs returns empty outputs by default
func (m *MockStage) ProducedOutputs() []operations.DataOutput {
	return []operations.DataOutput{}
}

// CanRun always returns true for mock
func (m *MockStage) CanRun(manifest *operations.PipelineManifest) bool {
	return true
}

// MockWebSocketHub captures WebSocket messages for testing
type MockWebSocketHub struct {
	mu       sync.Mutex
	Messages []WebSocketMessage
}

// WebSocketMessage represents a captured WebSocket message
type WebSocketMessage struct {
	EventType string
	Step     string
	Status    string
	Metadata  interface{}
	Time      time.Time
}

// BroadcastUpdate captures WebSocket messages
func (m *MockWebSocketHub) BroadcastUpdate(eventType, step, status string, metadata interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Messages = append(m.Messages, WebSocketMessage{
		EventType: eventType,
		Step:     step,
		Status:    status,
		Metadata:  metadata,
		Time:      time.Now(),
	})
}

// BroadcastRefresh captures refresh messages
func (m *MockWebSocketHub) BroadcastRefresh(source string, components []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Messages = append(m.Messages, WebSocketMessage{
		EventType: "refresh",
		Step:     source,
		Metadata: map[string]interface{}{
			"components": components,
		},
		Time: time.Now(),
	})
}

// GetMessages returns all captured messages
func (m *MockWebSocketHub) GetMessages() []WebSocketMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	messages := make([]WebSocketMessage, len(m.Messages))
	copy(messages, m.Messages)
	return messages
}

// GetMessagesByType returns messages of a specific type
func (m *MockWebSocketHub) GetMessagesByType(eventType string) []WebSocketMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var filtered []WebSocketMessage
	for _, msg := range m.Messages {
		if msg.EventType == eventType {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// Clear removes all captured messages
func (m *MockWebSocketHub) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = nil
}

// MockLogger captures log messages for testing (deprecated interface compatibility)
type MockLogger struct {
	mu          sync.Mutex
	InfoLogs    []LogEntry
	ErrorLogs   []LogEntry
	WarningLogs []LogEntry
	DebugLogs   []LogEntry
}

// LogEntry represents a captured log entry
type LogEntry struct {
	Format string
	Args   []interface{}
	Time   time.Time
}

// DEPRECATED: These methods maintain compatibility with old Logger interface
// New tests should use MockSlogHandler or slog.Default() for testing

// Info captures info log messages
func (m *MockLogger) Info(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.InfoLogs = append(m.InfoLogs, LogEntry{
		Format: format,
		Args:   args,
		Time:   time.Now(),
	})
}

// Error captures error log messages
func (m *MockLogger) Error(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ErrorLogs = append(m.ErrorLogs, LogEntry{
		Format: format,
		Args:   args,
		Time:   time.Now(),
	})
}

// Warn captures warning log messages
func (m *MockLogger) Warn(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.WarningLogs = append(m.WarningLogs, LogEntry{
		Format: format,
		Args:   args,
		Time:   time.Now(),
	})
}

// Debug captures debug log messages
func (m *MockLogger) Debug(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.DebugLogs = append(m.DebugLogs, LogEntry{
		Format: format,
		Args:   args,
		Time:   time.Now(),
	})
}

// GetInfoLogs returns all info logs
func (m *MockLogger) GetInfoLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	logs := make([]LogEntry, len(m.InfoLogs))
	copy(logs, m.InfoLogs)
	return logs
}

// GetErrorLogs returns all error logs
func (m *MockLogger) GetErrorLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	logs := make([]LogEntry, len(m.ErrorLogs))
	copy(logs, m.ErrorLogs)
	return logs
}

// GetWarningLogs returns all warning logs
func (m *MockLogger) GetWarningLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	logs := make([]LogEntry, len(m.WarningLogs))
	copy(logs, m.WarningLogs)
	return logs
}

// Clear removes all captured logs
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.InfoLogs = nil
	m.ErrorLogs = nil
	m.WarningLogs = nil
	m.DebugLogs = nil
}

// MockSlogHandler captures slog messages for testing
type MockSlogHandler struct {
	mu      sync.Mutex
	records []MockLogRecord
}

// MockLogRecord represents a captured slog record
type MockLogRecord struct {
	Level   slog.Level
	Message string
	Attrs   map[string]interface{}
	Time    time.Time
}

// NewMockSlogHandler creates a new mock slog handler
func NewMockSlogHandler() *MockSlogHandler {
	return &MockSlogHandler{
		records: make([]MockLogRecord, 0),
	}
}

// Handle implements slog.Handler interface
func (h *MockSlogHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	attrs := make(map[string]interface{})
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})
	
	h.records = append(h.records, MockLogRecord{
		Level:   record.Level,
		Message: record.Message,
		Attrs:   attrs,
		Time:    record.Time,
	})
	
	return nil
}

// Enabled implements slog.Handler interface
func (h *MockSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// WithAttrs implements slog.Handler interface
func (h *MockSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For testing purposes, return the same handler
	// In a production implementation, you'd create a new handler with base attributes
	return h
}

// WithGroup implements slog.Handler interface
func (h *MockSlogHandler) WithGroup(name string) slog.Handler {
	// For testing purposes, return the same handler
	// In a production implementation, you'd create a new handler with group context
	return h
}

// GetRecords returns all captured log records
func (h *MockSlogHandler) GetRecords() []MockLogRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	records := make([]MockLogRecord, len(h.records))
	copy(records, h.records)
	return records
}

// GetRecordsByLevel returns records filtered by level
func (h *MockSlogHandler) GetRecordsByLevel(level slog.Level) []MockLogRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	var filtered []MockLogRecord
	for _, record := range h.records {
		if record.Level == level {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

// GetInfoRecords returns info level records
func (h *MockSlogHandler) GetInfoRecords() []MockLogRecord {
	return h.GetRecordsByLevel(slog.LevelInfo)
}

// GetErrorRecords returns error level records
func (h *MockSlogHandler) GetErrorRecords() []MockLogRecord {
	return h.GetRecordsByLevel(slog.LevelError)
}

// GetWarnRecords returns warn level records
func (h *MockSlogHandler) GetWarnRecords() []MockLogRecord {
	return h.GetRecordsByLevel(slog.LevelWarn)
}

// GetDebugRecords returns debug level records
func (h *MockSlogHandler) GetDebugRecords() []MockLogRecord {
	return h.GetRecordsByLevel(slog.LevelDebug)
}

// Clear removes all captured records
func (h *MockSlogHandler) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = nil
}

// HasMessage checks if any record contains the given message
func (h *MockSlogHandler) HasMessage(message string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	for _, record := range h.records {
		if record.Message == message {
			return true
		}
	}
	return false
}

// HasAttr checks if any record contains the given attribute
func (h *MockSlogHandler) HasAttr(key string, value interface{}) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	for _, record := range h.records {
		if attrValue, exists := record.Attrs[key]; exists {
			if attrValue == value {
				return true
			}
		}
	}
	return false
}

// CountRecords returns the total number of captured records
func (h *MockSlogHandler) CountRecords() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.records)
}

// CountRecordsByLevel returns the number of records at the given level
func (h *MockSlogHandler) CountRecordsByLevel(level slog.Level) int {
	return len(h.GetRecordsByLevel(level))
}

// CreateTestSlogLogger creates a slog.Logger with MockSlogHandler for testing
func CreateTestSlogLogger() (*slog.Logger, *MockSlogHandler) {
	handler := NewMockSlogHandler()
	logger := slog.New(handler)
	return logger, handler
}