package operations_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

func TestManager(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	
	manager := operations.NewManager(hub, nil, nil)
	
	testutil.AssertNotNil(t, manager)
	testutil.AssertNotNil(t, manager.GetRegistry())
	testutil.AssertNotNil(t, manager.GetConfig())
}

func TestManagerRegisterStage(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Create and register a Step
	Step := testutil.CreateSuccessfulStage("test", "Test Step")
	
	testutil.AssertNoError(t, manager.RegisterStage(Step))
	
	// Verify Step is in registry
	registry := manager.GetRegistry()
	if !registry.Has("test") {
		t.Error("Step not found in registry after registration")
	}
}

func TestManagerSetConfig(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Create custom config
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeParallel).
		WithMaxConcurrency(4).
		Build()
	
	manager.SetConfig(config)
	
	// Verify config was set
	gotConfig := manager.GetConfig()
	testutil.AssertEqual(t, gotConfig.ExecutionMode, operations.ExecutionModeParallel)
	testutil.AssertEqual(t, gotConfig.MaxConcurrency, 4)
}

func TestManagerExecuteSequential(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Configure for sequential execution
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeSequential).
		Build()
	manager.SetConfig(config)
	
	// Create steps
	stage1 := testutil.CreateSuccessfulStage("stage1", "Step 1")
	stage2 := testutil.CreateSuccessfulStage("stage2", "Step 2", "stage1")
	stage3 := testutil.CreateSuccessfulStage("stage3", "Step 3", "stage2")
	
	manager.RegisterStage(stage1)
	manager.RegisterStage(stage2)
	manager.RegisterStage(stage3)
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-sequential"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Verify execution order
	testutil.AssertStageOrder(t, []*testutil.MockStage{stage1, stage2, stage3}, 
		[]string{"stage1", "stage2", "stage3"})
}

func TestManagerExecuteWithRetries(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Configure with retries
	config := operations.NewConfigBuilder().
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     50 * time.Millisecond,
			Multiplier:   2.0,
		}).
		Build()
	manager.SetConfig(config)
	
	// Create Step that fails twice then succeeds
	retryStage := testutil.CreateRetryableStage("retry", "Retry Step", 2)
	manager.RegisterStage(retryStage)
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-retry"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Verify Step was called 3 times (2 failures + 1 success)
	testutil.AssertEqual(t, retryStage.GetExecuteCalls(), 3)
	
	// Check logs would be verified via structured logging in a real test
	// For now, we just verify the retry behavior succeeded
	// In a real implementation, we would check structured logs for retry attempts
	// The important thing is that the Step was called 3 times (verified above)
}

func TestManagerExecuteWithTimeout(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Configure with short timeout
	config := operations.NewConfigBuilder().
		WithStageTimeout("slow", 50*time.Millisecond).
		Build()
	manager.SetConfig(config)
	
	// Create slow Step
	slowStage := testutil.CreateSlowStage("slow", "Slow Step", 200*time.Millisecond)
	manager.RegisterStage(slowStage)
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-timeout"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertError(t, err, true)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusFailed)
	
	// Verify timeout error
	testutil.AssertStageFailed(t, &operations.OperationState{Steps: resp.Steps}, "slow")
}

func TestManagerExecuteWithCancellation(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create steps
	fastStage := testutil.CreateSuccessfulStage("fast", "Fast Step")
	slowStage := testutil.CreateSlowStage("slow", "Slow Step", 500*time.Millisecond, "fast")
	
	manager.RegisterStage(fastStage)
	manager.RegisterStage(slowStage)
	
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start operation in goroutine
	done := make(chan struct{})
	var resp *operations.OperationResponse
	var err error
	
	go func() {
		req := operations.OperationRequest{ID: "test-cancel"}
		resp, err = manager.Execute(ctx, req)
		close(done)
	}()
	
	// Cancel after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()
	
	// Wait for completion
	<-done
	
	// Should have error
	testutil.AssertError(t, err, true)
	
	// operation should be failed or cancelled
	if resp != nil && resp.Status != operations.OperationStatusFailed && resp.Status != operations.OperationStatusCancelled {
		t.Errorf("operation status = %v, want failed or cancelled", resp.Status)
	}
}

func TestManagerExecuteWithDependencies(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create steps with dependencies
	stage1 := testutil.CreateSuccessfulStage("s1", "Step 1")
	stage2 := testutil.CreateSuccessfulStage("s2", "Step 2", "s1")
	stage3 := testutil.CreateSuccessfulStage("s3", "Step 3", "s1")
	stage4 := testutil.CreateSuccessfulStage("s4", "Step 4", "s2", "s3")
	
	manager.RegisterStage(stage1)
	manager.RegisterStage(stage2)
	manager.RegisterStage(stage3)
	manager.RegisterStage(stage4)
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-deps"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Verify all steps completed
	for _, stageID := range []string{"s1", "s2", "s3", "s4"} {
		testutil.AssertStageCompleted(t, &operations.OperationState{Steps: resp.Steps}, stageID)
	}
	
	// Verify s1 ran before s2 and s3
	s1Time := stage1.ExecuteArgs[0].Time
	s2Time := stage2.ExecuteArgs[0].Time
	s3Time := stage3.ExecuteArgs[0].Time
	
	if s2Time.Before(s1Time) || s3Time.Before(s1Time) {
		t.Error("Dependent steps ran before their dependency")
	}
	
	// Verify s4 ran after s2 and s3
	s4Time := stage4.ExecuteArgs[0].Time
	if s4Time.Before(s2Time) || s4Time.Before(s3Time) {
		t.Error("Step 4 ran before its dependencies")
	}
}

func TestManagerExecuteWithFailure(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	config.ContinueOnError = false
	manager.SetConfig(config)
	
	// Create steps where second fails
	stage1 := testutil.CreateSuccessfulStage("s1", "Step 1")
	stage2 := testutil.CreateFailingStage("s2", "Step 2", errors.New("Step 2 failed"), "s1")
	stage3 := testutil.CreateSuccessfulStage("s3", "Step 3", "s2")
	
	manager.RegisterStage(stage1)
	manager.RegisterStage(stage2)
	manager.RegisterStage(stage3)
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-failure"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertError(t, err, true)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusFailed)
	
	// Verify Step statuses
	testutil.AssertStageCompleted(t, &operations.OperationState{Steps: resp.Steps}, "s1")
	testutil.AssertStageFailed(t, &operations.OperationState{Steps: resp.Steps}, "s2")
	testutil.AssertStageSkipped(t, &operations.OperationState{Steps: resp.Steps}, "s3")
	
	// Verify Step 3 was not executed
	testutil.AssertEqual(t, stage3.GetExecuteCalls(), 0)
}

func TestManagerGetOperation(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Should return error for non-existent operation
	_, err1 := manager.GetOperation("nonexistent")
	testutil.AssertError(t, err1, true)
	
	// Create and execute a operation
	Step := testutil.CreateSuccessfulStage("test", "Test")
	manager.RegisterStage(Step)
	
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-get"}
	
	// Start execution in background
	done := make(chan struct{})
	go func() {
		manager.Execute(ctx, req)
		close(done)
	}()
	
	// Wait for operation to be registered (with timeout)
	var state *operations.OperationState
	var err error
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		state, err = manager.GetOperation("test-get")
		if err == nil {
			break
		}
	}
	
	// Should be able to get the operation
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, state.ID, "test-get")
	
	// Wait for completion
	<-done
}

func TestManagerListOperations(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Initially should be empty
	ops := manager.ListOperations()
	testutil.AssertEqual(t, len(ops), 0)
	
	// Create Step for testing
	Step := testutil.CreateSlowStage("test", "Test", 100*time.Millisecond)
	manager.RegisterStage(Step)
	
	// Start multiple operations
	ctx := context.Background()
	count := 3
	done := make(chan struct{}, count)
	
	for i := 0; i < count; i++ {
		go func(n int) {
			req := operations.OperationRequest{ID: fmt.Sprintf("operation-%d", n)}
			manager.Execute(ctx, req)
			done <- struct{}{}
		}(i)
	}
	
	// Give them time to start
	time.Sleep(20 * time.Millisecond)
	
	// Should have active operations
	ops = manager.ListOperations()
	if len(ops) != count {
		t.Errorf("Active operations = %d, want %d", len(ops), count)
	}
	
	// Wait for completion
	for i := 0; i < count; i++ {
		<-done
	}
}

func TestManagerCancelOperation(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Should error on non-existent operation
	err := manager.CancelOperation("nonexistent")
	testutil.AssertError(t, err, true)
	
	// Create slow Step
	Step := testutil.CreateSlowStage("test", "Test", 500*time.Millisecond)
	manager.RegisterStage(Step)
	
	// Start operation
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-cancel-mgr"}
	
	done := make(chan struct{})
	go func() {
		manager.Execute(ctx, req)
		close(done)
	}()
	
	// Give it time to start
	time.Sleep(50 * time.Millisecond)
	
	// Cancel the operation
	err = manager.CancelOperation("test-cancel-mgr")
	testutil.AssertNoError(t, err)
	
	// Wait for completion
	<-done
	
	// Check for cancellation status message
	messages := hub.GetMessagesByType(operations.EventTypeOperationStatus)
	found := false
	for _, msg := range messages {
		if metadata, ok := msg.Metadata.(map[string]interface{}); ok {
			if status, ok := metadata["status"].(operations.OperationStatus); ok {
				if status == operations.OperationStatusCancelled {
					found = true
					break
				}
			}
		}
	}
	
	if !found {
		t.Error("Expected cancellation status message")
	}
}

func TestManagerWebSocketUpdates(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create Step
	Step := testutil.CreateSuccessfulStage("test", "Test Step")
	manager.RegisterStage(Step)
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-ws"}
	
	_, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Verify WebSocket messages
	messages := hub.GetMessages()
	
	// Should have specific message types
	messageTypes := make(map[string]int)
	for _, msg := range messages {
		messageTypes[msg.EventType]++
	}
	
	// Verify required message types
	requiredTypes := []string{
		operations.EventTypePipelineReset,
		operations.EventTypeOperationStatus,
		operations.EventTypePipelineProgress,
		operations.EventTypePipelineComplete,
	}
	
	for _, msgType := range requiredTypes {
		if count := messageTypes[msgType]; count == 0 {
			t.Errorf("Missing WebSocket message type: %s", msgType)
		}
	}
}

func TestManagerConcurrentExecutions(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create Step
	Step := testutil.CreateSuccessfulStage("test", "Test Step")
	manager.RegisterStage(Step)
	
	// Execute multiple operations concurrently
	ctx := context.Background()
	count := 5
	errors := make(chan error, count)
	
	for i := 0; i < count; i++ {
		go func(n int) {
			req := operations.OperationRequest{ID: fmt.Sprintf("concurrent-%d", n)}
			_, err := manager.Execute(ctx, req)
			errors <- err
		}(i)
	}
	
	// Collect results
	for i := 0; i < count; i++ {
		err := <-errors
		testutil.AssertNoError(t, err)
	}
	
	// Verify all operations completed
	// Note: They should have been removed from active operations after completion
	ops := manager.ListOperations()
	testutil.AssertEqual(t, len(ops), 0)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}