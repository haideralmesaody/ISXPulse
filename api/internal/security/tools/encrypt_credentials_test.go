package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"isxcli/internal/security"
)

// TestBuildConfig tests BuildConfig structure
func TestBuildConfig(t *testing.T) {
	config := &BuildConfig{
		InputFile:      "test.json",
		OutputFile:     "test.dat",
		AppSalt:        "test-salt",
		SkipValidation: true,
	}

	if config.InputFile != "test.json" {
		t.Errorf("Expected InputFile 'test.json', got '%s'", config.InputFile)
	}

	if config.OutputFile != "test.dat" {
		t.Errorf("Expected OutputFile 'test.dat', got '%s'", config.OutputFile)
	}

	if config.AppSalt != "test-salt" {
		t.Errorf("Expected AppSalt 'test-salt', got '%s'", config.AppSalt)
	}

	if !config.SkipValidation {
		t.Error("Expected SkipValidation to be true")
	}
}

// TestReadAndValidateCredentials tests credential reading and validation
func TestReadAndValidateCredentials(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		skipValidation bool
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid_service_account",
			fileContent: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "test-key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...\n-----END PRIVATE KEY-----\n",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789012345678901",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token"
			}`,
			skipValidation: false,
			expectError:    false,
		},
		{
			name: "valid_rsa_private_key",
			fileContent: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "test-key-id",
				"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...\n-----END RSA PRIVATE KEY-----\n",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789012345678901",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token"
			}`,
			skipValidation: false,
			expectError:    false,
		},
		{
			name:           "skip_validation",
			fileContent:    `{"invalid": "json"}`,
			skipValidation: true,
			expectError:    false,
		},
		{
			name:           "invalid_json",
			fileContent:    `{invalid json}`,
			skipValidation: false,
			expectError:    true,
			errorContains:  "invalid JSON format",
		},
		{
			name: "missing_required_field",
			fileContent: `{
				"type": "service_account",
				"project_id": "test-project"
			}`,
			skipValidation: false,
			expectError:    true,
			errorContains:  "missing required field",
		},
		{
			name: "wrong_credential_type",
			fileContent: `{
				"type": "user_account",
				"project_id": "test-project",
				"private_key_id": "test-key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----\n",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789012345678901",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token"
			}`,
			skipValidation: false,
			expectError:    true,
			errorContains:  "invalid credential type",
		},
		{
			name: "invalid_private_key_format",
			fileContent: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "test-key-id",
				"private_key": "invalid-private-key",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789012345678901",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token"
			}`,
			skipValidation: false,
			expectError:    true,
			errorContains:  "invalid private key format",
		},
		{
			name: "private_key_not_string",
			fileContent: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "test-key-id",
				"private_key": 12345,
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789012345678901",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token"
			}`,
			skipValidation: false,
			expectError:    true,
			errorContains:  "private_key must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, "test-credentials.json")

			if err := os.WriteFile(tempFile, []byte(tt.fileContent), 0600); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test reading and validation
			credentials, err := readAndValidateCredentials(tempFile, tt.skipValidation)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(credentials) == 0 {
				t.Error("Credentials should not be empty")
			}

			// Verify content matches
			if string(credentials) != tt.fileContent {
				t.Error("Credentials content should match file content")
			}
		})
	}
}

// TestReadAndValidateCredentialsFileErrors tests file reading errors
func TestReadAndValidateCredentialsFileErrors(t *testing.T) {
	// Test non-existent file
	_, err := readAndValidateCredentials("/non/existent/file.json", false)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected file read error, got: %v", err)
	}
}

// TestContainsPrivateKeyMarkers tests private key validation
func TestContainsPrivateKeyMarkers(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "valid_private_key",
			key:      "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...\n-----END PRIVATE KEY-----\n",
			expected: true,
		},
		{
			name:     "valid_rsa_private_key",
			key:      "-----BEGIN RSA PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...\n-----END RSA PRIVATE KEY-----\n",
			expected: true,
		},
		{
			name:     "invalid_short_key",
			key:      "-----BEGIN PRIVATE KEY-----\nshort\n-----END PRIVATE KEY-----\n",
			expected: true, // This will pass length check but should be caught by other validation
		},
		{
			name:     "invalid_no_markers",
			key:      "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...",
			expected: false,
		},
		{
			name:     "invalid_wrong_markers",
			key:      "-----BEGIN CERTIFICATE-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...\n-----END CERTIFICATE-----\n",
			expected: false,
		},
		{
			name:     "empty_key",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsPrivateKeyMarkers(tt.key)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for key: %s", tt.expected, result, tt.key[:min(50, len(tt.key))])
			}
		})
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestEncryptCredentials tests credential encryption
func TestEncryptCredentials(t *testing.T) {
	tests := []struct {
		name        string
		credentials []byte
		appSalt     string
		expectError bool
	}{
		{
			name:        "valid_credentials",
			credentials: []byte(`{"type": "service_account", "project_id": "test"}`),
			appSalt:     DefaultAppSalt,
			expectError: false,
		},
		{
			name:        "empty_credentials",
			credentials: []byte{},
			appSalt:     DefaultAppSalt,
			expectError: true,
		},
		{
			name:        "short_salt",
			credentials: []byte(`{"type": "service_account"}`),
			appSalt:     "short",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := encryptCredentials(tt.credentials, tt.appSalt)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if payload == nil {
				t.Error("Payload should not be nil")
				return
			}

			// Verify payload structure
			if payload.Version != 1 {
				t.Errorf("Expected version 1, got %d", payload.Version)
			}

			if len(payload.Salt) == 0 {
				t.Error("Salt should not be empty")
			}

			if len(payload.Nonce) == 0 {
				t.Error("Nonce should not be empty")
			}

			if len(payload.Ciphertext) == 0 {
				t.Error("Ciphertext should not be empty")
			}

			if len(payload.AuthTag) == 0 {
				t.Error("AuthTag should not be empty")
			}

			// Test decryption to verify encryption worked
			credentials, err := security.DecryptCredentials(payload, []byte(tt.appSalt), nil)
			if err != nil {
				t.Errorf("Failed to decrypt: %v", err)
				return
			}
			defer credentials.Clear()

			if string(credentials.Data()) != string(tt.credentials) {
				t.Error("Decrypted data does not match original")
			}
		})
	}
}

// TestSaveEncryptedPayload tests saving encrypted payload to file
func TestSaveEncryptedPayload(t *testing.T) {
	// Create test payload
	testCredentials := []byte(`{"type": "service_account", "project_id": "test"}`)
	payload, err := encryptCredentials(testCredentials, DefaultAppSalt)
	if err != nil {
		t.Fatalf("Failed to create test payload: %v", err)
	}

	tests := []struct {
		name        string
		outputPath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_output_path",
			outputPath:  "test-output.dat",
			expectError: false,
		},
		{
			name:        "nested_directory",
			outputPath:  "subdir/nested/test-output.dat",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, tt.outputPath)

			err := saveEncryptedPayload(payload, outputFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file was created
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				t.Error("Output file was not created")
				return
			}

			// Verify file content
			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Errorf("Failed to read output file: %v", err)
				return
			}

			var loadedPayload security.EncryptedPayload
			if err := json.Unmarshal(data, &loadedPayload); err != nil {
				t.Errorf("Failed to unmarshal saved payload: %v", err)
				return
			}

			// Verify payload matches
			if loadedPayload.Version != payload.Version {
				t.Error("Saved payload version does not match")
			}

			if len(loadedPayload.Ciphertext) != len(payload.Ciphertext) {
				t.Error("Saved payload ciphertext length does not match")
			}

			// Verify file permissions (Unix-like systems)
			fileInfo, err := os.Stat(outputFile)
			if err != nil {
				t.Errorf("Failed to get file info: %v", err)
				return
			}

			// Check that file is not world-readable (on Unix-like systems)
			mode := fileInfo.Mode()
			if mode&0004 != 0 {
				t.Log("Warning: File is world-readable (may be expected on Windows)")
			}
		})
	}
}

// TestLoadBuildConfig tests loading build configuration from JSON
func TestLoadBuildConfig(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectError    bool
		expectedConfig *BuildConfig
	}{
		{
			name: "valid_config",
			configContent: `{
				"input_file": "custom-input.json",
				"output_file": "custom-output.dat",
				"app_salt": "custom-salt",
				"skip_validation": true
			}`,
			expectError: false,
			expectedConfig: &BuildConfig{
				InputFile:      "custom-input.json",
				OutputFile:     "custom-output.dat",
				AppSalt:        "custom-salt",
				SkipValidation: true,
			},
		},
		{
			name:          "invalid_json",
			configContent: `{invalid json}`,
			expectError:   true,
		},
		{
			name: "partial_config",
			configContent: `{
				"input_file": "partial.json"
			}`,
			expectError: false,
			expectedConfig: &BuildConfig{
				InputFile:      "partial.json",
				OutputFile:     "",
				AppSalt:        "",
				SkipValidation: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.json")

			if err := os.WriteFile(configFile, []byte(tt.configContent), 0600); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			config, err := loadBuildConfig(configFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Error("Config should not be nil")
				return
			}

			// Compare with expected config
			if tt.expectedConfig != nil {
				if config.InputFile != tt.expectedConfig.InputFile {
					t.Errorf("Expected InputFile %q, got %q", tt.expectedConfig.InputFile, config.InputFile)
				}
				if config.OutputFile != tt.expectedConfig.OutputFile {
					t.Errorf("Expected OutputFile %q, got %q", tt.expectedConfig.OutputFile, config.OutputFile)
				}
				if config.AppSalt != tt.expectedConfig.AppSalt {
					t.Errorf("Expected AppSalt %q, got %q", tt.expectedConfig.AppSalt, config.AppSalt)
				}
				if config.SkipValidation != tt.expectedConfig.SkipValidation {
					t.Errorf("Expected SkipValidation %v, got %v", tt.expectedConfig.SkipValidation, config.SkipValidation)
				}
			}
		})
	}
}

// TestLoadBuildConfigFileErrors tests file reading errors
func TestLoadBuildConfigFileErrors(t *testing.T) {
	_, err := loadBuildConfig("/non/existent/config.json")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
}

// TestMaskSensitiveData tests sensitive data masking
func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long_data",
			input:    "sensitive-data-that-should-be-masked",
			expected: "sens***sked",
		},
		{
			name:     "medium_data",
			input:    "password123",
			expected: "pass***d123",
		},
		{
			name:     "short_data",
			input:    "short",
			expected: "***",
		},
		{
			name:     "very_short_data",
			input:    "abc",
			expected: "***",
		},
		{
			name:     "empty_data",
			input:    "",
			expected: "***",
		},
		{
			name:     "exactly_8_chars",
			input:    "12345678",
			expected: "***",
		},
		{
			name:     "9_chars",
			input:    "123456789",
			expected: "1234***6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestDefaultConstants tests default constants
func TestDefaultConstants(t *testing.T) {
	if DefaultAppSalt == "" {
		t.Error("DefaultAppSalt should not be empty")
	}

	if len(DefaultAppSalt) < 16 {
		t.Error("DefaultAppSalt should be at least 16 characters for security")
	}

	if BuildVersion == "" {
		t.Error("BuildVersion should not be empty")
	}

	if BuildTool == "" {
		t.Error("BuildTool should not be empty")
	}

	// Test that default salt is reasonable
	if !strings.Contains(DefaultAppSalt, "ISX") {
		t.Error("DefaultAppSalt should contain application identifier")
	}

	t.Logf("DefaultAppSalt: %s", maskSensitiveData(DefaultAppSalt))
	t.Logf("BuildVersion: %s", BuildVersion)
	t.Logf("BuildTool: %s", BuildTool)
}

// TestGenerateIntegrationCodeOutput tests that generateIntegrationCode doesn't panic
func TestGenerateIntegrationCodeOutput(t *testing.T) {
	// Capture stdout to verify output
	testOutputFile := "test-output.dat"
	
	// This function writes to stdout, so we just verify it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("generateIntegrationCode panicked: %v", r)
		}
	}()
	
	generateIntegrationCode(testOutputFile)
	
	// If we get here without panicking, the test passes
	t.Log("generateIntegrationCode executed without panic")
}

// TestEndToEndEncryption tests the complete encryption workflow
func TestEndToEndEncryption(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test credentials file
	testCredentials := `{
		"type": "service_account",
		"project_id": "test-project-12345",
		"private_key_id": "test-key-id-67890",
		"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKB\nwxm34567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\n-----END PRIVATE KEY-----\n",
		"client_email": "test-service@test-project-12345.iam.gserviceaccount.com",
		"client_id": "123456789012345678901",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token"
	}`
	
	inputFile := filepath.Join(tempDir, "test-credentials.json")
	outputFile := filepath.Join(tempDir, "encrypted-credentials.dat")
	
	// Write test credentials
	if err := os.WriteFile(inputFile, []byte(testCredentials), 0600); err != nil {
		t.Fatalf("Failed to create test credentials: %v", err)
	}
	
	// Step 1: Read and validate credentials
	credentials, err := readAndValidateCredentials(inputFile, false)
	if err != nil {
		t.Fatalf("Failed to read credentials: %v", err)
	}
	
	// Step 2: Encrypt credentials
	payload, err := encryptCredentials(credentials, DefaultAppSalt)
	if err != nil {
		t.Fatalf("Failed to encrypt credentials: %v", err)
	}
	
	// Step 3: Save encrypted payload
	if err := saveEncryptedPayload(payload, outputFile); err != nil {
		t.Fatalf("Failed to save encrypted payload: %v", err)
	}
	
	// Step 4: Verify we can load and decrypt
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}
	
	var loadedPayload security.EncryptedPayload
	if err := json.Unmarshal(data, &loadedPayload); err != nil {
		t.Fatalf("Failed to unmarshal encrypted payload: %v", err)
	}
	
	// Decrypt and verify
	decryptedCreds, err := security.DecryptCredentials(&loadedPayload, []byte(DefaultAppSalt), nil)
	if err != nil {
		t.Fatalf("Failed to decrypt credentials: %v", err)
	}
	defer decryptedCreds.Clear()
	
	// Verify content matches
	if string(decryptedCreds.Data()) != testCredentials {
		t.Error("Decrypted credentials do not match original")
	}
	
	// Verify JSON structure is intact
	var originalCreds, decryptedCredsJSON map[string]interface{}
	if err := json.Unmarshal([]byte(testCredentials), &originalCreds); err != nil {
		t.Fatalf("Failed to unmarshal original credentials: %v", err)
	}
	
	if err := json.Unmarshal(decryptedCreds.Data(), &decryptedCredsJSON); err != nil {
		t.Fatalf("Failed to unmarshal decrypted credentials: %v", err)
	}
	
	// Compare key fields
	if originalCreds["project_id"] != decryptedCredsJSON["project_id"] {
		t.Error("Project ID does not match after encryption/decryption")
	}
	
	if originalCreds["client_email"] != decryptedCredsJSON["client_email"] {
		t.Error("Client email does not match after encryption/decryption")
	}
	
	t.Log("End-to-end encryption test completed successfully")
}