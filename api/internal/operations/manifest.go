package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PipelineManifest tracks the state and available data for a pipeline operation
// Following CLAUDE.md: Single Source of Truth for pipeline state
type PipelineManifest struct {
	mu sync.RWMutex `json:"-"`
	
	// Identity
	ID          string    `json:"id"`
	OperationID string    `json:"operation_id"`
	StartTime   time.Time `json:"start_time"`
	
	// Configuration
	FromDate string                 `json:"from_date,omitempty"`
	ToDate   string                 `json:"to_date,omitempty"`
	Mode     string                 `json:"mode"`
	Config   map[string]interface{} `json:"config,omitempty"`
	
	// Available data tracking
	AvailableData map[string]*DataInfo `json:"available_data"`
	
	// Execution tracking
	CompletedStages []StageExecution `json:"completed_stages"`
	
	// Current status
	Status      string    `json:"status"` // "pending", "running", "completed", "failed"
	LastUpdated time.Time `json:"last_updated"`
	Error       string    `json:"error,omitempty"`
}

// DataInfo tracks information about available data
type DataInfo struct {
	Type        string    `json:"type"`         // Type of data (e.g., "excel_files")
	Location    string    `json:"location"`     // Directory where data is stored
	FileCount   int       `json:"file_count"`   // Number of files
	FilePattern string    `json:"file_pattern"` // Pattern of files (e.g., "*.xls")
	TotalSize   int64     `json:"total_size"`   // Total size in bytes
	Files       []string  `json:"files"`        // List of file names
	CreatedAt   time.Time `json:"created_at"`   // When this data was created
	CreatedBy   string    `json:"created_by"`   // Which stage created this
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}

// StageExecution tracks the execution of a single stage
type StageExecution struct {
	StageID    string    `json:"stage_id"`
	StageName  string    `json:"stage_name"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Duration   string    `json:"duration"`
	Status     string    `json:"status"` // "completed", "failed", "skipped"
	OutputData []string  `json:"output_data"` // Types of data produced
	Error      string    `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewPipelineManifest creates a new pipeline manifest
func NewPipelineManifest(operationID string, fromDate, toDate string) *PipelineManifest {
	return &PipelineManifest{
		ID:              fmt.Sprintf("manifest-%d", time.Now().Unix()),
		OperationID:     operationID,
		StartTime:       time.Now(),
		FromDate:        fromDate,
		ToDate:          toDate,
		Mode:            "full",
		AvailableData:   make(map[string]*DataInfo),
		CompletedStages: []StageExecution{},
		Status:          "pending",
		LastUpdated:     time.Now(),
	}
}

// HasData checks if a specific type of data is available
func (m *PipelineManifest) HasData(dataType string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.AvailableData[dataType]
	return exists
}

// GetData returns information about available data
func (m *PipelineManifest) GetData(dataType string) (*DataInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, exists := m.AvailableData[dataType]
	return data, exists
}

// AddData records newly available data
func (m *PipelineManifest) AddData(dataType string, info *DataInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	info.CreatedAt = time.Now()
	m.AvailableData[dataType] = info
	m.LastUpdated = time.Now()
}

// RecordStageStart records the start of a stage execution
func (m *PipelineManifest) RecordStageStart(stageID, stageName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if stage already exists (in case of retry)
	for i, stage := range m.CompletedStages {
		if stage.StageID == stageID {
			// Update existing entry
			m.CompletedStages[i].StartTime = time.Now()
			m.CompletedStages[i].Status = "running"
			m.LastUpdated = time.Now()
			return
		}
	}
	
	// Add new stage execution
	m.CompletedStages = append(m.CompletedStages, StageExecution{
		StageID:   stageID,
		StageName: stageName,
		StartTime: time.Now(),
		Status:    "running",
	})
	m.LastUpdated = time.Now()
}

// RecordStageCompletion records the completion of a stage
func (m *PipelineManifest) RecordStageCompletion(stageID string, outputData []string, metadata map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, stage := range m.CompletedStages {
		if stage.StageID == stageID {
			m.CompletedStages[i].EndTime = time.Now()
			m.CompletedStages[i].Duration = time.Since(stage.StartTime).String()
			m.CompletedStages[i].Status = "completed"
			m.CompletedStages[i].OutputData = outputData
			m.CompletedStages[i].Metadata = metadata
			break
		}
	}
	m.LastUpdated = time.Now()
}

// RecordStageFailure records a stage failure
func (m *PipelineManifest) RecordStageFailure(stageID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, stage := range m.CompletedStages {
		if stage.StageID == stageID {
			m.CompletedStages[i].EndTime = time.Now()
			m.CompletedStages[i].Duration = time.Since(stage.StartTime).String()
			m.CompletedStages[i].Status = "failed"
			m.CompletedStages[i].Error = err.Error()
			break
		}
	}
	m.Status = "failed"
	m.Error = fmt.Sprintf("Stage %s failed: %v", stageID, err)
	m.LastUpdated = time.Now()
}

// IsStageCompleted checks if a stage has been completed
func (m *PipelineManifest) IsStageCompleted(stageID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, stage := range m.CompletedStages {
		if stage.StageID == stageID && stage.Status == "completed" {
			return true
		}
	}
	return false
}

// ScanDataDirectory scans a directory and updates available data
func (m *PipelineManifest) ScanDataDirectory(dataType, location, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if directory exists
	if _, err := os.Stat(location); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", location)
	}
	
	// Find matching files
	searchPattern := filepath.Join(location, pattern)
	files, err := filepath.Glob(searchPattern)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}
	
	// Calculate total size and get file names
	var totalSize int64
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileNames = append(fileNames, filepath.Base(file))
		}
	}
	
	// Update or add data info
	m.AvailableData[dataType] = &DataInfo{
		Type:        dataType,
		Location:    location,
		FileCount:   len(fileNames),
		FilePattern: pattern,
		TotalSize:   totalSize,
		Files:       fileNames,
		CreatedAt:   time.Now(),
	}
	
	m.LastUpdated = time.Now()
	return nil
}

// SaveToFile saves the manifest to a JSON file
func (m *PipelineManifest) SaveToFile(filepath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}
	
	return nil
}

// LoadFromFile loads a manifest from a JSON file
func LoadManifestFromFile(filepath string) (*PipelineManifest, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	
	var manifest PipelineManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	
	return &manifest, nil
}

// Clone creates a deep copy of the manifest
func (m *PipelineManifest) Clone() *PipelineManifest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Use JSON marshaling for deep copy
	data, _ := json.Marshal(m)
	var clone PipelineManifest
	json.Unmarshal(data, &clone)
	
	return &clone
}

// GetProgress calculates overall progress percentage
func (m *PipelineManifest) GetProgress() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.CompletedStages) == 0 {
		return 0
	}
	
	completed := 0
	for _, stage := range m.CompletedStages {
		if stage.Status == "completed" {
			completed++
		}
	}
	
	// Assuming 4 total stages for now (scraping, processing, indices, liquidity)
	totalStages := 4
	return (completed * 100) / totalStages
}