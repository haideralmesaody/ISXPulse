package websocket

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	meterName = "isxcli.websocket"
)

// OTelMetrics provides OpenTelemetry metrics for WebSocket operations
type OTelMetrics struct {
	// Connection metrics
	connectionsTotal    metric.Int64Counter
	connectionsActive   metric.Int64UpDownCounter
	connectionDuration  metric.Float64Histogram
	connectionErrors    metric.Int64Counter

	// Message metrics
	messagesTotal       metric.Int64Counter
	messageBytes        metric.Int64Counter
	messageErrors       metric.Int64Counter
	messageLatency      metric.Float64Histogram

	// Queue metrics
	queueDepth          metric.Int64Gauge
	queueOperations     metric.Int64Counter
	droppedMessages     metric.Int64Counter

	// Hub metrics
	broadcastOperations metric.Int64Counter
	clientCount         metric.Int64Gauge
}

// NewOTelMetrics creates a new OpenTelemetry metrics instance
func NewOTelMetrics() (*OTelMetrics, error) {
	meter := otel.Meter(meterName)

	connectionsTotal, err := meter.Int64Counter(
		"websocket_connections_total",
		metric.WithDescription("Total number of WebSocket connections"),
	)
	if err != nil {
		return nil, err
	}

	connectionsActive, err := meter.Int64UpDownCounter(
		"websocket_connections_active",
		metric.WithDescription("Number of active WebSocket connections"),
	)
	if err != nil {
		return nil, err
	}

	connectionDuration, err := meter.Float64Histogram(
		"websocket_connection_duration_seconds",
		metric.WithDescription("Duration of WebSocket connections"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	connectionErrors, err := meter.Int64Counter(
		"websocket_connection_errors_total",
		metric.WithDescription("Total number of WebSocket connection errors"),
	)
	if err != nil {
		return nil, err
	}

	messagesTotal, err := meter.Int64Counter(
		"websocket_messages_total",
		metric.WithDescription("Total number of WebSocket messages"),
	)
	if err != nil {
		return nil, err
	}

	messageBytes, err := meter.Int64Counter(
		"websocket_message_bytes_total",
		metric.WithDescription("Total bytes of WebSocket messages"),
	)
	if err != nil {
		return nil, err
	}

	messageErrors, err := meter.Int64Counter(
		"websocket_message_errors_total",
		metric.WithDescription("Total number of WebSocket message errors"),
	)
	if err != nil {
		return nil, err
	}

	messageLatency, err := meter.Float64Histogram(
		"websocket_message_latency_seconds",
		metric.WithDescription("Latency of WebSocket message processing"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	queueDepth, err := meter.Int64Gauge(
		"websocket_queue_depth",
		metric.WithDescription("Current depth of WebSocket message queue"),
	)
	if err != nil {
		return nil, err
	}

	queueOperations, err := meter.Int64Counter(
		"websocket_queue_operations_total",
		metric.WithDescription("Total number of WebSocket queue operations"),
	)
	if err != nil {
		return nil, err
	}

	droppedMessages, err := meter.Int64Counter(
		"websocket_dropped_messages_total",
		metric.WithDescription("Total number of dropped WebSocket messages"),
	)
	if err != nil {
		return nil, err
	}

	broadcastOperations, err := meter.Int64Counter(
		"websocket_broadcast_operations_total",
		metric.WithDescription("Total number of WebSocket broadcast operations"),
	)
	if err != nil {
		return nil, err
	}

	clientCount, err := meter.Int64Gauge(
		"websocket_client_count",
		metric.WithDescription("Current number of connected WebSocket clients"),
	)
	if err != nil {
		return nil, err
	}

	return &OTelMetrics{
		connectionsTotal:    connectionsTotal,
		connectionsActive:   connectionsActive,
		connectionDuration:  connectionDuration,
		connectionErrors:    connectionErrors,
		messagesTotal:       messagesTotal,
		messageBytes:        messageBytes,
		messageErrors:       messageErrors,
		messageLatency:      messageLatency,
		queueDepth:          queueDepth,
		queueOperations:     queueOperations,
		droppedMessages:     droppedMessages,
		broadcastOperations: broadcastOperations,
		clientCount:         clientCount,
	}, nil
}

// Connection Metrics

// RecordConnection records a new WebSocket connection
func (m *OTelMetrics) RecordConnection(ctx context.Context, clientID, remoteAddr string) {
	attrs := []attribute.KeyValue{
		attribute.String("client_id", clientID),
		attribute.String("remote_addr", remoteAddr),
	}

	m.connectionsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.connectionsActive.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordDisconnection records a WebSocket disconnection
func (m *OTelMetrics) RecordDisconnection(ctx context.Context, clientID string, duration time.Duration, reason string) {
	attrs := []attribute.KeyValue{
		attribute.String("client_id", clientID),
		attribute.String("disconnect_reason", reason),
	}

	m.connectionsActive.Add(ctx, -1, metric.WithAttributes(attrs...))
	m.connectionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordConnectionError records a WebSocket connection error
func (m *OTelMetrics) RecordConnectionError(ctx context.Context, clientID, errorType string, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("client_id", clientID),
		attribute.String("error_type", errorType),
		attribute.String("error", err.Error()),
	}

	m.connectionErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Message Metrics

// RecordMessageSent records a sent WebSocket message
func (m *OTelMetrics) RecordMessageSent(ctx context.Context, messageType, clientID string, size int64) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.messageLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(
			attribute.String("direction", "outbound"),
			attribute.String("message_type", messageType),
		))
	}()

	attrs := []attribute.KeyValue{
		attribute.String("direction", "outbound"),
		attribute.String("message_type", messageType),
		attribute.String("client_id", clientID),
	}

	m.messagesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.messageBytes.Add(ctx, size, metric.WithAttributes(attrs...))
}

// RecordMessageReceived records a received WebSocket message
func (m *OTelMetrics) RecordMessageReceived(ctx context.Context, messageType, clientID string, size int64) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.messageLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(
			attribute.String("direction", "inbound"),
			attribute.String("message_type", messageType),
		))
	}()

	attrs := []attribute.KeyValue{
		attribute.String("direction", "inbound"),
		attribute.String("message_type", messageType),
		attribute.String("client_id", clientID),
	}

	m.messagesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.messageBytes.Add(ctx, size, metric.WithAttributes(attrs...))
}

// RecordMessageError records a WebSocket message error
func (m *OTelMetrics) RecordMessageError(ctx context.Context, messageType, clientID, errorType string, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("message_type", messageType),
		attribute.String("client_id", clientID),
		attribute.String("error_type", errorType),
		attribute.String("error", err.Error()),
	}

	m.messageErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Queue Metrics

// RecordQueueDepth records the current message queue depth
func (m *OTelMetrics) RecordQueueDepth(ctx context.Context, depth int64, queueType string) {
	attrs := []attribute.KeyValue{
		attribute.String("queue_type", queueType),
	}

	m.queueDepth.Record(ctx, depth, metric.WithAttributes(attrs...))
}

// RecordQueueOperation records a queue operation (enqueue/dequeue)
func (m *OTelMetrics) RecordQueueOperation(ctx context.Context, operation, queueType string) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("queue_type", queueType),
	}

	m.queueOperations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordDroppedMessage records a dropped message
func (m *OTelMetrics) RecordDroppedMessage(ctx context.Context, messageType, reason string) {
	attrs := []attribute.KeyValue{
		attribute.String("message_type", messageType),
		attribute.String("drop_reason", reason),
	}

	m.droppedMessages.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Hub Metrics

// RecordBroadcast records a broadcast operation
func (m *OTelMetrics) RecordBroadcast(ctx context.Context, messageType string, clientCount, successCount, failCount int64) {
	attrs := []attribute.KeyValue{
		attribute.String("message_type", messageType),
		attribute.Int64("client_count", clientCount),
		attribute.Int64("success_count", successCount),
		attribute.Int64("fail_count", failCount),
	}

	m.broadcastOperations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordClientCount records the current number of connected clients
func (m *OTelMetrics) RecordClientCount(ctx context.Context, count int64) {
	m.clientCount.Record(ctx, count)
}

// Business Logic Metrics

// RecordOperationEvent records operation-related WebSocket events
func (m *OTelMetrics) RecordOperationEvent(ctx context.Context, pipelineID, eventType, step string) {
	attrs := []attribute.KeyValue{
		attribute.String("pipeline_id", pipelineID),
		attribute.String("event_type", eventType),
		attribute.String("step", step),
	}

	m.messagesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordSystemEvent records system-related WebSocket events
func (m *OTelMetrics) RecordSystemEvent(ctx context.Context, eventType, severity string) {
	attrs := []attribute.KeyValue{
		attribute.String("event_type", eventType),
		attribute.String("severity", severity),
	}

	m.messagesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Global OTel metrics instance
var globalOTelMetrics *OTelMetrics

// InitOTelMetrics initializes the global OpenTelemetry metrics
func InitOTelMetrics() error {
	metrics, err := NewOTelMetrics()
	if err != nil {
		return err
	}
	globalOTelMetrics = metrics
	return nil
}

// GetOTelMetrics returns the global OpenTelemetry metrics instance
func GetOTelMetrics() *OTelMetrics {
	return globalOTelMetrics
}

// Middleware function to automatically record metrics
func WithOTelMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if globalOTelMetrics == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Record WebSocket upgrade requests
		if r.Header.Get("Upgrade") == "websocket" {
			// This will be handled by the WebSocket handler
		}

		next.ServeHTTP(w, r)
	})
}

// Helper function for integration with existing metrics
func RecordOTelMetrics(ctx context.Context, operation string, attrs map[string]interface{}) {
	if globalOTelMetrics == nil {
		return
	}

	switch operation {
	case "connection":
		if clientID, ok := attrs["client_id"].(string); ok {
			if remoteAddr, ok := attrs["remote_addr"].(string); ok {
				globalOTelMetrics.RecordConnection(ctx, clientID, remoteAddr)
			}
		}
	case "disconnection":
		if clientID, ok := attrs["client_id"].(string); ok {
			if duration, ok := attrs["duration"].(time.Duration); ok {
				reason := "normal"
				if r, ok := attrs["reason"].(string); ok {
					reason = r
				}
				globalOTelMetrics.RecordDisconnection(ctx, clientID, duration, reason)
			}
		}
	case "message_sent":
		if messageType, ok := attrs["message_type"].(string); ok {
			if clientID, ok := attrs["client_id"].(string); ok {
				if size, ok := attrs["size"].(int64); ok {
					globalOTelMetrics.RecordMessageSent(ctx, messageType, clientID, size)
				}
			}
		}
	case "message_received":
		if messageType, ok := attrs["message_type"].(string); ok {
			if clientID, ok := attrs["client_id"].(string); ok {
				if size, ok := attrs["size"].(int64); ok {
					globalOTelMetrics.RecordMessageReceived(ctx, messageType, clientID, size)
				}
			}
		}
	case "broadcast":
		if messageType, ok := attrs["message_type"].(string); ok {
			clientCount := int64(0)
			successCount := int64(0)
			failCount := int64(0)
			
			if c, ok := attrs["client_count"].(int64); ok {
				clientCount = c
			}
			if s, ok := attrs["success_count"].(int64); ok {
				successCount = s
			}
			if f, ok := attrs["fail_count"].(int64); ok {
				failCount = f
			}
			
			globalOTelMetrics.RecordBroadcast(ctx, messageType, clientCount, successCount, failCount)
		}
	}
}