// Package monitoring provides logging examples for scratch card license operations
// This file demonstrates the structured logging standards for ISX Pulse
package monitoring

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// LoggerConfig holds configuration for structured logging
type LoggerConfig struct {
	Environment string
	LogLevel    slog.Level
	Service     string
	Version     string
}

// ScratchCardLogger provides structured logging for scratch card operations
type ScratchCardLogger struct {
	logger      *slog.Logger
	auditLogger *slog.Logger
	config      LoggerConfig
}

// NewScratchCardLogger creates a new structured logger for scratch card operations
func NewScratchCardLogger(config LoggerConfig) *ScratchCardLogger {
	// Setup main logger
	var handler slog.Handler
	if config.Environment == "production" {
		// JSON output for production
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     config.LogLevel,
			AddSource: true,
		})
	} else {
		// Human-readable output for development
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     config.LogLevel,
			AddSource: true,
		})
	}

	logger := slog.New(handler).With(
		"service", config.Service,
		"version", config.Version,
		"environment", config.Environment,
	)

	// Setup separate audit logger
	auditFile, err := os.OpenFile("audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Error("failed to open audit log file", "error", err)
		auditFile = os.Stdout
	}

	auditHandler := slog.NewJSONHandler(auditFile, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false, // Don't include source for audit logs
	})

	auditLogger := slog.New(auditHandler).With(
		"log_type", "audit",
		"service", config.Service,
		"version", config.Version,
	)

	return &ScratchCardLogger{
		logger:      logger,
		auditLogger: auditLogger,
		config:      config,
	}
}

// Enhanced context key types for type safety
type contextKey string

const (
	TraceIDKey    contextKey = "trace_id"
	OperationKey  contextKey = "operation"
	UserIDKey     contextKey = "user_id"
	BatchIDKey    contextKey = "batch_id"
	CardTypeKey   contextKey = "card_type"
)

// Logging helper functions

// getTraceID extracts or generates a trace ID from context
func (scl *ScratchCardLogger) getTraceID(ctx context.Context) string {
	// Try to get trace ID from OpenTelemetry span
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}

	// Try to get from context value
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return traceID
	}

	// Generate new trace ID
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

// hashFingerprint creates a secure hash for logging without exposing full fingerprint
func (scl *ScratchCardLogger) hashFingerprint(fingerprint string) string {
	if fingerprint == "" {
		return "empty"
	}
	hash := sha256.Sum256([]byte(fingerprint))
	return fmt.Sprintf("%x", hash[:4]) // First 8 characters for tracking
}

// getStackTrace captures stack trace for error logging
func (scl *ScratchCardLogger) getStackTrace(skip int) string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var stack string
	for {
		frame, more := frames.Next()
		stack += fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
		if !more {
			break
		}
	}
	return stack
}

// Core logging methods for scratch card operations

// LogBatchOperationStart logs the start of a batch operation
func (scl *ScratchCardLogger) LogBatchOperationStart(ctx context.Context, batchID, cardType string, cardCount int) {
	scl.logger.InfoContext(ctx, "batch operation started",
		"operation", "card_batch_activation",
		"trace_id", scl.getTraceID(ctx),
		"batch_id", batchID,
		"card_count", cardCount,
		"card_type", cardType,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"stage", "initialization",
	)
}

// LogBatchOperationComplete logs the completion of a batch operation
func (scl *ScratchCardLogger) LogBatchOperationComplete(ctx context.Context, batchID string, cardCount int, duration time.Duration, successful int) {
	success_rate := float64(successful) / float64(cardCount) * 100

	scl.logger.InfoContext(ctx, "batch operation completed",
		"operation", "card_batch_activation",
		"trace_id", scl.getTraceID(ctx),
		"batch_id", batchID,
		"total_cards", cardCount,
		"successful_cards", successful,
		"success_rate_percent", success_rate,
		"duration_ms", duration.Milliseconds(),
		"throughput_cards_per_second", float64(cardCount)/duration.Seconds(),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"stage", "completion",
	)

	// Also log to audit for tracking
	scl.auditLogger.InfoContext(ctx, "batch activation completed",
		"event_type", "batch_completion",
		"trace_id", scl.getTraceID(ctx),
		"batch_id", batchID,
		"cards_processed", cardCount,
		"success_rate", success_rate,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// LogCardActivationAttempt logs individual card activation attempts
func (scl *ScratchCardLogger) LogCardActivationAttempt(ctx context.Context, cardType, batchID, fingerprint string) {
	scl.logger.InfoContext(ctx, "scratch card activation attempt",
		"operation", "card_activation",
		"trace_id", scl.getTraceID(ctx),
		"card_type", cardType,
		"batch_id", batchID,
		"device_fingerprint_hash", scl.hashFingerprint(fingerprint),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"stage", "validation",
	)
}

// LogCardActivationSuccess logs successful card activations
func (scl *ScratchCardLogger) LogCardActivationSuccess(ctx context.Context, cardType, batchID, fingerprint string, duration time.Duration) {
	scl.logger.InfoContext(ctx, "scratch card activation successful",
		"operation", "card_activation",
		"trace_id", scl.getTraceID(ctx),
		"card_type", cardType,
		"batch_id", batchID,
		"device_fingerprint_hash", scl.hashFingerprint(fingerprint),
		"duration_ms", duration.Milliseconds(),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"stage", "success",
		"result", "activated",
	)

	// Log to audit stream
	scl.auditLogger.InfoContext(ctx, "card activation successful",
		"event_type", "card_activation",
		"trace_id", scl.getTraceID(ctx),
		"card_type", cardType,
		"batch_id", batchID,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// LogCardActivationFailure logs failed card activations
func (scl *ScratchCardLogger) LogCardActivationFailure(ctx context.Context, cardType, batchID, fingerprint string, err error, duration time.Duration) {
	errorType := scl.classifyError(err)

	scl.logger.ErrorContext(ctx, "scratch card activation failed",
		"operation", "card_activation",
		"trace_id", scl.getTraceID(ctx),
		"error", err.Error(),
		"error_type", errorType,
		"card_type", cardType,
		"batch_id", batchID,
		"device_fingerprint_hash", scl.hashFingerprint(fingerprint),
		"duration_ms", duration.Milliseconds(),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"stage", "failure",
		"stack_trace", scl.getStackTrace(1),
	)

	// Log to audit stream for security tracking
	scl.auditLogger.ErrorContext(ctx, "card activation failed",
		"event_type", "activation_failure",
		"trace_id", scl.getTraceID(ctx),
		"error_type", errorType,
		"card_type", cardType,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// LogAppsScriptRequest logs Apps Script API calls
func (scl *ScratchCardLogger) LogAppsScriptRequest(ctx context.Context, functionName string, requestSize int) {
	scl.logger.InfoContext(ctx, "apps script request",
		"operation", "apps_script_call",
		"trace_id", scl.getTraceID(ctx),
		"script_function", functionName,
		"request_size_bytes", requestSize,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"stage", "request",
	)
}

// LogAppsScriptResponse logs Apps Script API responses
func (scl *ScratchCardLogger) LogAppsScriptResponse(ctx context.Context, functionName string, duration time.Duration, success bool, err error) {
	if success {
		scl.logger.InfoContext(ctx, "apps script response successful",
			"operation", "apps_script_call",
			"trace_id", scl.getTraceID(ctx),
			"script_function", functionName,
			"duration_ms", duration.Milliseconds(),
			"timestamp", time.Now().UTC().Format(time.RFC3339),
			"stage", "response",
			"result", "success",
		)
	} else {
		scl.logger.ErrorContext(ctx, "apps script response failed",
			"operation", "apps_script_call",
			"trace_id", scl.getTraceID(ctx),
			"script_function", functionName,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
			"error_type", scl.classifyAppsScriptError(err),
			"timestamp", time.Now().UTC().Format(time.RFC3339),
			"stage", "response",
			"result", "failure",
		)
	}
}

// LogFingerprintOperation logs device fingerprint operations
func (scl *ScratchCardLogger) LogFingerprintOperation(ctx context.Context, operation string, fingerprint string, duration time.Duration, success bool) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelWarn
	}

	scl.logger.Log(ctx, level, "device fingerprint operation",
		"operation", "fingerprint_"+operation,
		"trace_id", scl.getTraceID(ctx),
		"fingerprint_hash", scl.hashFingerprint(fingerprint),
		"fingerprint_length", len(fingerprint),
		"operation_type", operation, // "generation" or "validation"
		"duration_ms", duration.Milliseconds(),
		"success", success,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)

	// Log mismatches to audit
	if operation == "validation" && !success {
		scl.auditLogger.WarnContext(ctx, "device fingerprint mismatch",
			"event_type", "fingerprint_mismatch",
			"trace_id", scl.getTraceID(ctx),
			"timestamp", time.Now().UTC().Format(time.RFC3339),
		)
	}
}

// Security and audit logging methods

// LogSecurityEvent logs security-related events
func (scl *ScratchCardLogger) LogSecurityEvent(ctx context.Context, eventType, description string, clientIP string) {
	scl.logger.WarnContext(ctx, "security event detected",
		"operation", "security_monitoring",
		"trace_id", scl.getTraceID(ctx),
		"event_type", eventType,
		"description", description,
		"client_ip", clientIP,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)

	// Always log security events to audit
	scl.auditLogger.WarnContext(ctx, "security event",
		"event_type", eventType,
		"trace_id", scl.getTraceID(ctx),
		"description", description,
		"client_ip", clientIP,
		"severity", "security_warning",
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// LogRateLimitEvent logs rate limiting events
func (scl *ScratchCardLogger) LogRateLimitEvent(ctx context.Context, limitType string, clientIP string, currentRate, threshold float64) {
	scl.logger.WarnContext(ctx, "rate limit triggered",
		"operation", "rate_limiting",
		"trace_id", scl.getTraceID(ctx),
		"limit_type", limitType,
		"client_ip", clientIP,
		"current_rate", currentRate,
		"threshold", threshold,
		"exceeded_by", currentRate-threshold,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)

	// Log to audit for security tracking
	scl.auditLogger.WarnContext(ctx, "rate limit triggered",
		"event_type", "rate_limit",
		"trace_id", scl.getTraceID(ctx),
		"limit_type", limitType,
		"client_ip", clientIP,
		"severity", "rate_limit_warning",
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// Performance and cache logging

// LogCacheOperation logs cache hit/miss events
func (scl *ScratchCardLogger) LogCacheOperation(ctx context.Context, operation string, key string, hit bool, duration time.Duration) {
	result := "miss"
	if hit {
		result = "hit"
	}

	scl.logger.DebugContext(ctx, "cache operation",
		"operation", "license_validation_cache",
		"trace_id", scl.getTraceID(ctx),
		"cache_operation", operation,
		"cache_result", result,
		"key_hash", scl.hashFingerprint(key), // Don't log actual key
		"lookup_duration_ms", duration.Milliseconds(),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// LogWebSocketMessage logs WebSocket message processing
func (scl *ScratchCardLogger) LogWebSocketMessage(ctx context.Context, connectionID, messageType string, clientIP string, duration time.Duration) {
	scl.logger.InfoContext(ctx, "websocket message processed",
		"operation", "websocket_message",
		"trace_id", scl.getTraceID(ctx),
		"connection_id", connectionID,
		"message_type", messageType,
		"client_ip", clientIP,
		"processing_duration_ms", duration.Milliseconds(),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// Error classification helpers

// classifyError categorizes general errors for better observability
func (scl *ScratchCardLogger) classifyError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()
	switch {
	case contains(errStr, "card_already_activated"):
		return "card_already_activated"
	case contains(errStr, "invalid_card_format"):
		return "invalid_card_format"
	case contains(errStr, "card_expired"):
		return "card_expired"
	case contains(errStr, "device_mismatch"):
		return "device_mismatch"
	case contains(errStr, "fingerprint_validation_failed"):
		return "fingerprint_validation_failed"
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "network"):
		return "network_error"
	case contains(errStr, "rate_limit"):
		return "rate_limited"
	default:
		return "unknown_error"
	}
}

// classifyAppsScriptError categorizes Apps Script specific errors
func (scl *ScratchCardLogger) classifyAppsScriptError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()
	switch {
	case contains(errStr, "quota_exceeded"):
		return "quota_exceeded"
	case contains(errStr, "script_timeout"):
		return "script_timeout"
	case contains(errStr, "permission_denied"):
		return "permission_denied"
	case contains(errStr, "service_unavailable"):
		return "service_unavailable"
	case contains(errStr, "invalid_request"):
		return "invalid_request"
	default:
		return "apps_script_error"
	}
}

// Helper function for string contains check (simple implementation)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     func() bool {
		         for i := 0; i <= len(s)-len(substr); i++ {
		             if s[i:i+len(substr)] == substr {
		                 return true
		             }
		         }
		         return false
		     }()))
}

// Example usage demonstrating the logging standards
func ExampleUsage() {
	// Setup logger
	config := LoggerConfig{
		Environment: "production",
		LogLevel:    slog.LevelInfo,
		Service:     "isx-pulse-license",
		Version:     "3.0.0",
	}

	logger := NewScratchCardLogger(config)

	// Create context with trace ID
	ctx := context.WithValue(context.Background(), TraceIDKey, "trace-12345")

	// Example: Batch operation
	batchID := "BATCH-001"
	cardType := "PREMIUM"
	cardCount := 10

	logger.LogBatchOperationStart(ctx, batchID, cardType, cardCount)

	// Simulate processing time
	start := time.Now()
	time.Sleep(100 * time.Millisecond)
	duration := time.Since(start)

	// Example: Individual card activation
	fingerprint := "device-fingerprint-hash-123"
	logger.LogCardActivationAttempt(ctx, cardType, batchID, fingerprint)

	// Example: Apps Script call
	logger.LogAppsScriptRequest(ctx, "activateCard", 256)
	
	// Complete operations
	logger.LogCardActivationSuccess(ctx, cardType, batchID, fingerprint, duration)
	logger.LogBatchOperationComplete(ctx, batchID, cardCount, duration, 10)

	// Example: Cache operation
	logger.LogCacheOperation(ctx, "license_lookup", "license-key-hash", true, 5*time.Millisecond)

	// Example: Security event
	logger.LogSecurityEvent(ctx, "suspicious_activity", "Multiple failed activations from same IP", "192.168.1.100")
}