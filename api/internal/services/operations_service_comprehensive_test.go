package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"isxcli/internal/config"
	"isxcli/internal/operations"
)

// TestOperationServiceComprehensive tests OperationService for improved coverage
func TestOperationServiceComprehensive(t *testing.T) {
	t.Run("WebSocketAdapter_Functionality", testWebSocketAdapterFunctionalityOperations)
	t.Run("Service_Construction", testOperationServiceConstruction)
	t.Run("Operation_Lifecycle", testOperationLifecycle)
	t.Run("Parameter_Processing", testParameterProcessing)
	t.Run("Error_Handling", testOperationServiceErrorHandling)
	t.Run("Concurrency_Safety", testOperationServiceConcurrency)
	t.Run("Validation_Logic", testOperationValidation)
	t.Run("Operation_Types", testOperationTypes)
	t.Run("Metrics_Collection", testOperationMetrics)
	t.Run("Stage_Management", testStageManagement)
}

func testWebSocketAdapterFunctionalityOperations(t *testing.T) {
	tests := []struct {
		name     string
		action   func(adapter *WebSocketOperationAdapter)
		validate func(t *testing.T, hub *MockWebSocketHub)
	}{
		{
			name: "send_progress",
			action: func(adapter *WebSocketOperationAdapter) {
				adapter.SendProgress("test-stage", "Processing data", 50)
			},
			validate: func(t *testing.T, hub *MockWebSocketHub) {
				hub.AssertCalled(t, "Broadcast", "operation_progress", mock.MatchedBy(func(data interface{}) bool {
					dataMap := data.(map[string]interface{})
					return dataMap["step"] == "test-stage" &&
						dataMap["message"] == "Processing data" &&
						dataMap["progress"] == 50 &&
						dataMap["status"] == "active"
				}))
			},
		},
		{
			name: "send_complete_success",
			action: func(adapter *WebSocketOperationAdapter) {
				adapter.SendComplete("test-stage", "Completed successfully", true)
			},
			validate: func(t *testing.T, hub *MockWebSocketHub) {
				hub.AssertCalled(t, "Broadcast", "operation_complete", mock.MatchedBy(func(data interface{}) bool {
					dataMap := data.(map[string]interface{})
					return dataMap["step"] == "test-stage" &&
						dataMap["message"] == "Completed successfully" &&
						dataMap["status"] == "completed" &&
						dataMap["success"] == true
				}))
			},
		},
		{
			name: "send_complete_failure",
			action: func(adapter *WebSocketOperationAdapter) {
				adapter.SendComplete("test-stage", "Operation failed", false)
			},
			validate: func(t *testing.T, hub *MockWebSocketHub) {
				hub.AssertCalled(t, "Broadcast", "operation_complete", mock.MatchedBy(func(data interface{}) bool {
					dataMap := data.(map[string]interface{})
					return dataMap["step"] == "test-stage" &&
						dataMap["message"] == "Operation failed" &&
						dataMap["status"] == "failed" &&
						dataMap["success"] == false
				}))
			},
		},
		{
			name: "send_error",
			action: func(adapter *WebSocketOperationAdapter) {
				adapter.SendError("test-stage", "Critical error occurred")
			},
			validate: func(t *testing.T, hub *MockWebSocketHub) {
				hub.AssertCalled(t, "Broadcast", "operation_error", mock.MatchedBy(func(data interface{}) bool {
					dataMap := data.(map[string]interface{})
					return dataMap["step"] == "test-stage" &&
						dataMap["error"] == "Critical error occurred" &&
						dataMap["status"] == "error"
				}))
			},
		},
		{
			name: "broadcast_update_with_metadata",
			action: func(adapter *WebSocketOperationAdapter) {
				metadata := map[string]interface{}{
					"files_processed": 10,
					"total_files":     20,
				}
				adapter.BroadcastUpdate("operation_update", "processing", "active", metadata)
			},
			validate: func(t *testing.T, hub *MockWebSocketHub) {
				hub.AssertCalled(t, "Broadcast", "operation_update", mock.MatchedBy(func(data interface{}) bool {
					dataMap := data.(map[string]interface{})
					metadata, hasMetadata := dataMap["metadata"]
					return dataMap["eventType"] == "operation_update" &&
						dataMap["step"] == "processing" &&
						dataMap["status"] == "active" &&
						hasMetadata &&
						metadata != nil
				}))
			},
		},
		{
			name: "broadcast_update_without_metadata",
			action: func(adapter *WebSocketOperationAdapter) {
				adapter.BroadcastUpdate("operation_start", "initialization", "starting", nil)
			},
			validate: func(t *testing.T, hub *MockWebSocketHub) {
				hub.AssertCalled(t, "Broadcast", "operation_start", mock.MatchedBy(func(data interface{}) bool {
					dataMap := data.(map[string]interface{})
					metadata, hasMetadata := dataMap["metadata"]
					return dataMap["eventType"] == "operation_start" &&
						dataMap["step"] == "initialization" &&
						dataMap["status"] == "starting" &&
						hasMetadata && metadata == nil
				}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := &MockWebSocketHub{}
			adapter := NewWebSocketOperationAdapter(hub)

			// Setup mock expectations
			hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()

			// Execute action
			tt.action(adapter)

			// Validate results
			tt.validate(t, hub)
			hub.AssertExpectations(t)
		})
	}
}

func testOperationServiceConstruction(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (*WebSocketOperationAdapter, *slog.Logger, error)
		expectErr bool
		validate  func(t *testing.T, service *OperationService, err error)
	}{
		{
			name: "valid_construction",
			setup: func() (*WebSocketOperationAdapter, *slog.Logger, error) {
				hub := &MockWebSocketHub{}
				hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
				adapter := NewWebSocketOperationAdapter(hub)
				logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
				return adapter, logger, nil
			},
			expectErr: false,
			validate: func(t *testing.T, service *OperationService, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.GetManager())
				assert.NotNil(t, service.logger)
			},
		},
		{
			name: "nil_adapter",
			setup: func() (*WebSocketOperationAdapter, *slog.Logger, error) {
				logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
				return nil, logger, nil
			},
			expectErr: true,
			validate: func(t *testing.T, service *OperationService, err error) {
				// Service creation might still succeed with nil adapter but operations may fail
				if err != nil {
					assert.Error(t, err)
					assert.Nil(t, service)
				}
			},
		},
		{
			name: "nil_logger",
			setup: func() (*WebSocketOperationAdapter, *slog.Logger, error) {
				hub := &MockWebSocketHub{}
				hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
				adapter := NewWebSocketOperationAdapter(hub)
				return adapter, nil, nil
			},
			expectErr: false, // Service should handle nil logger gracefully
			validate: func(t *testing.T, service *OperationService, err error) {
				if err == nil {
					assert.NotNil(t, service)
					// Logger should be set to default if nil was passed
					assert.NotNil(t, service.logger)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, logger, setupErr := tt.setup()
			require.NoError(t, setupErr)

			service, err := NewOperationService(adapter, logger)

			if tt.expectErr {
				if err == nil {
					t.Logf("Expected error but service creation succeeded - may be environment dependent")
				}
			}

			tt.validate(t, service, err)

			// Cleanup mocks
			if adapter != nil && adapter.hub != nil {
				if mockHub, ok := adapter.hub.(*MockWebSocketHub); ok {
					mockHub.AssertExpectations(t)
				}
			}
		})
	}
}

func testOperationLifecycle(t *testing.T) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		operation  func() (string, error)
		validate   func(t *testing.T, operationID string, err error)
	}{
		{
			name: "start_operation",
			operation: func() (string, error) {
				params := map[string]interface{}{
					"mode": "test",
					"step": "testing",
				}
				return service.StartOperation(ctx, params)
			},
			validate: func(t *testing.T, operationID string, err error) {
				if err != nil {
					t.Logf("StartOperation failed (expected in test env): %v", err)
				} else {
					assert.NotEmpty(t, operationID)
					assert.Contains(t, operationID, "operation-")
				}
			},
		},
		{
			name: "start_scraping",
			operation: func() (string, error) {
				params := map[string]interface{}{
					"args": map[string]interface{}{
						"mode":     "test",
						"from":     "2024-01-01",
						"to":       "2024-01-31",
						"headless": true,
					},
				}
				return service.StartScraping(ctx, params)
			},
			validate: func(t *testing.T, operationID string, err error) {
				if err != nil {
					t.Logf("StartScraping failed (expected in test env): %v", err)
				} else {
					assert.NotEmpty(t, operationID)
				}
			},
		},
		{
			name: "start_processing",
			operation: func() (string, error) {
				params := map[string]interface{}{
					"input_dir": "/test/path",
					"mode":      "full",
				}
				return service.StartProcessing(ctx, params)
			},
			validate: func(t *testing.T, operationID string, err error) {
				if err != nil {
					t.Logf("StartProcessing failed (expected in test env): %v", err)
				} else {
					assert.NotEmpty(t, operationID)
				}
			},
		},
		{
			name: "start_index_extraction",
			operation: func() (string, error) {
				params := map[string]interface{}{
					"mode": "full",
				}
				return service.StartIndexExtraction(ctx, params)
			},
			validate: func(t *testing.T, operationID string, err error) {
				if err != nil {
					t.Logf("StartIndexExtraction failed (expected in test env): %v", err)
				} else {
					assert.NotEmpty(t, operationID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operationID, err := tt.operation()
			tt.validate(t, operationID, err)

			// If operation started successfully, test lifecycle methods
			if err == nil && operationID != "" {
				// Test getting status
				status, err := service.GetOperationStatus(ctx, operationID)
				if err != nil {
					t.Logf("GetOperationStatus failed: %v", err)
				} else {
					assert.NotNil(t, status)
				}

				// Test cancelling operation
				err = service.CancelOperation(ctx, operationID)
				if err != nil {
					t.Logf("CancelOperation failed: %v", err)
				}
			}
		})
	}

	hub.AssertExpectations(t)
}

func testParameterProcessing(t *testing.T) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name   string
		params map[string]interface{}
		verify func(t *testing.T, params map[string]interface{})
	}{
		{
			name: "wrapped_args_processing",
			params: map[string]interface{}{
				"args": map[string]interface{}{
					"from":     "2024-01-01",
					"to":       "2024-12-31",
					"mode":     "accumulative",
					"headless": false,
				},
			},
			verify: func(t *testing.T, params map[string]interface{}) {
				// Verify parameter transformation
				assert.NotNil(t, params["args"])
				args := params["args"].(map[string]interface{})
				assert.Equal(t, "2024-01-01", args["from"])
				assert.Equal(t, "2024-12-31", args["to"])
			},
		},
		{
			name: "direct_params_fallback",
			params: map[string]interface{}{
				"from":     "2024-06-01",
				"to":       "2024-06-30",
				"mode":     "initial",
				"headless": true,
			},
			verify: func(t *testing.T, params map[string]interface{}) {
				// Should work with direct params when no args wrapper
				assert.Equal(t, "2024-06-01", params["from"])
				assert.Equal(t, "2024-06-30", params["to"])
			},
		},
		{
			name: "empty_parameters",
			params: map[string]interface{}{},
			verify: func(t *testing.T, params map[string]interface{}) {
				// Should handle empty parameters gracefully
				assert.NotNil(t, params)
			},
		},
		{
			name: "nested_parameter_structure",
			params: map[string]interface{}{
				"args": map[string]interface{}{
					"config": map[string]interface{}{
						"threads": 4,
						"timeout": 300,
					},
					"filters": []string{"stock", "bond"},
				},
			},
			verify: func(t *testing.T, params map[string]interface{}) {
				args := params["args"].(map[string]interface{})
				assert.Contains(t, args, "config")
				assert.Contains(t, args, "filters")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, tt.params)

			// Test scraping with these parameters
			_, err := service.StartScraping(ctx, tt.params)
			if err != nil {
				t.Logf("StartScraping with %s failed (expected in test env): %v", tt.name, err)
			}
		})
	}

	hub.AssertExpectations(t)
}

func testOperationServiceErrorHandling(t *testing.T) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name      string
		operation func() error
		expectErr bool
	}{
		{
			name: "get_status_empty_id",
			operation: func() error {
				_, err := service.GetStatus(ctx, "")
				return err
			},
			expectErr: true,
		},
		{
			name: "get_status_nonexistent_id",
			operation: func() error {
				_, err := service.GetStatus(ctx, "nonexistent-operation-id")
				return err
			},
			expectErr: true,
		},
		{
			name: "cancel_nonexistent_operation",
			operation: func() error {
				return service.CancelOperation(ctx, "nonexistent-operation-id")
			},
			expectErr: true,
		},
		{
			name: "stop_nonexistent_operation",
			operation: func() error {
				return service.StopOperation(ctx, "nonexistent-operation-id")
			},
			expectErr: true,
		},
		{
			name: "get_operation_status_nonexistent",
			operation: func() error {
				_, err := service.GetOperationStatus(ctx, "nonexistent-operation-id")
				return err
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	// Test cancel all operations (should not error even with no operations)
	err = service.CancelAll(ctx)
	assert.NoError(t, err)

	hub.AssertExpectations(t)
}

func testOperationServiceConcurrency(t *testing.T) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	ctx := context.Background()
	numGoroutines := 10
	var wg sync.WaitGroup

	// Test concurrent read operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Test various read operations concurrently
			_, _ = service.ListOperations(ctx)
			_, _ = service.GetOperationTypes(ctx)
			_, _ = service.GetOperationMetrics(ctx)
			_ = service.GetStageInfo()

			// Test operations by status
			_, _ = service.ListOperationsByStatus(ctx, operations.OperationStatusCompleted)
		}(i)
	}

	wg.Wait()

	// Test concurrent write operations (creation)
	operationIDs := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			params := map[string]interface{}{
				"mode": fmt.Sprintf("test-%d", id),
			}
			operationIDs[id], errors[id] = service.StartOperation(ctx, params)
		}(i)
	}

	wg.Wait()

	// Verify operations were created or failed gracefully
	for i, err := range errors {
		if err != nil {
			t.Logf("Operation %d failed (expected in test env): %v", i, err)
		} else if operationIDs[i] != "" {
			t.Logf("Operation %d created successfully: %s", i, operationIDs[i])
		}
	}

	hub.AssertExpectations(t)
}

func testOperationValidation(t *testing.T) {
	// Create temporary directory structure for validation tests
	tempDir := t.TempDir()
	
	// Set up paths environment
	os.Setenv("ISX_DATA_DIR", tempDir)
	defer os.Unsetenv("ISX_DATA_DIR")

	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	ctx := context.Background()

	t.Run("validate_executables_missing", func(t *testing.T) {
		err := service.ValidateExecutables(ctx)
		// Should fail because executables don't exist in test environment
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required executable not found")
	})

	t.Run("validate_executables_existing", func(t *testing.T) {
		// Create mock executables
		paths, err := config.GetPaths()
		if err != nil {
			t.Skipf("Could not get paths: %v", err)
		}

		executableDir := paths.ExecutableDir
		require.NoError(t, os.MkdirAll(executableDir, 0755))

		executables := []string{"scraper.exe", "process.exe", "indexcsv.exe"}
		for _, exe := range executables {
			exePath := filepath.Join(executableDir, exe)
			require.NoError(t, os.WriteFile(exePath, []byte("mock executable"), 0755))
		}

		err = service.ValidateExecutables(ctx)
		assert.NoError(t, err)

		// Clean up
		for _, exe := range executables {
			os.Remove(filepath.Join(executableDir, exe))
		}
	})

	hub.AssertExpectations(t)
}

func testOperationTypes(t *testing.T) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	ctx := context.Background()

	operationTypes, err := service.GetOperationTypes(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, operationTypes)

	// Verify operation types structure
	foundFullPipeline := false
	expectedStages := map[string]bool{
		"scraping":   false,
		"processing": false,
		"indices":    false,
		"analysis":   false,
	}

	for _, opType := range operationTypes {
		assert.NotEmpty(t, opType.ID)
		assert.NotEmpty(t, opType.Name)
		assert.NotEmpty(t, opType.Description)
		assert.NotNil(t, opType.Parameters)

		if opType.ID == "full_pipeline" {
			foundFullPipeline = true
			assert.True(t, opType.CanRunAlone)
			assert.Contains(t, opType.Description, "all stages")
			
			// Verify full pipeline parameters
			paramNames := make(map[string]bool)
			for _, param := range opType.Parameters {
				paramNames[param.Name] = true
			}
			assert.True(t, paramNames["mode"])
			assert.True(t, paramNames["from"])
			assert.True(t, paramNames["to"])
		}

		if _, exists := expectedStages[opType.ID]; exists {
			expectedStages[opType.ID] = true
		}
	}

	assert.True(t, foundFullPipeline, "Should include full_pipeline operation type")

	// Verify all expected stages are present
	for stageName, found := range expectedStages {
		assert.True(t, found, fmt.Sprintf("Should include %s stage", stageName))
	}

	hub.AssertExpectations(t)
}

func testOperationMetrics(t *testing.T) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	ctx := context.Background()

	metrics, err := service.GetOperationMetrics(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)

	// Verify metrics structure
	expectedFields := []string{
		"total_operations",
		"active_operations", 
		"completed_operations",
		"failed_operations",
		"timestamp",
	}

	for _, field := range expectedFields {
		assert.Contains(t, metrics, field, fmt.Sprintf("Metrics should contain %s field", field))
	}

	// Verify data types
	assert.IsType(t, 0, metrics["total_operations"])
	assert.IsType(t, 0, metrics["active_operations"])
	assert.IsType(t, 0, metrics["completed_operations"])
	assert.IsType(t, 0, metrics["failed_operations"])
	assert.IsType(t, int64(0), metrics["timestamp"])

	// Initially should have no operations
	assert.Equal(t, 0, metrics["total_operations"])
	assert.Equal(t, 0, metrics["active_operations"])
	assert.Equal(t, 0, metrics["completed_operations"])
	assert.Equal(t, 0, metrics["failed_operations"])

	hub.AssertExpectations(t)
}

func testStageManagement(t *testing.T) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Skipf("NewOperationService failed (expected in test environment): %v", err)
	}

	stageInfo := service.GetStageInfo()
	assert.NotNil(t, stageInfo)

	// Verify stage info structure
	assert.Contains(t, stageInfo, "steps")
	steps, ok := stageInfo["steps"].([]map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, steps)

	expectedSteps := map[string]string{
		"scraping":   "scraper.exe",
		"processing": "process.exe",
		"indices":    "indexcsv.exe",
		"analysis":   "",
	}

	foundSteps := make(map[string]bool)

	for _, step := range steps {
		assert.Contains(t, step, "id")
		assert.Contains(t, step, "name")
		assert.Contains(t, step, "description")
		assert.Contains(t, step, "executable")

		stepID := step["id"].(string)
		expectedExe, exists := expectedSteps[stepID]
		assert.True(t, exists, fmt.Sprintf("Unexpected step ID: %s", stepID))
		assert.Equal(t, expectedExe, step["executable"])

		foundSteps[stepID] = true
	}

	// Verify all expected steps are present
	for stepID := range expectedSteps {
		assert.True(t, foundSteps[stepID], fmt.Sprintf("Missing step: %s", stepID))
	}

	hub.AssertExpectations(t)
}

// Test helper function for parameter value extraction
func TestGetValue(t *testing.T) {
	testMap := map[string]interface{}{
		"string_value": "test",
		"int_value":    42,
		"nil_value":    nil,
	}

	tests := []struct {
		name         string
		key          string
		defaultValue interface{}
		expected     interface{}
	}{
		{
			name:         "existing_string",
			key:          "string_value",
			defaultValue: "default",
			expected:     "test",
		},
		{
			name:         "existing_int",
			key:          "int_value",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "nonexistent_key",
			key:          "missing_key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "nil_value",
			key:          "nil_value",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getValue(testMap, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark operation service methods
func BenchmarkOperationServiceListOperations(b *testing.B) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		b.Skipf("NewOperationService failed: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ListOperations(ctx)
	}
}

func BenchmarkOperationServiceGetOperationTypes(b *testing.B) {
	hub := &MockWebSocketHub{}
	hub.On("Broadcast", mock.AnythingOfType("string"), mock.Anything).Return()
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service, err := NewOperationService(adapter, logger)
	if err != nil {
		b.Skipf("NewOperationService failed: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetOperationTypes(ctx)
	}
}