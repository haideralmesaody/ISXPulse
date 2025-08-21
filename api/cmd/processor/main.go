package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"isxcli/internal/config"
	"isxcli/internal/infrastructure"
	"isxcli/internal/dataprocessing"
	"isxcli/internal/license"
	"isxcli/pkg/contracts/domain"
)

// ExcelFileInfo holds information about an Excel file
type ExcelFileInfo struct {
	Name string
	Date time.Time
}

func main() {
	inDir := flag.String("in", "", "input directory for .xlsx files (defaults to data/downloads relative to executable)")
	outDir := flag.String("out", "", "output directory for CSV files (defaults to data/reports relative to executable)")
	fullRework := flag.Bool("full", false, "force full rework of all files")
	flag.Parse()

	// Initialize paths first to get default directories
	paths, err := config.GetPaths()
	if err != nil {
		slog.Error("Failed to initialize paths", "error", err)
		os.Exit(1)
	}

	// License validation (similar to scraper.exe)
	slog.Info("Validating license...")
	
	// Get license path from centralized paths system
	licensePath, err := config.GetLicensePath()
	if err != nil {
		slog.Error("Failed to get license path", "error", err)
		os.Exit(1)
	}
	
	// Initialize license manager
	licenseManager, err := license.NewManager(licensePath)
	if err != nil {
		slog.Error("License system initialization failed", "error", err)
		os.Exit(1)
	}
	
	// Check if license is valid
	valid, err := licenseManager.ValidateLicense()
	if !valid {
		if err != nil {
			slog.Error("License validation failed", "error", err)
		} else {
			slog.Error("Invalid or expired license")
		}
		os.Exit(1)
	}
	slog.Info("License validated successfully")

	// Use centralized directories as defaults if not specified
	if *inDir == "" {
		*inDir = paths.DownloadsDir
	}
	if *outDir == "" {
		*outDir = paths.ReportsDir
	}
	
	// Ensure all required directories exist
	if err := paths.EnsureDirectories(); err != nil {
		slog.Error("Failed to create required directories", "error", err)
		os.Exit(1)
	}

	// Initialize structured logger per CLAUDE.md
	cfg, err := config.Load()
	if err != nil {
		slog.Warn("Failed to load config, using defaults", "error", err)
		cfg = &config.Config{
			Logging: config.LoggingConfig{
				Level:       "info",
				Format:      "json",
				Output:      "both",
				FilePath:    paths.GetLogPath("process.log"),
				Development: false,
			},
		}
	}

	logger, err := infrastructure.InitializeLogger(cfg.Logging)
	if err != nil {
		slog.Warn("Failed to initialize logger, using default", "error", err)
		logger = slog.Default()
	}

	logger.Info("Starting ISX Daily Reports processing",
		slog.String("input_dir", *inDir),
		slog.String("output_dir", *outDir),
		slog.Bool("full_rework", *fullRework),
		slog.String("executable_dir", paths.ExecutableDir))

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		logger.Error("Error creating output directory", slog.String("error", err.Error()))
		slog.Error("Error creating output directory", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting ISX Daily Reports processing...")
	logger.Info("Processing directories",
		slog.String("input_dir", *inDir),
		slog.String("output_dir", *outDir),
		slog.String("executable_dir", paths.ExecutableDir))
	slog.Info("Full rework mode", "enabled", *fullRework)

	// Get all available Excel files
	files, err := ioutil.ReadDir(*inDir)
	if err != nil {
		logger.Error("Failed to read input directory", slog.String("error", err.Error()))
		slog.Error("Failed to read input directory", "error", err)
		os.Exit(1)
	}

	// Parse and sort all available files by date
	var excelFiles []ExcelFileInfo
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".xlsx") || strings.HasPrefix(file.Name(), "~$") {
			continue
		}

		// Extract date from filename (e.g., "YYYY MM DD ISX Daily Report.xlsx")
		parts := strings.Split(file.Name(), " ")
		if len(parts) < 4 {
			continue // Skip malformed filenames
		}

		dateStr := strings.Join(parts[0:3], " ")
		date, err := time.Parse("2006 01 02", dateStr)
		if err != nil {
				logger.Warn("Could not parse date from filename",
				slog.String("filename", file.Name()),
				slog.String("error", err.Error()))
			continue
		}

		excelFiles = append(excelFiles, ExcelFileInfo{
			Name: file.Name(),
			Date: date,
		})
	}

	// Sort files by date
	sort.Slice(excelFiles, func(i, j int) bool {
		return excelFiles[i].Date.Before(excelFiles[j].Date)
	})

	logger.Info("Excel files discovered", slog.Int("count", len(excelFiles)))
	
	// Output progress message for stages.go to parse
	fmt.Printf("Found %d Excel files\n", len(excelFiles))
	
	// Graceful exit if no Excel files found
	if len(excelFiles) == 0 {
		logger.Warn("No Excel files found in input directory",
			slog.String("input_dir", *inDir),
			slog.String("pattern", "*.xlsx"))
		
		// Create empty but valid output structure
		slog.Info("No files to process, creating empty output structure")
		
		// Ensure output directory exists
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			logger.Error("Failed to create output directory", slog.String("error", err.Error()))
			os.Exit(1)
		}
		
		// Create empty combined CSV with headers in proper subdirectory
		combinedDir := filepath.Join(*outDir, "combined")
		if err := os.MkdirAll(combinedDir, 0755); err != nil {
			logger.Error("Failed to create combined directory", slog.String("error", err.Error()))
			os.Exit(1)
		}
		combinedCSVPath := filepath.Join(combinedDir, "isx_combined_data.csv")
		if err := saveCombinedCSV(combinedCSVPath, []domain.TradeRecord{}); err != nil {
			logger.Error("Failed to create empty combined CSV", slog.String("error", err.Error()))
			os.Exit(1)
		}
		
		logger.Info("Created empty output files", slog.String("combined_csv", combinedCSVPath))
		fmt.Println("Processing complete: 0 files")
		fmt.Println("All files processed")
		return
	}
	
	// Output file list for stages.go to parse (for segmented progress)
	if len(excelFiles) > 0 {
		var fileNames []string
		for _, f := range excelFiles {
			fileNames = append(fileNames, f.Name)
		}
		fmt.Printf("Files to process: %s\n", strings.Join(fileNames, "|"))
	}

	// Check what needs to be processed
	var filesToProcess []ExcelFileInfo
	var existingRecords []domain.TradeRecord

	if *fullRework {
		slog.Info("Full rework requested - processing all files")
		filesToProcess = excelFiles
	} else {
		// Smart update: check what's already processed
		filesToProcess, existingRecords = determineFilesToProcess(excelFiles, *outDir, logger)
		logger.Info("Smart update status", slog.Int("files_to_process", len(filesToProcess)))
	}

	// Process the required files
	var newRecords []domain.TradeRecord
	totalFiles := len(filesToProcess)

	for i, fileInfo := range filesToProcess {
		logger.Info("Processing file",
			slog.Int("current", i+1),
			slog.Int("total", totalFiles),
			slog.String("filename", fileInfo.Name))
		
		// Output progress message for stages.go to parse
		fmt.Printf("Processing file %d of %d: %s\n", i+1, totalFiles, fileInfo.Name)

		report, err := dataprocessing.ParseFile(filepath.Join(*inDir, fileInfo.Name))
		if err != nil {
			logger.Error("Error parsing file",
				slog.String("filename", fileInfo.Name),
				slog.String("error", err.Error()))
			continue
		}

		// Update all records with the correct date
		for i := range report.Records {
			report.Records[i].Date = fileInfo.Date
		}

		logger.Info("Records processed from file",
			slog.Int("record_count", len(report.Records)),
			slog.String("filename", fileInfo.Name))

		// Note: Daily CSV files will be generated after forward-fill processing
		// to ensure they include forward-filled data with proper trading status

		// Add to new records
		newRecords = append(newRecords, report.Records...)

		// Log sample records for verification
		for i, record := range report.Records {
			if i >= 3 { // Log up to 3 records
				break
			}
			logger.Debug("Sample record processed",
				slog.String("symbol", record.CompanySymbol),
				slog.String("company", record.CompanyName),
				slog.String("date", record.Date.Format("2006-01-02")),
				slog.Float64("close_price", record.ClosePrice),
				slog.Int64("volume", record.Volume))
		}
	}

	// Combine existing and new records
	allRecords := append(existingRecords, newRecords...)

	// Apply forward-fill and generate all output files
	if len(allRecords) > 0 {
		slog.Info("Generating dataset with forward-fill...")
		filledRecords := forwardFillMissingData(allRecords)

		logger.Info("Record processing summary",
			slog.Int("total_records", len(filledRecords)),
			slog.Int("active_trading_records", len(allRecords)),
			slog.Int("forward_filled_records", len(filledRecords)-len(allRecords)))

		// Save combined CSV with forward-fill in proper subdirectory
		combinedDir := filepath.Join(*outDir, "combined")
		if err := os.MkdirAll(combinedDir, 0755); err != nil {
			logger.Error("Failed to create combined directory", slog.String("error", err.Error()))
			return
		}
		combinedCSVPath := filepath.Join(combinedDir, "isx_combined_data.csv")
		if err := saveCombinedCSV(combinedCSVPath, filledRecords); err != nil {
				logger.Error("Error saving combined CSV", slog.String("error", err.Error()))
			slog.Error("Error saving combined CSV", "error", err)
		} else {
			logger.Info("Saved combined report", slog.String("path", combinedCSVPath))
		}

		// Generate daily CSV files with forward-fill in proper subdirectory
		slog.Info("Generating daily CSV files with forward-fill...")
		dailyDir := filepath.Join(*outDir, "daily")
		if err := os.MkdirAll(dailyDir, 0755); err != nil {
			logger.Error("Failed to create daily directory", slog.String("error", err.Error()))
			return
		}
		if err := generateDailyFiles(filledRecords, dailyDir); err != nil {
			logger.Error("Error generating daily files", slog.String("error", err.Error()))
			slog.Error("Error generating daily files", "error", err)
		} else {
			logger.Info("Daily files generated successfully")
			slog.Info("Daily files generated successfully")
		}

		// Generate individual ticker CSV files with forward-fill in proper subdirectory
		slog.Info("Generating individual ticker CSV files with forward-fill...")
		tickerDir := filepath.Join(*outDir, "ticker")
		if err := os.MkdirAll(tickerDir, 0755); err != nil {
			logger.Error("Failed to create ticker directory", slog.String("error", err.Error()))
			return
		}
		if err := generateTickerFiles(filledRecords, tickerDir); err != nil {
			logger.Error("Error generating ticker files", slog.String("error", err.Error()))
			slog.Error("Error generating ticker files", "error", err)
		} else {
			logger.Info("Ticker files generated successfully")
			slog.Info("Ticker files generated successfully")
		}
	}

	logger.Info("Processing complete")
	
	// Output completion message for stages.go to parse
	fmt.Printf("Processing complete: %d files\n", len(filesToProcess))

	// Generate ticker summary using SSOT Summarizer
	logger.Info("Generating ticker summary using SSOT implementation")
	ctx := context.Background()
	integrator := dataprocessing.NewIntegrationExample(logger)
	combinedCSVPath := filepath.Join(*outDir, "combined", "isx_combined_data.csv")
	
	if err := integrator.GenerateTickerSummaryFromCombinedCSV(ctx, combinedCSVPath, *outDir); err != nil {
		logger.Warn("Failed to generate ticker summary using SSOT", slog.String("error", err.Error()))
		slog.Warn("Failed to generate ticker summary using SSOT", "error", err)
	} else {
		logger.Info("Ticker summary generated successfully using SSOT")
	}
	
	// Output completion message for stages.go to parse
	fmt.Println("All files processed")
}

// determineFilesToProcess checks which files need to be processed based on existing CSV files
func determineFilesToProcess(excelFiles []ExcelFileInfo, outDir string, logger *slog.Logger) ([]ExcelFileInfo, []domain.TradeRecord) {
	var filesToProcess []ExcelFileInfo
	var existingRecords []domain.TradeRecord

	// Check which daily CSV files already exist in the new directory structure
	existingDates := make(map[string]bool)
	dailyDir := filepath.Join(outDir, "daily")
	
	// Walk through all daily subdirectories
	filepath.Walk(dailyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}
		
		if !info.IsDir() && strings.HasPrefix(info.Name(), "isx_daily_") && strings.HasSuffix(info.Name(), ".csv") {
			// Extract date from filename: isx_daily_YYYY_MM_DD.csv
			dateStr := strings.TrimPrefix(info.Name(), "isx_daily_")
			dateStr = strings.TrimSuffix(dateStr, ".csv")
			existingDates[dateStr] = true
		}
		return nil
	})

	logger.Info("Found existing daily CSV files", slog.Int("count", len(existingDates)))

	// Load existing records from combined CSV if it exists
	combinedCSVPath := filepath.Join(outDir, "combined", "isx_combined_data.csv")
	if _, err := os.Stat(combinedCSVPath); err == nil {
		logger.Info("Loading existing combined CSV data")
		slog.Info("Loading existing combined CSV data...")
		if records, err := loadExistingRecords(combinedCSVPath); err == nil {
			existingRecords = records
			logger.Info("Loaded existing records", slog.Int("count", len(existingRecords)))
		} else {
			logger.Warn("Could not load existing combined CSV", slog.String("error", err.Error()))
			slog.Warn("Could not load existing combined CSV", "error", err)
		}
	}

	// Determine which files need processing
	for _, fileInfo := range excelFiles {
		dateStr := fileInfo.Date.Format("2006_01_02")
		if !existingDates[dateStr] {
			filesToProcess = append(filesToProcess, fileInfo)
			logger.Info("Need to process file",
				slog.String("filename", fileInfo.Name),
				slog.String("date", dateStr))
		} else {
			logger.Info("Already processed file",
				slog.String("filename", fileInfo.Name),
				slog.String("date", dateStr))
		}
	}

	// If we have existing records but files to process, we need to filter out records for dates we're reprocessing
	if len(existingRecords) > 0 && len(filesToProcess) > 0 {
		logger.Info("Filtering existing records to avoid duplicates")
		slog.Info("Filtering existing records to avoid duplicates...")
		reprocessDates := make(map[string]bool)
		for _, fileInfo := range filesToProcess {
			reprocessDates[fileInfo.Date.Format("2006-01-02")] = true
		}

		var filteredRecords []domain.TradeRecord
		originalCount := len(existingRecords)
		for _, record := range existingRecords {
			if !reprocessDates[record.Date.Format("2006-01-02")] {
				filteredRecords = append(filteredRecords, record)
			}
		}
		existingRecords = filteredRecords
		logger.Info("Filtered existing records", 
			slog.Int("remaining_records", len(existingRecords)),
			slog.Int("removed_records", originalCount-len(filteredRecords)))
	}

	return filesToProcess, existingRecords
}

// loadExistingRecords loads records from an existing combined CSV file
func loadExistingRecords(filePath string) ([]domain.TradeRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var tradeRecords []domain.TradeRecord
	for i, record := range records {
		if i == 0 { // Skip header
			continue
		}

		if len(record) < 16 {
			continue // Skip malformed records
		}

		// Parse the record
		date, _ := time.Parse("2006-01-02", record[0])
		openPrice, _ := strconv.ParseFloat(record[3], 64)
		highPrice, _ := strconv.ParseFloat(record[4], 64)
		lowPrice, _ := strconv.ParseFloat(record[5], 64)
		avgPrice, _ := strconv.ParseFloat(record[6], 64)
		prevAvgPrice, _ := strconv.ParseFloat(record[7], 64)
		closePrice, _ := strconv.ParseFloat(record[8], 64)
		prevClosePrice, _ := strconv.ParseFloat(record[9], 64)
		change, _ := strconv.ParseFloat(record[10], 64)
		changePct, _ := strconv.ParseFloat(record[11], 64)
		numTrades, _ := strconv.ParseInt(record[12], 10, 64)
		volume, _ := strconv.ParseInt(record[13], 10, 64)
		value, _ := strconv.ParseFloat(record[14], 64)
		tradingStatus, _ := strconv.ParseBool(record[15])

		tradeRecord := domain.TradeRecord{
			CompanyName:      record[1],
			CompanySymbol:    record[2],
			Date:             date,
			OpenPrice:        openPrice,
			HighPrice:        highPrice,
			LowPrice:         lowPrice,
			AveragePrice:     avgPrice,
			PrevAveragePrice: prevAvgPrice,
			ClosePrice:       closePrice,
			PrevClosePrice:   prevClosePrice,
			Change:           change,
			ChangePercent:    changePct,
			NumTrades:        numTrades,
			Volume:           volume,
			Value:            value,
			TradingStatus:    tradingStatus,
		}
		tradeRecords = append(tradeRecords, tradeRecord)
	}

	return tradeRecords, nil
}

func saveDailyCSV(filePath string, records []domain.TradeRecord) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header with all fields
	header := []string{
		"Date", "CompanyName", "Symbol", "OpenPrice", "HighPrice", "LowPrice",
		"AveragePrice", "PrevAveragePrice", "ClosePrice", "PrevClosePrice",
		"Change", "ChangePercent", "NumTrades", "Volume", "Value", "TradingStatus",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write records
	for _, record := range records {
		row := []string{
			record.Date.Format("2006-01-02"),
			record.CompanyName,
			record.CompanySymbol,
			fmt.Sprintf("%.3f", record.OpenPrice),
			fmt.Sprintf("%.3f", record.HighPrice),
			fmt.Sprintf("%.3f", record.LowPrice),
			fmt.Sprintf("%.3f", record.AveragePrice),
			fmt.Sprintf("%.3f", record.PrevAveragePrice),
			fmt.Sprintf("%.3f", record.ClosePrice),
			fmt.Sprintf("%.3f", record.PrevClosePrice),
			fmt.Sprintf("%.3f", record.Change),
			fmt.Sprintf("%.2f", record.ChangePercent),
			fmt.Sprintf("%d", record.NumTrades),
			fmt.Sprintf("%d", record.Volume),
			fmt.Sprintf("%.2f", record.Value),
			fmt.Sprintf("%t", record.TradingStatus),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// forwardFillMissingData fills in missing trading data for symbols that don't trade on certain days
func forwardFillMissingData(records []domain.TradeRecord) []domain.TradeRecord {
	if len(records) == 0 {
		return records
	}

	// Group records by symbol and date
	symbolsByDate := make(map[string]map[string]domain.TradeRecord) // date -> symbol -> record
	allSymbols := make(map[string]bool)
	allDates := make(map[string]bool)

	for _, record := range records {
		dateStr := record.Date.Format("2006-01-02")
		symbol := record.CompanySymbol

		if symbolsByDate[dateStr] == nil {
			symbolsByDate[dateStr] = make(map[string]domain.TradeRecord)
		}
		symbolsByDate[dateStr][symbol] = record
		allSymbols[symbol] = true
		allDates[dateStr] = true
	}

	// Convert to sorted slices
	var dates []string
	for date := range allDates {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	var symbols []string
	for symbol := range allSymbols {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	// Keep track of last known data for each symbol
	lastKnownData := make(map[string]domain.TradeRecord)

	var result []domain.TradeRecord

	for _, dateStr := range dates {
		date, _ := time.Parse("2006-01-02", dateStr)
		dayRecords := symbolsByDate[dateStr]

		for _, symbol := range symbols {
			if record, exists := dayRecords[symbol]; exists {
				// Symbol traded on this day - use actual data
				result = append(result, record)
				lastKnownData[symbol] = record
			} else if lastRecord, hasHistory := lastKnownData[symbol]; hasHistory {
				// Symbol didn't trade - forward fill from last known data
				filledRecord := domain.TradeRecord{
					CompanyName:      lastRecord.CompanyName,
					CompanySymbol:    symbol,
					Date:             date,
					OpenPrice:        lastRecord.ClosePrice,   // Open = previous close
					HighPrice:        lastRecord.ClosePrice,   // High = previous close
					LowPrice:         lastRecord.ClosePrice,   // Low = previous close
					AveragePrice:     lastRecord.ClosePrice,   // Average = previous close
					PrevAveragePrice: lastRecord.AveragePrice, // Keep previous average
					ClosePrice:       lastRecord.ClosePrice,   // Close = previous close
					PrevClosePrice:   lastRecord.ClosePrice,   // Prev close = previous close
					Change:           0.0,                     // No change
					ChangePercent:    0.0,                     // No change %
					NumTrades:        0,                       // No trades
					Volume:           0,                       // No volume
					Value:            0.0,                     // No value
					TradingStatus:    false,                   // Forward-filled data
				}
				result = append(result, filledRecord)
				// Don't update lastKnownData since this is filled data
			}
			// If no history exists, skip this symbol for this date
		}
	}

	return result
}

func saveCombinedCSV(filePath string, records []domain.TradeRecord) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header with all fields
	header := []string{
		"Date", "CompanyName", "Symbol", "OpenPrice", "HighPrice", "LowPrice",
		"AveragePrice", "PrevAveragePrice", "ClosePrice", "PrevClosePrice",
		"Change", "ChangePercent", "NumTrades", "Volume", "Value", "TradingStatus",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write records
	for _, record := range records {
		row := []string{
			record.Date.Format("2006-01-02"),
			record.CompanyName,
			record.CompanySymbol,
			fmt.Sprintf("%.3f", record.OpenPrice),
			fmt.Sprintf("%.3f", record.HighPrice),
			fmt.Sprintf("%.3f", record.LowPrice),
			fmt.Sprintf("%.3f", record.AveragePrice),
			fmt.Sprintf("%.3f", record.PrevAveragePrice),
			fmt.Sprintf("%.3f", record.ClosePrice),
			fmt.Sprintf("%.3f", record.PrevClosePrice),
			fmt.Sprintf("%.3f", record.Change),
			fmt.Sprintf("%.2f", record.ChangePercent),
			fmt.Sprintf("%d", record.NumTrades),
			fmt.Sprintf("%d", record.Volume),
			fmt.Sprintf("%.2f", record.Value),
			fmt.Sprintf("%t", record.TradingStatus),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// generateDailyFiles generates daily CSV files grouped by date from forward-filled records
func generateDailyFiles(records []domain.TradeRecord, outDir string) error {
	// Group records by date
	recordsByDate := make(map[string][]domain.TradeRecord)
	for _, record := range records {
		dateStr := record.Date.Format("2006_01_02")
		recordsByDate[dateStr] = append(recordsByDate[dateStr], record)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Generate CSV files for each date
	for dateStr, dailyRecords := range recordsByDate {
		slog.Debug("Generating daily CSV for date", slog.String("date", dateStr))

		// Save CSV directly to reports directory (no subdirectories)
		dailyCSVPath := filepath.Join(outDir, fmt.Sprintf("isx_daily_%s.csv", dateStr))
		if err := saveDailyCSV(dailyCSVPath, dailyRecords); err != nil {
			slog.Error("Error saving daily CSV",
				slog.String("path", dailyCSVPath),
				slog.String("error", err.Error()))
		} else {
			slog.Debug("Saved daily CSV",
				slog.String("path", dailyCSVPath),
				slog.Int("record_count", len(dailyRecords)))
		}
	}

	return nil
}

// generateTickerFiles generates individual CSV files for each ticker with their complete trading history
func generateTickerFiles(records []domain.TradeRecord, outDir string) error {
	// Extract all unique tickers
	tickers := make(map[string]bool)
	for _, record := range records {
		tickers[record.CompanySymbol] = true
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Generate CSV files for each ticker
	for ticker := range tickers {
		slog.Debug("Generating CSV for ticker", slog.String("ticker", ticker))

		// Filter records for the current ticker
		var tickerRecords []domain.TradeRecord
		for _, record := range records {
			if record.CompanySymbol == ticker {
				tickerRecords = append(tickerRecords, record)
			}
		}

		// Save CSV directly to reports directory (no sector-based folders)
		tickerCSVPath := filepath.Join(outDir, fmt.Sprintf("%s_trading_history.csv", ticker))
		if err := saveDailyCSV(tickerCSVPath, tickerRecords); err != nil {
			slog.Error("Error saving ticker CSV",
				slog.String("ticker", ticker),
				slog.String("path", tickerCSVPath),
				slog.String("error", err.Error()))
		} else {
			slog.Debug("Saved ticker CSV",
				slog.String("ticker", ticker),
				slog.String("path", tickerCSVPath))
		}
	}

	return nil
}



