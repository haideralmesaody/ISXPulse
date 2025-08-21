package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"isxcli/internal/config"
	"isxcli/internal/operations"
)

// OperationService manages operation operations
type OperationService struct {
	manager *operations.Manager
	logger  *slog.Logger
	paths   *config.Paths
}

// WebSocketOperationAdapter adapts WebSocket communication for operation

type WebSocketOperationAdapter struct {
	hub WebSocketHub
}

// NewWebSocketOperationAdapter creates a new WebSocket operation adapter
func NewWebSocketOperationAdapter(hub WebSocketHub) *WebSocketOperationAdapter {
	return &WebSocketOperationAdapter{hub: hub}
}

// WebSocketHub interface for WebSocket communication
type WebSocketHub interface {
	Broadcast(messageType string, data interface{})
}

// WebSocketOperationAdapter implements OperationAdapter
func (w *WebSocketOperationAdapter) SendProgress(stageID, message string, progress int) {
	w.hub.Broadcast("operation_progress", map[string]interface{}{
		"step":    stageID,
		"message":  message,
		"progress": progress,
		"status":   "active",
	})
}

func (w *WebSocketOperationAdapter) SendComplete(stageID, message string, success bool) {
	status := "completed"
	if !success {
		status = "failed"
	}
	w.hub.Broadcast("operation_complete", map[string]interface{}{
		"step":   stageID,
		"message": message,
		"status":  status,
		"success": success,
	})
}

func (w *WebSocketOperationAdapter) SendError(stageID, error string) {
	w.hub.Broadcast("operation_error", map[string]interface{}{
		"step": stageID,
		"error": error,
		"status": "error",
	})
}

// BroadcastUpdate implements operations.WebSocketHub interface
func (w *WebSocketOperationAdapter) BroadcastUpdate(eventType, step, status string, metadata interface{}) {
	data := map[string]interface{}{
		"eventType": eventType,
		"step":     step,
		"status":    status,
	}
	if metadata != nil {
		data["metadata"] = metadata
	}
	w.hub.Broadcast(eventType, data)
}


// NewOperationService creates a new operation service
func NewOperationService(adapter *WebSocketOperationAdapter, logger *slog.Logger) (*OperationService, error) {
	// Get the centralized paths
	paths, err := config.GetPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get paths: %w", err)
	}
	
	// Log startup paths for visibility
	if logger != nil {
		logger.Info("OperationService initialized with paths",
			slog.String("executable_dir", paths.ExecutableDir),
			slog.String("data_dir", paths.DataDir),
			slog.String("downloads_dir", paths.DownloadsDir),
			slog.String("reports_dir", paths.ReportsDir))
	}

	manager := operations.NewManager(adapter, nil, nil)
	
	// Register operation steps with WebSocket adapter
	if err := registerStages(manager, paths.ExecutableDir, logger, adapter); err != nil {
		return nil, fmt.Errorf("failed to register steps: %w", err)
	}

	return &OperationService{
		manager: manager,
		logger:  logger,
		paths:   paths,
	}, nil
}

// registerStages registers all operation steps
func registerStages(manager *operations.Manager, executableDir string, logger *slog.Logger, wsAdapter *WebSocketOperationAdapter) error {
	// Create stage options with WebSocket integration and StatusBroadcaster
	stageOptions := &operations.StageOptions{
		EnableProgress: true,
		WebSocketManager: wsAdapter,
		StatusBroadcaster: manager.GetBroadcaster(), // Pass the centralized StatusBroadcaster
	}
	
	// Create steps with WebSocket integration for progress reporting
	scraper := operations.NewScrapingStage(executableDir, logger, stageOptions)
	processor := operations.NewProcessingStage(executableDir, logger, stageOptions)
	indices := operations.NewIndicesStage(executableDir, logger, stageOptions)
	liquidity := operations.NewLiquidityStage(executableDir, logger, stageOptions)

	// Register steps
	manager.GetRegistry().Register(scraper)
	manager.GetRegistry().Register(processor)
	manager.GetRegistry().Register(indices)
	manager.GetRegistry().Register(liquidity)

	return nil
}

// StartOperation starts a new operation execution
func (ps *OperationService) StartOperation(ctx context.Context, params map[string]interface{}) (string, error) {
	// Use the passed context
	
	// Extract dates from parameters if present
	// Check both 'from_date'/'to_date' and 'from'/'to' for compatibility
	fromDate := ""
	toDate := ""
	mode := "full"
	
	// Check for from_date or from
	if fd, ok := params["from_date"].(string); ok {
		fromDate = fd
	} else if f, ok := params["from"].(string); ok {
		fromDate = f
	}
	
	// Check for to_date or to  
	if td, ok := params["to_date"].(string); ok {
		toDate = td
	} else if t, ok := params["to"].(string); ok {
		toDate = t
	}
	if m, ok := params["mode"].(string); ok {
		mode = m
	}
	
	// Create operation request with dates at root level
	request := operations.OperationRequest{
		ID:         fmt.Sprintf("operation-%d", time.Now().Unix()),
		Mode:       mode,
		FromDate:   fromDate,
		ToDate:     toDate,
		Parameters: params,
	}

	// Log the request details
	if ps.logger != nil {
		ps.logger.Info("Creating operation request",
			slog.String("id", request.ID),
			slog.String("mode", request.Mode),
			slog.String("from_date", request.FromDate),
			slog.String("to_date", request.ToDate),
			slog.Any("parameters", request.Parameters))
	}

	resp, err := ps.manager.Execute(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to start operation: %w", err)
	}

	if ps.logger != nil {
		ps.logger.Info("operation started",
			slog.String("id", resp.ID),
			slog.String("status", string(resp.Status)))
	}
	return resp.ID, nil
}

// StartScraping starts the scraping step
func (ps *OperationService) StartScraping(ctx context.Context, params map[string]interface{}) (string, error) {
	// Log received parameters for debugging
	if ps.logger != nil {
		ps.logger.Info("StartScraping received params",
			slog.Any("params", params))
	}
	
	// Extract args from the params structure
	args, ok := params["args"].(map[string]interface{})
	if !ok {
		// Fallback to direct params if no args wrapper
		args = params
		if ps.logger != nil {
			ps.logger.Warn("No 'args' wrapper found, using params directly")
		}
	}
	
	// Build operation parameters with correct field names
	fromValue := getValue(args, "from", "")
	toValue := getValue(args, "to", "")
	
	scrapingParams := map[string]interface{}{
		"mode":      getValue(args, "mode", "initial"),
		"from_date": fromValue,  // Map 'from' to 'from_date'
		"to_date":   toValue,    // Map 'to' to 'to_date'
		"headless":  getValue(args, "headless", true),
		"step":     "scraping",
	}
	
	// Log transformed parameters with detailed mapping
	if ps.logger != nil {
		ps.logger.Info("Parameter transformation for scraping",
			slog.String("from_input", fmt.Sprintf("%v", args["from"])),
			slog.String("from_mapped", fromValue.(string)),
			slog.String("to_input", fmt.Sprintf("%v", args["to"])),
			slog.String("to_mapped", toValue.(string)),
			slog.Any("final_params", scrapingParams))
	}

	return ps.StartOperation(ctx, scrapingParams)
}

// ExecuteOperation executes an operation with the given request
func (ps *OperationService) ExecuteOperation(ctx context.Context, request *operations.OperationRequest) (*operations.OperationResponse, error) {
	resp, err := ps.manager.Execute(ctx, *request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute operation: %w", err)
	}

	if ps.logger != nil {
		ps.logger.Info("operation executed",
			slog.String("id", resp.ID),
			slog.String("status", string(resp.Status)))
	}

	return resp, nil
}

// GetOperationStatus returns the status of a specific operation
func (ps *OperationService) GetOperationStatus(ctx context.Context, operationID string) (*operations.OperationState, error) {
	state, err := ps.GetStatus(ctx, operationID)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// CancelOperation cancels a running operation
func (ps *OperationService) CancelOperation(ctx context.Context, operationID string) error {
	return ps.StopOperation(ctx, operationID)
}

// StartProcessing starts the processing step
func (ps *OperationService) StartProcessing(ctx context.Context, params map[string]interface{}) (string, error) {
	// Processing uses default directories, no input_dir needed
	processingParams := map[string]interface{}{
		"step": "processing",
		"mode": getValue(params, "mode", "full"),
	}

	return ps.StartOperation(ctx, processingParams)
}

// StartIndexExtraction starts the index extraction step
func (ps *OperationService) StartIndexExtraction(ctx context.Context, params map[string]interface{}) (string, error) {
	indexParams := map[string]interface{}{
		"step": "indices",
		"mode":  "full",
	}

	return ps.StartOperation(ctx, indexParams)
}

// StopOperation stops a running operation
func (ps *OperationService) StopOperation(ctx context.Context, pipelineID string) error {
	if err := ps.manager.CancelOperation(pipelineID); err != nil {
		return fmt.Errorf("failed to stop operation: %w", err)
	}

	if ps.logger != nil {
		ps.logger.Info("operation stopped",
			slog.String("id", pipelineID))
	}
	return nil
}

// GetStatus returns operation status
func (ps *OperationService) GetStatus(ctx context.Context, pipelineID string) (*operations.OperationState, error) {
	if pipelineID == "" {
		return nil, fmt.Errorf("operation ID is required")
	}

	state, err := ps.manager.GetOperation(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("operation not found: %w", err)
	}

	return state, nil
}

// ListOperations returns all operations
func (ps *OperationService) ListOperations(ctx context.Context) ([]*operations.OperationState, error) {
	states := ps.manager.ListOperations()
	return states, nil
}

// ListOperationsByStatus returns operations filtered by status
func (ps *OperationService) ListOperationsByStatus(ctx context.Context, status operations.OperationStatusValue) ([]*operations.OperationState, error) {
	states := ps.manager.ListOperations()
	var result []*operations.OperationState
	for _, state := range states {
		if state.Status == status {
			result = append(result, state)
		}
	}
	return result, nil
}

// GetOperationTypes returns all available operation types (stages)
func (ps *OperationService) GetOperationTypes(ctx context.Context) ([]operations.OperationType, error) {
	// Get all registered stages from the registry
	stages := ps.manager.GetRegistry().List()
	
	types := make([]operations.OperationType, 0, len(stages))
	for _, stage := range stages {
		opType := operations.OperationType{
			ID:           stage.ID(),
			Name:         stage.Name(),
			Description:  getStageDescription(stage.ID()),
			Dependencies: stage.GetDependencies(),
			CanRunAlone:  len(stage.GetDependencies()) == 0,
			Parameters:   getStageParameters(stage.ID()),
		}
		types = append(types, opType)
	}
	
	// Add a "full pipeline" type that runs all stages
	// Calculate default dates for full pipeline
	today := time.Now().Format("2006-01-02")
	defaultFromDate := "2025-01-01"
	
	types = append(types, operations.OperationType{
		ID:          "full_pipeline",
		Name:        "Full Pipeline",
		Description: "Run all stages in sequence: scraping → processing → indices → liquidity",
		Dependencies: []string{},
		CanRunAlone: true,
		Parameters: []operations.ParameterDefinition{
			{
				Name:        "mode",
				Type:        "select",
				Description: "Operation mode",
				Required:    false,
				Default:     "initial",
				Options:     []string{"initial", "accumulative", "full"},
			},
			{
				Name:        "from",
				Type:        "date",
				Description: "Start date for data collection",
				Required:    false,
				Default:     defaultFromDate,
			},
			{
				Name:        "to",
				Type:        "date",
				Description: "End date for data collection",
				Required:    false,
				Default:     today,
			},
		},
	})
	
	return types, nil
}

// getStageDescription returns a user-friendly description for each stage
func getStageDescription(stageID string) string {
	descriptions := map[string]string{
		operations.StageIDScraping:   "Download ISX daily trading reports from the official website",
		operations.StageIDProcessing: "Convert Excel files to CSV format with data normalization",
		operations.StageIDIndices:    "Extract ISX60 and ISX15 index values from processed data",
		operations.StageIDLiquidity:   "Calculate hybrid liquidity metrics and generate liquidity analysis reports",
	}
	
	if desc, ok := descriptions[stageID]; ok {
		return desc
	}
	return "Process data"
}

// getStageParameters returns the parameters accepted by each stage
func getStageParameters(stageID string) []operations.ParameterDefinition {
	switch stageID {
	case operations.StageIDScraping:
		// Calculate default dates
		today := time.Now().Format("2006-01-02")
		defaultFromDate := "2025-01-01"
		
		return []operations.ParameterDefinition{
			{
				Name:        "mode",
				Type:        "select",
				Description: "Scraping mode",
				Required:    false,
				Default:     "initial",
				Options:     []string{"initial", "accumulative", "full"},
			},
			{
				Name:        "from",
				Type:        "date",
				Description: "Start date (YYYY-MM-DD)",
				Required:    false,
				Default:     defaultFromDate,
			},
			{
				Name:        "to",
				Type:        "date",
				Description: "End date (YYYY-MM-DD)",
				Required:    false,
				Default:     today,
			},
		}
	case operations.StageIDProcessing:
		// Processing stage uses default directories, no parameters needed
		return []operations.ParameterDefinition{}
	default:
		return []operations.ParameterDefinition{}
	}
}

// CancelAll stops all running operations
func (ps *OperationService) CancelAll(ctx context.Context) error {
	ops := ps.manager.ListOperations()
	for _, p := range ops {
		if p.Status == operations.OperationStatusRunning {
			if err := ps.manager.CancelOperation(p.ID); err != nil {
				if ps.logger != nil {
					ps.logger.Error("Failed to cancel operation",
						slog.String("id", p.ID),
						slog.String("error", err.Error()))
				}
				return err
			}
		}
	}
	return nil
}

// ExecuteStage executes a specific step
func (ps *OperationService) ExecuteStage(stageID string, ctx context.Context) error {
	// This would execute individual steps - implement as needed
	return fmt.Errorf("individual step execution not implemented")
}

// ValidateExecutables checks if required executables exist
func (ps *OperationService) ValidateExecutables(ctx context.Context) error {
	executables := []string{
		"scraper.exe",
		"process.exe",
		"indexcsv.exe",
	}

	for _, exe := range executables {
		path := filepath.Join(ps.paths.ExecutableDir, exe)
		if ps.logger != nil {
			ps.logger.Debug("Checking for executable",
				slog.String("exe", exe),
				slog.String("path", path))
		}
		
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if ps.logger != nil {
				ps.logger.Error("Required executable not found",
					slog.String("exe", exe),
					slog.String("path", path))
			}
			return fmt.Errorf("required executable not found: %s at %s", exe, path)
		}
		
		if ps.logger != nil {
			ps.logger.Info("Found required executable",
				slog.String("exe", exe),
				slog.String("path", path))
		}
	}

	return nil
}

// GetManager returns the underlying operation manager
func (ps *OperationService) GetManager() *operations.Manager {
	return ps.manager
}

// GetStageInfo returns information about available steps
func (ps *OperationService) GetStageInfo() map[string]interface{} {
	return map[string]interface{}{
		"steps": []map[string]interface{}{
			{
				"id":   "scraping",
				"name": "Scraping",
				"description": "Download daily reports from ISX website",
				"executable":  "scraper.exe",
			},
			{
				"id":   "processing",
				"name": "Processing",
				"description": "Process Excel files into CSV format",
				"executable":  "process.exe",
			},
			{
				"id":   "indices",
				"name": "Index Extraction",
				"description": "Extract market indices from processed data",
				"executable":  "indexcsv.exe",
			},
			{
				"id":   "liquidity",
				"name": "Liquidity Calculation",
				"description": "Calculate hybrid liquidity metrics and generate liquidity analysis reports",
				"executable":  "",
			},
		},
	}
}

// getValue safely extracts a value from a map with a default
func getValue(m map[string]interface{}, key string, defaultValue interface{}) interface{} {
	if val, ok := m[key]; ok && val != nil {
		return val
	}
	return defaultValue
}

// GetOperationMetrics returns metrics about operations
func (ps *OperationService) GetOperationMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Get basic metrics - simplified implementation for Phase 1
	operations, err := ps.ListOperations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}
	
	activeCount := 0
	completedCount := 0
	failedCount := 0
	
	for _, op := range operations {
		switch op.Status {
		case "running", "pending":
			activeCount++
		case "completed":
			completedCount++
		case "failed", "cancelled":
			failedCount++
		}
	}
	
	metrics := map[string]interface{}{
		"total_operations": len(operations),
		"active_operations": activeCount,
		"completed_operations": completedCount,
		"failed_operations": failedCount,
		"timestamp": time.Now().Unix(),
	}
	
	if ps.logger != nil {
		ps.logger.DebugContext(ctx, "Retrieved operation metrics",
			slog.Int("total", len(operations)),
			slog.Int("active", activeCount))
	}
	
	return metrics, nil
}

