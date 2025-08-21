package operations_test

import (
	"errors"
	"fmt"
	"testing"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

func TestOperationErrorUnwrap(t *testing.T) {
	tests := []struct {
		name          string
		OperationError *operations.OperationError
		expectedCause error
	}{
		{
			name: "error with cause",
			OperationError: &operations.OperationError{
				Type:    operations.ErrorTypeExecution,
				Step:   "test-Step",
				Message: "execution failed",
				Cause:   errors.New("underlying error"),
			},
			expectedCause: errors.New("underlying error"),
		},
		{
			name: "error without cause",
			OperationError: &operations.OperationError{
				Type:    operations.ErrorTypeValidation,
				Step:   "test-Step",
				Message: "validation failed",
				Cause:   nil,
			},
			expectedCause: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := tt.OperationError.Unwrap()
			
			if tt.expectedCause == nil {
				if unwrapped != nil {
					t.Errorf("Unwrap() = %v, want nil", unwrapped)
				}
			} else {
				if unwrapped == nil {
					t.Errorf("Unwrap() = nil, want %v", tt.expectedCause)
				} else if unwrapped.Error() != tt.expectedCause.Error() {
					t.Errorf("Unwrap() = %v, want %v", unwrapped, tt.expectedCause)
				}
			}
		})
	}
}

func TestNewDependencyError(t *testing.T) {
	tests := []struct {
		name      string
		Step     string
		dependsOn string
		message   string
		expected  *operations.OperationError
	}{
		{
			name:      "basic dependency error",
			Step:     "Step-b",
			dependsOn: "Step-a",
			message:   "Step-a must complete first",
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeDependency,
				Step:     "Step-b",
				Message:   "Step-a must complete first",
				Retryable: false,
				Context: map[string]interface{}{
					"depends_on": "Step-a",
				},
			},
		},
		{
			name:      "empty dependency name",
			Step:     "Step-c",
			dependsOn: "",
			message:   "missing dependency",
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeDependency,
				Step:     "Step-c",
				Message:   "missing dependency",
				Retryable: false,
				Context: map[string]interface{}{
					"depends_on": "",
				},
			},
		},
		{
			name:      "complex dependency message",
			Step:     "processing",
			dependsOn: "scraping",
			message:   "processing requires successful scraping completion with valid data",
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeDependency,
				Step:     "processing",
				Message:   "processing requires successful scraping completion with valid data",
				Retryable: false,
				Context: map[string]interface{}{
					"depends_on": "scraping",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := operations.NewDependencyError(tt.Step, tt.dependsOn, tt.message)
			
			testutil.AssertEqual(t, err.Type, tt.expected.Type)
			testutil.AssertEqual(t, err.Step, tt.expected.Step)
			testutil.AssertEqual(t, err.Message, tt.expected.Message)
			testutil.AssertEqual(t, err.Retryable, tt.expected.Retryable)
			
			if err.Context == nil {
				t.Error("NewDependencyError() should set Context")
			} else {
				dependsOn, ok := err.Context["depends_on"]
				if !ok {
					t.Error("NewDependencyError() Context should contain 'depends_on' key")
				} else {
					testutil.AssertEqual(t, dependsOn, tt.dependsOn)
				}
			}
		})
	}
}

func TestNewFatalError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		cause    error
		expected *operations.OperationError
	}{
		{
			name:    "fatal error with cause",
			message: "system initialization failed",
			cause:   errors.New("database connection failed"),
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeFatal,
				Message:   "system initialization failed",
				Retryable: false,
			},
		},
		{
			name:    "fatal error without cause",
			message: "unrecoverable error occurred",
			cause:   nil,
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeFatal,
				Message:   "unrecoverable error occurred",
				Retryable: false,
			},
		},
		{
			name:    "global system failure",
			message: "global system failure",
			cause:   fmt.Errorf("config parse error"),
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeFatal,
				Message:   "global system failure",
				Retryable: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := operations.NewFatalError(tt.message, tt.cause)
			
			testutil.AssertEqual(t, err.Type, tt.expected.Type)
			testutil.AssertEqual(t, err.Message, tt.expected.Message)
			testutil.AssertEqual(t, err.Retryable, tt.expected.Retryable)
			testutil.AssertEqual(t, err.Cause, tt.cause)
		})
	}
}

func TestNewCancellationError(t *testing.T) {
	tests := []struct {
		name     string
		Step    string
		expected *operations.OperationError
	}{
		{
			name:    "basic cancellation error",
			Step:   "long-running-Step",
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeCancellation,
				Step:     "long-running-Step",
				Message:   "operation was cancelled",
				Retryable: false,
			},
		},
		{
			name:    "timeout cancellation",
			Step:   "slow-Step",
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeCancellation,
				Step:     "slow-Step",
				Message:   "operation was cancelled",
				Retryable: false,
			},
		},
		{
			name:    "empty Step cancellation",
			Step:   "",
			expected: &operations.OperationError{
				Type:      operations.ErrorTypeCancellation,
				Step:     "",
				Message:   "operation was cancelled",
				Retryable: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := operations.NewCancellationError(tt.Step)
			
			testutil.AssertEqual(t, err.Type, tt.expected.Type)
			testutil.AssertEqual(t, err.Step, tt.expected.Step)
			testutil.AssertEqual(t, err.Message, tt.expected.Message)
			testutil.AssertEqual(t, err.Retryable, tt.expected.Retryable)
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType operations.ErrorType
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedType: "",
		},
		{
			name: "operation validation error",
			err: &operations.OperationError{
				Type:    operations.ErrorTypeValidation,
				Step:   "test-Step",
				Message: "validation failed",
			},
			expectedType: operations.ErrorTypeValidation,
		},
		{
			name: "operation dependency error",
			err: &operations.OperationError{
				Type:    operations.ErrorTypeDependency,
				Step:   "dependent-Step",
				Message: "dependency not met",
			},
			expectedType: operations.ErrorTypeDependency,
		},
		{
			name: "operation execution error",
			err: &operations.OperationError{
				Type:    operations.ErrorTypeExecution,
				Step:   "exec-Step",
				Message: "execution failed",
			},
			expectedType: operations.ErrorTypeExecution,
		},
		{
			name: "operation timeout error",
			err: &operations.OperationError{
				Type:    operations.ErrorTypeTimeout,
				Step:   "slow-Step",
				Message: "operation timed out",
			},
			expectedType: operations.ErrorTypeTimeout,
		},
		{
			name: "operation cancellation error",
			err: &operations.OperationError{
				Type:    operations.ErrorTypeCancellation,
				Step:   "cancelled-Step",
				Message: "operation cancelled",
			},
			expectedType: operations.ErrorTypeCancellation,
		},
		{
			name: "operation fatal error",
			err: &operations.OperationError{
				Type:    operations.ErrorTypeFatal,
				Step:   "critical-Step",
				Message: "fatal error occurred",
			},
			expectedType: operations.ErrorTypeFatal,
		},
		{
			name:         "regular Go error",
			err:          errors.New("regular error"),
			expectedType: operations.ErrorTypeExecution, // Default for non-operation errors
		},
		{
			name:         "fmt error",
			err:          fmt.Errorf("formatted error: %s", "details"),
			expectedType: operations.ErrorTypeExecution,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := operations.GetErrorType(tt.err)
			testutil.AssertEqual(t, errorType, tt.expectedType)
		})
	}
}

func TestErrorListError(t *testing.T) {
	tests := []struct {
		name       string
		errorList  *operations.ErrorList
		expected   string
	}{
		{
			name: "single error",
			errorList: &operations.ErrorList{
				Errors: []*operations.OperationError{
					{
						Type:    operations.ErrorTypeExecution,
						Step:   "stage1",
						Message: "execution failed",
					},
				},
			},
			expected: "[execution] stage1: execution failed",
		},
		{
			name: "multiple errors",
			errorList: &operations.ErrorList{
				Errors: []*operations.OperationError{
					{
						Type:    operations.ErrorTypeValidation,
						Step:   "stage1",
						Message: "validation failed",
					},
					{
						Type:    operations.ErrorTypeTimeout,
						Step:   "stage2",
						Message: "operation timed out",
					},
				},
			},
			expected: "multiple errors: 2 errors occurred",
		},
		{
			name: "no errors",
			errorList: &operations.ErrorList{
				Errors: []*operations.OperationError{},
			},
			expected: "no errors",
		},
		{
			name: "nil errors slice",
			errorList: &operations.ErrorList{
				Errors: nil,
			},
			expected: "no errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errorList.Error()
			testutil.AssertEqual(t, result, tt.expected)
		})
	}
}

func TestErrorListAdd(t *testing.T) {
	errorList := &operations.ErrorList{}
	
	// Add first error
	err1 := operations.NewValidationError("stage1", "validation failed")
	errorList.Add(err1)
	
	testutil.AssertEqual(t, len(errorList.Errors), 1)
	testutil.AssertEqual(t, errorList.Errors[0], err1)
	
	// Add second error  
	err2 := operations.NewExecutionError("stage2", errors.New("exec failed"), true)
	errorList.Add(err2)
	
	testutil.AssertEqual(t, len(errorList.Errors), 2)
	testutil.AssertEqual(t, errorList.Errors[1], err2)
	
	// Add nil error (should be ignored)
	errorList.Add(nil)
	testutil.AssertEqual(t, len(errorList.Errors), 2) // Should remain 2
}

func TestErrorListHasErrors(t *testing.T) {
	tests := []struct {
		name       string
		collection *operations.ErrorList
		expected   bool
	}{
		{
			name: "has errors",
			collection: &operations.ErrorList{
				Errors: []*operations.OperationError{
					operations.NewValidationError("stage1", "validation failed"),
				},
			},
			expected: true,
		},
		{
			name: "no errors",
			collection: &operations.ErrorList{
				Errors: []*operations.OperationError{},
			},
			expected: false,
		},
		{
			name: "nil errors slice",
			collection: &operations.ErrorList{
				Errors: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.collection.HasErrors()
			testutil.AssertEqual(t, result, tt.expected)
		})
	}
}

func TestErrorListGetByStage(t *testing.T) {
	errorList := &operations.ErrorList{
		Errors: []*operations.OperationError{
			operations.NewValidationError("stage1", "validation failed"),
			operations.NewExecutionError("stage2", errors.New("exec failed"), true),
			operations.NewTimeoutError("stage1", "operation timed out"),
			operations.NewDependencyError("stage3", "stage2", "dependency failed"),
		},
	}

	tests := []struct {
		name          string
		Step         string
		expectedCount int
	}{
		{
			name:          "Step with multiple errors",
			Step:         "stage1",
			expectedCount: 2, // validation and timeout errors
		},
		{
			name:          "Step with single error",
			Step:         "stage2",
			expectedCount: 1,
		},
		{
			name:          "Step with single error (dependency)",
			Step:         "stage3",
			expectedCount: 1,
		},
		{
			name:          "Step with no errors",
			Step:         "stage4",
			expectedCount: 0,
		},
		{
			name:          "empty Step name",
			Step:         "",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := errorList.GetByStage(tt.Step)
			testutil.AssertEqual(t, len(errors), tt.expectedCount)
			
			// Verify all returned errors are for the requested Step
			for _, err := range errors {
				testutil.AssertEqual(t, err.Step, tt.Step)
			}
		})
	}
}