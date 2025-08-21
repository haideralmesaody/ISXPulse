package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"log/slog"

	"isxcli/internal/operations"
)

// Mock WebSocket Hub
type mockHub struct {
	mock.Mock
}

func (m *mockHub) BroadcastUpdate(updateType, subtype, action string, data interface{}) {
	m.Called(updateType, subtype, action, data)
}

// Mock Operations Service
type mockOperationsService struct {
	mock.Mock
}

func (m *mockOperationsService) ExecuteOperation(ctx context.Context, request *operations.OperationRequest) (*operations.OperationResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*operations.OperationResponse), args.Error(1)
}

func (m *mockOperationsService) GetOperationStatus(ctx context.Context, operationID string) (*operations.OperationState, error) {
	args := m.Called(ctx, operationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*operations.OperationState), args.Error(1)
}

func (m *mockOperationsService) CancelOperation(ctx context.Context, operationID string) error {
	args := m.Called(ctx, operationID)
	return args.Error(0)
}

func (m *mockOperationsService) ListOperations(ctx context.Context) ([]*operations.OperationState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*operations.OperationState), args.Error(1)
}

func (m *mockOperationsService) ListOperationsByStatus(ctx context.Context, status operations.OperationStatusValue) ([]*operations.OperationState, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*operations.OperationState), args.Error(1)
}

func (m *mockOperationsService) GetOperationMetrics(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockOperationsService) GetOperationTypes(ctx context.Context) ([]operations.OperationType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]operations.OperationType), args.Error(1)
}

// Test StartOperation with tracing
func TestOperationsHandler_StartOperation_Observability(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Setup mocks
	mockService := new(mockOperationsService)
	mockWsHub := new(mockHub)
	logger := slog.Default()

	// Create handler
	handler := NewOperationsHandler(mockService, mockWsHub, logger)

	// Create test request
	reqBody := `{
		"mode": "full",
		"steps": [
			{"id": "step1", "type": "scrape", "parameters": {}}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/operations/start", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Add request ID to context
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-req-123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Setup mock expectations
	expectedResult := &operations.OperationResponse{
		ID:       "test-op-123",
		Status:   operations.OperationStatusCompleted,
		Duration: 100 * time.Millisecond,
		Steps: map[string]*operations.StepState{
			"step1": {
				ID:     "step1",
				Name:   "Step 1",
				Status: operations.StepStatusCompleted,
			},
		},
	}

	mockService.On("ExecuteOperation", mock.Anything, mock.AnythingOfType("*operations.OperationRequest")).
		Return(expectedResult, nil)

	mockWsHub.On("BroadcastUpdate", "operation_update", "started", "active", mock.Anything).
		Return()

	// Execute request
	handler.StartOperation(w, req)

	// Verify response
	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotEmpty(t, response["id"])

	// Verify spans were created
	spans := exporter.GetSpans()
	assert.NotEmpty(t, spans)

	// Find the operation handler span
	var handlerSpan *tracetest.SpanStub
	for _, span := range spans {
		if span.Name == "operations_handler.start_operation" {
			handlerSpan = &span
			break
		}
	}

	assert.NotNil(t, handlerSpan)
	assert.Equal(t, "operations_handler.start_operation", handlerSpan.Name)

	// Verify span attributes
	attrs := handlerSpan.Attributes
	assert.NotEmpty(t, attrs)

	// Check for expected attributes
	hasRequestID := false
	hasOperationID := false
	hasStepsCount := false

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "request_id":
			hasRequestID = true
			assert.Equal(t, "test-req-123", attr.Value.AsString())
		case "operation.id":
			hasOperationID = true
		case "operation.steps_count":
			hasStepsCount = true
			assert.Equal(t, int64(1), attr.Value.AsInt64())
		}
	}

	assert.True(t, hasRequestID, "Span should have request_id attribute")
	assert.True(t, hasOperationID, "Span should have operation.id attribute")
	assert.True(t, hasStepsCount, "Span should have operation.steps_count attribute")

	// Verify mock expectations
	mockService.AssertExpectations(t)
	mockWsHub.AssertExpectations(t)
}

// Test GetOperationStatus with tracing
func TestOperationsHandler_GetOperationStatus_Observability(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Setup mocks
	mockService := new(mockOperationsService)
	mockWsHub := new(mockHub)
	logger := slog.Default()

	// Create handler
	handler := NewOperationsHandler(mockService, mockWsHub, logger)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/operations/op-123/status", nil)
	
	// Add URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "op-123")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.RequestIDKey, "test-req-456")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Setup mock expectations
	startTime := time.Now().Add(-5 * time.Minute)
	expectedStatus := &operations.OperationState{
		ID:        "op-123",
		Status:    operations.OperationStatusRunning,
		StartTime: startTime,
		Steps: map[string]*operations.StepState{
			"step1": {
				ID:     "step1",
				Name:   "Step 1",
				Status: operations.StepStatusCompleted,
			},
		},
	}

	mockService.On("GetOperationStatus", mock.Anything, "op-123").
		Return(expectedStatus, nil)

	// Execute request
	handler.GetOperationStatus(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "op-123", response["id"])
	assert.Equal(t, "running", response["status"])

	// Verify spans were created
	spans := exporter.GetSpans()
	assert.NotEmpty(t, spans)

	// Find the get status span
	var statusSpan *tracetest.SpanStub
	for _, span := range spans {
		if span.Name == "operations_handler.get_status" {
			statusSpan = &span
			break
		}
	}

	assert.NotNil(t, statusSpan)

	// Verify span attributes
	hasOperationID := false
	hasOperationStatus := false

	for _, attr := range statusSpan.Attributes {
		switch string(attr.Key) {
		case "operation.id":
			hasOperationID = true
			assert.Equal(t, "op-123", attr.Value.AsString())
		case "operation.status":
			hasOperationStatus = true
			assert.Equal(t, "running", attr.Value.AsString())
		}
	}

	assert.True(t, hasOperationID, "Span should have operation.id attribute")
	assert.True(t, hasOperationStatus, "Span should have operation.status attribute")

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

// Test StopOperation with tracing and metrics
func TestOperationsHandler_StopOperation_Observability(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Setup mocks
	mockService := new(mockOperationsService)
	mockWsHub := new(mockHub)
	logger := slog.Default()

	// Create handler
	handler := NewOperationsHandler(mockService, mockWsHub, logger)

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/api/operations/op-789/stop", nil)
	
	// Add URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "op-789")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.RequestIDKey, "test-req-789")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Setup mock expectations
	mockService.On("CancelOperation", mock.Anything, "op-789").
		Return(nil)

	mockWsHub.On("BroadcastUpdate", "operation_update", "cancelled", "cancelled", mock.Anything).
		Return()

	// Execute request
	handler.StopOperation(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify spans were created
	spans := exporter.GetSpans()
	assert.NotEmpty(t, spans)

	// Find the stop operation span
	var stopSpan *tracetest.SpanStub
	for _, span := range spans {
		if span.Name == "operations_handler.stop_operation" {
			stopSpan = &span
			break
		}
	}

	assert.NotNil(t, stopSpan)

	// Verify span attributes include cancellation duration
	hasCancellationDuration := false
	for _, attr := range stopSpan.Attributes {
		if string(attr.Key) == "cancellation.duration_ms" {
			hasCancellationDuration = true
			// Duration should be positive
			assert.Greater(t, attr.Value.AsFloat64(), float64(0))
		}
	}

	assert.True(t, hasCancellationDuration, "Span should have cancellation.duration_ms attribute")

	// Verify mock expectations
	mockService.AssertExpectations(t)
	mockWsHub.AssertExpectations(t)
}

// Test error scenarios with proper span error recording
func TestOperationsHandler_ErrorScenarios_Observability(t *testing.T) {
	tests := []struct {
		name           string
		operation      string
		setupMock      func(*mockOperationsService)
		expectedStatus int
		expectSpanError bool
	}{
		{
			name:      "operation not found",
			operation: "get_status",
			setupMock: func(m *mockOperationsService) {
				m.On("GetOperationStatus", mock.Anything, "op-notfound").
					Return(nil, operations.ErrOperationNotFound)
			},
			expectedStatus:  http.StatusNotFound,
			expectSpanError: true,
		},
		{
			name:      "operation already completed",
			operation: "cancel",
			setupMock: func(m *mockOperationsService) {
				m.On("CancelOperation", mock.Anything, "op-completed").
					Return(operations.ErrOperationCompleted)
			},
			expectedStatus:  http.StatusConflict,
			expectSpanError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create in-memory span exporter for testing
			exporter := tracetest.NewInMemoryExporter()
			tp := trace.NewTracerProvider(
				trace.WithSyncer(exporter),
			)
			otel.SetTracerProvider(tp)
			defer tp.Shutdown(context.Background())

			// Setup mocks
			mockService := new(mockOperationsService)
			mockWsHub := new(mockHub)
			logger := slog.Default()

			// Create handler
			handler := NewOperationsHandler(mockService, mockWsHub, logger)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Create appropriate request based on operation
			var req *http.Request
			var operationID string

			switch tt.operation {
			case "get_status":
				operationID = "op-notfound"
				req = httptest.NewRequest(http.MethodGet, "/api/operations/"+operationID+"/status", nil)
			case "cancel":
				operationID = "op-completed"
				req = httptest.NewRequest(http.MethodPost, "/api/operations/"+operationID+"/stop", nil)
			}

			// Add URL params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", operationID)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "test-error-req")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Execute request
			switch tt.operation {
			case "get_status":
				handler.GetOperationStatus(w, req)
			case "cancel":
				handler.StopOperation(w, req)
			}

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify error was recorded in span
			spans := exporter.GetSpans()
			assert.NotEmpty(t, spans)

			// Find the relevant span
			var span *tracetest.SpanStub
			for _, s := range spans {
				if strings.Contains(s.Name, "operations_handler") {
					span = &s
					break
				}
			}

			assert.NotNil(t, span)

			if tt.expectSpanError {
				// Check that error was recorded
				assert.NotEmpty(t, span.Events)
				
				// Look for error event
				hasErrorEvent := false
				for _, event := range span.Events {
					if event.Name == "exception" {
						hasErrorEvent = true
						break
					}
				}
				assert.True(t, hasErrorEvent, "Span should have error event")
				
				// Status should be error
				assert.Equal(t, codes.Error, span.Status.Code)
			}

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}