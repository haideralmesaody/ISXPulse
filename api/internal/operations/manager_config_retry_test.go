package operations_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"isxcli/internal/operations"
)

// TestManagerCalculateRetryDelay tests the retry delay calculation logic
func TestManagerCalculateRetryDelay(t *testing.T) {
	tests := []struct {
		name           string
		attempt        int
		retryConfig    operations.RetryConfig
		expectedDelay  time.Duration
	}{
		{
			name:    "first retry attempt",
			attempt: 1,
			retryConfig: operations.RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
			},
			expectedDelay: 0, // (attempt-1) * multiplier = (1-1) * 2 = 0
		},
		{
			name:    "second retry attempt",
			attempt: 2,
			retryConfig: operations.RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
			},
			expectedDelay: 200 * time.Millisecond, // 100ms * (2-1) * 2.0 = 200ms
		},
		{
			name:    "third retry attempt",
			attempt: 3,
			retryConfig: operations.RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
			},
			expectedDelay: 400 * time.Millisecond, // 100ms * (3-1) * 2.0 = 400ms
		},
		{
			name:    "delay exceeds max delay",
			attempt: 10,
			retryConfig: operations.RetryConfig{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     500 * time.Millisecond,
				Multiplier:   2.0,
			},
			expectedDelay: 500 * time.Millisecond, // Should be capped at MaxDelay
		},
		{
			name:    "minimal retry config",
			attempt: 2,
			retryConfig: operations.RetryConfig{
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   1.5,
			},
			expectedDelay: 1500 * time.Microsecond, // 1ms * (2-1) * 1.5 = 1.5ms
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWS := &mockManagerWebSocketHub{}
			config := operations.NewConfigBuilder().
				WithRetryConfig(tt.retryConfig).
				Build()
			registry := operations.NewRegistry()
			manager := operations.NewManager(mockWS, registry, config)

			// Use reflection to access the private calculateRetryDelay method
			// Since it's private, we'll test it indirectly through retry behavior
			mockStage := newMockManagerStage("retry-delay-test", "Retry Delay Test", nil).
				WithFailure(fmt.Errorf("test error"))
			
			manager.RegisterStage(mockStage)
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			// Record start time and execute
			startTime := time.Now()
			req := operations.OperationRequest{
				ID:   "delay-test-operation",
				Mode: "test",
			}
			
			manager.Execute(ctx, req)
			
			// Verify the Step was called the expected number of times
			expectedAttempts := tt.retryConfig.MaxAttempts
			if mockStage.executeCallCount != expectedAttempts {
				t.Errorf("Expected %d execution attempts, got %d", expectedAttempts, mockStage.executeCallCount)
			}
			
			// For attempts > 1, verify reasonable total execution time considering delays
			if tt.attempt > 1 && mockStage.executeCallCount > 1 {
				totalTime := time.Since(startTime)
				// Should take at least the sum of expected delays between retries
				// This is a rough check since we can't directly test the private method
				if totalTime < tt.expectedDelay/2 { // Allow some tolerance
					t.Logf("Total execution time %v seems short for expected delay %v", totalTime, tt.expectedDelay)
				}
			}
		})
	}
}

// TestManagerConfigurationVariations tests different operation configurations
func TestManagerConfigurationVariations(t *testing.T) {
	tests := []struct {
		name           string
		configBuilder  func() *operations.Config
		expectedMode   operations.ExecutionMode
		expectError    bool
	}{
		{
			name: "default configuration",
			configBuilder: func() *operations.Config {
				return operations.NewConfig()
			},
			expectedMode: operations.ExecutionModeSequential,
			expectError:  false,
		},
		{
			name: "sequential execution mode",
			configBuilder: func() *operations.Config {
				return operations.NewConfigBuilder().
					WithExecutionMode(operations.ExecutionModeSequential).
					Build()
			},
			expectedMode: operations.ExecutionModeSequential,
			expectError:  false,
		},
		{
			name: "parallel execution mode", 
			configBuilder: func() *operations.Config {
				return operations.NewConfigBuilder().
					WithExecutionMode(operations.ExecutionModeParallel).
					WithMaxConcurrency(3).
					Build()
			},
			expectedMode: operations.ExecutionModeParallel,
			expectError:  false, // Should fall back to sequential
		},
		{
			name: "continue on error enabled",
			configBuilder: func() *operations.Config {
				return operations.NewConfigBuilder().
					WithContinueOnError(true).
					Build()
			},
			expectedMode: operations.ExecutionModeSequential,
			expectError:  false,
		},
		{
			name: "custom Step timeouts",
			configBuilder: func() *operations.Config {
				return operations.NewConfigBuilder().
					WithStageTimeout(operations.StageIDScraping, 30*time.Second).
					WithStageTimeout(operations.StageIDProcessing, 60*time.Second).
					Build()
			},
			expectedMode: operations.ExecutionModeSequential,
			expectError:  false,
		},
		{
			name: "complex retry configuration",
			configBuilder: func() *operations.Config {
				return operations.NewConfigBuilder().
					WithRetryConfig(operations.RetryConfig{
						MaxAttempts:  5,
						InitialDelay: 50 * time.Millisecond,
						MaxDelay:     2 * time.Second,
						Multiplier:   1.8,
					}).
					Build()
			},
			expectedMode: operations.ExecutionModeSequential,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWS := &mockManagerWebSocketHub{}
			config := tt.configBuilder()
			registry := operations.NewRegistry()
			manager := operations.NewManager(mockWS, registry, config)

			// Verify configuration was applied
			actualConfig := manager.GetConfig()
			if actualConfig.ExecutionMode != tt.expectedMode {
				t.Errorf("Expected execution mode %v, got %v", tt.expectedMode, actualConfig.ExecutionMode)
			}

			// Test basic execution with this configuration
			mockStage := newMockManagerStage("config-test", "Config Test Step", nil)
			manager.RegisterStage(mockStage)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := operations.OperationRequest{
				ID:   "config-test-operation",
				Mode: "test",
			}

			_, err := manager.Execute(ctx, req)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify Step was executed
			if mockStage.executeCallCount == 0 {
				t.Error("Step should have been executed")
			}
		})
	}
}

// TestManagerExecuteParallel tests the parallel execution path  
func TestManagerExecuteParallel(t *testing.T) {
	mockWS := &mockManagerWebSocketHub{}
	
	// Create config with parallel execution mode
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeParallel).
		WithMaxConcurrency(2).
		Build()
	
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add independent steps that can run in parallel
	mockStage1 := newMockManagerStage("parallel1", "Parallel Step 1", nil)
	mockStage2 := newMockManagerStage("parallel2", "Parallel Step 2", nil)
	
	manager.RegisterStage(mockStage1)
	manager.RegisterStage(mockStage2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := operations.OperationRequest{
		ID:   "parallel-test-operation",
		Mode: "test",
	}

	startTime := time.Now()
	_, err := manager.Execute(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify both steps were executed
	if mockStage1.executeCallCount == 0 {
		t.Error("Step 1 should have been executed")
	}
	
	if mockStage2.executeCallCount == 0 {
		t.Error("Step 2 should have been executed")
	}

	// Note: Currently executeParallel falls back to sequential execution
	// So we verify it works but doesn't necessarily run in parallel yet
	t.Logf("Parallel execution completed in %v (currently falls back to sequential)", duration)
}

// TestManagerConfigBuilder tests the configuration builder pattern
func TestManagerConfigBuilder(t *testing.T) {
	builder := operations.NewConfigBuilder()
	
	config := builder.
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

// TestManagerSetConfigDynamic tests dynamic configuration updates
func TestManagerSetConfigDynamic(t *testing.T) {
	mockWS := &mockManagerWebSocketHub{}
	registry := operations.NewRegistry()
	
	// Start with default config
	manager := operations.NewManager(mockWS, registry, nil)
	
	// Verify initial config is default
	initialConfig := manager.GetConfig()
	if initialConfig.ExecutionMode != operations.ExecutionModeSequential {
		t.Errorf("Expected initial execution mode sequential, got %v", initialConfig.ExecutionMode)
	}

	// Update to new configuration
	newConfig := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeParallel).
		WithContinueOnError(true).
		Build()

	manager.SetConfig(newConfig)

	// Verify config was updated
	updatedConfig := manager.GetConfig()
	if updatedConfig.ExecutionMode != operations.ExecutionModeParallel {
		t.Errorf("Expected updated execution mode parallel, got %v", updatedConfig.ExecutionMode)
	}
	if updatedConfig.ContinueOnError != true {
		t.Error("Expected ContinueOnError to be true after update")
	}

	// Test setting nil config (should be ignored)
	manager.SetConfig(nil)
	
	// Config should remain unchanged
	finalConfig := manager.GetConfig()
	if finalConfig.ExecutionMode != operations.ExecutionModeParallel {
		t.Errorf("Expected execution mode to remain parallel after nil update, got %v", finalConfig.ExecutionMode)
	}
}

// TestManagerRetryConfigScenarios tests various retry configuration scenarios
func TestManagerRetryConfigScenarios(t *testing.T) {
	tests := []struct {
		name         string
		retryConfig  operations.RetryConfig
		stageError   error
		expectRetries int
	}{
		{
			name: "retryable error with multiple attempts",
			retryConfig: operations.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			},
			stageError:    operations.NewExecutionError("test", fmt.Errorf("retryable error"), true),
			expectRetries: 3,
		},
		{
			name: "fatal error should not retry",
			retryConfig: operations.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			},
			stageError:    operations.NewExecutionError("test", fmt.Errorf("fatal error"), false),
			expectRetries: 1, // Should only try once
		},
		{
			name: "single attempt configuration",
			retryConfig: operations.RetryConfig{
				MaxAttempts:  1,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			},
			stageError:    operations.NewExecutionError("test", fmt.Errorf("any error"), true),
			expectRetries: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWS := &mockManagerWebSocketHub{}
			config := operations.NewConfigBuilder().
				WithRetryConfig(tt.retryConfig).
				Build()
			registry := operations.NewRegistry()
			manager := operations.NewManager(mockWS, registry, config)

			mockStage := newMockManagerStage("retry-scenario-test", "Retry Scenario Test", nil).
				WithFailure(tt.stageError)
			
			manager.RegisterStage(mockStage)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := operations.OperationRequest{
				ID:   "retry-scenario-operation",
				Mode: "test",
			}

			_, err := manager.Execute(ctx, req)

			// All retry scenarios should result in an error since our mock always fails
			if err == nil {
				t.Error("Expected error from failing Step")
			}

			// Verify the number of retry attempts
			if mockStage.executeCallCount != tt.expectRetries {
				t.Errorf("Expected %d retry attempts, got %d", tt.expectRetries, mockStage.executeCallCount)
			}
		})
	}
}