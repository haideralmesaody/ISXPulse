package operations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// SimpleMockWebSocketHub is a simple mock for testing
type SimpleMockWebSocketHub struct {
	messages []string
}

func (m *SimpleMockWebSocketHub) BroadcastUpdate(eventType, step, status string, metadata interface{}) {
	m.messages = append(m.messages, eventType)
}

// SimpleMockStage is a simple mock stage for testing
type SimpleMockStage struct {
	id           string
	name         string
	dependencies []string
	executeFunc  func(ctx context.Context, state *OperationState) error
	validateFunc func(state *OperationState) error
	executeCalls int
}

func (s *SimpleMockStage) ID() string                               { return s.id }
func (s *SimpleMockStage) Name() string                             { return s.name }
func (s *SimpleMockStage) GetDependencies() []string                { return s.dependencies }
func (s *SimpleMockStage) Validate(state *OperationState) error {
	if s.validateFunc != nil {
		return s.validateFunc(state)
	}
	return nil
}
func (s *SimpleMockStage) Execute(ctx context.Context, state *OperationState) error {
	s.executeCalls++
	if s.executeFunc != nil {
		return s.executeFunc(ctx, state)
	}
	return nil
}
func (s *SimpleMockStage) RequiredInputs() []DataRequirement { return []DataRequirement{} }
func (s *SimpleMockStage) ProducedOutputs() []DataOutput { return []DataOutput{} }
func (s *SimpleMockStage) CanRun(manifest *PipelineManifest) bool { return true }

// TestManagerCalculateRetryDelay tests the retry delay calculation function
func TestManagerCalculateRetryDelay(t *testing.T) {
	hub := &SimpleMockWebSocketHub{}
	manager := NewManager(hub, nil, nil)

	config := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}

	// Test first attempt (attempt 1)
	delay1 := manager.calculateRetryDelay(1, config)
	assert.Equal(t, 0*time.Second, delay1) // Should be 0 for first attempt

	// Test second attempt (attempt 2)  
	delay2 := manager.calculateRetryDelay(2, config)
	assert.Equal(t, 1*time.Second, delay2)

	// Test with high attempt number to test MaxDelay cap
	delay10 := manager.calculateRetryDelay(10, config)
	assert.Equal(t, 30*time.Second, delay10) // Should be capped at MaxDelay

	// Test with zero multiplier
	configZeroMultiplier := RetryConfig{
		InitialDelay: 5 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   0.0,
	}
	delayZero := manager.calculateRetryDelay(3, configZeroMultiplier)
	assert.Equal(t, 0*time.Second, delayZero) // Should be 0 with zero multiplier
}

// TestManagerValidationFailure tests validation failure path
func TestManagerValidationFailure(t *testing.T) {
	hub := &SimpleMockWebSocketHub{}
	manager := NewManager(hub, nil, nil)

	// Create stage that fails validation
	stage := &SimpleMockStage{
		id:   "validation-fail-test",
		name: "Validation Fail Test Stage",
		validateFunc: func(state *OperationState) error {
			return errors.New("validation failed")
		},
	}
	manager.RegisterStage(stage)

	ctx := context.Background()
	req := OperationRequest{ID: "validation-fail-test"}

	_, err := manager.Execute(ctx, req)
	assert.Error(t, err) // Should fail due to validation error
	assert.Equal(t, 0, stage.executeCalls) // Execute should not be called
}

// TestManagerSkipDependentStages tests the skipDependentStages path
func TestManagerSkipDependentStages(t *testing.T) {
	hub := &SimpleMockWebSocketHub{}
	manager := NewManager(hub, nil, nil)

	// Configure to NOT continue on error
	config := NewConfig()
	config.ContinueOnError = false
	manager.SetConfig(config)

	// Create stages with dependency chain where middle stage fails
	stage1 := &SimpleMockStage{id: "dep-stage1", name: "Dep Stage 1"}
	stage2 := &SimpleMockStage{
		id:           "dep-stage2",
		name:         "Dep Stage 2",
		dependencies: []string{"dep-stage1"},
		executeFunc: func(ctx context.Context, state *OperationState) error {
			return errors.New("stage 2 failed")
		},
	}
	stage3 := &SimpleMockStage{
		id:           "dep-stage3", 
		name:         "Dep Stage 3",
		dependencies: []string{"dep-stage2"},
	}

	manager.RegisterStage(stage1)
	manager.RegisterStage(stage2)
	manager.RegisterStage(stage3)

	ctx := context.Background()
	req := OperationRequest{ID: "skip-dependent-test"}
	
	resp, err := manager.Execute(ctx, req)

	// Should fail and have skipped dependent stages
	assert.Error(t, err)
	assert.Equal(t, OperationStatusFailed, resp.Status)

	// stage1 should have run successfully
	assert.Equal(t, 1, stage1.executeCalls)
	// stage2 should have run and failed
	assert.Equal(t, 1, stage2.executeCalls)
	// stage3 should have been skipped (not executed)
	assert.Equal(t, 0, stage3.executeCalls)
}

// TestManagerConfigNilHandling tests nil config handling
func TestManagerConfigNilHandling(t *testing.T) {
	hub := &SimpleMockWebSocketHub{}
	manager := NewManager(hub, nil, nil)

	// Test setting nil config doesn't change anything
	originalConfig := manager.GetConfig()
	manager.SetConfig(nil)
	assert.Equal(t, originalConfig, manager.GetConfig())
}

// TestConfigStageTimeoutEdgeCases tests stage timeout configuration edge cases
func TestConfigStageTimeoutEdgeCases(t *testing.T) {
	// Test with nil StageTimeouts map
	config := &Config{
		ExecutionMode: ExecutionModeSequential,
		// Leave StageTimeouts nil
	}

	// SetStageTimeout should handle nil map
	config.SetStageTimeout("test-stage", 30*time.Second)
	assert.NotNil(t, config.StageTimeouts)
	assert.Equal(t, 30*time.Second, config.GetStageTimeout("test-stage"))

	// Test with non-existent stage
	timeout := config.GetStageTimeout("non-existent")
	assert.Equal(t, DefaultStageTimeout, timeout)
}

// TestConfigStepConfigEdgeCases tests step config edge cases
func TestConfigStepConfigEdgeCases(t *testing.T) {
	config := &Config{
		ExecutionMode: ExecutionModeSequential,
		// Leave StepConfigs nil
	}

	// SetStepConfig should handle nil map
	config.SetStepConfig("test-stage", "test-config")
	assert.NotNil(t, config.StepConfigs)
	value, exists := config.GetStepConfig("test-stage")
	assert.True(t, exists)
	assert.Equal(t, "test-config", value)

	// GetStepConfig with nil map
	config.StepConfigs = nil
	value, exists = config.GetStepConfig("any-stage")
	assert.Nil(t, value)
	assert.False(t, exists)
}

// TestRegistryDependencyOrder tests the registry dependency ordering
func TestRegistryDependencyOrder(t *testing.T) {
	registry := NewRegistry()

	// Create stages with dependencies
	stageA := &SimpleMockStage{id: "A", name: "Stage A"}
	stageB := &SimpleMockStage{id: "B", name: "Stage B", dependencies: []string{"A"}}
	stageC := &SimpleMockStage{id: "C", name: "Stage C", dependencies: []string{"A"}}
	stageD := &SimpleMockStage{id: "D", name: "Stage D", dependencies: []string{"B", "C"}}

	// Register in random order
	assert.NoError(t, registry.Register(stageD))
	assert.NoError(t, registry.Register(stageA))
	assert.NoError(t, registry.Register(stageC))
	assert.NoError(t, registry.Register(stageB))

	// Test duplicate registration
	err := registry.Register(stageA)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test Has method
	assert.True(t, registry.Has("A"))
	assert.False(t, registry.Has("non-existent"))

	// Test Get
	retrievedStage, err := registry.Get("A")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedStage)
	assert.Equal(t, "A", retrievedStage.ID())

	_, err = registry.Get("non-existent")
	assert.Error(t, err)

	// Test List
	allStages := registry.List()
	assert.Len(t, allStages, 4)

	// Get dependency order
	stages, err := registry.GetDependencyOrder()
	assert.NoError(t, err)
	assert.Len(t, stages, 4)

	// Verify A comes before B and C
	// B and C come before D
	indexA := findStageIndex(stages, "A")
	indexB := findStageIndex(stages, "B")
	indexC := findStageIndex(stages, "C")
	indexD := findStageIndex(stages, "D")

	assert.True(t, indexA < indexB)
	assert.True(t, indexA < indexC)
	assert.True(t, indexB < indexD)
	assert.True(t, indexC < indexD)
}

// TestManagerTimeoutHandling tests timeout handling in manager
func TestManagerTimeoutHandling(t *testing.T) {
	hub := &SimpleMockWebSocketHub{}
	manager := NewManager(hub, nil, nil)

	// Configure very short timeout
	config := NewConfig()
	config.SetStageTimeout("timeout-test", 10*time.Millisecond)
	manager.SetConfig(config)

	// Create stage that takes longer than timeout
	stage := &SimpleMockStage{
		id:   "timeout-test",
		name: "Timeout Test Stage",
		executeFunc: func(ctx context.Context, state *OperationState) error {
			// Sleep longer than timeout
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}
	manager.RegisterStage(stage)

	ctx := context.Background()
	req := OperationRequest{ID: "timeout-test"}

	resp, err := manager.Execute(ctx, req)

	// Should fail with timeout
	assert.Error(t, err)
	assert.Equal(t, OperationStatusFailed, resp.Status)
}

// TestManagerRetryWithNonRetryableError tests retry logic with non-retryable errors
func TestManagerRetryWithNonRetryableError(t *testing.T) {
	hub := &SimpleMockWebSocketHub{}
	manager := NewManager(hub, nil, nil)

	// Configure retries
	config := NewConfig()
	config.RetryConfig.MaxAttempts = 3
	config.RetryConfig.InitialDelay = 1 * time.Millisecond
	manager.SetConfig(config)

	// Create stage that fails with non-retryable error
	stage := &SimpleMockStage{
		id:   "non-retryable",
		name: "Non-retryable Stage",
		executeFunc: func(ctx context.Context, state *OperationState) error {
			return NewFatalError("fatal error", nil) // Non-retryable
		},
	}
	manager.RegisterStage(stage)

	ctx := context.Background()
	req := OperationRequest{ID: "non-retryable-test"}

	resp, err := manager.Execute(ctx, req)

	// Should fail without retries
	assert.Error(t, err)
	assert.Equal(t, OperationStatusFailed, resp.Status)
	assert.Equal(t, 1, stage.executeCalls) // Should only be called once (no retries)
}

// Helper function to find stage index in slice
func findStageIndex(stages []Step, stageID string) int {
	for i, stage := range stages {
		if stage.ID() == stageID {
			return i
		}
	}
	return -1
}