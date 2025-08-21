package operations

import (
	"context"
	"log/slog"
	"time"
)

// logOperationStart logs the start of a operation execution
func (m *Manager) logOperationStart(ctx context.Context, operationID string, req OperationRequest) {
	slog.InfoContext(ctx, "operation_start",
		slog.String("operation_id", operationID),
		slog.String("mode", req.Mode),
		slog.String("from_date", req.FromDate),
		slog.String("to_date", req.ToDate),
		slog.Any("parameters", req.Parameters))
}

// logOperationComplete logs the completion of a operation execution
func (m *Manager) logOperationComplete(ctx context.Context, operationID string, duration time.Duration, status string) {
	slog.InfoContext(ctx, "operation_complete",
		slog.String("operation_id", operationID),
		slog.String("status", status),
		slog.Duration("duration", duration))
}

// logOperationError logs a operation error
func (m *Manager) logOperationError(ctx context.Context, operationID string, err error) {
	errorMsg := "unknown error"
	if err != nil {
		errorMsg = err.Error()
	}
	slog.ErrorContext(ctx, "operation_error",
		slog.String("operation_id", operationID),
		slog.String("error", errorMsg))
}

// logStageStart logs the start of a Step execution
func (m *Manager) logStageStart(ctx context.Context, operationID, stageID string) {
	slog.InfoContext(ctx, "stage_start",
		slog.String("operation_id", operationID),
		slog.String("Step", stageID))
}

// logStageComplete logs the completion of a Step execution
func (m *Manager) logStageComplete(ctx context.Context, operationID, stageID string, duration time.Duration) {
	slog.InfoContext(ctx, "stage_complete",
		slog.String("operation_id", operationID),
		slog.String("Step", stageID),
		slog.Duration("duration", duration))
}

// logStageError logs a Step error
func (m *Manager) logStageError(ctx context.Context, operationID, stageID string, err error) {
	errorMsg := "unknown error"
	if err != nil {
		errorMsg = err.Error()
	}
	slog.ErrorContext(ctx, "stage_error",
		slog.String("operation_id", operationID),
		slog.String("Step", stageID),
		slog.String("error", errorMsg))
}

// logStageProgress logs Step progress
func (m *Manager) logStageProgress(ctx context.Context, operationID, stageID string, progress int, message string) {
	slog.DebugContext(ctx, "stage_progress",
		slog.String("operation_id", operationID),
		slog.String("Step", stageID),
		slog.Int("progress", progress),
		slog.String("message", message))
}