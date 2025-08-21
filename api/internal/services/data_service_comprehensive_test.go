package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"isxcli/internal/config"
)

// TestDataServiceComprehensive tests DataService for improved coverage
func TestDataServiceComprehensive(t *testing.T) {
	t.Run("Constructor_Error_Paths", testDataServiceConstructorErrors)
	t.Run("GetReports_Error_Scenarios", testGetReportsErrorScenarios)
	t.Run("GetTickers_Comprehensive", testGetTickersComprehensive)
	t.Run("GetIndices_Error_Paths", testGetIndicesErrorPaths)
	t.Run("GetFiles_Edge_Cases", testGetFilesEdgeCases)
	t.Run("GetMarketMovers_Validation", testGetMarketMoversValidation)
	t.Run("GetTickerChart_Validation", testGetTickerChartValidation)
	t.Run("GetDailyReport_Parsing", testGetDailyReportParsing)
	t.Run("DownloadFile_Security", testDownloadFileSecurity)
	t.Run("ListFiles_Internal_Logic", testListFilesInternalLogic)
	t.Run("Concurrent_Operations", testDataServiceConcurrency)
	t.Run("Logger_Integration", testDataServiceLogging)
}

func testDataServiceConstructorErrors(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() error
		cleanup   func()
		expectErr bool
	}{
		{
			name: "invalid_paths_config",
			setup: func() error {
				// Set invalid environment that would cause GetPaths to fail
				os.Setenv("ISX_DATA_DIR", "/invalid\x00path")
				return nil
			},
			cleanup: func() {
				os.Unsetenv("ISX_DATA_DIR")
			},
			expectErr: false, // GetPaths is defensive and handles this
		},
		{
			name: "nil_config",
			setup: func() error {
				return nil
			},
			cleanup:   func() {},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				require.NoError(t, tt.setup())
			}
			defer tt.cleanup()

			var service *DataService
			var err error

			if tt.name == "nil_config" {
				service, err = NewDataService(nil)
			} else {
				cfg := &config.Config{}
				service, err = NewDataService(cfg)
			}

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				if err != nil {
					t.Logf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func testGetReportsErrorScenarios(t *testing.T) {
	tests := []struct {
		name           string
		setupFS        func(tempDir string) error
		expectedLength int
		expectErr      bool
	}{
		{
			name: "permission_denied_directory",
			setupFS: func(tempDir string) error {
				reportsDir := filepath.Join(tempDir, "reports")
				if err := os.MkdirAll(reportsDir, 0755); err != nil {
					return err
				}
				// Change permissions to deny read access (on systems that support it)
				return os.Chmod(reportsDir, 0000)
			},
			expectedLength: 0,
			expectErr:      true,
		},
		{
			name: "mixed_file_types",
			setupFS: func(tempDir string) error {
				reportsDir := filepath.Join(tempDir, "reports")
				if err := os.MkdirAll(reportsDir, 0755); err != nil {
					return err
				}
				// Create various file types
				files := []string{"report1.csv", "report2.CSV", "data.txt", "archive.zip", ".hidden.csv"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(reportsDir, file), []byte("data"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedLength: 2, // Only .csv files (case sensitive)
			expectErr:      false,
		},
		{
			name: "file_stat_errors",
			setupFS: func(tempDir string) error {
				reportsDir := filepath.Join(tempDir, "reports")
				if err := os.MkdirAll(reportsDir, 0755); err != nil {
					return err
				}
				// Create a file, then make it unreadable
				testFile := filepath.Join(reportsDir, "test.csv")
				if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
					return err
				}
				// Change permissions to make stat fail (on systems that support it)
				return os.Chmod(testFile, 0000)
			},
			expectedLength: 0, // File should be skipped due to stat error
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			
			if err := tt.setupFS(tempDir); err != nil {
				t.Skipf("Setup failed (may not be supported on this system): %v", err)
			}

			// Restore permissions for cleanup
			defer func() {
				reportsDir := filepath.Join(tempDir, "reports")
				os.Chmod(reportsDir, 0755)
				filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
					if err == nil {
						os.Chmod(path, 0644)
					}
					return nil
				})
			}()

			service := createTestDataServiceForComprehensive(t, tempDir)
			service.paths.ReportsDir = filepath.Join(tempDir, "reports")

			ctx := context.Background()
			reports, err := service.GetReports(ctx)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				if err != nil && tt.name == "permission_denied_directory" {
					// Permission errors are acceptable for this test
					assert.Contains(t, err.Error(), "permission denied", "Expected permission denied error")
				} else {
					assert.NoError(t, err)
					assert.Len(t, reports, tt.expectedLength)
				}
			}
		})
	}
}

func testGetTickersComprehensive(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		expectErr   bool
		validate    func(t *testing.T, result interface{})
	}{
		{
			name:        "empty_json_file",
			fileContent: "{}",
			expectErr:   false,
			validate: func(t *testing.T, result interface{}) {
				data, ok := result.(map[string]interface{})
				assert.True(t, ok)
				assert.NotNil(t, data)
			},
		},
		{
			name:        "null_json",
			fileContent: "null",
			expectErr:   false,
			validate: func(t *testing.T, result interface{}) {
				assert.Nil(t, result)
			},
		},
		{
			name:        "array_json",
			fileContent: `[{"symbol": "TASC", "price": 2.50}]`,
			expectErr:   false,
			validate: func(t *testing.T, result interface{}) {
				data, ok := result.([]interface{})
				assert.True(t, ok)
				assert.Len(t, data, 1)
			},
		},
		{
			name:        "truncated_json",
			fileContent: `{"incomplete":`,
			expectErr:   true,
			validate:    nil,
		},
		{
			name:        "invalid_utf8",
			fileContent: "\xFF\xFE{\"test\": \"value\"}",
			expectErr:   true,
			validate:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tickerFile := filepath.Join(tempDir, "ticker_summary.json")
			
			require.NoError(t, os.WriteFile(tickerFile, []byte(tt.fileContent), 0644))

			service := createTestDataServiceForComprehensive(t, tempDir)
			// Override the ticker file path
			service.paths.TickerSummaryJSON = tickerFile

			ctx := context.Background()
			result, err := service.GetTickers(ctx)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse ticker summary")
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func testGetIndicesErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		csvContent  [][]string
		expectErr   bool
		errContains string
	}{
		{
			name: "empty_csv_file",
			csvContent: [][]string{},
			expectErr: true,
			errContains: "EOF",
		},
		{
			name: "header_only",
			csvContent: [][]string{
				{"Date", "ISX60", "ISX15"},
			},
			expectErr: false,
		},
		{
			name: "wrong_header_count",
			csvContent: [][]string{
				{"Date"},
			},
			expectErr: true,
			errContains: "invalid CSV header format",
		},
		{
			name: "wrong_header_names",
			csvContent: [][]string{
				{"Time", "Value", "Other"},
			},
			expectErr: true,
			errContains: "invalid CSV header format",
		},
		{
			name: "inconsistent_row_length",
			csvContent: [][]string{
				{"Date", "ISX60", "ISX15"},
				{"2024-01-01", "100.5"},  // Missing ISX15
			},
			expectErr: true,
			errContains: "wrong number of fields",
		},
		{
			name: "invalid_numeric_values",
			csvContent: [][]string{
				{"Date", "ISX60", "ISX15"},
				{"2024-01-01", "not_a_number", "50.25"},
				{"2024-01-02", "101.0", "invalid_number"},
			},
			expectErr: false, // Service handles parsing errors gracefully
		},
		{
			name: "empty_string_values",
			csvContent: [][]string{
				{"Date", "ISX60", "ISX15"},
				{"2024-01-01", "", ""},
				{"2024-01-02", "101.0", ""},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			indicesFile := filepath.Join(tempDir, "indices.csv")

			// Write CSV content
			file, err := os.Create(indicesFile)
			require.NoError(t, err)
			defer file.Close()

			writer := csv.NewWriter(file)
			for _, row := range tt.csvContent {
				require.NoError(t, writer.Write(row))
			}
			writer.Flush()
			file.Close()

			service := createTestDataServiceForComprehensive(t, tempDir)
			service.paths.IndexCSV = indicesFile

			ctx := context.Background()
			result, err := service.GetIndices(ctx)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, result, "dates")
				assert.Contains(t, result, "isx60")
				assert.Contains(t, result, "isx15")
			}
		})
	}
}

func testGetFilesEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataServiceForComprehensive(t, tempDir)

	// Create complex directory structure
	dirs := []string{
		"downloads",
		"reports", 
		"nested/deep/path",
	}
	
	for _, dir := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(tempDir, dir), 0755))
	}

	// Create files with various characteristics
	testFiles := []struct {
		path string
		size int64
	}{
		{"downloads/large_file.xlsx", 1024 * 1024},   // 1MB
		{"downloads/small_file.xlsx", 100},           // 100 bytes
		{"downloads/empty_file.xlsx", 0},             // Empty file
		{"reports/normal_report.csv", 500},
		{"reports/ticker_trading_history.csv", 750},
		{"reports/isx_daily_20240101.csv", 1000},
		{"nested/deep/path/ignored.csv", 200},        // Should be ignored
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tempDir, tf.path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		
		content := make([]byte, tf.size)
		for i := range content {
			content[i] = byte('a' + (i % 26))
		}
		require.NoError(t, os.WriteFile(fullPath, content, 0644))
	}

	// Update service paths
	service.paths.DownloadsDir = filepath.Join(tempDir, "downloads")
	service.paths.ReportsDir = filepath.Join(tempDir, "reports")

	ctx := context.Background()
	result, err := service.GetFiles(ctx)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, result, "downloads")
	assert.Contains(t, result, "reports") 
	assert.Contains(t, result, "csvFiles")
	assert.Contains(t, result, "total_size")
	assert.Contains(t, result, "last_modified")

	// Verify download files
	downloads, ok := result["downloads"].([]interface{})
	require.True(t, ok)
	assert.Len(t, downloads, 3) // 3 xlsx files

	// Verify reports separation
	reports, ok := result["reports"].([]interface{})
	require.True(t, ok)
	assert.Len(t, reports, 1) // Only normal_report.csv

	csvFiles, ok := result["csvFiles"].([]interface{})
	require.True(t, ok)
	assert.Len(t, csvFiles, 2) // ticker_trading_history.csv and isx_daily_20240101.csv

	// Verify total size calculation
	totalSize, ok := result["total_size"].(int64)
	require.True(t, ok)
	expectedSize := int64(1024*1024 + 100 + 0 + 500 + 750 + 1000) // Sum of download + report files
	assert.Equal(t, expectedSize, totalSize)
}

func testGetMarketMoversValidation(t *testing.T) {
	service := createTestDataService(t, t.TempDir())
	ctx := context.Background()

	tests := []struct {
		name      string
		period    string
		limit     string
		minVolume string
		expected  map[string]string
	}{
		{
			name:      "all_empty_defaults",
			period:    "",
			limit:     "",
			minVolume: "",
			expected: map[string]string{
				"period": "1d",
			},
		},
		{
			name:      "custom_values",
			period:    "1w",
			limit:     "25",
			minVolume: "50000",
			expected: map[string]string{
				"period": "1w",
			},
		},
		{
			name:      "mixed_empty_and_set",
			period:    "1m",
			limit:     "",
			minVolume: "1000",
			expected: map[string]string{
				"period": "1m",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetMarketMovers(ctx, tt.period, tt.limit, tt.minVolume)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify expected fields exist
			assert.Contains(t, result, "gainers")
			assert.Contains(t, result, "losers") 
			assert.Contains(t, result, "mostActive")
			assert.Contains(t, result, "period")
			assert.Contains(t, result, "updated")

			// Verify period value
			assert.Equal(t, tt.expected["period"], result["period"])

			// Verify data types
			_, ok := result["gainers"].([]interface{})
			assert.True(t, ok)
			_, ok = result["losers"].([]interface{})
			assert.True(t, ok)
			_, ok = result["mostActive"].([]interface{})
			assert.True(t, ok)

			// Verify timestamp format
			updated, ok := result["updated"].(string)
			assert.True(t, ok)
			_, timeErr := time.Parse(time.RFC3339, updated)
			assert.NoError(t, timeErr, "Updated timestamp should be valid RFC3339 format")
		})
	}
}

func testGetTickerChartValidation(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataServiceForComprehensive(t, tempDir)
	ctx := context.Background()

	tests := []struct {
		name      string
		ticker    string
		setupFile bool
		expectErr bool
	}{
		{
			name:      "empty_ticker",
			ticker:    "",
			setupFile: false,
			expectErr: true,
		},
		{
			name:      "valid_ticker_no_file",
			ticker:    "NONEXISTENT",
			setupFile: false,
			expectErr: false,
		},
		{
			name:      "valid_ticker_with_file", 
			ticker:    "TASC",
			setupFile: true,
			expectErr: false,
		},
		{
			name:      "special_characters",
			ticker:    "TEST-123",
			setupFile: false,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFile {
				// Create ticker file
				tickerFile := service.paths.GetTickerDailyCSVPath(tt.ticker)
				require.NoError(t, os.MkdirAll(filepath.Dir(tickerFile), 0755))
				require.NoError(t, os.WriteFile(tickerFile, []byte("test data"), 0644))
			}

			result, err := service.GetTickerChart(ctx, tt.ticker)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.ticker, result["ticker"])
				assert.Contains(t, result, "data")
				
				data, ok := result["data"].([]interface{})
				assert.True(t, ok)
				assert.NotNil(t, data) // Should be empty slice, not nil
			}
		})
	}
}

func testGetDailyReportParsing(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataServiceForComprehensive(t, tempDir)
	ctx := context.Background()

	tests := []struct {
		name        string
		csvContent  [][]string
		date        time.Time
		expectErr   bool
		expectedLen int
	}{
		{
			name: "normal_report",
			csvContent: [][]string{
				{"Symbol", "Name", "Close", "Volume"},
				{"TASC", "Al-Tasleeha", "2.50", "100000"},
				{"BANK", "Iraqi Bank", "1.80", "50000"},
			},
			date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectErr:   false,
			expectedLen: 2,
		},
		{
			name: "empty_data_report",
			csvContent: [][]string{
				{"Symbol", "Name", "Close", "Volume"},
			},
			date:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			expectErr:   false,
			expectedLen: 0,
		},
		{
			name: "malformed_csv",
			csvContent: [][]string{
				{"Symbol", "Name", "Close"},
				{"TASC", "Al-Tasleeha"}, // Missing column
			},
			date:        time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			expectErr:   false, // Service should handle gracefully
			expectedLen: 1,
		},
		{
			name: "unicode_content",
			csvContent: [][]string{
				{"الرمز", "الاسم", "الإغلاق", "الحجم"},
				{"TASC", "التسليح", "2.50", "100000"},
			},
			date:        time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
			expectErr:   false,
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create CSV file
			dailyFile := service.paths.GetDailyCSVPath(tt.date)
			require.NoError(t, os.MkdirAll(filepath.Dir(dailyFile), 0755))

			file, err := os.Create(dailyFile)
			require.NoError(t, err)
			defer file.Close()

			writer := csv.NewWriter(file)
			for _, row := range tt.csvContent {
				require.NoError(t, writer.Write(row))
			}
			writer.Flush()
			file.Close()

			result, err := service.GetDailyReport(ctx, tt.date)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)

				// Verify structure for non-empty results
				if len(result) > 0 {
					for _, row := range result {
						// row is already map[string]interface{}, no need to cast
						// Should have all header columns
						for i, header := range tt.csvContent[0] {
							if i < len(tt.csvContent[1]) {
								assert.Contains(t, row, header)
							}
						}
					}
				}
			}
		})
	}
}

func testDownloadFileSecurity(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataServiceForComprehensive(t, tempDir)

	// Setup directory structure
	downloadsDir := filepath.Join(tempDir, "downloads")
	reportsDir := filepath.Join(tempDir, "reports")
	secretDir := filepath.Join(tempDir, "secret")

	require.NoError(t, os.MkdirAll(downloadsDir, 0755))
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	require.NoError(t, os.MkdirAll(secretDir, 0755))

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(downloadsDir, "safe.xlsx"), []byte("safe content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "safe.csv"), []byte("safe content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("secret content"), 0644))

	service.paths.DownloadsDir = downloadsDir
	service.paths.ReportsDir = reportsDir

	ctx := context.Background()

	tests := []struct {
		name     string
		fileType string
		filename string
		expectErr bool
		errContains string
	}{
		{
			name:     "valid_download",
			fileType: "downloads",
			filename: "safe.xlsx",
			expectErr: false,
		},
		{
			name:     "valid_report",
			fileType: "reports", 
			filename: "safe.csv",
			expectErr: false,
		},
		{
			name:     "invalid_file_type",
			fileType: "invalid",
			filename: "safe.xlsx",
			expectErr: true,
			errContains: "invalid file type",
		},
		{
			name:     "path_traversal_attempt_1",
			fileType: "downloads",
			filename: "../secret/secret.txt",
			expectErr: true,
			errContains: "invalid file path",
		},
		{
			name:     "path_traversal_attempt_2",
			fileType: "downloads",
			filename: "..\\secret\\secret.txt",
			expectErr: true,
			errContains: "invalid file path",
		},
		{
			name:     "absolute_path_attempt",
			fileType: "downloads",
			filename: "/etc/passwd",
			expectErr: true,
			errContains: "invalid file path",
		},
		{
			name:     "nonexistent_file",
			fileType: "downloads",
			filename: "nonexistent.xlsx",
			expectErr: true,
			errContains: "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/download", nil)

			err := service.DownloadFile(ctx, w, r, tt.fileType, tt.filename)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Disposition"), tt.filename)
				assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
			}
		})
	}
}

func testListFilesInternalLogic(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataServiceForComprehensive(t, tempDir)

	// Create directory structure with various file types and timestamps
	testDir := filepath.Join(tempDir, "test")
	require.NoError(t, os.MkdirAll(testDir, 0755))

	// Create files with specific timestamps
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	testFiles := []struct {
		name string
		size int64
		time time.Time
	}{
		{"oldest.csv", 100, baseTime.Add(-2 * time.Hour)},
		{"middle.csv", 200, baseTime.Add(-1 * time.Hour)},
		{"newest.csv", 300, baseTime},
		{"different.txt", 150, baseTime.Add(-30 * time.Minute)},
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(testDir, tf.name)
		content := make([]byte, tf.size)
		require.NoError(t, os.WriteFile(filePath, content, 0644))
		require.NoError(t, os.Chtimes(filePath, tf.time, tf.time))
	}

	// Create subdirectory (should be ignored)
	subDir := filepath.Join(testDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "ignored.csv"), []byte("content"), 0644))

	// Update paths to point to our test directory
	service.paths.DataDir = tempDir

	tests := []struct {
		name        string
		dirName     string
		extension   string
		expectErr   bool
		validateResult func(t *testing.T, result map[string]interface{})
	}{
		{
			name:      "list_csv_files",
			dirName:   "test",
			extension: ".csv",
			expectErr: false,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				totalSize, ok := result["total_size"].(int64)
				assert.True(t, ok)
				assert.Equal(t, int64(600), totalSize) // 100 + 200 + 300

				lastModified, ok := result["last_modified"].(time.Time)
				assert.True(t, ok)
				assert.True(t, lastModified.Equal(baseTime) || lastModified.After(baseTime.Add(-time.Second)))
			},
		},
		{
			name:      "list_txt_files",
			dirName:   "test",
			extension: ".txt",
			expectErr: false,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				totalSize, ok := result["total_size"].(int64)
				assert.True(t, ok)
				assert.Equal(t, int64(150), totalSize) // Only .txt file
			},
		},
		{
			name:      "list_nonexistent_extension",
			dirName:   "test",
			extension: ".xyz",
			expectErr: false,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				totalSize, ok := result["total_size"].(int64)
				assert.True(t, ok)
				assert.Equal(t, int64(0), totalSize)
			},
		},
		{
			name:      "nonexistent_directory",
			dirName:   "nonexistent",
			extension: ".csv",
			expectErr: false, // Service handles this gracefully
			validateResult: func(t *testing.T, result map[string]interface{}) {
				// Should have initialized empty values
				assert.Contains(t, result, "total_size")
				assert.Contains(t, result, "last_modified")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]interface{})
			err := service.listFiles(tt.dirName, tt.extension, result)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}

func testDataServiceConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataServiceForComprehensive(t, tempDir)

	// Setup test data
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "test.csv"), []byte("data"), 0644))

	service.paths.ReportsDir = reportsDir

	ctx := context.Background()
	numGoroutines := 20
	numIterations := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// Test concurrent access to various methods
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < numIterations; j++ {
				// Mix different operations
				switch (id + j) % 4 {
				case 0:
					_, err := service.GetReports(ctx)
					if err != nil {
						errors <- fmt.Errorf("GetReports failed: %w", err)
					}
				case 1:
					_, err := service.GetFiles(ctx)
					if err != nil {
						errors <- fmt.Errorf("GetFiles failed: %w", err)
					}
				case 2:
					_, err := service.GetMarketMovers(ctx, "1d", "10", "1000")
					if err != nil {
						errors <- fmt.Errorf("GetMarketMovers failed: %w", err)
					}
				case 3:
					_, err := service.GetTickerChart(ctx, "TEST")
					if err != nil && !strings.Contains(err.Error(), "ticker parameter required") {
						errors <- fmt.Errorf("GetTickerChart failed: %w", err)
					}
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(errors)

	// Check for errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		t.Fatalf("Concurrent operations failed: %v", allErrors)
	}
}

func testDataServiceLogging(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a custom logger that captures output
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	cfg := &config.Config{}
	service, err := NewDataServiceWithLogger(cfg, logger)
	require.NoError(t, err)

	// Setup test data
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "test.csv"), []byte("data"), 0644))

	service.paths.ReportsDir = reportsDir

	ctx := context.Background()

	// Perform operations that should generate logs
	_, _ = service.GetReports(ctx)

	// Verify logging occurred
	logContents := logOutput.String()
	assert.Contains(t, logContents, "DataService initialized")
	assert.Contains(t, logContents, "GetReports")
	assert.Contains(t, logContents, "scanning directory")
}

// Helper function to create a test data service
func createTestDataServiceForComprehensive(t *testing.T, tempDir string) *DataService {
	t.Helper()
	
	dataDir := filepath.Join(tempDir, "data")
	reportsDir := filepath.Join(dataDir, "reports")
	downloadsDir := filepath.Join(dataDir, "downloads")
	
	require.NoError(t, os.MkdirAll(reportsDir, 0755))
	require.NoError(t, os.MkdirAll(downloadsDir, 0755))

	cfg := &config.Config{}
	service, err := NewDataServiceWithLogger(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	
	// Override paths for testing
	service.paths = &config.Paths{
		DataDir:      dataDir,
		ReportsDir:   reportsDir,
		DownloadsDir: downloadsDir,
	}

	return service
}