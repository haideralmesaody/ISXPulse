package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"isxcli/internal/operations"
)

// IntegrationTestHarness provides utilities for integration testing
type IntegrationTestHarness struct {
	t         *testing.T
	manager   *operations.Manager
	hub       *MockWebSocketHub
	logger    *slog.Logger
	handler   *MockSlogHandler
	dataGen   *TestDataGenerator
	baseDir   string
}

// NewIntegrationTestHarness creates a new test harness
func NewIntegrationTestHarness(t *testing.T) *IntegrationTestHarness {
	baseDir := CreateTestDirectory(t, "integration-test")
	hub := &MockWebSocketHub{}
	logger, handler := CreateTestSlogLogger()
	manager := operations.NewManager(hub, nil, nil)
	
	return &IntegrationTestHarness{
		t:       t,
		manager: manager,
		hub:     hub,
		logger:  logger,
		handler: handler,
		dataGen: NewTestDataGenerator(t, baseDir),
		baseDir: baseDir,
	}
}

// SetupStandardPipeline sets up a standard 4-step operation
func (h *IntegrationTestHarness) SetupStandardPipeline() {
	// Create standard steps with test implementations
	scrapingStage := CreateSuccessfulStage(operations.StageIDScraping, operations.StageNameScraping)
	processingStage := CreateSuccessfulStage(operations.StageIDProcessing, operations.StageNameProcessing, operations.StageIDScraping)
	indicesStage := CreateSuccessfulStage(operations.StageIDIndices, operations.StageNameIndices, operations.StageIDProcessing)
	analysisStage := CreateSuccessfulStage(operations.StageIDLiquidity, operations.StageNameLiquidity, operations.StageIDIndices)
	
	h.manager.RegisterStage(scrapingStage)
	h.manager.RegisterStage(processingStage)
	h.manager.RegisterStage(indicesStage)
	h.manager.RegisterStage(analysisStage)
}

// ExecutePipeline executes a operation with standard configuration
func (h *IntegrationTestHarness) ExecutePipeline(fromDate, toDate string) (*operations.OperationResponse, error) {
	req := operations.OperationRequest{
		ID:       "test-operation",
		Mode:     operations.ModeInitial,
		FromDate: fromDate,
		ToDate:   toDate,
		Parameters: map[string]interface{}{
			operations.ContextKeyDownloadDir: h.baseDir + "/downloads",
			operations.ContextKeyReportDir:   h.baseDir + "/reports",
		},
	}
	
	ctx := context.Background()
	return h.manager.Execute(ctx, req)
}

// ExecutePipelineWithTimeout executes a operation with a timeout
func (h *IntegrationTestHarness) ExecutePipelineWithTimeout(fromDate, toDate string, timeout time.Duration) (*operations.OperationResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	req := operations.OperationRequest{
		ID:       "test-operation-timeout",
		Mode:     operations.ModeInitial,
		FromDate: fromDate,
		ToDate:   toDate,
		Parameters: map[string]interface{}{
			operations.ContextKeyDownloadDir: h.baseDir + "/downloads",
			operations.ContextKeyReportDir:   h.baseDir + "/reports",
		},
	}
	
	return h.manager.Execute(ctx, req)
}

// GetManager returns the operation manager
func (h *IntegrationTestHarness) GetManager() *operations.Manager {
	return h.manager
}

// GetHub returns the mock WebSocket hub
func (h *IntegrationTestHarness) GetHub() *MockWebSocketHub {
	return h.hub
}

// GetLogger returns the slog logger
func (h *IntegrationTestHarness) GetLogger() *slog.Logger {
	return h.logger
}

// GetLogHandler returns the mock slog handler for test assertions
func (h *IntegrationTestHarness) GetLogHandler() *MockSlogHandler {
	return h.handler
}

// GetDataGenerator returns the test data generator
func (h *IntegrationTestHarness) GetDataGenerator() *TestDataGenerator {
	return h.dataGen
}

// GetBaseDir returns the base test directory
func (h *IntegrationTestHarness) GetBaseDir() string {
	return h.baseDir
}

// AssertPipelineSuccess verifies a operation completed successfully
func (h *IntegrationTestHarness) AssertPipelineSuccess(resp *operations.OperationResponse, err error) {
	h.t.Helper()
	
	AssertNoError(h.t, err)
	if resp.Status != operations.OperationStatusCompleted {
		h.t.Errorf("operation status = %v, want %v", resp.Status, operations.OperationStatusCompleted)
	}
	
	// Verify all steps completed
	for _, step := range resp.Steps {
		AssertStepStatus(h.t, step, operations.StepStatusCompleted)
		AssertProgress(h.t, step, 100)
	}
}

// AssertWebSocketMessages verifies expected WebSocket messages were sent
func (h *IntegrationTestHarness) AssertWebSocketMessages() {
	h.t.Helper()
	
	// Check for required message types
	AssertWebSocketMessage(h.t, h.hub, operations.EventTypePipelineReset)
	AssertWebSocketMessage(h.t, h.hub, operations.EventTypeOperationStatus)
	AssertWebSocketMessage(h.t, h.hub, operations.EventTypePipelineProgress)
	AssertWebSocketMessage(h.t, h.hub, operations.EventTypePipelineComplete)
}

// ClearMessages clears all captured messages
func (h *IntegrationTestHarness) ClearMessages() {
	h.hub.Clear()
	h.handler.Clear()
}

// WaitForPipelineCompletion waits for a operation to complete
func (h *IntegrationTestHarness) WaitForPipelineCompletion(ctx context.Context, pipelineID string, timeout time.Duration) (*operations.OperationState, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		state, err := h.manager.GetOperation(pipelineID)
		if err == nil && (state.Status == operations.OperationStatusCompleted || 
			state.Status == operations.OperationStatusFailed ||
			state.Status == operations.OperationStatusCancelled) {
			return state, nil
		}
		// Use context-aware timing instead of time.Sleep
		timer := time.NewTimer(10 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			// Continue polling
		}
	}
	
	return nil, &TimeoutError{Message: "operation did not complete in time"}
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	Message string
}

func (e *TimeoutError) Error() string {
	return e.Message
}

// RunConcurrentPipelines runs multiple operations concurrently
func (h *IntegrationTestHarness) RunConcurrentPipelines(count int) []error {
	errors := make(chan error, count)
	
	for i := 0; i < count; i++ {
		go func(n int) {
			req := operations.OperationRequest{
				ID:       fmt.Sprintf("concurrent-%d", n),
				Mode:     operations.ModeInitial,
				FromDate: "2024-01-01",
				ToDate:   "2024-01-05",
				Parameters: map[string]interface{}{
					operations.ContextKeyDownloadDir: h.baseDir + "/downloads",
					operations.ContextKeyReportDir:   h.baseDir + "/reports",
				},
			}
			
			_, err := h.manager.Execute(context.Background(), req)
			errors <- err
		}(i)
	}
	
	// Collect results
	var results []error
	for i := 0; i < count; i++ {
		results = append(results, <-errors)
	}
	
	return results
}

// SimulateStageFailure creates a step that will fail
func (h *IntegrationTestHarness) SimulateStageFailure(stageID string, errorMsg string) {
	failingStage := CreateFailingStage(stageID, "Failing "+stageID, fmt.Errorf(errorMsg))
	
	// Replace existing step if present
	steps := []operations.Step{
		failingStage,
	}
	
	// Re-register all steps
	for _, step := range steps {
		h.manager.RegisterStage(step)
	}
}