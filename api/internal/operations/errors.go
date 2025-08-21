package operations

import (
	"fmt"
)

// ErrorType represents the type of operation error
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeDependency   ErrorType = "dependency"
	ErrorTypeExecution    ErrorType = "execution"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeCancellation ErrorType = "cancellation"
	ErrorTypeRetryable    ErrorType = "retryable"
	ErrorTypeFatal        ErrorType = "fatal"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeInvalidState ErrorType = "invalid_state"
)

// OperationError represents a operation-specific error
type OperationError struct {
	Type      ErrorType              `json:"type"`
	Step     string                 `json:"Step,omitempty"`
	Message   string                 `json:"message"`
	Cause     error                  `json:"cause,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Retryable bool                   `json:"retryable"`
}

// Error implements the error interface
func (e *OperationError) Error() string {
	if e == nil {
		return "unknown operation error"
	}
	if e.Step != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Type, e.Step, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *OperationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(Step, message string) *OperationError {
	return &OperationError{
		Type:      ErrorTypeValidation,
		Step:     Step,
		Message:   message,
		Retryable: false,
	}
}

// NewDependencyError creates a new dependency error
func NewDependencyError(Step, dependsOn, message string) *OperationError {
	return &OperationError{
		Type:    ErrorTypeDependency,
		Step:   Step,
		Message: message,
		Context: map[string]interface{}{
			"depends_on": dependsOn,
		},
		Retryable: false,
	}
}

// NewExecutionError creates a new execution error
func NewExecutionError(Step string, cause error, retryable bool) *OperationError {
	return &OperationError{
		Type:      ErrorTypeExecution,
		Step:     Step,
		Message:   "Step execution failed",
		Cause:     cause,
		Retryable: retryable,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(Step string, timeout string) *OperationError {
	return &OperationError{
		Type:    ErrorTypeTimeout,
		Step:   Step,
		Message: fmt.Sprintf("Step exceeded timeout of %s", timeout),
		Context: map[string]interface{}{
			"timeout": timeout,
		},
		Retryable: true,
	}
}

// NewCancellationError creates a new cancellation error
func NewCancellationError(Step string) *OperationError {
	return &OperationError{
		Type:      ErrorTypeCancellation,
		Step:     Step,
		Message:   "operation was cancelled",
		Retryable: false,
	}
}

// NewFatalError creates a new fatal error
func NewFatalError(message string, cause error) *OperationError {
	return &OperationError{
		Type:      ErrorTypeFatal,
		Message:   message,
		Cause:     cause,
		Retryable: false,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if pErr, ok := err.(*OperationError); ok {
		return pErr.Retryable
	}
	return false
}

// GetErrorType returns the type of the error
func GetErrorType(err error) ErrorType {
	if err == nil {
		return ""
	}
	if pErr, ok := err.(*OperationError); ok {
		return pErr.Type
	}
	return ErrorTypeExecution
}

// WrapError wraps an error with operation context
func WrapError(err error, Step string, message string) *OperationError {
	if err == nil {
		return nil
	}
	
	// If it's already a OperationError, enhance it
	if pErr, ok := err.(*OperationError); ok {
		if pErr.Step == "" {
			pErr.Step = Step
		}
		if message != "" {
			pErr.Message = fmt.Sprintf("%s: %s", message, pErr.Message)
		}
		return pErr
	}
	
	// Otherwise create a new execution error
	return &OperationError{
		Type:      ErrorTypeExecution,
		Step:     Step,
		Message:   message,
		Cause:     err,
		Retryable: false,
	}
}

// ErrorList represents multiple errors
type ErrorList struct {
	Errors []*OperationError `json:"errors"`
}

// Error implements the error interface
func (e *ErrorList) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("multiple errors: %d errors occurred", len(e.Errors))
}

// Add adds an error to the list
func (e *ErrorList) Add(err *OperationError) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (e *ErrorList) HasErrors() bool {
	return len(e.Errors) > 0
}

// GetByStage returns errors for a specific Step
func (e *ErrorList) GetByStage(Step string) []*OperationError {
	var stageErrors []*OperationError
	for _, err := range e.Errors {
		if err.Step == Step {
			stageErrors = append(stageErrors, err)
		}
	}
	return stageErrors
}

// Common operation errors
var (
	// ErrOperationNotFound is returned when a operation cannot be found
	ErrOperationNotFound = &OperationError{
		Type:    ErrorTypeNotFound,
		Message: "operation not found",
	}

	// ErrOperationCompleted is returned when trying to modify a completed operation
	ErrOperationCompleted = &OperationError{
		Type:    ErrorTypeInvalidState,
		Message: "operation has already completed",
	}

	// ErrOperationNotRunning is returned when trying to stop a operation that's not running
	ErrOperationNotRunning = &OperationError{
		Type:    ErrorTypeInvalidState,
		Message: "operation is not running",
	}
)