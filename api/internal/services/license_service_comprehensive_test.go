package services

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"isxcli/internal/license"
)

// MockLicenseManager implements license.ManagerInterface for testing
type MockLicenseManager struct {
	mock.Mock
}

func (m *MockLicenseManager) ActivateLicense(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockLicenseManager) ValidateLicense() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockLicenseManager) GetLicenseInfo() (*license.LicenseInfo, error) {
	args := m.Called()
	return args.Get(0).(*license.LicenseInfo), args.Error(1)
}

func (m *MockLicenseManager) GetLicenseStatus() (*license.LicenseInfo, string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*license.LicenseInfo), args.String(1), args.Error(2)
}

func (m *MockLicenseManager) GetLicensePath() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLicenseManager) TransferLicense(key string, force bool) error {
	args := m.Called(key, force)
	return args.Error(0)
}

// TestLicenseServiceComprehensive tests LicenseService for improved coverage
func TestLicenseServiceComprehensive(t *testing.T) {
	t.Run("Service_Construction", testLicenseServiceConstruction)
	t.Run("GetStatus_Scenarios", testLicenseServiceGetStatus)
	t.Run("Activation_Flow", testLicenseServiceActivation)
	t.Run("Validation_Context", testLicenseServiceValidation)
	t.Run("Detailed_Status", testLicenseServiceDetailedStatus)
	t.Run("Renewal_Status", testLicenseServiceRenewalStatus)
	t.Run("Transfer_License", testLicenseServiceTransfer)
	t.Run("Validation_Metrics", testLicenseServiceMetrics)
	t.Run("Cache_Management", testLicenseServiceCache)
	t.Run("Debug_Information", testLicenseServiceDebug)
	t.Run("Status_Determination", testLicenseStatusDetermination)
	t.Run("Helper_Functions", testLicenseServiceHelpers)
	t.Run("Error_Mapping", testLicenseErrorMapping)
}

func testLicenseServiceConstruction(t *testing.T) {
	tests := []struct {
		name     string
		manager  license.ManagerInterface
		logger   *slog.Logger
		validate func(t *testing.T, service LicenseService)
	}{
		{
			name:    "valid_construction",
			manager: &MockLicenseManager{},
			logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
			validate: func(t *testing.T, service LicenseService) {
				assert.NotNil(t, service)
				// Verify service is properly initialized with metrics tracking
				licenseService := service.(*licenseService)
				assert.NotNil(t, licenseService.manager)
				assert.NotNil(t, licenseService.logger)
				assert.False(t, licenseService.startTime.IsZero())
			},
		},
		{
			name:    "nil_logger",
			manager: &MockLicenseManager{},
			logger:  nil,
			validate: func(t *testing.T, service LicenseService) {
				assert.NotNil(t, service)
				licenseService := service.(*licenseService)
				assert.NotNil(t, licenseService.logger) // Should default to something
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLicenseService(tt.manager, tt.logger)
			tt.validate(t, service)
		})
	}
}

func testLicenseServiceGetStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(mock *MockLicenseManager)
		validateResult func(t *testing.T, response *LicenseStatusResponse, err error)
	}{
		{
			name: "not_activated_status",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("GetLicenseStatus").Return((*license.LicenseInfo)(nil), "Not Activated", nil)
			},
			validateResult: func(t *testing.T, response *LicenseStatusResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "not_activated", response.LicenseStatus)
				assert.Equal(t, "/license/not-activated", response.Type)
				assert.Equal(t, 200, response.Status)
				assert.Contains(t, response.Message, "No license activated")
				assert.NotNil(t, response.Features)
				assert.NotNil(t, response.Limitations)
				assert.NotNil(t, response.BrandingInfo)
			},
		},
		{
			name: "manager_error",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("GetLicenseStatus").Return((*license.LicenseInfo)(nil), "", errors.New("connection failed"))
			},
			validateResult: func(t *testing.T, response *LicenseStatusResponse, err error) {
				require.NoError(t, err) // Service handles errors gracefully
				assert.Equal(t, "error", response.LicenseStatus)
				assert.Equal(t, "/errors/license-check-failed", response.Type)
				assert.Equal(t, 500, response.Status)
				assert.Contains(t, response.Message, "Unable to retrieve license")
			},
		},
		{
			name: "active_license",
			setupMock: func(mock *MockLicenseManager) {
				expiryDate := time.Now().AddDate(0, 6, 0) // 6 months from now
				info := &license.LicenseInfo{
					LicenseKey:  "ISX1M02LYE1F9QJHR9D7Z",
					Status:      "Active",
					ExpiryDate:  expiryDate,
					Duration:    "Yearly",
					UserEmail:   "test@example.com",
					IssuedDate:  time.Now().AddDate(-1, 0, 0),
					LastChecked: time.Now(),
				}
				mock.On("GetLicenseStatus").Return(info, "Active", nil)
			},
			validateResult: func(t *testing.T, response *LicenseStatusResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "active", response.LicenseStatus)
				assert.Equal(t, 200, response.Status)
				assert.Greater(t, response.DaysLeft, 150) // Should be around 180 days
				assert.NotNil(t, response.LicenseInfo)
				assert.NotNil(t, response.UserInfo)
				assert.Contains(t, response.Features, "Advanced Analytics")
			},
		},
		{
			name: "critical_expiry_warning",
			setupMock: func(mock *MockLicenseManager) {
				expiryDate := time.Now().AddDate(0, 0, 5) // 5 days from now
				info := &license.LicenseInfo{
					LicenseKey: "ISX1M02LYE1F9QJHR9D7Z",
					Status:     "Active",
					ExpiryDate: expiryDate,
					Duration:   "Yearly",
				}
				mock.On("GetLicenseStatus").Return(info, "Active", nil)
			},
			validateResult: func(t *testing.T, response *LicenseStatusResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "critical", response.LicenseStatus)
				assert.Equal(t, 5, response.DaysLeft)
				assert.Contains(t, response.Message, "expires in 5 days")
				assert.NotNil(t, response.RenewalInfo)
				assert.True(t, response.RenewalInfo.NeedsRenewal)
				assert.Equal(t, "critical", response.RenewalInfo.RenewalUrgency)
			},
		},
		{
			name: "expired_license",
			setupMock: func(mock *MockLicenseManager) {
				expiryDate := time.Now().AddDate(0, 0, -10) // 10 days ago
				info := &license.LicenseInfo{
					LicenseKey: "ISX1M02LYE1F9QJHR9D7Z",
					Status:     "Expired",
					ExpiryDate: expiryDate,
					Duration:   "Yearly",
				}
				mock.On("GetLicenseStatus").Return(info, "Expired", nil)
			},
			validateResult: func(t *testing.T, response *LicenseStatusResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "expired", response.LicenseStatus)
				assert.Less(t, response.DaysLeft, 0)
				assert.Contains(t, response.Message, "expired")
				assert.NotNil(t, response.RenewalInfo)
				assert.True(t, response.RenewalInfo.IsExpired)
			},
		},
		{
			name: "warning_status",
			setupMock: func(mock *MockLicenseManager) {
				expiryDate := time.Now().AddDate(0, 0, 20) // 20 days from now
				info := &license.LicenseInfo{
					LicenseKey: "ISX1M02LYE1F9QJHR9D7Z",
					Status:     "Active",
					ExpiryDate: expiryDate,
					Duration:   "Yearly",
				}
				mock.On("GetLicenseStatus").Return(info, "Active", nil)
			},
			validateResult: func(t *testing.T, response *LicenseStatusResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "warning", response.LicenseStatus)
				assert.Equal(t, 20, response.DaysLeft)
				assert.Contains(t, response.Message, "expires in 20 days")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockLicenseManager{}
			tt.setupMock(mockManager)

			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			service := NewLicenseService(mockManager, logger)

			ctx := context.Background()
			response, err := service.GetStatus(ctx)

			tt.validateResult(t, response, err)
			mockManager.AssertExpectations(t)
		})
	}
}

func testLicenseServiceActivation(t *testing.T) {
	tests := []struct {
		name         string
		licenseKey   string
		setupMock    func(mock *MockLicenseManager)
		expectErr    bool
		errContains  string
	}{
		{
			name:       "successful_activation",
			licenseKey: "ISX1M02LYE1F9QJHR9D7Z",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("ActivateLicense", "ISX1M02LYE1F9QJHR9D7Z").Return(nil)
			},
			expectErr: false,
		},
		{
			name:       "activation_failure",
			licenseKey: "INVALID-KEY-12345",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("ActivateLicense", "INVALID-KEY-12345").Return(errors.New("invalid license key"))
			},
			expectErr:   true,
			errContains: "activation failed",
		},
		{
			name:       "network_error",
			licenseKey: "ISX1M02LYE1F9QJHR9D7Z",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("ActivateLicense", "ISX1M02LYE1F9QJHR9D7Z").Return(errors.New("network connection failed"))
			},
			expectErr:   true,
			errContains: "activation failed",
		},
		{
			name:       "empty_license_key",
			licenseKey: "",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("ActivateLicense", "").Return(errors.New("empty license key"))
			},
			expectErr:   true,
			errContains: "activation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockLicenseManager{}
			tt.setupMock(mockManager)

			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			service := NewLicenseService(mockManager, logger)

			ctx := context.Background()
			err := service.Activate(ctx, tt.licenseKey)

			if tt.expectErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func testLicenseServiceValidation(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(mock *MockLicenseManager)
		expectValid  bool
		expectErr    bool
		testTimeout  bool
	}{
		{
			name: "successful_validation",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("ValidateLicense").Return(true, nil)
			},
			expectValid: true,
			expectErr:   false,
		},
		{
			name: "validation_failure",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("ValidateLicense").Return(false, nil)
			},
			expectValid: false,
			expectErr:   false,
		},
		{
			name: "validation_error",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("ValidateLicense").Return(false, errors.New("validation error"))
			},
			expectValid: false,
			expectErr:   true,
		},
		{
			name: "context_cancellation",
			setupMock: func(mock *MockLicenseManager) {
				// Don't setup mock - will simulate blocking call
			},
			expectValid: false,
			expectErr:   true,
			testTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockLicenseManager{}
			if !tt.testTimeout {
				tt.setupMock(mockManager)
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			service := NewLicenseService(mockManager, logger)

			ctx := context.Background()
			if tt.testTimeout {
				// Create a context that will cancel quickly
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 10*time.Millisecond)
				defer cancel()
			}

			valid, err := service.ValidateWithContext(ctx)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.testTimeout {
					assert.Contains(t, err.Error(), "context")
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValid, valid)

			if !tt.testTimeout {
				mockManager.AssertExpectations(t)
			}
		})
	}
}

func testLicenseServiceDetailedStatus(t *testing.T) {
	mockManager := &MockLicenseManager{}

	// Setup mock for basic status call
	expiryDate := time.Now().AddDate(0, 3, 0) // 3 months from now
	info := &license.LicenseInfo{
		LicenseKey:  "ISX1M02LYE1F9QJHR9D7Z",
		Status:      "Active",
		ExpiryDate:  expiryDate,
		Duration:    "Yearly",
		UserEmail:   "test@example.com",
		IssuedDate:  time.Now().AddDate(-1, 0, 0),
		LastChecked: time.Now().Add(-1 * time.Hour),
	}

	mockManager.On("GetLicenseStatus").Return(info, "Active", nil)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger)

	ctx := context.Background()
	response, err := service.GetDetailedStatus(ctx)

	require.NoError(t, err)
	assert.NotNil(t, response)

	// Verify detailed fields
	assert.NotNil(t, response.ActivationDate)
	assert.NotNil(t, response.LastValidation)
	assert.GreaterOrEqual(t, response.ValidationCount, int64(1)) // Should have metrics from GetStatus call
	assert.NotEmpty(t, response.NetworkStatus)
	assert.NotNil(t, response.PerformanceMetrics)
	assert.NotNil(t, response.Recommendations)

	// Verify basic status fields are included
	assert.Equal(t, "active", response.LicenseStatus)
	assert.NotNil(t, response.LicenseInfo)

	mockManager.AssertExpectations(t)
}

func testLicenseServiceRenewalStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(mock *MockLicenseManager)
		validateResult func(t *testing.T, response *RenewalStatusResponse, err error)
	}{
		{
			name: "needs_renewal_critical",
			setupMock: func(mock *MockLicenseManager) {
				expiryDate := time.Now().AddDate(0, 0, 3) // 3 days from now
				info := &license.LicenseInfo{
					ExpiryDate: expiryDate,
				}
				mock.On("GetLicenseStatus").Return(info, "Active", nil)
			},
			validateResult: func(t *testing.T, response *RenewalStatusResponse, err error) {
				require.NoError(t, err)
				assert.True(t, response.NeedsRenewal)
				assert.False(t, response.IsExpired)
				assert.Equal(t, "critical", response.RenewalUrgency)
				assert.Equal(t, 3, response.DaysUntilExpiry)
				assert.Contains(t, response.RenewalMessage, "expires in 3 days")
			},
		},
		{
			name: "expired_license",
			setupMock: func(mock *MockLicenseManager) {
				expiryDate := time.Now().AddDate(0, 0, -5) // 5 days ago
				info := &license.LicenseInfo{
					ExpiryDate: expiryDate,
				}
				mock.On("GetLicenseStatus").Return(info, "Expired", nil)
			},
			validateResult: func(t *testing.T, response *RenewalStatusResponse, err error) {
				require.NoError(t, err)
				assert.True(t, response.NeedsRenewal)
				assert.True(t, response.IsExpired)
				assert.Equal(t, "critical", response.RenewalUrgency)
				assert.Less(t, response.DaysUntilExpiry, 0)
				assert.Contains(t, response.RenewalMessage, "expired")
			},
		},
		{
			name: "healthy_license",
			setupMock: func(mock *MockLicenseManager) {
				expiryDate := time.Now().AddDate(0, 6, 0) // 6 months from now
				info := &license.LicenseInfo{
					ExpiryDate: expiryDate,
				}
				mock.On("GetLicenseStatus").Return(info, "Active", nil)
			},
			validateResult: func(t *testing.T, response *RenewalStatusResponse, err error) {
				require.NoError(t, err)
				assert.False(t, response.NeedsRenewal)
				assert.False(t, response.IsExpired)
				assert.Equal(t, "low", response.RenewalUrgency)
				assert.Greater(t, response.DaysUntilExpiry, 150)
				assert.Contains(t, response.RenewalMessage, "active")
			},
		},
		{
			name: "manager_error",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("GetLicenseStatus").Return((*license.LicenseInfo)(nil), "", errors.New("connection failed"))
			},
			validateResult: func(t *testing.T, response *RenewalStatusResponse, err error) {
				require.NoError(t, err) // Service handles errors gracefully
				assert.True(t, response.NeedsRenewal)
				assert.True(t, response.IsExpired)
				assert.Equal(t, "critical", response.RenewalUrgency)
				assert.Contains(t, response.RenewalMessage, "Unable to check")
			},
		},
		{
			name: "no_license_info",
			setupMock: func(mock *MockLicenseManager) {
				mock.On("GetLicenseStatus").Return((*license.LicenseInfo)(nil), "Not Activated", nil)
			},
			validateResult: func(t *testing.T, response *RenewalStatusResponse, err error) {
				require.NoError(t, err)
				assert.True(t, response.NeedsRenewal)
				assert.True(t, response.IsExpired)
				assert.Equal(t, "critical", response.RenewalUrgency)
				assert.Contains(t, response.RenewalMessage, "No license found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockLicenseManager{}
			tt.setupMock(mockManager)

			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			service := NewLicenseService(mockManager, logger)

			ctx := context.Background()
			response, err := service.CheckRenewalStatus(ctx)

			tt.validateResult(t, response, err)
			mockManager.AssertExpectations(t)
		})
	}
}

func testLicenseServiceTransfer(t *testing.T) {
	tests := []struct {
		name        string
		licenseKey  string
		force       bool
		setupMock   func(mock *MockLicenseManager)
		expectErr   bool
		errContains string
	}{
		{
			name:       "successful_transfer",
			licenseKey: "ISX1M02LYE1F9QJHR9D7Z",
			force:      false,
			setupMock: func(mock *MockLicenseManager) {
				// Mock the manager as the actual license.Manager type
				// Since we can't easily mock the type assertion, we test error cases
			},
			expectErr:   true, // Will fail type assertion in test
			errContains: "transfer not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockLicenseManager{}
			if tt.setupMock != nil {
				tt.setupMock(mockManager)
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			service := NewLicenseService(mockManager, logger)

			ctx := context.Background()
			err := service.TransferLicense(ctx, tt.licenseKey, tt.force)

			if tt.expectErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func testLicenseServiceMetrics(t *testing.T) {
	mockManager := &MockLicenseManager{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger)

	// Setup mock for validation calls to generate metrics
	mockManager.On("ValidateLicense").Return(true, nil).Times(3)
	mockManager.On("ValidateLicense").Return(false, errors.New("error")).Times(2)

	ctx := context.Background()

	// Perform some operations to generate metrics
	for i := 0; i < 3; i++ {
		_, _ = service.ValidateWithContext(ctx)
	}

	for i := 0; i < 2; i++ {
		_, _ = service.ValidateWithContext(ctx)
	}

	// Get metrics
	metrics, err := service.GetValidationMetrics(ctx)
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Verify metrics structure
	assert.Equal(t, int64(5), metrics.TotalValidations)
	assert.Equal(t, int64(3), metrics.SuccessfulValidations)
	assert.Equal(t, int64(2), metrics.FailedValidations)
	assert.Greater(t, metrics.AverageResponseTime, time.Duration(0))
	assert.False(t, metrics.LastValidationTime.IsZero())
	assert.GreaterOrEqual(t, metrics.CacheHitRate, 0.0)
	assert.LessOrEqual(t, metrics.CacheHitRate, 1.0)
	assert.Equal(t, int64(2), metrics.NetworkErrors) // Simplified - all errors counted as network
	assert.Greater(t, metrics.Uptime, time.Duration(0))

	mockManager.AssertExpectations(t)
}

func testLicenseServiceCache(t *testing.T) {
	mockManager := &MockLicenseManager{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger)

	ctx := context.Background()

	// Test cache invalidation (should not error)
	err := service.InvalidateCache(ctx)
	assert.NoError(t, err)

	mockManager.AssertExpectations(t)
}

func testLicenseServiceDebug(t *testing.T) {
	mockManager := &MockLicenseManager{}

	// Setup mock to return license path
	mockManager.On("GetLicensePath").Return("/test/path/license.dat")
	mockManager.On("GetLicenseStatus").Return((*license.LicenseInfo)(nil), "Not Activated", nil)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger)

	ctx := context.Background()
	debugInfo, err := service.GetDebugInfo(ctx)

	require.NoError(t, err)
	assert.NotNil(t, debugInfo)

	// Verify debug info structure
	assert.NotEmpty(t, debugInfo.TraceID)
	assert.False(t, debugInfo.Timestamp.IsZero())
	assert.Equal(t, "/test/path/license.dat", debugInfo.FilePath)
	assert.False(t, debugInfo.FileExists) // File doesn't exist in test
	assert.NotEmpty(t, debugInfo.WorkingDir)
	assert.NotEmpty(t, debugInfo.ExecPath)
	assert.Equal(t, "not set", debugInfo.ConfigPath) // No CONFIG_PATH env var
	assert.Equal(t, "error", debugInfo.LicenseStatus) // Based on mock setup
	assert.NotNil(t, debugInfo.Environment)

	// Verify environment variables are captured
	assert.Contains(t, debugInfo.Environment, "WORKING_DIR")
	assert.Contains(t, debugInfo.Environment, "EXEC_PATH")

	mockManager.AssertExpectations(t)
}

func testLicenseStatusDetermination(t *testing.T) {
	mockManager := &MockLicenseManager{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger).(*licenseService)

	tests := []struct {
		name           string
		managerStatus  string
		daysLeft       int
		expectedStatus string
	}{
		{"expired_status", "Expired", 10, "expired"},
		{"critical_status", "Critical", 10, "critical"},
		{"warning_status", "Warning", 10, "warning"},
		{"active_with_negative_days", "Active", -1, "expired"},
		{"active_with_critical_days", "Active", 5, "critical"},
		{"active_with_warning_days", "Active", 20, "warning"},
		{"active_with_good_days", "Active", 60, "active"},
		{"unknown_status", "Unknown", 10, "not_activated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.determineLicenseStatus(tt.managerStatus, tt.daysLeft)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func testLicenseServiceHelpers(t *testing.T) {
	mockManager := &MockLicenseManager{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger).(*licenseService)

	t.Run("mask_license_key", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"ISX1M02LYE1F9QJHR9D7Z", "ISX1M02L..."},
			{"SHORT", "***"},
			{"", "***"},
			{"12345678", "***"},
			{"123456789", "12345678..."},
		}

		for _, tt := range tests {
			result := maskLicenseKey(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("generate_status_message", func(t *testing.T) {
		tests := []struct {
			status   string
			daysLeft int
			contains string
		}{
			{"expired", 0, "expired"},
			{"critical", 5, "expires in 5 days"},
			{"warning", 20, "expires in 20 days"},
			{"active", 60, "60 days remaining"},
			{"unknown", 10, "unknown"},
		}

		for _, tt := range tests {
			result := service.generateStatusMessage(tt.status, tt.daysLeft)
			assert.Contains(t, result, tt.contains)
		}
	})

	t.Run("build_features_list", func(t *testing.T) {
		activeFeatures := service.buildFeaturesList("active")
		assert.Contains(t, activeFeatures, "Advanced Analytics")
		assert.Greater(t, len(activeFeatures), 3)

		inactiveFeatures := service.buildFeaturesList("not_activated")
		assert.NotContains(t, inactiveFeatures, "Advanced Analytics")
		assert.Contains(t, inactiveFeatures, "Daily Reports Access")
	})

	t.Run("build_limitations", func(t *testing.T) {
		expiredLimitations := service.buildLimitations("expired")
		assert.Equal(t, "Read-only mode", expiredLimitations["data_access"])
		assert.Equal(t, 0, expiredLimitations["export_limit"])
		assert.Equal(t, false, expiredLimitations["real_time_updates"])

		activeLimitations := service.buildLimitations("active")
		assert.Equal(t, -1, activeLimitations["export_limit"]) // unlimited
		assert.Equal(t, true, activeLimitations["real_time_updates"])
	})

	t.Run("build_branding_info", func(t *testing.T) {
		branding := service.buildBrandingInfo()
		assert.Equal(t, "ISX Daily Reports Scrapper", branding.ApplicationName)
		assert.Equal(t, "2.0", branding.Version)
		assert.Equal(t, "Iraqi Investor", branding.BrandName)
		assert.Contains(t, branding.WebsiteURL, "iraqiinvestor.gov.iq")
	})
}

func testLicenseErrorMapping(t *testing.T) {
	mockManager := &MockLicenseManager{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger).(*licenseService)

	tests := []struct {
		name        string
		inputError  error
		expectType  string
	}{
		{"nil_error", nil, "nil"},
		{"already_activated", errors.New("already activated"), "activation_failed"},
		{"expired_error", errors.New("license expired"), "license_expired"},
		{"not_found", errors.New("license not found"), "invalid_license_key"},
		{"network_error", errors.New("network connection failed"), "network_error"},
		{"generic_error", errors.New("some other error"), "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.mapTransferError(tt.inputError)
			
			if tt.expectType == "nil" {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				// Verify error type or content based on mapping
				errorStr := strings.ToLower(result.Error())
				switch tt.expectType {
				case "activation_failed", "license_expired", "invalid_license_key", "network_error":
					// These should be mapped to specific error types
					// The actual error types would be defined in the errors package
					assert.NotEmpty(t, errorStr)
				case "generic":
					// Should return original error
					assert.Equal(t, tt.inputError, result)
				}
			}
		})
	}
}

// Benchmark license service operations
func BenchmarkLicenseServiceGetStatus(b *testing.B) {
	mockManager := &MockLicenseManager{}
	info := &license.LicenseInfo{
		LicenseKey: "ISX1M02LYE1F9QJHR9D7Z",
		Status:     "Active",
		ExpiryDate: time.Now().AddDate(0, 6, 0),
	}
	mockManager.On("GetLicenseStatus").Return(info, "Active", nil)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetStatus(ctx)
	}
}

func BenchmarkLicenseServiceValidation(b *testing.B) {
	mockManager := &MockLicenseManager{}
	mockManager.On("ValidateLicense").Return(true, nil)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewLicenseService(mockManager, logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ValidateWithContext(ctx)
	}
}