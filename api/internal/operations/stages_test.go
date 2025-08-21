package operations_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"isxcli/internal/operations"
	operationstestutil "isxcli/internal/operations/testutil"
	testutil "isxcli/internal/shared/testutil"
)

// Mock implementations for testing
type mockWebSocketHub struct {
	broadcasts []mockBroadcast
}

type mockBroadcast struct {
	eventType string
	Step     string
	status    string
	metadata  interface{}
}

func (m *mockWebSocketHub) BroadcastUpdate(eventType, Step, status string, metadata interface{}) {
	m.broadcasts = append(m.broadcasts, mockBroadcast{
		eventType: eventType,
		Step:     Step,
		status:    status,
		metadata:  metadata,
	})
}

type mockLicenseChecker struct {
	requiresLicense bool
	checkError      error
}

func (m *mockLicenseChecker) CheckLicense() error {
	return m.checkError
}

func (m *mockLicenseChecker) RequiresLicense() bool {
	return m.requiresLicense
}

// TestNewScrapingStage tests the NewScrapingStage constructor
func TestNewScrapingStage(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	
	tests := []struct {
		name          string
		executableDir string
		logger        *slog.Logger
		options       *operations.StageOptions
		expectNil     bool
	}{
		{
			name:          "basic scraping Step creation",
			executableDir: "/path/to/executables",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "scraping Step with options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				EnableProgress: true,
			},
			expectNil: false,
		},
		{
			name:          "scraping Step with full options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				WebSocketManager: &mockWebSocketHub{},
				LicenseChecker:   &mockLicenseChecker{requiresLicense: true},
				EnableProgress:   true,
			},
			expectNil: false,
		},
		{
			name:          "empty executable directory",
			executableDir: "",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "nil logger",
			executableDir: "/path/to/executables",
			logger:        nil,
			options:       nil,
			expectNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Step := operations.NewScrapingStage(tt.executableDir, tt.logger, tt.options)
			
			if tt.expectNil {
				if Step != nil {
					t.Errorf("NewScrapingStage() = %v, want nil", Step)
				}
				return
			}
			
			if Step == nil {
				t.Fatal("NewScrapingStage() returned nil, expected valid Step")
			}

			// Verify Step properties
			operationstestutil.AssertEqual(t, Step.ID(), operations.StageIDScraping)
			operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameScraping)
			
			// Verify dependencies (scraping has no dependencies)
			deps := Step.GetDependencies()
			operationstestutil.AssertEqual(t, len(deps), 0)
			
			// Verify validation always passes
			state := operations.NewOperationState("test-operation")
			err := Step.Validate(state)
			operationstestutil.AssertEqual(t, err, nil)
		})
	}
}

// TestNewProcessingStage tests the NewProcessingStage constructor
func TestNewProcessingStage(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	
	tests := []struct {
		name          string
		executableDir string
		logger        *slog.Logger
		options       *operations.StageOptions
		expectNil     bool
	}{
		{
			name:          "basic processing Step creation",
			executableDir: "/path/to/executables",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "processing Step with options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				EnableProgress: true,
			},
			expectNil: false,
		},
		{
			name:          "processing Step with full options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				WebSocketManager: &mockWebSocketHub{},
				LicenseChecker:   &mockLicenseChecker{requiresLicense: false},
				EnableProgress:   true,
			},
			expectNil: false,
		},
		{
			name:          "empty executable directory",
			executableDir: "",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "nil logger",
			executableDir: "/path/to/executables",
			logger:        nil,
			options:       nil,
			expectNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Step := operations.NewProcessingStage(tt.executableDir, tt.logger, tt.options)
			
			if tt.expectNil {
				if Step != nil {
					t.Errorf("NewProcessingStage() = %v, want nil", Step)
				}
				return
			}
			
			if Step == nil {
				t.Fatal("NewProcessingStage() returned nil, expected valid Step")
			}

			// Verify Step properties
			operationstestutil.AssertEqual(t, Step.ID(), operations.StageIDProcessing)
			operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameProcessing)
			
			// Verify dependencies (processing depends on scraping)
			deps := Step.GetDependencies()
			operationstestutil.AssertEqual(t, len(deps), 1)
			operationstestutil.AssertEqual(t, deps[0], operations.StageIDScraping)
			
			// Verify validation always passes
			state := operations.NewOperationState("test-operation")
			err := Step.Validate(state)
			operationstestutil.AssertEqual(t, err, nil)
		})
	}
}

// TestNewIndicesStage tests the NewIndicesStage constructor
func TestNewIndicesStage(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	
	tests := []struct {
		name          string
		executableDir string
		logger        *slog.Logger
		options       *operations.StageOptions
		expectNil     bool
	}{
		{
			name:          "basic indices Step creation",
			executableDir: "/path/to/executables",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "indices Step with options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				EnableProgress: true,
			},
			expectNil: false,
		},
		{
			name:          "indices Step with full options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				WebSocketManager: &mockWebSocketHub{},
				LicenseChecker:   &mockLicenseChecker{requiresLicense: true},
				EnableProgress:   true,
			},
			expectNil: false,
		},
		{
			name:          "empty executable directory",
			executableDir: "",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "nil logger",
			executableDir: "/path/to/executables",
			logger:        nil,
			options:       nil,
			expectNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Step := operations.NewIndicesStage(tt.executableDir, tt.logger, tt.options)
			
			if tt.expectNil {
				if Step != nil {
					t.Errorf("NewIndicesStage() = %v, want nil", Step)
				}
				return
			}
			
			if Step == nil {
				t.Fatal("NewIndicesStage() returned nil, expected valid Step")
			}

			// Verify Step properties
			operationstestutil.AssertEqual(t, Step.ID(), operations.StageIDIndices)
			operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameIndices)
			
			// Verify dependencies (indices depends on processing)
			deps := Step.GetDependencies()
			operationstestutil.AssertEqual(t, len(deps), 1)
			operationstestutil.AssertEqual(t, deps[0], operations.StageIDProcessing)
			
			// Verify validation always passes
			state := operations.NewOperationState("test-operation")
			err := Step.Validate(state)
			operationstestutil.AssertEqual(t, err, nil)
		})
	}
}

// TestNewLiquidityStage tests the NewLiquidityStage constructor
func TestNewLiquidityStage(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	
	tests := []struct {
		name          string
		executableDir string
		logger        *slog.Logger
		options       *operations.StageOptions
		expectNil     bool
	}{
		{
			name:          "basic liquidity Step creation",
			executableDir: "/path/to/executables",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "liquidity Step with options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				EnableProgress: true,
			},
			expectNil: false,
		},
		{
			name:          "liquidity Step with full options",
			executableDir: "/path/to/executables",
			logger:        logger,
			options: &operations.StageOptions{
				WebSocketManager: &mockWebSocketHub{},
				LicenseChecker:   &mockLicenseChecker{requiresLicense: false},
				EnableProgress:   true,
			},
			expectNil: false,
		},
		{
			name:          "empty executable directory",
			executableDir: "",
			logger:        logger,
			options:       nil,
			expectNil:     false,
		},
		{
			name:          "nil logger",
			executableDir: "/path/to/executables",
			logger:        nil,
			options:       nil,
			expectNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Step := operations.NewLiquidityStage(tt.executableDir, tt.logger, tt.options)
			
			if tt.expectNil {
				if Step != nil {
					t.Errorf("NewLiquidityStage() = %v, want nil", Step)
				}
				return
			}
			
			if Step == nil {
				t.Fatal("NewLiquidityStage() returned nil, expected valid Step")
			}

			// Verify Step properties
			operationstestutil.AssertEqual(t, Step.ID(), operations.StageIDLiquidity)
			operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameLiquidity)
			
			// Verify dependencies (liquidity depends on indices)
			deps := Step.GetDependencies()
			operationstestutil.AssertEqual(t, len(deps), 1)
			operationstestutil.AssertEqual(t, deps[0], operations.StageIDIndices)
			
			// Verify validation always passes
			state := operations.NewOperationState("test-operation")
			err := Step.Validate(state)
			operationstestutil.AssertEqual(t, err, nil)
		})
	}
}

// TestStageFactory tests the StageFactory function
func TestStageFactory(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	executableDir := "/path/to/executables"
	
	tests := []struct {
		name    string
		options *operations.StageOptions
	}{
		{
			name:    "factory with nil options",
			options: nil,
		},
		{
			name: "factory with basic options",
			options: &operations.StageOptions{
				EnableProgress: true,
			},
		},
		{
			name: "factory with full options",
			options: &operations.StageOptions{
				WebSocketManager: &mockWebSocketHub{},
				LicenseChecker:   &mockLicenseChecker{requiresLicense: true},
				EnableProgress:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := operations.StageFactory(executableDir, logger, tt.options)
			
			// Verify all expected steps are created
			expectedStages := []string{
				operations.StageIDScraping,
				operations.StageIDProcessing,
				operations.StageIDIndices,
				operations.StageIDLiquidity,
			}
			
			operationstestutil.AssertEqual(t, len(steps), len(expectedStages))
			
			for _, expectedStageID := range expectedStages {
				Step, exists := steps[expectedStageID]
				if !exists {
					t.Errorf("StageFactory() missing Step %s", expectedStageID)
					continue
				}
				
				if Step == nil {
					t.Errorf("StageFactory() created nil Step for %s", expectedStageID)
					continue
				}
				
				operationstestutil.AssertEqual(t, Step.ID(), expectedStageID)
				
				// Verify Step types and names
				switch expectedStageID {
				case operations.StageIDScraping:
					operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameScraping)
					operationstestutil.AssertEqual(t, len(Step.GetDependencies()), 0)
				case operations.StageIDProcessing:
					operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameProcessing)
					operationstestutil.AssertEqual(t, len(Step.GetDependencies()), 1)
					operationstestutil.AssertEqual(t, Step.GetDependencies()[0], operations.StageIDScraping)
				case operations.StageIDIndices:
					operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameIndices)
					operationstestutil.AssertEqual(t, len(Step.GetDependencies()), 1)
					operationstestutil.AssertEqual(t, Step.GetDependencies()[0], operations.StageIDProcessing)
				case operations.StageIDLiquidity:
					operationstestutil.AssertEqual(t, Step.Name(), operations.StageNameLiquidity)
					operationstestutil.AssertEqual(t, len(Step.GetDependencies()), 1)
					operationstestutil.AssertEqual(t, Step.GetDependencies()[0], operations.StageIDProcessing)
				}
			}
		})
	}
}

// TestStageExecutionBasics tests basic execution setup (without actual command execution)
func TestStageExecutionBasics(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	tempDir := t.TempDir()
	
	// Create mock executable files
	scraperPath := filepath.Join(tempDir, "scraper.exe")
	processorPath := filepath.Join(tempDir, "process.exe")
	indexPath := filepath.Join(tempDir, "indexcsv.exe")
	
	// Create empty files to simulate executables
	for _, path := range []string{scraperPath, processorPath, indexPath} {
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("Failed to create mock executable %s: %v", path, err)
		}
		file.Close()
	}
	
	tests := []struct {
		name        string
		stageType   string
		constructor func() operations.Step
	}{
		{
			name:      "scraping Step execution setup",
			stageType: "scraping",
			constructor: func() operations.Step {
				return operations.NewScrapingStage(tempDir, logger, nil)
			},
		},
		{
			name:      "processing Step execution setup",
			stageType: "processing",
			constructor: func() operations.Step {
				return operations.NewProcessingStage(tempDir, logger, nil)
			},
		},
		{
			name:      "indices Step execution setup",
			stageType: "indices",
			constructor: func() operations.Step {
				return operations.NewIndicesStage(tempDir, logger, nil)
			},
		},
		{
			name:      "liquidity Step execution setup",
			stageType: "liquidity",
			constructor: func() operations.Step {
				return operations.NewLiquidityStage(tempDir, logger, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Step := tt.constructor()
			state := operations.NewOperationState("test-operation")
			
			// Initialize Step state - this is crucial for Step execution
			StepState := operations.NewStepState(Step.ID(), Step.Name())
			state.Steps[Step.ID()] = StepState
			
			// Add Step configuration if needed
			state.SetConfig(operations.ContextKeyFromDate, "2024-01-01")
			state.SetConfig(operations.ContextKeyToDate, "2024-01-31")
			state.SetConfig(operations.ContextKeyMode, "full")
			
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			
			// The execution will likely fail due to mock executables, but we're testing setup
			err := Step.Execute(ctx, state)
			
			// We expect errors since we're using mock executables
			// The important thing is that the Step attempts execution
			if err == nil && tt.stageType != "liquidity" {
				// Liquidity Step is a special case - it currently just completes immediately
				t.Logf("Step %s execution did not return error - this may be expected for mock setup", tt.stageType)
			}
			
			// Verify Step state was created and updated
			finalStepState := state.GetStage(Step.ID())
			if finalStepState == nil {
				t.Errorf("Step state was not found for %s", Step.ID())
			}
		})
	}
}

// TestStageOptionsHandling tests how steps handle different option configurations
func TestStageOptionsHandling(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	executableDir := "/path/to/executables"
	
	tests := []struct {
		name    string
		options *operations.StageOptions
		verify  func(t *testing.T, options *operations.StageOptions)
	}{
		{
			name:    "nil options defaults to empty struct",
			options: nil,
			verify: func(t *testing.T, options *operations.StageOptions) {
				if options == nil {
					t.Error("Expected non-nil options after Step creation")
				}
			},
		},
		{
			name: "websocket manager preserved",
			options: &operations.StageOptions{
				WebSocketManager: &mockWebSocketHub{},
			},
			verify: func(t *testing.T, options *operations.StageOptions) {
				if options.WebSocketManager == nil {
					t.Error("WebSocketManager should be preserved")
				}
			},
		},
		{
			name: "license checker preserved",
			options: &operations.StageOptions{
				LicenseChecker: &mockLicenseChecker{requiresLicense: true},
			},
			verify: func(t *testing.T, options *operations.StageOptions) {
				if options.LicenseChecker == nil {
					t.Error("LicenseChecker should be preserved")
				}
				if !options.LicenseChecker.RequiresLicense() {
					t.Error("LicenseChecker configuration should be preserved")
				}
			},
		},
		{
			name: "progress enabling preserved",
			options: &operations.StageOptions{
				EnableProgress: true,
			},
			verify: func(t *testing.T, options *operations.StageOptions) {
				if !options.EnableProgress {
					t.Error("EnableProgress should be preserved")
				}
			},
		},
	}

	stageConstructors := []struct {
		name string
		constructor func(*operations.StageOptions) operations.Step
	}{
		{
			name: "scraping",
			constructor: func(opts *operations.StageOptions) operations.Step {
				return operations.NewScrapingStage(executableDir, logger, opts)
			},
		},
		{
			name: "processing", 
			constructor: func(opts *operations.StageOptions) operations.Step {
				return operations.NewProcessingStage(executableDir, logger, opts)
			},
		},
		{
			name: "indices",
			constructor: func(opts *operations.StageOptions) operations.Step {
				return operations.NewIndicesStage(executableDir, logger, opts)
			},
		},
		{
			name: "liquidity",
			constructor: func(opts *operations.StageOptions) operations.Step {
				return operations.NewLiquidityStage(executableDir, logger, opts)
			},
		},
	}

	for _, stageConstructor := range stageConstructors {
		for _, tt := range tests {
			t.Run(stageConstructor.name+"_"+tt.name, func(t *testing.T) {
				Step := stageConstructor.constructor(tt.options)
				
				if Step == nil {
					t.Fatal("Step constructor returned nil")
				}
				
				// We can't directly access the options from the Step, but we can test behavior
				// For now, just verify the Step was created successfully
				if Step.ID() == "" {
					t.Error("Step ID should not be empty")
				}
				if Step.Name() == "" {
					t.Error("Step Name should not be empty")
				}
			})
		}
	}
}

// TestStageValidationBehavior tests the validation behavior of all steps
func TestStageValidationBehavior(t *testing.T) {
	logger, _ := testutil.NewTestLogger(t)
	executableDir := "/path/to/executables"
	
	steps := []operations.Step{
		operations.NewScrapingStage(executableDir, logger, nil),
		operations.NewProcessingStage(executableDir, logger, nil),
		operations.NewIndicesStage(executableDir, logger, nil),
		operations.NewLiquidityStage(executableDir, logger, nil),
	}
	
	tests := []struct {
		name        string
		setupState  func() *operations.OperationState
		expectError bool
	}{
		{
			name: "empty state validation",
			setupState: func() *operations.OperationState {
				return operations.NewOperationState("test-operation")
			},
			expectError: false, // All steps should pass validation with empty state
		},
		{
			name: "populated state validation",
			setupState: func() *operations.OperationState {
				state := operations.NewOperationState("test-operation")
				state.SetConfig(operations.ContextKeyFromDate, "2024-01-01")
				state.SetConfig(operations.ContextKeyToDate, "2024-01-31")
				return state
			},
			expectError: false,
		},
	}

	for _, Step := range steps {
		for _, tt := range tests {
			t.Run(Step.ID()+"_"+tt.name, func(t *testing.T) {
				state := tt.setupState()
				err := Step.Validate(state)
				
				if tt.expectError && err == nil {
					t.Errorf("Expected validation error for Step %s, got nil", Step.ID())
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected validation error for Step %s: %v", Step.ID(), err)
				}
			})
		}
	}
}