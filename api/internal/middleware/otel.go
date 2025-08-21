package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"isxcli/internal/infrastructure"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelMiddleware provides OpenTelemetry instrumentation for HTTP requests
type OTelMiddleware struct {
	tracer          trace.Tracer
	meter           metric.Meter
	businessMetrics *infrastructure.BusinessMetrics
	logger          *slog.Logger
}

// NewOTelMiddleware creates a new OpenTelemetry middleware
func NewOTelMiddleware(providers *infrastructure.OTelProviders) (*OTelMiddleware, error) {
	businessMetrics, err := infrastructure.CreateBusinessMetrics(providers.Meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create business metrics: %w", err)
	}

	return &OTelMiddleware{
		tracer:          providers.Tracer,
		meter:           providers.Meter,
		businessMetrics: businessMetrics,
		logger:          providers.Logger,
	}, nil
}

// Handler returns the middleware handler function
func (m *OTelMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract trace context from incoming request
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

		// Create span for the HTTP request
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := m.tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.HTTPRouteKey.String(r.URL.Path),
				semconv.URLSchemeKey.String(r.URL.Scheme),
				semconv.ServerAddressKey.String(r.Host),
				semconv.UserAgentOriginalKey.String(r.UserAgent()),
				semconv.HTTPRequestBodySizeKey.Int64(r.ContentLength),
				semconv.ClientAddressKey.String(GetRealIP(r)),
			),
		)
		defer span.End()

		// Add trace ID to context for logging correlation
		traceID := span.SpanContext().TraceID().String()
		ctx = infrastructure.WithTraceID(ctx, traceID)
		r = r.WithContext(ctx)

		// Create instrumented response writer
		ww := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		// Record active request
		m.businessMetrics.HTTPActiveRequests.Add(ctx, 1)
		defer m.businessMetrics.HTTPActiveRequests.Add(ctx, -1)

		start := time.Now()

		// Call next handler
		next.ServeHTTP(ww, r)

		// Record metrics and span attributes
		duration := time.Since(start)
		statusCode := ww.statusCode

		// HTTP metrics
		attrs := []attribute.KeyValue{
			attribute.String("method", r.Method),
			attribute.String("route", getRoutePattern(r)),
			attribute.Int("status_code", statusCode),
		}

		m.businessMetrics.HTTPRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
		m.businessMetrics.HTTPRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

		// Update span attributes
		span.SetAttributes(
			semconv.HTTPResponseStatusCodeKey.Int(statusCode),
			semconv.HTTPResponseBodySizeKey.Int64(ww.bytesWritten),
			attribute.Float64("http.request.duration", duration.Seconds()),
		)

		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(statusCode))
		}

		// Log request with correlation
		m.logger.InfoContext(ctx, "HTTP request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("route", getRoutePattern(r)),
			slog.Int("status_code", statusCode),
			slog.Duration("duration", duration),
			slog.String("user_agent", r.UserAgent()),
			slog.String("remote_addr", GetRealIP(r)),
			slog.Int64("bytes_written", ww.bytesWritten),
			slog.String("trace_id", traceID),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// getRoutePattern extracts the route pattern from request context
func getRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return r.URL.Path
}

// TraceMiddleware is a simplified tracing middleware for specific endpoints
func TraceMiddleware(operationName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracer := otel.Tracer("isxcli")
			ctx, span := tracer.Start(r.Context(), operationName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.HTTPRouteKey.String(r.URL.Path),
				),
			)
			defer span.End()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// WebSocketTraceMiddleware creates tracing middleware for WebSocket connections
func WebSocketTraceMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracer := otel.Tracer("isxcli.websocket")
			ctx, span := tracer.Start(r.Context(), "websocket_upgrade",
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.HTTPRouteKey.String("/ws"),
					attribute.String("connection.type", "websocket"),
					attribute.String("origin", r.Header.Get("Origin")),
					attribute.String("sec_websocket_protocol", r.Header.Get("Sec-WebSocket-Protocol")),
				),
			)
			defer span.End()

			// Add trace ID for WebSocket correlation
			traceID := span.SpanContext().TraceID().String()
			ctx = infrastructure.WithTraceID(ctx, traceID)
			r = r.WithContext(ctx)

			logger.InfoContext(ctx, "WebSocket upgrade attempt",
				slog.String("origin", r.Header.Get("Origin")),
				slog.String("trace_id", traceID),
			)

			next.ServeHTTP(w, r)
		})
	}
}

// BusinessMetricsMiddleware provides access to business metrics for handlers
func BusinessMetricsMiddleware(businessMetrics *infrastructure.BusinessMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "business_metrics", businessMetrics)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// GetBusinessMetricsFromContext extracts business metrics from request context
func GetBusinessMetricsFromContext(ctx context.Context) *infrastructure.BusinessMetrics {
	if metrics, ok := ctx.Value("business_metrics").(*infrastructure.BusinessMetrics); ok {
		return metrics
	}
	return nil
}

// PipelineTraceHandler creates a handler that starts a operation execution trace
func PipelineTraceHandler(pipelineType string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tracer := otel.Tracer("isxcli.operation")
		ctx, span := tracer.Start(r.Context(), fmt.Sprintf("operation.%s.start", pipelineType),
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("operation.type", pipelineType),
				attribute.String("operation", "start"),
			),
		)
		defer span.End()

		r = r.WithContext(ctx)

		// Record operation metric
		if metrics := GetBusinessMetricsFromContext(ctx); metrics != nil {
			metrics.OperationExecutionsTotal.Add(ctx, 1,
				metric.WithAttributes(
					attribute.String("pipeline_type", pipelineType),
					attribute.String("operation", "start"),
				),
			)
		}

		handler(w, r)
	}
}

// RecordPipelineStageMetrics records metrics for operation step execution
func RecordPipelineStageMetrics(ctx context.Context, pipelineID, stageName string, duration time.Duration, success bool) {
	if metrics := GetBusinessMetricsFromContext(ctx); metrics != nil {
		status := "success"
		if !success {
			status = "failure"
		}

		attrs := []attribute.KeyValue{
			attribute.String("pipeline_id", pipelineID),
			attribute.String("step", stageName),
			attribute.String("status", status),
		}

		metrics.OperationStepsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	// Add span event
	infrastructure.AddSpanEvent(ctx, "operation.step.completed", map[string]interface{}{
		"pipeline_id": pipelineID,
		"step":       stageName,
		"duration":    duration.Seconds(),
		"success":     success,
	})
}

// RecordLicenseMetrics records license-related metrics
func RecordLicenseMetrics(ctx context.Context, operation string, success bool) {
	if metrics := GetBusinessMetricsFromContext(ctx); metrics != nil {
		attrs := []attribute.KeyValue{
			attribute.String("operation", operation),
		}

		switch operation {
		case "activation":
			metrics.LicenseActivationAttempts.Add(ctx, 1, metric.WithAttributes(attrs...))
			if success {
				metrics.LicenseActivationSuccess.Add(ctx, 1, metric.WithAttributes(attrs...))
			}
		case "validation":
			metrics.LicenseValidationChecks.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}
}

// RecordSystemError records system error metrics
func RecordSystemError(ctx context.Context, errorType, component string) {
	if metrics := GetBusinessMetricsFromContext(ctx); metrics != nil {
		attrs := []attribute.KeyValue{
			attribute.String("error_type", errorType),
			attribute.String("component", component),
		}
		metrics.SystemErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// GetRealIP extracts the real IP address from the request
func GetRealIP(r *http.Request) string {
	// Check for forwarded headers
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}