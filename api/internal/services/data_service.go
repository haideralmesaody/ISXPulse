package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"isxcli/internal/config"
	"isxcli/pkg/contracts/domain"
)

// DataService provides data access functionality
type DataService struct {
	config *config.Config
	paths  *config.Paths
	logger *slog.Logger
}

// NewDataService creates a new data service using default logger
func NewDataService(cfg *config.Config) (*DataService, error) {
	return NewDataServiceWithLogger(cfg, slog.Default())
}

// NewDataServiceWithLogger creates a new data service with a specific logger
func NewDataServiceWithLogger(cfg *config.Config, logger *slog.Logger) (*DataService, error) {
	// Get the centralized paths
	paths, err := config.GetPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get paths: %w", err)
	}
	
	// Ensure we have a logger
	if logger == nil {
		logger = slog.Default()
	}
	
	// Log startup paths for visibility using injected logger
	logger.Info("DataService initialized with paths",
		slog.String("data_dir", paths.DataDir),
		slog.String("reports_dir", paths.ReportsDir),
		slog.String("downloads_dir", paths.DownloadsDir))
	
	return &DataService{
		config: cfg,
		paths:  paths,
		logger: logger,
	}, nil
}

// GetReports returns a list of available reports with categorization
func (ds *DataService) GetReports(ctx context.Context) ([]map[string]interface{}, error) {
	reportsDir := ds.paths.ReportsDir
	
	// Use injected logger
	ds.logger.Debug("GetReports: scanning directory",
		slog.String("reports_dir", reportsDir))
	
	var reports []map[string]interface{}
	
	// Define report categories and their directories
	reportDirs := map[string]string{
		"daily":     filepath.Join(reportsDir, "daily"),
		"ticker":    filepath.Join(reportsDir, "ticker"),
		"liquidity": filepath.Join(reportsDir, "liquidity_reports"), // Updated to new folder name
		"summary":   filepath.Join(reportsDir, "summary"),
		"combined":  filepath.Join(reportsDir, "combined"),
		"indexes":   filepath.Join(reportsDir, "indexes"),
	}
	
	// Scan each category directory
	for category, dir := range reportDirs {
		// Walk through the directory recursively
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Log but continue with other files
				ds.logger.Debug("Error accessing path",
					slog.String("path", path),
					slog.String("error", err.Error()))
				return nil
			}
			
			// Skip directories and non-CSV/JSON files
			if info.IsDir() {
				return nil
			}
			
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext != ".csv" && ext != ".json" && ext != ".txt" {
				return nil
			}
			
			// Get relative path from reports directory
			relPath, err := filepath.Rel(reportsDir, path)
			if err != nil {
				ds.logger.Debug("Failed to get relative path",
					slog.String("path", path),
					slog.String("error", err.Error()))
				return nil
			}
			
			// Convert backslashes to forward slashes for consistency
			relPath = strings.ReplaceAll(relPath, "\\", "/")
			
			reports = append(reports, map[string]interface{}{
				"name":     info.Name(),
				"path":     relPath,
				"category": category,
				"size":     info.Size(),
				"modified": info.ModTime(),
				"fullPath": path,
			})
			
			return nil
		})
		
		if err != nil {
			// Log but continue with other categories
			ds.logger.Debug("Error walking directory",
				slog.String("category", category),
				slog.String("dir", dir),
				slog.String("error", err.Error()))
		}
	}
	
	// Also check root reports directory for any files (processor outputs to root now)
	rootFiles, err := os.ReadDir(reportsDir)
	if err == nil {
		for _, file := range rootFiles {
			if !file.IsDir() {
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if ext == ".csv" || ext == ".json" || ext == ".txt" {
					info, err := file.Info()
					if err != nil {
						continue
					}
					
					// Detect category based on filename pattern
					category := "uncategorized"
					fileName := file.Name()
					
					if strings.HasPrefix(fileName, "isx_daily_") {
						category = "daily"
					} else if strings.HasSuffix(fileName, "_trading_history.csv") {
						category = "ticker"
					} else if strings.HasPrefix(fileName, "liquidity_scores_") {
						category = "liquidity"
					} else if strings.HasPrefix(fileName, "liquidity_insights_") {
						// Mark as liquidity_insights so frontend can filter it out
						category = "liquidity_insights"
					} else if strings.Contains(fileName, "combined") {
						category = "combined"
					} else if strings.Contains(fileName, "index") {
						category = "indexes"
					} else if strings.Contains(fileName, "summary") {
						category = "summary"
					}
					
					reports = append(reports, map[string]interface{}{
						"name":     file.Name(),
						"path":     file.Name(),
						"category": category,
						"size":     info.Size(),
						"modified": info.ModTime(),
						"fullPath": filepath.Join(reportsDir, file.Name()),
					})
				}
			}
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i]["modified"].(time.Time).After(reports[j]["modified"].(time.Time))
	})

	ds.logger.Debug("GetReports: found reports",
		slog.Int("count", len(reports)))

	return reports, nil
}

// GetTickers returns ticker information
func (ds *DataService) GetTickers(ctx context.Context) (interface{}, error) {
	tickerFile := ds.paths.GetTickerSummaryJSONPath()
	
	// Use injected logger
	ds.logger.Debug("GetTickers: reading ticker summary",
		slog.String("ticker_file", tickerFile))
	
	data, err := os.ReadFile(tickerFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read ticker summary: %w", err)
	}

	var tickerData interface{}
	if err := json.Unmarshal(data, &tickerData); err != nil {
		return nil, fmt.Errorf("failed to parse ticker summary: %w", err)
	}

	return tickerData, nil
}

// GetIndices returns market indices data
func (ds *DataService) GetIndices(ctx context.Context) (map[string]interface{}, error) {
	indicesFile := ds.paths.GetIndexCSVPath()
	
	// Use injected logger
	ds.logger.Debug("GetIndices: reading indices file",
		slog.String("indices_file", indicesFile))
	
	file, err := os.Open(indicesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{
				"dates": []string{},
				"isx60": []float64{},
				"isx15": []float64{},
			}, nil
		}
		return nil, fmt.Errorf("failed to open indices file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	
	// Validate header
	if len(header) < 2 || header[0] != "Date" || header[1] != "ISX60" {
		return nil, fmt.Errorf("invalid CSV header format")
	}
	
	var dates []string
	var isx60Values []float64
	var isx15Values []float64
	
	// Read data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}
		
		if len(record) < 2 {
			continue // Skip invalid rows
		}
		
		// Parse date
		dates = append(dates, record[0])
		
		// Parse ISX60
		isx60, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			logDataError(ctx, "parse_value", "Failed to parse ISX60 value",
				slog.String("value", record[1]),
				slog.String("error", err.Error()),
			)
			isx60 = 0
		}
		isx60Values = append(isx60Values, isx60)
		
		// Parse ISX15 if present
		if len(record) > 2 && record[2] != "" {
			isx15, err := strconv.ParseFloat(record[2], 64)
			if err != nil {
				logDataError(ctx, "parse_value", "Failed to parse ISX15 value",
					slog.String("value", record[2]),
					slog.String("error", err.Error()),
				)
				isx15 = 0
			}
			isx15Values = append(isx15Values, isx15)
		} else {
			isx15Values = append(isx15Values, 0)
		}
	}
	
	return map[string]interface{}{
		"dates": dates,
		"isx60": isx60Values,
		"isx15": isx15Values,
	}, nil
}

// GetFiles returns file listings from different directories
func (ds *DataService) GetFiles(ctx context.Context) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"downloads":     []interface{}{},
		"reports":       []interface{}{},
		"csvFiles":      []interface{}{},
		"total_size":    int64(0),
		"last_modified": time.Time{},
	}

	// List downloaded Excel files
	if err := ds.listFiles("downloads", ".xlsx", result); err != nil {
		logDataError(ctx, "list_files", "Failed to list downloads",
			slog.String("error", err.Error()),
		)
	}

	// List report files
	if err := ds.listFiles("reports", ".csv", result); err != nil {
		logDataError(ctx, "list_files", "Failed to list reports",
			slog.String("error", err.Error()),
		)
	}

	return result, nil
}

// GetMarketMovers returns market movers data
func (ds *DataService) GetMarketMovers(ctx context.Context, period, limit, minVolume string) (map[string]interface{}, error) {
	// Default values
	if period == "" {
		period = "1d"
	}
	if limit == "" {
		limit = "10"
	}
	if minVolume == "" {
		minVolume = "0"
	}

	return map[string]interface{}{
		"gainers":    []interface{}{},
		"losers":     []interface{}{},
		"mostActive": []interface{}{},
		"period":     period,
		"updated":    time.Now().Format(time.RFC3339),
	}, nil
}

// GetTickerChart returns chart data for a specific ticker
func (ds *DataService) GetTickerChart(ctx context.Context, ticker string) (map[string]interface{}, error) {
	if ticker == "" {
		return nil, fmt.Errorf("ticker parameter required")
	}

	tickerFile := ds.paths.GetTickerDailyCSVPath(ticker)
	
	logger := slog.Default()
	if logger != nil {
		logger.Debug("GetTickerChart: reading ticker data",
			slog.String("ticker", ticker),
			slog.String("ticker_file", tickerFile))
	}
	
	_, err := os.Stat(tickerFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{
				"ticker": ticker,
				"data":   []interface{}{},
			}, nil
		}
		return nil, fmt.Errorf("failed to check ticker file: %w", err)
	}

	// For now, return empty structure - implement CSV parsing later
	return map[string]interface{}{
		"ticker": ticker,
		"data":   []interface{}{},
	}, nil
}

// GetDailyReport returns data for a specific date
func (ds *DataService) GetDailyReport(ctx context.Context, date time.Time) ([]map[string]interface{}, error) {
	dailyFile := ds.paths.GetDailyCSVPath(date)
	
	logger := slog.Default()
	if logger != nil {
		logger.Debug("GetDailyReport: reading daily report",
			slog.Time("date", date),
			slog.String("daily_file", dailyFile))
	}
	
	file, err := os.Open(dailyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to open daily report: %w", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	
	var results []map[string]interface{}
	
	// Read data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}
		
		// Convert record to map
		row := make(map[string]interface{})
		for i, value := range record {
			if i < len(header) {
				row[header[i]] = value
			}
		}
		
		results = append(results, row)
	}
	
	return results, nil
}

// DownloadFile serves a file for download (supports nested paths)
func (ds *DataService) DownloadFile(ctx context.Context, w http.ResponseWriter, r *http.Request, fileType, filename string) error {
	var dir string
	switch fileType {
	case "downloads":
		dir = ds.paths.DownloadsDir
	case "reports", "report", "csv": // Support multiple aliases for reports
		dir = ds.paths.ReportsDir
	default:
		return fmt.Errorf("invalid file type: %s", fileType)
	}
	
	// Use injected logger
	ds.logger.Debug("DownloadFile: serving file",
		slog.String("file_type", fileType),
		slog.String("filename", filename),
		slog.String("directory", dir))
	
	// The filename can now be a relative path with subdirectories
	// Clean the path to prevent directory traversal attacks
	cleanedFilename := filepath.Clean(filename)
	
	// Convert forward slashes to OS-specific separator
	cleanedFilename = filepath.FromSlash(cleanedFilename)
	
	// Log path transformation for debugging
	ds.logger.Debug("Path transformation",
		slog.String("original", filename),
		slog.String("cleaned", cleanedFilename),
		slog.String("base_dir", dir))
	
	// Security check - ensure the file is within the expected directory
	filePath := filepath.Join(dir, cleanedFilename)
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		ds.logger.Error("Failed to resolve absolute path",
			slog.String("error", err.Error()),
			slog.String("file_path", filePath))
		return fmt.Errorf("invalid file path")
	}
	
	absDir, err := filepath.Abs(dir)
	if err != nil {
		ds.logger.Error("Failed to resolve directory path",
			slog.String("error", err.Error()),
			slog.String("dir", dir))
		return fmt.Errorf("invalid directory path")
	}
	
	// Normalize paths for comparison (important on Windows)
	absFilePath = filepath.Clean(absFilePath)
	absDir = filepath.Clean(absDir)
	
	// Ensure the resolved path is within the allowed directory
	if !strings.HasPrefix(absFilePath, absDir) {
		ds.logger.Warn("Attempted directory traversal",
			slog.String("requested_path", filename),
			slog.String("resolved_path", absFilePath),
			slog.String("base_dir", absDir))
		return fmt.Errorf("invalid file path")
	}
	
	// Check if file exists
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		ds.logger.Warn("File not found",
			slog.String("requested_file", filename),
			slog.String("cleaned_file", cleanedFilename),
			slog.String("full_path", absFilePath),
			slog.String("base_dir", dir))
		return fmt.Errorf("file not found")
	}

	// Set headers for download
	// Use just the filename (not the full path) in the header
	baseFilename := filepath.Base(cleanedFilename)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", baseFilename))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Serve the file
	http.ServeFile(w, r, absFilePath)
	return nil
}

// listFiles lists files in a directory with filtering
func (ds *DataService) listFiles(dirName, extension string, result map[string]interface{}) error {
	var dir string
	switch dirName {
	case "downloads":
		dir = ds.paths.DownloadsDir
	case "reports":
		dir = ds.paths.ReportsDir
	default:
		dir = filepath.Join(ds.paths.DataDir, dirName)
	}
	
	logger := slog.Default()
	if logger != nil {
		logger.Debug("listFiles: scanning directory",
			slog.String("dir_name", dirName),
			slog.String("directory", dir),
			slog.String("extension", extension))
	}
	
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var fileList []map[string]interface{}
	var totalSize int64
	var lastModified time.Time

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), extension) {
			info, err := file.Info()
			if err != nil {
				continue
			}

			fileInfo := map[string]interface{}{
				"name":     file.Name(),
				"size":     info.Size(),
				"modified": info.ModTime().Format(time.RFC3339),
			}

			fileList = append(fileList, fileInfo)
			totalSize += info.Size()

			if info.ModTime().After(lastModified) {
				lastModified = info.ModTime()
			}
		}
	}

	// Sort files by modification time (newest first)
	sort.Slice(fileList, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, fileList[i]["modified"].(string))
		timeJ, _ := time.Parse(time.RFC3339, fileList[j]["modified"].(string))
		return timeI.After(timeJ)
	})

	// Update result based on directory
	switch dirName {
	case "downloads":
		// Convert to []interface{}
		downloads := make([]interface{}, len(fileList))
		for i, f := range fileList {
			downloads[i] = f
		}
		result["downloads"] = downloads
	case "reports":
		// Separate ticker CSV files from other reports
		var reports []interface{}
		var csvFiles []interface{}

		for _, file := range fileList {
			name := file["name"].(string)
			if strings.Contains(name, "_trading_history.csv") || strings.Contains(name, "isx_daily_") {
				csvFiles = append(csvFiles, file)
			} else {
				reports = append(reports, file)
			}
		}

		// Convert empty slices to proper type
		if reports == nil {
			reports = []interface{}{}
		}
		if csvFiles == nil {
			csvFiles = []interface{}{}
		}

		result["reports"] = reports
		result["csvFiles"] = csvFiles
	}

	result["total_size"] = totalSize
	result["last_modified"] = lastModified

	return nil
}

// GetSafeTradingLimits returns safe trading limits for a ticker based on liquidity metrics
func (ds *DataService) GetSafeTradingLimits(ctx context.Context, ticker string) (interface{}, error) {
	// Read the latest liquidity report
	liquidityReportPath := filepath.Join(ds.paths.ReportsDir, "liquidity_report.csv")
	
	ds.logger.Debug("GetSafeTradingLimits: reading liquidity report",
		slog.String("ticker", ticker),
		slog.String("report_path", liquidityReportPath))
	
	file, err := os.Open(liquidityReportPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTickerNotFound
		}
		return nil, fmt.Errorf("failed to open liquidity report: %w", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	
	// Find column indices for safe trading values
	symbolIdx := -1
	safeValue05Idx := -1
	safeValue10Idx := -1
	safeValue20Idx := -1
	optimalTradeIdx := -1
	illiqIdx := -1
	valueIdx := -1
	hybridScoreIdx := -1
	
	for i, col := range header {
		switch col {
		case "Symbol":
			symbolIdx = i
		case "Safe_Trade_0.5%":
			safeValue05Idx = i
		case "Safe_Trade_1%":
			safeValue10Idx = i
		case "Safe_Trade_2%":
			safeValue20Idx = i
		case "Optimal_Trade":
			optimalTradeIdx = i
		case "ILLIQ_Raw":
			illiqIdx = i
		case "Value_Raw":
			valueIdx = i
		case "Hybrid_Score":
			hybridScoreIdx = i
		}
	}
	
	// Read data rows and find the ticker
	var latestMetrics map[string]interface{}
	
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}
		
		// Check if this is our ticker
		if symbolIdx >= 0 && symbolIdx < len(record) && record[symbolIdx] == ticker {
			// Parse safe trading values
			safeValue05, _ := strconv.ParseFloat(record[safeValue05Idx], 64)
			safeValue10, _ := strconv.ParseFloat(record[safeValue10Idx], 64)
			safeValue20, _ := strconv.ParseFloat(record[safeValue20Idx], 64)
			optimalTrade, _ := strconv.ParseFloat(record[optimalTradeIdx], 64)
			illiq, _ := strconv.ParseFloat(record[illiqIdx], 64)
			value, _ := strconv.ParseFloat(record[valueIdx], 64)
			hybridScore, _ := strconv.ParseFloat(record[hybridScoreIdx], 64)
			
			// Update latest metrics (keep the last row for each ticker)
			latestMetrics = map[string]interface{}{
				"ticker": ticker,
				"safe_trading_limits": map[string]interface{}{
					"safe_value_0_5_percent": safeValue05,
					"safe_value_1_percent":   safeValue10,
					"safe_value_2_percent":   safeValue20,
					"optimal_trade_size":     optimalTrade,
				},
				"liquidity_metrics": map[string]interface{}{
					"illiq":        illiq,
					"avg_value":    value,
					"hybrid_score": hybridScore,
				},
				"recommendations": map[string]interface{}{
					"small_trades":  fmt.Sprintf("Trades up to %.0f IQD will have minimal (<0.5%%) price impact", safeValue05),
					"medium_trades": fmt.Sprintf("Trades up to %.0f IQD will have moderate (<1%%) price impact", safeValue10),
					"large_trades":  fmt.Sprintf("Trades up to %.0f IQD will have significant (<2%%) price impact", safeValue20),
					"optimal":       fmt.Sprintf("Optimal trade size: %.0f IQD for best execution", optimalTrade),
				},
			}
		}
	}
	
	if latestMetrics == nil {
		return nil, ErrTickerNotFound
	}
	
	return latestMetrics, nil
}

// EstimateTradeImpact estimates the price impact for a proposed trade
func (ds *DataService) EstimateTradeImpact(ctx context.Context, ticker string, tradeValue float64) (float64, error) {
	// Get safe trading limits for the ticker
	limits, err := ds.GetSafeTradingLimits(ctx, ticker)
	if err != nil {
		return 0, err
	}
	
	// Extract ILLIQ from the limits response
	limitsMap := limits.(map[string]interface{})
	liquidityMetrics := limitsMap["liquidity_metrics"].(map[string]interface{})
	illiq := liquidityMetrics["illiq"].(float64)
	
	// Calculate estimated impact using ILLIQ
	// ILLIQ = |Return| / Volume_millions
	// So estimated impact = ILLIQ * (TradeValue / 1,000,000)
	estimatedImpact := illiq * (tradeValue / 1_000_000)
	
	// Convert to percentage
	impactPercentage := estimatedImpact * 100
	
	ds.logger.Info("EstimateTradeImpact calculated",
		slog.String("ticker", ticker),
		slog.Float64("trade_value", tradeValue),
		slog.Float64("illiq", illiq),
		slog.Float64("impact_percentage", impactPercentage))
	
	return impactPercentage, nil
}

// CreateTradeSchedule creates an execution schedule for a large trade
func (ds *DataService) CreateTradeSchedule(ctx context.Context, ticker string, totalTradeValue float64) (interface{}, error) {
	// Get safe trading limits
	limits, err := ds.GetSafeTradingLimits(ctx, ticker)
	if err != nil {
		return nil, err
	}
	
	// Extract safe trading values
	limitsMap := limits.(map[string]interface{})
	safeLimits := limitsMap["safe_trading_limits"].(map[string]interface{})
	optimalTradeSize := safeLimits["optimal_trade_size"].(float64)
	safeValue10 := safeLimits["safe_value_1_percent"].(float64)
	
	// If trade is small enough, no need to split
	if totalTradeValue <= safeValue10 {
		return map[string]interface{}{
			"ticker":            ticker,
			"total_trade_value": totalTradeValue,
			"strategy":          "single_execution",
			"tranches":          1,
			"schedule": []map[string]interface{}{
				{
					"tranche":           1,
					"value":             totalTradeValue,
					"estimated_impact":  "< 1%",
					"execution_time":    "immediate",
					"recommendation":    "Execute as single trade",
				},
			},
		}, nil
	}
	
	// Calculate number of tranches needed
	numTranches := int(math.Ceil(totalTradeValue / optimalTradeSize))
	trancheSize := totalTradeValue / float64(numTranches)
	
	// Create schedule
	schedule := make([]map[string]interface{}, numTranches)
	for i := 0; i < numTranches; i++ {
		// Calculate value for this tranche
		trancheValue := trancheSize
		if i == numTranches-1 {
			// Last tranche might be slightly different due to rounding
			trancheValue = totalTradeValue - (trancheSize * float64(numTranches-1))
		}
		
		// Estimate impact for this tranche
		impact, _ := ds.EstimateTradeImpact(ctx, ticker, trancheValue)
		
		schedule[i] = map[string]interface{}{
			"tranche":          i + 1,
			"value":            trancheValue,
			"estimated_impact": fmt.Sprintf("%.2f%%", impact),
			"execution_time":   fmt.Sprintf("T+%d minutes", i*15), // 15 minutes between tranches
			"recommendation":   getTrancheRecommendation(i, numTranches),
		}
	}
	
	return map[string]interface{}{
		"ticker":            ticker,
		"total_trade_value": totalTradeValue,
		"strategy":          "split_execution",
		"tranches":          numTranches,
		"tranche_size":      trancheSize,
		"time_interval":     "15 minutes",
		"total_duration":    fmt.Sprintf("%d minutes", (numTranches-1)*15),
		"schedule":          schedule,
		"notes": map[string]interface{}{
			"rationale":     fmt.Sprintf("Trade split into %d tranches to minimize market impact", numTranches),
			"optimal_size":  fmt.Sprintf("Each tranche ~%.0f IQD based on optimal trade size", optimalTradeSize),
			"flexibility":   "Adjust timing based on market conditions and liquidity",
		},
	}, nil
}

// GetHistoricalData retrieves historical trading data for a ticker within a date range
func (ds *DataService) GetHistoricalData(ctx context.Context, ticker string, startDate, endDate time.Time) ([]domain.TradeRecord, error) {
	ds.logger.InfoContext(ctx, "loading historical data",
		"ticker", ticker,
		"start_date", startDate.Format("2006-01-02"),
		"end_date", endDate.Format("2006-01-02"),
	)

	var records []domain.TradeRecord
	
	// Look for ticker-specific CSV file first
	tickerFile := filepath.Join(ds.paths.ReportsDir, fmt.Sprintf("%s_daily.csv", ticker))
	if _, err := os.Stat(tickerFile); err == nil {
		// Load from ticker-specific file
		file, err := os.Open(tickerFile)
		if err != nil {
			return nil, fmt.Errorf("open ticker file: %w", err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		// Skip header
		if _, err := reader.Read(); err != nil {
			return nil, fmt.Errorf("read header: %w", err)
		}

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				ds.logger.WarnContext(ctx, "error reading row", "error", err)
				continue
			}

			// Parse date
			dateStr := row[0]
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}

			// Check date range
			if date.Before(startDate) || date.After(endDate) {
				continue
			}

			// Parse numeric fields
			openPrice, _ := strconv.ParseFloat(row[2], 64)
			highPrice, _ := strconv.ParseFloat(row[3], 64)
			lowPrice, _ := strconv.ParseFloat(row[4], 64)
			closePrice, _ := strconv.ParseFloat(row[5], 64)
			volume, _ := strconv.ParseInt(row[6], 10, 64)
			value, _ := strconv.ParseFloat(row[7], 64)

			record := domain.TradeRecord{
				CompanySymbol: ticker,
				CompanyName:   row[1],
				Date:          date,
				OpenPrice:     openPrice,
				HighPrice:     highPrice,
				LowPrice:      lowPrice,
				ClosePrice:    closePrice,
				Volume:        volume,
				Value:         value,
				TradingStatus: true, // Historical data is actively traded
			}

			records = append(records, record)
		}
	} else {
		// Fallback to daily report files
		// List all CSV files in reports directory
		files, err := os.ReadDir(ds.paths.ReportsDir)
		if err != nil {
			return nil, fmt.Errorf("read reports directory: %w", err)
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".csv") || strings.Contains(file.Name(), "_daily") {
				continue
			}

			// Parse date from filename (assuming format: YYYY-MM-DD.csv or similar)
			datePart := strings.TrimSuffix(file.Name(), ".csv")
			fileDate, err := time.Parse("2006-01-02", datePart)
			if err != nil {
				// Try other date formats
				fileDate, err = time.Parse("20060102", datePart)
				if err != nil {
					continue
				}
			}

			// Check if file date is within range
			if fileDate.Before(startDate) || fileDate.After(endDate) {
				continue
			}

			// Load file and look for ticker
			filePath := filepath.Join(ds.paths.ReportsDir, file.Name())
			fileRecords, err := ds.loadDailyReportFile(ctx, filePath, ticker)
			if err != nil {
				ds.logger.WarnContext(ctx, "failed to load daily report",
					"file", file.Name(),
					"error", err,
				)
				continue
			}

			records = append(records, fileRecords...)
		}
	}

	// Sort by date
	sort.Slice(records, func(i, j int) bool {
		return records[i].Date.Before(records[j].Date)
	})

	ds.logger.InfoContext(ctx, "loaded historical data",
		"ticker", ticker,
		"record_count", len(records),
	)

	return records, nil
}

// loadDailyReportFile loads records for a specific ticker from a daily report file
func (ds *DataService) loadDailyReportFile(ctx context.Context, filePath, ticker string) ([]domain.TradeRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []domain.TradeRecord
	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		// Check if this row is for our ticker
		if len(row) < 8 || row[0] != ticker {
			continue
		}

		// Parse the record
		date, _ := time.Parse("2006-01-02", row[1])
		openPrice, _ := strconv.ParseFloat(row[2], 64)
		highPrice, _ := strconv.ParseFloat(row[3], 64)
		lowPrice, _ := strconv.ParseFloat(row[4], 64)
		closePrice, _ := strconv.ParseFloat(row[5], 64)
		volume, _ := strconv.ParseInt(row[6], 10, 64)
		value, _ := strconv.ParseFloat(row[7], 64)

		record := domain.TradeRecord{
			CompanySymbol: ticker,
			Date:          date,
			OpenPrice:     openPrice,
			HighPrice:     highPrice,
			LowPrice:      lowPrice,
			ClosePrice:    closePrice,
			Volume:        volume,
			Value:         value,
			TradingStatus: true,
		}

		records = append(records, record)
	}

	return records, nil
}

// getTrancheRecommendation provides execution recommendations for each tranche
func getTrancheRecommendation(trancheIndex, totalTranches int) string {
	if trancheIndex == 0 {
		return "Initial tranche - monitor market response"
	} else if trancheIndex < totalTranches/2 {
		return "Early tranche - assess liquidity conditions"
	} else if trancheIndex < totalTranches-1 {
		return "Mid-execution - adjust if needed based on impact"
	} else {
		return "Final tranche - complete remaining volume"
	}
}