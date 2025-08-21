package operations

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"isxcli/internal/liquidity"
)

// ScrapingStage handles the scraping process
type ScrapingStage struct {
	BaseStage
	executableDir string
	logger        *slog.Logger
	options       *StageOptions
}

// NewScrapingStage creates a new scraping Step
func NewScrapingStage(executableDir string, logger *slog.Logger, options *StageOptions) *ScrapingStage {
	if options == nil {
		options = &StageOptions{}
	}

	// Create logger with Step context
	if logger != nil {
		logger = logger.With(slog.String("Step", StageIDScraping))
		logger.Info("Scraping Step initialized",
			slog.String("executable_dir", executableDir))
	}

	return &ScrapingStage{
		BaseStage:     NewBaseStage(StageIDScraping, StageNameScraping, nil),
		executableDir: executableDir,
		logger:        logger,
		options:       options,
	}
}

// Execute runs the scraper to download ISX daily reports
func (s *ScrapingStage) Execute(ctx context.Context, state *OperationState) error {
	StepState := state.GetStage(s.ID())

	// Log Step execution start
	if s.logger != nil {
		s.logger.Info("Starting scraping Step",
			slog.String("pipeline_id", state.ID))
	}

	// Check license if required
	if s.options.LicenseChecker != nil && s.options.LicenseChecker.RequiresLicense() {
		if err := s.options.LicenseChecker.CheckLicense(); err != nil {
			if s.logger != nil {
				s.logger.Error("License check failed",
					slog.String("error", err.Error()))
			}
			return fmt.Errorf("license check failed: %w", err)
		}
	}

	s.updateProgress(state.ID, StepState, 2, "Starting scraper...")

	scraperPath := filepath.Join(s.executableDir, "scraper.exe")
	if _, err := os.Stat(scraperPath); err != nil {
		if s.logger != nil {
			s.logger.Error("Scraper executable not found",
				slog.String("path", scraperPath),
				slog.String("error", err.Error()))
		}
		return fmt.Errorf("scraper.exe not found: %w", err)
	}

	// Build command arguments
	args := s.buildScraperArgs(state)
	cmd := exec.CommandContext(ctx, scraperPath, args...)
	cmd.Dir = s.executableDir

	s.updateProgress(state.ID, StepState, 3, "Running scraper...")

	// Execute with progress tracking if enabled
	if s.options.EnableProgress && s.options.WebSocketManager != nil {
		if err := s.executeWithProgress(ctx, cmd, state.ID, StepState); err != nil {
			if s.logger != nil {
				s.logger.Error("Scraper execution failed",
					slog.String("error", err.Error()))
			}
			return fmt.Errorf("scraper failed: %w", err)
		}
	} else {
		output, err := cmd.CombinedOutput()
		if err != nil {
			if s.logger != nil {
				s.logger.Error("Scraper execution failed",
					slog.String("error", err.Error()),
					slog.String("output", string(output)))
			}
			return fmt.Errorf("scraper failed: %w, output: %s", err, string(output))
		}
	}

	s.updateProgress(state.ID, StepState, 100, "Scraping completed")
	return nil
}

// buildScraperArgs builds command line arguments from operation state
func (s *ScrapingStage) buildScraperArgs(state *OperationState) []string {
	args := []string{}

	// Log all available configuration for debugging
	if s.logger != nil {
		allConfig := make(map[string]interface{})
		// Show all config keys for debugging
		keysToCheck := []string{ContextKeyFromDate, ContextKeyToDate, ContextKeyMode, "from", "to", "from_date", "to_date"}
		for _, key := range keysToCheck {
			if val, exists := state.GetConfig(key); exists {
				allConfig[key] = val
			}
		}
		s.logger.Info("Available configuration in state",
			slog.Any("config", allConfig),
			slog.String("operation_id", state.ID))
	}

	// Get configuration from operation state
	var actualFromDate string
	var actualToDate string
	
	// Handle from_date - APPLY BUFFER HERE to extend range backwards
	if fromDateI, exists := state.GetConfig(ContextKeyFromDate); exists {
		if fromDate, ok := fromDateI.(string); ok && fromDate != "" {
			actualFromDate = fromDate
			
			// Add 7-day buffer BEFORE from_date for edge case handling
			// This extends the range to capture more historical data
			if parsedDate, err := time.Parse("2006-01-02", fromDate); err == nil {
				bufferDate := parsedDate.AddDate(0, 0, -7) // 7 days before from_date
				bufferDateStr := bufferDate.Format("2006-01-02")
				
				args = append(args, "--from", bufferDateStr)
				
				if s.logger != nil {
					s.logger.Info("Added buffer to from date for edge case handling",
						slog.String("original_from_date", actualFromDate),
						slog.String("buffer_from_date", bufferDateStr),
						slog.String("buffer_days", "7"))
				}
			} else {
				// Fallback if date parsing fails
				args = append(args, "--from", fromDate)
				if s.logger != nil {
					s.logger.Warn("Could not parse from_date for buffer, using original",
						slog.String("from_date", fromDate),
						slog.String("error", err.Error()))
				}
			}
		} else {
			if s.logger != nil {
				s.logger.Warn("FromDate exists but not a string or empty",
					slog.Any("value", fromDateI),
					slog.String("type", fmt.Sprintf("%T", fromDateI)))
			}
		}
	} else {
		if s.logger != nil {
			s.logger.Warn("No from_date found in state config",
				slog.String("key_checked", ContextKeyFromDate))
		}
	}

	// Handle to_date - NO BUFFER, use actual to_date
	if toDateI, exists := state.GetConfig(ContextKeyToDate); exists {
		if toDate, ok := toDateI.(string); ok && toDate != "" {
			actualToDate = toDate
			args = append(args, "--to", toDate)
			
			// Pass actual dates for progress calculation
			args = append(args, "--actual-from", actualFromDate)
			args = append(args, "--actual-to", actualToDate)
			
			if s.logger != nil {
				s.logger.Info("Added to date to scraper args",
					slog.String("to_date", toDate),
					slog.String("actual_from", actualFromDate),
					slog.String("actual_to", actualToDate),
					slog.String("key_used", ContextKeyToDate))
			}
		} else {
			if s.logger != nil {
				s.logger.Warn("ToDate exists but not a string or empty",
					slog.Any("value", toDateI),
					slog.String("type", fmt.Sprintf("%T", toDateI)))
			}
		}
	} else {
		if s.logger != nil {
			s.logger.Warn("No to_date found in state config",
				slog.String("key_checked", ContextKeyToDate))
		}
	}

	if modeI, exists := state.GetConfig(ContextKeyMode); exists {
		if mode, ok := modeI.(string); ok && mode != "" {
			args = append(args, "--mode", mode)
		} else {
			args = append(args, "--mode", "full")
		}
	} else {
		args = append(args, "--mode", "full")
	}

	if s.logger != nil {
		s.logger.Info("Final scraper args",
			slog.Any("args", args),
			slog.Int("arg_count", len(args)),
			slog.String("operation_id", state.ID))
	}
	return args
}

// executeWithProgress runs the command with real-time progress tracking
func (s *ScrapingStage) executeWithProgress(ctx context.Context, cmd *exec.Cmd, operationID string, StepState *StepState) error {
	// Extract dates from the command args for metadata
	var fromDate, toDate, actualFromDate, actualToDate string
	for i, arg := range cmd.Args {
		if arg == "--from" && i+1 < len(cmd.Args) {
			fromDate = cmd.Args[i+1]
		} else if arg == "--to" && i+1 < len(cmd.Args) {
			toDate = cmd.Args[i+1]  // This is the buffer-extended to_date
		} else if arg == "--actual-from" && i+1 < len(cmd.Args) {
			actualFromDate = cmd.Args[i+1]  // User's actual requested from_date
		} else if arg == "--actual-to" && i+1 < len(cmd.Args) {
			actualToDate = cmd.Args[i+1]  // User's actual requested to_date
		}
	}
	
	// Use actual dates for display and progress calculation
	displayFromDate := actualFromDate
	if displayFromDate == "" {
		displayFromDate = fromDate  // Fallback if no actual-from specified
	}
	displayToDate := actualToDate
	if displayToDate == "" {
		displayToDate = toDate  // Fallback if no actual-to specified
	}
	
	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start scraper: %w", err)
	}

	// Simple progress tracking
	type ScrapingState string
	const (
		StateInitializing ScrapingState = "initializing"
		StateScanning     ScrapingState = "scanning"
		StateDownloading  ScrapingState = "downloading"
		StateCompleting   ScrapingState = "completing"
		StateCompleted    ScrapingState = "completed"
		StateStopped      ScrapingState = "stopped"
	)

	var (
		currentState    ScrapingState = StateInitializing
		filesProcessed  int
		currentFile     string
		startTime       = time.Now()
		currentPage     int
		downloadedFiles []string // Track all downloaded files
		skippedFiles    []string // Track skipped files (holidays)
		expectedFiles   int      // Total trading days in range
		seenFiles       = make(map[string]bool) // Track unique files to prevent double counting
	)
	
	// Calculate expected files (trading days in ACTUAL date range, not buffer)
	if displayFromDate != "" && displayToDate != "" {
		start, err1 := time.Parse("2006-01-02", displayFromDate)  // Use actual from_date (start of range)
		end, err2 := time.Parse("2006-01-02", displayToDate)      // Use actual to_date (end of range)
		if err1 == nil && err2 == nil {
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				// Iraq weekend is Friday (5) and Saturday (6)
				if d.Weekday() != time.Friday && d.Weekday() != time.Saturday {
					expectedFiles++
				}
			}
		}
	}

	progressChan := make(chan string, 100)
	errChan := make(chan error, 2)
	
	// Helper to check if a file date is within the ACTUAL requested range (not buffer)
	isFileInRange := func(filename string) bool {
		// Always use the display from date (actual user-requested date)
		if displayFromDate == "" || displayToDate == "" {
			return true // If no range specified, count all files
		}
		
		// Try to extract date from filename (handle both formats)
		var dateStr string
		if matches := regexp.MustCompile(`(\d{4})\s+(\d{2})\s+(\d{2})`).FindStringSubmatch(filename); len(matches) > 3 {
			dateStr = fmt.Sprintf("%s-%s-%s", matches[1], matches[2], matches[3])
		} else if matches := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`).FindStringSubmatch(filename); len(matches) > 0 {
			dateStr = matches[0]
		} else {
			return false // Can't extract date, don't count
		}
		
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return false
		}
		
		start, _ := time.Parse("2006-01-02", displayFromDate)
		end, _ := time.Parse("2006-01-02", displayToDate)
		
		return !fileDate.Before(start) && !fileDate.After(end)
	}

	// Simple helper to calculate processing speed
	calculateSpeed := func() float64 {
		elapsed := time.Since(startTime).Minutes()
		if elapsed > 0 && filesProcessed > 0 {
			return float64(filesProcessed) / elapsed
		}
		return 0
	}

	// Simple progress calculation based on state and time
	calculateProgress := func() int {
		switch currentState {
		case StateInitializing:
			return 5
		case StateScanning:
			return 10
		case StateDownloading:
			// If we know expected files, use accurate percentage
			if expectedFiles > 0 && filesProcessed > 0 {
				// 10% for scanning, 80% for downloading (10-90%), so scale accordingly
				progress := 10 + int(float64(filesProcessed)/float64(expectedFiles)*80)
				if progress > 90 {
					return 90
				}
				return progress
			}
			// Fallback: Smooth progress from 10-90% based on files or time
			if filesProcessed > 0 {
				// Approximately 3% per file, capped at 90%
				progress := 10 + filesProcessed*3
				if progress > 90 {
					return 90
				}
				return progress
			}
			// Time-based fallback: 8% per minute
			elapsed := time.Since(startTime).Minutes()
			progress := 10 + int(elapsed*8)
			if progress > 90 {
				return 90
			}
			return progress
		case StateCompleting:
			return 95
		case StateCompleted:
			return 100
		case StateStopped:
			// If we know expected files and processed them all, return 100
			if expectedFiles > 0 && filesProcessed >= expectedFiles {
				return 100
			}
			// Keep whatever progress we had
			if filesProcessed > 0 {
				if expectedFiles > 0 {
					progress := 10 + int(float64(filesProcessed)/float64(expectedFiles)*80)
					if progress > 90 {
						return 90
					}
					return progress
				}
				progress := 10 + filesProcessed*3
				if progress > 90 {
					return 90
				}
				return progress
			}
			return 10
		default:
			return 0
		}
	}

	// Update metadata helper
	updateMetadata := func() {
		StepState.Metadata["status"] = string(currentState)
		StepState.Metadata["files_processed"] = filesProcessed
		StepState.Metadata["current_file"] = currentFile
		StepState.Metadata["speed"] = calculateSpeed()
		StepState.Metadata["started_at"] = startTime.Format(time.RFC3339)
		// Add total expected files if we know them
		if expectedFiles > 0 {
			StepState.Metadata["total_files"] = expectedFiles
		}
		// Add dates for progress bar (use actual requested dates, not buffer dates)
		if displayFromDate != "" {
			StepState.Metadata["from_date"] = displayFromDate
		}
		if displayToDate != "" {
			StepState.Metadata["to_date"] = displayToDate
		}
		// Add downloaded files list
		if len(downloadedFiles) > 0 {
			StepState.Metadata["downloaded_files"] = downloadedFiles
		}
		// Add skipped files list (holidays)
		if len(skippedFiles) > 0 {
			StepState.Metadata["skipped_files"] = skippedFiles
		}
		// Add phase for frontend
		StepState.Metadata["phase"] = string(currentState)
	}

	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			progressChan <- line
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("stdout scan error: %w", err)
		}
		close(progressChan)
	}()

	// Read stderr in goroutine with enhanced error capture
	go func() {
		scanner := bufio.NewScanner(stderr)
		// Increase buffer size to capture more error output
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var errOutput strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			errOutput.WriteString(line + "\n")
			// Also log each stderr line immediately for debugging
			if s.logger != nil && line != "" {
				s.logger.Debug("Scraper stderr line",
					slog.String("line", line),
					slog.String("step", "scraping"))
			}
		}
		if errOutput.Len() > 0 {
			fullError := errOutput.String()
			// Log full error for debugging
			if s.logger != nil {
				s.logger.Error("Scraper complete stderr output",
					slog.String("stderr", fullError),
					slog.Int("stderr_length", len(fullError)),
					slog.String("step", "scraping"))
			}
			errChan <- fmt.Errorf("stderr output: %s", fullError)
		}
	}()

	// Track last activity time to detect if scraper is stuck
	lastActivityTime := time.Now()
	activityTimeout := 30 * time.Second // Reduced timeout for safety only

	// Process output lines
	for {
		select {
		case line, ok := <-progressChan:
			if !ok {
				// Channel closed, scraper finished
				goto waitForCompletion
			}

			// Update last activity time
			lastActivityTime = time.Now()

			if s.logger != nil {
				s.logger.Debug("Scraper output",
					slog.String("line", line))
			}

			// Try to parse as JSON first (for structured logs)
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
				// Successfully parsed JSON log
				msg, _ := logEntry["msg"].(string)

				switch {
				case strings.Contains(msg, "SCRAPER_COMPLETE"):
					// Scraper signals all files are already present
					currentState = StateCompleted
					updateMetadata()
					s.updateProgress(operationID, StepState, 100, "✅ All required files already exist")
					if s.logger != nil {
						s.logger.Info("Scraper signaled completion - all files exist",
							slog.Int("files_processed", filesProcessed))
					}
					goto waitForCompletion
					
				case strings.Contains(msg, "Expected files to download"), strings.Contains(msg, "Total expected files"):
					// Move to scanning state
					currentState = StateScanning
					updateMetadata()
					s.updateProgress(operationID, StepState, calculateProgress(), "Scanning for files...")
					if s.logger != nil {
						s.logger.Info("Started scanning",
							slog.String("Step", s.ID()))
					}

				case strings.Contains(msg, "Scraping page"):
					// Stay in current state, just update page number
					if page, ok := logEntry["page"].(float64); ok {
						currentPage = int(page)
						// Don't change progress, just update message
						s.updateProgress(operationID, StepState, calculateProgress(), fmt.Sprintf("Scanning page %d", currentPage))
					}
				
				case strings.Contains(msg, "No data for") || strings.Contains(msg, "No files found") || 
					 strings.Contains(msg, "Skipping date") || strings.Contains(msg, "not found"):
					// File was skipped (holiday or no data)
					// Try to extract date from the message
					if matches := regexp.MustCompile(`(\d{4})\s+(\d{2})\s+(\d{2})`).FindStringSubmatch(msg); len(matches) > 3 {
						skippedDate := fmt.Sprintf("%s %s %s", matches[1], matches[2], matches[3])
						// Check if it's in our date range before adding
						if isFileInRange(skippedDate) {
							skippedFiles = append(skippedFiles, skippedDate)
							updateMetadata()
							if s.logger != nil {
								s.logger.Info("Detected skipped date (holiday)",
									slog.String("date", skippedDate),
									slog.String("step", s.ID()))
							}
						}
					}

				case strings.Contains(msg, "Downloading file") && strings.Contains(msg, "of"):
					// Move to downloading state
					currentState = StateDownloading
					
					// Extract filename if available
					if fileName, ok := logEntry["file"].(string); ok {
						currentFile = fileName
						// Normalize filename for deduplication
						normalizedName := filepath.Base(strings.TrimSpace(fileName))
						
						// Debug logging
						slog.DebugContext(ctx, "Checking file for duplicate (Downloading)", 
							"original_filename", fileName,
							"normalized_name", normalizedName,
							"already_seen", seenFiles[normalizedName],
							"current_count", filesProcessed,
							"in_range", isFileInRange(fileName))
						
						// Only count and add to list if file is in date range AND not already seen
						if isFileInRange(fileName) && !seenFiles[normalizedName] {
							seenFiles[normalizedName] = true
							filesProcessed++
							// Add to downloaded files list
							downloadedFiles = append(downloadedFiles, fileName)
							slog.InfoContext(ctx, "File counted (Downloading)", 
								"normalized_name", normalizedName,
								"new_count", filesProcessed)
						} else if seenFiles[normalizedName] {
							slog.InfoContext(ctx, "Duplicate file skipped (Downloading)", 
								"normalized_name", normalizedName,
								"count_unchanged", filesProcessed)
						}
					}
					// Removed fallback increment to prevent counting when no filename
					
					updateMetadata()
					message := fmt.Sprintf("Processing file %d", filesProcessed)
					if currentFile != "" {
						message = fmt.Sprintf("Downloading: %s", currentFile)
					}
					if expectedFiles > 0 {
						message = fmt.Sprintf("Downloading: %s (%d/%d)", 
							filepath.Base(currentFile), filesProcessed, expectedFiles)
					}
					s.updateProgress(operationID, StepState, calculateProgress(), message)

				case strings.Contains(msg, "Already exists"):
					// File exists from pre-scan
					currentState = StateDownloading
					
					// Extract filename from JSON if available
					if fileName, ok := logEntry["file"].(string); ok {
						currentFile = fileName
						// Normalize filename for deduplication
						normalizedName := filepath.Base(strings.TrimSpace(fileName))
						
						// Always count pre-scan existing files
						if !seenFiles[normalizedName] {
							seenFiles[normalizedName] = true
							filesProcessed++
							downloadedFiles = append(downloadedFiles, fileName)
							
							slog.InfoContext(ctx, "Existing file counted from pre-scan", 
								"file", normalizedName,
								"count", filesProcessed,
								"expected", expectedFiles)
						}
						
						updateMetadata()
						message := fmt.Sprintf("Found existing file (%d/%d)", filesProcessed, expectedFiles)
						s.updateProgress(operationID, StepState, calculateProgress(), message)
					}
					
				case strings.Contains(msg, "already exists") && strings.Contains(msg, "of"):
					// File exists during scraping (old format for backward compatibility)
					currentState = StateDownloading
					
					// Extract filename from message if possible
					if matches := regexp.MustCompile(`(\d{4} \d{2} \d{2}) ISX Daily Report\.xlsx`).FindStringSubmatch(msg); len(matches) > 1 {
						currentFile = matches[0]
						// Normalize filename for deduplication
						normalizedName := filepath.Base(strings.TrimSpace(currentFile))
						
						// Debug logging
						slog.DebugContext(ctx, "Checking file for duplicate (Already exists)", 
							"original_filename", currentFile,
							"normalized_name", normalizedName,
							"already_seen", seenFiles[normalizedName],
							"current_count", filesProcessed,
							"in_range", isFileInRange(currentFile))
						
						// Only count and add to list if file is in date range AND not already seen
						if isFileInRange(currentFile) && !seenFiles[normalizedName] {
							seenFiles[normalizedName] = true
							filesProcessed++
							// Add to downloaded files list (already exists counts as downloaded)
							downloadedFiles = append(downloadedFiles, matches[0])
							slog.InfoContext(ctx, "File counted (Already exists)", 
								"normalized_name", normalizedName,
								"new_count", filesProcessed)
						} else if seenFiles[normalizedName] {
							slog.InfoContext(ctx, "Duplicate file skipped (Already exists)", 
								"normalized_name", normalizedName,
								"count_unchanged", filesProcessed)
						}
					}
					// Removed fallback increment to prevent counting when no filename
					
					updateMetadata()
					message := fmt.Sprintf("Processing file %d (exists, skipping)", filesProcessed)
					if expectedFiles > 0 {
						message = fmt.Sprintf("File exists, skipping (%d/%d)", filesProcessed, expectedFiles)
					}
					s.updateProgress(operationID, StepState, calculateProgress(), message)
				}
				continue
			}
			
			// If we get here, the line wasn't valid JSON - skip it
			// All scraper output should now be JSON via slog
			if s.logger != nil {
				s.logger.Debug("Skipping non-JSON line from scraper",
					slog.String("line", line))
			}

		case <-time.After(1 * time.Second):
			// Only check for completely stuck scraper (safety net)
			timeSinceLastActivity := time.Since(lastActivityTime)
			
			// Remove auto-completion - scraper now signals completion explicitly
			// Only keep safety timeout for completely stuck processes
			
			// Check if scraper has been completely inactive for too long
			if timeSinceLastActivity > activityTimeout {
				if s.logger != nil {
					s.logger.Warn("Scraper timeout - no output for 2 minutes",
						slog.Int("files_processed", filesProcessed))
				}
				// Mark as stopped rather than killing immediately
				currentState = StateStopped
				updateMetadata()
				s.updateProgress(operationID, StepState, calculateProgress(), 
					fmt.Sprintf("Stopped after processing %d files", filesProcessed))
				// Kill the scraper process
				if err := cmd.Process.Kill(); err != nil {
					s.logger.Error("Failed to kill stuck scraper process", slog.String("error", err.Error()))
				}
				goto waitForCompletion
			}
		}
	}

waitForCompletion:
	// Wait for command to complete
	err = cmd.Wait()

	// Final status based on what we observed
	if currentState == StateCompleted || filesProcessed > 0 {
		// Ensure we're in completed state and clear transient data
		currentState = StateCompleted
		currentFile = "" // CRITICAL: Clear current file to prevent blue blinking
		
		// Update final metadata
		StepState.Metadata["files_processed"] = filesProcessed
		StepState.Metadata["completed"] = true
		StepState.Metadata["current_file"] = "" // Explicitly clear in metadata
		StepState.Metadata["phase"] = "completed" // Ensure phase is set
		
		if s.logger != nil {
			s.logger.Info("Scraper completed",
				slog.Int("files_processed", filesProcessed),
				slog.String("final_state", string(currentState)))
		}
	}

	// Check if command failed
	if err != nil {
		// Check if we at least processed some files
		if filesProcessed > 0 {
			// Partial success
			if s.logger != nil {
				s.logger.Info("Scraper completed with partial success",
					slog.Int("files_processed", filesProcessed),
					slog.String("error", err.Error()))
			}
			// Don't treat as error if we got some files
			err = nil
		} else {
			// Complete failure
			if s.logger != nil {
				s.logger.Error("Scraper command wait failed",
					slog.String("error", err.Error()),
					slog.String("pipeline_id", operationID),
					slog.String("stage", s.ID()),
					slog.Int("files_processed", filesProcessed),
					slog.Int("expected_files", expectedFiles),
					slog.String("current_state", string(currentState)))
			}
			select {
			case stderr := <-errChan:
				if s.logger != nil {
					s.logger.Error("Scraper stderr captured",
						slog.Any("stderr", stderr),
						slog.String("pipeline_id", operationID))
				}
				return fmt.Errorf("scraper failed: %w, stderr: %v", err, stderr)
			default:
				return fmt.Errorf("scraper failed: %w", err)
			}
		}
	}

	// Final cleanup and progress update
	currentState = StateCompleted
	currentFile = "" // Final insurance against blue blinking
	updateMetadata() // Ensure metadata is fully updated
	
	// Verify files were actually processed
	if filesProcessed == 0 && len(downloadedFiles) == 0 {
		// Double-check downloads folder
		downloadsDir := filepath.Join(s.executableDir, "data", "downloads")
		pattern := filepath.Join(downloadsDir, "*.xlsx")
		existingFiles, _ := filepath.Glob(pattern)
		
		if len(existingFiles) == 0 {
			return fmt.Errorf("no files were downloaded or found")
		}
		
		// Files exist but weren't counted properly - still allow processing
		slog.WarnContext(ctx, "Files exist but weren't counted, proceeding anyway",
			"existing_files", len(existingFiles))
		filesProcessed = len(existingFiles)
	}

	// Update to 100% completion with clean state
	StepState.Metadata["phase"] = "completed"
	StepState.Metadata["current_file"] = ""
	StepState.Metadata["completed"] = true
	s.updateProgress(operationID, StepState, 100, fmt.Sprintf("✅ Scraping completed successfully: %d files processed", filesProcessed))

	return nil
}

// updateProgress updates progress through the centralized StatusBroadcaster
func (s *ScrapingStage) updateProgress(operationID string, StepState *StepState, progress int, message string) {
	StepState.UpdateProgress(float64(progress), message)

	// Use centralized StatusBroadcaster for all updates
	if s.options.StatusBroadcaster != nil {
		// Update through the broadcaster - single source of truth
		// Pass the metadata to ensure frontend receives all the details
		s.options.StatusBroadcaster.UpdateStepWithMetadata(operationID, s.ID(), progress, message, StepState.Metadata)
	}
}

// RequiredInputs returns empty requirements as scraping needs no inputs
func (s *ScrapingStage) RequiredInputs() []DataRequirement {
	return []DataRequirement{} // No inputs needed - scraping is the first step
}

// ProducedOutputs returns the Excel files produced by scraping
func (s *ScrapingStage) ProducedOutputs() []DataOutput {
	return []DataOutput{
		{
			Type:     "excel_files",
			Location: "data/downloads",
			Pattern:  "*.xls",
		},
	}
}

// CanRun always returns true as scraping has no dependencies
func (s *ScrapingStage) CanRun(manifest *PipelineManifest) bool {
	return true // Can always run - no dependencies
}

// ProcessingStage handles data processing
type ProcessingStage struct {
	BaseStage
	executableDir string
	logger        *slog.Logger
	options       *StageOptions
}

// NewProcessingStage creates a new processing Step
func NewProcessingStage(executableDir string, logger *slog.Logger, options *StageOptions) *ProcessingStage {
	if options == nil {
		options = &StageOptions{}
	}

	// Create logger with Step context
	if logger != nil {
		logger = logger.With(slog.String("Step", StageIDProcessing))
		logger.Info("Processing Step initialized",
			slog.String("executable_dir", executableDir))
	}
	return &ProcessingStage{
		BaseStage:     NewBaseStage(StageIDProcessing, StageNameProcessing, []string{StageIDScraping}), // Depends on scraping
		executableDir: executableDir,
		logger:        logger,
		options:       options,
	}
}

// Execute runs the processor to convert Excel files to CSV
func (p *ProcessingStage) Execute(ctx context.Context, state *OperationState) error {
	StepState := state.GetStage(p.ID())

	// Log Step execution start
	if p.logger != nil {
		p.logger.Info("Processing Step started",
			slog.String("pipeline_id", state.ID))

		if inputDir, ok := state.GetConfig("input_dir"); ok {
			p.logger.Info("Processing configuration",
				slog.String("input_dir", fmt.Sprintf("%v", inputDir)))
		}
	}

	p.updateProgress(state.ID, StepState, 10, "Starting processor...")

	processorPath := filepath.Join(p.executableDir, "processor.exe")
	if _, err := os.Stat(processorPath); err != nil {
		if p.logger != nil {
			p.logger.Error("Processor executable not found",
				slog.String("path", processorPath),
				slog.String("error", err.Error()))
		}
		return fmt.Errorf("processor.exe not found: %w", err)
	}

	// Set up input and output directories relative to executable
	inputDir := filepath.Join(p.executableDir, "data", "downloads")
	outputDir := filepath.Join(p.executableDir, "data", "reports")  // Fixed: Use reports directory for consistency
	
	// Create processor command with proper arguments
	cmd := exec.CommandContext(ctx, processorPath, "--in", inputDir, "--out", outputDir)
	cmd.Dir = p.executableDir
	
	if p.logger != nil {
		p.logger.Info("Running processor with directories",
			slog.String("input", inputDir),
			slog.String("output", outputDir))
	}

	p.updateProgress(state.ID, StepState, 50, "Processing data...")

	if p.options.EnableProgress && p.options.WebSocketManager != nil {
		if err := p.executeWithProgress(ctx, cmd, state.ID, StepState, state); err != nil {
			if p.logger != nil {
				p.logger.Error("Processor execution failed",
					slog.String("error", err.Error()))
			}
			return fmt.Errorf("processor failed: %w", err)
		}
	} else {
		output, err := cmd.CombinedOutput()
		if err != nil {
			if p.logger != nil {
				p.logger.Error("Processor execution failed",
					slog.String("error", err.Error()),
					slog.String("output", string(output)),
					slog.String("pipeline_id", state.ID),
					slog.String("stage", p.ID()),
					slog.String("command", processorPath),
					slog.String("input_dir", inputDir),
					slog.String("output_dir", outputDir))
			}
			return fmt.Errorf("processor failed: %w, output: %s", err, string(output))
		}
	}

	p.updateProgress(state.ID, StepState, 100, "Processing completed")
	return nil
}

// executeWithProgress runs the command with real-time progress tracking
func (p *ProcessingStage) executeWithProgress(ctx context.Context, cmd *exec.Cmd, operationID string, StepState *StepState, state *OperationState) error {
	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start processor: %w", err)
	}

	// Track progress
	var processedFiles, totalFiles int
	var fileList []string
	var currentFileName string
	progressChan := make(chan string, 100)
	errChan := make(chan error, 2)

	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			progressChan <- line
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("stdout scan error: %w", err)
		}
		close(progressChan)
	}()

	// Read stderr in goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		var errOutput strings.Builder
		for scanner.Scan() {
			errOutput.WriteString(scanner.Text() + "\n")
		}
		if errOutput.Len() > 0 {
			errChan <- fmt.Errorf("stderr output: %s", errOutput.String())
		}
	}()

	// Process output lines
	for line := range progressChan {
		if p.logger != nil {
			p.logger.Debug("Processor output",
				slog.String("line", line))
		}

		// Parse different types of messages
		switch {
		case strings.Contains(line, "Files to process:"):
			// Parse file list: "Files to process: file1.xlsx|file2.xlsx|file3.xlsx"
			if idx := strings.Index(line, "Files to process:"); idx >= 0 {
				filesStr := strings.TrimSpace(line[idx+17:])
				if filesStr != "" {
					fileList = strings.Split(filesStr, "|")
					StepState.Metadata["file_list"] = fileList
					StepState.Metadata["total_files"] = len(fileList)
				}
			}
			
		case strings.Contains(line, "Processing file") && strings.Contains(line, "of"):
			// Parse: "Processing file X of Y: filename"
			var current, total int
			var fileName string
			if n, _ := fmt.Sscanf(line, "Processing file %d of %d: %s", &current, &total, &fileName); n >= 2 {
				processedFiles = current
				if total > 0 {
					totalFiles = total
				}
				currentFileName = fileName
				
				// Calculate actual progress based on files processed
				progress := 0
				if totalFiles > 0 {
					progress = int(float64(processedFiles) * 100 / float64(totalFiles))
				}
				
				message := fmt.Sprintf("Processing file %d of %d: %s", processedFiles, totalFiles, fileName)
				p.updateProgress(operationID, StepState, progress, message)
				
				// Update metadata
				StepState.Metadata["files_processed"] = processedFiles
				StepState.Metadata["total_files"] = totalFiles
				StepState.Metadata["current_file"] = currentFileName
				if len(fileList) == 0 && totalFiles > 0 {
					// If we don't have the file list yet, create a placeholder
					StepState.Metadata["file_list"] = fileList
				}
			}

		case strings.Contains(line, "Found") && strings.Contains(line, "Excel files"):
			// Total files detected
			if _, err := fmt.Sscanf(line, "Found %d Excel files", &totalFiles); err == nil {
				p.updateProgress(operationID, StepState, 15, fmt.Sprintf("Found %d Excel files to process", totalFiles))
				StepState.Metadata["total_files"] = totalFiles
			}

		case strings.Contains(line, "Converted") && strings.Contains(line, "to CSV"):
			// File conversion complete
			if p.logger != nil {
				p.logger.Info("File conversion",
					slog.String("message", line))
			}

		case strings.Contains(line, "Processing complete"):
			// Processing complete - don't need artificial "finalizing" stage
			// The actual file processing is already at 100%
			StepState.Metadata["csv_files_created"] = processedFiles
		}
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		select {
		case stderr := <-errChan:
			return fmt.Errorf("processor failed: %w, stderr: %v", err, stderr)
		default:
			return fmt.Errorf("processor failed: %w", err)
		}
	}

	// Final progress update
	p.updateProgress(operationID, StepState, 100, fmt.Sprintf("Completed: %d files processed", processedFiles))

	// Verify files were processed
	if processedFiles == 0 {
		return fmt.Errorf("no files were processed")
	}
	
	// TODO: Update manifest with produced CSV files for the index stage
	// Currently the manifest is not accessible from OperationState
	// This needs to be refactored to properly pass data between stages

	return nil
}

// updateProgress updates progress through the centralized StatusBroadcaster
func (p *ProcessingStage) updateProgress(operationID string, StepState *StepState, progress int, message string) {
	StepState.UpdateProgress(float64(progress), message)

	// Use centralized StatusBroadcaster for all updates
	if p.options.StatusBroadcaster != nil {
		// Update through the broadcaster - single source of truth
		p.options.StatusBroadcaster.UpdateStepProgress(operationID, p.ID(), progress, message)
	}
}

// RequiredInputs returns the Excel files needed for processing
func (p *ProcessingStage) RequiredInputs() []DataRequirement {
	return []DataRequirement{
		{
			Type:     "excel_files",
			Location: "data/downloads",
			MinCount: 1, // Need at least one Excel file to process
			Optional: false,
		},
	}
}

// ProducedOutputs returns the CSV files produced by processing
func (p *ProcessingStage) ProducedOutputs() []DataOutput {
	return []DataOutput{
		{
			Type:     "csv_files",
			Location: "data/reports",
			Pattern:  "*.csv",
		},
	}
}

// CanRun checks if Excel files are available for processing
func (p *ProcessingStage) CanRun(manifest *PipelineManifest) bool {
	// Defensive: Check if stage is properly initialized
	if p == nil {
		return false
	}

	// Log entry per CLAUDE.md
	if p.logger != nil {
		p.logger.Debug("ProcessingStage.CanRun starting",
			slog.String("stage", "processing"),
			slog.String("executable_dir", p.executableDir))
	}

	// Primary check: Look in manifest first (faster)
	if manifest != nil {
		if data, exists := manifest.GetData("excel_files"); exists {
			if p.logger != nil {
				p.logger.Info("Checking manifest for excel files",
					slog.String("stage", "processing"),
					slog.Bool("exists", exists),
					slog.Int("file_count", data.FileCount),
					slog.Bool("can_run", data.FileCount >= 1))
			}
			if data.FileCount >= 1 {
				return true
			}
		}
	}

	// Fallback: Check filesystem using centralized FileDetector (SSOT)
	downloadsDir := filepath.Join(p.executableDir, "data", "downloads")
	
	if p.logger != nil {
		p.logger.Info("Manifest check negative, checking filesystem",
			slog.String("stage", "processing"),
			slog.String("downloads_dir", downloadsDir))
	}

	// Use FileDetector for consistent file detection
	detector := NewFileDetector(p.logger)
	fileCount, err := detector.DetectExcelFiles(downloadsDir)
	
	if err != nil {
		if p.logger != nil {
			p.logger.Error("Error detecting Excel files",
				slog.String("stage", "processing"),
				slog.String("directory", downloadsDir),
				slog.String("error", err.Error()))
		}
		// Even with error, if we found some files, allow running
		if fileCount > 0 {
			if p.logger != nil {
				p.logger.Warn("Detection had errors but found files, allowing run",
					slog.String("stage", "processing"),
					slog.Int("file_count", fileCount))
			}
			return true
		}
		return false
	}

	// Log final decision
	canRun := fileCount > 0
	if p.logger != nil {
		p.logger.Info("ProcessingStage.CanRun decision",
			slog.String("stage", "processing"),
			slog.String("method", "FileDetector"),
			slog.Int("excel_files_found", fileCount),
			slog.Bool("can_run", canRun))
	}

	return canRun
}

// IndicesStage handles index extraction
type IndicesStage struct {
	BaseStage
	executableDir string
	logger        *slog.Logger
	options       *StageOptions
}

// NewIndicesStage creates a new indices extraction Step
func NewIndicesStage(executableDir string, logger *slog.Logger, options *StageOptions) *IndicesStage {
	if options == nil {
		options = &StageOptions{}
	}

	// Create logger with Step context
	if logger != nil {
		logger = logger.With(slog.String("Step", StageIDIndices))
		logger.Info("Indices Step initialized",
			slog.String("executable_dir", executableDir))
	}

	return &IndicesStage{
		BaseStage:     NewBaseStage(StageIDIndices, StageNameIndices, []string{StageIDProcessing}), // Depends on processing
		executableDir: executableDir,
		logger:        logger,
		options:       options,
	}
}

// Execute runs the index extractor
func (i *IndicesStage) Execute(ctx context.Context, state *OperationState) error {
	StepState := state.GetStage(i.ID())

	// Log Step execution start
	if i.logger != nil {
		i.logger.Info("Indices extraction Step started",
			slog.String("pipeline_id", state.ID))
	}

	i.updateProgress(state.ID, StepState, 10, "Starting index extractor...")

	indexPath := filepath.Join(i.executableDir, "indexcsv.exe")
	if _, err := os.Stat(indexPath); err != nil {
		if i.logger != nil {
			i.logger.Error("Index extractor executable not found",
				slog.String("path", indexPath),
				slog.String("error", err.Error()))
		}
		return fmt.Errorf("indexcsv.exe not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, indexPath)
	cmd.Dir = i.executableDir

	i.updateProgress(state.ID, StepState, 50, "Extracting indices...")

	if i.options.EnableProgress && i.options.WebSocketManager != nil {
		if err := i.executeWithProgress(ctx, cmd, state.ID, StepState, state); err != nil {
			if i.logger != nil {
				i.logger.Error("Index extraction failed",
					slog.String("error", err.Error()))
			}
			return fmt.Errorf("index extraction failed: %w", err)
		}
	} else {
		output, err := cmd.CombinedOutput()
		if err != nil {
			if i.logger != nil {
				i.logger.Error("Index extraction failed",
					slog.String("error", err.Error()),
					slog.String("output", string(output)))
			}
			return fmt.Errorf("index extraction failed: %w, output: %s", err, string(output))
		}
	}

	i.updateProgress(state.ID, StepState, 100, "Index extraction completed")
	return nil
}

// executeWithProgress runs the command with real-time progress tracking
func (i *IndicesStage) executeWithProgress(ctx context.Context, cmd *exec.Cmd, operationID string, StepState *StepState, state *OperationState) error {
	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start index extractor: %w", err)
	}

	// Track progress
	var processedFiles int
	var totalFiles int
	var fileList []string
	progressChan := make(chan string, 100)
	errChan := make(chan error, 2)

	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			progressChan <- line
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("stdout scan error: %w", err)
		}
		close(progressChan)
	}()

	// Read stderr in goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		var errOutput strings.Builder
		for scanner.Scan() {
			errOutput.WriteString(scanner.Text() + "\n")
		}
		if errOutput.Len() > 0 {
			errChan <- fmt.Errorf("stderr output: %s", errOutput.String())
		}
	}()

	// Process output lines
	for line := range progressChan {
		if i.logger != nil {
			i.logger.Debug("Index extractor output",
				slog.String("line", line))
		}

		// Parse different types of messages
		switch {
		case strings.Contains(line, "Files to process:"):
			// Parse file list
			if idx := strings.Index(line, "Files to process:"); idx >= 0 {
				filesStr := strings.TrimSpace(line[idx+17:])
				if filesStr != "" {
					fileList = strings.Split(filesStr, "|")
					StepState.Metadata["file_list"] = fileList
					StepState.Metadata["total_files"] = len(fileList)
				}
			}
			
		case strings.Contains(line, "Found") && strings.Contains(line, "Excel files"):
			// Parse total files
			fmt.Sscanf(line, "Found %d Excel files", &totalFiles)
			i.updateProgress(operationID, StepState, 0, fmt.Sprintf("Found %d files to process", totalFiles))
			StepState.Metadata["total_files"] = totalFiles
			
		case strings.Contains(line, "Processing file") && strings.Contains(line, "of"):
			// Parse: "Processing file X of Y: filename"
			var current, total int
			var filename string
			if n, _ := fmt.Sscanf(line, "Processing file %d of %d: %s", &current, &total, &filename); n >= 2 {
				processedFiles = current
				if total > 0 {
					totalFiles = total
				}
				
				// Calculate actual progress
				progress := 0
				if totalFiles > 0 {
					progress = int(float64(processedFiles) * 100 / float64(totalFiles))
				}
				
				message := fmt.Sprintf("Extracting indices from file %d of %d", processedFiles, totalFiles)
				i.updateProgress(operationID, StepState, progress, message)
				StepState.Metadata["files_processed"] = processedFiles
				StepState.Metadata["total_files"] = totalFiles
				StepState.Metadata["current_file"] = filename
				if len(fileList) == 0 && totalFiles > 0 {
					StepState.Metadata["file_list"] = fileList
				}
			}

		case strings.Contains(line, "Index extraction complete"):
			// All done - parsed from indexcsv output
			StepState.Metadata["indices_extracted"] = []string{"ISX60", "ISX15"}
		}
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		select {
		case stderr := <-errChan:
			return fmt.Errorf("index extractor failed: %w, stderr: %v", err, stderr)
		default:
			return fmt.Errorf("index extractor failed: %w", err)
		}
	}

	// Verify index file was created - single source of truth
	indexesDir := filepath.Join(i.executableDir, "data", "reports", "indexes")
	if err := os.MkdirAll(indexesDir, 0755); err != nil {
		return fmt.Errorf("create indexes directory: %w", err)
	}
	indexFile := filepath.Join(indexesDir, "indexes.csv")
	
	// Check if index file was created
	if _, err := os.Stat(indexFile); err != nil {
		return fmt.Errorf("index file not created: %w", err)
	}

	// Final progress update
	i.updateProgress(operationID, StepState, 100, fmt.Sprintf("Completed: %d files processed", processedFiles))
	
	// TODO: Update manifest with produced index data for the liquidity stage
	// Currently the manifest is not accessible from OperationState
	// This needs to be refactored to properly pass data between stages

	return nil
}

// updateProgress updates progress through the centralized StatusBroadcaster
func (i *IndicesStage) updateProgress(operationID string, StepState *StepState, progress int, message string) {
	StepState.UpdateProgress(float64(progress), message)

	// Use centralized StatusBroadcaster for all updates
	if i.options.StatusBroadcaster != nil {
		// Update through the broadcaster - single source of truth
		i.options.StatusBroadcaster.UpdateStepProgress(operationID, i.ID(), progress, message)
	}
}

// RequiredInputs returns the Excel files needed for index extraction
func (i *IndicesStage) RequiredInputs() []DataRequirement {
	return []DataRequirement{
		{
			Type:     "excel_files",
			Location: "data/downloads",
			MinCount: 1, // Need at least one Excel file to extract indices
			Optional: false,
		},
	}
}

// ProducedOutputs returns the index data produced
func (i *IndicesStage) ProducedOutputs() []DataOutput {
	return []DataOutput{
		{
			Type:     "index_data",
			Location: "data/output",
			Pattern:  "indexes.csv",
		},
	}
}

// CanRun checks if CSV files are available for index extraction
func (i *IndicesStage) CanRun(manifest *PipelineManifest) bool {
	// Check if Excel files are available (indices extracts from Excel, not CSV)
	if data, exists := manifest.GetData("excel_files"); exists {
		return data.FileCount >= 1
	}
	// Also check the actual downloads directory for Excel files
	downloadsDir := filepath.Join(i.executableDir, "data", "downloads")
	files, _ := filepath.Glob(filepath.Join(downloadsDir, "*.xlsx"))
	return len(files) > 0
}

// LiquidityStage handles liquidity calculation
type LiquidityStage struct {
	BaseStage
	executableDir string
	logger        *slog.Logger
	options       *StageOptions
}

// NewLiquidityStage creates a new liquidity calculation step
func NewLiquidityStage(executableDir string, logger *slog.Logger, options *StageOptions) *LiquidityStage {
	if options == nil {
		options = &StageOptions{}
	}

	// Create logger with Step context
	if logger != nil {
		logger = logger.With(slog.String("Step", StageIDLiquidity))
		logger.Info("Liquidity calculation step initialized",
			slog.String("executable_dir", executableDir))
	}
	return &LiquidityStage{
		BaseStage:     NewBaseStage(StageIDLiquidity, StageNameLiquidity, []string{StageIDProcessing}), // Depends on processing (for CSV files)
		executableDir: executableDir,
		logger:        logger,
		options:       options,
	}
}

// Execute runs the liquidity calculation
func (l *LiquidityStage) Execute(ctx context.Context, state *OperationState) error {
	StepState := state.GetStage(l.ID())

	// Log step execution start
	if l.logger != nil {
		l.logger.InfoContext(ctx, "Liquidity calculation step started",
			slog.String("pipeline_id", state.ID))
	}

	l.updateProgress(state.ID, StepState, 10, "Starting liquidity calculation...")

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("liquidity calculation cancelled: %w", ctx.Err())
	default:
	}

	// 1. Initialize liquidity calculator with 60-day window (default)
	window := liquidity.Window60
	// Use penalty parameters from ISX Hybrid Liquidity Metric paper
	// β=0.75 for mild penalty, γ=1.5 for steep penalty, p*=0.5 transition point
	penaltyParams := liquidity.PenaltyParams{
		PiecewiseP0:       1.0,   // Not used for inactivity-based penalties
		PiecewiseBeta:     0.75,  // Mild penalty slope for low inactivity (p0 < p*)
		PiecewiseGamma:    1.5,   // Steep penalty slope for high inactivity (p0 > p*)
		PiecewisePStar:    0.5,   // Transition at 50% inactivity ratio
		PiecewiseMaxMult:  3.0,   // Maximum penalty multiplier
		ExponentialP0:     1.0,   // Reference point for exponential penalty
		ExponentialAlpha:  0.2,   // Exponential growth rate
		ExponentialMaxMult: 2.5,  // Maximum exponential penalty
	}
	weights := liquidity.ComponentWeights{
		Impact:     0.35,  // Price impact component
		Value:      0.40,  // Trading value component (with SMA)
		Continuity: 0.05,  // Minimal - mostly redundant with Value
		Spread:     0.20,  // Bid-ask spread proxy - transaction cost dimension
	}
	weights.Normalize() // Ensure weights sum to 1

	calculator := liquidity.NewCalculator(window, penaltyParams, weights, l.logger)

	if l.logger != nil {
		l.logger.InfoContext(ctx, "Liquidity calculator initialized",
			slog.String("window", window.String()))
	}

	l.updateProgress(state.ID, StepState, 20, "Loading trading data...")

	// 2. Load trading data from CSV files in data/reports/
	tradingData, err := l.loadTradingDataFromCSV(ctx)
	if err != nil {
		if l.logger != nil {
			l.logger.ErrorContext(ctx, "Failed to load trading data",
				slog.String("error", err.Error()))
		}
		return fmt.Errorf("load trading data: %w", err)
	}

	if l.logger != nil {
		l.logger.InfoContext(ctx, "Trading data loaded successfully",
			slog.Int("data_points", len(tradingData)))
	}

	l.updateProgress(state.ID, StepState, 50, "Calculating liquidity metrics...")

	// Check for context cancellation before calculation
	select {
	case <-ctx.Done():
		return fmt.Errorf("liquidity calculation cancelled: %w", ctx.Err())
	default:
	}

	// 3. Calculate liquidity metrics
	metrics, err := calculator.Calculate(ctx, tradingData)
	if err != nil {
		if l.logger != nil {
			l.logger.ErrorContext(ctx, "Liquidity calculation failed",
				slog.String("error", err.Error()))
		}
		return fmt.Errorf("liquidity calculation failed: %w", err)
	}

	if l.logger != nil {
		l.logger.InfoContext(ctx, "Liquidity metrics calculated",
			slog.Int("metric_count", len(metrics)))
	}

	l.updateProgress(state.ID, StepState, 80, "Scaling and ranking results...")

	// 4. Save results to CSV file
	currentDate := time.Now()
	
	// Create liquidity_reports subdirectory if it doesn't exist
	liquidityReportsDir := filepath.Join(l.executableDir, "data", "reports", "liquidity_reports")
	if err := os.MkdirAll(liquidityReportsDir, 0755); err != nil {
		if l.logger != nil {
			l.logger.ErrorContext(ctx, "Failed to create liquidity reports directory",
				slog.String("dir", liquidityReportsDir),
				slog.String("error", err.Error()))
		}
		return fmt.Errorf("create liquidity reports directory: %w", err)
	}
	
	outputFilename := fmt.Sprintf("liquidity_scores_%s.csv", currentDate.Format("2006-01-02"))
	outputPath := filepath.Join(liquidityReportsDir, outputFilename)

	l.updateProgress(state.ID, StepState, 90, "Saving results...")

	// Save liquidity metrics to CSV
	if err := liquidity.SaveToCSV(metrics, outputPath); err != nil {
		if l.logger != nil {
			l.logger.ErrorContext(ctx, "Failed to save liquidity results",
				slog.String("output_path", outputPath),
				slog.String("error", err.Error()))
		}
		return fmt.Errorf("save liquidity results: %w", err)
	}

	// 5. Generate insights from liquidity scores
	l.updateProgress(state.ID, StepState, 95, "Generating trading insights...")
	
	if err := liquidity.GenerateInsights(outputPath, liquidityReportsDir); err != nil {
		if l.logger != nil {
			l.logger.WarnContext(ctx, "Failed to generate liquidity insights",
				slog.String("error", err.Error()))
		}
		// Don't fail the operation if insights generation fails
	} else {
		if l.logger != nil {
			l.logger.InfoContext(ctx, "Liquidity insights generated successfully")
		}
	}

	// 6. Update manifest with output location
	StepState.Metadata["output_file"] = outputFilename
	StepState.Metadata["output_path"] = outputPath
	StepState.Metadata["metrics_calculated"] = len(metrics)
	StepState.Metadata["calculation_window"] = window.String()

	if l.logger != nil {
		l.logger.InfoContext(ctx, "Liquidity calculation completed successfully",
			slog.String("output_file", outputFilename),
			slog.Int("metrics_count", len(metrics)))
	}

	l.updateProgress(state.ID, StepState, 100, fmt.Sprintf("Liquidity calculation completed: %d metrics generated", len(metrics)))
	return nil
}

// updateProgress updates progress through the centralized StatusBroadcaster
func (l *LiquidityStage) updateProgress(operationID string, StepState *StepState, progress int, message string) {
	StepState.UpdateProgress(float64(progress), message)

	// Use centralized StatusBroadcaster for all updates
	if l.options.StatusBroadcaster != nil {
		// Update through the broadcaster - single source of truth
		l.options.StatusBroadcaster.UpdateStepProgress(operationID, l.ID(), progress, message)
	}
}

// RequiredInputs returns the CSV trading data needed for liquidity calculation
func (l *LiquidityStage) RequiredInputs() []DataRequirement {
	return []DataRequirement{
		{
			Type:     "csv_files",
			Location: "data/reports",
			MinCount: 1, // Need at least one trading history CSV file
			Optional: false,
		},
	}
}

// ProducedOutputs returns the liquidity analysis results produced
func (l *LiquidityStage) ProducedOutputs() []DataOutput {
	return []DataOutput{
		{
			Type:     "liquidity_results",
			Location: "data/reports/liquidity_reports",
			Pattern:  "liquidity_*.csv",
		},
	}
}

// CanRun checks if CSV trading data is available for liquidity calculation
func (l *LiquidityStage) CanRun(manifest *PipelineManifest) bool {
	// Check if CSV files are available in manifest
	if data, exists := manifest.GetData("csv_files"); exists {
		if data.FileCount >= 1 {
			return true
		}
	}

	// Fallback: Check the ticker subdirectory for trading history CSV files
	tickersDir := filepath.Join(l.executableDir, "data", "reports", "ticker")
	files, err := filepath.Glob(filepath.Join(tickersDir, "*_trading_history.csv"))
	if err == nil && len(files) > 0 {
		if l.logger != nil {
			l.logger.Info("Found trading history CSV files for liquidity calculation",
				slog.Int("file_count", len(files)))
		}
		return true
	}

	// Also check for any CSV files as fallback in old location
	if len(files) == 0 {
		reportsDir := filepath.Join(l.executableDir, "data", "reports")
		files, err = filepath.Glob(filepath.Join(reportsDir, "*_trading_history.csv"))
		if len(files) == 0 {
			files, err = filepath.Glob(filepath.Join(reportsDir, "*.csv"))
		}
	}
	canRun := err == nil && len(files) > 0

	if l.logger != nil {
		l.logger.Info("LiquidityStage.CanRun decision",
			slog.String("tickers_dir", tickersDir),
			slog.Int("csv_files_found", len(files)),
			slog.Bool("can_run", canRun))
	}

	return canRun
}

// loadTradingDataFromCSV loads trading data from ticker-specific CSV files and calculates metrics per ticker
func (l *LiquidityStage) loadTradingDataFromCSV(ctx context.Context) ([]liquidity.TradingDay, error) {
	// Look for ticker files in the ticker subdirectory first
	tickersDir := filepath.Join(l.executableDir, "data", "reports", "ticker")
	
	if l.logger != nil {
		l.logger.InfoContext(ctx, "Loading trading data from ticker-specific CSV files",
			slog.String("tickers_dir", tickersDir))
	}

	// Find all ticker-specific trading history files
	tickerFiles, err := filepath.Glob(filepath.Join(tickersDir, "*_trading_history.csv"))
	if err != nil {
		return nil, fmt.Errorf("find ticker CSV files: %w", err)
	}

	if len(tickerFiles) == 0 {
		// Fallback: check old location
		reportsDir := filepath.Join(l.executableDir, "data", "reports")
		tickerFiles, err = filepath.Glob(filepath.Join(reportsDir, "*_trading_history.csv"))
		if err != nil {
			return nil, fmt.Errorf("find ticker CSV files: %w", err)
		}
		if len(tickerFiles) == 0 {
			return nil, fmt.Errorf("no ticker trading history files found in %s or %s", tickersDir, reportsDir)
		}
		if l.logger != nil {
			l.logger.InfoContext(ctx, "Using ticker files from old location",
				slog.String("reports_dir", reportsDir),
				slog.Int("file_count", len(tickerFiles)))
		}
	}

	if l.logger != nil {
		l.logger.InfoContext(ctx, "Found ticker history files",
			slog.Int("ticker_count", len(tickerFiles)))
	}

	var allTradingData []liquidity.TradingDay
	filesProcessed := 0
	tickersWithSufficientData := 0

	// Process each ticker file individually
	for _, tickerFile := range tickerFiles {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("loading cancelled: %w", ctx.Err())
		default:
		}

		filename := filepath.Base(tickerFile)
		// Extract ticker symbol from filename (e.g., "BMNS_trading_history.csv" -> "BMNS")
		tickerSymbol := strings.TrimSuffix(filename, "_trading_history.csv")
		
		if l.logger != nil {
			l.logger.DebugContext(ctx, "Processing ticker file",
				slog.String("file", filename),
				slog.String("ticker", tickerSymbol))
		}

		// Load data for this specific ticker
		tickerData, err := l.loadTradingDataFromSingleCSV(ctx, tickerFile)
		if err != nil {
			if l.logger != nil {
				l.logger.WarnContext(ctx, "Failed to load ticker data",
					slog.String("ticker", tickerSymbol),
					slog.String("file", filename),
					slog.String("error", err.Error()))
			}
			continue // Skip problematic files
		}

		// Check if ticker has sufficient data for analysis
		if len(tickerData) >= 20 { // Minimum 20 days for meaningful analysis
			allTradingData = append(allTradingData, tickerData...)
			tickersWithSufficientData++
			
			if l.logger != nil {
				l.logger.DebugContext(ctx, "Loaded ticker data successfully",
					slog.String("ticker", tickerSymbol),
					slog.Int("days", len(tickerData)))
			}
		} else {
			if l.logger != nil {
				l.logger.WarnContext(ctx, "Insufficient data for ticker",
					slog.String("ticker", tickerSymbol),
					slog.Int("days", len(tickerData)),
					slog.Int("minimum_required", 20))
			}
		}
		
		filesProcessed++
	}

	if len(allTradingData) == 0 {
		return nil, fmt.Errorf("no valid trading data found in any ticker files")
	}

	if l.logger != nil {
		l.logger.InfoContext(ctx, "Ticker data loading completed",
			slog.Int("files_processed", filesProcessed),
			slog.Int("tickers_with_data", tickersWithSufficientData),
			slog.Int("total_records", len(allTradingData)))
	}

	return allTradingData, nil
}

// loadTradingDataFromSingleCSV loads trading data from a single CSV file
func (l *LiquidityStage) loadTradingDataFromSingleCSV(ctx context.Context, csvPath string) ([]liquidity.TradingDay, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV records: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file has insufficient data (need header + at least 1 data row)")
	}

	// Parse header to determine column indices
	header := records[0]
	columnMap := make(map[string]int)
	for i, col := range header {
		columnMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Required columns (with flexible naming)
	requiredCols := map[string][]string{
		"date":      {"date", "trading_date", "day"},
		"symbol":    {"symbol", "ticker", "code", "companyname"},
		"open":      {"openprice", "open", "opening_price", "open_price"},
		"high":      {"highprice", "high", "highest_price", "high_price"},
		"low":       {"lowprice", "low", "lowest_price", "low_price"},
		"close":     {"closeprice", "close", "closing_price", "close_price"},
		"volume":    {"volume", "trading_volume", "total_volume"},
		"value":     {"value", "trading_value", "total_value", "amount"},
	}

	// Find column indices
	colIndices := make(map[string]int)
	for field, variations := range requiredCols {
		found := false
		for _, variation := range variations {
			if idx, exists := columnMap[variation]; exists {
				colIndices[field] = idx
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("required column '%s' not found (tried: %v)", field, variations)
		}
	}

	// Optional columns
	numTradesCol := -1
	if idx, exists := columnMap["numtrades"]; exists {
		numTradesCol = idx
	} else if idx, exists := columnMap["num_trades"]; exists {
		numTradesCol = idx
	} else if idx, exists := columnMap["number_of_trades"]; exists {
		numTradesCol = idx
	}

	statusCol := -1
	if idx, exists := columnMap["tradingstatus"]; exists {
		statusCol = idx
	} else if idx, exists := columnMap["status"]; exists {
		statusCol = idx
	} else if idx, exists := columnMap["trading_status"]; exists {
		statusCol = idx
	}

	var tradingData []liquidity.TradingDay

	// Parse data rows
	for i, record := range records[1:] { // Skip header
		if len(record) < len(colIndices) {
			if l.logger != nil {
				l.logger.WarnContext(ctx, "Skipping incomplete record",
					slog.Int("row", i+2),
					slog.Int("columns", len(record)))
			}
			continue
		}

		// Parse date
		dateStr := strings.TrimSpace(record[colIndices["date"]])
		date, err := l.parseDate(dateStr)
		if err != nil {
			if l.logger != nil {
				l.logger.WarnContext(ctx, "Skipping record with invalid date",
					slog.Int("row", i+2),
					slog.String("date_str", dateStr),
					slog.String("error", err.Error()))
			}
			continue
		}

		// Parse numeric fields
		symbol := strings.TrimSpace(record[colIndices["symbol"]])
		if symbol == "" {
			continue
		}

		open, err := l.parseFloat(record[colIndices["open"]])
		if err != nil {
			continue
		}
		high, err := l.parseFloat(record[colIndices["high"]])
		if err != nil {
			continue
		}
		low, err := l.parseFloat(record[colIndices["low"]])
		if err != nil {
			continue
		}
		close, err := l.parseFloat(record[colIndices["close"]])
		if err != nil {
			continue
		}
		volume, err := l.parseFloat(record[colIndices["volume"]])
		if err != nil {
			continue
		}
		value, err := l.parseFloat(record[colIndices["value"]])
		if err != nil {
			continue
		}

		// Parse optional fields
		numTrades := 0
		if numTradesCol >= 0 && numTradesCol < len(record) {
			if nt, err := strconv.Atoi(strings.TrimSpace(record[numTradesCol])); err == nil {
				numTrades = nt
			}
		}

		// Parse status - CSV contains "true"/"false", normalize to standard format
		status := "ACTIVE" // Default status
		if statusCol >= 0 && statusCol < len(record) {
			rawStatus := strings.TrimSpace(record[statusCol])
			// Normalize boolean string values to standard status
			if rawStatus == "true" {
				status = "true" // Keep as-is since IsTrading() now checks for this
			} else if rawStatus == "false" {
				status = "false"
			} else {
				status = rawStatus // Keep other values as-is (e.g., "ACTIVE", "SUSPENDED")
			}
		}
		
		// Don't override explicit status from CSV
		// Only infer if we have default "ACTIVE" and no trades
		if status == "ACTIVE" && (volume == 0 || numTrades == 0) {
			status = "SUSPENDED"
		}

		tradingDay := liquidity.TradingDay{
			Date:          date,
			Symbol:        symbol,
			Open:          open,
			High:          high,
			Low:           low,
			Close:         close,
			Volume:        volume,        // Keep for compatibility (share count)
			ShareVolume:   volume,        // Explicit: share count
			Value:         value,         // Trading value in IQD
			NumTrades:     numTrades,
			TradingStatus: status,
		}

		// Validate the trading day data
		if tradingDay.IsValid() {
			tradingData = append(tradingData, tradingDay)
		}
	}

	return tradingData, nil
}

// parseDate parses date strings in various formats
func (l *LiquidityStage) parseDate(dateStr string) (time.Time, error) {
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"2006/01/02",
		"01-02-2006",
		"02-01-2006",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// parseFloat parses float strings, handling common formatting
func (l *LiquidityStage) parseFloat(str string) (float64, error) {
	str = strings.TrimSpace(str)
	if str == "" || str == "-" || str == "N/A" {
		return 0, nil
	}
	
	// Remove commas and other common separators
	str = strings.ReplaceAll(str, ",", "")
	str = strings.ReplaceAll(str, " ", "")
	
	return strconv.ParseFloat(str, 64)
}

// StageFactory creates operation steps with optional configuration
func StageFactory(executableDir string, logger *slog.Logger, options *StageOptions) map[string]Step {
	return map[string]Step{
		StageIDScraping:   NewScrapingStage(executableDir, logger, options),
		StageIDProcessing: NewProcessingStage(executableDir, logger, options),
		StageIDIndices:    NewIndicesStage(executableDir, logger, options),
		StageIDLiquidity:   NewLiquidityStage(executableDir, logger, options),
	}
}

// extractDateFromFileName extracts date from ISX report filename
// Expected format: "2025 08 07 ISX Daily Report.xlsx" or similar
func extractDateFromFileName(fileName string) string {
	// Try to match the date pattern in the filename
	re := regexp.MustCompile(`(\d{4})\s+(\d{2})\s+(\d{2})`)
	matches := re.FindStringSubmatch(fileName)
	if len(matches) >= 4 {
		// Convert to ISO date format
		return fmt.Sprintf("%s-%s-%s", matches[1], matches[2], matches[3])
	}
	return ""
}

// Compile-time interface checks to ensure all stages properly implement Step interface
// This helps catch method receiver issues at compile time
var (
	_ Step = (*ScrapingStage)(nil)
	_ Step = (*ProcessingStage)(nil)
	_ Step = (*IndicesStage)(nil)
	_ Step = (*LiquidityStage)(nil)
)
