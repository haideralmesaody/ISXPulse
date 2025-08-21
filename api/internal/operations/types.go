package operations

import (
	"time"
)

// operation Step identifiers
const (
	StageIDScraping  = "scraping"
	StageIDProcessing = "processing"
	StageIDIndices   = "indices"
	StageIDLiquidity  = "liquidity"
)

// operation Step names
const (
	StageNameScraping  = "Data Collection"
	StageNameProcessing = "Data Processing"
	StageNameIndices   = "Index Extraction"
	StageNameLiquidity  = "Liquidity Calculation"
)

// Context keys for operation state
const (
	ContextKeyFromDate      = "from_date"
	ContextKeyToDate        = "to_date"
	ContextKeyMode          = "mode"
	ContextKeyDownloadDir   = "download_dir"
	ContextKeyReportDir     = "report_dir"
	ContextKeyFilesFound    = "files_found"
	ContextKeyFilesProcessed = "files_processed"
	ContextKeyScraperSuccess = "scraper_success"
)

// operation modes
const (
	ModeInitial     = "initial"
	ModeAccumulative = "accumulative"
	ModeFull        = "full"
)

// WebSocket event types - using frontend format
const (
	EventTypeOperationStatus   = "operation:status"
	EventTypePipelineProgress = "operation:progress"
	EventTypePipelineComplete = "operation:complete"
	EventTypeOperationError    = "operation:error"
	EventTypePipelineReset    = "operation:reset"
)

// Default timeouts
const (
	DefaultStageTimeout     = 30 * time.Minute
	DefaultScrapingTimeout  = 60 * time.Minute
	DefaultProcessingTimeout = 30 * time.Minute
	DefaultIndicesTimeout   = 10 * time.Minute
	DefaultLiquidityTimeout  = 5 * time.Minute
)

// ExecutionMode defines how steps are executed
type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential"
	ExecutionModeParallel   ExecutionMode = "parallel"
)

// OperationMode defines the mode of operation execution
type OperationMode string

const (
	OperationModeFull    OperationMode = "full"
	OperationModePartial OperationMode = "partial"
	OperationModeResume  OperationMode = "resume"
)

// RetryConfig defines retry behavior for steps
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
	Multiplier  float64       `json:"multiplier"`
}

// NewRetryConfig returns the default retry configuration
func NewRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// StageExecutionResult represents the result of a Step execution
type StageExecutionResult struct {
	StageID   string                 `json:"stage_id"`
	Success   bool                   `json:"success"`
	Output    string                 `json:"output,omitempty"`
	Error     error                  `json:"error,omitempty"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// OperationRequest represents a request to execute a operation
type OperationRequest struct {
	ID         string                 `json:"id"`
	Mode       string                 `json:"mode"`
	FromDate   string                 `json:"from_date,omitempty"`
	ToDate     string                 `json:"to_date,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// OperationResponse represents the response from a operation execution
type OperationResponse struct {
	ID       string                    `json:"id"`
	Status   OperationStatusValue      `json:"status"`
	Duration time.Duration             `json:"duration"`
	Steps    map[string]*StepState     `json:"steps"`
	Error    string                    `json:"error,omitempty"`
}

// ProgressUpdate represents a progress update from a Step
type ProgressUpdate struct {
	StageID    string                 `json:"stage_id"`
	Progress   float64                `json:"progress"`
	Message    string                 `json:"message"`
	ETA        string                 `json:"eta,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StageMetrics represents performance metrics for a Step
type StageMetrics struct {
	StageID        string        `json:"stage_id"`
	ExecutionCount int           `json:"execution_count"`
	SuccessCount   int           `json:"success_count"`
	FailureCount   int           `json:"failure_count"`
	AverageDuration time.Duration `json:"average_duration"`
	LastExecution  *time.Time    `json:"last_execution,omitempty"`
}

// OperationType represents an available operation type
type OperationType struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description"`
	Dependencies []string                `json:"dependencies"`
	CanRunAlone  bool                    `json:"can_run_alone"`
	Parameters   []ParameterDefinition   `json:"parameters"`
}

// ParameterDefinition defines a parameter for an operation type
type ParameterDefinition struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, date, select, boolean
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Options     []string    `json:"options,omitempty"` // For select type
}