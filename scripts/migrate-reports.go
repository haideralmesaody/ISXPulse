package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run migrate-reports.go <reports-directory>")
		fmt.Println("Example: go run migrate-reports.go dist/data/reports")
		os.Exit(1)
	}

	reportsDir := os.Args[1]
	
	// Verify directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		log.Fatalf("Reports directory does not exist: %s", reportsDir)
	}

	fmt.Printf("Starting migration of reports in: %s\n", reportsDir)
	fmt.Println("=" + strings.Repeat("=", 60))

	// Create new directory structure
	if err := createDirectoryStructure(reportsDir); err != nil {
		log.Fatalf("Failed to create directory structure: %v", err)
	}

	// Migrate existing files
	if err := migrateReports(reportsDir); err != nil {
		log.Fatalf("Failed to migrate reports: %v", err)
	}

	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("Migration completed successfully!")
}

func createDirectoryStructure(reportsDir string) error {
	directories := []string{
		"daily",
		"ticker/banks",
		"ticker/telecom",
		"ticker/industry",
		"ticker/other",
		"liquidity/reports",
		"liquidity/summaries",
		"summary/ticker",
		"summary/market",
		"combined",
		"indexes",
	}

	for _, dir := range directories {
		fullPath := filepath.Join(reportsDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
		fmt.Printf("✓ Created directory: %s\n", dir)
	}

	return nil
}

func migrateReports(reportsDir string) error {
	files, err := os.ReadDir(reportsDir)
	if err != nil {
		return fmt.Errorf("failed to read reports directory: %w", err)
	}

	movedCount := 0
	skippedCount := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		oldPath := filepath.Join(reportsDir, file.Name())
		newPath := getNewPath(reportsDir, file.Name())

		if newPath == "" {
			skippedCount++
			fmt.Printf("⊘ Skipped: %s (unknown type)\n", file.Name())
			continue
		}

		// Don't move if already in correct location
		if oldPath == newPath {
			skippedCount++
			continue
		}

		// Create parent directory if needed
		newDir := filepath.Dir(newPath)
		if err := os.MkdirAll(newDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", newDir, err)
		}

		// Move the file
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s to %s: %w", file.Name(), newPath, err)
		}

		movedCount++
		relPath, _ := filepath.Rel(reportsDir, newPath)
		fmt.Printf("→ Moved: %s → %s\n", file.Name(), relPath)
	}

	fmt.Printf("\nSummary: %d files moved, %d files skipped\n", movedCount, skippedCount)
	return nil
}

func getNewPath(reportsDir, filename string) string {
	// Daily reports: isx_daily_YYYY_MM_DD.csv
	if strings.HasPrefix(filename, "isx_daily_") && strings.HasSuffix(filename, ".csv") {
		parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(filename, "isx_daily_"), ".csv"), "_")
		if len(parts) >= 3 {
			year := parts[0]
			month := parts[1]
			return filepath.Join(reportsDir, "daily", year, month, filename)
		}
	}

	// Ticker history: {TICKER}_trading_history.csv
	if strings.HasSuffix(filename, "_trading_history.csv") {
		ticker := strings.TrimSuffix(filename, "_trading_history.csv")
		sector := getTickerSector(ticker)
		return filepath.Join(reportsDir, "ticker", sector, filename)
	}

	// Liquidity reports
	if strings.HasPrefix(filename, "liquidity_report") {
		if strings.HasSuffix(filename, ".csv") {
			return filepath.Join(reportsDir, "liquidity/reports", filename)
		}
	}
	if strings.HasPrefix(filename, "liquidity_summary") {
		if strings.HasSuffix(filename, ".txt") {
			return filepath.Join(reportsDir, "liquidity/summaries", filename)
		}
	}

	// Summary files
	if filename == "ticker_summary.csv" {
		return filepath.Join(reportsDir, "summary/ticker", filename)
	}
	if filename == "ticker_summary.json" {
		return filepath.Join(reportsDir, "summary/ticker", filename)
	}

	// Index file
	if filename == "indexes.csv" {
		return filepath.Join(reportsDir, "indexes", filename)
	}

	// Combined data
	if filename == "isx_combined_data.csv" {
		return filepath.Join(reportsDir, "combined", filename)
	}

	return ""
}

func getTickerSector(ticker string) string {
	ticker = strings.ToUpper(ticker)

	// Banking tickers
	banks := []string{
		"BBOB", "BMNS", "BNOI", "BCOI", "BIME", "BIIB", "BKUI", "BROI",
		"BASH", "BEFI", "BGUC", "BIBI", "BIDB", "BINT", "BLAD", "BMFI",
		"BMUI", "BNAI", "BSUC", "BTIB", "BTRI", "BTRU", "BUND", "BUOI",
		"BAIB", "BELF", "BCIH",
	}

	// Telecom tickers
	telecom := []string{"TASC", "TZNI"}

	// Industry tickers
	industry := []string{
		"IBSD", "IFCM", "IITC", "IMAP", "IMCM", "IMIB", "INCP",
		"IRMC", "ITLI", "IELI", "IHFI", "IHLI", "IICM", "IIDP",
		"IIEW", "IKHC", "IKLV", "IMCI", "IMOS", "IBPM",
	}

	// Check categories
	for _, bank := range banks {
		if ticker == bank {
			return "banks"
		}
	}

	for _, tel := range telecom {
		if ticker == tel {
			return "telecom"
		}
	}

	for _, ind := range industry {
		if ticker == ind {
			return "industry"
		}
	}

	return "other"
}