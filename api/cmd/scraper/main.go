package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"isxcli/internal/config"
	"isxcli/internal/infrastructure"
	"isxcli/internal/license"

	"github.com/chromedp/chromedp"
)

const (
	baseURL  = "http://www.isx-iq.net"
	startURL = "http://www.isx-iq.net/isxportal/portal/uploadedFilesList.html?currLanguage=en"
)

func main() {
	// Add panic recovery at the very start to catch any crashes
	var logger *slog.Logger // Declare logger early for use in panic handler
	defer func() {
		if r := recover(); r != nil {
			// Log the panic with full stack trace
			fmt.Printf("PANIC RECOVERED: %v\n", r)
			fmt.Printf("Stack trace:\n%s\n", debug.Stack())
			
			// Try to log to file if logger is available
			if logger != nil {
				logger.Error("Scraper panicked",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())))
			}
			os.Exit(1)
		}
	}()
	
	mode := flag.String("mode", "initial", "scrape mode: initial | accumulative")
	fromStr := flag.String("from", "2025-01-01", "start date (YYYY-MM-DD) (used in initial mode if provided)")
	toStr := flag.String("to", "", "optional end date (YYYY-MM-DD); leave blank to keep site default")
	// Actual dates for progress tracking (not for scraper logic)
	actualFromStr := flag.String("actual-from", "", "actual from date for progress calculation")
	actualToStr := flag.String("actual-to", "", "actual to date for progress calculation")
	outDir := flag.String("out", "", "directory to save reports (defaults to data/downloads relative to executable)")
	headless := flag.Bool("headless", true, "run browser headless")
	stateFile := flag.String("state-file", "", "path to license state file (for validation bypass)")
	flag.Parse()

	// Initialize paths first to get default directories
	paths, err := config.GetPaths()
	if err != nil {
		fmt.Printf("Error: Failed to initialize paths: %v\n", err)
		os.Exit(1)
	}

	// Use centralized downloads directory as default if not specified
	if *outDir == "" {
		*outDir = paths.DownloadsDir
	}
	
	// Ensure all required directories exist
	if err := paths.EnsureDirectories(); err != nil {
		fmt.Printf("Error: Failed to create required directories: %v\n", err)
		os.Exit(1)
	}

	// Initialize structured logger per CLAUDE.md
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: Failed to load config, using defaults: %v\n", err)
		cfg = &config.Config{
			Logging: config.LoggingConfig{
				Level:       "info",
				Format:      "json",
				Output:      "both",
				FilePath:    paths.GetLogPath("scraper.log"),
				Development: false,
			},
		}
	}

	// Assign to pre-declared logger variable for panic handler
	var err2 error
	logger, err2 = infrastructure.InitializeLogger(cfg.Logging)
	if err2 != nil {
		fmt.Printf("Warning: Failed to initialize logger, using default: %v\n", err2)
		logger = slog.Default()
	}

	// Start resource monitoring in background
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			logger.Info("Resource usage",
				slog.Uint64("memory_alloc_mb", m.Alloc/1024/1024),
				slog.Uint64("memory_sys_mb", m.Sys/1024/1024),
				slog.Int("goroutines", runtime.NumGoroutine()))
		}
	}()
	
	// Initialize license system
	// Keep console output for user-facing messages
	slog.Info("🔐 ISX Daily Reports Scraper - Licensed Version")
	slog.Info("═══════════════════════════════════════════════")
	logger.Info("ISX Daily Reports Scraper starting", 
		slog.String("mode", *mode),
		slog.String("from", *fromStr),
		slog.String("to", *toStr),
		slog.String("actual_from", *actualFromStr),
		slog.String("actual_to", *actualToStr),
		slog.String("output_dir", *outDir),
		slog.String("executable_dir", paths.ExecutableDir))
	
	// Log resolved paths for debugging
	slog.Info("Output directory", "path", *outDir)
	slog.Info("Executable directory", "path", paths.ExecutableDir)

	if !checkLicense(*stateFile, logger) {
		slog.Info("❌ License validation failed. Application will exit.")
		slog.Info("📞 Contact The Iraqi Investor Group to get a new license.")
		logger.Error("License validation failed")
		os.Exit(1)
	}

	// Create output directory if it doesn't exist (but don't delete existing files)
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		logger.Error("failed to create output dir", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// determine fromSite depending on mode
	var fromSite string
	if *mode == "accumulative" {
		// scan downloads for latest file
		if d, ok := latestDownloadedDate(*outDir); ok {
			fromSite = d.AddDate(0, 0, 1).Format("02/01/2006") // next day
			slog.Info("[MODE accumulative] Detected last report date", "last_date", d.Format("2006-01-02"), "start_from", fromSite)
			logger.Info("Accumulative mode detected last report", 
				slog.String("last_date", d.Format("2006-01-02")),
				slog.String("start_from", fromSite))
		}
	}

	if fromSite == "" {
		// fallback to user provided from
		startDate, err := time.Parse("2006-01-02", *fromStr)
		if err != nil {
			logger.Error("invalid --from date", slog.String("error", err.Error()))
			fmt.Printf("Error: Invalid --from date: %v\n", err)
			os.Exit(1)
		}
		fromSite = startDate.Format("02/01/2006")
		slog.Info("[MODE initial] Starting from date (preserving existing files)", "from_date", startDate.Format("2006-01-02"))
		logger.Info("Initial mode starting", 
			slog.String("from_date", startDate.Format("2006-01-02")),
			slog.String("mode", "preserving existing files"))
	}

	var toSite string
	if *toStr != "" {
		endDate, err := time.Parse("2006-01-02", *toStr)
		if err != nil {
			logger.Error("invalid --to date", slog.String("error", err.Error()))
			os.Exit(1)
		}
		toSite = endDate.Format("02/01/2006")
	}

	// Calculate expected files based on actual date range (not buffered range)
	// Use actual dates if provided, otherwise fall back to buffer dates
	expectedFromStr := *fromStr
	expectedToStr := *toStr
	if *actualFromStr != "" {
		expectedFromStr = *actualFromStr
		logger.Info("Using actual-from date for expected files calculation",
			slog.String("actual_from", *actualFromStr))
	}
	if *actualToStr != "" {
		expectedToStr = *actualToStr
		logger.Info("Using actual-to date for expected files calculation",
			slog.String("actual_to", *actualToStr))
	}
	
	expectedFiles := calculateExpectedFiles(expectedFromStr, expectedToStr)
	slog.Info("Expected files to download", "count", expectedFiles, "from", expectedFromStr, "to", expectedToStr)
	logger.Info("Calculated expected files", 
		slog.Int("expected_files", expectedFiles),
		slog.String("calculation_from", expectedFromStr),
		slog.String("calculation_to", expectedToStr),
		slog.String("buffer_from", *fromStr),
		slog.String("buffer_to", *toStr))

	// Output for parsing by stages.go
	slog.Info("Total expected files", "count", expectedFiles, "from", *fromStr, "to", *toStr)

	// Parse dates for scanning existing files
	fromDateForScan, err := time.Parse("2006-01-02", expectedFromStr)
	if err != nil {
		logger.Warn("Failed to parse from date for scan", slog.String("error", err.Error()))
		fromDateForScan = time.Now().AddDate(0, -1, 0) // Default to 1 month ago
	}
	
	toDateForScan := time.Now()
	if expectedToStr != "" {
		toDateForScan, err = time.Parse("2006-01-02", expectedToStr)
		if err != nil {
			logger.Warn("Failed to parse to date for scan", slog.String("error", err.Error()))
			toDateForScan = time.Now()
		}
	}

	// Scan for existing files first
	existingFiles, existingHolidays := scanExistingFiles(*outDir, fromDateForScan, toDateForScan, logger)
	logger.Info("Pre-scan found existing files",
		slog.Int("existing_files", existingFiles),
		slog.Int("holidays_detected", existingHolidays))

	// Check if we already have all needed files
	if existingFiles + existingHolidays >= expectedFiles {
		logger.Info("All required files already exist",
			slog.Int("existing_files", existingFiles),
			slog.Int("holidays", existingHolidays),
			slog.Int("total", existingFiles + existingHolidays),
			slog.Int("expected", expectedFiles))
		
		// Signal completion to stages.go
		slog.Info("SCRAPER_COMPLETE: All required dates processed")
		
		// Exit successfully without launching browser
		return
	}

	// setup ChromeDP
	opts := chromedp.DefaultExecAllocatorOptions[:]
	if *headless {
		opts = append(opts, chromedp.Flag("headless", true))
	} else {
		opts = append(opts, chromedp.Flag("headless", false))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// Pass actual dates for progress tracking (if provided)
	// These are only for progress calculation, not for stopping logic
	if *actualFromStr != "" {
		logger.Info("Actual from date for progress", slog.String("actual_from", *actualFromStr))
	}
	if *actualToStr != "" {
		logger.Info("Actual to date for progress", slog.String("actual_to", *actualToStr))
	}

	if err := chromedp.Run(ctx, runScraper(fromSite, toSite, *outDir, logger, expectedFiles, *actualFromStr, *actualToStr)); err != nil {
		logger.Error("scraping failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	
	// Don't send automatic completion - it's now sent conditionally based on files+holidays count
	logger.Info("Scraper finished")
}

// scanExistingFiles scans the output directory for existing Excel files within the date range
func scanExistingFiles(outDir string, fromDate, toDate time.Time, logger *slog.Logger) (filesFound int, holidaysDetected int) {
	pattern := filepath.Join(outDir, "*.xlsx")
	files, err := filepath.Glob(pattern)
	if err != nil {
		logger.Warn("Failed to scan existing files", slog.String("error", err.Error()))
		return 0, 0
	}
	
	var lastDate *time.Time
	datePattern := regexp.MustCompile(`(\d{4})\s+(\d{2})\s+(\d{2})`)
	
	for _, file := range files {
		fname := filepath.Base(file)
		matches := datePattern.FindStringSubmatch(fname)
		if len(matches) != 4 {
			continue
		}
		
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		fileDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		
		// Check if file is in date range
		if !fileDate.Before(fromDate) && !fileDate.After(toDate) {
			filesFound++
			
			// Check for holidays (gaps in dates)
			if lastDate != nil {
				daysDiff := fileDate.Sub(*lastDate).Hours() / 24
				if daysDiff > 1 {
					// Detected gap - count holidays/weekends
					holidaysDetected += int(daysDiff) - 1
				}
			}
			lastDate = &fileDate
			
			// Log for stages.go to parse
			slog.Info("Already exists", "file", fname)
		}
	}
	
	return filesFound, holidaysDetected
}

func runScraper(fromSite, toSite, outDir string, logger *slog.Logger, expectedFiles int, actualFromStr, actualToStr string) chromedp.Tasks {
	// Track progress
	totalDownloaded := 0
	totalExisting := 0
	filesInRange := 0      // Files within actual date range
	holidaysInRange := 0   // Holidays within actual date range
	var lastProcessedDate *time.Time // Track for holiday detection
	actions := []chromedp.Action{
		timedAction("Navigate", chromedp.Navigate(startURL)),
		chromedp.WaitVisible(`#date`, chromedp.ByID),
		chromedp.SetValue(`#date`, fromSite, chromedp.ByID),
	}
	if toSite != "" {
		actions = append(actions, chromedp.SetValue(`#toDate`, toSite, chromedp.ByID))
	}
	actions = append(actions,
		chromedp.SetValue(`#reporttype`, "40", chromedp.ByID),
		timedAction("ExecuteSearch", chromedp.Click(`/html/body/div[2]/div/div[3]/div[3]/div[2]/div[4]/div/div[1]/form/div[8]/input`, chromedp.BySearch)),
		chromedp.WaitVisible(`#report`, chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			page := 1
			for {
				slog.Info("Scraping page", "page", page)
				logger.Info("Scraping page", slog.Int("page", page))
				_, _, shouldContinue, err := scrapePage(ctx, outDir, logger, &totalDownloaded, &totalExisting, &filesInRange, &holidaysInRange, expectedFiles, actualFromStr, actualToStr, &lastProcessedDate)
				if err != nil {
					return err
				}
				if !shouldContinue {
					slog.Info("Found existing files, stopping scraping process", "page", page)
					logger.Info("Found existing files, stopping scraping", slog.Int("page", page))
					return nil
				}
				// Check if we've accounted for all expected files
				if (filesInRange + holidaysInRange) >= expectedFiles {
					logger.Info("Completion criteria met",
						slog.Int("files_in_range", filesInRange),
						slog.Int("holidays_in_range", holidaysInRange),
						slog.Int("total_accounted", filesInRange + holidaysInRange),
						slog.Int("expected_files", expectedFiles))
					// Signal completion
					slog.Info("SCRAPER_COMPLETE: All required dates processed")
					return nil
				}
				
				// check if next arrow exists
				var nextHref string
				var ok bool
				err = chromedp.Run(ctx, chromedp.AttributeValue(`a img[src*='next.gif']`, "src", &nextHref, &ok))
				if err != nil || !ok {
					// No next arrow or not clickable
					return nil
				}
				// Click the parent anchor of the img
				if err := chromedp.Click(`a img[src*='next.gif']`, chromedp.ByQuery).Do(ctx); err != nil {
					return nil // assume finished when can't click
				}
				// wait for table refresh
				if err := chromedp.WaitVisible(`#report`, chromedp.ByID).Do(ctx); err != nil {
					return err
				}
				logger.Debug("Page processed", 
					slog.Int("page", page),
					slog.Duration("duration", time.Since(time.Now())))
				page++
			}
		}),
	)

	return chromedp.Tasks(actions)
}

func scrapePage(ctx context.Context, outDir string, logger *slog.Logger, totalDownloaded, totalExisting, filesInRange, holidaysInRange *int, expectedFiles int, actualFromStr, actualToStr string, lastProcessedDate **time.Time) (int, int, bool, error) {
	// Add panic recovery for this function
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in scrapePage",
				slog.Any("panic", r),
				slog.Int("files_downloaded", *totalDownloaded),
				slog.Int("files_existing", *totalExisting),
				slog.String("stack", string(debug.Stack())))
			panic(r) // Re-panic to be caught by main
		}
	}()
	
	// Parse actual dates for boundary and range checking
	var actualFromDate *time.Time
	var actualToDate *time.Time
	if actualFromStr != "" {
		if parsedDate, err := time.Parse("2006-01-02", actualFromStr); err == nil {
			actualFromDate = &parsedDate
		}
	}
	if actualToStr != "" {
		if parsedDate, err := time.Parse("2006-01-02", actualToStr); err == nil {
			actualToDate = &parsedDate
		}
	}
	
	// Helper to check if a date is within the actual range
	isDateInRange := func(t time.Time) bool {
		if actualFromDate != nil && t.Before(*actualFromDate) {
			return false
		}
		if actualToDate != nil && t.After(*actualToDate) {
			return false
		}
		return true
	}
	
	// Add progress checkpoint every 5 files
	totalProcessed := *totalDownloaded + *totalExisting
	if totalProcessed > 0 && totalProcessed%5 == 0 {
		progressPct := float64(totalProcessed) / float64(expectedFiles) * 100
		logger.Info("Progress checkpoint",
			slog.Int("total_processed", totalProcessed),
			slog.Int("downloaded", *totalDownloaded),
			slog.Int("existing", *totalExisting),
			slog.Int("expected", expectedFiles),
			slog.Float64("percentage", progressPct))
	}
	
	// Retrieve rows data: href, date text, type text
	var rows []struct {
		Href string `json:"href"`
		Date string `json:"date"`
		Typ  string `json:"typ"`
	}

	js := `Array.from(document.querySelectorAll('#report tbody tr')).map(tr => {
		const link = tr.querySelector('td.report-download a');
		if (!link) return null;
		const dateCell = tr.querySelector('td.report-titledata1');
		const typeCell = tr.querySelector('td.report-titledata3');
		return {href: link.getAttribute('href'), date: dateCell ? dateCell.innerText.trim() : '', typ: typeCell ? typeCell.innerText.trim() : ''};
	}).filter(Boolean)`

	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &rows)); err != nil {
		return 0, 0, false, err
	}

	foundExistingFiles := 0
	newDownloads := 0

	for _, r := range rows {
		// We only care about Daily type and xlsx file extension
		if strings.ToLower(r.Typ) != "daily" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(r.Href), ".xlsx") {
			continue
		}

		fullURL := r.Href
		if !strings.HasPrefix(r.Href, "http") {
			fullURL = baseURL + r.Href
		}

		// Parse date dd/mm/yyyy
		t, err := time.Parse("02/01/2006", r.Date)
		if err != nil {
			// fallback to original filename
			logger.Warn("unable to parse date", 
				slog.String("date", r.Date), 
				slog.String("error", err.Error()))
		}

		// Check for holiday gaps with previous file
		// Since files are served newest to oldest, lastProcessedDate is newer than current t
		if err == nil && *lastProcessedDate != nil {
			// Calculate days between current (older) and last (newer)
			daysDiff := (*lastProcessedDate).Sub(t).Hours() / 24
			if daysDiff > 1 {
				// Found gap - report holidays between t and lastProcessedDate
				// Start from day after current file (older) to day before last file (newer)
				for d := t.AddDate(0, 0, 1); d.Before(**lastProcessedDate); d = d.AddDate(0, 0, 1) {
					// Skip weekends (Friday=5, Saturday=6 in Iraq)
					if d.Weekday() != time.Friday && d.Weekday() != time.Saturday {
						// Check if this holiday is in our actual date range
						if isDateInRange(d) {
							*holidaysInRange++
							logger.Info("Detected holiday in range",
								slog.String("date", d.Format("2006-01-02")),
								slog.Int("holidays_in_range", *holidaysInRange))
						}
						logger.Info("Detected holiday/non-trading day",
							slog.String("date", d.Format("2006-01-02")))
					}
				}
			}
		}

		var fname string
		if err == nil {
			fname = fmt.Sprintf("%s ISX Daily Report.xlsx", t.Format("2006 01 02"))
		} else {
			fname = filepath.Base(r.Href)
		}

		destPath := filepath.Join(outDir, fname)
		if _, err := os.Stat(destPath); err == nil {
			foundExistingFiles++
			*totalExisting++
			// Check if this existing file is in range
			if err == nil && isDateInRange(t) {
				*filesInRange++
			}
			totalFiles := *totalDownloaded + *totalExisting
			progressMsg := fmt.Sprintf("File %d of %d already exists, skipping", totalFiles, expectedFiles)
			slog.Info(progressMsg, "file", fname)
			logger.Debug("File already exists", 
				slog.String("file", fname),
				slog.Int("total_processed", totalFiles),
				slog.Int("expected_files", expectedFiles),
				slog.Int("files_in_range", *filesInRange))
			continue
		}

		newDownloads++
		*totalDownloaded++
		totalFiles := *totalDownloaded + *totalExisting
		progressMsg := fmt.Sprintf("Downloading file %d of %d", totalFiles, expectedFiles)
		slog.Info(progressMsg, "file", fname)
		logger.Info("Downloading file", 
			slog.String("file", fname),
			slog.Int("file_number", totalFiles),
			slog.Int("expected_files", expectedFiles))
		
		if err := downloadFile(fullURL, destPath); err != nil {
			slog.Error("Failed to download file", "file", fname, "error", err)
			logger.Error("Failed to download file", 
				slog.String("file", fname),
				slog.String("error", err.Error()))
			// Revert counts on failure
			newDownloads--
			*totalDownloaded--
		} else {
			// Successfully downloaded - check if in range
			if err == nil && isDateInRange(t) {
				*filesInRange++
				logger.Info("Downloaded file in range",
					slog.String("file", fname),
					slog.Int("files_in_range", *filesInRange))
			}
		}
		
		// Rate limiting between downloads - respect context cancellation
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-timer.C:
			// Continue with next download
		case <-ctx.Done():
			timer.Stop()
			return newDownloads, foundExistingFiles, false, ctx.Err()
		}
		
		// Check if this file was before actual-from date (buffer zone)
		if err == nil && actualFromDate != nil && t.Before(*actualFromDate) {
			// This file is in the buffer zone - we've processed all files in range
			logger.Info("Reached buffer zone after processing files in range",
				slog.String("file_date", t.Format("2006-01-02")),
				slog.String("actual_from", actualFromDate.Format("2006-01-02")),
				slog.Int("files_downloaded", newDownloads),
				slog.Int("files_existing", foundExistingFiles),
				slog.Int("files_in_range", *filesInRange),
				slog.Int("holidays_in_range", *holidaysInRange))
			
			// Check if we have accounted for all expected files
			if (*filesInRange + *holidaysInRange) >= expectedFiles {
				logger.Info("Completion criteria met",
					slog.Int("files_in_range", *filesInRange),
					slog.Int("holidays_in_range", *holidaysInRange),
					slog.Int("total_accounted", *filesInRange + *holidaysInRange),
					slog.Int("expected_files", expectedFiles))
				// Signal completion
				slog.Info("SCRAPER_COMPLETE: All required dates processed")
			}
			
			return newDownloads, foundExistingFiles, false, nil // Stop scraping
		}
		
		// Update last processed date for holiday detection
		if err == nil {
			*lastProcessedDate = &t
		}
	}

	slog.Info("Page summary", "new_downloads", newDownloads, "existing_files", foundExistingFiles)
	logger.Info("Page summary", 
		slog.Int("new_downloads", newDownloads),
		slog.Int("existing_files", foundExistingFiles))

	// Output total progress summary
	totalFiles := *totalDownloaded + *totalExisting
	slog.Info("Progress summary", 
		"processed", totalFiles, 
		"expected", expectedFiles, 
		"downloaded", *totalDownloaded, 
		"existing", *totalExisting)

	// Simple heuristic: only stop if we found MANY more existing files than new ones
	// This prevents premature stopping when there are gaps or holidays
	// We allow some existing files because holidays and weekends create gaps
	if foundExistingFiles > 0 && foundExistingFiles > newDownloads*3 {
		// Found way more existing than new files, probably in old territory
		logger.Info("Found mostly existing files, considering stopping",
			slog.Int("existing", foundExistingFiles),
			slog.Int("new", newDownloads))
		return newDownloads, foundExistingFiles, false, nil // Stop scraping
	}

	return newDownloads, foundExistingFiles, true, nil // Continue scraping
}

func downloadFile(url, dest string) error {
	// Get default logger for detailed logging
	logger := slog.Default()
	
	logger.Debug("Starting file download",
		slog.String("url", url),
		slog.String("destination", dest))
	
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("HTTP GET failed",
			slog.String("url", url),
			slog.String("error", err.Error()),
			slog.String("error_type", fmt.Sprintf("%T", err)))
		return fmt.Errorf("download failed for %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Bad HTTP status",
			slog.String("url", url),
			slog.Int("status_code", resp.StatusCode),
			slog.String("status", resp.Status))
		return fmt.Errorf("bad status for %s: %s", url, resp.Status)
	}
	
	// Log file creation attempt
	logger.Debug("Creating output file",
		slog.String("path", dest))
	
	out, err := os.Create(dest)
	if err != nil {
		logger.Error("Failed to create file",
			slog.String("path", dest),
			slog.String("error", err.Error()))
		return fmt.Errorf("create file %s: %w", dest, err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		logger.Error("Failed to write file content",
			slog.String("path", dest),
			slog.Int64("bytes_written", written),
			slog.String("error", err.Error()))
		return fmt.Errorf("write file %s: %w", dest, err)
	}
	
	logger.Info("File downloaded successfully",
		slog.String("file", filepath.Base(dest)),
		slog.Int64("size_bytes", written),
		slog.Float64("size_mb", float64(written)/1024/1024))
	
	return nil
}

func timedAction(name string, act chromedp.Action) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		start := time.Now()
		err := act.Do(ctx)
		// Note: Logger not available in this context without passing it through
		// This is acceptable for Chrome actions as they're internal operations
		_ = time.Since(start) // Avoid unused variable
		return err
	})
}

// latestDownloadedDate looks for files named "YYYY MM DD ISX Daily Report.xlsx" in dir and returns the most recent date.
func latestDownloadedDate(dir string) (time.Time, bool) {
	pattern := regexp.MustCompile(`^(\d{4}) (\d{2}) (\d{2}) ISX Daily Report\.xlsx$`)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, false
	}
	var dates []time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := pattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		t, err := time.Parse("2006 01 02", strings.Join(m[1:4], " "))
		if err == nil {
			dates = append(dates, t)
		}
	}
	if len(dates) == 0 {
		return time.Time{}, false
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates[len(dates)-1], true
}

func checkLicense(stateFilePath string, logger *slog.Logger) bool {
	// Get license path from centralized paths system
	licensePath, err := config.GetLicensePath()
	if err != nil {
		logger.Error("Failed to get license path", slog.String("error", err.Error()))
		return false
	}
	
	// Initialize license manager
	licenseManager, err := license.NewManager(licensePath)
	if err != nil {
		logger.Error("License system initialization failed", slog.String("error", err.Error()))
		return false
	}

	// Check state file first if provided
	if stateFilePath != "" {
		logger.Info("Checking license state file", slog.String("path", stateFilePath))
		valid, err := licenseManager.ValidateStateFile(stateFilePath)
		if err != nil {
			logger.Warn("State file validation error", slog.String("error", err.Error()))
			// Continue with normal validation
		} else if valid {
			slog.Info("✅ License validated via state file")
			slog.Info("═══════════════════════════════════════════════")
			logger.Info("License validated via state file")
			return true
		} else {
			slog.Info("⚠️  State file invalid or expired, proceeding with normal validation")
			logger.Warn("State file invalid or expired, proceeding with normal validation")
		}
	}

	// Check if license is valid
	valid, err := licenseManager.ValidateLicense()
	if valid {
		// Get license info for display
		info, infoErr := licenseManager.GetLicenseInfo()
		if infoErr == nil {
			daysLeft := int(time.Until(info.ExpiryDate).Hours() / 24)
			slog.Info("License Valid", "days_remaining", daysLeft)
			logger.Info("License Valid", slog.Int("days_remaining", daysLeft))
			if daysLeft <= 7 {
				slog.Warn("License expires soon", "expiry_date", info.ExpiryDate.Format("2006-01-02"))
				slog.Info("Contact The Iraqi Investor Group for license renewal")
				logger.Warn("License expires soon", 
					slog.String("expiry_date", info.ExpiryDate.Format("2006-01-02")),
					slog.String("action", "Contact The Iraqi Investor Group for license renewal"))
			}
		}
		slog.Info("═══════════════════════════════════════════════")
		return true
	}

	// License is invalid or expired
	slog.Info("❌ Invalid or Expired License")
	slog.Info("═══════════════════════════════════════════════")
	logger.Error("Invalid or Expired License")

	if err != nil {
		logger.Error("License validation error", slog.String("error", err.Error()))
		fmt.Printf("Error: %v\n", err)
	}

	// Prompt for license key activation
	slog.Info("Please enter your ISX license key to activate")
	slog.Info("License keys look like: ISX3M-ABC123DEF456GHI789JKL")
	slog.Info("License Key: (waiting for input...)")

	reader := bufio.NewReader(os.Stdin)
	licenseKey, _ := reader.ReadString('\n')
	licenseKey = strings.TrimSpace(licenseKey)

	if licenseKey == "" {
		slog.Info("❌ No license key provided.")
		logger.Error("No license key provided")
		return false
	}

	// Validate license key format
	if !isValidLicenseFormat(licenseKey) {
		slog.Info("❌ Invalid license key format.")
		slog.Info("   License keys should start with ISX1M, ISX3M, ISX6M, or ISX1Y")
		logger.Error("Invalid license key format")
		return false
	}

	// Activate license
	slog.Info("🔄 Activating license...")
	logger.Info("Activating license...")
	if err := licenseManager.ActivateLicense(licenseKey); err != nil {
		logger.Error("License activation failed", slog.String("error", err.Error()))
		fmt.Printf("❌ License activation failed: %v\n", err)
		slog.Info("📞 Please contact The Iraqi Investor Group if you believe this is an error.")
		return false
	}

	slog.Info("✅ License activated successfully!")
	slog.Info("🎉 Welcome to ISX Daily Reports Scraper!")
	slog.Info("═══════════════════════════════════════════════")
	logger.Info("License activated successfully")
	return true
}

func isValidLicenseFormat(licenseKey string) bool {
	// Check if license key starts with valid prefixes
	validPrefixes := []string{"ISX1M", "ISX3M", "ISX6M", "ISX1Y"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(licenseKey, prefix) {
			return true
		}
	}
	return false
}

// calculateExpectedFiles calculates the expected number of files based on date range
// ISX publishes reports on working days (Sunday-Thursday in Iraq)
func calculateExpectedFiles(fromStr, toStr string) int {
	// Parse dates
	startDate, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return 0
	}
	
	endDate := time.Now()
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			endDate = parsed
		}
	}
	
	// Don't count future dates
	today := time.Now()
	if endDate.After(today) {
		endDate = today
	}
	
	// Count working days between start and end
	count := 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		// ISX is closed on Friday and Saturday
		if d.Weekday() != time.Friday && d.Weekday() != time.Saturday {
			count++
		}
	}
	
	return count
}
