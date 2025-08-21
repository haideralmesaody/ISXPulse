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

// TestFullPipelineExecution tests a complete operation flow
func TestFullPipelineExecution(t *testing.T) {
	// Create test infrastructure
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Configure operation
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeSequential).
		WithStageTimeout(operations.StageIDScraping, 1*time.Second).
		WithStageTimeout(operations.StageIDProcessing, 1*time.Second).
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
		}).
		Build()
	manager.SetConfig(config)
	
	// Create and register steps that simulate the real operation
	scrapingStage := testutil.NewStageBuilder(operations.StageIDScraping, operations.StageNameScraping).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			// Simulate scraping
			StepState := state.GetStage(operations.StageIDScraping)
			StepState.UpdateProgress(25, "Connecting to ISX website...")
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(50, "Downloading reports...")
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(75, "Saving files...")
			time.Sleep(20 * time.Millisecond)
			
			// Set context for next Step
			state.SetContext(operations.ContextKeyFilesFound, 10)
			state.SetContext(operations.ContextKeyScraperSuccess, true)
			
			StepState.UpdateProgress(100, "Scraping completed")
			return nil
		}).
		Build()
	
	processingStage := testutil.NewStageBuilder(operations.StageIDProcessing, operations.StageNameProcessing).
		WithDependencies(operations.StageIDScraping).
		WithValidate(func(state *operations.OperationState) error {
			// Check if scraper succeeded
			if success, ok := state.GetContext(operations.ContextKeyScraperSuccess); !ok || !success.(bool) {
				return errors.New("scraper did not succeed")
			}
			return nil
		}).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			// Simulate processing
			filesFound, _ := state.GetContext(operations.ContextKeyFilesFound)
			totalFiles := filesFound.(int)
			
			StepState := state.GetStage(operations.StageIDProcessing)
			for i := 0; i < totalFiles; i++ {
				progress := float64(i+1) / float64(totalFiles) * 100
				StepState.UpdateProgress(progress, fmt.Sprintf("Processing file %d of %d", i+1, totalFiles))
				time.Sleep(10 * time.Millisecond)
			}
			
			state.SetContext(operations.ContextKeyFilesProcessed, totalFiles)
			return nil
		}).
		Build()
	
	indicesStage := testutil.NewStageBuilder(operations.StageIDIndices, operations.StageNameIndices).
		WithDependencies(operations.StageIDProcessing).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage(operations.StageIDIndices)
			StepState.UpdateProgress(50, "Extracting ISX60...")
			time.Sleep(20 * time.Millisecond)
			StepState.UpdateProgress(100, "Indices extracted")
			return nil
		}).
		Build()
	
	analysisStage := testutil.NewStageBuilder(operations.StageIDLiquidity, operations.StageNameLiquidity).
		WithDependencies(operations.StageIDIndices).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage(operations.StageIDLiquidity)
			StepState.UpdateProgress(100, "Analysis completed")
			return nil
		}).
		Build()
	
	// Register steps
	testutil.AssertNoError(t, manager.RegisterStage(scrapingStage))
	testutil.AssertNoError(t, manager.RegisterStage(processingStage))
	testutil.AssertNoError(t, manager.RegisterStage(indicesStage))
	testutil.AssertNoError(t, manager.RegisterStage(analysisStage))
	
	// Execute operation
	ctx := context.Background()
	req := testutil.CreateOperationRequest(operations.ModeInitial)
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Verify response
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	testutil.AssertEqual(t, len(resp.Steps), 4)
	
	// Verify all steps completed
	for _, Step := range resp.Steps {
		testutil.AssertStepStatus(t, Step, operations.StepStatusCompleted)
	}
	
	// Verify WebSocket messages
	_ = hub.GetMessages()
	
	// Should have operation reset at start
	testutil.AssertWebSocketMessage(t, hub, operations.EventTypePipelineReset)
	
	// Should have multiple progress updates
	progressMessages := hub.GetMessagesByType(operations.EventTypePipelineProgress)
	if len(progressMessages) < 4 {
		t.Errorf("Expected at least 4 progress messages, got %d", len(progressMessages))
	}
	
	// Should have operation complete at end
	testutil.AssertWebSocketMessage(t, hub, operations.EventTypePipelineComplete)
	
	// Verify Step execution order
	testutil.AssertStageOrder(t, []*testutil.MockStage{
		scrapingStage, processingStage, indicesStage, analysisStage,
	}, []string{
		operations.StageIDScraping,
		operations.StageIDProcessing,
		operations.StageIDIndices,
		operations.StageIDLiquidity,
	})
}

// TestPipelineWithFailureAndRetry tests Step failure with retry
func TestPipelineWithFailureAndRetry(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Configure with retry
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create a Step that fails once then succeeds
	retryStage := testutil.CreateRetryableStage("retry-Step", "Retry Step", 1)
	
	// Register Step
	testutil.AssertNoError(t, manager.RegisterStage(retryStage))
	
	// Execute operation
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-retry"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Verify success after retry
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Verify Step was called twice
	testutil.AssertEqual(t, retryStage.GetExecuteCalls(), 2)
	
	// Note: Log verification would require structured logging capture
	// For now, we verify retry behavior through execution count
}

// TestPipelineWithDependencyFailure tests dependency handling
func TestPipelineWithDependencyFailure(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	config.ContinueOnError = false // Stop on error
	manager.SetConfig(config)
	
	// Create steps where first fails
	stage1 := testutil.CreateFailingStage("stage1", "Step 1", errors.New("stage1 failed"))
	stage2 := testutil.CreateSuccessfulStage("stage2", "Step 2", "stage1")
	stage3 := testutil.CreateSuccessfulStage("stage3", "Step 3", "stage2")
	
	// Register steps
	testutil.AssertNoError(t, manager.RegisterStage(stage1))
	testutil.AssertNoError(t, manager.RegisterStage(stage2))
	testutil.AssertNoError(t, manager.RegisterStage(stage3))
	
	// Execute operation
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-deps"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertError(t, err, true)
	
	// Verify operation failed
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusFailed)
	
	// Verify Step statuses
	testutil.AssertStageFailed(t, &operations.OperationState{Steps: resp.Steps}, "stage1")
	
	// steps 2 and 3 should be skipped due to dependency failure
	stage2State := resp.Steps["stage2"]
	stage3State := resp.Steps["stage3"]
	
	if stage2State.Status != operations.StepStatusSkipped {
		t.Errorf("Stage2 status = %v, want skipped", stage2State.Status)
	}
	if stage3State.Status != operations.StepStatusSkipped {
		t.Errorf("Stage3 status = %v, want skipped", stage3State.Status)
	}
}

// TestPipelineTimeout tests Step timeout handling
func TestPipelineTimeout(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	// Configure with short timeout
	config := operations.NewConfigBuilder().
		WithStageTimeout("slow-Step", 50*time.Millisecond).
		Build()
	manager.SetConfig(config)
	
	// Create a slow Step
	slowStage := testutil.CreateSlowStage("slow-Step", "Slow Step", 200*time.Millisecond)
	
	// Register Step
	testutil.AssertNoError(t, manager.RegisterStage(slowStage))
	
	// Execute operation
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-timeout"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertError(t, err, true)
	
	// Verify operation failed
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusFailed)
	
	// Verify timeout error
	StepState := resp.Steps["slow-Step"]
	if StepState.Status != operations.StepStatusFailed {
		t.Errorf("Step status = %v, want failed", StepState.Status)
	}
}

// TestPipelineCancellation tests context cancellation
func TestPipelineCancellation(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create a slow Step
	slowStage := testutil.CreateSlowStage("slow-Step", "Slow Step", 200*time.Millisecond)
	
	// Register Step
	testutil.AssertNoError(t, manager.RegisterStage(slowStage))
	
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	
	// Execute operation
	req := operations.OperationRequest{ID: "test-cancel"}
	_, err := manager.Execute(ctx, req)
	
	// Should have error
	testutil.AssertError(t, err, true)
	
	// Check for cancellation
	if ctx.Err() != context.Canceled {
		t.Error("Context should be cancelled")
	}
}

// TestComplexPipelineDependencies tests diamond dependency pattern
func TestComplexPipelineDependencies(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create diamond pattern steps
	steps := testutil.CreateComplexPipelineStages()
	
	// Register all steps
	for _, Step := range steps {
		testutil.AssertNoError(t, manager.RegisterStage(Step))
	}
	
	// Execute operation
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-diamond"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Verify all steps completed
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Verify execution order
	// A must run first
	// B and C can run in any order after A
	// D must run last
	mockStages := make([]*testutil.MockStage, len(steps))
	for i, s := range steps {
		mockStages[i] = s.(*testutil.MockStage)
	}
	
	// Check A was first
	aTime := mockStages[0].ExecuteArgs[0].Time
	for i := 1; i < len(mockStages); i++ {
		if mockStages[i].ExecuteArgs[0].Time.Before(aTime) {
			t.Error("Step A should execute first")
		}
	}
	
	// Check D was last
	dTime := mockStages[3].ExecuteArgs[0].Time
	for i := 0; i < 3; i++ {
		if mockStages[i].ExecuteArgs[0].Time.After(dTime) {
			t.Error("Step D should execute last")
		}
	}
}

// TestOperationStateSharing tests context sharing between steps
func TestOperationStateSharing(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create steps that share data
	stage1 := testutil.CreateContextAwareStage("stage1", "Step 1", "", "shared_data", "Hello from stage1")
	stage2 := testutil.CreateContextAwareStage("stage2", "Step 2", "shared_data", "stage2_data", "Modified", "stage1")
	
	// Step 2 should read from Step 1 and write its own data
	stage2.ExecuteFunc = func(ctx context.Context, state *operations.OperationState) error {
		// Read shared data
		data, ok := state.GetContext("shared_data")
		if !ok {
			return errors.New("shared_data not found")
		}
		
		// Modify and write back
		modified := data.(string) + " - Modified by stage2"
		state.SetContext("stage2_data", modified)
		return nil
	}
	
	// Register steps
	testutil.AssertNoError(t, manager.RegisterStage(stage1))
	testutil.AssertNoError(t, manager.RegisterStage(stage2))
	
	// Execute operation
	ctx := context.Background()
	req := operations.OperationRequest{ID: "test-sharing"}
	
	resp, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Verify operation completed
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Note: We can't verify context values from response as they're not included
	// In a real implementation, we might want to add a way to retrieve final state
}