# ISX Pulse - Scratch Card License Logging Standards

## Overview

This document defines comprehensive logging standards for the scratch card license system, ensuring complete observability and rapid debugging capabilities.

## Core Logging Principles

### 1. Structured Logging (slog)
- **Required**: All logging must use Go's slog package
- **Format**: JSON output for production environments
- **Context**: Include trace_id in every log entry
- **Consistency**: Use standardized field names across all components

### 2. Log Levels
- **DEBUG**: Detailed debugging information, disabled in production
- **INFO**: General operational messages and successful operations
- **WARN**: Warning conditions that don't prevent operation
- **ERROR**: Error conditions that require attention

### 3. Security Logging
- **Audit Stream**: Separate logging stream for security events
- **Data Protection**: Never log sensitive data (license keys, personal info)
- **Correlation**: Link all security events to operations via trace_id

## Scratch Card Specific Logging Requirements

### 1. Card Batch Operations
```go
// Required fields for all batch operations
logger.InfoContext(ctx, "batch operation started",
    "operation", "card_batch_activation",
    "trace_id", getTraceID(ctx),
    "batch_id", batchID,
    "card_count", cardCount,
    "card_type", cardType,
    "timestamp", time.Now().UTC().Format(time.RFC3339),
    "user_id", getUserID(ctx),
)
```

### 2. Individual Card Activation
```go
// Log every card activation attempt
logger.InfoContext(ctx, "scratch card activation attempt",
    "operation", "card_activation",
    "trace_id", getTraceID(ctx),
    "card_type", cardType,
    "batch_id", batchID,
    "device_fingerprint_hash", hashFingerprint(fingerprint),
    "timestamp", time.Now().UTC().Format(time.RFC3339),
    "duration_ms", duration.Milliseconds(),
)
```

### 3. Apps Script API Calls
```go
// Log all Apps Script interactions
logger.InfoContext(ctx, "apps script request",
    "operation", "apps_script_call",
    "trace_id", getTraceID(ctx),
    "script_function", functionName,
    "request_size_bytes", len(requestData),
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

### 4. Device Fingerprint Operations
```go
// Log fingerprint generation and validation
logger.InfoContext(ctx, "device fingerprint operation",
    "operation", "fingerprint_validation",
    "trace_id", getTraceID(ctx),
    "fingerprint_hash", hashFingerprint(fingerprint),
    "validation_result", "success", // or "mismatch"
    "generation_duration_ms", duration.Milliseconds(),
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

### 5. Error Logging
```go
// Log all errors with full context
logger.ErrorContext(ctx, "scratch card activation failed",
    "operation", "card_activation",
    "trace_id", getTraceID(ctx),
    "error", err.Error(),
    "error_type", classifyError(err),
    "card_type", cardType,
    "batch_id", batchID,
    "duration_ms", duration.Milliseconds(),
    "stack_trace", getStackTrace(err),
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

## Correlation ID Requirements

### 1. Trace ID Propagation
- **Source**: Generate unique trace ID for each request
- **Flow**: Propagate through all operations and services
- **Format**: Use OpenTelemetry trace ID format
- **Context**: Store in Go context and pass to all functions

### 2. Operation Manifest Tracking
```go
// Link operations to manifest for flow tracking
logger.InfoContext(ctx, "operation stage completed",
    "operation", "data_processing",
    "trace_id", getTraceID(ctx),
    "manifest_id", manifestID,
    "stage", "card_validation",
    "stage_duration_ms", duration.Milliseconds(),
    "next_stage", "activation",
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

### 3. WebSocket Message Correlation
```go
// Track WebSocket operations with correlation
logger.InfoContext(ctx, "websocket message processed",
    "operation", "websocket_message",
    "trace_id", getTraceID(ctx),
    "connection_id", connectionID,
    "message_type", messageType,
    "client_ip", clientIP,
    "processing_duration_ms", duration.Milliseconds(),
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

## Performance Logging

### 1. Timing Data
- **Start/End**: Log operation start and completion times
- **Duration**: Include processing duration in milliseconds
- **Throughput**: Log items processed per second for batch operations
- **Resource Usage**: Include memory and CPU metrics for expensive operations

### 2. Cache Operations
```go
// Log cache hit/miss with performance data
logger.DebugContext(ctx, "cache operation",
    "operation", "license_validation_cache",
    "trace_id", getTraceID(ctx),
    "cache_result", "hit", // or "miss"
    "key_hash", hashKey(key),
    "lookup_duration_ms", duration.Milliseconds(),
    "cache_size", cacheSize,
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

## Security Audit Logging

### 1. Authentication Events
```go
// Log all authentication attempts
auditLogger.InfoContext(ctx, "authentication attempt",
    "event_type", "auth_attempt",
    "trace_id", getTraceID(ctx),
    "user_agent", userAgent,
    "client_ip", clientIP,
    "result", "success", // or "failure"
    "failure_reason", reason, // if applicable
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

### 2. Rate Limiting Events
```go
// Log rate limiting triggers
auditLogger.WarnContext(ctx, "rate limit triggered",
    "event_type", "rate_limit",
    "trace_id", getTraceID(ctx),
    "client_ip", clientIP,
    "endpoint", endpoint,
    "limit_type", "apps_script", // or "license_activation"
    "current_rate", currentRate,
    "limit_threshold", threshold,
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

### 3. Suspicious Activity
```go
// Log potential security threats
auditLogger.ErrorContext(ctx, "suspicious activity detected",
    "event_type", "security_threat",
    "trace_id", getTraceID(ctx),
    "threat_type", "multiple_device_mismatch",
    "client_ip", clientIP,
    "attempt_count", attemptCount,
    "time_window_minutes", timeWindow,
    "blocked", true,
    "timestamp", time.Now().UTC().Format(time.RFC3339),
)
```

## Implementation Patterns

### 1. Logger Configuration
```go
// Production logger setup with structured JSON output
func SetupProductionLogger() *slog.Logger {
    opts := &slog.HandlerOptions{
        Level: slog.LevelInfo,
        AddSource: true,
    }
    
    handler := slog.NewJSONHandler(os.Stdout, opts)
    return slog.New(handler)
}

// Development logger setup with human-readable output
func SetupDevelopmentLogger() *slog.Logger {
    opts := &slog.HandlerOptions{
        Level: slog.LevelDebug,
        AddSource: true,
    }
    
    handler := slog.NewTextHandler(os.Stdout, opts)
    return slog.New(handler)
}
```

### 2. Context Enhancement
```go
// Add trace ID and operation context to all logging
func WithLoggingContext(ctx context.Context, operation string) context.Context {
    traceID := getOrGenerateTraceID(ctx)
    
    logger := slog.Default().With(
        "trace_id", traceID,
        "operation", operation,
        "service", "isx-license",
        "version", version.BuildVersion,
    )
    
    return context.WithValue(ctx, "logger", logger)
}
```

### 3. Middleware Integration
```go
// HTTP middleware for request logging
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        traceID := generateTraceID()
        
        ctx := context.WithValue(r.Context(), "trace_id", traceID)
        r = r.WithContext(ctx)
        
        // Log request start
        slog.InfoContext(ctx, "http request started",
            "method", r.Method,
            "path", r.URL.Path,
            "trace_id", traceID,
            "client_ip", getClientIP(r),
            "user_agent", r.UserAgent(),
            "timestamp", start.UTC().Format(time.RFC3339),
        )
        
        next.ServeHTTP(w, r)
        
        duration := time.Since(start)
        
        // Log request completion
        slog.InfoContext(ctx, "http request completed",
            "method", r.Method,
            "path", r.URL.Path,
            "trace_id", traceID,
            "duration_ms", duration.Milliseconds(),
            "timestamp", time.Now().UTC().Format(time.RFC3339),
        )
    })
}
```

## Log Aggregation and Analysis

### 1. Centralized Logging
- **Collection**: Use Fluent Bit or similar for log collection
- **Storage**: Elasticsearch or similar for searchable log storage
- **Retention**: 30 days for INFO/DEBUG, 1 year for WARN/ERROR, 3 years for audit logs

### 2. Real-time Monitoring
- **Alerting**: Set up alerts on ERROR and WARN patterns
- **Dashboards**: Create Grafana dashboards for log analysis
- **Anomaly Detection**: Implement log-based anomaly detection

### 3. Compliance Requirements
- **Data Protection**: Ensure no PII in logs
- **Audit Trail**: Maintain complete audit trail for license operations
- **Retention**: Follow legal requirements for log retention

## Tools and Integration

### 1. Development Tools
- **Local**: Use slog text handler for development
- **Testing**: Capture logs in tests for verification
- **Debugging**: Enable DEBUG level for troubleshooting

### 2. Production Tools
- **Format**: JSON-only output for production
- **Collection**: Fluent Bit or Vector for log shipping
- **Storage**: Elasticsearch, Loki, or CloudWatch Logs
- **Analysis**: Grafana, Kibana, or CloudWatch Insights

## Compliance Checklist

- [ ] All logging uses slog package
- [ ] JSON output format for production
- [ ] Trace ID in every log entry
- [ ] No sensitive data in logs
- [ ] Separate audit logging stream
- [ ] Performance timing data included
- [ ] Error logs include stack traces
- [ ] Context propagation implemented
- [ ] Security events properly logged
- [ ] Rate limiting events tracked
- [ ] Cache operations monitored
- [ ] WebSocket messages correlated
- [ ] Batch operations fully traced