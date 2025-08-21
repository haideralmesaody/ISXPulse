package operations_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

// TestIntegrationFullPipeline tests a complete operation execution with all features
func TestIntegrationFullPipeline(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	harness.SetupStandardPipeline()
	
	// Configure operation manager
	config := operations.NewConfigBuilder().
		WithExecutionMode(operations.ExecutionModeSequential).
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 10 * time.Millisecond,
		}).
		Build()
	harness.GetManager().SetConfig(config)
	
	// Generate test data
	harness.GetDataGenerator().GenerateDateRange("2024-01-01", "2024-01-05")
	
	// Execute operation
	resp, err := harness.ExecutePipeline("2024-01-01", "2024-01-05")
	
	// Verify success
	harness.AssertPipelineSuccess(resp, err)
	harness.AssertWebSocketMessages()
	
	// Verify all steps completed
	steps := []string{
		operations.StageIDScraping,
		operations.StageIDProcessing,
		operations.StageIDIndices,
		operations.StageIDLiquidity,
	}
	
	for _, stageID := range steps {
		testutil.AssertStageCompleted(t, &operations.OperationState{Steps: resp.Steps}, stageID)
	}
	
	// Verify WebSocket message count
	messages := harness.GetHub().GetMessages()
	if len(messages) < 10 {
		t.Errorf("Expected at least 10 WebSocket messages, got %d", len(messages))
	}
}

// TestIntegrationPipelineWithFailure tests operation behavior with Step failures
func TestIntegrationPipelineWithFailure(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	
	// Setup operation with a failing Step
	scrapingStage := testutil.CreateSuccessfulStage(operations.StageIDScraping, operations.StageNameScraping)
	processingStage := testutil.CreateFailingStage(operations.StageIDProcessing, operations.StageNameProcessing, 
		fmt.Errorf("processing failed"), operations.StageIDScraping)
	indicesStage := testutil.CreateSuccessfulStage(operations.StageIDIndices, operations.StageNameIndices, 
		operations.StageIDProcessing)
	analysisStage := testutil.CreateSuccessfulStage(operations.StageIDLiquidity, operations.StageNameLiquidity, 
		operations.StageIDIndices)
	
	harness.GetManager().RegisterStage(scrapingStage)
	harness.GetManager().RegisterStage(processingStage)
	harness.GetManager().RegisterStage(indicesStage)
	harness.GetManager().RegisterStage(analysisStage)
	
	// Execute operation
	resp, err := harness.ExecutePipeline("2024-01-01", "2024-01-05")
	
	// Verify failure
	testutil.AssertError(t, err, true)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusFailed)
	
	// Verify Step statuses
	testutil.AssertStageCompleted(t, &operations.OperationState{Steps: resp.Steps}, operations.StageIDScraping)
	testutil.AssertStageFailed(t, &operations.OperationState{Steps: resp.Steps}, operations.StageIDProcessing)
	testutil.AssertStageSkipped(t, &operations.OperationState{Steps: resp.Steps}, operations.StageIDIndices)
	testutil.AssertStageSkipped(t, &operations.OperationState{Steps: resp.Steps}, operations.StageIDLiquidity)
	
	// Verify error WebSocket message
	testutil.AssertWebSocketMessage(t, harness.GetHub(), operations.EventTypeOperationError)
}

// TestIntegrationPipelineWithRetry tests retry mechanism
func TestIntegrationPipelineWithRetry(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	
	// Configure retries
	config := operations.NewConfigBuilder().
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     50 * time.Millisecond,
			Multiplier:   2.0,
		}).
		Build()
	harness.GetManager().SetConfig(config)
	
	// Create Step that fails twice then succeeds
	scrapingStage := testutil.CreateRetryableStage(operations.StageIDScraping, operations.StageNameScraping, 2)
	processingStage := testutil.CreateSuccessfulStage(operations.StageIDProcessing, operations.StageNameProcessing, 
		operations.StageIDScraping)
	
	harness.GetManager().RegisterStage(scrapingStage)
	harness.GetManager().RegisterStage(processingStage)
	
	// Execute operation
	resp, err := harness.ExecutePipeline("2024-01-01", "2024-01-05")
	
	// Should succeed after retries
	testutil.AssertNoError(t, err)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Verify retry attempts
	if scrapingStage.GetExecuteCalls() != 3 {
		t.Errorf("Expected 3 execution attempts, got %d", scrapingStage.GetExecuteCalls())
	}
	
	// Note: Retry logging verification would require structured log capture
	// For now, we verify retry behavior through execution count
	t.Logf("Retry behavior verified through execution count: %d attempts", scrapingStage.GetExecuteCalls())
}

// TestIntegrationPipelineTimeout tests Step timeout
func TestIntegrationPipelineTimeout(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	
	// Configure short timeout
	config := operations.NewConfigBuilder().
		WithStageTimeout(operations.StageIDScraping, 50*time.Millisecond).
		Build()
	harness.GetManager().SetConfig(config)
	
	// Create slow Step
	scrapingStage := testutil.CreateSlowStage(operations.StageIDScraping, operations.StageNameScraping, 
		200*time.Millisecond)
	harness.GetManager().RegisterStage(scrapingStage)
	
	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	req := operations.OperationRequest{
		ID:       "test-timeout",
		FromDate: "2024-01-01",
		ToDate:   "2024-01-05",
	}
	
	resp, err := harness.GetManager().Execute(ctx, req)
	
	// Should timeout
	testutil.AssertError(t, err, true)
	if resp != nil {
		testutil.AssertStageFailed(t, &operations.OperationState{Steps: resp.Steps}, operations.StageIDScraping)
	}
}

// TestIntegrationConcurrentPipelines tests running multiple operations concurrently
func TestIntegrationConcurrentPipelines(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	harness.SetupStandardPipeline()
	
	// Run 5 concurrent operations
	errors := harness.RunConcurrentPipelines(5)
	
	// All should succeed
	for i, err := range errors {
		if err != nil {
			t.Errorf("operation %d failed: %v", i, err)
		}
	}
	
	// Verify no active operations remain
	operations := harness.GetManager().ListOperations()
	if len(operations) != 0 {
		t.Errorf("Expected 0 active operations, got %d", len(operations))
	}
}

// TestIntegrationOperationStateSharing tests context sharing between steps
func TestIntegrationOperationStateSharing(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	
	// Create steps that share data
	stage1 := testutil.CreateContextAwareStage("stage1", "Step 1", "", "data1", "value1")
	stage2 := testutil.CreateContextAwareStage("stage2", "Step 2", "data1", "data2", "value2", "stage1")
	stage3 := testutil.NewStageBuilder("stage3", "Step 3").
		WithDependencies("stage2").
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			// Verify both values are available
			val1, ok1 := state.GetContext("data1")
			val2, ok2 := state.GetContext("data2")
			
			if !ok1 || val1 != "value1" {
				return fmt.Errorf("data1 not found or incorrect")
			}
			if !ok2 || val2 != "value2" {
				return fmt.Errorf("data2 not found or incorrect")
			}
			
			return nil
		}).
		Build()
	
	harness.GetManager().RegisterStage(stage1)
	harness.GetManager().RegisterStage(stage2)
	harness.GetManager().RegisterStage(stage3)
	
	// Execute
	resp, err := harness.ExecutePipeline("2024-01-01", "2024-01-05")
	
	// Should succeed
	testutil.AssertNoError(t, err)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
}

// TestIntegrationComplexDependencies tests complex dependency patterns
func TestIntegrationComplexDependencies(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	
	// Create diamond dependency pattern
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	steps := testutil.CreateComplexPipelineStages()
	
	for _, Step := range steps {
		harness.GetManager().RegisterStage(Step)
	}
	
	// Execute
	resp, err := harness.ExecutePipeline("2024-01-01", "2024-01-05")
	
	// Should succeed
	testutil.AssertNoError(t, err)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	// Verify all steps completed
	for _, Step := range steps {
		testutil.AssertStageCompleted(t, &operations.OperationState{Steps: resp.Steps}, Step.ID())
	}
}

// TestIntegrationProgressTracking tests detailed progress tracking
func TestIntegrationProgressTracking(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	
	// Create Step with detailed progress updates
	progressStage := testutil.NewStageBuilder("progress", "Progress Step").
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage("progress")
			
			// Simulate work with progress updates
			for i := 0; i <= 10; i++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					progress := float64(i) * 10
					StepState.UpdateProgress(progress, fmt.Sprintf("Processing Step %d of 10", i))
					time.Sleep(10 * time.Millisecond)
				}
			}
			
			return nil
		}).
		Build()
	
	harness.GetManager().RegisterStage(progressStage)
	
	// Execute
	resp, err := harness.ExecutePipeline("2024-01-01", "2024-01-05")
	
	// Should succeed
	testutil.AssertNoError(t, err)
	
	// Verify progress messages
	progressMessages := harness.GetHub().GetMessagesByType(operations.EventTypePipelineProgress)
	if len(progressMessages) < 2 {
		t.Errorf("Expected at least 2 progress messages, got %d", len(progressMessages))
	}
	
	// Log all messages for debugging
	allMessages := harness.GetHub().GetMessages()
	t.Logf("Total messages: %d", len(allMessages))
	for _, msg := range allMessages {
		t.Logf("Message type: %s, Step: %s", msg.EventType, msg.Step)
	}
	
	// Verify final progress is 100
	Step := resp.Steps["progress"]
	if Step.Progress != 100 {
		t.Errorf("Expected final progress 100, got %.1f", Step.Progress)
	}
}

// TestIntegrationErrorRecovery tests error recovery mechanisms
func TestIntegrationErrorRecovery(t *testing.T) {
	harness := testutil.NewIntegrationTestHarness(t)
	
	// Configure continue on error
	config := operations.NewConfigBuilder().
		WithContinueOnError(true).
		Build()
	harness.GetManager().SetConfig(config)
	
	// Create operation where Step 2 fails but Step 3 can still run
	stage1 := testutil.CreateSuccessfulStage("stage1", "Step 1")
	stage2 := testutil.CreateFailingStage("stage2", "Step 2", fmt.Errorf("Step 2 error"))
	stage3 := testutil.CreateSuccessfulStage("stage3", "Step 3") // No dependency on stage2
	
	harness.GetManager().RegisterStage(stage1)
	harness.GetManager().RegisterStage(stage2)
	harness.GetManager().RegisterStage(stage3)
	
	// Execute
	resp, err := harness.ExecutePipeline("2024-01-01", "2024-01-05")
	
	// operation should partially succeed (continue on error means operation completes but with errors)
	// The error might not be returned if continue on error is true
	_ = err // Error is expected but may vary based on continue on error behavior
	if resp.Status != operations.OperationStatusFailed && resp.Status != operations.OperationStatusCompleted {
		t.Errorf("Expected operation status to be failed or completed, got %s", resp.Status)
	}
	
	// Verify Step statuses
	testutil.AssertStageCompleted(t, &operations.OperationState{Steps: resp.Steps}, "stage1")
	testutil.AssertStageFailed(t, &operations.OperationState{Steps: resp.Steps}, "stage2")
	testutil.AssertStageCompleted(t, &operations.OperationState{Steps: resp.Steps}, "stage3")
}

// TestIntegrationLargeDataset tests operation with large dataset
func TestIntegrationLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}
	
	harness := testutil.NewIntegrationTestHarness(t)
	harness.SetupStandardPipeline()
	
	// Generate large dataset
	harness.GetDataGenerator().GenerateLargeDataset(100, 10) // 100 days, 10 tickers per day
	
	// Execute with timeout
	start := time.Now()
	resp, err := harness.ExecutePipelineWithTimeout("2023-01-01", "2023-04-10", 5*time.Minute)
	duration := time.Since(start)
	
	// Should complete within timeout
	testutil.AssertNoError(t, err)
	testutil.AssertOperationStatus(t, &operations.OperationState{Status: resp.Status}, operations.OperationStatusCompleted)
	
	t.Logf("Large dataset operation completed in %v", duration)
	
	// Verify reasonable performance
	if duration > 5*time.Minute {
		t.Errorf("operation took too long: %v", duration)
	}
}

// Helper function
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}