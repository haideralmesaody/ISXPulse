package testutil

import (
	"context"
	"errors"
	"fmt"
	"time"

	"isxcli/internal/operations"
)

// CreateTestOperationState creates a operation state for testing
func CreateTestOperationState(id string) *operations.OperationState {
	state := operations.NewOperationState(id)
	state.SetConfig(operations.ContextKeyFromDate, "2024-01-01")
	state.SetConfig(operations.ContextKeyToDate, "2024-01-31")
	state.SetConfig(operations.ContextKeyMode, operations.ModeInitial)
	return state
}

// CreateTestStepState creates a step state for testing
func CreateTestStepState(id, name string) *operations.StepState {
	return operations.NewStepState(id, name)
}

// CreateTestConfig creates a test configuration
func CreateTestConfig() *operations.Config {
	return operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeSequential).
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
		}).
		WithStageTimeout(operations.StageIDScraping, 1*time.Second).
		WithStageTimeout(operations.StageIDProcessing, 1*time.Second).
		WithStageTimeout(operations.StageIDIndices, 1*time.Second).
		WithStageTimeout(operations.StageIDLiquidity, 1*time.Second).
		Build()
}

// CreateTestRegistry creates a registry with test steps
func CreateTestRegistry() *operations.Registry {
	registry := operations.NewRegistry()
	
	// Register test steps
	registry.Register(CreateSuccessfulStage("stage1", "step 1"))
	registry.Register(CreateSuccessfulStage("stage2", "step 2"))
	registry.Register(CreateSuccessfulStage("stage3", "step 3"))
	
	return registry
}

// CreateSuccessfulStage creates a step that always succeeds
func CreateSuccessfulStage(id, name string, deps ...string) *MockStage {
	return &MockStage{
		IDValue:           id,
		NameValue:         name,
		DependenciesValue: deps,
		ExecuteFunc: func(ctx context.Context, state *operations.OperationState) error {
			// Simulate some work
			StepState := state.GetStage(id)
			if StepState != nil {
				StepState.UpdateProgress(50, "Processing...")
				// Use context-aware timing instead of time.Sleep
				timer := time.NewTimer(10 * time.Millisecond)
				defer timer.Stop()
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-timer.C:
					// Continue processing
				}
				StepState.UpdateProgress(100, "Completed")
			}
			return nil
		},
	}
}

// CreateFailingStage creates a step that always fails
func CreateFailingStage(id, name string, err error, deps ...string) *MockStage {
	if err == nil {
		err = errors.New("step failed")
	}
	
	return &MockStage{
		IDValue:           id,
		NameValue:         name,
		DependenciesValue: deps,
		ExecuteFunc: func(ctx context.Context, state *operations.OperationState) error {
			return err
		},
	}
}

// CreateRetryableStage creates a step that fails then succeeds
func CreateRetryableStage(id, name string, failCount int, deps ...string) *MockStage {
	attempts := 0
	
	return &MockStage{
		IDValue:           id,
		NameValue:         name,
		DependenciesValue: deps,
		ExecuteFunc: func(ctx context.Context, state *operations.OperationState) error {
			attempts++
			if attempts <= failCount {
				return operations.NewExecutionError(id, errors.New("temporary failure"), true)
			}
			return nil
		},
	}
}

// CreateSlowStage creates a step that takes a specific duration
func CreateSlowStage(id, name string, duration time.Duration, deps ...string) *MockStage {
	return &MockStage{
		IDValue:           id,
		NameValue:         name,
		DependenciesValue: deps,
		ExecuteFunc: func(ctx context.Context, state *operations.OperationState) error {
			select {
			case <-time.After(duration):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}

// CreateValidationFailingStage creates a step that fails validation
func CreateValidationFailingStage(id, name string, validationErr error, deps ...string) *MockStage {
	if validationErr == nil {
		validationErr = errors.New("validation failed")
	}
	
	return &MockStage{
		IDValue:           id,
		NameValue:         name,
		DependenciesValue: deps,
		ValidateFunc: func(state *operations.OperationState) error {
			return validationErr
		},
	}
}

// CreateContextAwareStage creates a step that reads/writes context
func CreateContextAwareStage(id, name string, readKey, writeKey string, writeValue interface{}, deps ...string) *MockStage {
	return &MockStage{
		IDValue:           id,
		NameValue:         name,
		DependenciesValue: deps,
		ExecuteFunc: func(ctx context.Context, state *operations.OperationState) error {
			// Read from context if readKey is provided
			if readKey != "" {
				if val, ok := state.GetContext(readKey); ok {
					// Log or use the value
					_ = val
				}
			}
			
			// Write to context if writeKey is provided
			if writeKey != "" {
				state.SetContext(writeKey, writeValue)
			}
			
			return nil
		},
	}
}

// CreateComplexPipelineStages creates steps with complex dependencies
func CreateComplexPipelineStages() []operations.Step {
	// Create a diamond dependency pattern:
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	
	stageA := CreateSuccessfulStage("A", "step A")
	stageB := CreateSuccessfulStage("B", "step B", "A")
	stageC := CreateSuccessfulStage("C", "step C", "A")
	stageD := CreateSuccessfulStage("D", "step D", "B", "C")
	
	return []operations.Step{stageA, stageB, stageC, stageD}
}

// CreateOperationRequest creates a test operation request
func CreateOperationRequest(mode string) operations.OperationRequest {
	return operations.OperationRequest{
		ID:       fmt.Sprintf("test-operation-%d", time.Now().UnixNano()),
		Mode:     mode,
		FromDate: "2024-01-01",
		ToDate:   "2024-01-31",
		Parameters: map[string]interface{}{
			"test": true,
		},
	}
}

// StageBuilder provides a fluent interface for creating test steps
type StageBuilder struct {
	step *MockStage
}

// NewStageBuilder creates a new step builder
func NewStageBuilder(id, name string) *StageBuilder {
	return &StageBuilder{
		step: &MockStage{
			IDValue:   id,
			NameValue: name,
		},
	}
}

// WithDependencies sets the step dependencies
func (b *StageBuilder) WithDependencies(deps ...string) *StageBuilder {
	b.step.DependenciesValue = deps
	return b
}

// WithExecute sets the execute function
func (b *StageBuilder) WithExecute(fn func(context.Context, *operations.OperationState) error) *StageBuilder {
	b.step.ExecuteFunc = fn
	return b
}

// WithValidate sets the validate function
func (b *StageBuilder) WithValidate(fn func(*operations.OperationState) error) *StageBuilder {
	b.step.ValidateFunc = fn
	return b
}

// Build returns the constructed step
func (b *StageBuilder) Build() *MockStage {
	return b.step
}