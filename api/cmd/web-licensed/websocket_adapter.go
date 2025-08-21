package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"isxcli/internal/operations"
	ws "isxcli/internal/websocket"
)

var debugMode = os.Getenv("ISX_DEBUG") == "true"

// CommandResponse represents the response from a command execution
type CommandResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// WebSocketAdapter adapts the operations.WebSocketHub interface to the existing ws.Manager
type WebSocketAdapter struct {
	manager *ws.Manager
}

// NewWebSocketAdapter creates a new adapter
func NewWebSocketAdapter(manager *ws.Manager) *WebSocketAdapter {
	return &WebSocketAdapter{
		manager: manager,
	}
}

// BroadcastUpdate implements the operations.WebSocketHub interface
func (w *WebSocketAdapter) BroadcastUpdate(eventType, step, status string, metadata interface{}) {
	// Log the WebSocket message being sent with more detail
	slog.Info("[WEBSOCKET ADAPTER] Sending update",
		slog.String("type", eventType),
		slog.String("step", step),
		slog.String("status", status))
	if metadata != nil {
		if data, err := json.Marshal(metadata); err == nil {
			slog.Info("[WEBSOCKET ADAPTER] Metadata", slog.String("data", string(data)))
		}
	}
	
	// Ensure stdout is flushed for log visibility
	os.Stdout.Sync()
	
	// The event types are already in frontend format now (operation:progress, etc)
	// So we just use them directly
	frontendEventType := eventType
	
	// Transform step ID from backend to frontend format
	frontendStage := step
	stageMapping := map[string]string{
		"scraping":   "scrape",
		"processing": "process",
		"indices":    "index",
		"analysis":   "complete",
	}
	if mapped, ok := stageMapping[step]; ok {
		frontendStage = mapped
	}
	
	// Transform metadata to include frontend step ID
	if metadataMap, ok := metadata.(map[string]interface{}); ok {
		// Add the frontend step ID to metadata if not present
		if _, hasStage := metadataMap["step"]; !hasStage && frontendStage != "" {
			metadataMap["step"] = frontendStage
		} else if existingStage, hasStage := metadataMap["step"].(string); hasStage {
			// Transform existing step ID
			if mapped, ok := stageMapping[existingStage]; ok {
				metadataMap["step"] = mapped
			}
		}
		metadata = metadataMap
	}
	
	// Use the manager to broadcast the update
	w.manager.Broadcast(ws.Message{
		Type:    frontendEventType,
		Step:    frontendStage,
		Message: status,
		Data:    metadata,
	})
}

// OperationLogger adapts the common.Logger to the operations.Logger interface
type OperationLogger struct {
	source    string
	wsManager *ws.Manager
}

// NewOperationLogger creates a new operation logger
func NewOperationLogger(source string, wsManager *ws.Manager) *OperationLogger {
	return &OperationLogger{
		source:    source,
		wsManager: wsManager,
	}
}

// Debug logs a debug message
func (l *OperationLogger) Debug(format string, v ...interface{}) {
	if debugMode {
		slog.Info("[DEBUG] [%s] %s", l.source, fmt.Sprintf(format, v...))
	}
}

// Info logs an info message
func (l *OperationLogger) Info(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	slog.Info("[INFO] [%s] %s", l.source, message)
	if l.wsManager != nil {
		l.wsManager.SendLog(ws.LevelInfo, message, map[string]interface{}{
			"source": l.source,
		})
	}
}

// Warn logs a warning message
func (l *OperationLogger) Warn(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	slog.Info("[WARN] [%s] %s", l.source, message)
	if l.wsManager != nil {
		l.wsManager.SendLog(ws.LevelWarning, message, map[string]interface{}{
			"source": l.source,
		})
	}
}

// Error logs an error message
func (l *OperationLogger) Error(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	slog.Info("[ERROR] [%s] %s", l.source, message)
	if l.wsManager != nil {
		l.wsManager.SendLog(ws.LevelError, message, map[string]interface{}{
			"source": l.source,
		})
	}
}

// OperationEventHandler handles operation events and converts them to the existing format
type OperationEventHandler struct {
	manager *operations.Manager
}

// NewOperationEventHandler creates a new event handler
func NewOperationEventHandler(manager *operations.Manager) *OperationEventHandler {
	return &OperationEventHandler{
		manager: manager,
	}
}

// ConvertOperationResponse converts a operation response to the existing CommandResponse format
func ConvertOperationResponse(resp *operations.OperationResponse) CommandResponse {
	return CommandResponse{
		Success: resp.Status == operations.OperationStatusCompleted,
		Output:  fmt.Sprintf("operation completed with status: %s", resp.Status),
		Error:   resp.Error,
	}
}

// SendOperationUpdate sends a operation update in the format expected by the frontend
func SendOperationUpdate(hub operations.WebSocketHub, pipelineID string, resp *operations.OperationResponse) {
	// Send overall operation status
	hub.BroadcastUpdate(ws.TypeOperationStatus, "", "", map[string]interface{}{
		"pipeline_id": pipelineID,
		"status":      string(resp.Status),
		"duration":    resp.Duration.Seconds(),
		"steps":      convertStepStates(resp.Steps),
	})
	
	// Send individual step updates
	for stageID, step := range resp.Steps {
		hub.BroadcastUpdate(ws.TypePipelineProgress, "", "", map[string]interface{}{
			"pipeline_id": pipelineID,
			"step":       stageID,
			"status":      string(step.Status),
			"progress":    step.Progress,
			"message":     step.Message,
			"metadata":    step.Metadata,
		})
	}
}

// convertStepStates converts step states to a format suitable for JSON
func convertStepStates(steps map[string]*operations.StepState) map[string]interface{} {
	result := make(map[string]interface{})
	for id, step := range steps {
		result[id] = map[string]interface{}{
			"name":       step.Name,
			"status":     string(step.Status),
			"progress":   step.Progress,
			"message":    step.Message,
			"start_time": step.StartTime,
			"end_time":   step.EndTime,
			"duration":   step.Duration().Seconds(),
			"error":      formatError(step.Error),
			"metadata":   step.Metadata,
		}
	}
	return result
}

// formatError formats an error for JSON serialization
func formatError(err error) interface{} {
	if err == nil {
		return nil
	}
	return map[string]string{
		"message": err.Error(),
	}
}

// MonitorOperationProgress monitors a operation and sends progress updates
func MonitorOperationProgress(hub operations.WebSocketHub, pipelineID string, manager *operations.Manager) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for range ticker.C {
		state, err := manager.GetOperation(pipelineID)
		if err != nil {
			// operation no longer exists
			return
		}
		
		// Send progress updates for active steps
		for _, step := range state.GetActiveStages() {
			// Send in the format the frontend expects for progress messages
			hub.BroadcastUpdate(ws.TypeProgress, "", "", map[string]interface{}{
				"step":       step.ID,
				"percentage":  step.Progress,
				"message":     step.Message,
				"eta":         nil, // Could calculate ETA if needed
				"metadata":    step.Metadata,
			})
		}
		
		// Check if operation is complete
		if state.Status == operations.OperationStatusCompleted || 
		   state.Status == operations.OperationStatusFailed ||
		   state.Status == operations.OperationStatusCancelled {
			// Send final status
			hub.BroadcastUpdate(ws.TypePipelineComplete, "", "", map[string]interface{}{
				"pipeline_id": pipelineID,
				"status":      string(state.Status),
				"duration":    state.Duration().Seconds(),
			})
			return
		}
	}
}

// ParseWebSocketMessage attempts to parse a WebSocket message from command output
func ParseWebSocketMessage(line string) (map[string]interface{}, error) {
	// Look for JSON between markers
	startMarker := "[WEBSOCKET_"
	
	startIdx := strings.Index(line, startMarker)
	if startIdx == -1 {
		return nil, fmt.Errorf("no WebSocket marker found")
	}
	
	// Find the end of the message type
	typeEndIdx := strings.Index(line[startIdx:], "]")
	if typeEndIdx == -1 {
		return nil, fmt.Errorf("no closing bracket for message type")
	}
	
	// Extract message type
	messageType := line[startIdx+len("[WEBSOCKET_") : startIdx+typeEndIdx]
	
	// Find JSON content
	jsonStartIdx := startIdx + typeEndIdx + 1
	jsonEndIdx := strings.LastIndex(line, "}")
	if jsonEndIdx == -1 || jsonEndIdx < jsonStartIdx {
		// No JSON content, just type
		return map[string]interface{}{
			"type": messageType,
		}, nil
	}
	
	// Extract and parse JSON
	jsonContent := line[jsonStartIdx : jsonEndIdx+1]
	jsonContent = strings.TrimSpace(jsonContent)
	
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	
	data["type"] = messageType
	return data, nil
}