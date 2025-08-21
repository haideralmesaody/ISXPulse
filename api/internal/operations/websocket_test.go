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

// TestWebSocketMessageFlow simulates the complete WebSocket message flow
// that the frontend expects during operation execution
func TestWebSocketMessageFlow(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create realistic steps
	steps := createRealisticPipelineStages()
	for _, Step := range steps {
		testutil.AssertNoError(t, manager.RegisterStage(Step))
	}
	
	// Execute operation
	ctx := context.Background()
	req := operations.OperationRequest{
		ID:       "frontend-test",
		Mode:     operations.ModeInitial,
		FromDate: "2024-01-01",
		ToDate:   "2024-01-31",
	}
	
	_, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Analyze WebSocket messages
	messages := hub.GetMessages()
	
	// Verify message sequence
	expectedSequence := []string{
		operations.EventTypePipelineReset,
		operations.EventTypeOperationStatus,  // operation started
		operations.EventTypePipelineProgress, // Multiple progress updates
		operations.EventTypePipelineComplete,
	}
	
	// Check minimum expected messages
	if len(messages) < len(expectedSequence) {
		t.Errorf("Expected at least %d messages, got %d", len(expectedSequence), len(messages))
	}
	
	// Verify first message is reset
	if messages[0].EventType != operations.EventTypePipelineReset {
		t.Errorf("First message should be reset, got %s", messages[0].EventType)
	}
	
	// Verify last message is complete
	lastMsg := messages[len(messages)-1]
	if lastMsg.EventType != operations.EventTypePipelineComplete {
		t.Errorf("Last message should be complete, got %s", lastMsg.EventType)
	}
	
	// Verify progress messages have required fields
	progressMessages := hub.GetMessagesByType(operations.EventTypePipelineProgress)
	for _, msg := range progressMessages {
		metadata, ok := msg.Metadata.(map[string]interface{})
		if !ok {
			t.Error("Progress message metadata should be a map")
			continue
		}
		
		// Check required fields
		requiredFields := []string{"operation_id", "Step", "status", "progress"}
		for _, field := range requiredFields {
			if _, ok := metadata[field]; !ok {
				t.Errorf("Progress message missing required field: %s", field)
			}
		}
	}
}

// TestWebSocketProgressUpdates verifies progress messages match frontend expectations
func TestWebSocketProgressUpdates(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create a Step that sends specific progress updates
	progressStage := testutil.NewStageBuilder("test-progress", "Progress Test").
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage("test-progress")
			
			// Simulate progress updates like the real scraper
			updates := []struct {
				progress float64
				message  string
			}{
				{0, "Starting..."},
				{25, "Connecting to ISX website..."},
				{50, "Downloading reports..."},
				{75, "Processing data..."},
				{100, "Completed"},
			}
			
			for _, update := range updates {
				StepState.UpdateProgress(update.progress, update.message)
				time.Sleep(10 * time.Millisecond)
			}
			
			return nil
		}).
		Build()
	
	testutil.AssertNoError(t, manager.RegisterStage(progressStage))
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "progress-test"}
	
	_, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Check progress messages
	progressMessages := hub.GetMessagesByType(operations.EventTypePipelineProgress)
	
	// Should have at least 2 progress updates (start and complete)
	if len(progressMessages) < 2 {
		t.Errorf("Expected at least 2 progress messages, got %d", len(progressMessages))
	}
	
	// Verify progress values are increasing
	var lastProgress float64 = -1
	for _, msg := range progressMessages {
		metadata := msg.Metadata.(map[string]interface{})
		progress, ok := metadata["progress"].(float64)
		if !ok {
			t.Error("Progress should be a float64")
			continue
		}
		
		if progress < lastProgress {
			t.Errorf("Progress decreased: %f -> %f", lastProgress, progress)
		}
		lastProgress = progress
	}
}

// TestWebSocketErrorMessages verifies error reporting to frontend
func TestWebSocketErrorMessages(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create a failing Step
	failStage := testutil.CreateFailingStage("fail-Step", "Fail Step", 
		operations.NewExecutionError("fail-Step", errors.New("simulated failure"), false))
	
	testutil.AssertNoError(t, manager.RegisterStage(failStage))
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "error-test"}
	
	_, err := manager.Execute(ctx, req)
	testutil.AssertError(t, err, true)
	
	// Check for error message
	errorMessages := hub.GetMessagesByType(operations.EventTypeOperationError)
	if len(errorMessages) != 1 {
		t.Errorf("Expected 1 error message, got %d", len(errorMessages))
	}
	
	if len(errorMessages) > 0 {
		metadata := errorMessages[0].Metadata.(map[string]interface{})
		
		// Should have error details
		if _, ok := metadata["error"]; !ok {
			t.Error("Error message should contain error details")
		}
	}
}

// TestWebSocketStageTransitions verifies Step status transitions
func TestWebSocketStageTransitions(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Track Step transitions
	stageTransitions := make(map[string][]string)
	
	// Create a Step that we can monitor
	monitoredStage := testutil.NewStageBuilder("monitored", "Monitored Step").
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		}).
		Build()
	
	testutil.AssertNoError(t, manager.RegisterStage(monitoredStage))
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{ID: "transition-test"}
	
	_, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Analyze progress messages for Step transitions
	progressMessages := hub.GetMessagesByType(operations.EventTypePipelineProgress)
	
	for _, msg := range progressMessages {
		metadata := msg.Metadata.(map[string]interface{})
		if Step, ok := metadata["Step"].(string); ok {
			if status, ok := metadata["status"].(operations.StepStatus); ok {
				stageTransitions[Step] = append(stageTransitions[Step], string(status))
			}
		}
	}
	
	// Verify Step went through expected transitions
	transitions := stageTransitions["monitored"]
	expectedTransitions := []string{
		string(operations.StepStatusActive),
		string(operations.StepStatusCompleted),
	}
	
	if len(transitions) < len(expectedTransitions) {
		t.Errorf("Expected at least %d transitions, got %d", len(expectedTransitions), len(transitions))
	}
}

// TestFrontendCompatibleMessages ensures messages are compatible with current frontend
func TestFrontendCompatibleMessages(t *testing.T) {
	hub := &testutil.MockWebSocketHub{}
	manager := operations.NewManager(hub, nil, nil)
	
	config := testutil.CreateTestConfig()
	manager.SetConfig(config)
	
	// Create steps that mimic real operation
	scrapingStage := createMockScrapingStage()
	processingStage := createMockProcessingStage()
	
	testutil.AssertNoError(t, manager.RegisterStage(scrapingStage))
	testutil.AssertNoError(t, manager.RegisterStage(processingStage))
	
	// Execute
	ctx := context.Background()
	req := operations.OperationRequest{
		ID:       "frontend-compat-test",
		Mode:     operations.ModeInitial,
		FromDate: "2024-01-01",
		ToDate:   "2024-01-31",
	}
	
	_, err := manager.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	
	// Verify specific message formats the frontend expects
	_ = hub.GetMessages()
	
	// Check operation_status messages
	statusMessages := hub.GetMessagesByType(operations.EventTypeOperationStatus)
	for _, msg := range statusMessages {
		metadata := msg.Metadata.(map[string]interface{})
		
		// Must have operation_id
		if _, ok := metadata["operation_id"]; !ok {
			t.Error("operation_status message missing operation_id")
		}
		
		// Must have status
		if _, ok := metadata["status"]; !ok {
			t.Error("operation_status message missing status")
		}
	}
	
	// Check operation_complete message format
	completeMessages := hub.GetMessagesByType(operations.EventTypePipelineComplete)
	if len(completeMessages) > 0 {
		metadata := completeMessages[0].Metadata.(map[string]interface{})
		
		// Frontend expects these fields
		expectedFields := []string{"operation_id", "status"}
		for _, field := range expectedFields {
			if _, ok := metadata[field]; !ok {
				t.Errorf("operation_complete missing expected field: %s", field)
			}
		}
	}
}

// Helper functions to create realistic steps

func createRealisticPipelineStages() []operations.Step {
	return []operations.Step{
		createMockScrapingStage(),
		createMockProcessingStage(),
		createMockIndicesStage(),
		createMockAnalysisStage(),
	}
}

func createMockScrapingStage() *testutil.MockStage {
	return testutil.NewStageBuilder(operations.StageIDScraping, operations.StageNameScraping).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage(operations.StageIDScraping)
			
			// Simulate real scraper progress
			StepState.UpdateProgress(0, "Initializing scraper...")
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(25, "Navigating to ISX website...")
			StepState.Metadata["url"] = "https://www.isx-iq.net"
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(50, "Downloading reports for January 2024...")
			StepState.Metadata["current_file"] = "2024 01 15 ISX Daily Report.xlsx"
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(75, "Saving files to disk...")
			StepState.Metadata["files_downloaded"] = 20
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(100, "Scraping completed successfully")
			state.SetContext(operations.ContextKeyScraperSuccess, true)
			
			return nil
		}).
		Build()
}

func createMockProcessingStage() *testutil.MockStage {
	return testutil.NewStageBuilder(operations.StageIDProcessing, operations.StageNameProcessing).
		WithDependencies(operations.StageIDScraping).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage(operations.StageIDProcessing)
			
			// Simulate processing multiple files
			totalFiles := 20
			for i := 0; i < totalFiles; i++ {
				progress := float64(i+1) / float64(totalFiles) * 100
				StepState.UpdateProgress(progress, fmt.Sprintf("Processing file %d of %d", i+1, totalFiles))
				StepState.Metadata["current_file"] = fmt.Sprintf("2024 01 %02d ISX Daily Report.xlsx", i+1)
				StepState.Metadata["records_processed"] = (i + 1) * 150
				time.Sleep(5 * time.Millisecond)
			}
			
			return nil
		}).
		Build()
}

func createMockIndicesStage() *testutil.MockStage {
	return testutil.NewStageBuilder(operations.StageIDIndices, operations.StageNameIndices).
		WithDependencies(operations.StageIDProcessing).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage(operations.StageIDIndices)
			
			StepState.UpdateProgress(33, "Extracting ISX60 index...")
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(66, "Extracting ISX15 index...")
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(100, "Index extraction completed")
			StepState.Metadata["indices_extracted"] = 2
			
			return nil
		}).
		Build()
}

func createMockAnalysisStage() *testutil.MockStage {
	return testutil.NewStageBuilder(operations.StageIDLiquidity, operations.StageNameLiquidity).
		WithDependencies(operations.StageIDIndices).
		WithExecute(func(ctx context.Context, state *operations.OperationState) error {
			StepState := state.GetStage(operations.StageIDLiquidity)
			
			StepState.UpdateProgress(50, "Calculating ticker statistics...")
			time.Sleep(20 * time.Millisecond)
			
			StepState.UpdateProgress(100, "Analysis completed")
			StepState.Metadata["tickers_analyzed"] = 104
			
			return nil
		}).
		Build()
}