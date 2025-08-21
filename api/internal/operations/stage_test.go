package operations_test

import (
	"testing"
	"time"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

func TestNewStepState(t *testing.T) {
	id := "test-Step"
	name := "Test Step"
	
	state := operations.NewStepState(id, name)
	
	// Verify initial values
	testutil.AssertEqual(t, state.ID, id)
	testutil.AssertEqual(t, state.Name, name)
	testutil.AssertStepStatus(t, state, operations.StepStatusPending)
	testutil.AssertProgress(t, state, 0)
	testutil.AssertNotNil(t, state.Metadata)
	
	if state.StartTime != nil {
		t.Error("StartTime should be nil initially")
	}
	if state.EndTime != nil {
		t.Error("EndTime should be nil initially")
	}
	if state.Error != nil {
		t.Error("Error should be nil initially")
	}
}

func TestStepStateTransitions(t *testing.T) {
	tests := []struct {
		name        string
		transition  func(*operations.StepState)
		wantStatus  operations.StepStatus
		wantProgress float64
		checkTime   func(*operations.StepState) bool
	}{
		{
			name: "Start",
			transition: func(s *operations.StepState) {
				s.Start()
			},
			wantStatus:   operations.StepStatusActive,
			wantProgress: 0,
			checkTime: func(s *operations.StepState) bool {
				return s.StartTime != nil && s.EndTime == nil
			},
		},
		{
			name: "Complete",
			transition: func(s *operations.StepState) {
				s.Complete()
			},
			wantStatus:   operations.StepStatusCompleted,
			wantProgress: 100,
			checkTime: func(s *operations.StepState) bool {
				return s.EndTime != nil
			},
		},
		{
			name: "Fail",
			transition: func(s *operations.StepState) {
				s.Fail(operations.NewExecutionError("test", nil, false))
			},
			wantStatus: operations.StepStatusFailed,
			checkTime: func(s *operations.StepState) bool {
				return s.EndTime != nil && s.Error != nil
			},
		},
		{
			name: "Skip",
			transition: func(s *operations.StepState) {
				s.Skip("Dependencies not met")
			},
			wantStatus: operations.StepStatusSkipped,
			checkTime: func(s *operations.StepState) bool {
				return s.EndTime != nil && s.Message == "Dependencies not met"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := operations.NewStepState("test", "Test")
			
			tt.transition(state)
			
			testutil.AssertStepStatus(t, state, tt.wantStatus)
			if tt.wantProgress > 0 {
				testutil.AssertProgress(t, state, tt.wantProgress)
			}
			if !tt.checkTime(state) {
				t.Error("Time fields not set correctly")
			}
		})
	}
}

func TestStepStateUpdateProgress(t *testing.T) {
	state := operations.NewStepState("test", "Test")
	
	// Update progress multiple times
	updates := []struct {
		progress float64
		message  string
	}{
		{25, "Starting"},
		{50, "Halfway"},
		{75, "Almost done"},
		{100, "Completed"},
	}
	
	for _, update := range updates {
		state.UpdateProgress(update.progress, update.message)
		testutil.AssertProgress(t, state, update.progress)
		testutil.AssertEqual(t, state.Message, update.message)
	}
}

func TestStepStateDuration(t *testing.T) {
	state := operations.NewStepState("test", "Test")
	
	// Duration should be 0 before start
	if state.Duration() != 0 {
		t.Error("Duration should be 0 before start")
	}
	
	// Start the Step
	state.Start()
	time.Sleep(50 * time.Millisecond)
	
	// Duration should be > 0 while running
	duration := state.Duration()
	if duration <= 0 {
		t.Error("Duration should be > 0 while running")
	}
	
	// Complete the Step
	state.Complete()
	finalDuration := state.Duration()
	
	// Duration should be fixed after completion
	time.Sleep(10 * time.Millisecond)
	if state.Duration() != finalDuration {
		t.Error("Duration should not change after completion")
	}
	
	// Verify duration is reasonable
	testutil.AssertDuration(t, finalDuration, 50*time.Millisecond, 20*time.Millisecond)
}

func TestBaseStage(t *testing.T) {
	id := "base-Step"
	name := "Base Step"
	deps := []string{"dep1", "dep2"}
	
	base := operations.NewBaseStage(id, name, deps)
	
	// Test getters
	testutil.AssertEqual(t, base.ID(), id)
	testutil.AssertEqual(t, base.Name(), name)
	
	// Test dependencies
	gotDeps := base.GetDependencies()
	if len(gotDeps) != len(deps) {
		t.Errorf("Dependencies count = %d, want %d", len(gotDeps), len(deps))
	}
	for i, dep := range gotDeps {
		if dep != deps[i] {
			t.Errorf("Dependency[%d] = %s, want %s", i, dep, deps[i])
		}
	}
	
	// Test default validation (should always pass)
	state := operations.NewOperationState("test")
	testutil.AssertNoError(t, base.Validate(state))
}

func TestBaseStageNilDependencies(t *testing.T) {
	// Test that nil dependencies are handled properly
	base := operations.NewBaseStage("test", "Test", nil)
	
	deps := base.GetDependencies()
	if deps == nil {
		t.Error("GetDependencies should return empty slice, not nil")
	}
	if len(deps) != 0 {
		t.Errorf("Dependencies count = %d, want 0", len(deps))
	}
}

func TestStepStateMetadata(t *testing.T) {
	state := operations.NewStepState("test", "Test")
	
	// Add metadata
	state.Metadata["key1"] = "value1"
	state.Metadata["key2"] = 42
	state.Metadata["key3"] = true
	
	// Verify metadata
	if state.Metadata["key1"] != "value1" {
		t.Error("Metadata key1 not set correctly")
	}
	if state.Metadata["key2"] != 42 {
		t.Error("Metadata key2 not set correctly")
	}
	if state.Metadata["key3"] != true {
		t.Error("Metadata key3 not set correctly")
	}
}

func TestStepStateErrorHandling(t *testing.T) {
	// Create different error types
	errors := []error{
		operations.NewExecutionError("test", nil, true),
		operations.NewTimeoutError("test", "30s"),
		operations.NewValidationError("test", "Invalid input"),
	}
	
	for _, err := range errors {
		s := operations.NewStepState("test", "Test")
		s.Fail(err)
		
		testutil.AssertStepStatus(t, s, operations.StepStatusFailed)
		if s.Error == nil {
			t.Error("Error should be set after Fail")
		}
		if s.EndTime == nil {
			t.Error("EndTime should be set after Fail")
		}
	}
}