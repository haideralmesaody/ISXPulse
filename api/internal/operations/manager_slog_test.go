package operations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"isxcli/internal/operations"
)

// logCapture captures structured log output for testing
type logCapture struct {
	buffer *bytes.Buffer
	logger *slog.Logger
}

// newLogCapture creates a new log capture instance
func newLogCapture() *logCapture {
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buffer, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Capture all levels
	}))
	
	return &logCapture{
		buffer: buffer,
		logger: logger,
	}
}

// getLogEntries parses captured log entries
func (lc *logCapture) getLogEntries() []map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(lc.buffer.String()), "\n")
	var entries []map[string]interface{}
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	
	return entries
}

// findLogEntry finds a log entry by action
func (lc *logCapture) findLogEntry(action string) map[string]interface{} {
	entries := lc.getLogEntries()
	for _, entry := range entries {
		if entry["action"] == action {
			return entry
		}
	}
	return nil
}

// TestManagerStructuredLogging tests the Manager's structured logging functions
func TestManagerStructuredLogging(t *testing.T) {
	// Set up log capture
	capture := newLogCapture()
	originalLogger := slog.Default()
	slog.SetDefault(capture.logger)
	defer slog.SetDefault(originalLogger)

	// Create a manager with mocked dependencies
	mockWS := &mockManagerWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a Step to execute
	Step := newMockManagerStage("logging-test", "Logging Test Step", nil)
	manager.RegisterStage(Step)

	ctx := context.Background()
	req := operations.OperationRequest{
		ID:       "test-logging-operation",
		Mode:     "test",
		FromDate: "2024-01-01",
		ToDate:   "2024-01-31",
		Parameters: map[string]interface{}{
			"test_param": "test_value",
		},
	}

	// Execute operation to trigger logging
	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify log entries were created
	entries := capture.getLogEntries()
	if len(entries) == 0 {
		t.Fatal("Expected log entries but got none")
	}

	// Test operation start logging
	t.Run("operation_start_logging", func(t *testing.T) {
		// Note: pipeline_start is not directly logged in the current implementation
		// but we can verify sequential_execution_start which is similar
		entry := capture.findLogEntry("sequential_execution_start")
		if entry == nil {
			t.Fatal("Expected sequential_execution_start log entry")
		}

		// Verify standard fields
		if entry["component"] != "operation" {
			t.Errorf("Expected component 'operation', got %v", entry["component"])
		}
		if entry["operation_id"] != req.ID {
			t.Errorf("Expected operation_id %s, got %v", req.ID, entry["operation_id"])
		}
	})

	// Test Step start logging
	t.Run("stage_start_logging", func(t *testing.T) {
		entry := capture.findLogEntry("stage_start")
		if entry == nil {
			t.Fatal("Expected stage_start log entry")
		}

		// Verify Step-specific fields
		if entry["component"] != "operation" {
			t.Errorf("Expected component 'operation', got %v", entry["component"])
		}
		if entry["operation_id"] != req.ID {
			t.Errorf("Expected operation_id %s, got %v", req.ID, entry["operation_id"])
		}
		if entry["Step"] != "logging-test" {
			t.Errorf("Expected Step 'logging-test', got %v", entry["Step"])
		}
	})

	// Test Step complete logging
	t.Run("stage_complete_logging", func(t *testing.T) {
		entry := capture.findLogEntry("stage_complete")
		if entry == nil {
			t.Fatal("Expected stage_complete log entry")
		}

		// Verify duration is present
		if _, exists := entry["duration"]; !exists {
			t.Error("Expected duration in stage_complete log")
		}

		// Verify Step identification
		if entry["Step"] != "logging-test" {
			t.Errorf("Expected Step 'logging-test', got %v", entry["Step"])
		}
	})

	// Test operation completion logging
	t.Run("operation_completion_logging", func(t *testing.T) {
		entry := capture.findLogEntry("all_stages_completed")
		if entry == nil {
			t.Fatal("Expected all_stages_completed log entry")
		}

		if entry["operation_id"] != req.ID {
			t.Errorf("Expected operation_id %s, got %v", req.ID, entry["operation_id"])
		}
	})

	t.Logf("Total log entries captured: %d", len(entries))
}

// TestManagerErrorLogging tests error logging scenarios
func TestManagerErrorLogging(t *testing.T) {
	// Set up log capture
	capture := newLogCapture()
	originalLogger := slog.Default()
	slog.SetDefault(capture.logger)
	defer slog.SetDefault(originalLogger)

	// Create a manager with a failing Step
	mockWS := &mockManagerWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a Step that will fail
	Step := newMockManagerStage("error-test", "Error Test Step", nil).
		WithFailure(fmt.Errorf("test error for logging"))
	manager.RegisterStage(Step)

	ctx := context.Background()
	req := operations.OperationRequest{
		ID:   "error-logging-operation",
		Mode: "test",
	}

	// Execute operation to trigger error logging
	_, err := manager.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error from failing Step")
	}

	// Test Step error logging
	t.Run("stage_error_logging", func(t *testing.T) {
		entry := capture.findLogEntry("stage_error")
		if entry == nil {
			t.Fatal("Expected stage_error log entry")
		}

		// Verify error details
		if entry["component"] != "operation" {
			t.Errorf("Expected component 'operation', got %v", entry["component"])
		}
		if entry["operation_id"] != req.ID {
			t.Errorf("Expected operation_id %s, got %v", req.ID, entry["operation_id"])
		}
		if entry["Step"] != "error-test" {
			t.Errorf("Expected Step 'error-test', got %v", entry["Step"])
		}

		// Verify error message is present
		if errorMsg, exists := entry["error"]; !exists {
			t.Error("Expected error message in stage_error log")
		} else {
			errorStr := fmt.Sprintf("%v", errorMsg)
			if !strings.Contains(errorStr, "Step execution failed") {
				t.Errorf("Expected error message to contain 'Step execution failed', got: %s", errorStr)
			}
		}
	})

	// Test execution failed logging
	t.Run("stage_execution_failed_logging", func(t *testing.T) {
		entry := capture.findLogEntry("stage_execution_failed")
		if entry == nil {
			t.Fatal("Expected stage_execution_failed log entry")
		}

		// Verify error details
		if entry["Step"] != "error-test" {
			t.Errorf("Expected Step 'error-test', got %v", entry["Step"])
		}

		// Verify duration is present (should be very short for our mock)
		if _, exists := entry["duration"]; !exists {
			t.Error("Expected duration in stage_execution_failed log")
		}
	})

	entries := capture.getLogEntries()
	t.Logf("Total error log entries captured: %d", len(entries))
}

// TestManagerRetryLogging tests retry scenario logging
func TestManagerRetryLogging(t *testing.T) {
	// Set up log capture
	capture := newLogCapture()
	originalLogger := slog.Default()
	slog.SetDefault(capture.logger)
	defer slog.SetDefault(originalLogger)

	// Create a manager with retry configuration
	mockWS := &mockManagerWebSocketHub{}
	config := operations.NewConfigBuilder().
		WithRetryConfig(operations.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			Multiplier:   2.0,
		}).
		Build()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a Step that will fail multiple times with a retryable error
	retryableError := operations.NewExecutionError("retry-test", fmt.Errorf("retryable error"), true)
	Step := newMockManagerStage("retry-test", "Retry Test Step", nil).
		WithFailure(retryableError)
	manager.RegisterStage(Step)

	ctx := context.Background()
	req := operations.OperationRequest{
		ID:   "retry-logging-operation",
		Mode: "test",
	}

	// Execute operation to trigger retry logging
	_, err := manager.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error after retries")
	}

	// Test retry logging
	t.Run("stage_retry_logging", func(t *testing.T) {
		entries := capture.getLogEntries()
		retryEntries := 0
		
		for _, entry := range entries {
			if entry["action"] == "stage_retry" {
				retryEntries++
				
				// Verify retry-specific fields
				if entry["Step"] != "retry-test" {
					t.Errorf("Expected Step 'retry-test', got %v", entry["Step"])
				}
				
				// Should have attempt, max_attempts, delay, error
				if _, exists := entry["attempt"]; !exists {
					t.Error("Expected attempt field in retry log")
				}
				if _, exists := entry["max_attempts"]; !exists {
					t.Error("Expected max_attempts field in retry log")
				}
				if _, exists := entry["delay"]; !exists {
					t.Error("Expected delay field in retry log")
				}
			}
		}

		// Should have 2 retry logs (attempts 1 and 2, since attempt 3 is final)
		if retryEntries < 2 {
			t.Errorf("Expected at least 2 retry log entries, got %d", retryEntries)
		}
	})

	// Test multiple execution attempts logging
	t.Run("multiple_execution_attempts", func(t *testing.T) {
		entries := capture.getLogEntries()
		executeEntries := 0
		
		for _, entry := range entries {
			if entry["action"] == "calling_execute" && entry["Step"] == "retry-test" {
				executeEntries++
				
				// Should have attempt number
				if _, exists := entry["attempt"]; !exists {
					t.Error("Expected attempt field in execute log")
				}
			}
		}

		// Should have 3 execute attempts
		if executeEntries != 3 {
			t.Errorf("Expected 3 execution attempts, got %d", executeEntries)
		}
	})

	entries := capture.getLogEntries()
	t.Logf("Total retry log entries captured: %d", len(entries))
}

// TestManagerProgressLogging tests progress logging functionality
func TestManagerProgressLogging(t *testing.T) {
	// Set up log capture with debug level to catch progress logs
	capture := newLogCapture()
	originalLogger := slog.Default()
	slog.SetDefault(capture.logger)
	defer slog.SetDefault(originalLogger)

	// Create a manager
	mockWS := &mockManagerWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Create a Step that simulates progress
	Step := newMockManagerStage("progress-test", "Progress Test Step", nil)
	manager.RegisterStage(Step)

	ctx := context.Background()
	req := operations.OperationRequest{
		ID:   "progress-logging-operation",
		Mode: "test",
	}

	// Execute operation
	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test that execution logging includes progress information
	t.Run("execution_progress_logging", func(t *testing.T) {
		entries := capture.getLogEntries()
		
		// Look for Step execution progression
		hasStageStart := false
		hasStageComplete := false
		
		for _, entry := range entries {
			if entry["action"] == "stage_start" && entry["Step"] == "progress-test" {
				hasStageStart = true
			}
			if entry["action"] == "stage_complete" && entry["Step"] == "progress-test" {
				hasStageComplete = true
				
				// Verify duration is logged
				if _, exists := entry["duration"]; !exists {
					t.Error("Expected duration in Step completion log")
				}
			}
		}

		if !hasStageStart {
			t.Error("Expected stage_start log entry for progress tracking")
		}
		if !hasStageComplete {
			t.Error("Expected stage_complete log entry for progress tracking")
		}
	})

	entries := capture.getLogEntries()
	t.Logf("Total progress log entries captured: %d", len(entries))
}

// TestManagerComplexLoggingScenario tests a complex multi-Step scenario
func TestManagerComplexLoggingScenario(t *testing.T) {
	// Set up log capture
	capture := newLogCapture()
	originalLogger := slog.Default()
	slog.SetDefault(capture.logger)
	defer slog.SetDefault(originalLogger)

	// Create a manager with multiple steps
	mockWS := &mockManagerWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add multiple steps with dependencies
	stage1 := newMockManagerStage("stage1", "Step 1", nil)
	stage2 := newMockManagerStage("stage2", "Step 2", []string{"stage1"})
	stage3 := newMockManagerStage("stage3", "Step 3", []string{"stage2"})

	manager.RegisterStage(stage1)
	manager.RegisterStage(stage2)
	manager.RegisterStage(stage3)

	ctx := context.Background()
	req := operations.OperationRequest{
		ID:         "complex-logging-operation",
		Mode:       "full",
		FromDate:   "2024-01-01",
		ToDate:     "2024-12-31",
		Parameters: map[string]interface{}{
			"complexity": "high",
			"steps":     3,
		},
	}

	// Execute operation
	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test comprehensive logging coverage
	t.Run("comprehensive_logging_coverage", func(t *testing.T) {
		entries := capture.getLogEntries()
		
		// Count different types of log entries
		counts := make(map[string]int)
		for _, entry := range entries {
			if action, ok := entry["action"].(string); ok {
				counts[action]++
			}
		}

		// Verify we have the expected log types
		expectedActions := []string{
			"sequential_execution_start",
			"executing_stage",
			"stage_start",
			"calling_execute",
			"stage_complete",
			"stage_completed_successfully",
			"all_stages_completed",
		}

		for _, expectedAction := range expectedActions {
			if counts[expectedAction] == 0 {
				t.Errorf("Expected to find log entries for action: %s", expectedAction)
			}
		}

		// Verify Step-specific logging
		stageActions := []string{"stage1", "stage2", "stage3"}
		for _, stageID := range stageActions {
			stageEntries := 0
			for _, entry := range entries {
				if entry["Step"] == stageID {
					stageEntries++
				}
			}
			if stageEntries == 0 {
				t.Errorf("Expected log entries for Step: %s", stageID)
			}
		}

		t.Logf("Log action counts: %+v", counts)
		t.Logf("Total log entries in complex scenario: %d", len(entries))
	})

	// Test log entry structure and consistency
	t.Run("log_structure_consistency", func(t *testing.T) {
		entries := capture.getLogEntries()
		
		for i, entry := range entries {
			// Every entry should have basic fields
			if _, exists := entry["time"]; !exists {
				t.Errorf("Entry %d missing time field", i)
			}
			if _, exists := entry["level"]; !exists {
				t.Errorf("Entry %d missing level field", i)
			}
			if _, exists := entry["msg"]; !exists {
				t.Errorf("Entry %d missing msg field", i)
			}
			
			// operation entries should have component and action
			if entry["component"] == "operation" {
				if _, exists := entry["action"]; !exists {
					t.Errorf("operation entry %d missing action field", i)
				}
				if _, exists := entry["operation_id"]; !exists {
					t.Errorf("operation entry %d missing operation_id field", i)
				}
			}
		}
	})
}