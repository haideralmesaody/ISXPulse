package operations_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"isxcli/internal/operations"
)

// Helper function to get map keys for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Enhanced mock WebSocket hub for testing WebSocket functionality
type detailedMockWebSocketHub struct {
	updates []webSocketUpdate
}

type webSocketUpdate struct {
	eventType string
	Step     string
	status    string
	metadata  interface{}
	timestamp time.Time
}

func (m *detailedMockWebSocketHub) BroadcastUpdate(eventType, Step, status string, metadata interface{}) {
	m.updates = append(m.updates, webSocketUpdate{
		eventType: eventType,
		Step:     Step,
		status:    status,
		metadata:  metadata,
		timestamp: time.Now(),
	})
}

func (m *detailedMockWebSocketHub) GetUpdatesByEventType(eventType string) []webSocketUpdate {
	var result []webSocketUpdate
	for _, update := range m.updates {
		if update.eventType == eventType {
			result = append(result, update)
		}
	}
	return result
}

func (m *detailedMockWebSocketHub) GetUpdatesByStage(Step string) []webSocketUpdate {
	var result []webSocketUpdate
	for _, update := range m.updates {
		if update.Step == Step {
			result = append(result, update)
		}
	}
	return result
}

func (m *detailedMockWebSocketHub) GetAllUpdates() []webSocketUpdate {
	return m.updates
}

func (m *detailedMockWebSocketHub) Clear() {
	m.updates = nil
}

// TestManagerWebSocketBroadcasting tests WebSocket update broadcasting
func TestManagerWebSocketBroadcasting(t *testing.T) {
	mockWS := &detailedMockWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a Step that will succeed
	Step := newMockManagerStage("websocket-test", "WebSocket Test Step", nil)
	manager.RegisterStage(Step)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := operations.OperationRequest{
		ID:   "websocket-test-operation",
		Mode: "test",
	}

	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify all expected WebSocket update types were sent
	updates := mockWS.GetAllUpdates()
	if len(updates) == 0 {
		t.Fatal("Expected WebSocket updates but got none")
	}

	// Check for operation reset event
	resetUpdates := mockWS.GetUpdatesByEventType(operations.EventTypePipelineReset)
	if len(resetUpdates) == 0 {
		t.Error("Expected operation reset event")
	}

	// Check for operation status events
	statusUpdates := mockWS.GetUpdatesByEventType(operations.EventTypeOperationStatus)
	if len(statusUpdates) < 1 {
		t.Error("Expected at least one operation status event")
	}

	// Check for operation progress events
	progressUpdates := mockWS.GetUpdatesByEventType(operations.EventTypePipelineProgress)
	if len(progressUpdates) == 0 {
		t.Error("Expected operation progress events")
	}

	// Check for operation complete event
	completeUpdates := mockWS.GetUpdatesByEventType(operations.EventTypePipelineComplete)
	if len(completeUpdates) == 0 {
		t.Error("Expected operation complete event")
	}

	// Verify Step-specific updates
	stageUpdates := mockWS.GetUpdatesByStage("websocket-test")
	if len(stageUpdates) == 0 {
		t.Error("Expected Step-specific updates")
	}

	t.Logf("Total WebSocket updates: %d", len(updates))
}

// TestManagerWebSocketUpdateContent tests the content of WebSocket messages
func TestManagerWebSocketUpdateContent(t *testing.T) {
	mockWS := &detailedMockWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a Step with metadata
	Step := newMockManagerStage("content-test", "Content Test Step", nil)
	manager.RegisterStage(Step)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := operations.OperationRequest{
		ID:   "content-test-operation",
		Mode: "test",
		Parameters: map[string]interface{}{
			"test_param": "test_value",
		},
	}

	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	updates := mockWS.GetAllUpdates()

	// Test operation status update content
	statusUpdates := mockWS.GetUpdatesByEventType(operations.EventTypeOperationStatus)
	if len(statusUpdates) > 0 {
		statusUpdate := statusUpdates[0]
		if metadata, ok := statusUpdate.metadata.(map[string]interface{}); ok {
			if pipelineID, exists := metadata["operation_id"]; !exists || pipelineID != "content-test-operation" {
				t.Errorf("Expected operation_id in status metadata, got: %v", metadata)
			}
			if status, exists := metadata["status"]; !exists {
				t.Error("Expected status in status metadata")
			} else {
				t.Logf("operation status: %v", status)
			}
		} else {
			t.Error("Expected metadata to be a map")
		}
	}

	// Test operation progress update content
	progressUpdates := mockWS.GetUpdatesByEventType(operations.EventTypePipelineProgress)
	if len(progressUpdates) > 0 {
		progressUpdate := progressUpdates[0]
		if metadata, ok := progressUpdate.metadata.(map[string]interface{}); ok {
			if pipelineID, exists := metadata["operation_id"]; !exists || pipelineID != "content-test-operation" {
				t.Errorf("Expected operation_id in progress metadata, got: %v", metadata)
			}
			if Step, exists := metadata["Step"]; !exists || Step != "content-test" {
				t.Errorf("Expected Step 'content-test' in progress metadata, got: %v", Step)
			}
		} else {
			t.Error("Expected progress metadata to be a map")
		}
	}

	t.Logf("Verified content of %d WebSocket updates", len(updates))
}

// TestManagerWebSocketErrorHandling tests WebSocket updates during error scenarios
func TestManagerWebSocketErrorHandling(t *testing.T) {
	mockWS := &detailedMockWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a Step that will fail
	Step := newMockManagerStage("error-test", "Error Test Step", nil).
		WithFailure(fmt.Errorf("test error"))
	manager.RegisterStage(Step)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := operations.OperationRequest{
		ID:   "error-test-operation",
		Mode: "test",
	}

	_, err := manager.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error from failing Step")
	}

	// Verify error-related WebSocket updates
	updates := mockWS.GetAllUpdates()
	if len(updates) == 0 {
		t.Fatal("Expected WebSocket updates even for failed operation")
	}

	// Check for operation error event
	errorUpdates := mockWS.GetUpdatesByEventType(operations.EventTypeOperationError)
	if len(errorUpdates) == 0 {
		t.Error("Expected operation error event")
	} else {
		errorUpdate := errorUpdates[0]
		if metadata, ok := errorUpdate.metadata.(map[string]interface{}); ok {
			if errorMsg, exists := metadata["error"]; !exists {
				t.Error("Expected error message in error metadata")
			} else {
				t.Logf("Error message: %v", errorMsg)
			}
		}
	}

	// Verify Step status updates for failed Step
	stageUpdates := mockWS.GetUpdatesByStage("error-test")
	if len(stageUpdates) == 0 {
		t.Error("Expected Step-specific updates for failed Step")
	}

	t.Logf("Verified error handling in %d WebSocket updates", len(updates))
}

// TestManagerOperationResponse tests the createResponse functionality
func TestManagerOperationResponse(t *testing.T) {
	tests := []struct {
		name           string
		stageFailure   error
		expectError    bool
		expectedStatus operations.OperationStatus
	}{
		{
			name:           "successful operation response",
			stageFailure:   nil,
			expectError:    false,
			expectedStatus: operations.OperationStatusCompleted,
		},
		{
			name:           "failed operation response",
			stageFailure:   fmt.Errorf("Step failure"),
			expectError:    true,
			expectedStatus: operations.OperationStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWS := &detailedMockWebSocketHub{}
			config := operations.NewConfig()
			registry := operations.NewRegistry()
			manager := operations.NewManager(mockWS, registry, config)

			// Add Step based on test case
			Step := newMockManagerStage("response-test", "Response Test Step", nil)
			if tt.stageFailure != nil {
				Step = Step.WithFailure(tt.stageFailure)
			}
			manager.RegisterStage(Step)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := operations.OperationRequest{
				ID:   "response-test-operation",
				Mode: "test",
			}

			response, err := manager.Execute(ctx, req)

			// Verify error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify response structure
			if response == nil {
				t.Fatal("Expected response but got nil")
			}

			if response.ID != req.ID {
				t.Errorf("Expected response ID %s, got %s", req.ID, response.ID)
			}

			if response.Status != tt.expectedStatus {
				t.Errorf("Expected response status %s, got %s", tt.expectedStatus, response.Status)
			}

			// Verify duration is set
			if response.Duration <= 0 {
				t.Error("Expected positive duration in response")
			}

			// Verify steps are included in response
			if len(response.Steps) == 0 {
				t.Error("Expected steps in response")
			}

			// For error cases, verify error message is included
			if tt.expectError && response.Error == "" {
				t.Error("Expected error message in response for failed operation")
			}

			t.Logf("Response: ID=%s, Status=%s, Duration=%v, steps=%d, Error=%s",
				response.ID, response.Status, response.Duration, len(response.Steps), response.Error)
		})
	}
}

// TestManagerStageWebSocketIntegration tests Step-level WebSocket integration
func TestManagerStageWebSocketIntegration(t *testing.T) {
	mockWS := &detailedMockWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add multiple steps to test Step transitions
	stage1 := newMockManagerStage("stage1", "Step 1", nil)
	stage2 := newMockManagerStage("stage2", "Step 2", []string{"stage1"})
	stage3 := newMockManagerStage("stage3", "Step 3", []string{"stage2"})

	manager.RegisterStage(stage1)
	manager.RegisterStage(stage2)
	manager.RegisterStage(stage3)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := operations.OperationRequest{
		ID:   "multi-Step-test-operation",
		Mode: "test",
	}

	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify each Step got WebSocket updates
	for _, stageID := range []string{"stage1", "stage2", "stage3"} {
		stageUpdates := mockWS.GetUpdatesByStage(stageID)
		if len(stageUpdates) == 0 {
			t.Errorf("Expected WebSocket updates for Step %s", stageID)
		} else {
			t.Logf("Step %s had %d WebSocket updates", stageID, len(stageUpdates))
		}
	}

	// Verify timeline of updates (steps should execute in dependency order)
	allUpdates := mockWS.GetAllUpdates()
	stageExecutionOrder := []string{}
	
	for _, update := range allUpdates {
		if update.eventType == "operation:start" && update.Step != "" {
			stageExecutionOrder = append(stageExecutionOrder, update.Step)
		}
	}

	// Should see steps in dependency order: stage1, stage2, stage3
	expectedOrder := []string{"stage1", "stage2", "stage3"}
	if len(stageExecutionOrder) >= 3 {
		for i, expectedStage := range expectedOrder {
			if i < len(stageExecutionOrder) && stageExecutionOrder[i] != expectedStage {
				t.Errorf("Expected Step %s at position %d, got %s", expectedStage, i, stageExecutionOrder[i])
			}
		}
	}

	t.Logf("Total operation WebSocket updates: %d", len(allUpdates))
	t.Logf("Step execution order: %v", stageExecutionOrder)
}

// TestManagerWebSocketMetadata tests WebSocket metadata content
func TestManagerWebSocketMetadata(t *testing.T) {
	mockWS := &detailedMockWebSocketHub{}
	config := operations.NewConfig()
	registry := operations.NewRegistry()
	manager := operations.NewManager(mockWS, registry, config)

	// Add a Step 
	Step := newMockManagerStage("metadata-test", "Metadata Test Step", nil)
	manager.RegisterStage(Step)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := operations.OperationRequest{
		ID:         "metadata-test-operation",
		Mode:       "test",
		FromDate:   "2024-01-01",
		ToDate:     "2024-01-31",
		Parameters: map[string]interface{}{
			"custom_param": "custom_value",
		},
	}

	_, err := manager.Execute(ctx, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Examine metadata in different update types
	updates := mockWS.GetAllUpdates()

	// Test that operation-level metadata includes correct information
	statusUpdates := mockWS.GetUpdatesByEventType(operations.EventTypeOperationStatus)
	operationLevelUpdates := 0
	if len(statusUpdates) > 0 {
		for _, update := range statusUpdates {
			if metadata, ok := update.metadata.(map[string]interface{}); ok {
				// Only check operation-level updates (those without a "Step" field)
				if _, hasStageField := metadata["Step"]; hasStageField {
					continue // Skip Step-specific updates
				}
				
				operationLevelUpdates++
				
				// Should contain operation_id
				if pipelineID, exists := metadata["operation_id"]; !exists || pipelineID != req.ID {
					t.Errorf("Expected operation_id %s in metadata, got: %v", req.ID, pipelineID)
				}

				// Should contain status
				if _, exists := metadata["status"]; !exists {
					t.Error("Expected status in operation metadata")
				}

				// Should contain steps information
				if _, exists := metadata["steps"]; !exists {
					t.Error("Expected steps in operation metadata")
				}
			}
		}
	}
	
	if operationLevelUpdates == 0 {
		t.Log("No operation-level status updates found")
	}

	// Test that progress updates include Step information
	progressUpdates := mockWS.GetUpdatesByEventType(operations.EventTypePipelineProgress)
	if len(progressUpdates) > 0 {
		for _, update := range progressUpdates {
			if metadata, ok := update.metadata.(map[string]interface{}); ok {
				// Should contain operation_id
				if pipelineID, exists := metadata["operation_id"]; !exists || pipelineID != req.ID {
					t.Errorf("Expected operation_id %s in progress metadata, got: %v", req.ID, pipelineID)
				}

				// Should contain Step
				if Step, exists := metadata["Step"]; !exists || Step == "" {
					t.Error("Expected non-empty Step in progress metadata")
				}

				// Should contain progress
				if _, exists := metadata["progress"]; !exists {
					t.Error("Expected progress in progress metadata")
				}
			}
		}
	}

	t.Logf("Verified metadata in %d WebSocket updates", len(updates))
}