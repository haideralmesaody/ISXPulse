package operations_test

import (
	"context"
	"testing"
	"time"

	"isxcli/internal/operations"
)

// TestManagerConfigurationBasics tests basic configuration functionality
func TestManagerConfigurationBasics(t *testing.T) {
	mockWS := &mockManagerWebSocketHub{}
	
	// Test with default config
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Verify initial configuration
	if config.ExecutionMode != operations.ExecutionModeSequential {
		t.Errorf("Expected default execution mode sequential, got %v", config.ExecutionMode)
	}

	// Test SetConfig
	newConfig := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeParallel).
		WithContinueOnError(true).
		Build()

	manager.SetConfig(newConfig)

	updatedConfig := manager.GetConfig()
	if updatedConfig.ExecutionMode != operations.ExecutionModeParallel {
		t.Errorf("Expected updated execution mode parallel, got %v", updatedConfig.ExecutionMode)
	}
}

// TestManagerExecuteParallelFallback tests that parallel execution falls back to sequential  
func TestManagerExecuteParallelFallback(t *testing.T) {
	mockWS := &mockManagerWebSocketHub{}
	
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeParallel).
		WithMaxConcurrency(2).
		Build()
	
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a simple Step that will succeed
	Step := newMockManagerStage("parallel-test", "Parallel Test Step", nil)
	manager.RegisterStage(Step)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := operations.OperationRequest{
		ID:   "parallel-fallback-test",
		Mode: "test",
	}

	_, err := manager.Execute(ctx, req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify Step was executed
	if Step.executeCallCount == 0 {
		t.Error("Step should have been executed")
	}

	// Note: Currently executeParallel falls back to sequential execution
	t.Log("Parallel execution completed (currently falls back to sequential)")
}

// TestManagerConfigBuilderDetailed tests the configuration builder pattern
func TestManagerConfigBuilderDetailed(t *testing.T) {
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeParallel).
		WithStageTimeout("custom-Step", 45*time.Second).
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  4,
			InitialDelay: 200 * time.Millisecond,
			MaxDelay:     3 * time.Second,
			Multiplier:   1.6,
		}).
		WithContinueOnError(true).
		WithMaxConcurrency(5).
		WithCheckpoints(true, "custom/checkpoint/dir").
		WithStepConfig("custom-Step", map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		}).
		Build()

	// Verify all configuration values
	if config.ExecutionMode != operations.ExecutionModeParallel {
		t.Errorf("Expected parallel execution mode, got %v", config.ExecutionMode)
	}

	if config.ContinueOnError != true {
		t.Error("Expected ContinueOnError to be true")
	}

	if config.MaxConcurrency != 5 {
		t.Errorf("Expected MaxConcurrency 5, got %d", config.MaxConcurrency)
	}

	if config.EnableCheckpoints != true {
		t.Error("Expected EnableCheckpoints to be true")
	}

	if config.CheckpointDir != "custom/checkpoint/dir" {
		t.Errorf("Expected CheckpointDir 'custom/checkpoint/dir', got %s", config.CheckpointDir)
	}

	// Verify Step timeout
	timeout := config.GetStageTimeout("custom-Step")
	if timeout != 45*time.Second {
		t.Errorf("Expected Step timeout 45s, got %v", timeout)
	}

	// Verify Step config
	StepConfig, exists := config.GetStepConfig("custom-Step")
	if !exists {
		t.Error("Expected Step config to exist")
	}

	if configMap, ok := StepConfig.(map[string]interface{}); ok {
		if configMap["param1"] != "value1" {
			t.Errorf("Expected param1 'value1', got %v", configMap["param1"])
		}
		if configMap["param2"] != 42 {
			t.Errorf("Expected param2 42, got %v", configMap["param2"])
		}
	} else {
		t.Error("Expected Step config to be a map")
	}

	// Verify retry config
	if config.RetryConfig.MaxAttempts != 4 {
		t.Errorf("Expected MaxAttempts 4, got %d", config.RetryConfig.MaxAttempts)
	}
	if config.RetryConfig.InitialDelay != 200*time.Millisecond {
		t.Errorf("Expected InitialDelay 200ms, got %v", config.RetryConfig.InitialDelay)
	}
	if config.RetryConfig.MaxDelay != 3*time.Second {
		t.Errorf("Expected MaxDelay 3s, got %v", config.RetryConfig.MaxDelay)
	}
	if config.RetryConfig.Multiplier != 1.6 {
		t.Errorf("Expected Multiplier 1.6, got %f", config.RetryConfig.Multiplier)
	}
}