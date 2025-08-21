package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("Updating import paths in Go files...")

	// Update imports in api directory
	err := filepath.Walk("api", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", path, err)
			return nil
		}

		// Replace import paths
		newContent := string(content)
		originalContent := newContent

		// Update internal imports
		newContent = strings.ReplaceAll(newContent, `"dev/internal/`, `"ISXDailyReportsScrapper/api/internal/`)
		newContent = strings.ReplaceAll(newContent, `"dev/pkg/`, `"ISXDailyReportsScrapper/api/pkg/`)
		
		// Also handle relative imports that might exist
		newContent = strings.ReplaceAll(newContent, `"internal/`, `"ISXDailyReportsScrapper/api/internal/`)
		newContent = strings.ReplaceAll(newContent, `"pkg/`, `"ISXDailyReportsScrapper/api/pkg/`)

		// Only write if changed
		if newContent != originalContent {
			err = os.WriteFile(path, []byte(newContent), info.Mode())
			if err != nil {
				fmt.Printf("Error writing %s: %v\n", path, err)
			} else {
				fmt.Printf("Updated imports in: %s\n", path)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}

	// Update go.mod in api directory
	goModPath := "api/go.mod"
	if content, err := os.ReadFile(goModPath); err == nil {
		newContent := strings.ReplaceAll(string(content), "module dev", "module ISXDailyReportsScrapper/api")
		if err := os.WriteFile(goModPath, []byte(newContent), 0644); err == nil {
			fmt.Printf("Updated module name in go.mod\n")
		}
	}

	fmt.Println("\n✅ Import paths updated!")
}