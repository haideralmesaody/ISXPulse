package license

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// StateFile represents a temporary license validation state
type StateFile struct {
	ValidatedAt  time.Time `json:"validated_at"`
	ValidUntil   time.Time `json:"valid_until"`
	Signature    string    `json:"signature"`
}

// stateFileSecret is used for HMAC signature generation
// In production, this should be generated dynamically or stored securely
const stateFileSecret = "ISX-State-File-Secret-2024-Do-Not-Share"

// CreateStateFile creates a new state file for license validation bypass
func (m *Manager) CreateStateFile(stateFilePath string) error {
	// Log the requested state file path
	ctx := context.Background()
	m.logInfo(ctx, "state_file_creation", "Creating license validation state file",
		slog.String("requested_path", stateFilePath),
	)
	
	// Create state file data
	now := time.Now()
	state := StateFile{
		ValidatedAt: now,
		ValidUntil:  now.Add(5 * time.Minute), // Valid for 5 minutes
	}
	
	// Generate signature
	state.Signature = generateStateSignature(state)
	
	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state file: %v", err)
	}
	
	// Write to file with restricted permissions
	if err := os.WriteFile(stateFilePath, data, 0600); err != nil {
		m.logError(ctx, "state_file_creation", "Failed to write state file",
			slog.String("path", stateFilePath),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to write state file: %v", err)
	}
	
	// Log state file creation per CLAUDE.md standards
	m.logInfo(ctx, "state_file_created", "License validation state file created successfully",
		slog.String("valid_until", state.ValidUntil.Format(time.RFC3339)),
		slog.String("path", stateFilePath),
		slog.Int("size_bytes", len(data)),
	)
	
	return nil
}

// ValidateStateFile checks if a state file is valid for the current machine
func (m *Manager) ValidateStateFile(stateFilePath string) (bool, error) {
	// Log validation attempt
	ctx := context.Background()
	m.logDebug(ctx, "state_file_validation_start", "Validating state file",
		slog.String("path", stateFilePath),
	)
	
	// Check if file exists
	if _, err := os.Stat(stateFilePath); os.IsNotExist(err) {
		m.logDebug(ctx, "state_file_validation", "State file does not exist",
			slog.String("path", stateFilePath),
		)
		return false, nil // File doesn't exist, not an error
	}
	
	// Read state file
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read state file: %v", err)
	}
	
	// Parse JSON
	var state StateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return false, fmt.Errorf("failed to parse state file: %v", err)
	}
	
	// Machine ID validation removed - licenses are now portable
	
	// Check validity period
	now := time.Now()
	if now.Before(state.ValidatedAt) || now.After(state.ValidUntil) {
		// Log state file expiration per CLAUDE.md standards
		m.logWarn(context.Background(), "state_file_validation", "State file expired",
			slog.String("validated_at", state.ValidatedAt.Format(time.RFC3339)),
			slog.String("valid_until", state.ValidUntil.Format(time.RFC3339)),
			slog.String("current_time", now.Format(time.RFC3339)),
		)
		return false, nil
	}
	
	// Verify signature
	expectedSignature := generateStateSignature(state)
	if state.Signature != expectedSignature {
		// Log signature mismatch per CLAUDE.md standards
		m.logError(context.Background(), "state_file_validation", "State file signature mismatch - possible tampering")
		return false, fmt.Errorf("invalid state file signature")
	}
	
	// State file is valid - log success per CLAUDE.md standards
	m.logInfo(context.Background(), "state_file_validation", "State file validated successfully",
		slog.String("remaining_validity", state.ValidUntil.Sub(now).String()),
	)
	
	return true, nil
}

// GetMachineID is deprecated - machine ID is no longer used
func (m *Manager) GetMachineID() string {
	return ""
}

// generateStateSignature creates an HMAC-SHA256 signature for the state file
func generateStateSignature(state StateFile) string {
	// Create signature data without the signature field
	signatureData := fmt.Sprintf("%s|%s",
		state.ValidatedAt.Format(time.RFC3339),
		state.ValidUntil.Format(time.RFC3339))
	
	// Generate HMAC
	h := hmac.New(sha256.New, []byte(stateFileSecret))
	h.Write([]byte(signatureData))
	
	return hex.EncodeToString(h.Sum(nil))
}

// CleanupStateFile removes a state file if it exists
func CleanupStateFile(stateFilePath string) error {
	if _, err := os.Stat(stateFilePath); err == nil {
		return os.Remove(stateFilePath)
	}
	return nil // File doesn't exist, nothing to clean up
}