package license

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"isxcli/internal/security"
)

// TestScratchCardOneTimeActivation ensures that scratch cards can only be activated once
func TestScratchCardOneTimeActivation(t *testing.T) {
	tests := []struct {
		name                string
		licenseKey          string
		fingerprint1        string
		fingerprint2        string
		expectSecondError   bool
		secondErrorContains string
	}{
		{
			name:                "same device activation twice",
			licenseKey:          "ISX-1M23-4567-890A",
			fingerprint1:        "device1_fingerprint",
			fingerprint2:        "device1_fingerprint",
			expectSecondError:   true,
			secondErrorContains: "already activated",
		},
		{
			name:                "different device activation after first",
			licenseKey:          "ISX-2M34-5678-901B",
			fingerprint1:        "device1_fingerprint",
			fingerprint2:        "device2_fingerprint",
			expectSecondError:   true,
			secondErrorContains: "already activated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock Apps Script server
			var activationCount int32
			server := createMockAppsScriptServer(t, func(action string, data map[string]interface{}) map[string]interface{} {
				switch action {
				case "activateScratchCard":
					code := data["licenseKey"].(string)
					fingerprint := data["deviceFingerprint"].(string)

					// First activation succeeds
					if atomic.AddInt32(&activationCount, 1) == 1 {
						return map[string]interface{}{
							"success":      true,
							"activationId": "act_" + code + "_" + fingerprint[:8],
							"message":      "Activation successful",
						}
					}

					// Subsequent activations fail
					return map[string]interface{}{
						"success": false,
						"error":   "License already activated on another device",
					}
				default:
					return map[string]interface{}{"success": false, "error": "Unknown action"}
				}
			})
			defer server.Close()

			manager := createTestManagerWithAppsScript(t, server.URL)
			ctx := context.Background()

			// First activation should succeed
			info1, err1 := manager.ActivateScratchCard(ctx, tt.licenseKey, tt.fingerprint1)
			require.NoError(t, err1)
			assert.True(t, info1.IsValid)
			assert.Contains(t, info1.ActivationID, "act_")
			assert.Equal(t, tt.fingerprint1, info1.DeviceFingerprint)

			// Second activation should fail
			info2, err2 := manager.ActivateScratchCard(ctx, tt.licenseKey, tt.fingerprint2)
			if tt.expectSecondError {
				require.Error(t, err2)
				assert.Contains(t, err2.Error(), tt.secondErrorContains)
				assert.False(t, info2.IsValid)
			} else {
				require.NoError(t, err2)
				assert.True(t, info2.IsValid)
			}
		})
	}
}

// TestScratchCardDuplicatePreventionStress tests concurrent activation attempts
func TestScratchCardDuplicatePreventionStress(t *testing.T) {
	licenseKey := "ISX-ABCD-EFGH-IJKL"
	deviceCount := 10
	goroutinesPerDevice := 5
	totalAttempts := deviceCount * goroutinesPerDevice

	var successCount int64
	var errorCount int64

	// Mock Apps Script server with atomic activation tracking
	var activatedDevices sync.Map
	server := createMockAppsScriptServer(t, func(action string, data map[string]interface{}) map[string]interface{} {
		if action != "activateScratchCard" {
			return map[string]interface{}{"success": false, "error": "Unknown action"}
		}

		code := data["licenseKey"].(string)
		fingerprint := data["deviceFingerprint"].(string)

		// Simulate race condition with slight delay
		time.Sleep(time.Millisecond * 10)

		// Check if already activated (atomic operation)
		if _, exists := activatedDevices.LoadOrStore(code, fingerprint); exists {
			return map[string]interface{}{
				"success": false,
				"error":   "License already activated",
			}
		}

		return map[string]interface{}{
			"success":      true,
			"activationId": "act_" + code + "_" + fingerprint[:8],
			"message":      "Activation successful",
		}
	})
	defer server.Close()

	manager := createTestManagerWithAppsScript(t, server.URL)
	ctx := context.Background()

	// Launch concurrent activation attempts
	var wg sync.WaitGroup
	for device := 0; device < deviceCount; device++ {
		deviceFingerprint := fmt.Sprintf("device_%d_fingerprint", device)

		for attempt := 0; attempt < goroutinesPerDevice; attempt++ {
			wg.Add(1)
			go func(fingerprint string) {
				defer wg.Done()

				_, err := manager.ActivateScratchCard(ctx, licenseKey, fingerprint)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}(deviceFingerprint)
		}
	}

	wg.Wait()

	// Only one activation should succeed
	assert.Equal(t, int64(1), successCount, "Expected exactly one successful activation")
	assert.Equal(t, int64(totalAttempts-1), errorCount, "Expected all other activations to fail")
}

// TestScratchCardRateLimiting tests rate limiting for activation attempts
func TestScratchCardRateLimiting(t *testing.T) {
	licenseKey := "ISX-RATE-LIMIT-TEST"
	deviceFingerprint := "rate_test_device"

	var requestCount int64
	server := createMockAppsScriptServer(t, func(action string, data map[string]interface{}) map[string]interface{} {
		atomic.AddInt64(&requestCount, 1)

		if action != "activateScratchCard" {
			return map[string]interface{}{"success": false, "error": "Unknown action"}
		}

		// Simulate rate limiting after 3 requests per minute
		if atomic.LoadInt64(&requestCount) > 3 {
			return map[string]interface{}{
				"success": false,
				"error":   "Rate limit exceeded",
			}
		}

		return map[string]interface{}{
			"success":      true,
			"activationId": "act_rate_test",
			"message":      "Activation successful",
		}
	})
	defer server.Close()

	manager := createTestManagerWithAppsScript(t, server.URL)
	ctx := context.Background()

	// Make rapid activation attempts
	var successCount, errorCount int
	for i := 0; i < 10; i++ {
		_, err := manager.ActivateScratchCard(ctx, licenseKey, deviceFingerprint)
		if err != nil {
			errorCount++
			if i >= 3 {
				// After 3 attempts, should be rate limited
				assert.Contains(t, err.Error(), "Rate limit", "Expected rate limit error")
			}
		} else {
			successCount++
		}

		// Small delay between attempts
		time.Sleep(time.Millisecond * 50)
	}

	assert.GreaterOrEqual(t, errorCount, 5, "Expected rate limiting to kick in")
	assert.GreaterOrEqual(t, successCount, 1, "Expected at least one success before rate limiting")
}

// TestScratchCardInvalidKeyHandling tests handling of invalid license keys
func TestScratchCardInvalidKeyHandling(t *testing.T) {
	tests := []struct {
		name             string
		licenseKey       string
		expectValidation bool
		errorContains    string
	}{
		{
			name:             "empty key",
			licenseKey:       "",
			expectValidation: false,
			errorContains:    "empty",
		},
		{
			name:             "invalid format",
			licenseKey:       "INVALID-KEY",
			expectValidation: false,
			errorContains:    "format",
		},
		{
			name:             "wrong prefix",
			licenseKey:       "ABC-1234-5678-90AB",
			expectValidation: false,
			errorContains:    "ISX",
		},
		{
			name:             "invalid characters",
			licenseKey:       "ISX-1!@#-$%^&-*()",
			expectValidation: false,
			errorContains:    "characters",
		},
		{
			name:             "SQL injection attempt",
			licenseKey:       "ISX-1'; DROP TABLE licenses; --",
			expectValidation: false,
			errorContains:    "format",
		},
		{
			name:             "XSS attempt",
			licenseKey:       "ISX-<script>alert('xss')</script>",
			expectValidation: false,
			errorContains:    "format",
		},
	}

	server := createMockAppsScriptServer(t, func(action string, data map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"success": false, "error": "Should not reach server"}
	})
	defer server.Close()

	manager := createTestManagerWithAppsScript(t, server.URL)
	ctx := context.Background()
	deviceFingerprint := "test_device_fingerprint"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := manager.ActivateScratchCard(ctx, tt.licenseKey, deviceFingerprint)

			if tt.expectValidation {
				require.NoError(t, err)
				assert.True(t, info.IsValid)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.False(t, info.IsValid)
			}
		})
	}
}

// TestScratchCardDeviceFingerprintValidation tests device fingerprint validation
func TestScratchCardDeviceFingerprintValidation(t *testing.T) {
	tests := []struct {
		name          string
		fingerprint   string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid fingerprint",
			fingerprint: "abc123def456789012345678901234567890abcd",
			expectError: false,
		},
		{
			name:          "empty fingerprint",
			fingerprint:   "",
			expectError:   true,
			errorContains: "empty",
		},
		{
			name:          "too short fingerprint",
			fingerprint:   "abc123",
			expectError:   true,
			errorContains: "length",
		},
		{
			name:          "invalid characters in fingerprint",
			fingerprint:   "abc123def456!@#$%^&*()",
			expectError:   true,
			errorContains: "characters",
		},
	}

	server := createMockAppsScriptServer(t, func(action string, data map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"success":      true,
			"activationId": "act_test",
			"message":      "Activation successful",
		}
	})
	defer server.Close()

	manager := createTestManagerWithAppsScript(t, server.URL)
	ctx := context.Background()
	licenseKey := "ISX-1234-5678-90AB"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := manager.ActivateScratchCard(ctx, licenseKey, tt.fingerprint)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.False(t, info.IsValid)
			} else {
				require.NoError(t, err)
				assert.True(t, info.IsValid)
			}
		})
	}
}

// TestScratchCardAppsScriptIntegration tests integration with Apps Script
func TestScratchCardAppsScriptIntegration(t *testing.T) {
	tests := []struct {
		name               string
		serverResponse     map[string]interface{}
		serverStatusCode   int
		expectError        bool
		expectActivationID bool
		errorContains      string
	}{
		{
			name: "successful activation",
			serverResponse: map[string]interface{}{
				"success":      true,
				"activationId": "act_12345678",
				"message":      "License activated successfully",
			},
			serverStatusCode:   200,
			expectError:        false,
			expectActivationID: true,
		},
		{
			name: "license already activated",
			serverResponse: map[string]interface{}{
				"success": false,
				"error":   "License already activated on another device",
			},
			serverStatusCode:   200,
			expectError:        true,
			expectActivationID: false,
			errorContains:      "already activated",
		},
		{
			name: "invalid license code",
			serverResponse: map[string]interface{}{
				"success": false,
				"error":   "License code not found",
			},
			serverStatusCode:   200,
			expectError:        true,
			expectActivationID: false,
			errorContains:      "not found",
		},
		{
			name:               "server error",
			serverResponse:     nil,
			serverStatusCode:   500,
			expectError:        true,
			expectActivationID: false,
			errorContains:      "500",
		},
		{
			name: "malformed response",
			serverResponse: map[string]interface{}{
				"invalid": "response",
			},
			serverStatusCode:   200,
			expectError:        true,
			expectActivationID: false,
			errorContains:      "unexpected response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatusCode)
				if tt.serverResponse != nil {
					w.Header().Set("Content-Type", "application/json")
					// Simulate JSON response based on the test case
					if tt.serverResponse["success"] == true {
						fmt.Fprintf(w, `{"success":true,"activationId":"%s","message":"%s"}`,
							tt.serverResponse["activationId"],
							tt.serverResponse["message"])
					} else if tt.serverResponse["success"] == false {
						fmt.Fprintf(w, `{"success":false,"error":"%s"}`, tt.serverResponse["error"])
					} else {
						fmt.Fprint(w, `{"invalid":"response"}`)
					}
				}
			}))
			defer server.Close()

			manager := createTestManagerWithAppsScript(t, server.URL)
			ctx := context.Background()

			info, err := manager.ActivateScratchCard(ctx, "ISX-1234-5678-90AB", "test_fingerprint")

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.False(t, info.IsValid)
			} else {
				require.NoError(t, err)
				assert.True(t, info.IsValid)
			}

			if tt.expectActivationID {
				assert.NotEmpty(t, info.ActivationID, "Expected activation ID to be set")
				assert.Contains(t, info.ActivationID, "act_")
			} else {
				assert.Empty(t, info.ActivationID, "Expected activation ID to be empty")
			}
		})
	}
}

// TestScratchCardFormatValidation tests comprehensive format validation
func TestScratchCardFormatValidation(t *testing.T) {
	tests := []struct {
		name           string
		licenseKey     string
		expectNormalized string
		expectError      bool
		errorContains    string
	}{
		{
			name:             "valid with dashes",
			licenseKey:       "ISX-1M23-4567-890A",
			expectNormalized: "ISX-1M23-4567-890A",
			expectError:      false,
		},
		{
			name:             "valid without dashes",
			licenseKey:       "ISX1M234567890A",
			expectNormalized: "ISX-1M23-4567-890A",
			expectError:      false,
		},
		{
			name:             "lowercase normalization",
			licenseKey:       "isx-1m23-4567-890a",
			expectNormalized: "ISX-1M23-4567-890A",
			expectError:      false,
		},
		{
			name:          "invalid prefix",
			licenseKey:    "ABC-1M23-4567-890A",
			expectError:   true,
			errorContains: "must start with",
		},
		{
			name:          "wrong length",
			licenseKey:    "ISX-12-45-67",
			expectError:   true,
			errorContains: "format",
		},
		{
			name:          "invalid characters",
			licenseKey:    "ISX-1!23-4567-890A",
			expectError:   true,
			errorContains: "characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test format validation
			err := ValidateScratchCardFormat(tt.licenseKey)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}

			// Test normalization (if validation passes)
			if !tt.expectError {
				normalized := NormalizeScratchCardKey(tt.licenseKey)
				formatted := FormatScratchCardKeyWithDashes(normalized)
				assert.Equal(t, tt.expectNormalized, formatted)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkScratchCardActivation(b *testing.B) {
	server := createMockAppsScriptServer(nil, func(action string, data map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"success":      true,
			"activationId": "act_benchmark",
			"message":      "Activation successful",
		}
	})
	defer server.Close()

	manager := createTestManagerWithAppsScript(nil, server.URL)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		licenseKey := fmt.Sprintf("ISX-BENCH-%04d-%03d", i%10000, i%1000)
		fingerprint := fmt.Sprintf("benchmark_device_%d", i)
		
		_, err := manager.ActivateScratchCard(ctx, licenseKey, fingerprint)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// Helper function to create mock Apps Script server
func createMockAppsScriptServer(t *testing.T, handler func(string, map[string]interface{}) map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var requestData struct {
			Action string                 `json:"action"`
			Data   map[string]interface{} `json:"data,omitempty"`
		}

		// For simplified testing, extract action and data from request body
		// In real implementation, this would parse the signed request structure
		var simpleRequest map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&simpleRequest); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		action, ok := simpleRequest["action"].(string)
		if !ok {
			action = "activateScratchCard" // Default for testing
		}

		response := handler(action, simpleRequest)

		w.Header().Set("Content-Type", "application/json")
		if response["success"] == true {
			fmt.Fprintf(w, `{"success":true,"activationId":"%s","message":"%s"}`,
				response["activationId"],
				response["message"])
		} else {
			fmt.Fprintf(w, `{"success":false,"error":"%s"}`, response["error"])
		}
	}))
}

// Helper function to create test manager with Apps Script URL
func createTestManagerWithAppsScript(t *testing.T, appsScriptURL string) *Manager {
	// Create a test manager with minimal configuration
	manager := &Manager{
		appsScriptURL: appsScriptURL,
		// Add other necessary fields for testing
	}

	return manager
}