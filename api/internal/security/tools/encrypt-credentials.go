package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"isxcli/internal/security"
	"log/slog"
)

// BuildConfig holds build-time encryption configuration
type BuildConfig struct {
	InputFile    string `json:"input_file"`
	OutputFile   string `json:"output_file"`
	AppSalt      string `json:"app_salt"`
	SkipValidation bool `json:"skip_validation"`
}

// Application constants for salt generation
const (
	// Application-specific salt (this should be unique per application)
	DefaultAppSalt = "ISX-Daily-Reports-Scrapper-v2.0-Salt-2025"
	
	// Build metadata
	BuildVersion = "2.0.0"
	BuildTool    = "ISX-Credential-Encryptor"
)

func main() {
	var (
		inputFile  = flag.String("input", "credentials.json", "Input credentials file")
		outputFile = flag.String("output", "credentials_encrypted.dat", "Output encrypted file")
		appSalt    = flag.String("salt", DefaultAppSalt, "Application salt for encryption")
		skipValidation = flag.Bool("skip-validation", false, "Skip input validation")
		configFile = flag.String("config", "", "Configuration file (JSON)")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Load configuration from file if provided
	var config *BuildConfig
	if *configFile != "" {
		var err error
		config, err = loadBuildConfig(*configFile)
		if err != nil {
			slog.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
		}
	} else {
		config = &BuildConfig{
			InputFile:      *inputFile,
			OutputFile:     *outputFile,
			AppSalt:        *appSalt,
			SkipValidation: *skipValidation,
		}
	}

	if *verbose {
		fmt.Printf("🔧 %s v%s\n", BuildTool, BuildVersion)
		fmt.Printf("📁 Input file: %s\n", config.InputFile)
		fmt.Printf("📁 Output file: %s\n", config.OutputFile)
		fmt.Printf("🧂 App salt: %s\n", maskSensitiveData(config.AppSalt))
	}

	// Validate input file exists
	if _, err := os.Stat(config.InputFile); os.IsNotExist(err) {
		slog.Error("Input file does not exist", slog.String("file", config.InputFile))
		os.Exit(1)
	}

	// Read and validate credentials
	credentials, err := readAndValidateCredentials(config.InputFile, config.SkipValidation)
	if err != nil {
		slog.Error("Failed to read credentials", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("✅ Credentials loaded and validated\n")
		fmt.Printf("📊 Credential size: %d bytes\n", len(credentials))
	}

	// Encrypt credentials
	encryptedPayload, err := encryptCredentials(credentials, config.AppSalt)
	if err != nil {
		slog.Error("Failed to encrypt credentials", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("🔐 Credentials encrypted successfully\n")
		fmt.Printf("📊 Encrypted size: %d bytes\n", len(encryptedPayload.Ciphertext))
	}

	// Save encrypted payload
	if err := saveEncryptedPayload(encryptedPayload, config.OutputFile); err != nil {
		slog.Error("Failed to save encrypted payload", slog.String("error", err.Error()))
		os.Exit(1)
	}

	fmt.Printf("✅ Credentials encrypted and saved to: %s\n", config.OutputFile)
	
	// Generate integration code
	if *verbose {
		generateIntegrationCode(config.OutputFile)
	}
}

// readAndValidateCredentials reads and validates Google service account credentials
func readAndValidateCredentials(inputFile string, skipValidation bool) ([]byte, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	if !skipValidation {
		// Validate JSON structure for Google service account
		var serviceAccount map[string]interface{}
		if err := json.Unmarshal(data, &serviceAccount); err != nil {
			return nil, fmt.Errorf("invalid JSON format: %v", err)
		}

		// Validate required fields
		requiredFields := []string{
			"type", "project_id", "private_key_id", "private_key",
			"client_email", "client_id", "auth_uri", "token_uri",
		}

		for _, field := range requiredFields {
			if _, exists := serviceAccount[field]; !exists {
				return nil, fmt.Errorf("missing required field: %s", field)
			}
		}

		// Validate service account type
		if serviceAccount["type"] != "service_account" {
			return nil, fmt.Errorf("invalid credential type: %v (expected: service_account)", serviceAccount["type"])
		}

		// Validate private key format
		privateKey, ok := serviceAccount["private_key"].(string)
		if !ok {
			return nil, fmt.Errorf("private_key must be a string")
		}

		if !containsPrivateKeyMarkers(privateKey) {
			return nil, fmt.Errorf("invalid private key format")
		}
	}

	return data, nil
}

// containsPrivateKeyMarkers checks if the private key has valid PEM markers
func containsPrivateKeyMarkers(key string) bool {
	return len(key) > 50 && // Minimum reasonable length
		   (key[:27] == "-----BEGIN PRIVATE KEY-----" || 
		    key[:31] == "-----BEGIN RSA PRIVATE KEY-----")
}

// encryptCredentials encrypts the credential data using the security package
func encryptCredentials(credentials []byte, appSalt string) (*security.EncryptedPayload, error) {
	config := security.DefaultEncryptionConfig()
	
	// Validate encryption configuration
	if err := security.ValidateEncryptionConfig(config); err != nil {
		return nil, fmt.Errorf("invalid encryption config: %v", err)
	}

	// Encrypt credentials
	payload, err := security.EncryptCredentials(credentials, []byte(appSalt), config)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %v", err)
	}

	return payload, nil
}

// saveEncryptedPayload saves the encrypted payload to file
func saveEncryptedPayload(payload *security.EncryptedPayload, outputFile string) error {
	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Marshal payload to JSON
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(outputFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// loadBuildConfig loads build configuration from JSON file
func loadBuildConfig(configFile string) (*BuildConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config BuildConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// maskSensitiveData masks sensitive data for logging
func maskSensitiveData(data string) string {
	if len(data) <= 8 {
		return "***"
	}
	return data[:4] + "***" + data[len(data)-4:]
}

// generateIntegrationCode generates sample integration code
func generateIntegrationCode(outputFile string) {
	fmt.Printf("\n🔗 Integration Code Example:\n")
	fmt.Printf("```go\n")
	fmt.Printf("// Load encrypted credentials\n")
	fmt.Printf("data, err := os.ReadFile(\"%s\")\n", outputFile)
	fmt.Printf("if err != nil {\n")
	fmt.Printf("    return fmt.Errorf(\"failed to load credentials: %%v\", err)\n")
	fmt.Printf("}\n\n")
	
	fmt.Printf("var payload security.EncryptedPayload\n")
	fmt.Printf("if err := json.Unmarshal(data, &payload); err != nil {\n")
	fmt.Printf("    return fmt.Errorf(\"failed to unmarshal payload: %%v\", err)\n")
	fmt.Printf("}\n\n")
	
	fmt.Printf("// Decrypt credentials\n")
	fmt.Printf("appSalt := \"%s\"\n", DefaultAppSalt)
	fmt.Printf("credentials, err := security.DecryptCredentials(&payload, []byte(appSalt), nil)\n")
	fmt.Printf("if err != nil {\n")
	fmt.Printf("    return fmt.Errorf(\"credential decryption failed: %%v\", err)\n")
	fmt.Printf("}\n")
	fmt.Printf("defer credentials.Clear() // Always clear from memory\n\n")
	
	fmt.Printf("// Use credentials\n")
	fmt.Printf("credentialsJSON := credentials.Data()\n")
	fmt.Printf("// ... use credentialsJSON with Google APIs ...\n")
	fmt.Printf("```\n\n")
}