package operations_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

func TestNewOperationState(t *testing.T) {
	id := "test-operation"
	state := operations.NewOperationState(id)
	
	// Verify initial values
	testutil.AssertEqual(t, state.ID, id)
	testutil.AssertOperationStatus(t, state, operations.OperationStatusPending)
	testutil.AssertNotNil(t, state.Steps)
	testutil.AssertNotNil(t, state.Context)
	testutil.AssertNotNil(t, state.Config)
	
	if state.EndTime != nil {
		t.Error("EndTime should be nil initially")
	}
	if state.Error != nil {
		t.Error("Error should be nil initially")
	}
	
	// Verify start time is set
	if state.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestOperationStateTransitions(t *testing.T) {
	tests := []struct {
		name       string
		transition func(*operations.OperationState)
		wantStatus operations.OperationStatus
		checkEnd   bool
		checkError bool
	}{
		{
			name: "Start",
			transition: func(p *operations.OperationState) {
				p.Start()
			},
			wantStatus: operations.OperationStatusRunning,
			checkEnd:   false,
		},
		{
			name: "Complete",
			transition: func(p *operations.OperationState) {
				p.Complete()
			},
			wantStatus: operations.OperationStatusCompleted,
			checkEnd:   true,
		},
		{
			name: "Fail",
			transition: func(p *operations.OperationState) {
				p.Fail(errors.New("test error"))
			},
			wantStatus: operations.OperationStatusFailed,
			checkEnd:   true,
			checkError: true,
		},
		{
			name: "Cancel",
			transition: func(p *operations.OperationState) {
				p.Cancel()
			},
			wantStatus: operations.OperationStatusCancelled,
			checkEnd:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := operations.NewOperationState("test")
			
			tt.transition(state)
			
			testutil.AssertOperationStatus(t, state, tt.wantStatus)
			
			if tt.checkEnd && state.EndTime == nil {
				t.Error("EndTime should be set")
			}
			if !tt.checkEnd && state.EndTime != nil {
				t.Error("EndTime should not be set")
			}
			if tt.checkError && state.Error == nil {
				t.Error("Error should be set")
			}
		})
	}
}

func TestOperationStateStageManagement(t *testing.T) {
	state := operations.NewOperationState("test")
	
	// Add steps
	stage1 := operations.NewStepState("stage1", "Step 1")
	stage2 := operations.NewStepState("stage2", "Step 2")
	stage3 := operations.NewStepState("stage3", "Step 3")
	
	state.SetStage("stage1", stage1)
	state.SetStage("stage2", stage2)
	state.SetStage("stage3", stage3)
	
	// Retrieve steps
	got1 := state.GetStage("stage1")
	got2 := state.GetStage("stage2")
	got3 := state.GetStage("stage3")
	gotNil := state.GetStage("nonexistent")
	
	if got1 != stage1 {
		t.Error("Step 1 not retrieved correctly")
	}
	if got2 != stage2 {
		t.Error("Step 2 not retrieved correctly")
	}
	if got3 != stage3 {
		t.Error("Step 3 not retrieved correctly")
	}
	if gotNil != nil {
		t.Error("Nonexistent Step should return nil")
	}
}

func TestOperationStateContext(t *testing.T) {
	state := operations.NewOperationState("test")
	
	// Test setting and getting context values
	state.SetContext("key1", "value1")
	state.SetContext("key2", 42)
	state.SetContext("key3", true)
	
	// Get values
	val1, ok1 := state.GetContext("key1")
	val2, ok2 := state.GetContext("key2")
	val3, ok3 := state.GetContext("key3")
	_, ok4 := state.GetContext("nonexistent")
	
	if !ok1 || val1 != "value1" {
		t.Error("Context key1 not retrieved correctly")
	}
	if !ok2 || val2 != 42 {
		t.Error("Context key2 not retrieved correctly")
	}
	if !ok3 || val3 != true {
		t.Error("Context key3 not retrieved correctly")
	}
	if ok4 {
		t.Error("Nonexistent key should return false")
	}
}

func TestOperationStateConfig(t *testing.T) {
	state := operations.NewOperationState("test")
	
	// Test setting and getting config values
	state.SetConfig("mode", "initial")
	state.SetConfig("timeout", 30)
	state.SetConfig("retry", true)
	
	// Get values
	val1, ok1 := state.GetConfig("mode")
	val2, ok2 := state.GetConfig("timeout")
	val3, ok3 := state.GetConfig("retry")
	_, ok4 := state.GetConfig("nonexistent")
	
	if !ok1 || val1 != "initial" {
		t.Error("Config mode not retrieved correctly")
	}
	if !ok2 || val2 != 30 {
		t.Error("Config timeout not retrieved correctly")
	}
	if !ok3 || val3 != true {
		t.Error("Config retry not retrieved correctly")
	}
	if ok4 {
		t.Error("Nonexistent key should return false")
	}
}

func TestOperationStateDuration(t *testing.T) {
	state := operations.NewOperationState("test")
	
	// Start the operation
	state.Start()
	time.Sleep(50 * time.Millisecond)
	
	// Check duration while running
	duration := state.Duration()
	if duration <= 0 {
		t.Error("Duration should be > 0 while running")
	}
	
	// Complete the operation
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

func TestOperationStateStageQueries(t *testing.T) {
	state := operations.NewOperationState("test")
	
	// Add steps with different statuses
	active1 := operations.NewStepState("active1", "Active 1")
	active1.Status = operations.StepStatusActive
	
	active2 := operations.NewStepState("active2", "Active 2")
	active2.Status = operations.StepStatusActive
	
	completed := operations.NewStepState("completed", "Completed")
	completed.Status = operations.StepStatusCompleted
	
	failed := operations.NewStepState("failed", "Failed")
	failed.Status = operations.StepStatusFailed
	
	pending := operations.NewStepState("pending", "Pending")
	pending.Status = operations.StepStatusPending
	
	state.SetStage("active1", active1)
	state.SetStage("active2", active2)
	state.SetStage("completed", completed)
	state.SetStage("failed", failed)
	state.SetStage("pending", pending)
	
	// Test GetActiveStages
	activeStages := state.GetActiveStages()
	if len(activeStages) != 2 {
		t.Errorf("Active steps count = %d, want 2", len(activeStages))
	}
	
	// Test GetCompletedStages
	completedStages := state.GetCompletedStages()
	if len(completedStages) != 1 {
		t.Errorf("Completed steps count = %d, want 1", len(completedStages))
	}
	
	// Test GetFailedStages
	failedStages := state.GetFailedStages()
	if len(failedStages) != 1 {
		t.Errorf("Failed steps count = %d, want 1", len(failedStages))
	}
}

func TestOperationStateIsComplete(t *testing.T) {
	tests := []struct {
		name     string
		steps   map[string]operations.StepStatus
		want     bool
	}{
		{
			name: "All completed",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusCompleted,
				"s2": operations.StepStatusCompleted,
				"s3": operations.StepStatusCompleted,
			},
			want: true,
		},
		{
			name: "Some skipped",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusCompleted,
				"s2": operations.StepStatusSkipped,
				"s3": operations.StepStatusCompleted,
			},
			want: true,
		},
		{
			name: "Has pending",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusCompleted,
				"s2": operations.StepStatusPending,
				"s3": operations.StepStatusCompleted,
			},
			want: false,
		},
		{
			name: "Has active",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusCompleted,
				"s2": operations.StepStatusActive,
				"s3": operations.StepStatusCompleted,
			},
			want: false,
		},
		{
			name: "Has failed",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusCompleted,
				"s2": operations.StepStatusFailed,
				"s3": operations.StepStatusCompleted,
			},
			want: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := operations.NewOperationState("test")
			
			for id, status := range tt.steps {
				Step := operations.NewStepState(id, id)
				Step.Status = status
				state.SetStage(id, Step)
			}
			
			got := state.IsComplete()
			if got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationStateHasFailures(t *testing.T) {
	tests := []struct {
		name   string
		steps map[string]operations.StepStatus
		want   bool
	}{
		{
			name: "No failures",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusCompleted,
				"s2": operations.StepStatusCompleted,
			},
			want: false,
		},
		{
			name: "Has failure",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusCompleted,
				"s2": operations.StepStatusFailed,
			},
			want: true,
		},
		{
			name: "Multiple failures",
			steps: map[string]operations.StepStatus{
				"s1": operations.StepStatusFailed,
				"s2": operations.StepStatusFailed,
			},
			want: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := operations.NewOperationState("test")
			
			for id, status := range tt.steps {
				Step := operations.NewStepState(id, id)
				Step.Status = status
				state.SetStage(id, Step)
			}
			
			got := state.HasFailures()
			if got != tt.want {
				t.Errorf("HasFailures() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationStateClone(t *testing.T) {
	// Create original state
	original := operations.NewOperationState("test")
	original.Status = operations.OperationStatusRunning
	original.SetContext("key1", "value1")
	original.SetConfig("config1", "configValue1")
	
	// Add steps
	stage1 := operations.NewStepState("stage1", "Step 1")
	stage1.Status = operations.StepStatusCompleted
	original.SetStage("stage1", stage1)
	
	// Clone
	clone := original.Clone()
	
	// Verify clone has same values
	testutil.AssertEqual(t, clone.ID, original.ID)
	testutil.AssertOperationStatus(t, clone, original.Status)
	
	// Verify context was cloned
	val, ok := clone.GetContext("key1")
	if !ok || val != "value1" {
		t.Error("Context not cloned correctly")
	}
	
	// Verify config was cloned
	val, ok = clone.GetConfig("config1")
	if !ok || val != "configValue1" {
		t.Error("Config not cloned correctly")
	}
	
	// Verify steps were cloned
	clonedStage := clone.GetStage("stage1")
	if clonedStage == nil || clonedStage.Status != operations.StepStatusCompleted {
		t.Error("steps not cloned correctly")
	}
	
	// Verify modifications to clone don't affect original
	clone.SetContext("key2", "value2")
	_, ok = original.GetContext("key2")
	if ok {
		t.Error("Clone modifications affected original")
	}
}

func TestOperationStateConcurrency(t *testing.T) {
	state := operations.NewOperationState("test")
	
	// Run concurrent operations
	var wg sync.WaitGroup
	ops := 100
	
	// Concurrent context writes
	wg.Add(ops)
	for i := 0; i < ops; i++ {
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", n)
			state.SetContext(key, n)
		}(i)
	}
	
	// Concurrent config writes
	wg.Add(ops)
	for i := 0; i < ops; i++ {
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("config%d", n)
			state.SetConfig(key, n)
		}(i)
	}
	
	// Concurrent Step writes
	wg.Add(ops)
	for i := 0; i < ops; i++ {
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("Step%d", n)
			Step := operations.NewStepState(id, id)
			state.SetStage(id, Step)
		}(i)
	}
	
	// Concurrent reads
	wg.Add(ops)
	for i := 0; i < ops; i++ {
		go func(n int) {
			defer wg.Done()
			state.GetActiveStages()
			state.GetCompletedStages()
			state.GetFailedStages()
			state.IsComplete()
			state.HasFailures()
			state.Duration()
		}(i)
	}
	
	wg.Wait()
	
	// Verify all writes succeeded
	for i := 0; i < ops; i++ {
		key := fmt.Sprintf("key%d", i)
		val, ok := state.GetContext(key)
		if !ok || val != i {
			t.Errorf("Context %s not set correctly", key)
		}
		
		key = fmt.Sprintf("config%d", i)
		val, ok = state.GetConfig(key)
		if !ok || val != i {
			t.Errorf("Config %s not set correctly", key)
		}
		
		id := fmt.Sprintf("Step%d", i)
		Step := state.GetStage(id)
		if Step == nil {
			t.Errorf("Step %s not set correctly", id)
		}
	}
}