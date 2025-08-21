package license

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"isxcli/internal/config"
	licenseErrors "isxcli/internal/errors"
	"isxcli/internal/security"
)

// Constants for scratch card license system
const (
	// Scratch card license format patterns
	ScratchCardPrefix = "ISX-"
	ScratchCardLength = 17 // ISX-XXXX-XXXX-XXXX
)

// LicenseInfo represents license data
type LicenseInfo struct {
	LicenseKey        string    `json:"license_key"`
	UserEmail         string    `json:"user_email"`
	ExpiryDate        time.Time `json:"expiry_date"`
	Duration          string    `json:"duration"`
	IssuedDate        time.Time `json:"issued_date"`
	Status            string    `json:"status"`
	LastChecked       time.Time `json:"last_checked"`
	ActivationID      string    `json:"activation_id"`      // Unique activation identifier from Apps Script
	DeviceFingerprint string    `json:"device_fingerprint"` // Device fingerprint for validation
}

// ReactivationDetails holds information about license reactivation
type ReactivationDetails struct {
	ReactivationCount      int       `json:"reactivation_count"`
	MaxReactivations       int       `json:"max_reactivations"`
	SimilarityScore        float64   `json:"similarity_score"`
	PreviousDeviceInfo     string    `json:"previous_device_info"`
	ReactivationTimestamp  time.Time `json:"reactivation_timestamp"`
}

// GoogleSheetsConfig represents Google Sheets configuration
type GoogleSheetsConfig struct {
	SheetID            string `json:"sheet_id"`
	APIKey             string `json:"api_key"`
	SheetName          string `json:"sheet_name"`
	UseServiceAccount  bool   `json:"use_service_account"`
	// Legacy credential fields removed - only embedded encrypted credentials are supported
}

// PerformanceMetrics tracks operation performance
type PerformanceMetrics struct {
	Count        int64         `json:"count"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	MaxTime      time.Duration `json:"max_time"`
	MinTime      time.Duration `json:"min_time"`
	ErrorCount   int64         `json:"error_count"`
	SuccessCount int64         `json:"success_count"`
	LastUpdated  time.Time     `json:"last_updated"`
}

// Manager handles license operations with enhanced logging, caching, and security
type Manager struct {
	config          GoogleSheetsConfig
	licenseFile     string
	sheetsService   *sheets.Service
	cache           *LicenseCache
	security        *SecurityManager
	performanceData map[string]*PerformanceMetrics
	perfMutex       sync.RWMutex
	// Add validation state tracking
	lastValidationResult *ValidationResult
	lastValidationTime   time.Time
	validationMutex      sync.RWMutex
	// OpenTelemetry metrics
	metrics         *LicenseMetrics
	// Secure credentials management
	credentialsManager   *security.SecureCredentialsManager
	secureMode          bool
	// Device fingerprinting for scratch card system
	fingerprintManager   *security.FingerprintManager
}

// ValidationResult holds cached validation results
type ValidationResult struct {
	IsValid     bool
	Error       error
	ErrorType   string // "expired", "network_error", etc.
	CachedUntil time.Time
	RetryAfter  time.Duration
}

// RenewalInfo contains information about license renewal requirements
type RenewalInfo struct {
	DaysLeft     int    `json:"days_left"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	NeedsRenewal bool   `json:"needs_renewal"`
	IsExpired    bool   `json:"is_expired"`
}

// ManagerInterface defines the interface for license managers to enable proper testing and mocking
type ManagerInterface interface {
	// Core license operations
	GetLicenseStatus() (*LicenseInfo, string, error)
	ActivateLicense(key string) error
	ValidateLicense() (bool, error)
	
	// License stacking operations
	CheckExistingLicense() (*ExistingLicenseInfo, error)
	GetLicenseInfo() (*LicenseInfo, error)
	
	// Path operations
	GetLicensePath() string
}

// Helper function for min operation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ValidateScratchCardFormat validates the scratch card license key format
func ValidateScratchCardFormat(licenseKey string) error {
	// Remove any dashes for validation
	cleanKey := strings.ReplaceAll(licenseKey, "-", "")
	
	// Check if it starts with ISX
	if !strings.HasPrefix(cleanKey, "ISX") {
		return fmt.Errorf("license key must start with 'ISX'")
	}
	
	// Check length - should be 15 characters (ISX + 12 chars) without dashes
	if len(cleanKey) != 15 {
		return fmt.Errorf("license key must be 15 characters long (ISX + 12 characters)")
	}
	
	// Check that all characters after ISX are alphanumeric
	suffix := cleanKey[3:]
	for _, char := range suffix {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("license key must contain only uppercase letters and numbers")
		}
	}
	
	return nil
}

// NormalizeScratchCardKey normalizes scratch card key to standard format
func NormalizeScratchCardKey(licenseKey string) string {
	// Remove all dashes and spaces
	cleanKey := strings.ReplaceAll(strings.ReplaceAll(licenseKey, "-", ""), " ", "")
	cleanKey = strings.ToUpper(cleanKey)
	
	// If it's in the format ISXXX12CHARS, return as-is
	if len(cleanKey) == 15 && strings.HasPrefix(cleanKey, "ISX") {
		return cleanKey
	}
	
	return cleanKey
}

// FormatScratchCardKeyWithDashes formats key with dashes for display
func FormatScratchCardKeyWithDashes(licenseKey string) string {
	cleanKey := NormalizeScratchCardKey(licenseKey)
	
	if len(cleanKey) != 15 {
		return cleanKey // Return as-is if invalid length
	}
	
	// Format as ISX-XXXX-XXXX-XXXX
	return fmt.Sprintf("%s-%s-%s-%s", 
		cleanKey[:3],   // ISX
		cleanKey[3:7],  // Next 4 chars
		cleanKey[7:11], // Next 4 chars  
		cleanKey[11:15], // Last 4 chars
	)
}


// getBuiltInConfig returns the Google Sheets configuration for secure credential mode
// This function now uses encrypted credentials instead of hardcoded ones
func getBuiltInConfig() GoogleSheetsConfig {
	// Sheet configuration (non-sensitive)
	sheetID := "1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc"
	sheetName := "Licenses"

	config := GoogleSheetsConfig{
		SheetID:           sheetID,
		SheetName:         sheetName,
		UseServiceAccount: true,
		// Credentials are loaded from embedded encrypted data only
	}
	
	return config
}

// NewManager creates a new license manager with enhanced capabilities and secure credentials
func NewManager(licenseFile string) (*Manager, error) {
	// Use centralized path management system
	licensePath, err := config.GetLicensePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get license path: %v", err)
	}
	
	// Enhanced logging for license path resolution
	ctx := context.Background()
	logger := slog.Default()
	if logger != nil {
		// Get absolute paths for clarity
		absRequestedPath, _ := filepath.Abs(licenseFile)
		absResolvedPath, _ := filepath.Abs(licensePath)
		workingDir, _ := os.Getwd()
		execPath, _ := os.Executable()
		
		logger.Info("License manager initialization - Path Resolution Details",
			slog.String("requested_path", licenseFile),
			slog.String("requested_abs_path", absRequestedPath),
			slog.String("resolved_path", licensePath),
			slog.String("resolved_abs_path", absResolvedPath),
			slog.String("working_directory", workingDir),
			slog.String("executable_path", execPath),
			slog.Bool("file_exists", config.FileExists(licensePath)),
			slog.String("config_method", "config.GetLicensePath()"),
		)
	}
	
	// Use built-in configuration (self-contained mode)
	sheetsConfig := getBuiltInConfig()


	// Initialize cache (5 minute TTL as requested, max 1000 entries)
	cache := NewLicenseCache(5*time.Minute, 1000)

	// Initialize security manager (max 5 attempts, 15 minute block, 5 minute window)
	securityMgr := NewSecurityManager(5, 15*time.Minute, 5*time.Minute)

	// Initialize secure credentials manager
	credentialsManager, err := security.NewSecureCredentialsManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secure credentials manager: %v", err)
	}

	// Initialize device fingerprint manager
	fingerprintManager := security.NewFingerprintManager()

	manager := &Manager{
		config:             sheetsConfig,
		licenseFile:        licensePath, // Use the resolved path from centralized system
		cache:              cache,
		security:           securityMgr,
		performanceData:    make(map[string]*PerformanceMetrics),
		credentialsManager: credentialsManager,
		secureMode:         true,
		fingerprintManager: fingerprintManager,
	}

	// Log manager initialization using slog with path information
	manager.logInfo(ctx, "manager_initialization", "License manager initialized successfully with secure credentials",
		slog.String("license_path", licensePath),
		slog.Bool("license_exists", config.FileExists(licensePath)),
		slog.String("cache_ttl", "5m"),
		slog.Int("cache_max_size", 1000),
		slog.Int("security_max_attempts", 5),
		slog.String("security_block_duration", "15m"),
		slog.Bool("secure_mode", true),
	)

	// Skip complex security validation for simplified Google Sheets access
	manager.logInfo(ctx, "security_initialization", "Security manager initialized with encrypted credentials")

	// Initialize Google Sheets service using ONLY embedded encrypted credentials
	if sheetsConfig.UseServiceAccount {
		// Get decrypted credentials from embedded encrypted data only
		credentialsJSON, err := credentialsManager.GetCredentials(ctx)
		if err != nil {
			manager.logError(ctx, "sheets_initialization", "Failed to get embedded credentials",
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to get embedded credentials: %v", err)
		}
		
		// Validate credentials are not empty
		if len(credentialsJSON) == 0 {
			manager.logError(ctx, "sheets_initialization", "Embedded credentials are empty")
			return nil, fmt.Errorf("embedded credentials are empty - ensure build includes encrypted credentials")
		}

		// Create Google Sheets service with embedded credentials only
		credentialsOption := option.WithCredentialsJSON(credentialsJSON)
		sheetsService, err := sheets.NewService(ctx, credentialsOption)
		if err != nil {
			manager.logError(ctx, "sheets_initialization", "Failed to create Google Sheets service",
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to create sheets service with embedded credentials: %v", err)
		}
		
		// Store service
		manager.sheetsService = sheetsService

		manager.logInfo(ctx, "sheets_initialization", "Google Sheets service initialized with embedded encrypted credentials only")
	}

	return manager, nil
}

// SetMetrics sets the OpenTelemetry metrics for the manager
func (m *Manager) SetMetrics(metrics *LicenseMetrics) {
	m.metrics = metrics
}

// NewManagerWithConfig creates a new license manager with custom configuration (for backward compatibility)
func NewManagerWithConfig(configFile, licenseFile string) (*Manager, error) {
	config, err := loadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}


	manager := &Manager{
		config:      config,
		licenseFile: licenseFile,
	}

	// NewManagerWithConfig is deprecated - use NewManager instead for embedded credential support
	return nil, fmt.Errorf("NewManagerWithConfig is deprecated - use NewManager instead for embedded credential support")

	return manager, nil
}

// GenerateLicense creates a new license key
func (m *Manager) GenerateLicense(userEmail string, duration string) (string, error) {
	// Generate random license key
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	licenseKey := base64.URLEncoding.EncodeToString(bytes)
	licenseKey = strings.ReplaceAll(licenseKey, "=", "")

	// Add prefix based on duration
	prefix := "ISX"
	switch duration {
	case "1m":
		prefix = "ISX1M"
	case "3m":
		prefix = "ISX3M"
	case "6m":
		prefix = "ISX6M"
	case "1y":
		prefix = "ISX1Y"
	}

	licenseKey = fmt.Sprintf("%s%s", prefix, licenseKey)

	// Calculate expiry date - expires at 12am next day after standard period
	var standardExpiry time.Time
	switch duration {
	case "1m":
		standardExpiry = time.Now().AddDate(0, 1, 0)
	case "3m":
		standardExpiry = time.Now().AddDate(0, 3, 0)
	case "6m":
		standardExpiry = time.Now().AddDate(0, 6, 0)
	case "1y":
		standardExpiry = time.Now().AddDate(1, 0, 0)
	default:
		standardExpiry = time.Now().AddDate(0, 1, 0)
	}
	// Set expiry to 12:00 AM next day after standard expiry
	expiryDate := time.Date(standardExpiry.Year(), standardExpiry.Month(), standardExpiry.Day()+1, 0, 0, 0, 0, standardExpiry.Location())

	// Create license info
	license := LicenseInfo{
		LicenseKey:  licenseKey,
		UserEmail:   userEmail,
		ExpiryDate:  expiryDate,
		Duration:    duration,
		IssuedDate:  time.Now(),
		Status:      "issued",
		LastChecked: time.Now(),
	}

	// Save to Google Sheets
	if err := m.saveLicenseToSheets(license); err != nil {
		return "", fmt.Errorf("failed to save license: %v", err)
	}

	return licenseKey, nil
}

// ActivateLicense activates a license with enhanced tracking and security
func (m *Manager) ActivateLicense(licenseKey string) error {
	return m.ActivateLicenseWithContext(context.Background(), licenseKey)
}

// ActivateLicenseWithContext activates a license with context and enhanced observability
func (m *Manager) ActivateLicenseWithContext(ctx context.Context, licenseKey string) error {
	return m.TraceActivation(ctx, licenseKey, func() error {
		return m.TrackOperation("license_activation", func() error {
			return m.performActivation(licenseKey)
		})
	})
}

// performActivation contains the actual license activation logic for scratch card system with enhanced security
func (m *Manager) performActivation(licenseKey string) error {
	ctx := context.Background()
	
	// Enhanced input validation and sanitization
	inputValidator := security.NewInputValidator(nil)
	licenseValidation := inputValidator.ValidateLicenseKey(ctx, licenseKey)
	
	if !licenseValidation.IsValid {
		m.logWarn(ctx, "license_activation", "License key validation failed",
			slog.Any("validation_errors", licenseValidation.Errors),
			slog.Int("risk_score", licenseValidation.RiskScore),
			slog.Any("threat_types", licenseValidation.ThreatTypes),
		)
		return fmt.Errorf("invalid license key: %v", licenseValidation.Errors)
	}
	
	// Use sanitized license key
	normalizedKey := licenseValidation.SanitizedValue
	
	// Validate input
	if normalizedKey == "" {
		return fmt.Errorf("license key cannot be empty")
	}

	// Generate device fingerprint early for security checking
	deviceFingerprint, err := m.fingerprintManager.GenerateFingerprint()
	if err != nil {
		m.logWarn(ctx, "license_activation", "Failed to generate device fingerprint, using fallback",
			slog.String("error", err.Error()),
		)
		// Continue with activation but use a fallback fingerprint
		deviceFingerprint = &security.DeviceFingerprint{
			Fingerprint: "fallback-fingerprint",
			Hostname:    "unknown",
			OS:          "unknown",
		}
	}

	// Get client IP (if available from context)
	clientIP := "unknown"
	// Note: In a real HTTP context, this would be extracted from the request

	// Enhanced security checks with multiple identifiers
	identifier := normalizedKey[:min(8, len(normalizedKey))]
	
	// Check enhanced rate limiting
	if m.security != nil {
		if blocked, reason, remaining := m.security.IsBlockedEnhanced(identifier, deviceFingerprint.Fingerprint, clientIP); blocked {
			m.logWarn(ctx, "license_activation", "Activation blocked by enhanced security",
				slog.String("license_key_prefix", identifier),
				slog.String("block_reason", reason),
				slog.Duration("remaining_time", remaining),
				slog.String("client_ip", clientIP),
				slog.String("device_fingerprint", deviceFingerprint.Fingerprint),
			)
			
			// Provide user-friendly error messages based on block reason
			switch reason {
			case "honeypot_detection":
				return fmt.Errorf("security violation detected - access denied")
			case "permanent_ban":
				return fmt.Errorf("this device has been permanently blocked due to suspicious activity")
			case "suspicious_activity":
				return fmt.Errorf("temporary security block due to suspicious activity - please try again later")
			case "device_blocked":
				return fmt.Errorf("this device is temporarily blocked - please try again in %v", remaining)
			default:
				return fmt.Errorf("too many failed attempts - please try again later")
			}
		}
	}

	// Log activation attempt with enhanced security context
	m.logInfo(ctx, "license_activation", "Starting enhanced scratch card license activation",
		slog.String("license_key_prefix", identifier),
		slog.String("formatted_key", FormatScratchCardKeyWithDashes(normalizedKey)),
		slog.String("device_fingerprint", deviceFingerprint.Fingerprint),
		slog.String("client_ip", clientIP),
		slog.Int("validation_risk_score", licenseValidation.RiskScore),
	)

	// Call secure Apps Script for activation
	licenseInfo, err := m.callAppsScriptActivation(normalizedKey, deviceFingerprint, clientIP)
	if err != nil {
		// Enhanced error recording with context
		errorType := "unknown"
		if strings.Contains(err.Error(), "timeout") {
			errorType = "timeout"
		} else if strings.Contains(err.Error(), "network") {
			errorType = "network_error"
		} else if strings.Contains(err.Error(), "reactivation limit exceeded") {
			errorType = "reactivation_limit_exceeded"
		} else if strings.Contains(err.Error(), "already activated on different device") {
			errorType = "already_activated_different_device"
		} else if strings.Contains(err.Error(), "already activated") {
			errorType = "already_activated"
		} else if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid") {
			errorType = "not_found"
		} else if strings.Contains(err.Error(), "validation") {
			errorType = "validation_failed"
		}

		// Record enhanced failed attempt
		if m.security != nil {
			m.security.RecordAttemptEnhanced(identifier, deviceFingerprint.Fingerprint, clientIP, normalizedKey, "ISX-Pulse-Client", false, errorType)
		}

		// Provide more specific error context
		switch errorType {
		case "timeout":
			return fmt.Errorf("connection timeout while accessing license activation service - please check your internet connection")
		case "network_error":
			return fmt.Errorf("network connection error - please check your internet connection and firewall settings")
		case "reactivation_limit_exceeded":
			return licenseErrors.ErrReactivationLimitExceeded
		case "already_activated_different_device":
			return licenseErrors.ErrAlreadyActivatedOnDevice
		case "already_activated":
			return fmt.Errorf("license has already been activated on another device")
		case "not_found":
			return fmt.Errorf("invalid license key - license not found in our system")
		case "validation_failed":
			return fmt.Errorf("license key validation failed - please check the format and try again")
		default:
			return fmt.Errorf("license activation failed: %v", err)
		}
	}

	// Check if already activated license has expired
	if !licenseInfo.ExpiryDate.IsZero() && time.Now().After(licenseInfo.ExpiryDate) {
		if m.security != nil {
			m.security.RecordAttemptEnhanced(identifier, deviceFingerprint.Fingerprint, clientIP, normalizedKey, "ISX-Pulse-Client", false, "expired")
		}
		return fmt.Errorf("license has expired on %s", licenseInfo.ExpiryDate.Format("2006-01-02"))
	}

	// Check for existing valid license and stack if applicable
	existingLicense, err := m.loadLicenseLocal()
	if err == nil && time.Now().Before(existingLicense.ExpiryDate) {
		// We have a valid existing license - stack the new one
		m.logInfo(ctx, "license_stacking", "Stacking new license with existing valid license",
			slog.String("existing_key_masked", MaskLicenseKey(existingLicense.LicenseKey)),
			slog.String("new_key_masked", MaskLicenseKey(normalizedKey)),
			slog.String("existing_expiry", existingLicense.ExpiryDate.Format("2006-01-02")),
			slog.String("new_duration", licenseInfo.Duration),
		)
		
		// Parse the duration from the new license
		additionalDuration := parseLicenseDuration(licenseInfo.Duration)
		
		// Calculate new expiry by adding duration to existing expiry
		newExpiry := existingLicense.ExpiryDate.Add(additionalDuration)
		
		// Update license info with stacked values
		licenseInfo.ExpiryDate = newExpiry
		licenseInfo.Status = "Stacked"
		// Combine activation IDs to maintain history
		licenseInfo.ActivationID = fmt.Sprintf("%s+%s", existingLicense.ActivationID, licenseInfo.ActivationID)
		// Preserve the original license key from existing license
		licenseInfo.LicenseKey = existingLicense.LicenseKey + "+" + normalizedKey
		
		// Audit the stacking operation
		m.auditLicenseChange(ctx, "stacked", existingLicense, licenseInfo, deviceFingerprint.Fingerprint)
		
		m.logInfo(ctx, "license_stacked", "License successfully stacked",
			slog.String("combined_keys", MaskLicenseKey(licenseInfo.LicenseKey)),
			slog.String("new_expiry", newExpiry.Format("2006-01-02")),
			slog.Int("total_days", int(time.Until(newExpiry).Hours()/24)),
		)
	} else if err == nil && time.Now().After(existingLicense.ExpiryDate) {
		// Existing license is expired - replace it
		m.logInfo(ctx, "license_replacement", "Replacing expired license with new one",
			slog.String("expired_key_masked", MaskLicenseKey(existingLicense.LicenseKey)),
			slog.String("new_key_masked", MaskLicenseKey(normalizedKey)),
			slog.String("expired_date", existingLicense.ExpiryDate.Format("2006-01-02")),
		)
		
		// Audit the replacement
		m.auditLicenseChange(ctx, "replaced_expired", existingLicense, licenseInfo, deviceFingerprint.Fingerprint)
		
		licenseInfo.Status = "Activated"
	} else {
		// No existing license - normal activation
		licenseInfo.Status = "Activated"
		
		// Audit new activation
		m.auditLicenseChange(ctx, "new_activation", LicenseInfo{}, licenseInfo, deviceFingerprint.Fingerprint)
	}

	// Store device fingerprint
	licenseInfo.DeviceFingerprint = deviceFingerprint.Fingerprint
	licenseInfo.LastChecked = time.Now()

	// Save license locally
	if err := m.saveLicenseLocal(licenseInfo); err != nil {
		return fmt.Errorf("failed to save license locally: %v", err)
	}

	// Invalidate cache to ensure fresh data on next validation
	if m.cache != nil {
		m.cache.Invalidate(normalizedKey)
	}

	// Record enhanced successful attempt
	if m.security != nil {
		m.security.RecordAttemptEnhanced(identifier, deviceFingerprint.Fingerprint, clientIP, normalizedKey, "ISX-Pulse-Client", true, "")
	}
	
	// Handle reactivation success scenario - return appropriate error for service layer handling
	if licenseInfo.Status == "reactivated" {
		m.logInfo(ctx, "license_reactivation_success", "License reactivation completed successfully",
			slog.String("license_key_prefix", identifier),
			slog.String("device_fingerprint", deviceFingerprint.Fingerprint),
			slog.String("activation_id", licenseInfo.ActivationID),
		)
		// Return special reactivation "error" that will be handled as success by service layer
		return licenseErrors.ErrLicenseReactivated
	}

	// Log successful activation with enhanced security context
	daysLeft := int(time.Until(licenseInfo.ExpiryDate).Hours() / 24)
	m.logLicenseAction(ctx, slog.LevelInfo, "license_activation", "Enhanced scratch card license activated successfully", 
		identifier, licenseInfo.UserEmail,
		slog.String("expiry_date", licenseInfo.ExpiryDate.Format("2006-01-02")),
		slog.String("duration", licenseInfo.Duration),
		slog.String("activation_id", licenseInfo.ActivationID),
		slog.String("device_fingerprint", deviceFingerprint.Fingerprint[:min(16, len(deviceFingerprint.Fingerprint))]),
		slog.String("client_ip", clientIP),
		slog.Int("days_left", daysLeft),
		slog.Int("validation_risk_score", licenseValidation.RiskScore),
		slog.String("security_level", "enhanced"),
	)

	return nil
}

// ValidateLicense checks if current license is valid with enhanced tracking
func (m *Manager) ValidateLicense() (bool, error) {
	return m.ValidateLicenseWithContext(context.Background())
}

// ValidateLicenseWithContext checks if current license is valid with context and enhanced observability
func (m *Manager) ValidateLicenseWithContext(ctx context.Context) (bool, error) {
	// Check if we have a recent cached result
	m.validationMutex.RLock()
	if m.lastValidationResult != nil && time.Now().Before(m.lastValidationResult.CachedUntil) {
		result := m.lastValidationResult
		m.validationMutex.RUnlock()
		
		// Record cache hit metric
		if m.metrics != nil {
			m.metrics.ValidationCacheHits.Add(ctx, 1, metric.WithAttributes(
				attribute.String("component", "license_manager"),
				attribute.String("cache_result", "hit"),
			))
		}
		
		return result.IsValid, result.Error
	}
	m.validationMutex.RUnlock()

	// Record cache miss metric
	if m.metrics != nil {
		m.metrics.ValidationCacheMisses.Add(ctx, 1, metric.WithAttributes(
			attribute.String("component", "license_manager"),
			attribute.String("cache_result", "miss"),
		))
	}

	// Perform actual validation with tracing
	return m.TraceValidation(ctx, func() (bool, error) {
		var valid bool
		var err error

		trackErr := m.TrackOperation("license_validation_complete", func() error {
			valid, err = m.performValidation()

			// Cache the result with appropriate duration
			m.cacheValidationResult(valid, err)

			if !valid {
				return err
			}
			return nil
		})

		if trackErr != nil {
			return false, trackErr
		}

		return valid, err
	})
}

// cacheValidationResult caches validation results with appropriate durations
func (m *Manager) cacheValidationResult(isValid bool, err error) {
	m.validationMutex.Lock()
	defer m.validationMutex.Unlock()

	result := &ValidationResult{
		IsValid: isValid,
		Error:   err,
	}

	if isValid {
		// Cache successful validations for 5 minutes as requested
		result.CachedUntil = time.Now().Add(5 * time.Minute)
	} else if err != nil {
		// Determine error type and cache duration
		errorMsg := err.Error()

		if strings.Contains(errorMsg, "expired") {
			result.ErrorType = "expired"
			// Cache expiry errors for 1 hour
			result.CachedUntil = time.Now().Add(1 * time.Hour)
			result.RetryAfter = 5 * time.Minute
		} else {
			result.ErrorType = "network_error"
			// Cache network errors for 2 minutes
			result.CachedUntil = time.Now().Add(2 * time.Minute)
			result.RetryAfter = 30 * time.Second
		}
	}

	m.lastValidationResult = result
	m.lastValidationTime = time.Now()
}

// GetValidationState returns the current validation state for better user feedback
func (m *Manager) GetValidationState() (*ValidationResult, error) {
	m.validationMutex.RLock()
	defer m.validationMutex.RUnlock()

	if m.lastValidationResult == nil {
		return nil, fmt.Errorf("no validation performed yet")
	}

	// Return a copy to avoid concurrent access issues
	result := *m.lastValidationResult
	return &result, nil
}

// performValidation contains the actual validation logic - simplified to only check expiry
func (m *Manager) performValidation() (bool, error) {
	// Load local license
	license, err := m.loadLicenseLocal()
	if err != nil {
		return false, fmt.Errorf("no local license found: %v", err)
	}

	// Check expiry - this is the only validation we do now
	if time.Now().After(license.ExpiryDate) {
		license.Status = "expired"
		m.saveLicenseLocal(license)

		m.logLicenseAction(context.Background(), slog.LevelWarn, "license_validation", "License expired",
			license.LicenseKey, license.UserEmail,
			slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
			slog.String("expiry_date", license.ExpiryDate.Format("2006-01-02")),
		)

		return false, fmt.Errorf("license expired on %s", license.ExpiryDate.Format("2006-01-02"))
	}

	// Validate device fingerprint (with optional enforcement)
	if license.DeviceFingerprint != "" {
		currentFingerprint, err := m.fingerprintManager.GenerateFingerprint()
		if err != nil {
			m.logWarn(context.Background(), "license_validation", "Failed to generate current device fingerprint",
				slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
				slog.String("error", err.Error()),
			)
			// Don't fail validation if fingerprint generation fails - allow offline usage
		} else if currentFingerprint.Fingerprint != license.DeviceFingerprint {
			m.logWarn(context.Background(), "license_validation", "Device fingerprint mismatch detected",
				slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
				slog.String("stored_fingerprint", license.DeviceFingerprint[:min(16, len(license.DeviceFingerprint))]),
				slog.String("current_fingerprint", currentFingerprint.Fingerprint[:min(16, len(currentFingerprint.Fingerprint))]),
			)
			// For now, log the mismatch but don't fail validation
			// In the future, this could be configurable for stricter security
		}
	}

	// Periodic validation with Apps Script (every 6 hours for better security)
	if time.Since(license.LastChecked) > 6*time.Hour {
		if err := m.validateWithAppsScript(license); err != nil {
			// For better user experience, don't fail immediately on network issues
			// Log the error but allow offline usage for up to 48 hours total
			if time.Since(license.LastChecked) > 48*time.Hour {
				m.logLicenseAction(context.Background(), slog.LevelError, "license_validation", "Remote validation failed and grace period expired",
					license.LicenseKey, license.UserEmail,
					slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
					slog.String("error", err.Error()),
				)
				return false, fmt.Errorf("remote validation failed and offline grace period expired: %v", err)
			}
			// Just log the warning but continue with local validation
			m.logLicenseAction(context.Background(), slog.LevelWarn, "license_validation", "Remote validation failed, using local cache",
				license.LicenseKey, license.UserEmail,
				slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
				slog.String("error", err.Error()),
			)
		}
	}

	return true, nil
}

// TransferLicense transfers a license - simplified since no machine binding
func (m *Manager) TransferLicense(licenseKey string, forceTransfer bool) error {
	// Since we don't have machine binding anymore, transfer is essentially just activation
	return m.ActivateLicense(licenseKey)
}


// UpdateLastConnected updates the last connected time in both local storage and Google Sheets
func (m *Manager) UpdateLastConnected() error {
	// Load current license
	license, err := m.loadLicenseLocal()
	if err != nil {
		return fmt.Errorf("no local license found: %v", err)
	}

	// Update last checked time
	license.LastChecked = time.Now()

	// Save locally
	if err := m.saveLicenseLocal(license); err != nil {
		return fmt.Errorf("failed to save license locally: %v", err)
	}

	// Update Google Sheets with expire status
	if err := m.updateLicenseInSheets(license); err != nil {
		return fmt.Errorf("failed to update last connected time in sheets: %v", err)
	}

	return nil
}

// GetLicenseInfo returns current license information
func (m *Manager) GetLicenseInfo() (*LicenseInfo, error) {
	license, err := m.loadLicenseLocal()
	if err != nil {
		return nil, err
	}
	return &license, nil
}

// ExistingLicenseInfo provides information about an existing license for pre-activation checks
type ExistingLicenseInfo struct {
	HasLicense     bool      `json:"has_license"`
	DaysRemaining  int       `json:"days_remaining"`
	ExpiryDate     time.Time `json:"expiry_date,omitempty"`
	LicenseKey     string    `json:"license_key_masked"` // Masked format: ISX-XXXX-****-****
	Status         string    `json:"status"`
	IsExpired      bool      `json:"is_expired"`
}

// CheckExistingLicense checks if there's an existing license and returns its details
func (m *Manager) CheckExistingLicense() (*ExistingLicenseInfo, error) {
	ctx := context.Background()
	
	// Try to load existing license
	license, err := m.loadLicenseLocal()
	if err != nil {
		// No existing license
		m.logDebug(ctx, "check_existing_license", "No existing license found",
			slog.String("error", err.Error()),
		)
		return &ExistingLicenseInfo{
			HasLicense: false,
			Status:     "not_activated",
		}, nil
	}
	
	// Calculate days remaining
	now := time.Now()
	daysRemaining := 0
	isExpired := false
	
	if now.Before(license.ExpiryDate) {
		daysRemaining = int(time.Until(license.ExpiryDate).Hours() / 24)
	} else {
		isExpired = true
	}
	
	// Mask the license key for security (show only first segment)
	maskedKey := MaskLicenseKey(license.LicenseKey)
	
	m.logInfo(ctx, "check_existing_license", "Existing license found",
		slog.String("license_key_masked", maskedKey),
		slog.Int("days_remaining", daysRemaining),
		slog.Bool("is_expired", isExpired),
		slog.String("status", license.Status),
	)
	
	return &ExistingLicenseInfo{
		HasLicense:    true,
		DaysRemaining: daysRemaining,
		ExpiryDate:    license.ExpiryDate,
		LicenseKey:    maskedKey,
		Status:        license.Status,
		IsExpired:     isExpired,
	}, nil
}

// MaskLicenseKey masks a license key for display (ISX-XXXX-****-****)
func MaskLicenseKey(key string) string {
	if len(key) < 8 {
		return "****"
	}
	
	// Handle stacked licenses (key1+key2)
	if strings.Contains(key, "+") {
		parts := strings.Split(key, "+")
		masked := ""
		for i, part := range parts {
			if i > 0 {
				masked += "+"
			}
			masked += maskSingleKey(part)
		}
		return masked
	}
	
	return maskSingleKey(key)
}

// maskSingleKey masks a single license key
func maskSingleKey(key string) string {
	// Handle both formats: ISX-XXXX-XXXX-XXXX and ISX1MXXXXXX
	if strings.Contains(key, "-") {
		parts := strings.Split(key, "-")
		if len(parts) >= 2 {
			// Keep first two parts, mask the rest
			masked := parts[0] + "-" + parts[1]
			for i := 2; i < len(parts); i++ {
				masked += "-****"
			}
			return masked
		}
	}
	
	// For non-dash format, show first 8 chars
	if len(key) > 8 {
		return key[:8] + "****"
	}
	
	return key
}

// parseLicenseDuration converts license duration string to time.Duration
func parseLicenseDuration(duration string) time.Duration {
	// Handle various duration formats: "30 days", "1 month", "3 months", "6 months", "1 year"
	duration = strings.ToLower(strings.TrimSpace(duration))
	
	// Extract number and unit
	var value int
	var unit string
	
	// Try to parse "X days", "X months", "X month", "X year", etc.
	if _, err := fmt.Sscanf(duration, "%d %s", &value, &unit); err != nil {
		// Try without space: "30days", "1month", etc.
		if _, err := fmt.Sscanf(duration, "%d%s", &value, &unit); err != nil {
			// Default to 30 days if can't parse
			return 30 * 24 * time.Hour
		}
	}
	
	// Normalize unit
	unit = strings.TrimSuffix(unit, "s") // Remove plural 's'
	
	switch unit {
	case "day":
		return time.Duration(value) * 24 * time.Hour
	case "month":
		// Approximate: 30 days per month
		return time.Duration(value) * 30 * 24 * time.Hour
	case "year":
		// Approximate: 365 days per year
		return time.Duration(value) * 365 * 24 * time.Hour
	default:
		// Default to days if unit not recognized
		return time.Duration(value) * 24 * time.Hour
	}
}

// LicenseAudit represents a license change audit entry
type LicenseAudit struct {
	Timestamp      time.Time `json:"timestamp"`
	Action         string    `json:"action"` // "activated", "stacked", "replaced_expired", "new_activation"
	PreviousKey    string    `json:"previous_key,omitempty"`
	NewKey         string    `json:"new_key"`
	PreviousExpiry time.Time `json:"previous_expiry,omitempty"`
	NewExpiry      time.Time `json:"new_expiry"`
	DeviceID       string    `json:"device_id"`
	TraceID        string    `json:"trace_id"`
	UserEmail      string    `json:"user_email,omitempty"`
}

// auditLicenseChange logs license changes to audit file
func (m *Manager) auditLicenseChange(ctx context.Context, action string, previousLicense, newLicense LicenseInfo, deviceID string) {
	audit := LicenseAudit{
		Timestamp:      time.Now(),
		Action:         action,
		NewKey:         MaskLicenseKey(newLicense.LicenseKey),
		NewExpiry:      newLicense.ExpiryDate,
		DeviceID:       deviceID[:min(16, len(deviceID))], // Truncate device ID for privacy
		TraceID:        newLicense.ActivationID,
		UserEmail:      newLicense.UserEmail,
	}
	
	// Add previous license info if applicable
	if previousLicense.LicenseKey != "" {
		audit.PreviousKey = MaskLicenseKey(previousLicense.LicenseKey)
		audit.PreviousExpiry = previousLicense.ExpiryDate
	}
	
	// Log using structured logging per CLAUDE.md
	m.logInfo(ctx, "license_audit", "License change audited",
		slog.String("action", action),
		slog.String("new_key", audit.NewKey),
		slog.String("previous_key", audit.PreviousKey),
		slog.Time("new_expiry", audit.NewExpiry),
		slog.Time("previous_expiry", audit.PreviousExpiry),
		slog.String("device_id", audit.DeviceID),
		slog.String("trace_id", audit.TraceID),
	)
	
	// Also write to dedicated audit file
	auditFile := filepath.Join("logs", "license_audit.json")
	if err := m.writeAuditToFile(audit, auditFile); err != nil {
		m.logError(ctx, "audit_write", "Failed to write audit to file",
			slog.String("error", err.Error()),
			slog.String("file", auditFile),
		)
	}
}

// writeAuditToFile appends audit entry to JSON file
func (m *Manager) writeAuditToFile(audit LicenseAudit, filename string) error {
	// Ensure logs directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}
	
	// Marshal audit entry to JSON
	data, err := json.Marshal(audit)
	if err != nil {
		return fmt.Errorf("marshal audit: %w", err)
	}
	
	// Append to file with newline
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer file.Close()
	
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write audit: %w", err)
	}
	
	return nil
}













// loadConfig loads Google Sheets configuration
func loadConfig(configFile string) (GoogleSheetsConfig, error) {
	var config GoogleSheetsConfig

	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	return config, err
}

// saveLicenseLocal saves license to local file
func (m *Manager) saveLicenseLocal(license LicenseInfo) error {
	// Enhanced logging for license save operation
	ctx := context.Background()
	absPath, _ := filepath.Abs(m.licenseFile)
	dir := filepath.Dir(absPath)
	dirInfo, dirErr := os.Stat(dir)
	var dirDetails string
	if dirErr == nil {
		dirDetails = fmt.Sprintf("exists=%t, writable=%t", dirInfo.IsDir(), isWritable(dir))
	} else {
		dirDetails = fmt.Sprintf("error=%v", dirErr)
	}
	
	m.logInfo(ctx, "license_save_attempt", "Attempting to save license file",
		slog.String("configured_path", m.licenseFile),
		slog.String("absolute_path", absPath),
		slog.String("directory", dir),
		slog.String("directory_details", dirDetails),
		slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
		slog.String("working_dir", m.getWorkingDir()),
	)
	
	data, err := json.MarshalIndent(license, "", "  ")
	if err != nil {
		m.logError(ctx, "license_save", "Failed to marshal license data",
			slog.String("error", err.Error()),
		)
		return err
	}

	err = os.WriteFile(m.licenseFile, data, 0600)
	if err != nil {
		m.logError(ctx, "license_save", "Failed to write license file",
			slog.String("path", m.licenseFile),
			slog.String("error", err.Error()),
		)
		return err
	}
	
	m.logInfo(ctx, "license_save", "License saved successfully",
		slog.String("path", m.licenseFile),
		slog.Int("size_bytes", len(data)),
	)
	
	return nil
}

// loadLicenseLocal loads license from local file
func (m *Manager) loadLicenseLocal() (LicenseInfo, error) {
	var license LicenseInfo
	
	// Log the load operation with full path
	ctx := context.Background()
	
	// Enhanced logging to show exact path being checked
	absPath, _ := filepath.Abs(m.licenseFile)
	fileInfo, statErr := os.Stat(m.licenseFile)
	var fileDetails string
	if statErr == nil {
		fileDetails = fmt.Sprintf("size=%d, mode=%s, modtime=%s", 
			fileInfo.Size(), fileInfo.Mode(), fileInfo.ModTime().Format(time.RFC3339))
	} else {
		fileDetails = fmt.Sprintf("stat_error=%v", statErr)
	}
	
	m.logInfo(ctx, "license_load_attempt", "Attempting to load license file",
		slog.String("configured_path", m.licenseFile),
		slog.String("absolute_path", absPath),
		slog.String("working_dir", m.getWorkingDir()),
		slog.Bool("file_exists", config.FileExists(m.licenseFile)),
		slog.String("file_details", fileDetails),
	)

	data, err := os.ReadFile(m.licenseFile)
	if err != nil {
		m.logDebug(ctx, "license_load", "Failed to read license file",
			slog.String("path", m.licenseFile),
			slog.String("error", err.Error()),
		)
		return license, err
	}

	err = json.Unmarshal(data, &license)
	if err != nil {
		m.logError(ctx, "license_load", "Failed to unmarshal license data",
			slog.String("path", m.licenseFile),
			slog.String("error", err.Error()),
		)
		return license, err
	}
	
	m.logDebug(ctx, "license_load", "License loaded successfully",
		slog.String("path", m.licenseFile),
		slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
		slog.String("status", license.Status),
	)
	
	return license, nil
}

// saveLicenseToSheets saves license to Google Sheets
func (m *Manager) saveLicenseToSheets(license LicenseInfo) error {
	// Implementation for Google Sheets API
	// This would use the Google Sheets API to append a new row
	// Format: [LicenseKey, UserEmail, ExpiryDate, Duration, MachineID, IssuedDate, Status, LastChecked]

	values := []interface{}{
		license.LicenseKey,
		license.UserEmail,
		license.ExpiryDate.Format("2006-01-02 15:04:05"),
		license.Duration,
		"", // Machine ID removed
		license.IssuedDate.Format("2006-01-02 15:04:05"),
		license.Status,
		license.LastChecked.Format("2006-01-02 15:04:05"),
	}

	url := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s:append?valueInputOption=RAW&key=%s",
		m.config.SheetID, m.config.SheetName, m.config.APIKey)

	payload := map[string]interface{}{
		"values": [][]interface{}{values},
	}

	return m.makeSheetRequest("POST", url, payload)
}

// validateLicenseFromAppsScript validates license via Google Apps Script endpoint
func (m *Manager) validateLicenseFromAppsScript(licenseKey string) (LicenseInfo, error) {
	var license LicenseInfo
	
	ctx := context.Background()
	creds := config.GetCredentials()
	m.logInfo(ctx, "apps_script_validation", "Validating license via Apps Script",
		slog.String("license_key_prefix", licenseKey[:min(8, len(licenseKey))]),
		slog.String("endpoint", creds.AppsScriptURL),
	)

	// Prepare request payload
	requestData := map[string]interface{}{
		"action": "validate",
		"code":   licenseKey,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		m.logError(ctx, "apps_script_validation", "Failed to marshal request data",
			slog.String("error", err.Error()),
		)
		return license, fmt.Errorf("failed to prepare request: %w", err)
	}

	// Create HTTP request  
	req, err := http.NewRequestWithContext(ctx, "POST", creds.AppsScriptURL, bytes.NewBuffer(jsonData))
	if err != nil {
		m.logError(ctx, "apps_script_validation", "Failed to create HTTP request",
			slog.String("error", err.Error()),
		)
		return license, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ISX-Pulse-License-Client/1.0")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send request
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		m.logError(ctx, "apps_script_validation", "HTTP request failed",
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)),
		)
		return license, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logError(ctx, "apps_script_validation", "Failed to read response",
			slog.String("error", err.Error()),
		)
		return license, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		m.logError(ctx, "apps_script_validation", "Apps Script returned error status",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response_body", string(body)),
		)
		return license, fmt.Errorf("Apps Script returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		m.logError(ctx, "apps_script_validation", "Failed to parse response JSON",
			slog.String("error", err.Error()),
			slog.String("response_body", string(body)),
		)
		return license, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for success
	success, ok := response["success"].(bool)
	if !ok || !success {
		errorMsg := "unknown error"
		if msg, exists := response["error"]; exists {
			errorMsg = fmt.Sprintf("%v", msg)
		}
		
		m.logWarn(ctx, "apps_script_validation", "Apps Script validation failed",
			slog.String("error", errorMsg),
			slog.String("license_key_prefix", licenseKey[:min(8, len(licenseKey))]),
		)
		return license, fmt.Errorf("validation failed: %s", errorMsg)
	}

	// Extract license data
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		m.logError(ctx, "apps_script_validation", "Invalid response format - missing data field",
			slog.String("response_body", string(body)),
		)
		return license, fmt.Errorf("invalid response format")
	}

	// Parse license information
	license.LicenseKey = licenseKey
	
	if duration, exists := data["duration"]; exists {
		license.Duration = fmt.Sprintf("%v", duration)
	}
	
	if status, exists := data["status"]; exists {
		license.Status = fmt.Sprintf("%v", status)
	}
	
	// Check for expiry_date first (for compatibility), then expires_at (what Apps Script actually sends)
	if expiryStr, exists := data["expiry_date"]; exists && fmt.Sprintf("%v", expiryStr) != "" {
		if expiryDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", expiryStr)); err == nil {
			license.ExpiryDate = expiryDate
			m.logInfo(ctx, "apps_script_activation", "Parsed expiry date from 'expiry_date' field",
				slog.String("expiry_date", expiryDate.Format("2006-01-02")),
			)
		}
	} else if expiryStr, exists := data["expires_at"]; exists && fmt.Sprintf("%v", expiryStr) != "" {
		// Try parsing ISO format first (what Apps Script sends)
		if expiryDate, err := time.Parse(time.RFC3339, fmt.Sprintf("%v", expiryStr)); err == nil {
			license.ExpiryDate = expiryDate
			m.logInfo(ctx, "apps_script_activation", "Parsed expiry date from 'expires_at' field",
				slog.String("expiry_date", expiryDate.Format("2006-01-02")),
			)
		} else if expiryDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", expiryStr)); err == nil {
			// Fallback to date-only format
			license.ExpiryDate = expiryDate
			m.logInfo(ctx, "apps_script_activation", "Parsed expiry date from 'expires_at' field (date format)",
				slog.String("expiry_date", expiryDate.Format("2006-01-02")),
			)
		}
	}
	
	if issuedStr, exists := data["issued_date"]; exists && fmt.Sprintf("%v", issuedStr) != "" {
		if issuedDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", issuedStr)); err == nil {
			license.IssuedDate = issuedDate
		}
	}
	
	if activationID, exists := data["activation_id"]; exists {
		license.ActivationID = fmt.Sprintf("%v", activationID)
	}
	
	if deviceFingerprint, exists := data["device_fingerprint"]; exists {
		license.DeviceFingerprint = fmt.Sprintf("%v", deviceFingerprint)
	}

	// Set default values
	license.UserEmail = "" // Scratch cards don't have user emails
	license.LastChecked = time.Now()

	duration := time.Since(start)
	m.logInfo(ctx, "apps_script_validation", "License validated successfully via Apps Script",
		slog.String("license_key_prefix", licenseKey[:min(8, len(licenseKey))]),
		slog.String("status", license.Status),
		slog.String("duration", license.Duration),
		slog.String("activation_id", license.ActivationID),
		slog.Duration("request_duration", duration),
	)

	return license, nil
}

// callAppsScriptActivation calls the Apps Script endpoint for license activation with enhanced security
func (m *Manager) callAppsScriptActivation(licenseKey string, deviceFingerprint *security.DeviceFingerprint, clientIP string) (LicenseInfo, error) {
	var license LicenseInfo
	
	// Create context with 10 second timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Input validation first
	inputValidator := security.NewInputValidator(nil)
	
	// Validate license key
	licenseValidation := inputValidator.ValidateLicenseKey(ctx, licenseKey)
	if !licenseValidation.IsValid {
		m.logError(ctx, "apps_script_activation", "License key validation failed",
			slog.Any("validation_errors", licenseValidation.Errors),
			slog.Int("risk_score", licenseValidation.RiskScore),
		)
		return license, fmt.Errorf("invalid license key: %v", licenseValidation.Errors)
	}
	
	// Validate client IP
	if clientIP != "" {
		ipValidation := inputValidator.ValidateIPAddress(ctx, clientIP)
		if !ipValidation.IsValid {
			m.logWarn(ctx, "apps_script_activation", "Client IP validation failed",
				slog.Any("validation_errors", ipValidation.Errors),
				slog.String("client_ip", clientIP),
			)
			// Don't fail activation for IP issues, but log them
		}
	}
	
	// Use sanitized license key
	sanitizedLicenseKey := licenseValidation.SanitizedValue
	
	m.logInfo(ctx, "apps_script_activation", "Calling Apps Script for license activation with enhanced security",
		slog.String("license_key_prefix", sanitizedLicenseKey[:min(8, len(sanitizedLicenseKey))]),
		slog.String("device_fingerprint", deviceFingerprint.Fingerprint),
		slog.String("hostname", deviceFingerprint.Hostname),
		slog.String("client_ip", clientIP),
	)

	// Initialize secure Apps Script client
	certPinner := security.NewCertificatePinner(security.DefaultPinningConfig())
	secureClient := security.NewSecureAppsScriptClient(nil, certPinner)
	
	// Prepare request payload
	requestPayload := map[string]interface{}{
		"action": "activate",
		"code":   sanitizedLicenseKey,
		"deviceInfo": map[string]interface{}{
			"fingerprint": deviceFingerprint.Fingerprint,
			"hostname":    deviceFingerprint.Hostname,
			"mac_address": deviceFingerprint.MACAddress,
			"cpu_id":      deviceFingerprint.CPUID,
			"os":          deviceFingerprint.OS,
			"platform":    deviceFingerprint.Platform,
			"ip":          clientIP,
		},
	}

	// Send secure request using embedded Apps Script URL
	creds := config.GetCredentials()
	start := time.Now()
	signedResponse, err := secureClient.SecureRequest(ctx, creds.AppsScriptURL, requestPayload, deviceFingerprint.Fingerprint)
	if err != nil {
		// Check for timeout specifically
		if errors.Is(err, context.DeadlineExceeded) {
			m.logError(ctx, "apps_script_activation", "Activation request timed out",
				slog.Duration("timeout", 10*time.Second),
				slog.Duration("duration", time.Since(start)),
			)
			return license, fmt.Errorf("activation request timed out - please check your internet connection and try again")
		}
		
		m.logError(ctx, "apps_script_activation", "Secure activation request failed",
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)),
		)
		return license, fmt.Errorf("secure activation request failed: %w", err)
	}

	// Check for success
	if !signedResponse.Success {
		errorMsg := "unknown activation error"
		if signedResponse.Error != "" {
			errorMsg = signedResponse.Error
		}
		
		m.logWarn(ctx, "apps_script_activation", "Apps Script activation failed",
			slog.String("error", errorMsg),
			slog.String("license_key_prefix", sanitizedLicenseKey[:min(8, len(sanitizedLicenseKey))]),
		)
		
		// Handle specific error cases for better user feedback
		if strings.Contains(strings.ToLower(errorMsg), "already activated") {
			if strings.Contains(strings.ToLower(errorMsg), "different device") {
				return license, fmt.Errorf("this license has already been activated on a different device")
			}
			return license, fmt.Errorf("this license has already been activated")
		}
		
		if strings.Contains(strings.ToLower(errorMsg), "expired") {
			return license, fmt.Errorf("this license has expired")
		}
		
		if strings.Contains(strings.ToLower(errorMsg), "invalid") || strings.Contains(strings.ToLower(errorMsg), "not found") {
			return license, fmt.Errorf("invalid license key")
		}
		
		return license, fmt.Errorf("activation failed: %s", errorMsg)
	}

	// Extract activation data
	data := signedResponse.Data
	if data == nil {
		m.logError(ctx, "apps_script_activation", "Invalid activation response format - missing data field")
		return license, fmt.Errorf("invalid activation response format")
	}

	// Parse license information from activation response
	license.LicenseKey = sanitizedLicenseKey
	license.DeviceFingerprint = deviceFingerprint.Fingerprint
	
	if duration, exists := data["duration"]; exists {
		license.Duration = fmt.Sprintf("%v", duration)
	}
	
	if status, exists := data["status"]; exists {
		license.Status = fmt.Sprintf("%v", status)
	}
	
	// Check for expiry_date first (for compatibility), then expires_at (what Apps Script actually sends)
	if expiryStr, exists := data["expiry_date"]; exists && fmt.Sprintf("%v", expiryStr) != "" {
		if expiryDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", expiryStr)); err == nil {
			license.ExpiryDate = expiryDate
			m.logInfo(ctx, "apps_script_activation", "Parsed expiry date from 'expiry_date' field",
				slog.String("expiry_date", expiryDate.Format("2006-01-02")),
			)
		}
	} else if expiryStr, exists := data["expires_at"]; exists && fmt.Sprintf("%v", expiryStr) != "" {
		// Try parsing ISO format first (what Apps Script sends)
		if expiryDate, err := time.Parse(time.RFC3339, fmt.Sprintf("%v", expiryStr)); err == nil {
			license.ExpiryDate = expiryDate
			m.logInfo(ctx, "apps_script_activation", "Parsed expiry date from 'expires_at' field",
				slog.String("expiry_date", expiryDate.Format("2006-01-02")),
			)
		} else if expiryDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", expiryStr)); err == nil {
			// Fallback to date-only format
			license.ExpiryDate = expiryDate
			m.logInfo(ctx, "apps_script_activation", "Parsed expiry date from 'expires_at' field (date format)",
				slog.String("expiry_date", expiryDate.Format("2006-01-02")),
			)
		}
	}
	
	if issuedStr, exists := data["issued_date"]; exists && fmt.Sprintf("%v", issuedStr) != "" {
		if issuedDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", issuedStr)); err == nil {
			license.IssuedDate = issuedDate
		}
	}
	
	if activationID, exists := data["activation_id"]; exists {
		license.ActivationID = fmt.Sprintf("%v", activationID)
	}
	
	// Handle reactivation-specific data and scenarios
	var reactivationDetails *ReactivationDetails
	if license.Status == "reactivated" {
		reactivationDetails = &ReactivationDetails{}
		
		// Extract reactivation count
		if reactivationCount, exists := data["reactivation_count"]; exists {
			if count, ok := reactivationCount.(float64); ok {
				reactivationDetails.ReactivationCount = int(count)
			}
		}
		
		// Extract max reactivations
		if maxReactivations, exists := data["max_reactivations"]; exists {
			if max, ok := maxReactivations.(float64); ok {
				reactivationDetails.MaxReactivations = int(max)
			}
		}
		
		// Extract similarity score
		if similarityScore, exists := data["similarity_score"]; exists {
			if score, ok := similarityScore.(float64); ok {
				reactivationDetails.SimilarityScore = score
			}
		}
		
		// Extract previous device info
		if previousDevice, exists := data["previous_device_info"]; exists {
			reactivationDetails.PreviousDeviceInfo = fmt.Sprintf("%v", previousDevice)
		}
		
		// Set reactivation timestamp
		reactivationDetails.ReactivationTimestamp = time.Now()
		
		m.logInfo(ctx, "license_reactivation", "License reactivated on this device",
			slog.String("license_key_prefix", sanitizedLicenseKey[:min(8, len(sanitizedLicenseKey))]),
			slog.Int("reactivation_count", reactivationDetails.ReactivationCount),
			slog.Int("max_reactivations", reactivationDetails.MaxReactivations),
			slog.Float64("similarity_score", reactivationDetails.SimilarityScore),
			slog.String("previous_device", reactivationDetails.PreviousDeviceInfo),
		)
	}

	// Set default values
	license.UserEmail = "" // Scratch cards don't have user emails
	license.LastChecked = time.Now()

	// If no expiry date was set by the Apps Script, calculate it based on duration
	if license.ExpiryDate.IsZero() && license.Duration != "" {
		m.logWarn(ctx, "apps_script_activation", "No expiry date received from Apps Script, calculating from duration",
			slog.String("duration", license.Duration),
			slog.String("status", license.Status),
		)
		license.ExpiryDate = m.calculateExpiryDateFromDuration(license.Duration)
		license.IssuedDate = time.Now()
		m.logInfo(ctx, "apps_script_activation", "Calculated expiry date from duration",
			slog.String("calculated_expiry", license.ExpiryDate.Format("2006-01-02")),
		)
	}

	duration := time.Since(start)
	m.logInfo(ctx, "apps_script_activation", "License activated successfully via secure Apps Script",
		slog.String("license_key_prefix", sanitizedLicenseKey[:min(8, len(sanitizedLicenseKey))]),
		slog.String("status", license.Status),
		slog.String("duration", license.Duration),
		slog.String("activation_id", license.ActivationID),
		slog.Time("expiry_date", license.ExpiryDate),
		slog.Duration("request_duration", duration),
		slog.String("request_id", signedResponse.RequestID),
	)

	return license, nil
}

// calculateExpiryDateFromDuration calculates expiry date from duration string
func (m *Manager) calculateExpiryDateFromDuration(duration string) time.Time {
	var standardExpiry time.Time
	switch duration {
	case "1m":
		standardExpiry = time.Now().AddDate(0, 1, 0)
	case "3m":
		standardExpiry = time.Now().AddDate(0, 3, 0)
	case "6m":
		standardExpiry = time.Now().AddDate(0, 6, 0)
	case "1y":
		standardExpiry = time.Now().AddDate(1, 0, 0)
	default:
		standardExpiry = time.Now().AddDate(0, 1, 0) // Default to 1 month
	}
	
	// Set expiry to 12:00 AM next day after standard expiry
	return time.Date(standardExpiry.Year(), standardExpiry.Month(), standardExpiry.Day()+1, 0, 0, 0, 0, standardExpiry.Location())
}

// validateLicenseFromSheets validates license against Google Sheets (legacy method)
func (m *Manager) validateLicenseFromSheets(licenseKey string) (LicenseInfo, error) {
	var license LicenseInfo

	if m.config.UseServiceAccount && m.sheetsService != nil {
		// Use service account authentication
		resp, err := m.sheetsService.Spreadsheets.Values.Get(m.config.SheetID, m.config.SheetName).Do()
		if err != nil {
			return license, fmt.Errorf("failed to read from sheets: %v", err)
		}

		// Parse sheet data and find license
		for i, row := range resp.Values {
			if i == 0 {
				continue // Skip header row
			}
			if len(row) >= 1 && row[0].(string) == licenseKey {
				// Recharge card format: LicenseKey | Duration | ExpiryDate | Status | MachineID | ActivatedDate | LastConnected
				license.LicenseKey = row[0].(string)

				// Duration (column B)
				if len(row) > 1 {
					license.Duration = row[1].(string)
				}

				// ExpiryDate (column C) - may be empty for Available licenses
				if len(row) > 2 && row[2].(string) != "" {
					if expiryDate, err := time.Parse("2006-01-02", row[2].(string)); err == nil {
						license.ExpiryDate = expiryDate
					}
				}

				// Status (column D)
				if len(row) > 3 {
					license.Status = row[3].(string)
				}

				// MachineID (column E) - no longer used, skip
				// Column E was previously used for machine ID

				// ActivatedDate (column F)
				if len(row) > 5 && row[5].(string) != "" {
					if activatedDate, err := time.Parse("2006-01-02", row[5].(string)); err == nil {
						license.IssuedDate = activatedDate
					}
				}

				// LastConnected (column G) - new field
				if len(row) > 6 && row[6].(string) != "" {
					if lastConnected, err := time.Parse("2006-01-02 15:04:05", row[6].(string)); err == nil {
						license.LastChecked = lastConnected
					}
				} else {
					// Set default if column doesn't exist yet
					license.LastChecked = time.Now()
				}

				// ExpireStatus (column H) - new field (optional, for future use)
				// This is automatically calculated, so we don't need to parse it here

				// Set defaults for recharge cards
				license.UserEmail = "" // Recharge cards don't have user emails

				return license, nil
			}
		}
	} else {
		// Fallback to API key method
		url := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s?key=%s",
			m.config.SheetID, m.config.SheetName, m.config.APIKey)

		resp, err := http.Get(url)
		if err != nil {
			return license, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return license, err
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return license, err
		}

		// Parse sheet data and find license
		if values, ok := result["values"].([]interface{}); ok {
			for i, row := range values {
				if i == 0 {
					continue // Skip header row
				}
				if rowData, ok := row.([]interface{}); ok && len(rowData) >= 4 {
					// Check if this is our license key
					if len(rowData) > 0 && rowData[0].(string) == licenseKey {
						// Recharge card format: LicenseKey | Duration | ExpiryDate | Status | MachineID | ActivatedDate
						license.LicenseKey = rowData[0].(string)

						// Duration (column B)
						if len(rowData) > 1 {
							license.Duration = rowData[1].(string)
						}

						// ExpiryDate (column C) - may be empty for Available licenses
						if len(rowData) > 2 && rowData[2].(string) != "" {
							if expiryDate, err := time.Parse("2006-01-02", rowData[2].(string)); err == nil {
								license.ExpiryDate = expiryDate
							}
						}

						// Status (column D)
						if len(rowData) > 3 {
							license.Status = rowData[3].(string)
						}

						// MachineID (column E) - no longer used, skip
						// Column E was previously used for machine ID

						// ActivatedDate (column F)
						if len(rowData) > 5 && rowData[5].(string) != "" {
							if activatedDate, err := time.Parse("2006-01-02", rowData[5].(string)); err == nil {
								license.IssuedDate = activatedDate
							}
						}

						// LastConnected (column G) - new field
						if len(rowData) > 6 && rowData[6].(string) != "" {
							if lastConnected, err := time.Parse("2006-01-02 15:04:05", rowData[6].(string)); err == nil {
								license.LastChecked = lastConnected
							}
						} else {
							// Set default if column doesn't exist yet
							license.LastChecked = time.Now()
						}

						// ExpireStatus (column H) - new field (optional, for future use)
						// This is automatically calculated, so we don't need to parse it here

						// Set defaults for recharge cards
						license.UserEmail = "" // Recharge cards don't have user emails

						return license, nil
					}
				}
			}
		}
	}

	return license, fmt.Errorf("license not found")
}

// updateLicenseInSheets updates license in Google Sheets
func (m *Manager) updateLicenseInSheets(license LicenseInfo) error {
	if m.config.UseServiceAccount && m.sheetsService != nil {
		// Use service account authentication
		// First, find the row number for this license
		resp, err := m.sheetsService.Spreadsheets.Values.Get(m.config.SheetID, m.config.SheetName).Do()
		if err != nil {
			return fmt.Errorf("failed to read from sheets: %v", err)
		}

		var rowIndex int = -1
		for i, row := range resp.Values {
			if i == 0 {
				continue // Skip header row
			}
			if len(row) > 0 && row[0].(string) == license.LicenseKey {
				rowIndex = i + 1 // Google Sheets uses 1-based indexing
				break
			}
		}

		if rowIndex == -1 {
			return fmt.Errorf("license not found in sheet")
		}

		// Calculate expire status
		expireStatus := m.calculateExpireStatus(license.ExpiryDate)

		// Update the row with new license data
		// Format: LicenseKey | Duration | ExpiryDate | Status | MachineID | ActivatedDate | LastConnected | ExpireStatus
		values := [][]interface{}{
			{
				license.LicenseKey,
				license.Duration,
				license.ExpiryDate.Format("2006-01-02"),
				license.Status,
				"", // Machine ID removed
				license.IssuedDate.Format("2006-01-02"),
				license.LastChecked.Format("2006-01-02 15:04:05"), // Add LastConnected timestamp
				expireStatus,                                      // Add ExpireStatus
			},
		}

		rangeStr := fmt.Sprintf("%s!A%d:H%d", m.config.SheetName, rowIndex, rowIndex) // Extended to column H
		valueRange := &sheets.ValueRange{Values: values}

		_, err = m.sheetsService.Spreadsheets.Values.Update(
			m.config.SheetID,
			rangeStr,
			valueRange,
		).ValueInputOption("RAW").Do()

		return err
	} else {
		// Fallback to API key method
		// First, find the row number for this license
		url := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s?key=%s",
			m.config.SheetID, m.config.SheetName, m.config.APIKey)

		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}

		var rowIndex int = -1
		if values, ok := result["values"].([]interface{}); ok {
			for i, row := range values {
				if i == 0 {
					continue // Skip header row
				}
				if rowData, ok := row.([]interface{}); ok && len(rowData) > 0 {
					if rowData[0].(string) == license.LicenseKey {
						rowIndex = i + 1 // Google Sheets uses 1-based indexing
						break
					}
				}
			}
		}

		if rowIndex == -1 {
			return fmt.Errorf("license not found in sheet")
		}

		// Calculate expire status
		expireStatus := m.calculateExpireStatus(license.ExpiryDate)

		// Update the row with new license data
		// Format: LicenseKey | Duration | ExpiryDate | Status | MachineID | ActivatedDate | LastConnected | ExpireStatus
		values := [][]interface{}{
			{
				license.LicenseKey,
				license.Duration,
				license.ExpiryDate.Format("2006-01-02"),
				license.Status,
				"", // Machine ID removed
				license.IssuedDate.Format("2006-01-02"),
				license.LastChecked.Format("2006-01-02 15:04:05"), // Add LastConnected timestamp
				expireStatus,                                      // Add ExpireStatus
			},
		}

		updateURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s!A%d:H%d?valueInputOption=RAW&key=%s",
			m.config.SheetID, m.config.SheetName, rowIndex, rowIndex, m.config.APIKey) // Extended to column H

		payload := map[string]interface{}{
			"values": values,
		}

		return m.makeSheetRequest("PUT", updateURL, payload)
	}
}

// validateWithAppsScript performs periodic validation with Apps Script
func (m *Manager) validateWithAppsScript(license LicenseInfo) error {
	ctx := context.Background()
	creds := config.GetCredentials()
	
	// Prepare validation request with ActivationID
	requestData := map[string]interface{}{
		"action":        "validate",
		"code":          license.LicenseKey,
		"activation_id": license.ActivationID,
	}

	// Include device fingerprint for validation if available
	if license.DeviceFingerprint != "" {
		requestData["device_fingerprint"] = license.DeviceFingerprint
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to prepare validation request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", creds.AppsScriptURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ISX-Pulse-License-Client/1.0")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read validation response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("validation failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse validation response: %w", err)
	}

	// Check for success
	success, ok := response["success"].(bool)
	if !ok || !success {
		errorMsg := "unknown validation error"
		if msg, exists := response["error"]; exists {
			errorMsg = fmt.Sprintf("%v", msg)
		}
		return fmt.Errorf("validation failed: %s", errorMsg)
	}

	// Extract validation data to check for any updates
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid validation response format")
	}

	// Check if license status changed
	if status, exists := data["status"]; exists {
		statusStr := fmt.Sprintf("%v", status)
		if statusStr == "Revoked" {
			return fmt.Errorf("license has been revoked - please contact support")
		}
		if statusStr != "Activated" && statusStr != "Active" {
			return fmt.Errorf("license is no longer active - status: %s", statusStr)
		}
	}

	// Check if expiry date was updated (e.g., license was extended)
	// Check for expiry_date first (for compatibility), then expires_at (what Apps Script actually sends)
	expiryUpdated := false
	var newExpiryDate time.Time
	
	if expiryStr, exists := data["expiry_date"]; exists && fmt.Sprintf("%v", expiryStr) != "" {
		if expiryDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", expiryStr)); err == nil {
			newExpiryDate = expiryDate
			expiryUpdated = true
		}
	} else if expiryStr, exists := data["expires_at"]; exists && fmt.Sprintf("%v", expiryStr) != "" {
		// Try parsing ISO format first (what Apps Script sends)
		if expiryDate, err := time.Parse(time.RFC3339, fmt.Sprintf("%v", expiryStr)); err == nil {
			newExpiryDate = expiryDate
			expiryUpdated = true
		} else if expiryDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", expiryStr)); err == nil {
			// Fallback to date-only format
			newExpiryDate = expiryDate
			expiryUpdated = true
		}
	}
	
	if expiryUpdated && !newExpiryDate.Equal(license.ExpiryDate) {
		// License expiry date updated - save locally
		license.ExpiryDate = newExpiryDate
		license.LastChecked = time.Now()
		
		if err := m.saveLicenseLocal(license); err != nil {
			m.logWarn(ctx, "license_validation", "Failed to save updated license locally",
				slog.String("error", err.Error()),
			)
		}
		
		m.logInfo(ctx, "license_validation", "License expiry date updated",
			slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
			slog.Time("new_expiry", newExpiryDate),
		)
	}

	// Update last checked time
	license.LastChecked = time.Now()
	if err := m.saveLicenseLocal(license); err != nil {
		// Don't fail validation if we can't save locally
		m.logWarn(ctx, "license_validation", "Failed to update last checked time",
			slog.String("error", err.Error()),
		)
	}

	return nil
}

// validateWithSheets performs periodic validation with Google Sheets (legacy method)
func (m *Manager) validateWithSheets(license LicenseInfo) error {
	sheetLicense, err := m.validateLicenseFromSheets(license.LicenseKey)
	if err != nil {
		return err
	}

	// Check if license status changed to revoked or invalid
	if sheetLicense.Status == "Revoked" {
		return fmt.Errorf("license has been revoked - please contact support")
	}

	if sheetLicense.Status != "Activated" && sheetLicense.Status != "Active" {
		return fmt.Errorf("license is no longer active - status: %s", sheetLicense.Status)
	}

	// Check if expiry date changed (e.g., license was extended)
	if !sheetLicense.ExpiryDate.IsZero() && !sheetLicense.ExpiryDate.Equal(license.ExpiryDate) {
		// License expiry date updated
		license.ExpiryDate = sheetLicense.ExpiryDate
	}

	// Update last checked time locally
	license.LastChecked = time.Now()

	// Save updated license locally
	if err := m.saveLicenseLocal(license); err != nil {
		return fmt.Errorf("failed to save license locally: %v", err)
	}

	// Update Google Sheets with current timestamp to track "last connected"
	if err := m.updateLicenseInSheets(license); err != nil {
		// Don't fail if Google Sheets update fails, but log it
		// This prevents loss of local functionality if there are connectivity issues
		// Failed to update last connected time, but continue operation
	}

	return nil
}

// makeSheetRequest makes HTTP request to Google Sheets API
func (m *Manager) makeSheetRequest(method, url string, payload interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// calculateExpireStatus calculates the expire status based on days remaining
func (m *Manager) calculateExpireStatus(expiryDate time.Time) string {
	if expiryDate.IsZero() {
		return "Available" // For unactivated licenses
	}

	daysLeft := int(time.Until(expiryDate).Hours() / 24)

	if daysLeft <= 0 {
		return "Expired"
	} else if daysLeft <= 7 {
		return "Critical" // Red - 7 or fewer days
	} else if daysLeft <= 30 {
		return "Warning" // Yellow - 8-30 days
	} else {
		return "Active" // Green - more than 30 days
	}
}

// TestNetworkConnectivity tests connectivity to Google Sheets API
func (m *Manager) TestNetworkConnectivity() error {
	ctx := context.Background()
	m.logInfo(ctx, "network_connectivity_test", "Starting network connectivity tests")
	fmt.Printf("🔍 Testing network connectivity...\n")

	// Test basic internet connectivity
	fmt.Printf("   • Testing basic internet connectivity...")
	m.logDebug(ctx, "connectivity_test", "Testing basic internet connectivity")
	resp, err := http.Get("https://www.google.com")
	if err != nil {
		m.logError(ctx, "connectivity_test", "Basic internet connectivity failed", slog.String("error", err.Error()))
		fmt.Printf(" ❌ FAILED\n")
		return fmt.Errorf("no internet connection: %v", err)
	}
	resp.Body.Close()
	m.logDebug(ctx, "connectivity_test", "Basic internet connectivity OK")
	fmt.Printf(" ✅ OK\n")

	// Test Google APIs connectivity
	fmt.Printf("   • Testing Google APIs connectivity...")
	m.logDebug(ctx, "connectivity_test", "Testing Google APIs connectivity")
	resp, err = http.Get("https://sheets.googleapis.com")
	if err != nil {
		m.logError(ctx, "connectivity_test", "Google APIs connectivity failed", slog.String("error", err.Error()))
		fmt.Printf(" ❌ FAILED\n")
		return fmt.Errorf("cannot reach Google APIs: %v", err)
	}
	resp.Body.Close()
	m.logDebug(ctx, "connectivity_test", "Google APIs connectivity OK")
	fmt.Printf(" ✅ OK\n")

	// Test Google Sheets service initialization
	fmt.Printf("   • Testing Google Sheets service...")
	m.logDebug(ctx, "connectivity_test", "Testing Google Sheets service initialization")
	if m.sheetsService == nil {
		m.logError(ctx, "connectivity_test", "Google Sheets service not initialized")
		fmt.Printf(" ❌ FAILED\n")
		return fmt.Errorf("Google Sheets service not initialized")
	}
	m.logDebug(ctx, "connectivity_test", "Google Sheets service OK")
	fmt.Printf(" ✅ OK\n")

	// Test actual Google Sheets access
	fmt.Printf("   • Testing Google Sheets access...")
	m.logDebug(ctx, "connectivity_test", "Testing Google Sheets access", slog.String("sheet_id", m.config.SheetID))
	_, err = m.sheetsService.Spreadsheets.Get(m.config.SheetID).Do()
	if err != nil {
		m.logError(ctx, "connectivity_test", "Google Sheets access failed", 
			slog.String("error", err.Error()),
			slog.String("sheet_id", m.config.SheetID))
		fmt.Printf(" ❌ FAILED\n")
		return fmt.Errorf("cannot access Google Sheets: %v", err)
	}
	m.logDebug(ctx, "connectivity_test", "Google Sheets access OK")
	fmt.Printf(" ✅ OK\n")

	m.logInfo(ctx, "network_connectivity_test", "All connectivity tests passed successfully")
	fmt.Printf("🎉 All connectivity tests passed!\n")
	return nil
}

// RevokeLicense revokes a license (admin operation)
func (m *Manager) RevokeLicense(licenseKey string) error {
	// Strip dashes for consistent processing
	licenseKey = strings.ReplaceAll(licenseKey, "-", "")
	
	if licenseKey == "" {
		return fmt.Errorf("license key cannot be empty")
	}

	// Revoking license

	// Try to validate the license from Google Sheets
	licenseInfo, err := m.validateLicenseFromSheets(licenseKey)
	if err != nil {
		return fmt.Errorf("license validation failed: %v", err)
	}

	// Update license status to revoked
	licenseInfo.Status = "Revoked"
	licenseInfo.LastChecked = time.Now()

	// Update license in Google Sheets
	if err := m.updateLicenseInSheets(licenseInfo); err != nil {
		return fmt.Errorf("failed to revoke license in Google Sheets: %v", err)
	}

	// License revoked successfully
	return nil
}

// GetLicenseStatus returns detailed license status information with comprehensive observability
func (m *Manager) GetLicenseStatus() (*LicenseInfo, string, error) {
	ctx := context.Background()
	start := time.Now()
	
	// VERBOSE LOGGING: Entry point for license status check
	m.logInfo(ctx, "[VERBOSE] GetLicenseStatus entry", "Starting license status check flow",
		slog.String("operation", "get_license_status"),
		slog.String("license_file_path", m.licenseFile),
		slog.String("working_dir", m.getWorkingDir()),
		slog.Time("check_time", time.Now()),
	)
	
	// Try to load local license file
	m.logDebug(ctx, "loading_local_license", "Loading license from local file",
		slog.String("license_file", m.licenseFile),
	)
	
	license, err := m.loadLicenseLocal()
	loadLatency := time.Since(start)
	
	if err != nil {
		// Log license file loading failure (this is expected for not-activated licenses)
		m.logDebug(ctx, "no_local_license", "No local license file found",
			slog.Duration("load_latency", loadLatency),
			slog.String("error", err.Error()),
			slog.String("status_result", "not_activated"),
		)
		
		// No license file means not activated - this is not an error condition
		return nil, "Not Activated", nil
	}
	
	// Log successful license loading
	m.logDebug(ctx, "local_license_loaded", "Successfully loaded local license",
		slog.Duration("load_latency", loadLatency),
		slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
		slog.String("duration", license.Duration),
		slog.Time("expiry_date", license.ExpiryDate),
		slog.String("current_status", license.Status),
	)

	// VERBOSE LOGGING: Status calculation begins
	now := time.Now()
	daysLeft := int(time.Until(license.ExpiryDate).Hours() / 24)
	var status string
	
	m.logInfo(ctx, "[VERBOSE] License status calculation", "Starting detailed status calculation",
		slog.Time("current_time", now),
		slog.Time("expiry_date", license.ExpiryDate),
		slog.Int("days_left", daysLeft),
		slog.Bool("is_expired", now.After(license.ExpiryDate)),
		slog.String("time_until_expiry", time.Until(license.ExpiryDate).String()),
		slog.String("stored_status", license.Status),
	)

	if now.After(license.ExpiryDate) {
		status = "Expired"
		m.logWarn(ctx, "license_expired", "License has expired",
			slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
			slog.Time("expiry_date", license.ExpiryDate),
			slog.Int("days_expired", -daysLeft),
		)
	} else if daysLeft <= 7 {
		status = "Critical" // 7 or fewer days
		m.logWarn(ctx, "license_critical", "License expires within 7 days",
			slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
			slog.Int("days_left", daysLeft),
			slog.Time("expiry_date", license.ExpiryDate),
		)
	} else if daysLeft <= 30 {
		status = "Warning" // 8-30 days
		m.logInfo(ctx, "license_warning", "License expires within 30 days",
			slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
			slog.Int("days_left", daysLeft),
			slog.Time("expiry_date", license.ExpiryDate),
		)
	} else {
		status = "Active" // More than 30 days
		m.logDebug(ctx, "license_active", "License is active",
			slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
			slog.Int("days_left", daysLeft),
			slog.Time("expiry_date", license.ExpiryDate),
		)
	}
	
	// No machine validation needed anymore - licenses are portable

	// VERBOSE LOGGING: Final result summary
	totalLatency := time.Since(start)
	m.logInfo(ctx, "[VERBOSE] GetLicenseStatus complete", "License status check completed with full details",
		slog.Duration("total_latency", totalLatency),
		slog.String("final_status", status),
		slog.Int("days_left", daysLeft),
		slog.String("license_key_prefix", license.LicenseKey[:min(8, len(license.LicenseKey))]),
		slog.Time("license_issued", license.IssuedDate),
		slog.Time("license_expiry", license.ExpiryDate),
		slog.String("license_duration", license.Duration),
		slog.String("stored_status", license.Status),
		slog.Time("last_checked", license.LastChecked),
		slog.Bool("returning_info", true),
		slog.Bool("returning_error", false),
	)

	return &license, status, nil
}

// CheckRenewalStatus checks if license needs renewal and returns detailed info
func (m *Manager) CheckRenewalStatus() (*RenewalInfo, error) {
	license, err := m.loadLicenseLocal()
	if err != nil {
		return &RenewalInfo{
			Status:       "No License",
			Message:      "No license found. Please activate a license.",
			NeedsRenewal: true,
			IsExpired:    true,
		}, fmt.Errorf("no local license found: %v", err)
	}

	daysLeft := int(time.Until(license.ExpiryDate).Hours() / 24)
	renewalInfo := &RenewalInfo{DaysLeft: daysLeft}

	if time.Now().After(license.ExpiryDate) {
		renewalInfo.Status = "Expired"
		renewalInfo.Message = fmt.Sprintf("License expired %d days ago. Please renew immediately.", -daysLeft)
		renewalInfo.NeedsRenewal = true
		renewalInfo.IsExpired = true
	} else if daysLeft <= 7 {
		renewalInfo.Status = "Critical"
		renewalInfo.Message = fmt.Sprintf("License expires in %d days! Please renew soon to avoid interruption.", daysLeft)
		renewalInfo.NeedsRenewal = true
		renewalInfo.IsExpired = false
	} else if daysLeft <= 30 {
		renewalInfo.Status = "Warning"
		renewalInfo.Message = fmt.Sprintf("License expires in %d days. Consider renewing soon.", daysLeft)
		renewalInfo.NeedsRenewal = true
		renewalInfo.IsExpired = false
	} else {
		renewalInfo.Status = "Active"
		renewalInfo.Message = fmt.Sprintf("License is active with %d days remaining.", daysLeft)
		renewalInfo.NeedsRenewal = false
		renewalInfo.IsExpired = false
	}

	return renewalInfo, nil
}

// ShowRenewalNotification displays renewal notification if needed
func (m *Manager) ShowRenewalNotification() error {
	renewalInfo, err := m.CheckRenewalStatus()
	if err != nil {
		return err
	}

	if renewalInfo.NeedsRenewal {
		fmt.Printf("\n")
		fmt.Printf("┌─────────────────────────────────────────────────────────────────┐\n")
		fmt.Printf("│                    LICENSE RENEWAL NOTICE                      │\n")
		fmt.Printf("├─────────────────────────────────────────────────────────────────┤\n")

		switch renewalInfo.Status {
		case "Expired":
			fmt.Printf("│ ❌ STATUS: EXPIRED                                             │\n")
		case "Critical":
			fmt.Printf("│ 🚨 STATUS: CRITICAL - EXPIRES SOON                            │\n")
		case "Warning":
			fmt.Printf("│ ⚠️  STATUS: WARNING - RENEWAL RECOMMENDED                      │\n")
		}

		fmt.Printf("│                                                                 │\n")
		fmt.Printf("│ %-63s │\n", renewalInfo.Message)
		fmt.Printf("│                                                                 │\n")

		if renewalInfo.IsExpired {
			fmt.Printf("│ 🔒 Application functionality is limited until renewal.         │\n")
		} else {
			fmt.Printf("│ 📧 Contact support for license renewal options.               │\n")
		}

		fmt.Printf("│                                                                 │\n")
		fmt.Printf("│ 📞 Support: contact your license provider                      │\n")
		fmt.Printf("│ 🌐 Web: Use license activation interface for new licenses     │\n")
		fmt.Printf("└─────────────────────────────────────────────────────────────────┘\n")
		fmt.Printf("\n")
	}

	return nil
}

// ExtendLicense extends a license for additional time (admin operation)
func (m *Manager) ExtendLicense(licenseKey string, additionalDuration string) error {
	// Strip dashes for consistent processing
	licenseKey = strings.ReplaceAll(licenseKey, "-", "")
	
	if licenseKey == "" {
		return fmt.Errorf("license key cannot be empty")
	}

	// Extending license

	// Try to validate the license from Google Sheets
	licenseInfo, err := m.validateLicenseFromSheets(licenseKey)
	if err != nil {
		return fmt.Errorf("license validation failed: %v", err)
	}

	// Calculate additional time
	var additionalTime time.Duration
	switch additionalDuration {
	case "1m":
		additionalTime = 30 * 24 * time.Hour // 30 days
	case "3m":
		additionalTime = 90 * 24 * time.Hour // 90 days
	case "6m":
		additionalTime = 180 * 24 * time.Hour // 180 days
	case "1y":
		additionalTime = 365 * 24 * time.Hour // 365 days
	default:
		return fmt.Errorf("invalid duration: %s (use 1m, 3m, 6m, or 1y)", additionalDuration)
	}

	// Extend the expiry date
	licenseInfo.ExpiryDate = licenseInfo.ExpiryDate.Add(additionalTime)
	licenseInfo.LastChecked = time.Now()

	// License extended successfully

	// Update license in Google Sheets
	if err := m.updateLicenseInSheets(licenseInfo); err != nil {
		return fmt.Errorf("failed to extend license in Google Sheets: %v", err)
	}

	// Extension updated in Google Sheets
	return nil
}

// ValidateWithRenewalCheck performs validation and checks for renewal needs
func (m *Manager) ValidateWithRenewalCheck() (bool, *RenewalInfo, error) {
	// First perform normal validation
	isValid, err := m.ValidateLicense()

	// Get renewal information regardless of validation result
	renewalInfo, renewalErr := m.CheckRenewalStatus()
	if renewalErr != nil {
		renewalInfo = &RenewalInfo{
			Status:       "No License",
			Message:      "No license found",
			NeedsRenewal: true,
			IsExpired:    true,
		}
	}

	// Show notification if renewal is needed
	if renewalInfo.NeedsRenewal {
		m.ShowRenewalNotification()
	}

	return isValid, renewalInfo, err
}

// TrackOperation wraps an operation with performance tracking and logging
func (m *Manager) TrackOperation(operation string, fn func() error) error {
	start := time.Now()

	// Log operation start
	m.logDebug(context.Background(), operation+"_start", "Operation initiated")

	err := fn()
	duration := time.Since(start)

	// Record performance metrics
	m.recordPerformanceMetric(operation, duration, err == nil)

	// Log operation completion
	if err != nil {
		m.logError(context.Background(), operation+"_complete", "Operation failed",
			slog.Duration("duration", duration),
			slog.String("error", err.Error()),
		)
	} else {
		m.logInfo(context.Background(), operation+"_complete", "Operation completed successfully",
			slog.Duration("duration", duration),
		)
	}

	return err
}

// recordPerformanceMetric updates performance statistics
func (m *Manager) recordPerformanceMetric(operation string, duration time.Duration, success bool) {
	m.perfMutex.Lock()
	defer m.perfMutex.Unlock()

	if m.performanceData == nil {
		m.performanceData = make(map[string]*PerformanceMetrics)
	}

	metric, exists := m.performanceData[operation]
	if !exists {
		metric = &PerformanceMetrics{
			MinTime: duration,
			MaxTime: duration,
		}
		m.performanceData[operation] = metric
	}

	// Update metrics
	metric.Count++
	metric.TotalTime += duration
	metric.AverageTime = time.Duration(int64(metric.TotalTime) / metric.Count)
	metric.LastUpdated = time.Now()

	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}
	if duration < metric.MinTime {
		metric.MinTime = duration
	}

	if success {
		metric.SuccessCount++
	} else {
		metric.ErrorCount++
	}
}

// GetPerformanceMetrics returns performance statistics
func (m *Manager) GetPerformanceMetrics() map[string]*PerformanceMetrics {
	m.perfMutex.RLock()
	defer m.perfMutex.RUnlock()

	// Create a copy to avoid concurrent access issues
	result := make(map[string]*PerformanceMetrics)
	for k, v := range m.performanceData {
		result[k] = &PerformanceMetrics{
			Count:        v.Count,
			TotalTime:    v.TotalTime,
			AverageTime:  v.AverageTime,
			MaxTime:      v.MaxTime,
			MinTime:      v.MinTime,
			ErrorCount:   v.ErrorCount,
			SuccessCount: v.SuccessCount,
			LastUpdated:  v.LastUpdated,
		}
	}
	return result
}

// GetSystemStats returns comprehensive system statistics
func (m *Manager) GetSystemStats() map[string]interface{} {
	stats := map[string]interface{}{
		"performance": m.GetPerformanceMetrics(),
		"timestamp":   time.Now(),
		"version":     "enhanced-v2.0.0",
	}

	if m.cache != nil {
		stats["cache"] = m.cache.GetStats()
	}

	if m.security != nil {
		stats["security"] = m.security.GetStats()
	}

	return stats
}

// GetLicensePath returns the path to the license file
func (m *Manager) GetLicensePath() string {
	return m.licenseFile
}

// getWorkingDir returns the current working directory for logging
func (m *Manager) getWorkingDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "unknown"
}

// isWritable checks if a directory is writable
func isWritable(path string) bool {
	testFile := filepath.Join(path, ".write_test_" + fmt.Sprintf("%d", time.Now().UnixNano()))
	if f, err := os.Create(testFile); err == nil {
		f.Close()
		os.Remove(testFile)
		return true
	}
	return false
}

// Close properly shuts down the manager and its components
func (m *Manager) Close() error {
	// Stop cache cleanup goroutine
	if m.cache != nil {
		m.cache.Stop()
	}
	
	// Stop security manager cleanup goroutine
	if m.security != nil {
		m.security.Stop()
	}
	
	// Close secure credentials manager if in secure mode
	if m.secureMode && m.credentialsManager != nil {
		if err := m.credentialsManager.Close(); err != nil {
			m.logError(context.Background(), "credentials_manager_close", "Failed to close credentials manager",
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("failed to close credentials manager: %v", err)
		}
	}
	
	// Log manager shutdown
	m.logInfo(context.Background(), "manager_shutdown", "License manager closed successfully",
		slog.Bool("secure_mode", m.secureMode),
	)
	
	return nil
}

// validateLicenseFromAppsScriptWithCache validates license via Apps Script with caching support
func (m *Manager) validateLicenseFromAppsScriptWithCache(licenseKey string) (LicenseInfo, error) {
	// Check cache first
	if m.cache != nil {
		if cachedInfo, found := m.cache.Get(licenseKey); found {
			m.logDebug(context.Background(), "cache_hit", "License found in cache",
				slog.String("license_key_prefix", licenseKey[:min(8, len(licenseKey))]),
			)
			return *cachedInfo, nil
		}
	}

	// Cache miss - fetch from Apps Script
	licenseInfo, err := m.validateLicenseFromAppsScript(licenseKey)
	if err != nil {
		return licenseInfo, err
	}

	// Store in cache
	if m.cache != nil {
		m.cache.Set(licenseKey, licenseInfo)
		m.logDebug(context.Background(), "cache_store", "License stored in cache",
			slog.String("license_key_prefix", licenseKey[:min(8, len(licenseKey))]),
		)
	}

	return licenseInfo, nil
}

// GetDeviceFingerprint returns the current device fingerprint
func (m *Manager) GetDeviceFingerprint() (*security.DeviceFingerprint, error) {
	if m.fingerprintManager == nil {
		return nil, fmt.Errorf("fingerprint manager not initialized")
	}
	
	return m.fingerprintManager.GenerateFingerprint()
}

// GetDeviceFingerprintComponents returns device fingerprint components for debugging
func (m *Manager) GetDeviceFingerprintComponents() (map[string]string, error) {
	if m.fingerprintManager == nil {
		return nil, fmt.Errorf("fingerprint manager not initialized")
	}
	
	return m.fingerprintManager.GetFingerprintComponents()
}

// ValidateDeviceFingerprint compares current device with stored fingerprint
func (m *Manager) ValidateDeviceFingerprint(storedFingerprint string) (bool, error) {
	if m.fingerprintManager == nil {
		return false, fmt.Errorf("fingerprint manager not initialized")
	}
	
	return m.fingerprintManager.ValidateFingerprint(storedFingerprint)
}

// validateLicenseFromSheetsWithCache validates license with caching support (legacy method)
func (m *Manager) validateLicenseFromSheetsWithCache(licenseKey string) (LicenseInfo, error) {
	// Check cache first
	if m.cache != nil {
		if cachedInfo, found := m.cache.Get(licenseKey); found {
			m.logDebug(context.Background(), "cache_hit", "License found in cache",
				slog.String("license_key_prefix", licenseKey[:min(8, len(licenseKey))]),
			)
			return *cachedInfo, nil
		}
	}

	// Cache miss - fetch from Google Sheets
	licenseInfo, err := m.validateLicenseFromSheets(licenseKey)
	if err != nil {
		return licenseInfo, err
	}

	// Store in cache
	if m.cache != nil {
		m.cache.Set(licenseKey, licenseInfo)
		m.logDebug(context.Background(), "cache_store", "License stored in cache",
			slog.String("license_key_prefix", licenseKey[:min(8, len(licenseKey))]),
		)
	}

	return licenseInfo, nil
}
