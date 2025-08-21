package websocket

import (
	"context"
	"fmt"
	"time"
)

// OperationsEventExamples demonstrates how operations events are sent through WebSocket
type OperationsEventExamples struct {
	hub *Hub
}

// NewOperationsEventExamples creates a new examples instance
func NewOperationsEventExamples(hub *Hub) *OperationsEventExamples {
	return &OperationsEventExamples{hub: hub}
}

// SendOperationLifecycleEvents demonstrates the complete lifecycle of an operation
func (e *OperationsEventExamples) SendOperationLifecycleEvents(ctx context.Context, operationID string) {
	// 1. Operation Reset/Initialized
	e.hub.BroadcastUpdate("operation:reset", operationID, "initialized", nil)
	
	// 2. Operation Started
	e.hub.BroadcastUpdate("operation:started", operationID, "running", map[string]interface{}{
		"operation_type": "full_pipeline",
		"mode":           "full",
		"steps_total":    4,
		"started_by":     "user@example.com",
		"started_at":     time.Now().UTC(),
	})
	
	// 3. Step Progress Updates
	steps := []struct {
		id       string
		name     string
		progress []float64
	}{
		{id: "step-download", name: "Downloading data", progress: []float64{25, 50, 75, 100}},
		{id: "step-process", name: "Processing files", progress: []float64{33, 66, 100}},
		{id: "step-analyze", name: "Analyzing results", progress: []float64{50, 100}},
		{id: "step-export", name: "Exporting reports", progress: []float64{100}},
	}
	
	for i, step := range steps {
		// Step started
		e.hub.BroadcastUpdate("operation:progress", step.id, "active", map[string]interface{}{
			"operation_id":   operationID,
			"step_name":      step.name,
			"step_number":    i + 1,
			"steps_complete": i,
			"steps_total":    4,
			"message":        fmt.Sprintf("Starting %s", step.name),
		})
		
		// Step progress
		for _, prog := range step.progress {
			e.hub.BroadcastUpdate("operation:progress", step.id, "active", map[string]interface{}{
				"operation_id":    operationID,
				"progress":        prog,
				"items_processed": int(prog * 10), // Example: processing 1000 items
				"items_total":     1000,
				"message":         fmt.Sprintf("%s: %.0f%% complete", step.name, prog),
			})
		}
		
		// Step completed
		e.hub.BroadcastUpdate("operation:progress", step.id, "completed", map[string]interface{}{
			"operation_id":   operationID,
			"step_name":      step.name,
			"steps_complete": i + 1,
			"steps_total":    4,
			"message":        fmt.Sprintf("Completed %s", step.name),
		})
	}
	
	// 4. Operation Completed
	e.hub.BroadcastUpdate("operation:completed", operationID, "completed", map[string]interface{}{
		"duration":     "2m30s",
		"completed_at": time.Now().UTC(),
		"results": map[string]interface{}{
			"files_processed": 10,
			"records_created": 1523,
			"errors":          0,
			"output_files": []string{
				"reports/daily_summary.csv",
				"reports/ticker_analysis.csv",
			},
		},
		"metrics": map[string]interface{}{
			"download_time_ms": 5230,
			"process_time_ms":  45000,
			"analyze_time_ms":  30000,
			"export_time_ms":   10000,
			"total_time_ms":    90230,
		},
	})
}

// SendOperationErrorExample demonstrates an operation that fails
func (e *OperationsEventExamples) SendOperationErrorExample(ctx context.Context, operationID string) {
	// Operation starts normally
	e.hub.BroadcastUpdate("operation:started", operationID, "running", map[string]interface{}{
		"operation_type": "data_import",
		"steps_total":    3,
	})
	
	// First step succeeds
	e.hub.BroadcastUpdate("operation:progress", "step-validate", "completed", map[string]interface{}{
		"operation_id": operationID,
		"step_name":    "Validation",
		"message":      "Data validation completed successfully",
	})
	
	// Second step fails
	e.hub.BroadcastUpdate("operation:error", "step-import", "failed", map[string]interface{}{
		"operation_id": operationID,
		"error":        "Database connection timeout",
		"error_code":   "DB_CONN_TIMEOUT",
		"step_name":    "Data Import",
		"can_retry":    true,
		"retry_count":  2,
		"stack_trace":  "connection.go:123 - timeout after 30s",
	})
	
	// Operation failed
	e.hub.BroadcastUpdate("operation:failed", operationID, "failed", map[string]interface{}{
		"error":         "Operation failed during data import step",
		"failed_at":     time.Now().UTC(),
		"partial_results": map[string]interface{}{
			"validated_records": 500,
			"imported_records":  0,
		},
	})
}

// SendOperationCancelExample demonstrates a cancelled operation
func (e *OperationsEventExamples) SendOperationCancelExample(ctx context.Context, operationID string) {
	// Operation in progress
	e.hub.BroadcastUpdate("operation:progress", "step-process", "active", map[string]interface{}{
		"operation_id": operationID,
		"progress":     35.5,
		"message":      "Processing large dataset...",
	})
	
	// Operation cancelled
	e.hub.BroadcastUpdate("operation:cancelled", operationID, "cancelled", map[string]interface{}{
		"cancelled_at": time.Now().UTC(),
		"cancelled_by": "user@example.com",
		"reason":       "User requested cancellation",
		"cleanup_status": "completed",
		"partial_results": map[string]interface{}{
			"processed_before_cancel": 355,
			"total_planned":           1000,
		},
	})
}

// Message Format Examples for Frontend
/*
The frontend will receive these WebSocket messages in the following format:

1. Operation Started:
{
  "type": "operation:start",
  "operation_id": "op-123",
  "status": "running",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "operation_type": "full_pipeline",
    "mode": "full",
    "steps_total": 4,
    "started_by": "user@example.com",
    "started_at": "2024-01-15T10:30:00Z"
  }
}

2. Step Progress:
{
  "type": "operation:progress",
  "timestamp": "2024-01-15T10:30:15Z",
  "data": {
    "step_id": "step-download",
    "status": "active",
    "operation_id": "op-123",
    "progress": 75,
    "items_processed": 750,
    "items_total": 1000,
    "message": "Downloading data: 75% complete"
  }
}

3. Operation Error:
{
  "type": "operation:failed",
  "timestamp": "2024-01-15T10:31:00Z",
  "data": {
    "operation_id": "op-123",
    "status": "failed",
    "error": "Database connection timeout",
    "error_code": "DB_CONN_TIMEOUT",
    "can_retry": true
  }
}

4. Operation Complete:
{
  "type": "operation:complete",
  "operation_id": "op-123",
  "status": "completed",
  "timestamp": "2024-01-15T10:35:00Z",
  "data": {
    "duration": "5m0s",
    "completed_at": "2024-01-15T10:35:00Z",
    "results": {
      "files_processed": 10,
      "records_created": 1523,
      "errors": 0
    }
  }
}

The frontend should handle these event types:
- operation:reset - Operation initialized
- operation:start - Operation began execution
- operation:progress - Step progress update
- operation:complete - Operation finished successfully
- operation:failed - Operation encountered an error
- operation:cancelled - Operation was cancelled
*/