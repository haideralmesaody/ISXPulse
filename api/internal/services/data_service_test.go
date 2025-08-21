package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"isxcli/internal/config"
)

// MockFileSystem allows us to control what files exist for testing
type MockFileSystem struct {
	files map[string][]byte
}

func (m *MockFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	var entries []os.DirEntry
	for path := range m.files {
		if filepath.Dir(path) == name {
			entries = append(entries, &mockDirEntry{
				name: filepath.Base(path),
				dir:  false,
			})
		}
	}
	if len(entries) == 0 {
		return nil, os.ErrNotExist
	}
	return entries, nil
}

type mockDirEntry struct {
	name string
	dir  bool
}

func (m *mockDirEntry) Name() string       { return m.name }
func (m *mockDirEntry) IsDir() bool        { return m.dir }
func (m *mockDirEntry) Type() os.FileMode  { return 0 }
func (m *mockDirEntry) Info() (os.FileInfo, error) {
	return &mockFileInfo{
		name:    m.name,
		size:    100,
		modTime: time.Now(),
	}, nil
}

type mockFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// TestNewDataService tests service creation
func TestNewDataService(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	
	// Set up config paths
	os.Setenv("ISX_DATA_DIR", tempDir)
	defer os.Unsetenv("ISX_DATA_DIR")
	
	cfg := &config.Config{}
	
	t.Run("Create with default logger", func(t *testing.T) {
		service, err := NewDataService(cfg)
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.NotNil(t, service.config)
		assert.NotNil(t, service.paths)
		assert.NotNil(t, service.logger)
	})
	
	t.Run("Create with custom logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		service, err := NewDataServiceWithLogger(cfg, logger)
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, logger, service.logger)
	})
	
	t.Run("Create with nil logger uses default", func(t *testing.T) {
		service, err := NewDataServiceWithLogger(cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.NotNil(t, service.logger)
	})
}

// TestGetReports tests getting reports list
func TestGetReports(t *testing.T) {
	tempDir := t.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	
	// Create test files
	file1 := filepath.Join(reportsDir, "report1.csv")
	file2 := filepath.Join(reportsDir, "report2.csv")
	file3 := filepath.Join(reportsDir, "notacsv.txt")
	
	require.NoError(t, os.WriteFile(file1, []byte("data1"), 0644))
	time.Sleep(10 * time.Millisecond) // Ensure different mod times
	require.NoError(t, os.WriteFile(file2, []byte("data2"), 0644))
	require.NoError(t, os.WriteFile(file3, []byte("data3"), 0644))
	
	// Create service with test paths
	paths := &config.Paths{
		DataDir:    tempDir,
		ReportsDir: reportsDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	t.Run("Success", func(t *testing.T) {
		reports, err := service.GetReports(ctx)
		require.NoError(t, err)
		assert.Len(t, reports, 2) // Only CSV files
		
		// Check ordering (newest first)
		assert.Equal(t, "report2.csv", reports[0]["name"])
		assert.Equal(t, "report1.csv", reports[1]["name"])
		
		// Check fields
		for _, report := range reports {
			assert.Contains(t, report, "name")
			assert.Contains(t, report, "size")
			assert.Contains(t, report, "modified")
		}
	})
	
	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		require.NoError(t, os.MkdirAll(emptyDir, 0755))
		
		service.paths.ReportsDir = emptyDir
		reports, err := service.GetReports(ctx)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})
	
	t.Run("Non-existent directory", func(t *testing.T) {
		service.paths.ReportsDir = filepath.Join(tempDir, "nonexistent")
		reports, err := service.GetReports(ctx)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})
}

// TestGetTickers tests getting ticker information
func TestGetTickers(t *testing.T) {
	tempDir := t.TempDir()
	tickerFile := filepath.Join(tempDir, "ticker_summary.json")
	
	// Create test ticker data
	tickerData := map[string]interface{}{
		"tickers": []map[string]interface{}{
			{"symbol": "AAPL", "price": 150.0},
			{"symbol": "GOOGL", "price": 2800.0},
		},
	}
	
	data, err := json.Marshal(tickerData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tickerFile, data, 0644))
	
	paths := &config.Paths{
		DataDir: tempDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	// Override the path directly since methods can't be reassigned
	paths.TickerSummaryJSON = tickerFile
	
	ctx := context.Background()
	
	t.Run("Success", func(t *testing.T) {
		result, err := service.GetTickers(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// Verify structure
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, resultMap, "tickers")
	})
	
	t.Run("File not found", func(t *testing.T) {
		paths.TickerSummaryJSON = filepath.Join(tempDir, "nonexistent.json")
		
		result, err := service.GetTickers(ctx)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{}, result)
	})
	
	t.Run("Invalid JSON", func(t *testing.T) {
		invalidFile := filepath.Join(tempDir, "invalid.json")
		require.NoError(t, os.WriteFile(invalidFile, []byte("invalid json"), 0644))
		
		paths.TickerSummaryJSON = invalidFile
		
		result, err := service.GetTickers(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse ticker summary")
		assert.Nil(t, result)
	})
}

// TestGetIndices tests getting market indices
func TestGetIndices(t *testing.T) {
	tempDir := t.TempDir()
	indicesFile := filepath.Join(tempDir, "indices.csv")
	
	// Create test CSV data
	csvData := [][]string{
		{"Date", "ISX60", "ISX15"},
		{"2024-01-01", "100.5", "50.25"},
		{"2024-01-02", "101.0", "51.0"},
		{"2024-01-03", "99.5", ""}, // Missing ISX15
	}
	
	file, err := os.Create(indicesFile)
	require.NoError(t, err)
	writer := csv.NewWriter(file)
	for _, row := range csvData {
		require.NoError(t, writer.Write(row))
	}
	writer.Flush()
	file.Close()
	
	paths := &config.Paths{
		DataDir: tempDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	// Override the path directly
	paths.IndexCSV = indicesFile
	
	ctx := context.Background()
	
	t.Run("Success", func(t *testing.T) {
		result, err := service.GetIndices(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// Verify structure
		dates, ok := result["dates"].([]string)
		require.True(t, ok)
		assert.Len(t, dates, 3)
		
		isx60, ok := result["isx60"].([]float64)
		require.True(t, ok)
		assert.Len(t, isx60, 3)
		assert.Equal(t, 100.5, isx60[0])
		
		isx15, ok := result["isx15"].([]float64)
		require.True(t, ok)
		assert.Len(t, isx15, 3)
		assert.Equal(t, 50.25, isx15[0])
		assert.Equal(t, 0.0, isx15[2]) // Missing value defaults to 0
	})
	
	t.Run("File not found", func(t *testing.T) {
		paths.IndexCSV = filepath.Join(tempDir, "nonexistent.csv")
		
		result, err := service.GetIndices(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{}, result["dates"])
		assert.Equal(t, []float64{}, result["isx60"])
		assert.Equal(t, []float64{}, result["isx15"])
	})
	
	t.Run("Invalid header", func(t *testing.T) {
		invalidFile := filepath.Join(tempDir, "invalid_header.csv")
		f, err := os.Create(invalidFile)
		require.NoError(t, err)
		writer := csv.NewWriter(f)
		writer.Write([]string{"Wrong", "Header"})
		writer.Flush()
		f.Close()
		
		paths.IndexCSV = invalidFile
		
		_, err = service.GetIndices(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid CSV header format")
	})
	
	t.Run("Invalid data row", func(t *testing.T) {
		invalidDataFile := filepath.Join(tempDir, "invalid_data.csv")
		f, err := os.Create(invalidDataFile)
		require.NoError(t, err)
		writer := csv.NewWriter(f)
		writer.Write([]string{"Date", "ISX60", "ISX15"})
		writer.Write([]string{"2024-01-01"}) // Too few columns
		writer.Flush()
		f.Close()
		
		paths.IndexCSV = invalidDataFile
		
		// The service returns an error for CSV rows with wrong number of fields
		_, err = service.GetIndices(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wrong number of fields")
	})
}

// TestGetFiles tests getting file listings
func TestGetFiles(t *testing.T) {
	tempDir := t.TempDir()
	downloadsDir := filepath.Join(tempDir, "downloads")
	reportsDir := filepath.Join(tempDir, "reports")
	
	require.NoError(t, os.MkdirAll(downloadsDir, 0755))
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	
	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(downloadsDir, "file1.xlsx"), []byte("data1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(downloadsDir, "file2.xlsx"), []byte("data2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "report1.csv"), []byte("data3"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "ticker_trading_history.csv"), []byte("data4"), 0644))
	
	paths := &config.Paths{
		DataDir:      tempDir,
		DownloadsDir: downloadsDir,
		ReportsDir:   reportsDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	t.Run("Success", func(t *testing.T) {
		result, err := service.GetFiles(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// Check structure
		assert.Contains(t, result, "downloads")
		assert.Contains(t, result, "reports")
		assert.Contains(t, result, "csvFiles")
		assert.Contains(t, result, "total_size")
		assert.Contains(t, result, "last_modified")
		
		// Verify file counts
		downloads, ok := result["downloads"].([]interface{})
		require.True(t, ok)
		assert.Len(t, downloads, 2)
		
		reports, ok := result["reports"].([]interface{})
		require.True(t, ok)
		assert.Len(t, reports, 1)
		
		csvFiles, ok := result["csvFiles"].([]interface{})
		require.True(t, ok)
		assert.Len(t, csvFiles, 1)
	})
}

// TestGetMarketMovers tests getting market movers
func TestGetMarketMovers(t *testing.T) {
	service := &DataService{
		config: &config.Config{},
		paths:  &config.Paths{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	t.Run("Default values", func(t *testing.T) {
		result, err := service.GetMarketMovers(ctx, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "1d", result["period"])
		assert.Contains(t, result, "gainers")
		assert.Contains(t, result, "losers")
		assert.Contains(t, result, "mostActive")
		assert.Contains(t, result, "updated")
	})
	
	t.Run("Custom values", func(t *testing.T) {
		result, err := service.GetMarketMovers(ctx, "1w", "20", "1000000")
		require.NoError(t, err)
		assert.Equal(t, "1w", result["period"])
	})
}

// TestGetTickerChart tests getting ticker chart data
func TestGetTickerChart(t *testing.T) {
	tempDir := t.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	
	// Create test file in the reports directory where GetTickerDailyCSVPath expects it
	tickerFile := filepath.Join(reportsDir, "AAPL_daily.csv")
	require.NoError(t, os.WriteFile(tickerFile, []byte("data"), 0644))
	
	paths := &config.Paths{
		DataDir:    tempDir,
		ReportsDir: reportsDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	t.Run("Success", func(t *testing.T) {
		result, err := service.GetTickerChart(ctx, "AAPL")
		require.NoError(t, err)
		assert.Equal(t, "AAPL", result["ticker"])
		assert.Contains(t, result, "data")
	})
	
	t.Run("Empty ticker", func(t *testing.T) {
		result, err := service.GetTickerChart(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ticker parameter required")
		assert.Nil(t, result)
	})
	
	t.Run("File not found", func(t *testing.T) {
		result, err := service.GetTickerChart(ctx, "NONEXISTENT")
		require.NoError(t, err)
		assert.Equal(t, "NONEXISTENT", result["ticker"])
		data, ok := result["data"].([]interface{})
		require.True(t, ok)
		assert.Empty(t, data)
	})
}

// TestGetDailyReport tests getting daily report data
func TestGetDailyReport(t *testing.T) {
	tempDir := t.TempDir()
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	
	// Create files in the reports directory where GetDailyCSVPath expects them
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	dailyFile := filepath.Join(reportsDir, "isx_daily_20240101.csv")
	
	// Create test CSV data
	csvData := [][]string{
		{"ticker", "price", "volume"},
		{"AAPL", "150.00", "1000000"},
		{"GOOGL", "2800.00", "500000"},
	}
	
	file, err := os.Create(dailyFile)
	require.NoError(t, err)
	writer := csv.NewWriter(file)
	for _, row := range csvData {
		require.NoError(t, writer.Write(row))
	}
	writer.Flush()
	file.Close()
	
	paths := &config.Paths{
		DataDir:    tempDir,
		ReportsDir: reportsDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	t.Run("Success", func(t *testing.T) {
		// Verify file exists
		expectedPath := paths.GetDailyCSVPath(date)
		t.Logf("Expected path: %s", expectedPath)
		t.Logf("Created file: %s", dailyFile)
		
		_, err := os.Stat(expectedPath)
		require.NoError(t, err, "Daily CSV file should exist at expected path")
		
		result, err := service.GetDailyReport(ctx, date)
		require.NoError(t, err)
		require.Len(t, result, 2)
		
		// Check first row
		assert.Equal(t, "AAPL", result[0]["ticker"])
		assert.Equal(t, "150.00", result[0]["price"])
		assert.Equal(t, "1000000", result[0]["volume"])
	})
	
	t.Run("File not found", func(t *testing.T) {
		futureDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		result, err := service.GetDailyReport(ctx, futureDate)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
	
	t.Run("Empty CSV", func(t *testing.T) {
		emptyFile := filepath.Join(reportsDir, "isx_daily_20240102.csv")
		f, err := os.Create(emptyFile)
		require.NoError(t, err)
		writer := csv.NewWriter(f)
		writer.Write([]string{"header1", "header2"})
		writer.Flush()
		f.Close()
		
		emptyDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		result, err := service.GetDailyReport(ctx, emptyDate)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

// TestDownloadFile tests file download functionality
func TestDownloadFile(t *testing.T) {
	tempDir := t.TempDir()
	downloadsDir := filepath.Join(tempDir, "downloads")
	reportsDir := filepath.Join(tempDir, "reports")
	
	require.NoError(t, os.MkdirAll(downloadsDir, 0755))
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	
	// Create test files
	testContent := "test file content"
	require.NoError(t, os.WriteFile(filepath.Join(downloadsDir, "test.xlsx"), []byte(testContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "test.csv"), []byte(testContent), 0644))
	
	paths := &config.Paths{
		DataDir:      tempDir,
		DownloadsDir: downloadsDir,
		ReportsDir:   reportsDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	t.Run("Download from downloads", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download?type=downloads&file=test.xlsx", nil)
		w := httptest.NewRecorder()
		
		err := service.DownloadFile(ctx, w, req, "downloads", "test.xlsx")
		require.NoError(t, err)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, testContent, w.Body.String())
		assert.Contains(t, w.Header().Get("Content-Disposition"), "test.xlsx")
	})
	
	t.Run("Download from reports", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download?type=reports&file=test.csv", nil)
		w := httptest.NewRecorder()
		
		err := service.DownloadFile(ctx, w, req, "reports", "test.csv")
		require.NoError(t, err)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, testContent, w.Body.String())
	})
	
	t.Run("Invalid file type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download", nil)
		w := httptest.NewRecorder()
		
		err := service.DownloadFile(ctx, w, req, "invalid", "test.csv")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file type")
	})
	
	t.Run("File not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download", nil)
		w := httptest.NewRecorder()
		
		err := service.DownloadFile(ctx, w, req, "downloads", "nonexistent.xlsx")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})
	
	t.Run("Path traversal attempt", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download", nil)
		w := httptest.NewRecorder()
		
		err := service.DownloadFile(ctx, w, req, "downloads", "../../../etc/passwd")
		assert.Error(t, err)
		// The error could be either "invalid file path" or "file not found" depending on OS
		assert.True(t, strings.Contains(err.Error(), "invalid file path") || 
			strings.Contains(err.Error(), "file not found"))
	})
}

// TestListFiles tests the listFiles helper function
func TestListFiles(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	require.NoError(t, os.MkdirAll(testDir, 0755))
	
	// Create test files with different extensions
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "file1.csv"), []byte("data1"), 0644))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "file2.csv"), []byte("data2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "file3.txt"), []byte("data3"), 0644))
	
	// Create subdirectory (should be ignored)
	require.NoError(t, os.MkdirAll(filepath.Join(testDir, "subdir"), 0755))
	
	paths := &config.Paths{
		DataDir:      tempDir,
		DownloadsDir: filepath.Join(tempDir, "downloads"),
		ReportsDir:   filepath.Join(tempDir, "reports"),
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	t.Run("List CSV files", func(t *testing.T) {
		result := make(map[string]interface{})
		err := service.listFiles("test", ".csv", result)
		require.NoError(t, err)
		
		assert.NotNil(t, result["total_size"])
		assert.NotNil(t, result["last_modified"])
	})
	
	t.Run("Non-existent directory", func(t *testing.T) {
		result := make(map[string]interface{})
		err := service.listFiles("nonexistent", ".csv", result)
		require.NoError(t, err) // Should not error for non-existent dirs
	})
	
	t.Run("Reports directory with mixed files", func(t *testing.T) {
		reportsDir := filepath.Join(tempDir, "reports")
		require.NoError(t, os.MkdirAll(reportsDir, 0755))
		
		// Create different types of CSV files
		require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "report.csv"), []byte("data"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "ticker_trading_history.csv"), []byte("data"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "isx_daily_2024.csv"), []byte("data"), 0644))
		
		service.paths.ReportsDir = reportsDir
		
		result := make(map[string]interface{})
		err := service.listFiles("reports", ".csv", result)
		require.NoError(t, err)
		
		// Check separation of files
		reports, ok := result["reports"].([]interface{})
		require.True(t, ok)
		assert.Len(t, reports, 1) // Only report.csv
		
		csvFiles, ok := result["csvFiles"].([]interface{})
		require.True(t, ok)
		assert.Len(t, csvFiles, 2) // ticker_trading_history.csv and isx_daily_2024.csv
	})
}

// BenchmarkGetReports benchmarks the GetReports function
func BenchmarkGetReports(b *testing.B) {
	tempDir := b.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(b, os.MkdirAll(reportsDir, 0755))
	
	// Create many test files
	for i := 0; i < 100; i++ {
		filename := filepath.Join(reportsDir, fmt.Sprintf("report%d.csv", i))
		require.NoError(b, os.WriteFile(filename, []byte("data"), 0644))
	}
	
	paths := &config.Paths{
		DataDir:    tempDir,
		ReportsDir: reportsDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetReports(ctx)
	}
}

// BenchmarkGetIndices benchmarks CSV parsing
func BenchmarkGetIndices(b *testing.B) {
	tempDir := b.TempDir()
	indicesFile := filepath.Join(tempDir, "indices.csv")
	
	// Create large CSV file
	file, err := os.Create(indicesFile)
	require.NoError(b, err)
	writer := csv.NewWriter(file)
	
	writer.Write([]string{"Date", "ISX60", "ISX15"})
	for i := 0; i < 1000; i++ {
		writer.Write([]string{
			fmt.Sprintf("2024-01-%02d", i%30+1),
			fmt.Sprintf("%.2f", 100.0+float64(i)*0.1),
			fmt.Sprintf("%.2f", 50.0+float64(i)*0.05),
		})
	}
	writer.Flush()
	file.Close()
	
	paths := &config.Paths{
		DataDir:  tempDir,
		IndexCSV: indicesFile,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetIndices(ctx)
	}
}

// TestConcurrentAccess tests concurrent access to the service
func TestConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	
	// Create test file
	require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "report.csv"), []byte("data"), 0644))
	
	paths := &config.Paths{
		DataDir:    tempDir,
		ReportsDir: reportsDir,
	}
	
	service := &DataService{
		config: &config.Config{},
		paths:  paths,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	
	ctx := context.Background()
	
	// Run multiple goroutines accessing the service
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = service.GetReports(ctx)
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}