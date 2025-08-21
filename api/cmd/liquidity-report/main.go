package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"isxcli/internal/config"
	"isxcli/internal/license"
	"isxcli/internal/liquidity"
)

func main() {
	outputDir := flag.String("out", "", "output directory for liquidity report (defaults to data/reports)")
	windowSize := flag.Int("window", 60, "window size for liquidity calculation (20, 60, or 120 days)")
	flag.Parse()

	// Initialize paths
	paths, err := config.GetPaths()
	if err != nil {
		slog.Error("Failed to initialize paths", "error", err)
		os.Exit(1)
	}

	// License validation
	slog.Info("Validating license...")
	licensePath, err := config.GetLicensePath()
	if err != nil {
		slog.Error("Failed to get license path", "error", err)
		os.Exit(1)
	}
	
	licenseManager, err := license.NewManager(licensePath)
	if err != nil {
		slog.Error("License system initialization failed", "error", err)
		os.Exit(1)
	}
	
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

	// Use default output directory if not specified
	if *outputDir == "" {
		*outputDir = paths.ReportsDir
	}

	// Load trading data from combined CSV
	combinedPath := filepath.Join(*outputDir, "isx_combined_data.csv")
	slog.Info("Loading trading data", "path", combinedPath)
	
	// Check if combined CSV exists
	if _, err := os.Stat(combinedPath); os.IsNotExist(err) {
		slog.Error("Combined CSV file not found", 
			"path", combinedPath,
			"hint", "Run processor first to generate combined data")
		os.Exit(1)
	}
	
	tradingData, err := loadTradingData(combinedPath)
	if err != nil {
		slog.Error("Failed to load trading data", "error", err)
		os.Exit(1)
	}
	
	// Validate loaded data
	if len(tradingData) == 0 {
		slog.Error("No trading data found in CSV", 
			"path", combinedPath,
			"hint", "Check if processor generated valid data")
		os.Exit(1)
	}
	
	slog.Info("Loaded trading data", "records", len(tradingData))

	// Set up liquidity calculation parameters
	window := liquidity.Window(*windowSize)
	
	// Default parameters
	penaltyParams := liquidity.PenaltyParams{
		PiecewiseP0:       1.0,
		PiecewiseBeta:     0.5,
		PiecewiseGamma:    1.5,
		PiecewisePStar:    100.0,
		PiecewiseMaxMult:  10.0,
		ExponentialP0:     1.0,
		ExponentialAlpha:  0.1,
		ExponentialMaxMult: 10.0,
	}
	
	weights := liquidity.ComponentWeights{
		Impact:     0.35,
		Value:      0.35,
		Continuity: 0.20,
		Spread:     0.10,
	}
	
	// Create calculator
	calc := liquidity.NewCalculator(window, penaltyParams, weights, slog.Default())
	
	// Calculate liquidity metrics
	slog.Info("Calculating liquidity metrics...")
	ctx := context.Background()
	metrics, err := calc.Calculate(ctx, tradingData)
	if err != nil {
		slog.Error("Failed to calculate liquidity metrics", "error", err)
		os.Exit(1)
	}
	slog.Info("Calculated liquidity metrics", "metrics", len(metrics))
	
	// Save results with timestamp
	timestamp := time.Now().Format("20060102")
	
	// Create liquidity reports directory
	reportDir := filepath.Join(*outputDir, "liquidity", "reports")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		slog.Error("Failed to create liquidity reports directory", "error", err)
		os.Exit(1)
	}
	
	outputPath := filepath.Join(reportDir, fmt.Sprintf("liquidity_report_%s.csv", timestamp))
	slog.Info("Saving liquidity report", "path", outputPath)
	
	if err := liquidity.SaveToCSV(metrics, outputPath); err != nil {
		slog.Error("Failed to save liquidity report", "error", err)
		os.Exit(1)
	}
	
	// Also save summary report
	summaryDir := filepath.Join(*outputDir, "liquidity", "summaries")
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		slog.Error("Failed to create liquidity summaries directory", "error", err)
		os.Exit(1)
	}
	
	summaryPath := filepath.Join(summaryDir, fmt.Sprintf("liquidity_summary_%s.txt", timestamp))
	if err := liquidity.SaveSummaryReport(metrics, summaryPath); err != nil {
		slog.Error("Failed to save summary report", "error", err)
		os.Exit(1)
	}
	
	slog.Info("Liquidity report generated successfully",
		"report", outputPath,
		"summary", summaryPath,
		"metrics", len(metrics))
	
	// Print summary statistics
	printSummaryStats(metrics)
}

func loadTradingData(csvPath string) ([]liquidity.TradingDay, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("open CSV file: %w", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	
	// Find column indices
	dateIdx := -1
	symbolIdx := -1
	openIdx := -1
	highIdx := -1
	lowIdx := -1
	closeIdx := -1
	volumeIdx := -1
	valueIdx := -1
	numTradesIdx := -1
	statusIdx := -1
	
	for i, col := range header {
		switch col {
		case "Date":
			dateIdx = i
		case "Symbol", "Ticker":  // Support both column names
			symbolIdx = i
		case "OpenPrice", "Open":
			openIdx = i
		case "HighPrice", "High":
			highIdx = i
		case "LowPrice", "Low":
			lowIdx = i
		case "ClosePrice", "Close":
			closeIdx = i
		case "Volume":
			volumeIdx = i
		case "Value":
			valueIdx = i
		case "NumTrades", "NumOfTrades":
			numTradesIdx = i
		case "TradingStatus", "Status":
			statusIdx = i
		}
	}
	
	// Read data
	var tradingData []liquidity.TradingDay
	
	for {
		record, err := reader.Read()
		if err != nil {
			break // EOF or error
		}
		
		// Parse date
		dateStr := record[dateIdx]
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue // Skip invalid dates
		}
		
		// Parse numeric fields
		open, _ := strconv.ParseFloat(record[openIdx], 64)
		high, _ := strconv.ParseFloat(record[highIdx], 64)
		low, _ := strconv.ParseFloat(record[lowIdx], 64)
		close, _ := strconv.ParseFloat(record[closeIdx], 64)
		volume, _ := strconv.ParseFloat(record[volumeIdx], 64)
		value, _ := strconv.ParseFloat(record[valueIdx], 64)
		numTrades, _ := strconv.Atoi(record[numTradesIdx])
		
		// Create trading day
		td := liquidity.TradingDay{
			Date:          date,
			Symbol:        record[symbolIdx],
			Open:          open,
			High:          high,
			Low:           low,
			Close:         close,
			Volume:        volume,
			ShareVolume:   volume, // Same as Volume
			Value:         value,  // Value in IQD
			NumTrades:     numTrades,
			TradingStatus: record[statusIdx],
		}
		
		tradingData = append(tradingData, td)
	}
	
	return tradingData, nil
}

func printSummaryStats(metrics []liquidity.TickerMetrics) {
	if len(metrics) == 0 {
		return
	}
	
	// Find top 10 most liquid stocks by hybrid score
	topLiquid := make([]liquidity.TickerMetrics, 0, 10)
	for _, m := range metrics {
		if len(topLiquid) < 10 {
			topLiquid = append(topLiquid, m)
		} else {
			// Find minimum and replace if current is higher
			minIdx := 0
			minScore := topLiquid[0].HybridScore
			for i, tm := range topLiquid {
				if tm.HybridScore < minScore {
					minScore = tm.HybridScore
					minIdx = i
				}
			}
			if m.HybridScore > minScore {
				topLiquid[minIdx] = m
			}
		}
	}
	
	fmt.Println("\n=== TOP 10 MOST LIQUID STOCKS (WITH SAFE TRADING VALUES) ===")
	fmt.Println("Symbol | Hybrid Score | ILLIQ | Safe@0.5% | Safe@1% | Safe@2% | Optimal Trade")
	fmt.Println("-------|--------------|-------|-----------|---------|---------|---------------")
	
	for _, m := range topLiquid {
		fmt.Printf("%-6s | %11.2f | %5.2f | %9.0f | %7.0f | %7.0f | %13.0f\n",
			m.Symbol, m.HybridScore, m.ILLIQ,
			m.SafeValue_0_5, m.SafeValue_1_0, m.SafeValue_2_0,
			m.OptimalTradeSize)
	}
	
	fmt.Println("\n=== SAFE TRADING INTERPRETATION ===")
	fmt.Println("Safe@0.5%: Maximum trade value (IQD) for minimal (<0.5%) price impact")
	fmt.Println("Safe@1%:   Maximum trade value (IQD) for moderate (<1%) price impact")
	fmt.Println("Safe@2%:   Maximum trade value (IQD) for significant (<2%) price impact")
	fmt.Println("Optimal:   Recommended trade size balancing impact and efficiency")
}