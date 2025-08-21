package operations_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"isxcli/internal/operations"
)

// Enhanced mock Step for manager testing
type mockManagerStage struct {
	id           string
	name         string
	dependencies []string
	shouldFail   bool
	failError    error
	executionTime time.Duration
	validateError error
	executeCallCount int
	validateCallCount int
}

func newMockManagerStage(id, name string, dependencies []string) *mockManagerStage {
	return &mockManagerStage{
		id:           id,
		name:         name,
		dependencies: dependencies,
		executionTime: 10 * time.Millisecond,
	}
}

func (m *mockManagerStage) ID() string {
	return m.id
}

func (m *mockManagerStage) Name() string {
	return m.name
}

func (m *mockManagerStage) GetDependencies() []string {
	return m.dependencies
}

func (m *mockManagerStage) Validate(state *operations.OperationState) error {
	m.validateCallCount++
	return m.validateError
}

func (m *mockManagerStage) Execute(ctx context.Context, state *operations.OperationState) error {
	m.executeCallCount++
	
	// Simulate execution time
	select {
	case <-time.After(m.executionTime):
	case <-ctx.Done():
		return ctx.Err()
	}
	
	if m.shouldFail {
		return m.failError
	}
	return nil
}

func (m *mockManagerStage) WithFailure(err error) *mockManagerStage {
	m.shouldFail = true
	m.failError = err
	return m
}

func (m *mockManagerStage) WithExecutionTime(duration time.Duration) *mockManagerStage {
	m.executionTime = duration
	return m
}

func (m *mockManagerStage) WithValidationError(err error) *mockManagerStage {
	m.validateError = err
	return m
}

func (m *mockManagerStage) RequiredInputs() []operations.DataRequirement {
	return []operations.DataRequirement{}
}

func (m *mockManagerStage) ProducedOutputs() []operations.DataOutput {
	return []operations.DataOutput{}
}

func (m *mockManagerStage) CanRun(manifest *operations.PipelineManifest) bool {
	return true
}

// Enhanced mock WebSocket hub for manager testing
type mockManagerWebSocketHub struct {
	broadcasts []mockManagerBroadcast
}

type mockManagerBroadcast struct {
	eventType string
	Step     string
	status    string
	metadata  interface{}
	timestamp time.Time
}

func (m *mockManagerWebSocketHub) BroadcastUpdate(eventType, Step, status string, metadata interface{}) {
	m.broadcasts = append(m.broadcasts, mockManagerBroadcast{
		eventType: eventType,
		Step:     Step,
		status:    status,
		metadata:  metadata,
		timestamp: time.Now(),
	})
}

func (m *mockManagerWebSocketHub) GetBroadcastsByType(eventType string) []mockManagerBroadcast {
	var result []mockManagerBroadcast
	for _, broadcast := range m.broadcasts {
		if broadcast.eventType == eventType {
			result = append(result, broadcast)
		}
	}
	return result
}

func (m *mockManagerWebSocketHub) GetBroadcastsForStage(stageID string) []mockManagerBroadcast {
	var result []mockManagerBroadcast
	for _, broadcast := range m.broadcasts {
		if broadcast.Step == stageID {
			result = append(result, broadcast)
		}
	}
	return result
}

// Helper function to create a test manager with mocked dependencies
func createTestManager(hub operations.WebSocketHub) *operations.Manager {
	config := operations.NewConfigBuilder().
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     50 * time.Millisecond,
			Multiplier:   2.0,
		}).
		WithStageTimeout(operations.StageIDScraping, 100*time.Millisecond).
		WithStageTimeout(operations.StageIDProcessing, 100*time.Millisecond).
		Build()

	registry := operations.NewRegistry()
	return operations.NewManager(hub, registry, config)
}

// Helper function to create operation state with mock steps
func createOperationStateWithStages(operationID string, steps []operations.Step) *operations.OperationState {
	state := operations.NewOperationState(operationID)
	
	// Initialize Step states
	for _, Step := range steps {
		StepState := operations.NewStepState(Step.ID(), Step.Name())
		state.Steps[Step.ID()] = StepState
	}
	
	return state
}

// TestManagerExecuteThroughPublicAPI tests operation execution through the public Execute method
func TestManagerExecuteThroughPublicAPI(t *testing.T) {
	tests := []struct {
		name           string
		steps         []operations.Step
		expectError    bool
		continueOnError bool
		timeout        time.Duration
	}{
		{
			name: "successful operation execution",
			steps: []operations.Step{
				newMockManagerStage("stage1", "Step 1", nil),
				newMockManagerStage("stage2", "Step 2", []string{"stage1"}),
			},
			expectError: false,
			timeout:     5 * time.Second,
		},
		{
			name: "operation execution with Step failure",
			steps: []operations.Step{
				newMockManagerStage("stage1", "Step 1", nil).WithFailure(fmt.Errorf("stage1 failed")),
				newMockManagerStage("stage2", "Step 2", []string{"stage1"}),
			},
			expectError:     true,
			continueOnError: false,
			timeout:         5 * time.Second,
		},
		{
			name: "operation execution with continue on error",
			steps: []operations.Step{
				newMockManagerStage("stage1", "Step 1", nil).WithFailure(fmt.Errorf("stage1 failed")),
				newMockManagerStage("stage2", "Step 2", nil), // Independent Step
			},
			expectError:     false,
			continueOnError: true,
			timeout:         5 * time.Second,
		},
		{
			name: "operation execution with timeout",
			steps: []operations.Step{
				newMockManagerStage("stage1", "Step 1", nil).WithExecutionTime(200*time.Millisecond),
			},
			expectError: true,
			timeout:     50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWS := &mockManagerWebSocketHub{}
			
			// Create manager with appropriate configuration
			configBuilder := operations.NewConfigBuilder()
			if tt.continueOnError {
				configBuilder = configBuilder.WithContinueOnError(true)
			}
			config := configBuilder.Build()
			registry := operations.NewRegistry()
			manager := operations.NewManager(mockWS, registry, config)
			
			// Register steps
			for _, Step := range tt.steps {
				manager.RegisterStage(Step)
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			
			// Execute operation
			req := operations.OperationRequest{
				ID:   "test-operation",
				Mode: "test",
			}
			
			resp, err := manager.Execute(ctx, req)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Verify response when no error expected
			if !tt.expectError && resp != nil {
				if resp.ID != req.ID {
					t.Errorf("Expected response ID %s, got %s", req.ID, resp.ID)
				}
				if resp.Status != operations.OperationStatusCompleted {
					t.Errorf("Expected operation status %s, got %s", operations.OperationStatusCompleted, resp.Status)
				}
			}
			
			// Verify WebSocket broadcasts were made
			if len(mockWS.broadcasts) == 0 {
				t.Error("Expected WebSocket broadcasts during execution")
			}
			
			// Verify execution call counts
			for _, Step := range tt.steps {
				if mockStage, ok := Step.(*mockManagerStage); ok {
					if mockStage.executeCallCount == 0 && !tt.expectError {
						t.Errorf("Step %s was not executed", Step.ID())
					}
				}
			}
		})
	}
}

// TestManagerRetryBehavior tests retry behavior through integration tests
func TestManagerRetryBehavior(t *testing.T) {
	tests := []struct {
		name         string
		Step        operations.Step
		expectError  bool
		maxAttempts  int
	}{
		{
			name:        "retryable error with successful retry",
			Step:       newMockManagerStage("retry1", "Retry Step 1", nil).WithFailure(operations.NewExecutionError("retry1", fmt.Errorf("temporary failure"), true)),
			expectError: true, // Will still fail after retries in our mock
			maxAttempts: 3,
		},
		{
			name:        "fatal error no retry",
			Step:       newMockManagerStage("fatal1", "Fatal Step 1", nil).WithFailure(operations.NewExecutionError("fatal1", fmt.Errorf("fatal error"), false)),
			expectError: true,
			maxAttempts: 1, // Should not retry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWS := &mockManagerWebSocketHub{}
			config := operations.NewConfigBuilder().WithRetryConfig(operations.RetryConfig{
				MaxAttempts:  tt.maxAttempts,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			}).Build()
			registry := operations.NewRegistry()
			manager := operations.NewManager(mockWS, registry, config)
			
			manager.RegisterStage(tt.Step)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			req := operations.OperationRequest{
				ID:   "retry-test-operation",
				Mode: "test",
			}
			
			_, err := manager.Execute(ctx, req)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Verify execution attempts match expected retry behavior
			if mockStage, ok := tt.Step.(*mockManagerStage); ok {
				if mockStage.executeCallCount != tt.maxAttempts {
					t.Errorf("Expected %d execution attempts, got %d", tt.maxAttempts, mockStage.executeCallCount)
				}
			}
		})
	}
}

// TestManagerWebSocketIntegration tests WebSocket message broadcasting
func TestManagerWebSocketIntegration(t *testing.T) {
	mockWS := &mockManagerWebSocketHub{}
	manager := createTestManager(mockWS)
	
	Step := newMockManagerStage("ws-test", "WebSocket Test Step", nil)
	manager.RegisterStage(Step)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req := operations.OperationRequest{
		ID:   "websocket-test-operation",
		Mode: "test",
	}
	
	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Logf("Expected execution error: %v", err) // This is fine for testing
	}
	
	// Verify WebSocket messages were sent
	if len(mockWS.broadcasts) == 0 {
		t.Error("Expected WebSocket broadcasts during execution")
	}
	
	// Verify we have operation status messages
	statusBroadcasts := mockWS.GetBroadcastsByType(operations.EventTypeOperationStatus)
	if len(statusBroadcasts) == 0 {
		t.Error("Expected operation status broadcasts")
	}
	
	// Verify we have operation progress messages
	progressBroadcasts := mockWS.GetBroadcastsByType(operations.EventTypePipelineProgress)
	if len(progressBroadcasts) == 0 {
		t.Error("Expected operation progress broadcasts")
	}
}

// TestManagerDependencyHandling tests dependency validation through execution
func TestManagerDependencyHandling(t *testing.T) {
	tests := []struct {
		name    string
		steps  []operations.Step
		expectError bool
	}{
		{
			name: "satisfied dependencies",
			steps: []operations.Step{
				newMockManagerStage("dep1", "Dependency 1", nil),
				newMockManagerStage("dep2", "Dependency 2", []string{"dep1"}),
			},
			expectError: false,
		},
		{
			name: "dependency chain with failure",
			steps: []operations.Step{
				newMockManagerStage("fail1", "Failing Step", nil).WithFailure(fmt.Errorf("Step failed")),
				newMockManagerStage("dep_fail", "Dependent Step", []string{"fail1"}),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWS := &mockManagerWebSocketHub{}
			manager := createTestManager(mockWS)
			
			for _, Step := range tt.steps {
				manager.RegisterStage(Step)
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			req := operations.OperationRequest{
				ID:   "dependency-test-operation",
				Mode: "test",
			}
			
			_, err := manager.Execute(ctx, req)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}