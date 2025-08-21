package operations

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents an async operation job
type Job struct {
	ID          string                 `json:"id"`
	OperationID string                 `json:"operation_id"`
	StageID     string                 `json:"stage_id"`
	StageName   string                 `json:"stage_name"`
	Status      JobStatus              `json:"status"`
	Progress    int                    `json:"progress"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Request     *OperationRequest      `json:"request,omitempty"`
}

// JobStore interface for job persistence
type JobStore interface {
	// Job operations
	CreateJob(job *Job) error
	GetJob(id string) (*Job, error)
	UpdateJob(job *Job) error
	ListJobs(filter JobFilter) ([]*Job, error)
	DeleteJob(id string) error
	
	// Manifest operations
	CreateManifest(manifest *PipelineManifest) error
	GetManifest(id string) (*PipelineManifest, error)
	UpdateManifest(manifest *PipelineManifest) error
	GetManifestByOperationID(operationID string) (*PipelineManifest, error)
}

// JobFilter for querying jobs
type JobFilter struct {
	Status      JobStatus
	OperationID string
	StageID     string
	Since       time.Time
	Limit       int
}

// JobQueue manages async job execution
type JobQueue struct {
	mu       sync.RWMutex
	jobs     chan *Job
	workers  int
	wg       sync.WaitGroup
	store    JobStore
	manager  *Manager
	logger   *slog.Logger
	shutdown chan struct{}
	active   map[string]*Job // Currently executing jobs
}

// NewJobQueue creates a new job queue
func NewJobQueue(workers int, store JobStore, manager *Manager, logger *slog.Logger) *JobQueue {
	if workers <= 0 {
		workers = 4 // Default number of workers
	}
	
	if logger == nil {
		logger = slog.Default()
	}
	
	return &JobQueue{
		jobs:     make(chan *Job, workers*2), // Buffer size = 2x workers
		workers:  workers,
		store:    store,
		manager:  manager,
		logger:   logger.With(slog.String("component", "jobqueue")),
		shutdown: make(chan struct{}),
		active:   make(map[string]*Job),
	}
}

// Start begins processing jobs
func (q *JobQueue) Start(ctx context.Context) {
	q.logger.Info("starting job queue", slog.Int("workers", q.workers))
	
	// Start worker goroutines
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i)
	}
	
	// Start job recovery (for jobs that were running when system stopped)
	go q.recoverJobs(ctx)
}

// Stop gracefully shuts down the job queue
func (q *JobQueue) Stop(timeout time.Duration) error {
	q.logger.Info("stopping job queue")
	
	// Signal shutdown
	close(q.shutdown)
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		q.logger.Info("job queue stopped gracefully")
		return nil
	case <-time.After(timeout):
		q.logger.Warn("job queue stop timeout exceeded")
		return fmt.Errorf("timeout waiting for workers to finish")
	}
}

// Enqueue adds a job to the queue
func (q *JobQueue) Enqueue(job *Job) error {
	// Set initial status
	job.Status = JobStatusPending
	job.CreatedAt = time.Now()
	
	// Save to store
	if err := q.store.CreateJob(job); err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}
	
	// Initialize operation in broadcaster
	broadcaster := q.manager.GetBroadcaster()
	// Determine stages based on job type
	var stages []string
	if job.StageID == "" || job.StageID == "full_pipeline" {
		stages = []string{"scraping", "processing", "indices", "liquidity"}
	} else {
		stages = []string{job.StageID}
	}
	broadcaster.CreateOperation(job.OperationID, stages)
	
	// Add to queue
	select {
	case q.jobs <- job:
		q.logger.Info("job enqueued",
			slog.String("job_id", job.ID),
			slog.String("stage_id", job.StageID))
		return nil
	default:
		// Queue is full, mark as failed
		job.Status = JobStatusFailed
		job.Error = "job queue is full"
		q.store.UpdateJob(job)
		return fmt.Errorf("job queue is full")
	}
}

// GetJob retrieves a job by ID
func (q *JobQueue) GetJob(id string) (*Job, error) {
	// Check if job is currently active
	q.mu.RLock()
	if activeJob, ok := q.active[id]; ok {
		q.mu.RUnlock()
		return activeJob, nil
	}
	q.mu.RUnlock()
	
	// Otherwise get from store
	return q.store.GetJob(id)
}

// CancelJob cancels a running job
func (q *JobQueue) CancelJob(id string) error {
	job, err := q.GetJob(id)
	if err != nil {
		return err
	}
	
	if job.Status != JobStatusRunning && job.Status != JobStatusPending {
		return fmt.Errorf("job %s cannot be cancelled (status: %s)", id, job.Status)
	}
	
	// Update status
	job.Status = JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now
	
	return q.store.UpdateJob(job)
}

// ListJobs returns jobs matching the filter
func (q *JobQueue) ListJobs(filter JobFilter) ([]*Job, error) {
	return q.store.ListJobs(filter)
}

// worker processes jobs from the queue
func (q *JobQueue) worker(ctx context.Context, workerID int) {
	defer q.wg.Done()
	
	logger := q.logger.With(slog.Int("worker_id", workerID))
	logger.Debug("worker started")
	
	for {
		select {
		case <-ctx.Done():
			logger.Debug("worker stopped by context")
			return
		case <-q.shutdown:
			logger.Debug("worker stopped by shutdown")
			return
		case job := <-q.jobs:
			q.processJob(ctx, job, logger)
		}
	}
}

// processJob executes a single job
func (q *JobQueue) processJob(ctx context.Context, job *Job, logger *slog.Logger) {
	// Add trace ID to context
	if job.Metadata != nil {
		if traceID, ok := job.Metadata["trace_id"].(string); ok {
			ctx = context.WithValue(ctx, middleware.RequestIDKey, traceID)
		}
	}
	
	logger = logger.With(
		slog.String("job_id", job.ID),
		slog.String("operation_id", job.OperationID),
		slog.String("stage_id", job.StageID),
	)
	
	logger.Info("processing job started")
	
	// Get the status broadcaster
	broadcaster := q.manager.GetBroadcaster()
	
	// Mark job as active
	q.mu.Lock()
	q.active[job.ID] = job
	q.mu.Unlock()
	
	defer func() {
		// Recover from any panics to prevent server crash
		if r := recover(); r != nil {
			logger.Error("job processing panicked",
				slog.Any("panic", r),
				slog.String("job_id", job.ID))
			
			// Mark job as failed
			job.Status = JobStatusFailed
			job.Error = fmt.Sprintf("job processing panicked: %v", r)
			job.Message = "Internal error occurred"
			completedAt := time.Now()
			job.CompletedAt = &completedAt
			
			if err := q.store.UpdateJob(job); err != nil {
				logger.Error("failed to update job after panic", slog.String("error", err.Error()))
			}
		}
		
		// Remove from active jobs
		q.mu.Lock()
		delete(q.active, job.ID)
		q.mu.Unlock()
	}()
	
	// Update job status to running
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	job.Progress = 0
	job.Message = "Job started"
	
	if err := q.store.UpdateJob(job); err != nil {
		logger.Error("failed to update job status", slog.String("error", err.Error()))
	}
	
	// Mark operation as started through broadcaster
	broadcaster.StartOperation(job.OperationID)
	
	// Get or create manifest
	manifest, err := q.getOrCreateManifest(job)
	if err != nil {
		q.handleJobError(job, err, logger)
		return
	}
	
	// Check if we're running a single stage or full pipeline
	if job.StageID != "" && job.StageID != "full_pipeline" {
		// Single stage execution
		if err := q.executeSingleStage(ctx, job, manifest, logger); err != nil {
			q.handleJobError(job, err, logger)
			return
		}
	} else {
		// Full pipeline execution
		if err := q.executeFullPipeline(ctx, job, manifest, logger); err != nil {
			q.handleJobError(job, err, logger)
			return
		}
	}
	
	// Mark job as completed
	job.Status = JobStatusCompleted
	job.Progress = 100
	job.Message = "Job completed successfully"
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	
	if err := q.store.UpdateJob(job); err != nil {
		logger.Error("failed to update job completion", slog.String("error", err.Error()))
	}
	
	// Broadcast operation completion through the centralized broadcaster
	broadcaster.CompleteOperation(job.OperationID, "Operation completed successfully")
	
	logger.Info("processing job completed")
}

// executeSingleStage runs a single stage
func (q *JobQueue) executeSingleStage(ctx context.Context, job *Job, manifest *PipelineManifest, logger *slog.Logger) error {
	// Get the stage from registry using the exported method
	stage, err := q.manager.GetRegistry().Get(job.StageID)
	if err != nil {
		return fmt.Errorf("stage not found: %w", err)
	}
	
	// Check if stage can run
	logger.Debug("Checking if stage can run",
		slog.String("stage_id", job.StageID),
		slog.String("operation_id", job.OperationID),
		slog.String("request_id", middleware.GetReqID(ctx)))
	
	canRun := stage.CanRun(manifest)
	
	logger.Info("Stage CanRun check completed",
		slog.String("stage_id", job.StageID),
		slog.Bool("can_run", canRun),
		slog.String("request_id", middleware.GetReqID(ctx)))
	
	if !canRun {
		return fmt.Errorf("stage %s cannot run: required inputs not available", job.StageID)
	}
	
	// Update job progress
	job.Progress = 10
	job.Message = fmt.Sprintf("Starting %s", stage.Name())
	q.store.UpdateJob(job)
	
	// Update status through broadcaster
	broadcaster := q.manager.GetBroadcaster()
	broadcaster.UpdateStepProgress(job.OperationID, stage.ID(), 10, fmt.Sprintf("Starting %s", stage.Name()))
	
	// Record stage start in manifest
	manifest.RecordStageStart(stage.ID(), stage.Name())
	
	// Create operation state for the stage
	state := NewOperationState(job.OperationID)
	state.SetConfig(ContextKeyFromDate, manifest.FromDate)
	state.SetConfig(ContextKeyToDate, manifest.ToDate)
	
	// Initialize the stage state to prevent nil pointer dereference
	stepState := NewStepState(stage.ID(), stage.Name())
	state.SetStage(stage.ID(), stepState)
	
	// Execute the stage
	logger.Info("executing stage", slog.String("stage", stage.ID()))
	
	if err := stage.Execute(ctx, state); err != nil {
		manifest.RecordStageFailure(stage.ID(), err)
		q.store.UpdateManifest(manifest)
		// Mark step as failed through broadcaster
		broadcaster.FailStep(job.OperationID, stage.ID(), err)
		return fmt.Errorf("stage %s failed: %w", stage.ID(), err)
	}
	
	// Update manifest with stage outputs
	outputs := stage.ProducedOutputs()
	outputTypes := make([]string, len(outputs))
	for i, output := range outputs {
		outputTypes[i] = output.Type
		// Scan directory for produced files
		manifest.ScanDataDirectory(output.Type, output.Location, output.Pattern)
	}
	
	manifest.RecordStageCompletion(stage.ID(), outputTypes, nil)
	q.store.UpdateManifest(manifest)
	
	// Update job progress
	job.Progress = 90
	job.Message = fmt.Sprintf("Completed %s", stage.Name())
	q.store.UpdateJob(job)
	
	// Mark step as completed through broadcaster
	broadcaster.CompleteStep(job.OperationID, stage.ID(), fmt.Sprintf("Completed %s", stage.Name()))
	
	return nil
}

// executeFullPipeline runs all stages in sequence
func (q *JobQueue) executeFullPipeline(ctx context.Context, job *Job, manifest *PipelineManifest, logger *slog.Logger) error {
	// Get all stages in dependency order using the exported method
	stages, err := q.manager.GetRegistry().GetDependencyOrder()
	if err != nil {
		return fmt.Errorf("failed to get stage order: %w", err)
	}
	
	totalStages := len(stages)
	
	for i, stage := range stages {
		// Check if stage can run
		if !stage.CanRun(manifest) {
			logger.Info("skipping stage - requirements not met",
				slog.String("stage", stage.ID()))
			continue
		}
		
		// Update job progress
		progress := (i * 90) / totalStages
		job.Progress = progress
		job.Message = fmt.Sprintf("Running %s (%d/%d)", stage.Name(), i+1, totalStages)
		q.store.UpdateJob(job)
		
		// Execute stage (reuse single stage logic)
		tempJob := *job
		tempJob.StageID = stage.ID()
		tempJob.StageName = stage.Name()
		
		if err := q.executeSingleStage(ctx, &tempJob, manifest, logger); err != nil {
			return err
		}
	}
	
	return nil
}

// handleJobError handles job execution errors
func (q *JobQueue) handleJobError(job *Job, err error, logger *slog.Logger) {
	logger.Error("job failed", slog.String("error", err.Error()))
	
	job.Status = JobStatusFailed
	job.Error = err.Error()
	job.Message = "Job failed"
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	
	if err := q.store.UpdateJob(job); err != nil {
		logger.Error("failed to update job error", slog.String("error", err.Error()))
	}
	
	// Broadcast operation failure through the centralized broadcaster
	broadcaster := q.manager.GetBroadcaster()
	broadcaster.FailOperation(job.OperationID, err)
}

// getOrCreateManifest gets existing or creates new manifest
func (q *JobQueue) getOrCreateManifest(job *Job) (*PipelineManifest, error) {
	// Try to get existing manifest
	manifest, err := q.store.GetManifestByOperationID(job.OperationID)
	if err == nil && manifest != nil {
		return manifest, nil
	}
	
	// Create new manifest
	fromDate := ""
	toDate := ""
	
	if job.Request != nil {
		fromDate = job.Request.FromDate
		toDate = job.Request.ToDate
	}
	
	manifest = NewPipelineManifest(job.OperationID, fromDate, toDate)
	
	// Scan existing data directories to populate available data
	// This allows resuming operations that find existing data
	manifest.ScanDataDirectory("excel_files", "data/downloads", "*.xlsx")
	manifest.ScanDataDirectory("csv_files", "data/reports", "*.csv")
	manifest.ScanDataDirectory("index_data", "data/reports", "ISX*.csv")
	manifest.ScanDataDirectory("liquidity_results", "data/reports/liquidity_reports", "liquidity_*.csv")
	
	if err := q.store.CreateManifest(manifest); err != nil {
		return nil, fmt.Errorf("failed to create manifest: %w", err)
	}
	
	return manifest, nil
}

// recoverJobs recovers jobs that were running when the system stopped
func (q *JobQueue) recoverJobs(ctx context.Context) {
	q.logger.Info("recovering pending and running jobs")
	
	// Find jobs that were running or pending
	jobs, err := q.store.ListJobs(JobFilter{
		Status: JobStatusRunning,
	})
	if err != nil {
		q.logger.Error("failed to recover running jobs", slog.String("error", err.Error()))
		return
	}
	
	pendingJobs, err := q.store.ListJobs(JobFilter{
		Status: JobStatusPending,
	})
	if err != nil {
		q.logger.Error("failed to recover pending jobs", slog.String("error", err.Error()))
	} else {
		jobs = append(jobs, pendingJobs...)
	}
	
	// Re-queue recovered jobs
	for _, job := range jobs {
		// Reset running jobs to pending
		if job.Status == JobStatusRunning {
			job.Status = JobStatusPending
			job.StartedAt = nil
			job.Progress = 0
			q.store.UpdateJob(job)
		}
		
		// Re-enqueue
		select {
		case q.jobs <- job:
			q.logger.Info("recovered job",
				slog.String("job_id", job.ID),
				slog.String("status", string(job.Status)))
		default:
			q.logger.Warn("could not recover job - queue full",
				slog.String("job_id", job.ID))
		}
	}
}

// GetQueueStats returns queue statistics
func (q *JobQueue) GetQueueStats() map[string]interface{} {
	q.mu.RLock()
	activeCount := len(q.active)
	q.mu.RUnlock()
	
	return map[string]interface{}{
		"workers":      q.workers,
		"queue_size":   len(q.jobs),
		"queue_cap":    cap(q.jobs),
		"active_jobs":  activeCount,
	}
}