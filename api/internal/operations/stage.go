package operations

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DataRequirement specifies data needed for a step to run
type DataRequirement struct {
	Type     string `json:"type"`      // Type of data needed (e.g., "excel_files", "csv_files")
	Location string `json:"location"`  // Where to find the data
	MinCount int    `json:"min_count"` // Minimum number of files/items needed
	Optional bool   `json:"optional"`  // Whether this requirement is optional
}

// DataOutput specifies data produced by a step
type DataOutput struct {
	Type     string `json:"type"`     // Type of data produced
	Location string `json:"location"` // Where the data is stored
	Pattern  string `json:"pattern"`  // File pattern (e.g., "*.csv")
}

// Step represents a single Step in the operation
type Step interface {
	// ID returns the unique identifier for this Step
	ID() string

	// Name returns the human-readable name for this Step
	Name() string

	// Execute runs the Step with the given context and operation state
	Execute(ctx context.Context, state *OperationState) error

	// Validate checks if the Step can be executed with the current state
	Validate(state *OperationState) error

	// GetDependencies returns the IDs of steps that must complete before this Step
	// DEPRECATED: Use RequiredInputs() for data-based dependencies instead
	GetDependencies() []string
	
	// RequiredInputs returns the data requirements for this step to run
	RequiredInputs() []DataRequirement
	
	// ProducedOutputs returns the data outputs this step produces
	ProducedOutputs() []DataOutput
	
	// CanRun checks if the step can run based on available data
	CanRun(manifest *PipelineManifest) bool
}

// StepStatus represents the current status of a Step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusActive    StepStatus = "active"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// StepState represents the runtime state of a Step
type StepState struct {
	mu          sync.RWMutex           `json:"-"`
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Status      StepStatus            `json:"status"`
	StartTime   *time.Time             `json:"start_time,omitempty"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Progress    float64                `json:"progress"`
	Message     string                 `json:"message"`
	Error       error                  `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewStepState creates a new Step state with default values
func NewStepState(id, name string) *StepState {
	return &StepState{
		ID:       id,
		Name:     name,
		Status:   StepStatusPending,
		Progress: 0,
		Metadata: make(map[string]interface{}),
	}
}

// Start marks the Step as active and sets the start time
func (s *StepState) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.StartTime = &now
	s.Status = StepStatusActive
	s.Progress = 0
}

// Complete marks the Step as completed and sets the end time
func (s *StepState) Complete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.EndTime = &now
	s.Status = StepStatusCompleted
	s.Progress = 100
}

// Fail marks the Step as failed with the given error
func (s *StepState) Fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.EndTime = &now
	s.Status = StepStatusFailed
	s.Error = err
}

// Skip marks the Step as skipped with the given reason
func (s *StepState) Skip(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.EndTime = &now
	s.Status = StepStatusSkipped
	s.Message = reason
}

// UpdateProgress updates the Step progress and message
func (s *StepState) UpdateProgress(progress float64, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.Progress = progress
	s.Message = message
}

// Duration returns the duration of the Step execution
func (s *StepState) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.StartTime == nil {
		return 0
	}
	if s.EndTime != nil {
		return s.EndTime.Sub(*s.StartTime)
	}
	return time.Since(*s.StartTime)
}

// BaseStage provides common functionality for Step implementations
type BaseStage struct {
	id           string
	name         string
	dependencies []string
}

// NewBaseStage creates a new base Step
func NewBaseStage(id, name string, dependencies []string) BaseStage {
	if dependencies == nil {
		dependencies = []string{}
	}
	return BaseStage{
		id:           id,
		name:         name,
		dependencies: dependencies,
	}
}

// ID returns the Step ID
func (b *BaseStage) ID() string {
	if b == nil {
		return ""
	}
	return b.id
}

// Name returns the Step name
func (b *BaseStage) Name() string {
	if b == nil {
		return ""
	}
	return b.name
}

// GetDependencies returns the Step dependencies
func (b *BaseStage) GetDependencies() []string {
	if b == nil {
		return nil
	}
	return b.dependencies
}

// Validate provides a default validation that always passes
func (b *BaseStage) Validate(state *OperationState) error {
	if b == nil {
		return fmt.Errorf("BaseStage is nil")
	}
	return nil
}

// RequiredInputs returns empty requirements by default (no inputs needed)
func (b *BaseStage) RequiredInputs() []DataRequirement {
	if b == nil {
		return nil
	}
	return []DataRequirement{}
}

// ProducedOutputs returns empty outputs by default
func (b *BaseStage) ProducedOutputs() []DataOutput {
	if b == nil {
		return nil
	}
	return []DataOutput{}
}

// CanRun checks if the stage can run based on available data
// Default implementation always returns true (no requirements)
func (b *BaseStage) CanRun(manifest *PipelineManifest) bool {
	if b == nil {
		return false
	}
	// If there are no required inputs, stage can always run
	requirements := b.RequiredInputs()
	if len(requirements) == 0 {
		return true
	}
	
	// Check each requirement
	for _, req := range requirements {
		if req.Optional {
			continue // Skip optional requirements
		}
		
		data, exists := manifest.GetData(req.Type)
		if !exists {
			return false // Required data not available
		}
		
		if req.MinCount > 0 && data.FileCount < req.MinCount {
			return false // Not enough files
		}
	}
	
	return true
}