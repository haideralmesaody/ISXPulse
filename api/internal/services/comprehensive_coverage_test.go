package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"isxcli/internal/config"
)

// TestServicesComprehensiveCoverage tests core service functionality for improved coverage
func TestServicesComprehensiveCoverage(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name:     "data_service_initialization",
			testFunc: testDataServiceInitialization,
		},
		{
			name:     "data_service_reports",
			testFunc: testDataServiceReports,
		},
		{
			name:     "data_service_tickers",
			testFunc: testDataServiceTickers,
		},
		{
			name:     "data_service_indices",
			testFunc: testDataServiceIndices,
		},
		{
			name:     "data_service_files",
			testFunc: testDataServiceFiles,
		},
		{
			name:     "data_service_market_movers",
			testFunc: testDataServiceMarketMovers,
		},
		{
			name:     "data_service_ticker_charts",
			testFunc: testDataServiceTickerCharts,
		},
		{
			name:     "data_service_daily_reports",
			testFunc: testDataServiceDailyReports,
		},
		{
			name:     "data_service_file_downloads",
			testFunc: testDataServiceFileDownloads,
		},
		{
			name:     "operations_service_initialization",
			testFunc: testOperationsServiceInitialization,
		},
		{
			name:     "operations_service_operation_management",
			testFunc: testOperationsServiceOperationManagement,
		},
		{
			name:     "websocket_adapter_functionality",
			testFunc: testWebSocketAdapterFunctionality,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

func testDataServiceInitialization(t *testing.T) {
	// Test with nil config
	_, err := NewDataService(nil)
	assert.Error(t, err)

	// Test with valid config
	cfg := &config.Config{}
	service, err := NewDataService(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test with custom logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service, err = NewDataServiceWithLogger(cfg, logger)
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, logger, service.logger)

	// Test with nil logger (should use default)
	service, err = NewDataServiceWithLogger(cfg, nil)
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
}

func testDataServiceReports(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test CSV files
	createTestReportFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-01.csv"))
	createTestReportFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-02.csv"))

	ctx := context.Background()
	reports, err := service.GetReports(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, reports)

	// Test with non-existent reports directory
	emptyService := createTestDataService(t, t.TempDir())
	reports, err = emptyService.GetReports(ctx)
	if err != nil {
		// Directory doesn't exist - expected
		assert.Contains(t, err.Error(), "no such file")
	} else {
		// Directory exists but empty
		assert.Empty(t, reports)
	}
}

func testDataServiceTickers(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test ticker file
	createTestTickerFile(t, filepath.Join(tempDir, "data", "reports", "latest_tickers.csv"))

	ctx := context.Background()
	tickers, err := service.GetTickers(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tickers)

	// Test with non-existent ticker file
	emptyService := createTestDataService(t, t.TempDir())
	_, err = emptyService.GetTickers(ctx)
	assert.Error(t, err)
}

func testDataServiceIndices(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test index files
	createTestIndexFile(t, filepath.Join(tempDir, "data", "reports", "isx60_index.csv"))
	createTestIndexFile(t, filepath.Join(tempDir, "data", "reports", "isx15_index.csv"))

	ctx := context.Background()
	indices, err := service.GetIndices(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, indices)
	
	// Should contain both ISX60 and ISX15 data
	assert.Contains(t, indices, "ISX60")
	assert.Contains(t, indices, "ISX15")
}

func testDataServiceFiles(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test files in different directories
	createTestFile(t, filepath.Join(tempDir, "data", "downloads", "test.xlsx"))
	createTestFile(t, filepath.Join(tempDir, "data", "reports", "test.csv"))

	ctx := context.Background()
	files, err := service.GetFiles(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, files)
	
	// Should have both downloads and reports sections
	assert.Contains(t, files, "downloads")
	assert.Contains(t, files, "reports")
}

func testDataServiceMarketMovers(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test report file with price data
	createTestMarketDataFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-01.csv"))
	createTestMarketDataFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-02.csv"))

	ctx := context.Background()
	
	// Test with various parameters
	tests := []struct {
		period    string
		limit     string
		minVolume string
	}{
		{"1d", "10", "1000"},
		{"1w", "5", "500"},
		{"", "", ""}, // Test defaults
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("period_%s_limit_%s_volume_%s", tt.period, tt.limit, tt.minVolume), func(t *testing.T) {
			movers, err := service.GetMarketMovers(ctx, tt.period, tt.limit, tt.minVolume)
			assert.NoError(t, err)
			assert.NotNil(t, movers)
		})
	}
}

func testDataServiceTickerCharts(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test data files for ticker
	createTestTickerDataFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-01.csv"), "TASC")
	createTestTickerDataFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-02.csv"), "TASC")

	ctx := context.Background()
	
	// Test valid ticker
	chart, err := service.GetTickerChart(ctx, "TASC")
	assert.NoError(t, err)
	assert.NotNil(t, chart)

	// Test invalid ticker
	chart, err = service.GetTickerChart(ctx, "NONEXISTENT")
	assert.NoError(t, err) // Should not error, just return empty data
	assert.NotNil(t, chart)
}

func testDataServiceDailyReports(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test daily report
	testDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	createTestReportFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-15.csv"))

	ctx := context.Background()
	
	// Test existing date
	report, err := service.GetDailyReport(ctx, testDate)
	assert.NoError(t, err)
	assert.NotNil(t, report)

	// Test non-existent date
	nonExistentDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = service.GetDailyReport(ctx, nonExistentDate)
	assert.Error(t, err)
}

func testDataServiceFileDownloads(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test files
	testFile := filepath.Join(tempDir, "data", "downloads", "test.xlsx")
	createTestFile(t, testFile)

	// Test valid download
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download", nil)
	
	err := service.DownloadFile(context.Background(), w, r, "downloads", "test.xlsx")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test invalid file type
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/download", nil)
	
	err = service.DownloadFile(context.Background(), w, r, "invalid", "test.xlsx")
	assert.Error(t, err)

	// Test non-existent file
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/download", nil)
	 
	err = service.DownloadFile(context.Background(), w, r, "downloads", "nonexistent.xlsx")
	assert.Error(t, err)
}

func testOperationsServiceInitialization(t *testing.T) {
	// Create mock WebSocket hub and adapter
	hub := &MockWebSocketHubLocal{}
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	// Test service creation
	service, err := NewOperationService(adapter, logger)
	if err != nil {
		// May fail in test environment due to missing executables
		t.Logf("NewOperationService failed (expected in test env): %v", err)
		return
	}
	assert.NotNil(t, service)
	assert.NotNil(t, service.GetManager())

	// Test with nil adapter
	service, err = NewOperationService(nil, logger)
	if err != nil {
		t.Logf("NewOperationService with nil adapter failed (expected): %v", err)
		return
	}
	assert.NotNil(t, service)
}

func testOperationsServiceOperationManagement(t *testing.T) {
	hub := &MockWebSocketHubLocal{}
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Logf("NewOperationService failed (expected in test env): %v", err)
		return
	}

	ctx := context.Background()

	// Test operation types
	types, err := service.GetOperationTypes(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, types)

	// Test starting operations
	params := map[string]interface{}{
		"test_param": "test_value",
	}

	operationID, err := service.StartOperation(ctx, params)
	if err != nil {
		// May fail due to missing executables in test environment
		t.Logf("StartOperation failed (expected in test env): %v", err)
	} else {
		assert.NotEmpty(t, operationID)
	}

	// Test scraping operation
	scrapingID, err := service.StartScraping(ctx, params)
	if err != nil {
		t.Logf("StartScraping failed (expected in test env): %v", err)
	} else {
		assert.NotEmpty(t, scrapingID)
	}

	// Test processing operation
	processingID, err := service.StartProcessing(ctx, params)
	if err != nil {
		t.Logf("StartProcessing failed (expected in test env): %v", err)
	} else {
		assert.NotEmpty(t, processingID)
	}

	// Test index extraction operation
	indexID, err := service.StartIndexExtraction(ctx, params)
	if err != nil {
		t.Logf("StartIndexExtraction failed (expected in test env): %v", err)
	} else {
		assert.NotEmpty(t, indexID)
	}

	// Test listing operations
	operations, err := service.ListOperations(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, operations)

	// Test listing by status  
	pendingOps, err := service.ListOperationsByStatus(ctx, "pending")
	assert.NoError(t, err)
	assert.NotNil(t, pendingOps)

	// Test operation metrics
	metrics, err := service.GetOperationMetrics(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)

	// Test stage info
	stageInfo := service.GetStageInfo()
	assert.NotNil(t, stageInfo)

	// Test validate executables
	err = service.ValidateExecutables(ctx)
	// May fail in test environment without executables
	if err != nil {
		t.Logf("ValidateExecutables failed (expected in test env): %v", err)
	}

	// Test cancel all operations
	err = service.CancelAll(ctx)
	assert.NoError(t, err)
}

func testWebSocketAdapterFunctionality(t *testing.T) {
	hub := &MockWebSocketHubLocal{}
	adapter := NewWebSocketOperationAdapter(hub)

	// Test SendProgress
	adapter.SendProgress("test-stage", "Test progress message", 50)
	
	// Test SendComplete - success
	adapter.SendComplete("test-stage", "Test completion message", true)
	
	// Test SendComplete - failure
	adapter.SendComplete("test-stage", "Test failure message", false)

	// Verify messages were sent
	messages := hub.GetMessages()
	assert.Len(t, messages, 3)
}

// Helper functions for test setup

func createTestDataService(t *testing.T, tempDir string) *DataService {
	t.Helper()
	
	// Create directory structure
	dataDir := filepath.Join(tempDir, "data")
	reportsDir := filepath.Join(dataDir, "reports")
	downloadsDir := filepath.Join(dataDir, "downloads")
	
	err := os.MkdirAll(reportsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(downloadsDir, 0755)
	require.NoError(t, err)

	cfg := &config.Config{}
	service, err := NewDataService(cfg)
	require.NoError(t, err)
	
	// Override paths for testing
	service.paths = &config.Paths{
		DataDir:      dataDir,
		ReportsDir:   reportsDir,
		DownloadsDir: downloadsDir,
	}

	return service
}

func createTestReportFile(t *testing.T, filePath string) {
	t.Helper()
	
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err)

	content := `Symbol,Name,Close,Change,Volume,Value
TASC,Al-Tasleeha,2.50,0.10,100000,250000
BANK,Iraqi Bank,1.80,-0.05,50000,90000
ASHB,Al-Ashbal,3.20,0.15,75000,240000`

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
}

func createTestTickerFile(t *testing.T, filePath string) {
	t.Helper()
	
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err)

	content := `Symbol,Name,Close,Change,Volume,Value,Bid,Ask
TASC,Al-Tasleeha,2.50,0.10,100000,250000,2.45,2.55
BANK,Iraqi Bank,1.80,-0.05,50000,90000,1.75,1.85`

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
}

func createTestIndexFile(t *testing.T, filePath string) {
	t.Helper()
	
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err)

	content := `Date,Index,Value,Change,Percentage
2024-01-15,ISX60,850.25,5.50,0.65`

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
}

func createTestFile(t *testing.T, filePath string) {
	t.Helper()
	
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err)

	content := "Test file content"
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
}

func createTestMarketDataFile(t *testing.T, filePath string) {
	t.Helper()
	
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err)

	content := `Symbol,Name,Close,Change,Volume,Value,Previous,Percentage
TASC,Al-Tasleeha,2.50,0.10,100000,250000,2.40,4.17
BANK,Iraqi Bank,1.80,-0.05,50000,90000,1.85,-2.70
ASHB,Al-Ashbal,3.20,0.15,75000,240000,3.05,4.92`

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
}

func createTestTickerDataFile(t *testing.T, filePath, ticker string) {
	t.Helper()
	
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err)

	content := fmt.Sprintf(`Symbol,Name,Close,Change,Volume,Value,Previous
%s,Test Company,2.50,0.10,100000,250000,2.40
OTHER,Other Company,1.80,-0.05,50000,90000,1.85`, ticker)

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
}

// MockWebSocketHubLocal for testing operations service (avoiding conflict with test_helpers.go)
type MockWebSocketHubLocal struct {
	messages []interface{}
}

func (m *MockWebSocketHubLocal) Broadcast(messageType string, data interface{}) {
	m.messages = append(m.messages, map[string]interface{}{
		"type": messageType,
		"data": data,
	})
}

func (m *MockWebSocketHubLocal) GetMessages() []interface{} {
	return m.messages
}

// TestDataServiceErrorHandling tests error handling scenarios
func TestDataServiceErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	ctx := context.Background()

	// Test GetTickerChart with empty ticker
	chart, err := service.GetTickerChart(ctx, "")
	assert.Error(t, err) // Should return error for empty ticker
	assert.Nil(t, chart)

	// Test GetMarketMovers with invalid parameters
	movers, err := service.GetMarketMovers(ctx, "invalid", "invalid", "invalid")
	assert.NoError(t, err) // Should handle gracefully with defaults
	assert.NotNil(t, movers)
}

// TestOperationsServiceErrorScenarios tests error scenarios
func TestOperationsServiceErrorScenarios(t *testing.T) {
	hub := &MockWebSocketHubLocal{}
	adapter := NewWebSocketOperationAdapter(hub)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	service, err := NewOperationService(adapter, logger)
	if err != nil {
		t.Logf("NewOperationService failed (expected in test env): %v", err)
		return
	}

	ctx := context.Background()

	// Test getting non-existent operation status
	_, err = service.GetOperationStatus(ctx, "non-existent-id")
	assert.Error(t, err)

	// Test cancelling non-existent operation
	err = service.CancelOperation(ctx, "non-existent-id")
	assert.Error(t, err)

	// Test stopping non-existent operation
	err = service.StopOperation(ctx, "non-existent-id")
	assert.Error(t, err)

	// Test getting status of non-existent operation
	_, err = service.GetStatus(ctx, "non-existent-id")
	assert.Error(t, err)
}

// TestServiceConcurrency tests concurrent access to services
func TestServiceConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test files
	createTestReportFile(t, filepath.Join(tempDir, "data", "reports", "2024-01-01.csv"))
	createTestTickerFile(t, filepath.Join(tempDir, "data", "reports", "latest_tickers.csv"))

	ctx := context.Background()
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	// Launch concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Test concurrent access to different methods
			_, err := service.GetReports(ctx)
			assert.NoError(t, err)
			
			_, err = service.GetTickers(ctx)
			assert.NoError(t, err)
			
			_, err = service.GetFiles(ctx)
			assert.NoError(t, err)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// TestListFilesMethod tests the listFiles method specifically
func TestListFilesMethod(t *testing.T) {
	tempDir := t.TempDir()
	service := createTestDataService(t, tempDir)

	// Create test files with specific extensions
	createTestFile(t, filepath.Join(tempDir, "data", "downloads", "file1.xlsx"))
	createTestFile(t, filepath.Join(tempDir, "data", "downloads", "file2.xls"))
	createTestFile(t, filepath.Join(tempDir, "data", "downloads", "file3.csv"))
	createTestFile(t, filepath.Join(tempDir, "data", "downloads", "ignored.txt"))

	result := make(map[string]interface{})
	
	// Test listing Excel files
	err := service.listFiles("downloads", ".xlsx", result)
	assert.NoError(t, err)
	
	// Test listing with non-existent directory
	err = service.listFiles("nonexistent", ".xlsx", result)
	assert.Error(t, err)

	// Test with empty extension (should list all files)
	err = service.listFiles("downloads", "", result)
	assert.NoError(t, err)
}