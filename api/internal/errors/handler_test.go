package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"isxcli/internal/shared/testutil"
)

func TestNewErrorHandler(t *testing.T) {
	tests := []struct {
		name         string
		includeStack bool
	}{
		{
			name:         "create handler with stack traces",
			includeStack: true,
		},
		{
			name:         "create handler without stack traces",
			includeStack: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			
			handler := NewErrorHandler(logger, tt.includeStack)
			
			assert.NotNil(t, handler)
			assert.Equal(t, tt.includeStack, handler.includeStack)
			assert.NotNil(t, handler.logger)
		})
	}
}

func TestErrorHandler_HandleError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantType    string
		wantTitle   string
		checkStack  bool
	}{
		{
			name:       "handle nil error",
			err:        nil,
			wantStatus: 0, // No response written
		},
		{
			name:       "handle context deadline exceeded",
			err:        context.DeadlineExceeded,
			wantStatus: http.StatusGatewayTimeout,
			wantType:   TypeTimeout,
			wantTitle:  "Request Timeout",
		},
		{
			name:       "handle context canceled",
			err:        context.Canceled,
			wantStatus: http.StatusGatewayTimeout,
			wantType:   TypeTimeout,
			wantTitle:  "Request Timeout",
		},
		{
			name:       "handle APIError",
			err:        ErrInvalidRequest,
			wantStatus: http.StatusBadRequest,
			wantType:   TypeValidation,
			wantTitle:  "Bad Request",
		},
		{
			name:       "handle license expired error",
			err:        fmt.Errorf("license expired"),
			wantStatus: http.StatusForbidden,
			wantType:   TypeLicenseExpired,
			wantTitle:  "License Expired",
		},
		{
			name:       "handle not found error",
			err:        fmt.Errorf("resource not found"),
			wantStatus: http.StatusNotFound,
			wantType:   TypeNotFound,
			wantTitle:  "Resource Not Found",
		},
		{
			name:       "handle generic error",
			err:        fmt.Errorf("something went wrong"),
			wantStatus: http.StatusInternalServerError,
			wantType:   TypeInternal,
			wantTitle:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logHandler := testutil.NewTestLogger(t)
			handler := NewErrorHandler(logger, true)
			
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			// Add request ID for tracing
			ctx := context.WithValue(r.Context(), "RequestID", "test-request-id")
			r = r.WithContext(ctx)

			handler.HandleError(w, r, tt.err)

			if tt.err == nil {
				// Should not write any response for nil error
				assert.Equal(t, 0, w.Code)
				return
			}

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response body
			var problem ProblemDetails
			err := json.NewDecoder(w.Body).Decode(&problem)
			require.NoError(t, err)

			assert.Equal(t, tt.wantType, problem.Type)
			assert.Equal(t, tt.wantTitle, problem.Title)
			assert.Equal(t, tt.wantStatus, problem.Status)

			// Check that error was logged
			assert.True(t, logHandler.ContainsMessage("request failed"))
		})
	}
}

func TestErrorHandler_ErrorToProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantType   string
		wantTitle  string
	}{
		{
			name:       "convert context deadline exceeded",
			err:        context.DeadlineExceeded,
			wantStatus: http.StatusGatewayTimeout,
			wantType:   TypeTimeout,
			wantTitle:  "Request Timeout",
		},
		{
			name:       "convert APIError validation failed",
			err:        ErrValidationFailed,
			wantStatus: http.StatusBadRequest,
			wantType:   TypeValidation,
			wantTitle:  "Bad Request",
		},
		{
			name:       "convert APIError not found",
			err:        ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantType:   TypeNotFound,
			wantTitle:  "Not Found",
		},
		{
			name:       "convert APIError unauthorized",
			err:        ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantType:   TypeUnauthorized,
			wantTitle:  "Unauthorized",
		},
		{
			name:       "convert string error with 'not found'",
			err:        fmt.Errorf("user not found"),
			wantStatus: http.StatusNotFound,
			wantType:   TypeNotFound,
			wantTitle:  "Resource Not Found",
		},
		{
			name:       "convert string error with 'unauthorized'",
			err:        fmt.Errorf("unauthorized access"),
			wantStatus: http.StatusUnauthorized,
			wantType:   TypeUnauthorized,
			wantTitle:  "Unauthorized",
		},
		{
			name:       "convert string error with 'forbidden'",
			err:        fmt.Errorf("forbidden resource"),
			wantStatus: http.StatusForbidden,
			wantType:   TypeForbidden,
			wantTitle:  "Forbidden",
		},
		{
			name:       "convert string error with 'rate limit'",
			err:        fmt.Errorf("rate limit exceeded"),
			wantStatus: http.StatusTooManyRequests,
			wantType:   TypeRateLimit,
			wantTitle:  "Rate Limit Exceeded",
		},
		{
			name:       "convert string error with 'conflict'",
			err:        fmt.Errorf("resource conflict"),
			wantStatus: http.StatusConflict,
			wantType:   TypeConflict,
			wantTitle:  "Conflict",
		},
		{
			name:       "convert string error with 'payload too large'",
			err:        fmt.Errorf("payload too large"),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantType:   TypePayloadTooLarge,
			wantTitle:  "Payload Too Large",
		},
		{
			name:       "convert generic error",
			err:        fmt.Errorf("generic error"),
			wantStatus: http.StatusInternalServerError,
			wantType:   TypeInternal,
			wantTitle:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			handler := NewErrorHandler(logger, false)
			
			r := httptest.NewRequest("GET", "/test", nil)
			
			problem := handler.ErrorToProblem(tt.err, r)
			
			assert.Equal(t, tt.wantStatus, problem.Status)
			assert.Equal(t, tt.wantType, problem.Type)
			assert.Equal(t, tt.wantTitle, problem.Title)
			assert.Equal(t, r.URL.Path, problem.Instance)
		})
	}
}

func TestErrorHandler_apiErrorToProblem(t *testing.T) {
	tests := []struct {
		name         string
		apiError     *APIError
		wantStatus   int
		wantType     string
		wantTitle    string
		checkDetails bool
	}{
		{
			name:       "convert validation error",
			apiError:   &APIError{StatusCode: http.StatusBadRequest, ErrorCode: "VALIDATION_FAILED", Message: "Validation failed"},
			wantStatus: http.StatusBadRequest,
			wantType:   TypeValidation,
			wantTitle:  "Bad Request",
		},
		{
			name:       "convert not found error",
			apiError:   &APIError{StatusCode: http.StatusNotFound, ErrorCode: "NOT_FOUND", Message: "Not found"},
			wantStatus: http.StatusNotFound,
			wantType:   TypeNotFound,
			wantTitle:  "Not Found",
		},
		{
			name:       "convert license not found error",
			apiError:   &APIError{StatusCode: http.StatusNotFound, ErrorCode: "LICENSE_NOT_FOUND", Message: "License not found"},
			wantStatus: http.StatusNotFound,
			wantType:   TypeNotFound,
			wantTitle:  "Not Found",
		},
		{
			name:       "convert unauthorized error",
			apiError:   &APIError{StatusCode: http.StatusUnauthorized, ErrorCode: "UNAUTHORIZED", Message: "Unauthorized"},
			wantStatus: http.StatusUnauthorized,
			wantType:   TypeUnauthorized,
			wantTitle:  "Unauthorized",
		},
		{
			name:       "convert forbidden error",
			apiError:   &APIError{StatusCode: http.StatusForbidden, ErrorCode: "FORBIDDEN", Message: "Forbidden"},
			wantStatus: http.StatusForbidden,
			wantType:   TypeForbidden,
			wantTitle:  "Forbidden",
		},
		{
			name:       "convert conflict error",
			apiError:   &APIError{StatusCode: http.StatusConflict, ErrorCode: "CONFLICT", Message: "Conflict"},
			wantStatus: http.StatusConflict,
			wantType:   TypeConflict,
			wantTitle:  "Conflict",
		},
		{
			name:       "convert rate limit error",
			apiError:   &APIError{StatusCode: http.StatusTooManyRequests, ErrorCode: "RATE_LIMIT_EXCEEDED", Message: "Rate limit exceeded"},
			wantStatus: http.StatusTooManyRequests,
			wantType:   TypeRateLimit,
			wantTitle:  "Too Many Requests",
		},
		{
			name:       "convert service unavailable error",
			apiError:   &APIError{StatusCode: http.StatusServiceUnavailable, ErrorCode: "SERVICE_UNAVAILABLE", Message: "Service unavailable"},
			wantStatus: http.StatusServiceUnavailable,
			wantType:   TypeServiceDown,
			wantTitle:  "Service Unavailable",
		},
		{
			name:         "convert error with details",
			apiError:     &APIError{StatusCode: http.StatusBadRequest, ErrorCode: "VALIDATION_FAILED", Message: "Validation failed", Details: map[string]string{"field": "email"}},
			wantStatus:   http.StatusBadRequest,
			wantType:     TypeValidation,
			wantTitle:    "Bad Request",
			checkDetails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			handler := NewErrorHandler(logger, false)
			
			r := httptest.NewRequest("GET", "/test", nil)
			
			problem := handler.apiErrorToProblem(tt.apiError, r)
			
			assert.Equal(t, tt.wantStatus, problem.Status)
			assert.Equal(t, tt.wantType, problem.Type)
			assert.Equal(t, tt.wantTitle, problem.Title)
			assert.Equal(t, tt.apiError.Message, problem.Detail)
			assert.Equal(t, r.URL.Path, problem.Instance)
			
			// Check error_code extension
			assert.Equal(t, tt.apiError.ErrorCode, problem.Extensions["error_code"])
			
			if tt.checkDetails && tt.apiError.Details != nil {
				assert.Equal(t, tt.apiError.Details, problem.Extensions["details"])
			}
		})
	}
}

func TestErrorHandler_HandlePanic(t *testing.T) {
	tests := []struct {
		name         string
		recovered    interface{}
		includeStack bool
		wantMsg      string
	}{
		{
			name:         "handle string panic with stack",
			recovered:    "something went wrong",
			includeStack: true,
			wantMsg:      "something went wrong",
		},
		{
			name:         "handle error panic without stack",
			recovered:    fmt.Errorf("error occurred"),
			includeStack: false,
			wantMsg:      "error occurred",
		},
		{
			name:         "handle integer panic",
			recovered:    42,
			includeStack: false,
			wantMsg:      "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logHandler := testutil.NewTestLogger(t)
			handler := NewErrorHandler(logger, tt.includeStack)
			
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			// Add request ID for tracing
			ctx := context.WithValue(r.Context(), "RequestID", "test-request-id")
			r = r.WithContext(ctx)

			handler.HandlePanic(w, r, tt.recovered)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response body
			var problem ProblemDetails
			err := json.NewDecoder(w.Body).Decode(&problem)
			require.NoError(t, err)

			assert.Equal(t, TypeInternal, problem.Type)
			assert.Equal(t, "Internal Server Error", problem.Title)
			assert.Equal(t, http.StatusInternalServerError, problem.Status)
			assert.Equal(t, "An unexpected error occurred", problem.Detail)

			// Check trace_id extension
			assert.Equal(t, "test-request-id", problem.Extensions["trace_id"])

			if tt.includeStack {
				assert.Contains(t, problem.Extensions, "panic") 
				assert.Contains(t, problem.Extensions, "stack")
				assert.Equal(t, tt.wantMsg, problem.Extensions["panic"])
			}

			// Check that panic was logged
			assert.True(t, logHandler.ContainsMessage("panic recovered"))
		})
	}
}

func TestErrorHandler_NotFound(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "handle 404 for root path",
			path: "/",
		},
		{
			name: "handle 404 for api path",
			path: "/api/v1/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			handler := NewErrorHandler(logger, false)
			
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", tt.path, nil)
			// Add request ID for tracing
			ctx := context.WithValue(r.Context(), "RequestID", "test-request-id")
			r = r.WithContext(ctx)

			handler.NotFound(w, r)

			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response body
			var problem ProblemDetails
			err := json.NewDecoder(w.Body).Decode(&problem)
			require.NoError(t, err)

			assert.Equal(t, TypeNotFound, problem.Type)
			assert.Equal(t, "Not Found", problem.Title)
			assert.Equal(t, http.StatusNotFound, problem.Status)
			assert.Equal(t, "The requested resource was not found", problem.Detail)
			assert.Equal(t, tt.path, problem.Instance)
			assert.Equal(t, "test-request-id", problem.Extensions["trace_id"])
		})
	}
}

func TestErrorHandler_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "handle POST not allowed",
			method: "POST",
			path:   "/api/v1/users",
		},
		{
			name:   "handle PUT not allowed",
			method: "PUT",
			path:   "/api/v1/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			handler := NewErrorHandler(logger, false)
			
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tt.method, tt.path, nil)
			// Add request ID for tracing
			ctx := context.WithValue(r.Context(), "RequestID", "test-request-id")
			r = r.WithContext(ctx)

			handler.MethodNotAllowed(w, r)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response body
			var problem ProblemDetails
			err := json.NewDecoder(w.Body).Decode(&problem)
			require.NoError(t, err)

			assert.Equal(t, TypeInternal, problem.Type)
			assert.Equal(t, "Method Not Allowed", problem.Title)
			assert.Equal(t, http.StatusMethodNotAllowed, problem.Status)
			assert.Equal(t, fmt.Sprintf("Method %s is not allowed for this endpoint", tt.method), problem.Detail)
			assert.Equal(t, tt.path, problem.Instance)
			assert.Equal(t, "test-request-id", problem.Extensions["trace_id"])
		})
	}
}

func TestErrorHandler_Middleware(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantStatus   int
		shouldPanic  bool
		includeStack bool
	}{
		{
			name: "successful request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "request that panics",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("something went wrong")
			},
			wantStatus:   http.StatusInternalServerError,
			shouldPanic:  true,
			includeStack: true,
		},
		{
			name: "request that writes error status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("bad request"))
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logHandler := testutil.NewTestLogger(t)
			errorHandler := NewErrorHandler(logger, tt.includeStack)
			
			middleware := errorHandler.Middleware(tt.handler)
			
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			// Add request ID for tracing
			ctx := context.WithValue(r.Context(), "RequestID", "test-request-id")
			r = r.WithContext(ctx)

			middleware.ServeHTTP(w, r)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.shouldPanic {
				// Should have logged the panic
				assert.True(t, logHandler.ContainsMessage("panic recovered"))
				
				// Response should be JSON problem details
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				
				var problem ProblemDetails
				err := json.NewDecoder(w.Body).Decode(&problem)
				require.NoError(t, err)
				
				assert.Equal(t, TypeInternal, problem.Type)
				assert.Equal(t, http.StatusInternalServerError, problem.Status)
			}
		})
	}
}

func TestErrorResponseWriter(t *testing.T) {
	tests := []struct {
		name        string
		writeStatus int
		writeData   string
		wantStatus  int
		wantLogged  bool
	}{
		{
			name:        "write success status",
			writeStatus: http.StatusOK,
			writeData:   "success",
			wantStatus:  http.StatusOK,
			wantLogged:  false,
		},
		{
			name:        "write client error status",
			writeStatus: http.StatusBadRequest,
			writeData:   "bad request",
			wantStatus:  http.StatusBadRequest,
			wantLogged:  true,
		},
		{
			name:        "write server error status",
			writeStatus: http.StatusInternalServerError,
			writeData:   "internal error",
			wantStatus:  http.StatusInternalServerError,
			wantLogged:  true,
		},
		{
			name:       "write without explicit status",
			writeData:  "default response",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logHandler := testutil.NewTestLogger(t)
			errorHandler := NewErrorHandler(logger, false)
			
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			
			ew := &errorResponseWriter{
				ResponseWriter: w,
				handler:        errorHandler,
				request:        r,
			}

			if tt.writeStatus > 0 {
				ew.WriteHeader(tt.writeStatus)
			}
			
			if tt.writeData != "" {
				ew.Write([]byte(tt.writeData))
			}

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.writeData != "" {
				assert.Contains(t, w.Body.String(), tt.writeData)
			}

			if tt.wantLogged {
				assert.True(t, logHandler.ContainsMessage("error response"))
			}
		})
	}
}

func TestErrorHandler_JSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "write JSON success response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "write JSON error response",
			status:     http.StatusBadRequest,
			data:       map[string]string{"error": "invalid input"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			handler := NewErrorHandler(logger, false)
			
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)

			handler.JSON(w, r, tt.status, tt.data)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			// Parse response to verify JSON structure
			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
		})
	}
}

func TestGetStackTrace(t *testing.T) {
	t.Run("get stack trace", func(t *testing.T) {
		stack := getStackTrace()
		
		assert.NotEmpty(t, stack)
		assert.True(t, strings.Contains(stack, "TestGetStackTrace"))
		assert.True(t, strings.Contains(stack, "getStackTrace"))
	})
}

func TestErrorHandlerEdgeCases(t *testing.T) {
	t.Run("handle validation error with details", func(t *testing.T) {
		logger, _ := testutil.NewTestLogger(t)
		handler := NewErrorHandler(logger, false)
		
		// Create APIError with validation error code and validation details
		validationErrors := []ValidationError{
			{Field: "email", Message: "invalid format"},
			{Field: "password", Message: "too short"},
		}
		apiErr := &APIError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "VALIDATION_ERROR",
			Message:    "Validation failed",
			Details:    validationErrors,
		}
		
		r := httptest.NewRequest("GET", "/test", nil)
		problem := handler.ErrorToProblem(apiErr, r)
		
		assert.Equal(t, http.StatusBadRequest, problem.Status)
		assert.Equal(t, TypeValidation, problem.Type)
		assert.Equal(t, "Validation Failed", problem.Title)
		assert.Equal(t, validationErrors, problem.Extensions["errors"])
	})

	t.Run("handle context with no request ID", func(t *testing.T) {
		logger, _ := testutil.NewTestLogger(t)
		handler := NewErrorHandler(logger, false)
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		// No request ID in context

		handler.HandleError(w, r, fmt.Errorf("test error"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var problem ProblemDetails
		err := json.NewDecoder(w.Body).Decode(&problem)
		require.NoError(t, err)
		
		// trace_id should be empty string when no request ID
		assert.Equal(t, "", problem.Extensions["trace_id"])
	})

	t.Run("multiple writes to error response writer", func(t *testing.T) {
		logger, _ := testutil.NewTestLogger(t)
		errorHandler := NewErrorHandler(logger, false)
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		
		ew := &errorResponseWriter{
			ResponseWriter: w,
			handler:        errorHandler,
			request:        r,
		}

		// First write should set status
		ew.WriteHeader(http.StatusBadRequest)
		// Second write should not change status
		ew.WriteHeader(http.StatusInternalServerError)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.True(t, ew.written)
	})
}

func TestErrorHandlerConcurrency(t *testing.T) {
	t.Run("concurrent error handling", func(t *testing.T) {
		logger, _ := testutil.NewTestLogger(t)
		handler := NewErrorHandler(logger, false)
		
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				defer func() { done <- true }()
				
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", fmt.Sprintf("/test-%d", i), nil)
				ctx := context.WithValue(r.Context(), "RequestID", fmt.Sprintf("req-%d", i))
				r = r.WithContext(ctx)
				
				handler.HandleError(w, r, fmt.Errorf("error %d", i))
				
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			}(i)
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for goroutines to complete")
			}
		}
	})
}