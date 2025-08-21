package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	licenseErrors "isxcli/internal/errors"
	"isxcli/internal/infrastructure"
	"isxcli/internal/middleware"
	"isxcli/internal/operations"
)

// Hub interface defines WebSocket hub operations
type Hub interface {
	BroadcastUpdate(updateType, subtype, action string, data interface{})
}

// OperationsHandler handles operation-related HTTP requests
type OperationsHandler struct {
	service  OperationServiceInterface
	wsHub    Hub
	logger   *slog.Logger
	metrics  *infrastructure.BusinessMetrics
	jobQueue *operations.JobQueue
}

// NewOperationsHandler creates a new operations handler
func NewOperationsHandler(service OperationServiceInterface, wsHub Hub, logger *slog.Logger) *OperationsHandler {
	if service == nil {
		panic("service cannot be nil")
	}
	if wsHub == nil {
		panic("wsHub cannot be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &OperationsHandler{
		service:  service,
		wsHub:    wsHub,
		logger:   logger.With(slog.String("handler", "operations")),
		metrics:  nil, // Will be set via SetMetrics method
		jobQueue: nil, // Will be set via SetJobQueue method
	}
}

// SetMetrics sets the business metrics for the handler
func (h *OperationsHandler) SetMetrics(metrics *infrastructure.BusinessMetrics) {
	h.metrics = metrics
}

// SetJobQueue sets the job queue for async operations
func (h *OperationsHandler) SetJobQueue(jobQueue *operations.JobQueue) {
	h.jobQueue = jobQueue
}

// OperationRequest represents the request to start a new operation
type OperationRequest struct {
	Mode       string                   `json:"mode" validate:"required,oneof=full partial resume"`
	Steps      []StepConfig            `json:"steps" validate:"required,min=1,dive"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Timeout    string                 `json:"timeout,omitempty"`
}

// StepConfig represents configuration for a single step
type StepConfig struct {
	ID           string                 `json:"id" validate:"required"`
	Type         string                 `json:"type" validate:"required"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Timeout      string                 `json:"timeout,omitempty"`
	Retries      int                    `json:"retries,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// Bind implements the render.Binder interface for request validation
func (r *OperationRequest) Bind(req *http.Request) error {
	// Validate mode
	if r.Mode == "" {
		return errors.New("mode is required")
	}
	
	validModes := map[string]bool{
		"full":    true,
		"partial": true,
		"resume":  true,
	}
	
	if !validModes[r.Mode] {
		return fmt.Errorf("invalid mode: %s", r.Mode)
	}
	
	// Validate steps
	if len(r.Steps) == 0 {
		return errors.New("at least one step is required")
	}
	
	// Validate each step
	stepIDs := make(map[string]bool)
	for i, step := range r.Steps {
		if step.ID == "" {
			return fmt.Errorf("step[%d]: step ID is required", i)
		}
		if step.Type == "" {
			return fmt.Errorf("step[%d]: step type is required", i)
		}
		
		// Check for duplicate IDs
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
		
		// Validate timeout format if provided
		if step.Timeout != "" {
			if _, err := time.ParseDuration(step.Timeout); err != nil {
				return fmt.Errorf("step[%d]: invalid timeout format: %s", i, step.Timeout)
			}
		}
		
		// Validate dependencies exist
		for _, dep := range step.Dependencies {
			if dep == step.ID {
				return fmt.Errorf("step[%d]: circular dependency - step cannot depend on itself", i)
			}
		}
	}
	
	// Check for circular dependencies
	if err := validateDependencies(r.Steps); err != nil {
		return err
	}
	
	// Validate operation timeout if provided
	if r.Timeout != "" {
		if _, err := time.ParseDuration(r.Timeout); err != nil {
			return fmt.Errorf("invalid operation timeout format: %s", r.Timeout)
		}
	}
	
	return nil
}

// Routes returns a chi router for operations endpoints
func (h *OperationsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	
	// Apply timeout middleware to all operations routes
	r.Use(middleware.Timeout(60*time.Second, h.logger))
	
	// Operations endpoints
	r.Get("/types", h.GetOperationTypes)
	r.Post("/start", h.StartOperation)
	r.Post("/{id}/stop", h.StopOperation)
	r.Get("/{id}/status", h.GetOperationStatus)
	r.Get("/", h.ListOperations)
	r.Delete("/{id}", h.DeleteOperation)
	
	// Async job endpoints
	r.Get("/jobs/{id}", h.GetJobStatus)
	r.Get("/jobs", h.ListJobs)
	
	return r
}

// StartOperation handles POST /api/operations/start
func (h *OperationsHandler) StartOperation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.start_operation",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations/start"),
			attribute.String("request_id", reqID),
			attribute.String("component", "operations_handler"),
		),
	)
	defer span.End()
	
	// Log request start
	h.logger.InfoContext(ctx, "operation start request",
		slog.String("request_id", reqID),
		slog.String("trace_id", infrastructure.TraceIDFromContext(ctx)),
		slog.String("operation", "start_operation"),
	)
	
	// Decode and validate request
	data := &OperationRequest{}
	if err := render.Bind(r, data); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "request_validation"))
		
		h.logger.ErrorContext(ctx, "failed to bind operation request",
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		problem := licenseErrors.NewProblemDetails(
			http.StatusBadRequest,
			"/errors/validation_failed",
			"validation_failed",
			err.Error(),
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx))
		
		render.Render(w, r, problem)
		return
	}
	
	// Create operation request with unique ID
	// Use UUID as fallback if request ID is empty
	operationID := reqID
	if operationID == "" {
		operationID = uuid.New().String()
		h.logger.WarnContext(ctx, "RequestID middleware returned empty, using UUID fallback",
			slog.String("generated_id", operationID))
	}
	
	request := &operations.OperationRequest{
		ID:         operationID,
		Mode:       data.Mode,
		Parameters: make(map[string]interface{}),
	}
	
	// If parameters are provided, use them
	if data.Parameters != nil {
		request.Parameters = data.Parameters
	}
	
	// If steps are specified, determine which operation to run
	if len(data.Steps) > 0 {
		// If there's only one step, use its parameters and add step info
		if len(data.Steps) == 1 {
			step := data.Steps[0]
			request.Parameters["step"] = step.ID
			// Merge step parameters with request parameters
			for k, v := range step.Parameters {
				request.Parameters[k] = v
			}
			
			// Debug logging for single step
			h.logger.DebugContext(ctx, "single step operation parameters",
				slog.String("step_id", step.ID),
				slog.Any("step_parameters", step.Parameters),
				slog.Any("merged_parameters", request.Parameters))
		} else {
			// Multiple steps means full pipeline
			request.Parameters["step"] = "full_pipeline"
			// Use parameters from the first step (usually scraping)
			if len(data.Steps) > 0 && data.Steps[0].Parameters != nil {
				for k, v := range data.Steps[0].Parameters {
					request.Parameters[k] = v
				}
			}
			
			// Debug logging for pipeline
			h.logger.DebugContext(ctx, "full pipeline operation parameters",
				slog.Int("steps_count", len(data.Steps)),
				slog.Any("first_step_parameters", data.Steps[0].Parameters),
				slog.Any("merged_parameters", request.Parameters))
		}
		
		// Extract dates from step parameters to set at root level
		// This ensures dates are properly passed to the operations service
		if len(data.Steps) > 0 && data.Steps[0].Parameters != nil {
			if fromDate, ok := data.Steps[0].Parameters["from"].(string); ok && fromDate != "" {
				request.FromDate = fromDate
				h.logger.InfoContext(ctx, "Extracted from_date from step parameters",
					slog.String("from_date", fromDate),
					slog.String("operation_id", request.ID))
			}
			if toDate, ok := data.Steps[0].Parameters["to"].(string); ok && toDate != "" {
				request.ToDate = toDate
				h.logger.InfoContext(ctx, "Extracted to_date from step parameters",
					slog.String("to_date", toDate),
					slog.String("operation_id", request.ID))
			}
		}
		
		// Also check request parameters for dates (fallback)
		if request.FromDate == "" {
			if fromDate, ok := request.Parameters["from"].(string); ok && fromDate != "" {
				request.FromDate = fromDate
			}
		}
		if request.ToDate == "" {
			if toDate, ok := request.Parameters["to"].(string); ok && toDate != "" {
				request.ToDate = toDate
			}
		}
		
		// Log final date values
		h.logger.InfoContext(ctx, "Final operation request dates",
			slog.String("from_date", request.FromDate),
			slog.String("to_date", request.ToDate),
			slog.String("operation_id", request.ID))
	}
	
	// Add span attributes
	span.SetAttributes(
		attribute.String("operation.id", request.ID),
		attribute.String("operation.mode", request.Mode),
		attribute.Int("operation.steps_count", len(data.Steps)),
	)
	
	// Check if async job queue is available
	if h.jobQueue != nil {
		// Create job for async execution
		job := &operations.Job{
			ID:          request.ID,
			OperationID: request.ID,
			StageID:     "", // Will be set based on steps
			StageName:   "Operation",
			Status:      operations.JobStatusPending,
			Progress:    0,
			CreatedAt:   time.Now(),
			Request:     request,
			Metadata: map[string]interface{}{
				"trace_id":    infrastructure.TraceIDFromContext(ctx),
				"request_id":  reqID,
				"mode":        request.Mode,
				"steps_count": len(data.Steps),
			},
		}
		
		// Determine stage ID from steps
		if len(data.Steps) == 1 {
			job.StageID = data.Steps[0].ID
			job.StageName = data.Steps[0].Type
		} else if len(data.Steps) > 1 {
			job.StageID = "full_pipeline"
			job.StageName = "Full Pipeline"
		}
		
		// Enqueue job
		if err := h.jobQueue.Enqueue(job); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "job enqueue failed")
			
			h.logger.ErrorContext(ctx, "failed to enqueue job",
				slog.String("job_id", job.ID),
				slog.String("error", err.Error()),
				slog.String("request_id", reqID))
			
			problem := licenseErrors.NewProblemDetails(
				http.StatusServiceUnavailable,
				"/errors/queue_full",
				"queue_full",
				"Operation queue is full. Please try again later.",
				r.URL.Path+"#"+reqID,
			).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx)).
				WithExtension("operation_id", request.ID)
			
			render.Render(w, r, problem)
			return
		}
		
		// Log successful enqueue
		h.logger.InfoContext(ctx, "operation job enqueued",
			slog.String("job_id", job.ID),
			slog.String("operation_id", request.ID),
			slog.String("stage_id", job.StageID),
			slog.String("request_id", reqID))
		
		// Send WebSocket notification
		h.wsHub.BroadcastUpdate("operation_update", "queued", "pending", map[string]interface{}{
			"job_id":       job.ID,
			"operation_id": request.ID,
			"mode":        request.Mode,
			"steps_count": len(data.Steps),
			"timestamp":   time.Now().UTC(),
		})
		
		// Return 202 Accepted with job ID
		response := map[string]interface{}{
			"job_id":      job.ID,
			"operation_id": request.ID,
			"status":      "pending",
			"message":     "Operation queued for processing",
			"poll_url":    "/api/operations/jobs/" + job.ID,
		}
		
		render.Status(r, http.StatusAccepted)
		render.JSON(w, r, response)
		return
	}
	
	// Fallback to synchronous execution if job queue not available
	h.logger.WarnContext(ctx, "job queue not available, falling back to synchronous execution",
		slog.String("operation_id", request.ID))
	
	startCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	h.logger.DebugContext(ctx, "executing operation synchronously",
		slog.String("operation_id", request.ID),
		slog.String("mode", request.Mode),
		slog.Int("steps_count", len(data.Steps)))
	
	// Record active operation increase
	if h.metrics != nil {
		infrastructure.RecordActiveOperationChange(ctx, h.metrics, 1, request.Mode)
		defer infrastructure.RecordActiveOperationChange(ctx, h.metrics, -1, request.Mode)
	}
	
	executionStart := time.Now()
	result, err := h.service.ExecuteOperation(startCtx, request)
	executionDuration := time.Since(executionStart)
	
	// Record operation metrics
	if h.metrics != nil {
		infrastructure.RecordOperationMetrics(ctx, h.metrics, request.ID, request.Mode, executionDuration, err == nil, err)
	}
	
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "operation execution failed")
		
		h.logger.ErrorContext(ctx, "operation execution failed",
			slog.String("operation_id", request.ID),
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		problem := licenseErrors.NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/operation_failed",
			"operation_failed",
			"Failed to execute operation: " + err.Error(),
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx)).
			WithExtension("operation_id", request.ID)
		
		render.Render(w, r, problem)
		return
	}
	
	// Success response for synchronous execution
	span.SetAttributes(
		attribute.Bool("operation.success", result.Status == operations.OperationStatusCompleted),
		attribute.Float64("operation.duration_ms", float64(result.Duration.Milliseconds())),
	)
	
	h.logger.InfoContext(ctx, "operation completed synchronously",
		slog.String("operation_id", request.ID),
		slog.Bool("success", result.Status == operations.OperationStatusCompleted),
		slog.Duration("duration", result.Duration),
		slog.String("request_id", reqID))
	
	// Send WebSocket notification
	h.wsHub.BroadcastUpdate("operation_update", "completed", "completed", map[string]interface{}{
		"operation_id": request.ID,
		"mode":        request.Mode,
		"steps_count": len(data.Steps),
		"timestamp":   time.Now().UTC(),
	})
	
	// Return result with operation ID
	response := map[string]interface{}{
		"id":      request.ID,
		"success": result.Status == operations.OperationStatusCompleted,
		"steps":   result.Steps,
	}
	
	if result.Error != "" {
		response["error"] = result.Error
	}
	
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// StopOperation handles POST /api/operations/{id}/stop
func (h *OperationsHandler) StopOperation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	operationID := chi.URLParam(r, "id")
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.stop_operation",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations/{id}/stop"),
			attribute.String("operation.id", operationID),
			attribute.String("request_id", reqID),
		),
	)
	defer span.End()
	
	h.logger.InfoContext(ctx, "operation stop request",
		slog.String("operation_id", operationID),
		slog.String("request_id", reqID),
		slog.String("trace_id", infrastructure.TraceIDFromContext(ctx)))
	
	// Cancel operation
	cancelCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	cancelStart := time.Now()
	err := h.service.CancelOperation(cancelCtx, operationID)
	cancelDuration := time.Since(cancelStart)
	
	// Record cancellation metric on success
	if err == nil && h.metrics != nil {
		infrastructure.RecordOperationCancellation(ctx, h.metrics, operationID, "unknown", "user_requested")
	}
	
	// Add cancellation duration to span
	span.SetAttributes(
		attribute.Float64("cancellation.duration_ms", float64(cancelDuration.Milliseconds())),
	)
	
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "operation cancellation failed")
		
		h.logger.ErrorContext(ctx, "failed to cancel operation",
			slog.String("operation_id", operationID),
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		// Check specific error types
		if errors.Is(err, operations.ErrOperationNotFound) {
			problem := licenseErrors.NewProblemDetails(
				http.StatusNotFound,
				"/errors/not_found",
				"not_found",
				"Operation not found",
				r.URL.Path+"#"+reqID,
			).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx)).
				WithExtension("operation_id", operationID)
			
			render.Render(w, r, problem)
			return
		}
		
		if errors.Is(err, operations.ErrOperationCompleted) {
			problem := licenseErrors.NewProblemDetails(
				http.StatusConflict,
				"/errors/invalid_state",
				"invalid_state",
				"Operation has already completed and cannot be cancelled",
				r.URL.Path+"#"+reqID,
			).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx)).
				WithExtension("operation_id", operationID)
			
			render.Render(w, r, problem)
			return
		}
		
		// Generic error
		problem := licenseErrors.NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/cancellation_failed",
			"cancellation_failed",
			"Failed to cancel operation",
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx)).
			WithExtension("operation_id", operationID)
		
		render.Render(w, r, problem)
		return
	}
	
	// Success
	h.logger.InfoContext(ctx, "operation cancelled successfully",
		slog.String("operation_id", operationID),
		slog.String("request_id", reqID))
	
	// Send WebSocket notification
	h.wsHub.BroadcastUpdate("operation_update", "cancelled", "cancelled", map[string]interface{}{
		"operation_id": operationID,
		"timestamp":    time.Now().UTC(),
	})
	
	render.JSON(w, r, map[string]string{
		"message": "Operation cancelled successfully",
	})
}

// GetOperationStatus handles GET /api/operations/{id}/status
func (h *OperationsHandler) GetOperationStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	operationID := chi.URLParam(r, "id")
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.get_status",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations/{id}/status"),
			attribute.String("operation.id", operationID),
			attribute.String("request_id", reqID),
		),
	)
	defer span.End()
	
	h.logger.DebugContext(ctx, "operation status request",
		slog.String("operation_id", operationID),
		slog.String("request_id", reqID))
	
	// Get status
	statusCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	status, err := h.service.GetOperationStatus(statusCtx, operationID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "status retrieval failed")
		
		h.logger.ErrorContext(ctx, "failed to get operation status",
			slog.String("operation_id", operationID),
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		h.handleError(w, r, err, map[string]interface{}{
			"operation_id": operationID,
		})
		return
	}
	
	// Add span attributes
	span.SetAttributes(
		attribute.String("operation.status", string(status.Status)),
		attribute.Int("operation.steps_count", len(status.Steps)),
	)
	
	// Convert to response format
	response := map[string]interface{}{
		"id":         status.ID,
		"status":     status.Status,
		"start_time": status.StartTime,
		"steps":      status.Steps,
	}
	
	if status.EndTime != nil {
		response["end_time"] = status.EndTime
		response["duration"] = status.Duration().String()
	}
	
	if status.Error != nil {
		response["error"] = status.Error.Error()
	}
	
	render.JSON(w, r, response)
}

// ListOperations handles GET /api/operations
func (h *OperationsHandler) ListOperations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.list_operations",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations"),
			attribute.String("request_id", reqID),
		),
	)
	defer span.End()
	
	// Check for status filter
	statusFilter := r.URL.Query().Get("status")
	
	h.logger.DebugContext(ctx, "listing operations",
		slog.String("status_filter", statusFilter),
		slog.String("request_id", reqID))
	
	// List operations
	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	var operationsList []*operations.OperationState
	var err error
	
	if statusFilter != "" {
		// Validate status filter
		validStatuses := map[string]operations.OperationStatusValue{
			"pending":   operations.OperationStatusPending,
			"running":   operations.OperationStatusRunning,
			"completed": operations.OperationStatusCompleted,
			"failed":    operations.OperationStatusFailed,
			"cancelled": operations.OperationStatusCancelled,
		}
		
		status, ok := validStatuses[statusFilter]
		if !ok {
			problem := licenseErrors.NewProblemDetails(
				http.StatusBadRequest,
				"/errors/validation_failed",
				"validation_failed",
				fmt.Sprintf("Invalid status filter: %s", statusFilter),
				r.URL.Path+"#"+reqID,
			).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx)).
				WithExtension("valid_statuses", []string{"pending", "running", "completed", "failed", "cancelled"})
			
			render.Render(w, r, problem)
			return
		}
		
		operationsList, err = h.service.ListOperationsByStatus(listCtx, status)
		span.SetAttributes(attribute.String("filter.status", statusFilter))
	} else {
		operationsList, err = h.service.ListOperations(listCtx)
	}
	
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list operations failed")
		
		h.logger.ErrorContext(ctx, "failed to list operations",
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		problem := licenseErrors.NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/list_failed",
			"list_failed",
			"Failed to list operations",
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx))
		
		render.Render(w, r, problem)
		return
	}
	
	// Add span attributes
	span.SetAttributes(attribute.Int("operations.count", len(operationsList)))
	
	// Convert to response format
	operations := make([]map[string]interface{}, len(operationsList))
	for i, op := range operationsList {
		operations[i] = map[string]interface{}{
			"id":         op.ID,
			"status":     op.Status,
			"start_time": op.StartTime,
		}
		
		if op.EndTime != nil {
			operations[i]["end_time"] = op.EndTime
			operations[i]["duration"] = op.Duration().String()
		}
		
		if op.Error != nil {
			operations[i]["error"] = op.Error.Error()
		}
		
		// Include step count
		operations[i]["steps_count"] = len(op.Steps)
	}
	
	render.JSON(w, r, operations)
}

// DeleteOperation handles DELETE /api/operations/{id}
func (h *OperationsHandler) DeleteOperation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	operationID := chi.URLParam(r, "id")
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.delete_operation",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations/{id}"),
			attribute.String("operation.id", operationID),
			attribute.String("request_id", reqID),
		),
	)
	defer span.End()
	
	h.logger.InfoContext(ctx, "operation delete request",
		slog.String("operation_id", operationID),
		slog.String("request_id", reqID))
	
	// For now, we don't actually delete operations from memory
	// This could be implemented to remove completed operations
	
	// Check if operation exists
	statusCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	_, err := h.service.GetOperationStatus(statusCtx, operationID)
	if err != nil {
		h.handleError(w, r, err, map[string]interface{}{
			"operation_id": operationID,
		})
		return
	}
	
	// Success (no actual deletion for now)
	h.logger.InfoContext(ctx, "operation deletion acknowledged",
		slog.String("operation_id", operationID),
		slog.String("request_id", reqID))
	
	w.WriteHeader(http.StatusNoContent)
}

// GetOperationTypes handles GET /api/operations/types
func (h *OperationsHandler) GetOperationTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.get_operation_types",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations/types"),
			attribute.String("request_id", reqID),
		),
	)
	defer span.End()
	
	h.logger.DebugContext(ctx, "getting operation types",
		slog.String("request_id", reqID))
	
	// Get operation types
	typesCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	types, err := h.service.GetOperationTypes(typesCtx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get operation types failed")
		
		h.logger.ErrorContext(ctx, "failed to get operation types",
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		problem := licenseErrors.NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/internal_error",
			"internal_error",
			"Failed to retrieve operation types",
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx))
		
		render.Render(w, r, problem)
		return
	}
	
	// Add span attributes
	span.SetAttributes(attribute.Int("operation_types.count", len(types)))
	
	h.logger.InfoContext(ctx, "operation types retrieved",
		slog.Int("count", len(types)),
		slog.String("request_id", reqID))
	
	render.JSON(w, r, types)
}

// handleError centralizes error handling for the handler
func (h *OperationsHandler) handleError(w http.ResponseWriter, r *http.Request, err error, extensions map[string]interface{}) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	traceID := infrastructure.TraceIDFromContext(ctx)
	
	// Log error
	h.logger.ErrorContext(ctx, "request failed",
		slog.String("error", err.Error()),
		slog.String("request_id", reqID),
		slog.String("trace_id", traceID),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method))
	
	// Determine status code and error type
	var problem *licenseErrors.ProblemDetails
	
	switch {
	case errors.Is(err, operations.ErrOperationNotFound):
		problem = licenseErrors.NewProblemDetails(
			http.StatusNotFound,
			"/errors/not_found",
			"not_found",
			"Operation not found",
			r.URL.Path+"#"+reqID,
		)
		
	case errors.Is(err, operations.ErrOperationCompleted):
		problem = licenseErrors.NewProblemDetails(
			http.StatusConflict,
			"/errors/invalid_state",
			"invalid_state",
			"Operation has already completed and cannot be cancelled",
			r.URL.Path+"#"+reqID,
		)
		
	case errors.Is(err, context.DeadlineExceeded):
		problem = licenseErrors.NewProblemDetails(
			http.StatusGatewayTimeout,
			"/errors/timeout",
			"Request Timeout",
			"The request timed out while processing",
			r.URL.Path+"#"+reqID,
		)
		
	case errors.Is(err, context.Canceled):
		problem = licenseErrors.NewProblemDetails(
			http.StatusRequestTimeout,
			"/errors/request_canceled",
			"Request Canceled",
			"The request was canceled",
			r.URL.Path+"#"+reqID,
		)
		
	default:
		problem = licenseErrors.NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/internal_error",
			"Internal Server Error",
			"An unexpected error occurred",
			r.URL.Path+"#"+reqID,
		)
	}
	
	// Add standard extensions
	problem.WithExtension("trace_id", traceID).
		WithExtension("timestamp", time.Now().UTC()).
		WithExtension("request_id", reqID)
		
	// Add custom extensions
	if extensions != nil {
		for k, v := range extensions {
			problem.WithExtension(k, v)
		}
	}
	
	render.Render(w, r, problem)
}

// Helper function to validate step dependencies
func validateDependencies(steps []StepConfig) error {
	// Build dependency graph
	deps := make(map[string][]string)
	stepExists := make(map[string]bool)
	
	for _, step := range steps {
		stepExists[step.ID] = true
		deps[step.ID] = step.Dependencies
	}
	
	// Check all dependencies exist
	for stepID, stepDeps := range deps {
		for _, dep := range stepDeps {
			if !stepExists[dep] {
				return fmt.Errorf("step %s depends on non-existent step %s", stepID, dep)
			}
		}
	}
	
	// Check for circular dependencies using DFS
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)
	
	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recursionStack[node] = true
		
		for _, dep := range deps[node] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recursionStack[dep] {
				return true
			}
		}
		
		recursionStack[node] = false
		return false
	}
	
	for stepID := range deps {
		if !visited[stepID] {
			if hasCycle(stepID) {
				return fmt.Errorf("circular dependency detected in steps")
			}
		}
	}
	
	return nil
}

// GetJobStatus handles GET /api/operations/jobs/{id}
func (h *OperationsHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := chi.URLParam(r, "id")
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.get_job_status",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations/jobs/{id}"),
			attribute.String("job.id", jobID),
			attribute.String("request_id", reqID),
		),
	)
	defer span.End()
	
	h.logger.DebugContext(ctx, "job status request",
		slog.String("job_id", jobID),
		slog.String("request_id", reqID))
	
	// Check if job queue is available
	if h.jobQueue == nil {
		problem := licenseErrors.NewProblemDetails(
			http.StatusServiceUnavailable,
			"/errors/service_unavailable",
			"service_unavailable",
			"Job queue service is not available",
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx))
		
		render.Render(w, r, problem)
		return
	}
	
	// Get job status
	job, err := h.jobQueue.GetJob(jobID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "job retrieval failed")
		
		h.logger.ErrorContext(ctx, "failed to get job status",
			slog.String("job_id", jobID),
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		problem := licenseErrors.NewProblemDetails(
			http.StatusNotFound,
			"/errors/not_found",
			"not_found",
			"Job not found",
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx)).
			WithExtension("job_id", jobID)
		
		render.Render(w, r, problem)
		return
	}
	
	// Add span attributes
	span.SetAttributes(
		attribute.String("job.status", string(job.Status)),
		attribute.Int("job.progress", job.Progress),
	)
	
	// Build response
	response := map[string]interface{}{
		"job_id":       job.ID,
		"operation_id": job.OperationID,
		"stage_id":     job.StageID,
		"stage_name":   job.StageName,
		"status":       job.Status,
		"progress":     job.Progress,
		"created_at":   job.CreatedAt,
	}
	
	if job.StartedAt != nil {
		response["started_at"] = job.StartedAt
	}
	
	if job.CompletedAt != nil {
		response["completed_at"] = job.CompletedAt
		
		// Calculate duration
		if job.StartedAt != nil {
			duration := job.CompletedAt.Sub(*job.StartedAt)
			response["duration"] = duration.String()
		}
	}
	
	if job.Message != "" {
		response["message"] = job.Message
	}
	
	if job.Error != "" {
		response["error"] = job.Error
	}
	
	if job.Metadata != nil {
		response["metadata"] = job.Metadata
	}
	
	// Add polling hints
	switch job.Status {
	case operations.JobStatusPending, operations.JobStatusRunning:
		response["poll_after"] = "2s" // Suggest polling interval
		response["is_complete"] = false
	case operations.JobStatusCompleted, operations.JobStatusFailed, operations.JobStatusCancelled:
		response["is_complete"] = true
	}
	
	render.JSON(w, r, response)
}

// ListJobs handles GET /api/operations/jobs
func (h *OperationsHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetReqID(ctx)
	tracer := otel.Tracer("operations-handler")
	
	// Start OpenTelemetry span
	ctx, span := tracer.Start(ctx, "operations_handler.list_jobs",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/api/operations/jobs"),
			attribute.String("request_id", reqID),
		),
	)
	defer span.End()
	
	// Check if job queue is available
	if h.jobQueue == nil {
		problem := licenseErrors.NewProblemDetails(
			http.StatusServiceUnavailable,
			"/errors/service_unavailable",
			"service_unavailable",
			"Job queue service is not available",
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx))
		
		render.Render(w, r, problem)
		return
	}
	
	// Parse query parameters
	filter := operations.JobFilter{}
	
	// Status filter
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = operations.JobStatus(status)
		span.SetAttributes(attribute.String("filter.status", status))
	}
	
	// Operation ID filter
	if opID := r.URL.Query().Get("operation_id"); opID != "" {
		filter.OperationID = opID
		span.SetAttributes(attribute.String("filter.operation_id", opID))
	}
	
	// Stage ID filter
	if stageID := r.URL.Query().Get("stage_id"); stageID != "" {
		filter.StageID = stageID
		span.SetAttributes(attribute.String("filter.stage_id", stageID))
	}
	
	// Limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
			span.SetAttributes(attribute.Int("filter.limit", limit))
		}
	}
	
	h.logger.DebugContext(ctx, "listing jobs",
		slog.String("status_filter", string(filter.Status)),
		slog.String("operation_filter", filter.OperationID),
		slog.String("stage_filter", filter.StageID),
		slog.Int("limit", filter.Limit),
		slog.String("request_id", reqID))
	
	// List jobs
	jobs, err := h.jobQueue.ListJobs(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list jobs failed")
		
		h.logger.ErrorContext(ctx, "failed to list jobs",
			slog.String("error", err.Error()),
			slog.String("request_id", reqID))
		
		problem := licenseErrors.NewProblemDetails(
			http.StatusInternalServerError,
			"/errors/list_failed",
			"list_failed",
			"Failed to list jobs",
			r.URL.Path+"#"+reqID,
		).WithExtension("trace_id", infrastructure.TraceIDFromContext(ctx))
		
		render.Render(w, r, problem)
		return
	}
	
	// Add span attributes
	span.SetAttributes(attribute.Int("jobs.count", len(jobs)))
	
	// Convert to response format
	jobList := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		jobData := map[string]interface{}{
			"job_id":       job.ID,
			"operation_id": job.OperationID,
			"stage_id":     job.StageID,
			"stage_name":   job.StageName,
			"status":       job.Status,
			"progress":     job.Progress,
			"created_at":   job.CreatedAt,
		}
		
		if job.StartedAt != nil {
			jobData["started_at"] = job.StartedAt
		}
		
		if job.CompletedAt != nil {
			jobData["completed_at"] = job.CompletedAt
			
			// Calculate duration
			if job.StartedAt != nil {
				duration := job.CompletedAt.Sub(*job.StartedAt)
				jobData["duration"] = duration.String()
			}
		}
		
		if job.Message != "" {
			jobData["message"] = job.Message
		}
		
		if job.Error != "" {
			jobData["error"] = job.Error
		}
		
		jobList[i] = jobData
	}
	
	// Get queue stats
	stats := h.jobQueue.GetQueueStats()
	
	response := map[string]interface{}{
		"jobs":  jobList,
		"count": len(jobList),
		"stats": stats,
	}
	
	render.JSON(w, r, response)
}