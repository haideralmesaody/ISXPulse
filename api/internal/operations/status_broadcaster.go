package operations

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// StatusBroadcaster is the single authority for all operation status updates
// It maintains the complete state of all operations and broadcasts snapshots
type StatusBroadcaster struct {
	mu         sync.RWMutex
	operations map[string]*OperationSnapshot
	hub        WebSocketHub
	logger     *slog.Logger
	updates    chan updateRequest
	stop       chan struct{}
}

// OperationSnapshot represents the complete state of an operation at a point in time
// This is the ONLY structure sent to the frontend
type OperationSnapshot struct {
	OperationID string         `json:"operation_id"`
	Status      string         `json:"status"`       // pending|running|completed|failed|cancelled
	Progress    int            `json:"progress"`     // 0-100
	CurrentStep string         `json:"current_step"` // Current active step name
	Steps       []StepSnapshot `json:"steps"`        // All steps with their status
	StartedAt   time.Time      `json:"started_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Error       string         `json:"error,omitempty"`
	Message     string         `json:"message,omitempty"`
}

// StepSnapshot represents the state of a single step
type StepSnapshot struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Status   string                 `json:"status"`   // pending|running|completed|failed|skipped
	Progress int                    `json:"progress"` // 0-100
	Message  string                 `json:"message,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type updateRequest struct {
	operationID string
	updateFunc  func(*OperationSnapshot)
	done        chan struct{}
}

// NewStatusBroadcaster creates a new status broadcaster
func NewStatusBroadcaster(hub WebSocketHub, logger *slog.Logger) *StatusBroadcaster {
	if logger == nil {
		logger = slog.Default()
	}

	sb := &StatusBroadcaster{
		operations: make(map[string]*OperationSnapshot),
		hub:        hub,
		logger:     logger,
		updates:    make(chan updateRequest, 100),
		stop:       make(chan struct{}),
	}

	// Start the update processor
	go sb.processUpdates()

	return sb
}

// processUpdates handles all updates sequentially to avoid race conditions
func (sb *StatusBroadcaster) processUpdates() {
	for {
		select {
		case <-sb.stop:
			return
		case req := <-sb.updates:
			sb.handleUpdate(req)
		}
	}
}

// handleUpdate processes a single update request
func (sb *StatusBroadcaster) handleUpdate(req updateRequest) {
	defer close(req.done)

	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Get or create snapshot
	snapshot, exists := sb.operations[req.operationID]
	if !exists {
		snapshot = &OperationSnapshot{
			OperationID: req.operationID,
			Status:      "pending",
			Progress:    0,
			StartedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Steps:       []StepSnapshot{},
		}
		sb.operations[req.operationID] = snapshot
	}

	// Apply the update
	req.updateFunc(snapshot)
	snapshot.UpdatedAt = time.Now()

	// Calculate overall progress from steps
	if len(snapshot.Steps) > 0 {
		totalProgress := 0
		for _, step := range snapshot.Steps {
			totalProgress += step.Progress
		}
		snapshot.Progress = totalProgress / len(snapshot.Steps)
	}

	// Set completed time if status is terminal
	if snapshot.Status == "completed" || snapshot.Status == "failed" || snapshot.Status == "cancelled" {
		if snapshot.CompletedAt == nil {
			now := time.Now()
			snapshot.CompletedAt = &now
		}
	}

	// Broadcast the complete snapshot
	sb.broadcast(snapshot)
}

// broadcast sends the complete snapshot to all connected clients
func (sb *StatusBroadcaster) broadcast(snapshot *OperationSnapshot) {
	if sb.hub == nil {
		sb.logger.Warn("no websocket hub configured for status broadcast")
		return
	}

	// Log the broadcast
	sb.logger.Info("broadcasting operation snapshot",
		slog.String("operation_id", snapshot.OperationID),
		slog.String("status", snapshot.Status),
		slog.Int("progress", snapshot.Progress),
		slog.String("current_step", snapshot.CurrentStep),
		slog.Int("steps", len(snapshot.Steps)),
	)

	// Send single, complete snapshot and include a minimal debug envelope too
	sb.hub.BroadcastUpdate("operation:snapshot", snapshot.OperationID, "update", snapshot)
}

// UpdateStatus updates the status of an operation
// This is the ONLY method that should be called to update operation status
func (sb *StatusBroadcaster) UpdateStatus(operationID string, updateFunc func(*OperationSnapshot)) {
	req := updateRequest{
		operationID: operationID,
		updateFunc:  updateFunc,
		done:        make(chan struct{}),
	}

	sb.updates <- req
	<-req.done // Wait for update to complete
}

// CreateOperation initializes a new operation with the given steps
// stepNames MUST be stable step IDs to allow future updates to match correctly.
func (sb *StatusBroadcaster) CreateOperation(operationID string, stepNames []string) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		snapshot.Status = "pending"
		snapshot.Progress = 0
		snapshot.Steps = make([]StepSnapshot, len(stepNames))
		for i, name := range stepNames {
			// Treat incoming slice values as step IDs; set Name to ID initially.
			// Later, CompleteStep/updates can set a human-readable Name if needed.
			snapshot.Steps[i] = StepSnapshot{
				ID:       name,
				Name:     name,
				Status:   "pending",
				Progress: 0,
			}
		}
		snapshot.Message = "Operation created"
	})
}

// StartOperation marks an operation as running
func (sb *StatusBroadcaster) StartOperation(operationID string) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		snapshot.Status = "running"
		snapshot.Message = "Operation started"
	})
}

// UpdateStepProgress updates a specific step's progress
func (sb *StatusBroadcaster) UpdateStepProgress(operationID, stepID string, progress int, message string) {
	sb.UpdateStepWithMetadata(operationID, stepID, progress, message, nil)
}

// UpdateStepWithMetadata updates a specific step's progress with metadata
func (sb *StatusBroadcaster) UpdateStepWithMetadata(operationID, stepID string, progress int, message string, metadata map[string]interface{}) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		for i := range snapshot.Steps {
			if snapshot.Steps[i].ID == stepID {
				// Enforce monotonic step progress while running to avoid regressions
				// that can happen when late/emitted events (e.g., total inference)
				// arrive out of order.
				if progress < snapshot.Steps[i].Progress && snapshot.Steps[i].Status == "running" {
					// Keep the higher progress already observed
				} else {
					snapshot.Steps[i].Progress = progress
				}
				snapshot.Steps[i].Message = message
				if metadata != nil {
					snapshot.Steps[i].Metadata = metadata
				}
				if progress > 0 && progress < 100 {
					snapshot.Steps[i].Status = "running"
					snapshot.CurrentStep = snapshot.Steps[i].Name
				} else if progress >= 100 {
					snapshot.Steps[i].Status = "completed"
					snapshot.Steps[i].Progress = 100
				}
				break
			}
		}
		// If no steps updated (e.g., ID mismatch), create or update the step by ID
		// This prevents the UI from stalling when step IDs differ from names.
		found := false
		for i := range snapshot.Steps {
			if snapshot.Steps[i].ID == stepID {
				found = true
				break
			}
		}
		if !found {
			// Append a minimal step entry so progress can continue
			snapshot.Steps = append(snapshot.Steps, StepSnapshot{
				ID:       stepID,
				Name:     stepID,
				Status:   map[bool]string{true: "completed", false: "running"}[progress >= 100],
				Progress: minInt(maxInt(progress, 0), 100),
				Message:  message,
				Metadata: metadata,
			})
			if progress > 0 && progress < 100 {
				snapshot.CurrentStep = stepID
			}
		}
	})
}

// Helpers to clamp ints without importing math just for this
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CompleteStep marks a step as completed
func (sb *StatusBroadcaster) CompleteStep(operationID, stepID string, message string) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		for i := range snapshot.Steps {
			if snapshot.Steps[i].ID == stepID {
				snapshot.Steps[i].Status = "completed"
				snapshot.Steps[i].Progress = 100
				snapshot.Steps[i].Message = message
				break
			}
		}
	})
}

// FailStep marks a step as failed
func (sb *StatusBroadcaster) FailStep(operationID, stepID string, err error) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		for i := range snapshot.Steps {
			if snapshot.Steps[i].ID == stepID {
				snapshot.Steps[i].Status = "failed"
				snapshot.Steps[i].Error = err.Error()
				break
			}
		}
	})
}

// CompleteOperation marks an operation as completed
func (sb *StatusBroadcaster) CompleteOperation(operationID string, message string) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		snapshot.Status = "completed"
		snapshot.Progress = 100
		snapshot.CurrentStep = ""
		snapshot.Message = message
		// Ensure all steps are marked as completed
		for i := range snapshot.Steps {
			if snapshot.Steps[i].Status == "running" || snapshot.Steps[i].Status == "pending" {
				snapshot.Steps[i].Status = "completed"
				snapshot.Steps[i].Progress = 100
			}
		}
	})
}

// FailOperation marks an operation as failed
func (sb *StatusBroadcaster) FailOperation(operationID string, err error) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		snapshot.Status = "failed"
		snapshot.Error = err.Error()
		snapshot.CurrentStep = ""
	})
}

// CancelOperation marks an operation as cancelled
func (sb *StatusBroadcaster) CancelOperation(operationID string) {
	sb.UpdateStatus(operationID, func(snapshot *OperationSnapshot) {
		snapshot.Status = "cancelled"
		snapshot.CurrentStep = ""
		snapshot.Message = "Operation cancelled by user"
	})
}

// GetSnapshot returns the current snapshot for an operation
func (sb *StatusBroadcaster) GetSnapshot(operationID string) (*OperationSnapshot, bool) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	snapshot, exists := sb.operations[operationID]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modification
	copy := *snapshot
	return &copy, true
}

// GetAllSnapshots returns all current operation snapshots
func (sb *StatusBroadcaster) GetAllSnapshots() []*OperationSnapshot {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	snapshots := make([]*OperationSnapshot, 0, len(sb.operations))
	for _, snapshot := range sb.operations {
		copy := *snapshot
		snapshots = append(snapshots, &copy)
	}

	return snapshots
}

// CleanupOldOperations removes operations older than the specified duration
func (sb *StatusBroadcaster) CleanupOldOperations(ctx context.Context, maxAge time.Duration) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	now := time.Now()
	for id, snapshot := range sb.operations {
		// Only cleanup completed/failed/cancelled operations
		if snapshot.Status == "completed" || snapshot.Status == "failed" || snapshot.Status == "cancelled" {
			if snapshot.CompletedAt != nil && now.Sub(*snapshot.CompletedAt) > maxAge {
				delete(sb.operations, id)
				sb.logger.Info("cleaned up old operation",
					slog.String("operation_id", id),
					slog.String("status", snapshot.Status),
					slog.Duration("age", now.Sub(*snapshot.CompletedAt)),
				)
			}
		}
	}
}

// Stop gracefully shuts down the broadcaster
func (sb *StatusBroadcaster) Stop() {
	close(sb.stop)
}
