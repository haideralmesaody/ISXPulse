package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	
	"isxcli/internal/operations"
)

// Test helper functions
func TestOperationHelperFunctions(t *testing.T) {
	t.Run("getValue", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": nil,
		}

		assert.Equal(t, "value1", getValue(m, "key1", "default"))
		assert.Equal(t, 42, getValue(m, "key2", 0))
		assert.Equal(t, "default", getValue(m, "key3", "default"))
		assert.Equal(t, "missing", getValue(m, "key4", "missing"))
	})

	t.Run("getStageDescription", func(t *testing.T) {
		assert.Contains(t, getStageDescription(operations.StageIDScraping), "Download")
		assert.Contains(t, getStageDescription(operations.StageIDProcessing), "Convert")
		assert.Contains(t, getStageDescription(operations.StageIDIndices), "Extract")
		assert.Contains(t, getStageDescription(operations.StageIDLiquidity), "Calculate")
		assert.Equal(t, "Process data", getStageDescription("unknown"))
	})

	t.Run("getStageParameters", func(t *testing.T) {
		scrapingParams := getStageParameters(operations.StageIDScraping)
		assert.Len(t, scrapingParams, 3)
		assert.Equal(t, "mode", scrapingParams[0].Name)
		assert.Equal(t, "from", scrapingParams[1].Name)
		assert.Equal(t, "to", scrapingParams[2].Name)

		processingParams := getStageParameters(operations.StageIDProcessing)
		assert.Len(t, processingParams, 1)
		assert.Equal(t, "input_dir", processingParams[0].Name)

		unknownParams := getStageParameters("unknown")
		assert.Empty(t, unknownParams)
	})
}

// TestWebSocketOperationAdapter tests the WebSocket adapter
func TestWebSocketOperationAdapterUnit(t *testing.T) {
	mockHub := new(MockWebSocketHub)
	adapter := NewWebSocketOperationAdapter(mockHub)

	t.Run("SendProgress", func(t *testing.T) {
		expectedData := map[string]interface{}{
			"step":    "scraping",
			"message":  "Processing files",
			"progress": 50,
			"status":   "active",
		}
		mockHub.On("Broadcast", "operation_progress", expectedData).Once()

		adapter.SendProgress("scraping", "Processing files", 50)

		mockHub.AssertExpectations(t)
	})

	t.Run("SendComplete Success", func(t *testing.T) {
		expectedData := map[string]interface{}{
			"step":   "processing",
			"message": "Completed successfully",
			"status":  "completed",
			"success": true,
		}
		mockHub.On("Broadcast", "operation_complete", expectedData).Once()

		adapter.SendComplete("processing", "Completed successfully", true)

		mockHub.AssertExpectations(t)
	})

	t.Run("SendComplete Failure", func(t *testing.T) {
		expectedData := map[string]interface{}{
			"step":   "indices",
			"message": "Failed to extract",
			"status":  "failed",
			"success": false,
		}
		mockHub.On("Broadcast", "operation_complete", expectedData).Once()

		adapter.SendComplete("indices", "Failed to extract", false)

		mockHub.AssertExpectations(t)
	})

	t.Run("SendError", func(t *testing.T) {
		expectedData := map[string]interface{}{
			"step": "liquidity",
			"error": "File not found",
			"status": "error",
		}
		mockHub.On("Broadcast", "operation_error", expectedData).Once()

		adapter.SendError("liquidity", "File not found")

		mockHub.AssertExpectations(t)
	})

	t.Run("BroadcastUpdate", func(t *testing.T) {
		metadata := map[string]interface{}{"count": 10}
		expectedData := map[string]interface{}{
			"eventType": "custom_event",
			"step":     "validation",
			"status":    "running",
			"metadata":  metadata,
		}
		mockHub.On("Broadcast", "custom_event", expectedData).Once()

		adapter.BroadcastUpdate("custom_event", "validation", "running", metadata)

		mockHub.AssertExpectations(t)
	})

	t.Run("BroadcastUpdate No Metadata", func(t *testing.T) {
		expectedData := map[string]interface{}{
			"eventType": "status_update",
			"step":     "cleanup",
			"status":    "pending",
		}
		mockHub.On("Broadcast", "status_update", expectedData).Once()

		adapter.BroadcastUpdate("status_update", "cleanup", "pending", nil)

		mockHub.AssertExpectations(t)
	})
}

// TestGetStageInfo tests the stage info method
func TestGetStageInfo(t *testing.T) {
	service := &OperationService{}
	info := service.GetStageInfo()
	
	steps, ok := info["steps"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, steps, 4)
	
	// Check first step
	assert.Equal(t, "scraping", steps[0]["id"])
	assert.Equal(t, "Scraping", steps[0]["name"])
	assert.Equal(t, "scraper.exe", steps[0]["executable"])
}

// BenchmarkGetValue benchmarks the getValue helper
func BenchmarkGetValue(b *testing.B) {
	m := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
		"key4": 3.14,
		"key5": []string{"a", "b", "c"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getValue(m, "key1", "default")
		_ = getValue(m, "key2", 0)
		_ = getValue(m, "missing", "default")
	}
}

// BenchmarkGetStageDescription benchmarks stage description lookup
func BenchmarkGetStageDescription(b *testing.B) {
	stages := []string{
		operations.StageIDScraping,
		operations.StageIDProcessing,
		operations.StageIDIndices,
		operations.StageIDLiquidity,
		"unknown",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, stage := range stages {
			_ = getStageDescription(stage)
		}
	}
}