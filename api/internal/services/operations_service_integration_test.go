package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	
	"isxcli/internal/operations"
)

// TestOperationServiceIntegration tests the OperationService with real dependencies
func TestOperationServiceIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	t.Run("CreateAndExecuteOperation", func(t *testing.T) {
		// Create a mock WebSocket hub
		mockHub := new(MockWebSocketHub)
		adapter := NewWebSocketOperationAdapter(mockHub)
		
		// Create the service
		service, err := NewOperationService(adapter, nil)
		require.NoError(t, err)
		require.NotNil(t, service)
		require.NotNil(t, service.manager)
		require.NotNil(t, service.paths)
		
		// Test operation execution
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Set up expectations - using mock.Anything from testify
		// The operations manager sends various event types
		mockHub.On("Broadcast", mock.Anything, mock.Anything).Maybe()
		
		// Start an operation
		params := map[string]interface{}{
			"mode": "test",
		}
		
		id, err := service.StartOperation(ctx, params)
		
		// We expect it to fail because executables don't exist in test environment
		// The error should indicate step execution failed
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Step execution failed")
		// ID might be empty on failure
		_ = id
	})
}

// TestGetOperationTypes tests the operation type listing
func TestGetOperationTypes(t *testing.T) {
	// This test can work without full integration
	mockHub := new(MockWebSocketHub)
	adapter := NewWebSocketOperationAdapter(mockHub)
	
	service, err := NewOperationService(adapter, nil)
	require.NoError(t, err)
	
	ctx := context.Background()
	types, err := service.GetOperationTypes(ctx)
	require.NoError(t, err)
	
	// Should have 4 stage types + 1 full_pipeline
	assert.Len(t, types, 5)
	
	// Check stage types
	stageIDs := make(map[string]bool)
	for _, opType := range types {
		stageIDs[opType.ID] = true
	}
	
	assert.True(t, stageIDs[operations.StageIDScraping])
	assert.True(t, stageIDs[operations.StageIDProcessing])
	assert.True(t, stageIDs[operations.StageIDIndices])
	assert.True(t, stageIDs[operations.StageIDAnalysis])
	assert.True(t, stageIDs["full_pipeline"])
}