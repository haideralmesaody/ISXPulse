package operations

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidationError tests validation error functionality
func TestValidationError(t *testing.T) {
	tests := []struct {
		name      string
		stageID   string
		message   string
		wantError string
	}{
		{
			name:      "basic validation error",
			stageID:   "test-stage",
			message:   "validation failed",
			wantError: "[validation] test-stage: validation failed",
		},
		{
			name:      "validation error with empty stage",
			stageID:   "",
			message:   "empty stage validation",
			wantError: "[validation] empty stage validation",
		},
		{
			name:      "validation error with empty message",
			stageID:   "stage-1",
			message:   "",
			wantError: "[validation] stage-1: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.stageID, tt.message)
			
			assert.Error(t, err)
			assert.Equal(t, tt.wantError, err.Error())
			assert.Equal(t, tt.stageID, err.Step)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, ErrorTypeValidation, err.Type)
			assert.False(t, err.Retryable)
			
			// Test error type assertion
			var operationErr *OperationError
			assert.True(t, errors.As(err, &operationErr))
		})
	}
}

// TestExecutionError tests execution error functionality
func TestExecutionError(t *testing.T) {
	tests := []struct {
		name      string
		stageID   string
		cause     error
		retryable bool
		wantError string
	}{
		{
			name:      "basic execution error retryable",
			stageID:   "exec-stage",
			cause:     errors.New("temporary failure"),
			retryable: true,
			wantError: "[execution] exec-stage: Step execution failed",
		},
		{
			name:      "execution error non-retryable",
			stageID:   "complex-stage", 
			cause:     errors.New("fatal error"),
			retryable: false,
			wantError: "[execution] complex-stage: Step execution failed",
		},
		{
			name:      "execution error with nil cause",
			stageID:   "nil-stage",
			cause:     nil,
			retryable: true,
			wantError: "[execution] nil-stage: Step execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewExecutionError(tt.stageID, tt.cause, tt.retryable)
			
			assert.Error(t, err)
			assert.Equal(t, tt.wantError, err.Error())
			assert.Equal(t, tt.stageID, err.Step)
			assert.Equal(t, "Step execution failed", err.Message)
			assert.Equal(t, ErrorTypeExecution, err.Type)
			assert.Equal(t, tt.retryable, err.Retryable)
			assert.Equal(t, tt.cause, err.Cause)
			
			// Test error type assertion
			var operationErr *OperationError
			assert.True(t, errors.As(err, &operationErr))
		})
	}
}

// TestTimeoutError tests timeout error functionality
func TestTimeoutError(t *testing.T) {
	tests := []struct {
		name      string
		stageID   string
		timeout   string
		wantError string
	}{
		{
			name:      "basic timeout error",
			stageID:   "timeout-stage",
			timeout:   "30s",
			wantError: "[timeout] timeout-stage: Step exceeded timeout of 30s",
		},
		{
			name:      "timeout error with minutes",
			stageID:   "long-stage",
			timeout:   "5m0s",
			wantError: "[timeout] long-stage: Step exceeded timeout of 5m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTimeoutError(tt.stageID, tt.timeout)
			
			assert.Error(t, err)
			assert.Equal(t, tt.wantError, err.Error())
			assert.Equal(t, tt.stageID, err.Step)
			assert.Equal(t, ErrorTypeTimeout, err.Type)
			assert.True(t, err.Retryable) // Timeout errors are retryable according to the implementation
			
			// Check context contains timeout
			assert.NotNil(t, err.Context)
			assert.Equal(t, tt.timeout, err.Context["timeout"])
			
			// Test error type assertion
			var operationErr *OperationError
			assert.True(t, errors.As(err, &operationErr))
		})
	}
}

// TestCancellationError tests cancellation error functionality
func TestCancellationError(t *testing.T) {
	tests := []struct {
		name      string
		stageID   string
		wantError string
	}{
		{
			name:      "basic cancellation error",
			stageID:   "cancelled-stage",
			wantError: "[cancellation] cancelled-stage: operation was cancelled",
		},
		{
			name:      "cancellation error with empty stage",
			stageID:   "",
			wantError: "[cancellation] operation was cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewCancellationError(tt.stageID)
			
			assert.Error(t, err)
			assert.Equal(t, tt.wantError, err.Error())
			assert.Equal(t, tt.stageID, err.Step)
			assert.Equal(t, ErrorTypeCancellation, err.Type)
			assert.False(t, err.Retryable)
			
			// Test error type assertion
			var operationErr *OperationError
			assert.True(t, errors.As(err, &operationErr))
		})
	}
}

// TestFatalError tests fatal error functionality
func TestFatalError(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		cause     error
		wantError string
	}{
		{
			name:      "fatal error without cause",
			message:   "fatal system error",
			cause:     nil,
			wantError: "[fatal] fatal system error",
		},
		{
			name:      "fatal error with cause",
			message:   "database connection failed",
			cause:     errors.New("connection refused"),
			wantError: "[fatal] database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewFatalError(tt.message, tt.cause)
			
			assert.Error(t, err)
			assert.Equal(t, tt.wantError, err.Error())
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.cause, err.Cause)
			assert.Equal(t, ErrorTypeFatal, err.Type)
			assert.False(t, err.Retryable)
			
			// Test error type assertion
			var operationErr *OperationError
			assert.True(t, errors.As(err, &operationErr))
		})
	}
}

// TestDependencyError tests dependency error functionality
func TestDependencyError(t *testing.T) {
	tests := []struct {
		name      string
		stageID   string
		dependsOn string
		message   string
		wantError string
	}{
		{
			name:      "basic dependency error",
			stageID:   "stage2",
			dependsOn: "stage1",
			message:   "stage1 not completed",
			wantError: "[dependency] stage2: stage1 not completed",
		},
		{
			name:      "complex dependency error",
			stageID:   "final-stage",
			dependsOn: "processing",
			message:   "processing stage failed validation",
			wantError: "[dependency] final-stage: processing stage failed validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDependencyError(tt.stageID, tt.dependsOn, tt.message)
			
			assert.Error(t, err)
			assert.Equal(t, tt.wantError, err.Error())
			assert.Equal(t, tt.stageID, err.Step)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, ErrorTypeDependency, err.Type)
			assert.False(t, err.Retryable)
			
			// Check context contains dependency info
			assert.NotNil(t, err.Context)
			assert.Equal(t, tt.dependsOn, err.Context["depends_on"])
			
			// Test error type assertion
			var operationErr *OperationError
			assert.True(t, errors.As(err, &operationErr))
		})
	}
}

// TestErrorUnwrapping tests error unwrapping functionality
func TestErrorUnwrapping(t *testing.T) {
	t.Run("unwrap operation error with cause", func(t *testing.T) {
		originalErr := errors.New("original error")
		operationErr := NewFatalError("fatal", originalErr)
		
		unwrapped := errors.Unwrap(operationErr)
		assert.Equal(t, originalErr, unwrapped)
	})

	t.Run("unwrap operation error without cause", func(t *testing.T) {
		operationErr := NewFatalError("fatal", nil)
		
		unwrapped := errors.Unwrap(operationErr)
		assert.Nil(t, unwrapped)
	})
	
	t.Run("unwrap nil error", func(t *testing.T) {
		var operationErr *OperationError
		unwrapped := operationErr.Unwrap()
		assert.Nil(t, unwrapped)
	})
}

// TestIsRetryable tests the IsRetryable function
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantRetryable bool
	}{
		{
			name:          "nil error",
			err:           nil,
			wantRetryable: false,
		},
		{
			name:          "validation error - not retryable",
			err:           NewValidationError("stage", "validation failed"),
			wantRetryable: false,
		},
		{
			name:          "timeout error - retryable", 
			err:           NewTimeoutError("stage", "30s"),
			wantRetryable: true,
		},
		{
			name:          "cancellation error - not retryable",
			err:           NewCancellationError("stage"),
			wantRetryable: false,
		},
		{
			name:          "fatal error - not retryable",
			err:           NewFatalError("fatal", nil),
			wantRetryable: false,
		},
		{
			name:          "execution error retryable",
			err:           NewExecutionError("stage", errors.New("temp failure"), true),
			wantRetryable: true,
		},
		{
			name:          "execution error non-retryable",
			err:           NewExecutionError("stage", errors.New("fatal failure"), false),
			wantRetryable: false,
		},
		{
			name:          "generic error - not retryable",
			err:           errors.New("generic error"),
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			assert.Equal(t, tt.wantRetryable, result)
		})
	}
}

// TestGetErrorType tests the GetErrorType function
func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedType: "",
		},
		{
			name:         "validation error",
			err:          NewValidationError("stage", "validation failed"),
			expectedType: ErrorTypeValidation,
		},
		{
			name:         "execution error",
			err:          NewExecutionError("stage", errors.New("failed"), true),
			expectedType: ErrorTypeExecution,
		},
		{
			name:         "timeout error",
			err:          NewTimeoutError("stage", "30s"),
			expectedType: ErrorTypeTimeout,
		},
		{
			name:         "cancellation error",
			err:          NewCancellationError("stage"),
			expectedType: ErrorTypeCancellation,
		},
		{
			name:         "fatal error",
			err:          NewFatalError("fatal", nil),
			expectedType: ErrorTypeFatal,
		},
		{
			name:         "dependency error",
			err:          NewDependencyError("stage2", "stage1", "not completed"),
			expectedType: ErrorTypeDependency,
		},
		{
			name:         "generic error defaults to execution",
			err:          errors.New("generic error"),
			expectedType: ErrorTypeExecution,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorType(tt.err)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

// TestWrapError tests the WrapError function
func TestWrapError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		stageID     string
		message     string
		wantNil     bool
		wantContains []string
	}{
		{
			name:     "wrap nil error",
			err:      nil,
			stageID:  "stage",
			message:  "context",
			wantNil:  true,
		},
		{
			name:    "wrap operation error enhances it",
			err:     NewValidationError("inner-stage", "validation failed"),
			stageID: "outer-stage",
			message: "during execution",
			wantNil: false,
			wantContains: []string{
				"validation",
				"inner-stage",
				"during execution",
				"validation failed",
			},
		},
		{
			name:    "wrap operation error with empty stage fills it",
			err:     &OperationError{Type: ErrorTypeExecution, Message: "original"},
			stageID: "new-stage",
			message: "context",
			wantNil: false,
			wantContains: []string{
				"execution",
				"new-stage",
				"context",
				"original",
			},
		},
		{
			name:    "wrap generic error",
			err:     errors.New("file not found"),
			stageID: "file-stage",
			message: "reading input",
			wantNil: false,
			wantContains: []string{
				"execution",
				"file-stage",
				"reading input",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.stageID, tt.message)
			
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			
			assert.Error(t, result)
			errorStr := result.Error()
			
			for _, contains := range tt.wantContains {
				assert.Contains(t, errorStr, contains)
			}
		})
	}
}

// TestErrorList tests the ErrorList functionality
func TestErrorList(t *testing.T) {
	t.Run("empty error list", func(t *testing.T) {
		el := &ErrorList{}
		assert.False(t, el.HasErrors())
		assert.Equal(t, "no errors", el.Error())
		assert.Empty(t, el.GetByStage("any-stage"))
	})

	t.Run("single error in list", func(t *testing.T) {
		el := &ErrorList{}
		err := NewValidationError("stage1", "validation failed")
		el.Add(err)
		
		assert.True(t, el.HasErrors())
		assert.Equal(t, err.Error(), el.Error())
		assert.Len(t, el.GetByStage("stage1"), 1)
		assert.Empty(t, el.GetByStage("stage2"))
	})

	t.Run("multiple errors in list", func(t *testing.T) {
		el := &ErrorList{}
		err1 := NewValidationError("stage1", "validation failed")
		err2 := NewExecutionError("stage2", errors.New("execution failed"), false)
		err3 := NewValidationError("stage1", "another validation error")
		
		el.Add(err1)
		el.Add(err2)
		el.Add(err3)
		el.Add(nil) // Should be ignored
		
		assert.True(t, el.HasErrors())
		assert.Contains(t, el.Error(), "multiple errors: 3 errors occurred")
		assert.Len(t, el.GetByStage("stage1"), 2)
		assert.Len(t, el.GetByStage("stage2"), 1)
		assert.Empty(t, el.GetByStage("stage3"))
	})
}

// TestCommonOperationErrors tests predefined operation errors
func TestCommonOperationErrors(t *testing.T) {
	t.Run("operation not found error", func(t *testing.T) {
		err := ErrOperationNotFound
		assert.Equal(t, ErrorTypeNotFound, err.Type)
		assert.Equal(t, "operation not found", err.Message)
		assert.Contains(t, err.Error(), "not_found")
	})

	t.Run("operation completed error", func(t *testing.T) {
		err := ErrOperationCompleted
		assert.Equal(t, ErrorTypeInvalidState, err.Type)
		assert.Equal(t, "operation has already completed", err.Message)
		assert.Contains(t, err.Error(), "invalid_state")
	})

	t.Run("operation not running error", func(t *testing.T) {
		err := ErrOperationNotRunning
		assert.Equal(t, ErrorTypeInvalidState, err.Type)
		assert.Equal(t, "operation is not running", err.Message)
		assert.Contains(t, err.Error(), "invalid_state")
	})
}

// TestErrorTypeDetection tests type detection of wrapped errors
func TestErrorTypeDetection(t *testing.T) {
	t.Run("detect operation error through wrapping", func(t *testing.T) {
		originalErr := NewValidationError("stage", "validation failed")
		wrappedErr := WrapError(originalErr, "wrapper", "context")
		
		var operationErr *OperationError
		assert.True(t, errors.As(wrappedErr, &operationErr))
		assert.Equal(t, "stage", operationErr.Step)
		assert.Equal(t, ErrorTypeValidation, operationErr.Type)
	})

	t.Run("detect error in chain", func(t *testing.T) {
		baseErr := errors.New("base error")
		execErr := WrapError(baseErr, "exec-stage", "execution context")
		wrappedErr := WrapError(execErr, "final-stage", "final context")
		
		// Should still be able to detect original error
		assert.True(t, errors.Is(wrappedErr, baseErr))
		
		// Should be an OperationError
		var operationErr *OperationError
		assert.True(t, errors.As(wrappedErr, &operationErr))
		assert.Equal(t, ErrorTypeExecution, operationErr.Type)
	})
}

// TestErrorEdgeCases tests edge cases and boundary conditions
func TestErrorEdgeCases(t *testing.T) {
	t.Run("empty string inputs", func(t *testing.T) {
		validationErr := NewValidationError("", "")
		execErr := NewExecutionError("", nil, false)
		timeoutErr := NewTimeoutError("", "")
		cancelErr := NewCancellationError("")
		fatalErr := NewFatalError("", nil)
		
		// Should not panic
		assert.NotNil(t, validationErr)
		assert.NotNil(t, execErr)
		assert.NotNil(t, timeoutErr)
		assert.NotNil(t, cancelErr)
		assert.NotNil(t, fatalErr)
		
		// Should still format correctly
		assert.Contains(t, validationErr.Error(), "[validation]")
		assert.Contains(t, execErr.Error(), "[execution]")
		assert.Contains(t, timeoutErr.Error(), "[timeout]")
		assert.Contains(t, cancelErr.Error(), "[cancellation]")
		assert.Contains(t, fatalErr.Error(), "[fatal]")
	})

	t.Run("very long error messages", func(t *testing.T) {
		longMessage := ""
		for i := 0; i < 10000; i++ {
			longMessage += "a"
		}
		
		err := NewValidationError("stage", longMessage)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), longMessage)
	})

	t.Run("nil error in wrap chain", func(t *testing.T) {
		// Wrapping nil should return nil
		result := WrapError(nil, "stage", "context")
		assert.Nil(t, result)
		
		// IsRetryable with nil should return false
		assert.False(t, IsRetryable(nil))
		
		// GetErrorType with nil should return empty
		assert.Equal(t, ErrorType(""), GetErrorType(nil))
	})
}