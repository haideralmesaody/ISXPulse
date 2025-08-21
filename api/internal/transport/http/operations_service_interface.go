package http

import (
	"context"
	"isxcli/internal/operations"
)

// OperationServiceInterface defines the interface for operations service
type OperationServiceInterface interface {
	ExecuteOperation(ctx context.Context, request *operations.OperationRequest) (*operations.OperationResponse, error)
	GetOperationStatus(ctx context.Context, operationID string) (*operations.OperationState, error)
	CancelOperation(ctx context.Context, operationID string) error
	ListOperations(ctx context.Context) ([]*operations.OperationState, error)
	ListOperationsByStatus(ctx context.Context, status operations.OperationStatusValue) ([]*operations.OperationState, error)
	GetOperationMetrics(ctx context.Context) (map[string]interface{}, error)
	GetOperationTypes(ctx context.Context) ([]operations.OperationType, error)
}