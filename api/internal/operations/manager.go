package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Manager orchestrates operation execution
type Manager struct {
	registry    *Registry
	config      *Config
	hub         WebSocketHub
	broadcaster *StatusBroadcaster

	// Active operations
	mu         sync.RWMutex
	operations map[string]*OperationState
}

// NewManager creates a new operation manager with dependency injection
func NewManager(hub WebSocketHub, registry *Registry, config *Config) *Manager {
	if registry == nil {
		registry = NewRegistry()
	}
	if config == nil {
		config = NewConfig()
	}

	// Create status broadcaster for centralized status management
	broadcaster := NewStatusBroadcaster(hub, slog.Default())

	return &Manager{
		registry:    registry,
		config:      config,
		hub:         hub,
		broadcaster: broadcaster,
		operations:  make(map[string]*OperationState),
	}
}

// RegisterStage registers a Step with the operation
func (m *Manager) RegisterStage(Step Step) error {
	return m.registry.Register(Step)
}

// SetConfig updates the operation configuration
func (m *Manager) SetConfig(config *Config) {
	if config != nil {
		m.config = config
	}
}

// GetRegistry returns the registry for accessing registered stages
func (m *Manager) GetRegistry() *Registry {
	return m.registry
}

// GetBroadcaster returns the status broadcaster for centralized status updates
func (m *Manager) GetBroadcaster() *StatusBroadcaster {
	return m.broadcaster
}

// Execute runs a operation with the given request
func (m *Manager) Execute(ctx context.Context, req OperationRequest) (*OperationResponse, error) {
	// Generate operation ID if not provided
	if req.ID == "" {
		req.ID = fmt.Sprintf("operation-%d", time.Now().Unix())
	}

	// Create operation state
	state := NewOperationState(req.ID)

	// Set configuration from request
	if req.FromDate != "" {
		state.SetConfig(ContextKeyFromDate, req.FromDate)
	}
	if req.ToDate != "" {
		state.SetConfig(ContextKeyToDate, req.ToDate)
	}
	if req.Mode != "" {
		state.SetConfig(ContextKeyMode, req.Mode)
	}

	// Copy additional parameters
	for k, v := range req.Parameters {
		state.SetConfig(k, v)
	}

	// Store operation state
	m.storeOperation(state)
	defer m.removeOperation(req.ID)

	// Initialize operation in broadcaster (reset handled internally)

	// Determine which steps to run based on request
	var steps []Step
	stepParam, hasStep := req.Parameters["step"].(string)

	if hasStep && stepParam != "" && stepParam != "full_pipeline" {
		// Single step requested
		requestedStep, err := m.registry.Get(stepParam)
		if err != nil || requestedStep == nil {
			if err == nil {
				err = fmt.Errorf("requested step not found: %s", stepParam)
			}
			m.logOperationError(ctx, req.ID, err)
			state.Fail(err)
			return m.createResponse(state), err
		}
		steps = []Step{requestedStep}

		slog.InfoContext(ctx, "executing_single_step",
			slog.String("step_id", stepParam),
			slog.String("operation_id", req.ID))
	} else {
		// Full pipeline requested or no step specified
		var err error
		steps, err = m.registry.GetDependencyOrder()
		if err != nil {
			m.logOperationError(ctx, req.ID, fmt.Errorf("failed to get dependency order: %w", err))
			state.Fail(err)
			return m.createResponse(state), err
		}

		slog.InfoContext(ctx, "executing_full_pipeline",
			slog.Int("step_count", len(steps)),
			slog.String("operation_id", req.ID))
	}

	// Initialize Step states
	// IMPORTANT: Use Step IDs for broadcaster snapshot IDs so that subsequent
	// UpdateStepProgress calls (which use Step.ID()) correctly match entries.
	// The human-readable name is still carried inside the in-memory StepState.
	stepNames := make([]string, len(steps))
	for i, Step := range steps {
		StepState := NewStepState(Step.ID(), Step.Name())
		state.SetStage(Step.ID(), StepState)
		// Pass the Step ID here to keep IDs consistent between creation and updates
		stepNames[i] = Step.ID()
	}

	// Create operation in broadcaster with all steps
	m.broadcaster.CreateOperation(req.ID, stepNames)

	// Start operation execution
	state.Start()
	m.broadcaster.StartOperation(req.ID)

	// Execute steps based on execution mode
	var err error
	if m.config.ExecutionMode == ExecutionModeSequential {
		err = m.executeSequential(ctx, state, steps)
	} else {
		err = m.executeParallel(ctx, state, steps)
	}

	// Update final operation state
	if err != nil {
		state.Fail(err)
		m.broadcaster.FailOperation(req.ID, err)
	} else {
		state.Complete()
		m.broadcaster.CompleteOperation(req.ID, "Operation completed successfully")
	}

	return m.createResponse(state), err
}

// executeSequential executes steps one by one
func (m *Manager) executeSequential(ctx context.Context, state *OperationState, steps []Step) error {
	slog.InfoContext(ctx, "sequential_execution_start",
		slog.String("operation_id", state.ID),
		slog.Int("stage_count", len(steps)))
	for i, Step := range steps {
		select {
		case <-ctx.Done():
			slog.WarnContext(ctx, "operation_cancelled",
				slog.String("operation_id", state.ID),
				slog.String("Step", Step.ID()))
			return NewCancellationError(Step.ID())
		default:
			// Check if Step should be skipped due to failed dependencies
			StepState := state.GetStage(Step.ID())
			if StepState != nil && StepState.Status == StepStatusSkipped {
				slog.InfoContext(ctx, "stage_skipped",
					slog.String("operation_id", state.ID),
					slog.String("Step", Step.ID()),
					slog.Int("stage_number", i+1),
					slog.Int("total_stages", len(steps)))
				continue
			}

			// Check if previous steps are actually complete (for sequential execution)
			if i > 0 {
				prevStage := steps[i-1]
				prevState := state.GetStage(prevStage.ID())
				if prevState != nil && prevState.Status != StepStatusCompleted && prevState.Status != StepStatusSkipped {
					// If continue on error is enabled and previous Step failed, allow this Step to continue
					if m.config.ContinueOnError && prevState.Status == StepStatusFailed {
						slog.InfoContext(ctx, "continuing_after_failed_stage",
							slog.String("operation_id", state.ID),
							slog.String("Step", Step.ID()),
							slog.String("previous_stage", prevStage.ID()),
							slog.String("previous_status", string(prevState.Status)))
					} else {
						slog.ErrorContext(ctx, "previous_stage_incomplete",
							slog.String("operation_id", state.ID),
							slog.String("Step", Step.ID()),
							slog.String("previous_stage", prevStage.ID()),
							slog.String("previous_status", string(prevState.Status)))
						StepState.Skip(fmt.Sprintf("Previous Step %s not completed", prevStage.ID()))
						m.broadcaster.UpdateStepProgress(state.ID, Step.ID(), int(StepState.Progress), fmt.Sprintf("Skipped: Previous step %s not completed", prevStage.ID()))
						continue
					}
				}
			}

			slog.InfoContext(ctx, "executing_stage",
				slog.String("operation_id", state.ID),
				slog.String("Step", Step.ID()),
				slog.Int("stage_number", i+1),
				slog.Int("total_stages", len(steps)))
			if err := m.executeStage(ctx, state, Step); err != nil {
				m.logStageError(ctx, state.ID, Step.ID(), err)
				if !m.config.ContinueOnError {
					// Skip all dependent steps
					m.skipDependentStages(state, steps, Step.ID())
					return err
				}
				slog.WarnContext(ctx, "stage_failed_continuing",
					slog.String("operation_id", state.ID),
					slog.String("Step", Step.ID()),
					slog.String("error", err.Error()))
			} else {
				// Verify Step actually completed
				updatedState := state.GetStage(Step.ID())
				if updatedState.Status == StepStatusCompleted {
					slog.InfoContext(ctx, "stage_completed_successfully",
						slog.String("operation_id", state.ID),
						slog.String("Step", Step.ID()))
				} else {
					slog.WarnContext(ctx, "stage_finished_wrong_status",
						slog.String("operation_id", state.ID),
						slog.String("Step", Step.ID()),
						slog.String("status", string(updatedState.Status)))
				}
			}
		}
	}
	slog.InfoContext(ctx, "all_stages_completed",
		slog.String("operation_id", state.ID))
	return nil
}

// executeParallel executes independent steps in parallel
func (m *Manager) executeParallel(ctx context.Context, state *OperationState, steps []Step) error {
	// NOTE: Parallel execution is intentionally not implemented.
	// ISX operations must remain sequential because each step depends on the output
	// of the previous step:
	// 1. Scraping produces Excel files
	// 2. Processing converts Excel to CSV (requires Excel files from step 1)
	// 3. Indexing extracts indices from CSV (requires CSV files from step 2)
	// 4. Analysis analyzes the indexed data (requires indices from step 3)
	// The data pipeline is inherently sequential by design.
	return m.executeSequential(ctx, state, steps)
}

// executeStage executes a single Step with retry logic
func (m *Manager) executeStage(ctx context.Context, OperationState *OperationState, Step Step) error {
	m.logStageStart(ctx, OperationState.ID, Step.ID())
	StepState := OperationState.GetStage(Step.ID())
	if StepState == nil {
		slog.ErrorContext(ctx, "stage_state_not_found",
			slog.String("operation_id", OperationState.ID),
			slog.String("Step", Step.ID()))
		return NewFatalError("Step state not found", nil)
	}

	// Check dependencies
	slog.DebugContext(ctx, "checking_dependencies",
		slog.String("operation_id", OperationState.ID),
		slog.String("Step", Step.ID()))
	if err := m.checkDependencies(OperationState, Step); err != nil {
		slog.WarnContext(ctx, "dependencies_not_met",
			slog.String("operation_id", OperationState.ID),
			slog.String("Step", Step.ID()),
			slog.String("error", err.Error()))
		StepState.Skip(fmt.Sprintf("Dependencies not met: %v", err))
		m.broadcaster.UpdateStepProgress(OperationState.ID, Step.ID(), int(StepState.Progress), fmt.Sprintf("Skipped: Dependencies not met - %v", err))
		return err
	}

	// Validate Step
	slog.DebugContext(ctx, "validating_stage",
		slog.String("operation_id", OperationState.ID),
		slog.String("Step", Step.ID()))
	if err := Step.Validate(OperationState); err != nil {
		slog.WarnContext(ctx, "validation_failed",
			slog.String("operation_id", OperationState.ID),
			slog.String("Step", Step.ID()),
			slog.String("error", err.Error()))
		StepState.Skip(fmt.Sprintf("Validation failed: %v", err))
		m.broadcaster.UpdateStepProgress(OperationState.ID, Step.ID(), int(StepState.Progress), fmt.Sprintf("Skipped: Validation failed - %v", err))
		return NewValidationError(Step.ID(), err.Error())
	}

	// Get Step timeout
	timeout := m.config.GetStageTimeout(Step.ID())
	stageCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute with retries
	retryConfig := m.config.RetryConfig
	var lastErr error

	for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
		// Start Step
		StepState.Start()
		// Use broadcaster for all updates - single source of truth
		m.broadcaster.UpdateStepProgress(OperationState.ID, Step.ID(), int(StepState.Progress), "Step started")

		// Execute Step
		slog.InfoContext(ctx, "calling_execute",
			slog.String("operation_id", OperationState.ID),
			slog.String("Step", Step.ID()),
			slog.Int("attempt", attempt))
		startTime := time.Now()
		err := Step.Execute(stageCtx, OperationState)
		duration := time.Since(startTime)

		if err == nil {
			// Success
			m.logStageComplete(ctx, OperationState.ID, Step.ID(), duration)
			StepState.Complete()
			m.broadcaster.CompleteStep(OperationState.ID, Step.ID(), "Step completed successfully")

			return nil
		}

		slog.ErrorContext(ctx, "stage_execution_failed",
			slog.String("operation_id", OperationState.ID),
			slog.String("Step", Step.ID()),
			slog.Duration("duration", duration),
			slog.String("error", err.Error()))

		// Log Step metadata for debugging
		if StepState.Metadata != nil {
			if metaJSON, err := json.Marshal(StepState.Metadata); err == nil {
				slog.ErrorContext(ctx, "stage_metadata",
					slog.String("operation_id", OperationState.ID),
					slog.String("Step", Step.ID()),
					slog.String("metadata", string(metaJSON)))
			}
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) || attempt >= retryConfig.MaxAttempts {
			StepState.Fail(err)
			m.broadcaster.UpdateStepProgress(OperationState.ID, Step.ID(), int(StepState.Progress), fmt.Sprintf("Step failed: %v", err))
			return WrapError(err, Step.ID(), "Step execution failed")
		}

		// Calculate retry delay
		delay := m.calculateRetryDelay(attempt, retryConfig)
		slog.WarnContext(ctx, "stage_retry",
			slog.String("operation_id", OperationState.ID),
			slog.String("Step", Step.ID()),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", retryConfig.MaxAttempts),
			slog.Duration("delay", delay),
			slog.String("error", err.Error()))

		// Wait before retry
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-stageCtx.Done():
			StepState.Fail(NewTimeoutError(Step.ID(), timeout.String()))
			m.broadcaster.UpdateStepProgress(OperationState.ID, Step.ID(), int(StepState.Progress), fmt.Sprintf("Step timed out after %s", timeout))
			return NewTimeoutError(Step.ID(), timeout.String())
		}
	}

	// All retries exhausted
	StepState.Fail(lastErr)
	m.broadcaster.UpdateStepProgress(OperationState.ID, Step.ID(), int(StepState.Progress), fmt.Sprintf("Step failed after %d retries: %v", retryConfig.MaxAttempts, lastErr))
	return WrapError(lastErr, Step.ID(), "Step execution failed after retries")
}

// skipDependentStages marks all steps that depend on the failed Step as skipped
func (m *Manager) skipDependentStages(state *OperationState, steps []Step, failedStageID string) {
	for _, Step := range steps {
		deps := Step.GetDependencies()
		for _, dep := range deps {
			if dep == failedStageID {
				StepState := state.GetStage(Step.ID())
				if StepState != nil && StepState.Status == StepStatusPending {
					StepState.Skip(fmt.Sprintf("Dependency %s failed", failedStageID))
					m.broadcaster.UpdateStepProgress(state.ID, Step.ID(), int(StepState.Progress), fmt.Sprintf("Skipped: Dependency %s failed", failedStageID))
					// Recursively skip steps that depend on this one
					m.skipDependentStages(state, steps, Step.ID())
				}
				break
			}
		}
	}
}

// checkDependencies verifies that all dependencies are satisfied
func (m *Manager) checkDependencies(state *OperationState, Step Step) error {
	deps := Step.GetDependencies()
	for _, dep := range deps {
		depState := state.GetStage(dep)
		if depState == nil {
			return fmt.Errorf("dependency %s not found", dep)
		}
		if depState.Status != StepStatusCompleted {
			return fmt.Errorf("dependency %s not completed (status: %s)", dep, depState.Status)
		}
	}
	return nil
}

// calculateRetryDelay calculates the delay before next retry
func (m *Manager) calculateRetryDelay(attempt int, config RetryConfig) time.Duration {
	delay := config.InitialDelay * time.Duration(float64(attempt-1)*config.Multiplier)
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}
	return delay
}

// All WebSocket updates now go through StatusBroadcaster - single source of truth
// Direct hub broadcasts have been removed to simplify data flow

// createResponse creates a operation response from state
func (m *Manager) createResponse(state *OperationState) *OperationResponse {
	resp := &OperationResponse{
		ID:       state.ID,
		Status:   state.Status,
		Duration: state.Duration(),
		Steps:    state.Steps,
	}

	if state.Error != nil {
		resp.Error = state.Error.Error()
	}

	return resp
}

// GetOperation retrieves the state of a running operation
func (m *Manager) GetOperation(id string) (*OperationState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.operations[id]
	if !exists {
		return nil, fmt.Errorf("operation %s not found", id)
	}

	return state.Clone(), nil
}

// ListOperations returns all active operations
func (m *Manager) ListOperations() []*OperationState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	operations := make([]*OperationState, 0, len(m.operations))
	for _, state := range m.operations {
		operations = append(operations, state.Clone())
	}

	return operations
}

// CancelOperation cancels a running operation
func (m *Manager) CancelOperation(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.operations[id]
	if !exists {
		return fmt.Errorf("operation %s not found", id)
	}

	state.Cancel()
	m.broadcaster.FailOperation(id, fmt.Errorf("operation cancelled by user"))
	return nil
}

// storeOperation stores a operation state
func (m *Manager) storeOperation(state *OperationState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[state.ID] = state
}

// removeOperation removes a operation state
func (m *Manager) removeOperation(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.operations, id)
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}
