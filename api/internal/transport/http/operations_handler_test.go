package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	
	"isxcli/internal/operations"
)

// MockOperationsService is a mock implementation of the operations service
type MockOperationsService struct {
	mock.Mock
}

func (m *MockOperationsService) ExecuteOperation(ctx context.Context, request *operations.OperationRequest) (*operations.OperationResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*operations.OperationResponse), args.Error(1)
}

func (m *MockOperationsService) GetOperationStatus(ctx context.Context, operationID string) (*operations.OperationState, error) {
	args := m.Called(ctx, operationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*operations.OperationState), args.Error(1)
}

func (m *MockOperationsService) CancelOperation(ctx context.Context, operationID string) error {
	args := m.Called(ctx, operationID)
	return args.Error(0)
}

func (m *MockOperationsService) ListOperations(ctx context.Context) ([]*operations.OperationState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*operations.OperationState), args.Error(1)
}

func (m *MockOperationsService) ListOperationsByStatus(ctx context.Context, status operations.OperationStatusValue) ([]*operations.OperationState, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*operations.OperationState), args.Error(1)
}

func (m *MockOperationsService) GetOperationMetrics(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockOperationsService) GetOperationTypes(ctx context.Context) ([]operations.OperationType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]operations.OperationType), args.Error(1)
}

// MockHub is a mock implementation of the Hub interface
type MockHub struct {
	mock.Mock
}

func (m *MockHub) BroadcastUpdate(updateType, subtype, action string, data interface{}) {
	m.Called(updateType, subtype, action, data)
}

// Test helper to create a new operations handler with mocks
func setupOperationsHandler(t *testing.T) (*OperationsHandler, *MockOperationsService, *MockHub) {
	service := &MockOperationsService{}
	hub := &MockHub{}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewOperationsHandler(service, hub, logger)
	
	// Setup default hub expectations
	hub.On("BroadcastUpdate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	
	return handler, service, hub
}

// Test helper to create a router with the handler
func setupRouter(handler *OperationsHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	
	r.Route("/api/v1/operations", func(r chi.Router) {
		r.Post("/", handler.StartOperation)
		r.Get("/", handler.ListOperations)
		r.Get("/{id}", handler.GetOperationStatus)
		r.Post("/{id}/stop", handler.StopOperation)
	})
	
	return r
}

func TestOperationsHandler_StartOperation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockOperationsService)
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful operation start",
			requestBody: OperationRequest{
				Mode: "full",
				Steps: []StepConfig{
					{ID: "scraper", Type: "scraping"},
					{ID: "processor", Type: "processing"},
				},
				Parameters: map[string]interface{}{
					"from_date": "2024-01-01",
					"to_date":   "2024-01-31",
				},
			},
			setupMocks: func(s *MockOperationsService) {
				s.On("ExecuteOperation", mock.Anything, mock.Anything).Return(&operations.OperationResponse{
					ID: "test-operation",
					Status: operations.OperationStatusCompleted,
					Duration: 5 * time.Second,
					Steps: map[string]*operations.StepState{
						"scraper": {
							ID:       "scraper",
							Name:     "Scraper",
							Status:   operations.StepStatusCompleted,
							Progress: 100,
						},
						"processor": {
							ID:       "processor",
							Name:     "Processor",
							Status:   operations.StepStatusCompleted,
							Progress: 100,
						},
					},
				}, nil)
			},
			expectedStatus: http.StatusAccepted,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["id"]) // ID is generated from request ID
				assert.NotNil(t, body["success"])
				assert.NotNil(t, body["steps"])
			},
		},
		{
			name: "invalid request body",
			requestBody: map[string]interface{}{
				"invalid": "data",
			},
			setupMocks: func(s *MockOperationsService) {
				// No mocks needed - validation should fail
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "/errors/validation_failed", body["type"])
				assert.NotEmpty(t, body["title"])
			},
		},
		{
			name: "missing required fields",
			requestBody: OperationRequest{
				// Missing Mode - empty request
				Steps: []StepConfig{
					{ID: "step1", Type: "processing"},
				},
			},
			setupMocks: func(s *MockOperationsService) {
				// No mocks needed - validation should fail
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "/errors/validation_failed", body["type"])
				assert.Contains(t, body["detail"], "mode is required")
			},
		},
		{
			name: "service error",
			requestBody: OperationRequest{
				Mode: "full",
				Steps: []StepConfig{
					{ID: "step1", Type: "processing"},
				},
			},
			setupMocks: func(s *MockOperationsService) {
				s.On("ExecuteOperation", mock.Anything, mock.Anything).
					Return(nil, errors.New("service unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "/errors/operation_failed", body["type"])
				assert.Contains(t, body["detail"], "Failed to execute operation")
			},
		},
		{
			name: "operation with dependencies",
			requestBody: OperationRequest{
				Mode: "full",
				Steps: []StepConfig{
					{ID: "step1", Type: "processing"},
					{ID: "step2", Type: "processing", Dependencies: []string{"step1"}},
				},
			},
			setupMocks: func(s *MockOperationsService) {
				s.On("ExecuteOperation", mock.Anything, mock.Anything).
					Return(&operations.OperationResponse{
						ID:       "test-operation",
						Status:   operations.OperationStatusCompleted,
						Duration: 3 * time.Second,
						Steps:    make(map[string]*operations.StepState),
					}, nil)
			},
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, service, _ := setupOperationsHandler(t)
			router := setupRouter(handler)

			if tt.setupMocks != nil {
				tt.setupMocks(service)
			}

			// Create request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v1/operations", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			if tt.validateBody != nil {
				tt.validateBody(t, responseBody)
			}

			service.AssertExpectations(t)
		})
	}
}

func TestOperationsHandler_GetOperationStatus(t *testing.T) {
	tests := []struct {
		name           string
		operationID    string
		setupMocks     func(*MockOperationsService)
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name:        "successful status retrieval",
			operationID: "op-123",
			setupMocks: func(s *MockOperationsService) {
				status := operations.NewOperationState("op-123")
				status.Start()
				status.SetStage("step1", &operations.StepState{
					ID:       "step1",
					Name:     "Scraper",
					Status:   operations.StepStatusCompleted,
					Progress: 100,
				})
				
				s.On("GetOperationStatus", mock.Anything, "op-123").Return(status, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "op-123", body["id"])
				assert.Equal(t, string(operations.OperationStatusRunning), body["status"])
				assert.NotNil(t, body["steps"])
			},
		},
		{
			name:        "operation not found",
			operationID: "non-existent",
			setupMocks: func(s *MockOperationsService) {
				s.On("GetOperationStatus", mock.Anything, "non-existent").
					Return(nil, operations.ErrOperationNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "/errors/not_found", body["type"])
				assert.Contains(t, body["detail"], "Operation not found")
			},
		},
		{
			name:        "service error",
			operationID: "op-123",
			setupMocks: func(s *MockOperationsService) {
				s.On("GetOperationStatus", mock.Anything, "op-123").
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "/errors/internal_error", body["type"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, service, _ := setupOperationsHandler(t)
			router := setupRouter(handler)

			if tt.setupMocks != nil {
				tt.setupMocks(service)
			}

			// Create request
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/operations/%s", tt.operationID), nil)

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			if tt.validateBody != nil {
				tt.validateBody(t, responseBody)
			}

			service.AssertExpectations(t)
		})
	}
}

func TestOperationsHandler_StopOperation(t *testing.T) {
	tests := []struct {
		name           string
		operationID    string
		setupMocks     func(*MockOperationsService)
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name:        "successful cancellation",
			operationID: "op-123",
			setupMocks: func(s *MockOperationsService) {
				s.On("CancelOperation", mock.Anything, "op-123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "Operation cancelled successfully", body["message"])
			},
		},
		{
			name:        "operation not found",
			operationID: "non-existent",
			setupMocks: func(s *MockOperationsService) {
				s.On("CancelOperation", mock.Anything, "non-existent").
					Return(operations.ErrOperationNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "/errors/not_found", body["type"])
			},
		},
		{
			name:        "operation already completed",
			operationID: "completed-op",
			setupMocks: func(s *MockOperationsService) {
				s.On("CancelOperation", mock.Anything, "completed-op").
					Return(operations.ErrOperationCompleted)
			},
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "/errors/invalid_state", body["type"])
				assert.Contains(t, body["detail"], "cannot be cancelled")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, service, _ := setupOperationsHandler(t)
			router := setupRouter(handler)

			if tt.setupMocks != nil {
				tt.setupMocks(service)
			}

			// Create request
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/operations/%s/stop", tt.operationID), nil)

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			if tt.validateBody != nil {
				tt.validateBody(t, responseBody)
			}

			service.AssertExpectations(t)
		})
	}
}

func TestOperationsHandler_ListOperations(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMocks     func(*MockOperationsService)
		expectedStatus int
		validateBody   func(*testing.T, interface{})
	}{
		{
			name: "list all operations",
			setupMocks: func(s *MockOperationsService) {
				operationsList := []*operations.OperationState{
					createTestOperationStatus("op-1", operations.OperationStatusRunning),
					createTestOperationStatus("op-2", operations.OperationStatusCompleted),
					createTestOperationStatus("op-3", operations.OperationStatusFailed),
				}
				s.On("ListOperations", mock.Anything).Return(operationsList, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body interface{}) {
				ops := body.([]interface{})
				assert.Len(t, ops, 3)
			},
		},
		{
			name: "filter by status",
			queryParams: map[string]string{
				"status": "running",
			},
			setupMocks: func(s *MockOperationsService) {
				operationsList := []*operations.OperationState{
					createTestOperationStatus("op-1", operations.OperationStatusRunning),
					createTestOperationStatus("op-4", operations.OperationStatusRunning),
				}
				s.On("ListOperationsByStatus", mock.Anything, operations.OperationStatusRunning).
					Return(operationsList, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body interface{}) {
				ops := body.([]interface{})
				assert.Len(t, ops, 2)
				for _, op := range ops {
					opMap := op.(map[string]interface{})
					assert.Equal(t, string(operations.OperationStatusRunning), opMap["status"])
				}
			},
		},
		{
			name: "invalid status filter",
			queryParams: map[string]string{
				"status": "invalid-status",
			},
			setupMocks:     func(s *MockOperationsService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body interface{}) {
				bodyMap := body.(map[string]interface{})
				assert.Equal(t, "/errors/validation_failed", bodyMap["type"])
				assert.Contains(t, bodyMap["detail"], "Invalid status")
			},
		},
		{
			name: "service error",
			setupMocks: func(s *MockOperationsService) {
				s.On("ListOperations", mock.Anything).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body interface{}) {
				bodyMap := body.(map[string]interface{})
				assert.Equal(t, "/errors/list_failed", bodyMap["type"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, service, _ := setupOperationsHandler(t)
			router := setupRouter(handler)

			if tt.setupMocks != nil {
				tt.setupMocks(service)
			}

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/operations", nil)
			
			// Add query parameters
			if tt.queryParams != nil {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Add(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			if tt.validateBody != nil {
				tt.validateBody(t, responseBody)
			}

			service.AssertExpectations(t)
		})
	}
}

// Test concurrent requests
func TestOperationsHandler_ConcurrentRequests(t *testing.T) {
	handler, service, _ := setupOperationsHandler(t)
	router := setupRouter(handler)

	// Setup mocks for concurrent access
	service.On("ExecuteOperation", mock.Anything, mock.Anything).
		Return(&operations.OperationResponse{
			ID:       "test-op",
			Status:   operations.OperationStatusCompleted,
			Duration: time.Second,
			Steps:    make(map[string]*operations.StepState),
		}, nil).Maybe()

	service.On("GetOperationStatus", mock.Anything, mock.Anything).
		Return(createTestOperationStatus("test-op", operations.OperationStatusRunning), nil).Maybe()

	service.On("ListOperations", mock.Anything).
		Return([]*operations.OperationState{}, nil).Maybe()

	// Create multiple concurrent requests
	const numRequests = 10
	done := make(chan bool, numRequests*3)

	// Start operations
	for i := 0; i < numRequests; i++ {
		go func() {
			req := OperationRequest{
				Mode: "full",
				Steps: []StepConfig{
					{ID: "step1", Type: "processing"},
				},
			}
			body, _ := json.Marshal(req)
			r := httptest.NewRequest("POST", "/api/v1/operations", bytes.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			assert.Equal(t, http.StatusAccepted, w.Code)
			done <- true
		}()
	}

	// Get status
	for i := 0; i < numRequests; i++ {
		go func() {
			r := httptest.NewRequest("GET", "/api/v1/operations/test-op", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// List operations
	for i := 0; i < numRequests; i++ {
		go func() {
			r := httptest.NewRequest("GET", "/api/v1/operations", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests*3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}

// Test request validation
func TestOperationsHandler_RequestValidation(t *testing.T) {
	handler, _, _ := setupOperationsHandler(t)
	router := setupRouter(handler)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "EOF",
		},
		{
			name:           "invalid JSON",
			requestBody:    "{invalid json}",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid character",
		},
		{
			name:           "empty object",
			requestBody:    "{}",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "mode is required",
		},
		{
			name: "invalid mode",
			requestBody: `{
				"mode": "invalid-mode",
				"steps": [{"id": "step1", "type": "test"}]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid mode",
		},
		{
			name: "empty steps array",
			requestBody: `{
				"mode": "full",
				"steps": []
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "at least one step is required",
		},
		{
			name: "step without ID",
			requestBody: `{
				"mode": "full",
				"steps": [{"type": "test"}]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "step ID is required",
		},
		{
			name: "step without type",
			requestBody: `{
				"mode": "full",
				"steps": [{"id": "step1"}]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "step type is required",
		},
		{
			name: "invalid timeout format",
			requestBody: `{
				"mode": "full",
				"steps": [{
					"id": "step1",
					"type": "test",
					"timeout": "invalid"
				}]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid timeout format",
		},
		{
			name: "circular dependency",
			requestBody: `{
				"mode": "full",
				"steps": [
					{"id": "step1", "type": "test", "dependencies": ["step2"]},
					{"id": "step2", "type": "test", "dependencies": ["step1"]}
				]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "circular dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/operations", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			assert.Equal(t, "/errors/validation_failed", responseBody["type"])
			assert.Contains(t, responseBody["detail"], tt.expectedError)
		})
	}
}

// Test error response format (RFC 7807)
func TestOperationsHandler_ErrorResponseFormat(t *testing.T) {
	handler, service, _ := setupOperationsHandler(t)
	router := setupRouter(handler)

	// Setup mock to return error
	service.On("GetOperationStatus", mock.Anything, "error-op").
		Return(nil, errors.New("internal error"))

	req := httptest.NewRequest("GET", "/api/v1/operations/error-op", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Validate RFC 7807 format
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	// Required fields
	assert.NotEmpty(t, errorResponse["type"])
	assert.NotEmpty(t, errorResponse["title"])
	assert.Equal(t, http.StatusInternalServerError, int(errorResponse["status"].(float64)))

	// Optional fields
	assert.NotEmpty(t, errorResponse["instance"])
	assert.NotEmpty(t, errorResponse["timestamp"])
	assert.NotEmpty(t, errorResponse["request_id"])
}

// Helper function to create test operation status
func createTestOperationStatus(id string, status operations.OperationStatusValue) *operations.OperationState {
	opStatus := operations.NewOperationState(id)
	
	switch status {
	case operations.OperationStatusRunning:
		opStatus.Start()
	case operations.OperationStatusCompleted:
		opStatus.Start()
		opStatus.Complete()
	case operations.OperationStatusFailed:
		opStatus.Start()
		opStatus.Fail(errors.New("test failure"))
	case operations.OperationStatusCancelled:
		opStatus.Start()
		opStatus.Cancel()
	}
	
	// Add a test step
	step := operations.NewStepState("test-step", "Test Step")
	step.Start()
	step.UpdateProgress(50, "In progress")
	opStatus.SetStage("test-step", step)
	
	return opStatus
}

// Benchmark handler performance
func BenchmarkOperationsHandler_StartOperation(b *testing.B) {
	handler, service, _ := setupOperationsHandler(&testing.T{})
	router := setupRouter(handler)

	// Setup mock
	service.On("ExecuteOperation", mock.Anything, mock.Anything).
		Return(&operations.OperationResponse{
			ID:       "benchmark-op",
			Status:   operations.OperationStatusCompleted,
			Duration: time.Second,
			Steps:    make(map[string]*operations.StepState),
		}, nil)

	req := OperationRequest{
		Mode: "full",
		Steps: []StepConfig{
			{ID: "step1", Type: "processing"},
		},
	}
	body, _ := json.Marshal(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := httptest.NewRequest("POST", "/api/v1/operations", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
	}
}

func BenchmarkOperationsHandler_GetStatus(b *testing.B) {
	handler, service, _ := setupOperationsHandler(&testing.T{})
	router := setupRouter(handler)

	// Setup mock
	status := createTestOperationStatus("bench-op", operations.OperationStatusRunning)
	service.On("GetOperationStatus", mock.Anything, "bench-op").Return(status, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := httptest.NewRequest("GET", "/api/v1/operations/bench-op", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
	}
}