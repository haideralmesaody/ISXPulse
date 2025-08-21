package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Migration mapping for directories
var migrations = map[string]string{
	"dev/cmd":      "api/cmd",
	"dev/internal": "api/internal",
	"dev/pkg":      "api/pkg",
	"dev/frontend": "web",
}

func main() {
	fmt.Println("Starting project structure migration...")

	// Step 1: Copy Go backend files
	fmt.Println("\n1. Migrating Go backend to api/...")
	if err := copyDir("dev/cmd", "api/cmd"); err != nil {
		fmt.Printf("Error copying cmd: %v\n", err)
	}
	if err := copyDir("dev/internal", "api/internal"); err != nil {
		fmt.Printf("Error copying internal: %v\n", err)
	}
	if err := copyDir("dev/pkg", "api/pkg"); err != nil {
		fmt.Printf("Error copying pkg: %v\n", err)
	}

	// Step 2: Copy Go module files
	fmt.Println("\n2. Copying Go module files...")
	if err := copyFile("dev/go.mod", "api/go.mod"); err != nil {
		fmt.Printf("Error copying go.mod: %v\n", err)
	}
	if err := copyFile("dev/go.sum", "api/go.sum"); err != nil {
		fmt.Printf("Error copying go.sum: %v\n", err)
	}

	// Step 3: Copy frontend files
	fmt.Println("\n3. Migrating frontend to web/...")
	frontendFiles := []string{
		"app", "components", "lib", "public", "styles",
		"package.json", "package-lock.json", "next.config.js",
		"tsconfig.json", "tailwind.config.js", "postcss.config.js",
		"jest.config.js", "components.json",
	}
	for _, file := range frontendFiles {
		src := filepath.Join("dev/frontend", file)
		dst := filepath.Join("web", file)
		
		info, err := os.Stat(src)
		if err != nil {
			continue // Skip if doesn't exist
		}
		
		if info.IsDir() {
			if err := copyDir(src, dst); err != nil {
				fmt.Printf("Error copying %s: %v\n", file, err)
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				fmt.Printf("Error copying %s: %v\n", file, err)
			}
		}
	}

	// Step 4: Move test files
	fmt.Println("\n4. Organizing test files...")
	if err := copyDir("dev/frontend/__tests__", "web/tests"); err != nil {
		fmt.Printf("Error copying frontend tests: %v\n", err)
	}
	if err := copyDir("dev/test", "api/tests"); err != nil {
		fmt.Printf("Error copying backend tests: %v\n", err)
	}

	fmt.Println("\n✅ Migration completed!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Update import paths in Go files")
	fmt.Println("2. Update build.go for new structure")
	fmt.Println("3. Test the build process")
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dst)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// copyDir recursively copies a directory tree
func copyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	// Create destination directory
	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Skip certain files/directories
		if shouldSkip(entry.Name()) {
			continue
		}

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				fmt.Printf("Warning: failed to copy directory %s: %v\n", srcPath, err)
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				fmt.Printf("Warning: failed to copy file %s: %v\n", srcPath, err)
			}
		}
	}

	return nil
}

// shouldSkip returns true if the file/directory should be skipped
func shouldSkip(name string) bool {
	skipList := []string{
		".git", "node_modules", ".next", "out", "dist",
		"coverage", "*.log", "*.out", ".DS_Store",
	}

	for _, skip := range skipList {
		if strings.Contains(skip, "*") {
			// Simple glob matching
			pattern := strings.ReplaceAll(skip, "*", "")
			if strings.HasSuffix(name, pattern) {
				return true
			}
		} else if name == skip {
			return true
		}
	}

	return false
}