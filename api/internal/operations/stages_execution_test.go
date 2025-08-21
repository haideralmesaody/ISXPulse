package operations_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"isxcli/internal/operations"
	operationstestutil "isxcli/internal/operations/testutil"
	testutil "isxcli/internal/shared/testutil"
)

// MockCommandExecutor simulates external command execution
type MockCommandExecutor struct {
	Commands []MockCommand
	Delay    time.Duration
	ShouldFail bool
	FailError  error
}

type MockCommand struct {
	Name string
	Args []string
	Dir  string
	Output string
	Error string
	ExitCode int
}

// Enhanced WebSocket mock with detailed tracking
type mockExecutionWebSocketHub struct {
	broadcasts []mockExecutionBroadcast
}

type mockExecutionBroadcast struct {
	eventType string
	Step     string
	status    string
	metadata  interface{}
	timestamp time.Time
}

func (m *mockExecutionWebSocketHub) BroadcastUpdate(eventType, Step, status string, metadata interface{}) {
	m.broadcasts = append(m.broadcasts, mockExecutionBroadcast{
		eventType: eventType,
		Step:     Step,
		status:    status,
		metadata:  metadata,
		timestamp: time.Now(),
	})
}

func (m *mockExecutionWebSocketHub) GetBroadcastsForStage(stageID string) []mockExecutionBroadcast {
	var result []mockExecutionBroadcast
	for _, broadcast := range m.broadcasts {
		if broadcast.Step == stageID {
			result = append(result, broadcast)
		}
	}
	return result
}

// Enhanced license checker mock
type mockExecutionLicenseChecker struct {
	requiresLicense bool
	checkError      error
	checkCalls      int
}

func (m *mockExecutionLicenseChecker) CheckLicense() error {
	m.checkCalls++
	return m.checkError
}

func (m *mockExecutionLicenseChecker) RequiresLicense() bool {
	return m.requiresLicense
}


// Helper function to create initialized operation state
func createInitializedOperationState(stageID, stageName string) *operations.OperationState {
	state := operations.NewOperationState("test-operation")
	StepState := operations.NewStepState(stageID, stageName)
	state.Steps[stageID] = StepState
	
	// Add common configuration
	state.SetConfig(operations.ContextKeyFromDate, "2024-01-01")
	state.SetConfig(operations.ContextKeyToDate, "2024-01-31")
	state.SetConfig(operations.ContextKeyMode, "full")
	
	return state
}

// TestScrapingStageExecution tests the scraping Step execution
func TestScrapingStageExecution(t *testing.T) {
	tests := []struct {
		name           string
		setupExecutable bool
		options        *operations.StageOptions
		expectError    bool
		expectedCalls  int
		timeout        time.Duration
	}{
		{
			name: "scraping with license check failure",
			setupExecutable: true,
			options: &operations.StageOptions{
				LicenseChecker: &mockExecutionLicenseChecker{
					requiresLicense: true,
					checkError:     fmt.Errorf("license expired"),
				},
			},
			expectError:   true,
			expectedCalls: 1,
			timeout:       5 * time.Second,
		},
		{
			name:          "scraping executable not found",
			setupExecutable: false, // No scraper.exe created
			options:       nil,
			expectError:   true,
			expectedCalls: 0,
			timeout:       5 * time.Second,
		},
		{
			name: "scraping license check called when required",
			setupExecutable: true,
			options: &operations.StageOptions{
				LicenseChecker: &mockExecutionLicenseChecker{requiresLicense: true},
			},
			expectError:   true, // Will fail because we can't execute the mock exe, but license should be checked first
			expectedCalls: 1,
			timeout:       5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			tempDir := t.TempDir()
			
			// Create executable if needed
			if tt.setupExecutable {
				exePath := filepath.Join(tempDir, "scraper.exe")
				// Create a simple text file that will fail to execute (testing error paths)
				err := os.WriteFile(exePath, []byte("mock executable"), 0755)
				if err != nil {
					t.Fatalf("Failed to create mock executable: %v", err)
				}
			}
			
			Step := operations.NewScrapingStage(tempDir, logger, tt.options)
			state := createInitializedOperationState(operations.StageIDScraping, operations.StageNameScraping)
			
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			
			err := Step.Execute(ctx, state)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Verify license checker calls
			if tt.options != nil && tt.options.LicenseChecker != nil {
				if mockChecker, ok := tt.options.LicenseChecker.(*mockExecutionLicenseChecker); ok {
					operationstestutil.AssertEqual(t, mockChecker.checkCalls, tt.expectedCalls)
				}
			}
			
			// Verify Step state updates
			StepState := state.GetStage(operations.StageIDScraping)
			if StepState == nil {
				t.Error("Step state should exist after execution")
			}
		})
	}
}

// TestProcessingStageExecution tests the processing Step execution
func TestProcessingStageExecution(t *testing.T) {
	tests := []struct {
		name           string
		setupExecutable bool
		expectError    bool
	}{
		{
			name:           "processing executable not found",
			setupExecutable: false,
			expectError:    true,
		},
		{
			name:           "processing with executable present", 
			setupExecutable: true,
			expectError:    true, // Will fail because we can't actually execute
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			tempDir := t.TempDir()
			
			// Create executable if needed
			if tt.setupExecutable {
				exePath := filepath.Join(tempDir, "process.exe")
				err := os.WriteFile(exePath, []byte("mock executable"), 0755)
				if err != nil {
					t.Fatalf("Failed to create mock executable: %v", err)
				}
			}
			
			Step := operations.NewProcessingStage(tempDir, logger, nil)
			state := createInitializedOperationState(operations.StageIDProcessing, operations.StageNameProcessing)
			
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			err := Step.Execute(ctx, state)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestIndicesStageExecution tests the indices Step execution
func TestIndicesStageExecution(t *testing.T) {
	tests := []struct {
		name           string
		setupExecutable bool
		expectError    bool
	}{
		{
			name:           "indices executable not found",
			setupExecutable: false,
			expectError:    true,
		},
		{
			name:           "indices with executable present",
			setupExecutable: true,
			expectError:    true, // Will fail because we can't actually execute
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := testutil.NewTestLogger(t)
			tempDir := t.TempDir()
			
			// Create data directory structure
			dataDir := filepath.Join(tempDir, "data", "reports")
			err := os.MkdirAll(dataDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create data directory: %v", err)
			}
			
			// Create executable if needed
			if tt.setupExecutable {
				exePath := filepath.Join(tempDir, "indexcsv.exe")
				err := os.WriteFile(exePath, []byte("mock executable"), 0755)
				if err != nil {
					t.Fatalf("Failed to create mock executable: %v", err)
				}
			}
			
			Step := operations.NewIndicesStage(tempDir, logger, nil)
			state := createInitializedOperationState(operations.StageIDIndices, operations.StageNameIndices)
			
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			err = Step.Execute(ctx, state)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestAnalysisStageExecution tests the analysis Step execution
func TestAnalysisStageExecution(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	tempDir := t.TempDir()
	
	Step := operations.NewLiquidityStage(tempDir, logger, nil)
	state := createInitializedOperationState(operations.StageIDLiquidity, operations.StageNameLiquidity)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err := Step.Execute(ctx, state)
	
	// Analysis Step should complete successfully (it's currently a no-op)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Verify Step state exists
	StepState := state.GetStage(operations.StageIDLiquidity)
	if StepState == nil {
		t.Error("Step state should exist after execution")
	}
}

// TestStageWebSocketIntegration tests WebSocket integration
func TestStageWebSocketIntegration(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	mockWS := &mockExecutionWebSocketHub{}
	
	options := &operations.StageOptions{
		EnableProgress:   true,
		WebSocketManager: mockWS,
	}
	
	// Test that steps call WebSocket when progress tracking is enabled
	tempDir := t.TempDir()
	
	// Create a mock executable that will fail to execute
	exePath := filepath.Join(tempDir, "scraper.exe")
	err := os.WriteFile(exePath, []byte("mock"), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock executable: %v", err)
	}
	
	Step := operations.NewScrapingStage(tempDir, logger, options)
	state := createInitializedOperationState(operations.StageIDScraping, operations.StageNameScraping)
	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Execute (will fail, but should make initial progress calls)
	Step.Execute(ctx, state)
	
	// Verify WebSocket broadcasts were made during execution setup
	broadcasts := mockWS.GetBroadcastsForStage(operations.StageIDScraping)
	if len(broadcasts) == 0 {
		t.Error("Expected WebSocket broadcasts for progress tracking, even for failed execution")
	}
}

// TestStageOptionsValidation tests Step options handling
func TestStageOptionsValidation(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	tempDir := t.TempDir()
	
	tests := []struct {
		name    string
		options *operations.StageOptions
	}{
		{
			name:    "nil options",
			options: nil,
		},
		{
			name: "progress tracking enabled",
			options: &operations.StageOptions{
				EnableProgress: true,
			},
		},
		{
			name: "websocket manager provided",
			options: &operations.StageOptions{
				WebSocketManager: &mockExecutionWebSocketHub{},
			},
		},
		{
			name: "license checker provided",
			options: &operations.StageOptions{
				LicenseChecker: &mockExecutionLicenseChecker{requiresLicense: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that all Step types accept the options without error
			steps := []operations.Step{
				operations.NewScrapingStage(tempDir, logger, tt.options),
				operations.NewProcessingStage(tempDir, logger, tt.options),
				operations.NewIndicesStage(tempDir, logger, tt.options),
				operations.NewLiquidityStage(tempDir, logger, tt.options),
			}
			
			for _, Step := range steps {
				if Step == nil {
					t.Errorf("Step should not be nil with options: %+v", tt.options)
				}
				
				// Verify Step basic properties
				if Step.ID() == "" {
					t.Error("Step ID should not be empty")
				}
				if Step.Name() == "" {
					t.Error("Step Name should not be empty")
				}
			}
		})
	}
}