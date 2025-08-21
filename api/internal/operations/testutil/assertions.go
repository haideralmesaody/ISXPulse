package testutil

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"isxcli/internal/operations"
)

// AssertStepStatus verifies a step has the expected status
func AssertStepStatus(t *testing.T, step *operations.StepState, expected operations.StepStatus) {
	t.Helper()
	if step == nil {
		t.Fatal("step state is nil")
	}
	if step.Status != expected {
		t.Errorf("step %s status = %v, want %v", step.ID, step.Status, expected)
	}
}

// AssertOperationStatus verifies a operation has the expected status
func AssertOperationStatus(t *testing.T, p *operations.OperationState, expected operations.OperationStatusValue) {
	t.Helper()
	if p == nil {
		t.Fatal("operation state is nil")
	}
	if p.Status != expected {
		t.Errorf("operation status = %v, want %v", p.Status, expected)
	}
}

// AssertWebSocketMessage verifies a WebSocket message was sent
func AssertWebSocketMessage(t *testing.T, hub *MockWebSocketHub, eventType string) {
	t.Helper()
	messages := hub.GetMessagesByType(eventType)
	if len(messages) == 0 {
		t.Errorf("no WebSocket message of type %s found", eventType)
	}
}

// AssertWebSocketMessageCount verifies the number of WebSocket messages
func AssertWebSocketMessageCount(t *testing.T, hub *MockWebSocketHub, eventType string, expected int) {
	t.Helper()
	messages := hub.GetMessagesByType(eventType)
	if len(messages) != expected {
		t.Errorf("WebSocket message count for %s = %d, want %d", eventType, len(messages), expected)
	}
}

// AssertStageCompleted verifies a step completed successfully
func AssertStageCompleted(t *testing.T, p *operations.OperationState, stageID string) {
	t.Helper()
	step := p.GetStage(stageID)
	if step == nil {
		t.Fatalf("step %s not found", stageID)
	}
	AssertStepStatus(t, step, operations.StepStatusCompleted)
	if step.Progress != 100 {
		t.Errorf("step %s progress = %v, want 100", stageID, step.Progress)
	}
}

// AssertStageFailed verifies a step failed
func AssertStageFailed(t *testing.T, p *operations.OperationState, stageID string) {
	t.Helper()
	step := p.GetStage(stageID)
	if step == nil {
		t.Fatalf("step %s not found", stageID)
	}
	AssertStepStatus(t, step, operations.StepStatusFailed)
	if step.Error == nil {
		t.Errorf("step %s has no error", stageID)
	}
}

// AssertStageSkipped verifies a step was skipped
func AssertStageSkipped(t *testing.T, p *operations.OperationState, stageID string) {
	t.Helper()
	step := p.GetStage(stageID)
	if step == nil {
		t.Fatalf("step %s not found", stageID)
	}
	AssertStepStatus(t, step, operations.StepStatusSkipped)
}

// AssertDuration verifies a duration is within tolerance
func AssertDuration(t *testing.T, actual, expected, tolerance time.Duration) {
	t.Helper()
	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("duration = %v, want %v ± %v", actual, expected, tolerance)
	}
}

// AssertProgress verifies step progress
func AssertProgress(t *testing.T, step *operations.StepState, expected float64) {
	t.Helper()
	if step == nil {
		t.Fatal("step state is nil")
	}
	if math.Abs(step.Progress-expected) > 0.01 {
		t.Errorf("step %s progress = %v, want %v", step.ID, step.Progress, expected)
	}
}

// AssertError verifies an error matches expected
func AssertError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Errorf("error = %v, wantErr %v", err, wantErr)
	}
}

// AssertErrorContains verifies an error contains a substring
func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error containing %q, got nil", substr)
		return
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error = %v, want error containing %q", err, substr)
	}
}

// AssertErrorType verifies the type of a operation error
func AssertErrorType(t *testing.T, err error, expectedType operations.ErrorType) {
	t.Helper()
	if err == nil {
		t.Fatal("error is nil")
	}
	pErr, ok := err.(*operations.OperationError)
	if !ok {
		t.Fatalf("error is not a OperationError: %T", err)
	}
	if pErr.Type != expectedType {
		t.Errorf("error type = %v, want %v", pErr.Type, expectedType)
	}
}

// AssertNoError fails if there is an error
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertEqual verifies two values are equal
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotNil verifies a value is not nil
func AssertNotNil(t *testing.T, v interface{}) {
	t.Helper()
	if v == nil {
		t.Fatal("value is nil")
	}
}

// AssertStageOrder verifies steps were executed in the expected order
func AssertStageOrder(t *testing.T, steps []*MockStage, expectedOrder []string) {
	t.Helper()
	
	// Build execution order from call times
	type execution struct {
		id   string
		time time.Time
	}
	
	var executions []execution
	for _, step := range steps {
		if len(step.ExecuteArgs) > 0 {
			executions = append(executions, execution{
				id:   step.ID(),
				time: step.ExecuteArgs[0].Time,
			})
		}
	}
	
	// Sort by time
	for i := 0; i < len(executions)-1; i++ {
		for j := i + 1; j < len(executions); j++ {
			if executions[j].time.Before(executions[i].time) {
				executions[i], executions[j] = executions[j], executions[i]
			}
		}
	}
	
	// Check order
	if len(executions) != len(expectedOrder) {
		t.Errorf("executed %d steps, expected %d", len(executions), len(expectedOrder))
		return
	}
	
	for i, exec := range executions {
		if exec.id != expectedOrder[i] {
			t.Errorf("execution order[%d] = %s, want %s", i, exec.id, expectedOrder[i])
		}
	}
}

// AssertLogContains verifies a log contains a message
func AssertLogContains(t *testing.T, logger *MockLogger, level, substr string) {
	t.Helper()
	
	var logs []LogEntry
	switch level {
	case "info":
		logs = logger.GetInfoLogs()
	case "error":
		logs = logger.GetErrorLogs()
	case "warning":
		logs = logger.GetWarningLogs()
	default:
		t.Fatalf("unknown log level: %s", level)
	}
	
	for _, log := range logs {
		msg := fmt.Sprintf(log.Format, log.Args...)
		if strings.Contains(msg, substr) {
			return
		}
	}
	
	t.Errorf("no %s log contains %q", level, substr)
}

// AssertContextValue verifies a operation context value
func AssertContextValue(t *testing.T, state *operations.OperationState, key string, expected interface{}) {
	t.Helper()
	val, ok := state.GetContext(key)
	if !ok {
		t.Errorf("context key %q not found", key)
		return
	}
	if val != expected {
		t.Errorf("context[%q] = %v, want %v", key, val, expected)
	}
}

// AssertConfigValue verifies a operation config value
func AssertConfigValue(t *testing.T, state *operations.OperationState, key string, expected interface{}) {
	t.Helper()
	val, ok := state.GetConfig(key)
	if !ok {
		t.Errorf("config key %q not found", key)
		return
	}
	if val != expected {
		t.Errorf("config[%q] = %v, want %v", key, val, expected)
	}
}