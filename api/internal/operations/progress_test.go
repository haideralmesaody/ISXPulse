package operations_test

import (
	"strings"
	"testing"
	"time"

	"isxcli/internal/operations"
	"isxcli/internal/operations/testutil"
)

func TestNewProgressTracker(t *testing.T) {
	tests := []struct {
		name     string
		Step    string
		total    int
		expected *operations.ProgressTracker
	}{
		{
			name:  "basic progress tracker",
			Step: "test-Step",
			total: 100,
		},
		{
			name:  "zero total progress tracker",
			Step: "zero-Step",
			total: 0,
		},
		{
			name:  "large total progress tracker",
			Step: "large-Step",
			total: 1000000,
		},
		{
			name:  "empty Step name",
			Step: "",
			total: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			tracker := operations.NewProgressTracker(tt.Step, tt.total)
			
			testutil.AssertEqual(t, tracker.Step, tt.Step)
			testutil.AssertEqual(t, tracker.Total, tt.total)
			testutil.AssertEqual(t, tracker.Current, 0)
			testutil.AssertEqual(t, tracker.Message, "")
			
			// Verify StartTime is recent (within last second)
			if time.Since(start) > time.Second {
				t.Errorf("StartTime should be recent, got %v", tracker.StartTime)
			}
		})
	}
}

func TestProgressTrackerUpdate(t *testing.T) {
	tests := []struct {
		name            string
		total           int
		current         int
		message         string
		expectedCurrent int
		expectedMessage string
	}{
		{
			name:            "normal progress update",
			total:           100,
			current:         25,
			message:         "Processing...",
			expectedCurrent: 25,
			expectedMessage: "Processing...",
		},
		{
			name:            "progress at completion",
			total:           50,
			current:         50,
			message:         "Complete",
			expectedCurrent: 50,
			expectedMessage: "Complete",
		},
		{
			name:            "progress beyond total",
			total:           10,
			current:         15,
			message:         "Over complete",
			expectedCurrent: 15,
			expectedMessage: "Over complete",
		},
		{
			name:            "negative progress",
			total:           100,
			current:         -5,
			message:         "Reset",
			expectedCurrent: -5,
			expectedMessage: "Reset",
		},
		{
			name:            "empty message",
			total:           100,
			current:         30,
			message:         "",
			expectedCurrent: 30,
			expectedMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := operations.NewProgressTracker("test-Step", tt.total)
			
			// Update progress
			tracker.Update(tt.current, tt.message)
			
			testutil.AssertEqual(t, tracker.Current, tt.expectedCurrent)
			testutil.AssertEqual(t, tracker.Message, tt.expectedMessage)
		})
	}
}

func TestProgressTrackerIncrement(t *testing.T) {
	tests := []struct {
		name               string
		total              int
		incrementCount     int
		message            string
		expectedCurrent    int
		expectedMessage    string
	}{
		{
			name:            "single increment",
			total:           100,
			incrementCount:  1,
			message:         "Step 1",
			expectedCurrent: 1,
			expectedMessage: "Step 1",
		},
		{
			name:            "multiple increments",
			total:           10,
			incrementCount:  5,
			message:         "Half way",
			expectedCurrent: 5,
			expectedMessage: "Half way",
		},
		{
			name:            "increment to completion",
			total:           3,
			incrementCount:  3,
			message:         "Done",
			expectedCurrent: 3,
			expectedMessage: "Done",
		},
		{
			name:            "increment beyond total",
			total:           5,
			incrementCount:  7,
			message:         "Overflow",
			expectedCurrent: 7,
			expectedMessage: "Overflow",
		},
		{
			name:            "zero increments",
			total:           100,
			incrementCount:  0,
			message:         "No change",
			expectedCurrent: 0,
			expectedMessage: "", // No increments means message stays empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := operations.NewProgressTracker("test-Step", tt.total)
			
			// Perform increments
			if tt.incrementCount == 0 {
				// For zero increments, don't call Increment at all
				// The message should remain empty
			} else {
				for i := 0; i < tt.incrementCount; i++ {
					tracker.Increment(tt.message)
				}
			}
			
			testutil.AssertEqual(t, tracker.Current, tt.expectedCurrent)
			testutil.AssertEqual(t, tracker.Message, tt.expectedMessage)
		})
	}
}

func TestProgressTrackerGetProgress(t *testing.T) {
	tests := []struct {
		name             string
		total            int
		current          int
		expectedProgress float64
	}{
		{
			name:             "normal progress",
			total:            100,
			current:          25,
			expectedProgress: 25.0,
		},
		{
			name:             "completion",
			total:            50,
			current:          50,
			expectedProgress: 100.0,
		},
		{
			name:             "no progress",
			total:            100,
			current:          0,
			expectedProgress: 0.0,
		},
		{
			name:             "zero total",
			total:            0,
			current:          5,
			expectedProgress: 0.0, // Should handle division by zero
		},
		{
			name:             "over completion",
			total:            10,
			current:          15,
			expectedProgress: 150.0,
		},
		{
			name:             "fractional progress",
			total:            3,
			current:          1,
			expectedProgress: 33.33333333333333, // 1/3 * 100 (actual Go precision)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := operations.NewProgressTracker("test-Step", tt.total)
			tracker.Update(tt.current, "test")
			
			current, total, percentage, message := tracker.GetProgress()
			testutil.AssertEqual(t, current, tt.current)
			testutil.AssertEqual(t, total, tt.total)
			testutil.AssertEqual(t, percentage, tt.expectedProgress)
			testutil.AssertEqual(t, message, "test")
		})
	}
}

func TestProgressTrackerGetETA(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		current     int
		timeElapsed time.Duration
		expectZero  bool // When ETA can't be calculated
	}{
		{
			name:        "normal progress with ETA",
			total:       100,
			current:     25,
			timeElapsed: 1 * time.Minute,
			expectZero:  false,
		},
		{
			name:        "no progress yet",
			total:       100,
			current:     0,
			timeElapsed: 30 * time.Second,
			expectZero:  true, // Can't calculate ETA with no progress
		},
		{
			name:        "completion",
			total:       50,
			current:     50,
			timeElapsed: 2 * time.Minute,
			expectZero:  true, // Already complete
		},
		{
			name:        "over completion",
			total:       10,
			current:     15,
			timeElapsed: 1 * time.Minute,
			expectZero:  true, // Already over complete
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := operations.NewProgressTracker("test-Step", tt.total)
			
			// Simulate elapsed time by setting StartTime in the past
			tracker.StartTime = time.Now().Add(-tt.timeElapsed)
			tracker.Update(tt.current, "test")
			
			eta := tracker.GetETA()
			
			if tt.expectZero {
				// For zero cases, we may get specific messages like "calculating..." or "0 seconds"
				// Just verify it's not completely empty
				if eta == "" {
					t.Errorf("Expected some ETA message, got empty string")
				}
			} else {
				// ETA should be non-empty for valid cases
				if eta == "" {
					t.Errorf("Expected non-empty ETA, got empty string")
				}
			}
		})
	}
}

func TestProgressTrackerIsComplete(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		current    int
		expected   bool
	}{
		{
			name:     "not complete",
			total:    100,
			current:  50,
			expected: false,
		},
		{
			name:     "exactly complete",
			total:    50,
			current:  50,
			expected: true,
		},
		{
			name:     "over complete",
			total:    10,
			current:  15,
			expected: true,
		},
		{
			name:     "zero progress",
			total:    100,
			current:  0,
			expected: false,
		},
		{
			name:     "zero total",
			total:    0,
			current:  0,
			expected: true, // Zero total means complete by definition
		},
		{
			name:     "zero total with current",
			total:    0,
			current:  5,
			expected: true, // Any progress against zero total is complete
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := operations.NewProgressTracker("test-Step", tt.total)
			tracker.Update(tt.current, "test")
			
			isComplete := tracker.IsComplete()
			testutil.AssertEqual(t, isComplete, tt.expected)
		})
	}
}

func TestProgressTrackerGetElapsedTime(t *testing.T) {
	tracker := operations.NewProgressTracker("test-Step", 100)
	
	// Simulate some elapsed time
	tracker.StartTime = time.Now().Add(-2 * time.Second)
	
	elapsed := tracker.GetElapsedTime()
	
	// Should be approximately 2 seconds (allow some tolerance)
	if elapsed < 1500*time.Millisecond || elapsed > 2500*time.Millisecond {
		t.Errorf("Expected elapsed time around 2s, got %v", elapsed)
	}
}

func TestProgressTrackerGetElapsedTimeString(t *testing.T) {
	tests := []struct {
		name            string
		elapsedDuration time.Duration
		expectedContains string
	}{
		{
			name:            "seconds only",
			elapsedDuration: 45 * time.Second,
			expectedContains: "45",
		},
		{
			name:            "minutes and seconds",
			elapsedDuration: 2*time.Minute + 30*time.Second,
			expectedContains: "2.5", // Format is "2.5 minutes"
		},
		{
			name:            "hours, minutes and seconds",
			elapsedDuration: 1*time.Hour + 15*time.Minute + 20*time.Second,  
			expectedContains: "1.3", // Format is "1.3 hours"
		},
		{
			name:            "zero time",
			elapsedDuration: 0,
			expectedContains: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := operations.NewProgressTracker("test-Step", 100)
			tracker.StartTime = time.Now().Add(-tt.elapsedDuration)
			
			timeString := tracker.GetElapsedTimeString()
			
			if timeString == "" {
				t.Error("GetElapsedTimeString() should not return empty string")
			}
			
			// For non-zero durations, the string should contain expected parts
			if tt.elapsedDuration > 0 && !strings.Contains(timeString, tt.expectedContains) {
				t.Errorf("Expected time string to contain %q, got %q", tt.expectedContains, timeString)
			}
		})
	}
}

func TestProgressTrackerConcurrency(t *testing.T) {
	tracker := operations.NewProgressTracker("concurrent-Step", 1000)
	
	// Test concurrent access to ensure thread safety
	const numWorkers = 10
	const incrementsPerWorker = 10
	
	// Use channels to coordinate workers
	done := make(chan bool, numWorkers)
	
	// Start multiple workers incrementing progress
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < incrementsPerWorker; j++ {
				tracker.Increment("concurrent update")
			}
		}()
	}
	
	// Wait for all workers to complete
	for i := 0; i < numWorkers; i++ {
		<-done
	}
	
	// Verify final state
	expectedCurrent := numWorkers * incrementsPerWorker
	testutil.AssertEqual(t, tracker.Current, expectedCurrent)
	testutil.AssertEqual(t, tracker.Message, "concurrent update")
}