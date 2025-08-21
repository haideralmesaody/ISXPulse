package updater

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpdater(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		repoURL        string
		expectError    bool
	}{
		{
			name:           "valid parameters",
			currentVersion: "v1.0.0",
			repoURL:        "https://github.com/user/repo",
			expectError:    false,
		},
		{
			name:           "empty version",
			currentVersion: "",
			repoURL:        "https://github.com/user/repo",
			expectError:    false, // Should still work
		},
		{
			name:           "empty repo URL",
			currentVersion: "v1.0.0",
			repoURL:        "",
			expectError:    false, // Should still work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater, err := NewUpdater(tt.currentVersion, tt.repoURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, updater)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updater)
				assert.Equal(t, tt.currentVersion, updater.currentVersion)
				assert.Equal(t, tt.repoURL, updater.repoURL)
				assert.NotEmpty(t, updater.executablePath)
			}
		})
	}
}

func TestCheckForUpdates(t *testing.T) {
	tests := []struct {
		name            string
		currentVersion  string
		serverResponse  interface{}
		statusCode      int
		expectedUpdate  *UpdateInfo
		expectError     bool
		description     string
	}{
		{
			name:           "update available",
			currentVersion: "v1.0.0",
			serverResponse: Release{
				TagName: "v1.1.0",
				Name:    "Version 1.1.0 - Bug fixes and improvements",
				Assets: []Asset{
					{
						Name:               "app-windows.zip",
						BrowserDownloadURL: "https://github.com/user/repo/releases/download/v1.1.0/app-windows.zip",
						Size:               1024000,
					},
				},
			},
			statusCode: http.StatusOK,
			expectedUpdate: &UpdateInfo{
				CurrentVersion: "v1.0.0",
				LatestVersion:  "v1.1.0",
				UpdateURL:      "https://github.com/user/repo/releases/download/v1.1.0/app-windows.zip",
				ReleaseNotes:   "Version 1.1.0 - Bug fixes and improvements",
				Size:           1024000,
			},
			expectError: false,
			description: "Should detect available update",
		},
		{
			name:           "no update needed",
			currentVersion: "v1.1.0",
			serverResponse: Release{
				TagName: "v1.1.0",
				Name:    "Version 1.1.0",
				Assets: []Asset{
					{
						Name:               "app-windows.zip",
						BrowserDownloadURL: "https://github.com/user/repo/releases/download/v1.1.0/app-windows.zip",
						Size:               1024000,
					},
				},
			},
			statusCode:     http.StatusOK,
			expectedUpdate: nil,
			expectError:    false,
			description:    "Should return nil when no update needed",
		},
		{
			name:           "GitHub API error",
			currentVersion: "v1.0.0",
			serverResponse: nil,
			statusCode:     http.StatusInternalServerError,
			expectedUpdate: nil,
			expectError:    true,
			description:    "Should handle GitHub API errors",
		},
		{
			name:           "no suitable asset",
			currentVersion: "v1.0.0",
			serverResponse: Release{
				TagName: "v1.1.0",
				Name:    "Version 1.1.0",
				Assets: []Asset{
					{
						Name:               "app-linux.zip",
						BrowserDownloadURL: "https://github.com/user/repo/releases/download/v1.1.0/app-linux.zip",
						Size:               1024000,
					},
				},
			},
			statusCode:     http.StatusOK,
			expectedUpdate: nil,
			expectError:    true,
			description:    "Should error when no suitable asset found",
		},
		{
			name:           "malformed JSON response",
			currentVersion: "v1.0.0",
			serverResponse: "invalid json",
			statusCode:     http.StatusOK,
			expectedUpdate: nil,
			expectError:    true,
			description:    "Should handle malformed JSON response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				
				if tt.statusCode == http.StatusOK && tt.serverResponse != nil {
					if release, ok := tt.serverResponse.(Release); ok {
						json.NewEncoder(w).Encode(release)
					} else {
						w.Write([]byte(tt.serverResponse.(string)))
					}
				}
			}))
			defer server.Close()

			// Create updater with test server URL
			repoURL := strings.Replace(server.URL, "http://", "https://github.com/", 1)
			updater, err := NewUpdater(tt.currentVersion, repoURL)
			require.NoError(t, err)
			
			// Override the repo URL to point to test server
			updater.repoURL = server.URL

			updateInfo, err := updater.CheckForUpdates()

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, updateInfo)
			} else {
				assert.NoError(t, err, tt.description)
				
				if tt.expectedUpdate == nil {
					assert.Nil(t, updateInfo, tt.description)
				} else {
					assert.NotNil(t, updateInfo, tt.description)
					assert.Equal(t, tt.expectedUpdate.CurrentVersion, updateInfo.CurrentVersion)
					assert.Equal(t, tt.expectedUpdate.LatestVersion, updateInfo.LatestVersion)
					assert.Equal(t, tt.expectedUpdate.UpdateURL, updateInfo.UpdateURL)
					assert.Equal(t, tt.expectedUpdate.ReleaseNotes, updateInfo.ReleaseNotes)
					assert.Equal(t, tt.expectedUpdate.Size, updateInfo.Size)
				}
			}
		})
	}
}

func TestGetAssetName(t *testing.T) {
	tests := []struct {
		name         string
		goos         string
		expectedName string
	}{
		{
			name:         "Windows",
			goos:         "windows",
			expectedName: "windows",
		},
		{
			name:         "macOS",
			goos:         "darwin",
			expectedName: "macos",
		},
		{
			name:         "Linux",
			goos:         "linux",
			expectedName: "linux",
		},
		{
			name:         "Unknown OS",
			goos:         "unknown",
			expectedName: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := &Updater{}
			
			// Note: We can't actually change runtime.GOOS in tests, so we test with current OS
			// This is a limitation of testing the getAssetName function
			
			assetName := updater.getAssetName()
			
			// Verify that it returns a non-empty string for the current OS
			assert.NotEmpty(t, assetName)
			
			// For the current OS, verify expected mapping
			switch runtime.GOOS {
			case "windows":
				assert.Equal(t, "windows", assetName)
			case "darwin":
				assert.Equal(t, "macos", assetName)
			case "linux":
				assert.Equal(t, "linux", assetName)
			default:
				assert.Equal(t, runtime.GOOS, assetName)
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectError    bool
		description    string
	}{
		{
			name:           "successful download",
			serverResponse: "test file content for download",
			statusCode:     http.StatusOK,
			expectError:    false,
			description:    "Should download file successfully",
		},
		{
			name:           "server error",
			serverResponse: "",
			statusCode:     http.StatusInternalServerError,
			expectError:    true,
			description:    "Should handle server errors",
		},
		{
			name:           "not found",
			serverResponse: "",
			statusCode:     http.StatusNotFound,
			expectError:    true,
			description:    "Should handle file not found",
		},
		{
			name:           "large file download",
			serverResponse: strings.Repeat("Large content test. ", 10000),
			statusCode:     http.StatusOK,
			expectError:    false,
			description:    "Should handle large file downloads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(tt.serverResponse))
				}
			}))
			defer server.Close()

			updater := &Updater{}
			
			// Create temporary file for download
			tmpDir := t.TempDir()
			downloadPath := filepath.Join(tmpDir, "downloaded_file.zip")

			err := updater.downloadFile(server.URL, downloadPath)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				
				// Verify file was downloaded
				assert.FileExists(t, downloadPath)
				
				// Verify content
				content, err := os.ReadFile(downloadPath)
				assert.NoError(t, err)
				assert.Equal(t, tt.serverResponse, string(content))
			}
		})
	}
}

func TestExtractZip(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string // filename -> content
		expectError bool
		description string
	}{
		{
			name: "simple zip extraction",
			files: map[string]string{
				"file1.txt":     "Content of file 1",
				"file2.txt":     "Content of file 2",
				"subdir/file3.txt": "Content of file 3 in subdirectory",
			},
			expectError: false,
			description: "Should extract zip file successfully",
		},
		{
			name: "empty zip file",
			files: map[string]string{},
			expectError: false,
			description: "Should handle empty zip file",
		},
		{
			name: "zip with directories",
			files: map[string]string{
				"dir1/":          "", // Directory entry
				"dir1/file.txt":  "File in directory",
				"dir2/subdir/":   "", // Nested directory
				"dir2/subdir/nested.txt": "Nested file",
			},
			expectError: false,
			description: "Should handle directories in zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			updater := &Updater{}

			// Create test zip file
			zipPath := filepath.Join(tmpDir, "test.zip")
			err := createTestZip(zipPath, tt.files)
			require.NoError(t, err)

			// Test extraction
			extractDir := filepath.Join(tmpDir, "extracted")
			err = updater.extractZip(zipPath, extractDir)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				
				// Verify extracted files
				for filename, expectedContent := range tt.files {
					if strings.HasSuffix(filename, "/") {
						// Directory entry - verify directory exists
						dirPath := filepath.Join(extractDir, filename)
						info, err := os.Stat(dirPath)
						assert.NoError(t, err)
						assert.True(t, info.IsDir())
					} else {
						// File entry - verify file content
						filePath := filepath.Join(extractDir, filename)
						content, err := os.ReadFile(filePath)
						assert.NoError(t, err)
						assert.Equal(t, expectedContent, string(content))
					}
				}
			}
		})
	}
}

func TestFindExecutable(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		expectedFound  bool
		description    string
	}{
		{
			name: "Windows executable found",
			files: []string{
				"web.exe",
				"other.txt",
				"readme.md",
			},
			expectedFound: true,
			description:   "Should find Windows executable",
		},
		{
			name: "Unix executable found",
			files: []string{
				"web",
				"other.txt", 
				"readme.md",
			},
			expectedFound: true,
			description:   "Should find Unix executable",
		},
		{
			name: "no executable found",
			files: []string{
				"other.txt",
				"readme.md",
				"config.json",
			},
			expectedFound: false,
			description:   "Should return error when no executable found",
		},
		{
			name: "executable in subdirectory",
			files: []string{
				"subdir/web.exe",
				"other.txt",
			},
			expectedFound: true,
			description:   "Should find executable in subdirectory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			updater := &Updater{}

			// Create test files
			for _, filename := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				
				// Create directory if needed
				dir := filepath.Dir(filePath)
				if dir != tmpDir {
					err := os.MkdirAll(dir, 0755)
					require.NoError(t, err)
				}
				
				// Create file
				err := os.WriteFile(filePath, []byte("test executable"), 0755)
				require.NoError(t, err)
			}

			executablePath, err := updater.findExecutable(tmpDir)

			if tt.expectedFound {
				assert.NoError(t, err, tt.description)
				assert.NotEmpty(t, executablePath)
				assert.FileExists(t, executablePath)
			} else {
				assert.Error(t, err, tt.description)
				assert.Empty(t, executablePath)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		description string
	}{
		{
			name:        "simple text file",
			content:     "Simple text content",
			expectError: false,
			description: "Should copy text file successfully",
		},
		{
			name:        "binary content",
			content:     string([]byte{0x00, 0x01, 0x02, 0xFF}),
			expectError: false,
			description: "Should copy binary content successfully",
		},
		{
			name:        "empty file",
			content:     "",
			expectError: false,
			description: "Should copy empty file successfully",
		},
		{
			name:        "large file",
			content:     strings.Repeat("Large content test. ", 5000),
			expectError: false,
			description: "Should copy large file successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			updater := &Updater{}

			// Create source file
			srcPath := filepath.Join(tmpDir, "source.txt")
			err := os.WriteFile(srcPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			// Test copy
			dstPath := filepath.Join(tmpDir, "destination.txt")
			err = updater.copyFile(srcPath, dstPath)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				
				// Verify destination file
				assert.FileExists(t, dstPath)
				
				// Verify content
				copiedContent, err := os.ReadFile(dstPath)
				assert.NoError(t, err)
				assert.Equal(t, tt.content, string(copiedContent))
			}
		})
	}
}

func TestPerformUpdate(t *testing.T) {
	t.Run("successful update flow", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create a mock executable to be "updated"
		executablePath := filepath.Join(tmpDir, "web.exe")
		err := os.WriteFile(executablePath, []byte("old version"), 0755)
		require.NoError(t, err)

		// Create test server for download
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a zip file with new executable
			zipData := createTestUpdateZip(t)
			w.Header().Set("Content-Type", "application/zip")
			w.Write(zipData)
		}))
		defer server.Close()

		updater := &Updater{
			currentVersion: "v1.0.0",
			repoURL:        "https://github.com/test/repo",
			executablePath: executablePath,
		}

		updateInfo := &UpdateInfo{
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.1.0",
			UpdateURL:      server.URL,
			ReleaseNotes:   "Test update",
			Size:           1024,
		}

		err = updater.PerformUpdate(updateInfo)
		assert.NoError(t, err)

		// Verify executable was updated
		newContent, err := os.ReadFile(executablePath)
		assert.NoError(t, err)
		assert.Equal(t, "new version", string(newContent))
	})

	t.Run("download failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		executablePath := filepath.Join(tmpDir, "web.exe")
		
		updater := &Updater{
			currentVersion: "v1.0.0",
			executablePath: executablePath,
		}

		updateInfo := &UpdateInfo{
			UpdateURL: "http://invalid-url-that-does-not-exist.com/file.zip",
		}

		err := updater.PerformUpdate(updateInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download update")
	})
}

func TestAutoUpdateChecker(t *testing.T) {
	t.Run("create auto update checker", func(t *testing.T) {
		updater := &Updater{
			currentVersion: "v1.0.0",
			repoURL:        "https://github.com/test/repo",
		}
		
		callback := func(info *UpdateInfo) bool {
			return false // Don't install update
		}
		
		checker := NewAutoUpdateChecker(updater, time.Minute, callback)
		assert.NotNil(t, checker)
		assert.Equal(t, updater, checker.updater)
		assert.Equal(t, time.Minute, checker.interval)
		assert.NotNil(t, checker.callback)
	})
}

// Helper function to create a test zip file
func createTestZip(zipPath string, files map[string]string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for filename, content := range files {
		if strings.HasSuffix(filename, "/") {
			// Directory entry
			_, err := zipWriter.Create(filename)
			if err != nil {
				return err
			}
		} else {
			// File entry
			writer, err := zipWriter.Create(filename)
			if err != nil {
				return err
			}
			
			_, err = writer.Write([]byte(content))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper function to create a test update zip
func createTestUpdateZip(t *testing.T) []byte {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "update.zip")
	
	files := map[string]string{
		"web.exe": "new version",
		"config.json": `{"version": "v1.1.0"}`,
	}
	
	err := createTestZip(zipPath, files)
	require.NoError(t, err)
	
	data, err := os.ReadFile(zipPath)
	require.NoError(t, err)
	
	return data
}

// Benchmark update check operation
func BenchmarkCheckForUpdates(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := Release{
			TagName: "v1.1.0",
			Name:    "Test Release",
			Assets: []Asset{
				{
					Name:               "app-windows.zip",
					BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.1.0/app-windows.zip",
					Size:               1024000,
				},
			},
		}
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	updater := &Updater{
		currentVersion: "v1.0.0",
		repoURL:        server.URL,
		executablePath: "/test/path",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = updater.CheckForUpdates()
	}
}

// Test concurrent update checks
func TestConcurrentUpdateChecks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := Release{
			TagName: "v1.1.0",
			Name:    "Test Release",
			Assets: []Asset{
				{
					Name:               "app-windows.zip",
					BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.1.0/app-windows.zip",
					Size:               1024000,
				},
			},
		}
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	updater := &Updater{
		currentVersion: "v1.0.0",
		repoURL:        server.URL,
		executablePath: "/test/path",
	}

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			
			updateInfo, err := updater.CheckForUpdates()
			assert.NoError(t, err)
			assert.NotNil(t, updateInfo)
		}()
	}

	// Wait for all checks to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}