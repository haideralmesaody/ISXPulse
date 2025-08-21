package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run migrate-logging.go <directory>")
		fmt.Println("Example: go run migrate-logging.go ../dev")
		os.Exit(1)
	}

	rootDir := os.Args[1]
	fmt.Printf("🔧 Starting logging migration in: %s\n", rootDir)

	var filesProcessed, changesTotal int

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor and node_modules
		if strings.Contains(path, "vendor/") || strings.Contains(path, "node_modules/") {
			return nil
		}

		changes, err := processFile(path)
		if err != nil {
			fmt.Printf("❌ Error processing %s: %v\n", path, err)
			return nil // Continue processing other files
		}

		if changes > 0 {
			filesProcessed++
			changesTotal += changes
			fmt.Printf("✅ %s: %d changes\n", path, changes)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("❌ Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🎉 Migration complete! %d files processed, %d total changes\n", filesProcessed, changesTotal)
}

func processFile(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	original := string(content)
	modified := original
	changes := 0

	// Migration patterns
	patterns := []struct {
		pattern     *regexp.Regexp
		replacement string
		description string
	}{
		{
			pattern:     regexp.MustCompile(`fmt\.Println\("([^"]+)"\)`),
			replacement: `slog.Info("$1")`,
			description: "fmt.Println string literal",
		},
		{
			pattern:     regexp.MustCompile(`fmt\.Println\(([^)]+)\)`),
			replacement: `slog.Info(fmt.Sprintf("%v", $1))`,
			description: "fmt.Println variable",
		},
		{
			pattern:     regexp.MustCompile(`fmt\.Printf\("([^"]*%[sd][^"]*)", ([^)]+)\)`),
			replacement: `slog.Info("$1", $2)`,
			description: "fmt.Printf with format",
		},
		{
			pattern:     regexp.MustCompile(`log\.Println\("([^"]+)"\)`),
			replacement: `slog.Info("$1")`,
			description: "log.Println string literal",
		},
		{
			pattern:     regexp.MustCompile(`log\.Println\(([^)]+)\)`),
			replacement: `slog.Info(fmt.Sprintf("%v", $1))`,
			description: "log.Println variable",
		},
		{
			pattern:     regexp.MustCompile(`log\.Printf\("([^"]*)", ([^)]+)\)`),
			replacement: `slog.Info("$1", $2)`,
			description: "log.Printf with format",
		},
		{
			pattern:     regexp.MustCompile(`log\.Fatalf\("([^"]*)", ([^)]+)\)`),
			replacement: `slog.Error("$1", $2); os.Exit(1)`,
			description: "log.Fatalf with format",
		},
		{
			pattern:     regexp.MustCompile(`log\.Fatal\("([^"]+)"\)`),
			replacement: `slog.Error("$1"); os.Exit(1)`,
			description: "log.Fatal string literal",
		},
		{
			pattern:     regexp.MustCompile(`log\.Fatal\(([^)]+)\)`),
			replacement: `slog.Error(fmt.Sprintf("%v", $1)); os.Exit(1)`,
			description: "log.Fatal variable",
		},
	}

	// Apply all patterns
	for _, p := range patterns {
		before := modified
		modified = p.pattern.ReplaceAllString(modified, p.replacement)
		if modified != before {
			changes++
		}
	}

	// Special handling for complex fmt.Printf patterns
	modified = handleComplexPrintf(modified, &changes)

	// Add required imports if we made changes
	if modified != original && changes > 0 {
		modified = ensureImports(modified)
	}

	// Write back if changed
	if modified != original {
		err = os.WriteFile(path, []byte(modified), 0644)
		if err != nil {
			return 0, err
		}
	}

	return changes, nil
}

func handleComplexPrintf(content string, changes *int) string {
	// Handle fmt.Printf patterns that need structured logging
	patterns := []struct {
		regex       *regexp.Regexp
		replacement func(matches []string) string
	}{
		{
			// fmt.Printf("Port: %d", port) -> slog.Info("Server configuration", "port", port)
			regex: regexp.MustCompile(`fmt\.Printf\("([^"]*): %([sd])", ([^)]+)\)`),
			replacement: func(matches []string) string {
				*changes++
				return fmt.Sprintf(`slog.Info("%s", "%s", %s)`, 
					strings.ToLower(matches[1]), 
					strings.ToLower(matches[1]), 
					matches[3])
			},
		},
		{
			// fmt.Printf("Error: %v", err) -> slog.Error("Operation failed", "error", err)
			regex: regexp.MustCompile(`fmt\.Printf\("Error: %v", ([^)]+)\)`),
			replacement: func(matches []string) string {
				*changes++
				return fmt.Sprintf(`slog.Error("Operation failed", "error", %s)`, matches[1])
			},
		},
	}

	for _, p := range patterns {
		content = p.regex.ReplaceAllStringFunc(content, func(match string) string {
			matches := p.regex.FindStringSubmatch(match)
			if len(matches) > 0 {
				return p.replacement(matches)
			}
			return match
		})
	}

	return content
}

func ensureImports(content string) string {
	// Check if slog import exists
	if !strings.Contains(content, `"log/slog"`) {
		// Find import block and add slog
		importRegex := regexp.MustCompile(`import \(([\s\S]*?)\)`)
		if importRegex.MatchString(content) {
			content = importRegex.ReplaceAllStringFunc(content, func(importBlock string) string {
				if !strings.Contains(importBlock, `"log/slog"`) {
					// Add slog import to existing import block
					return strings.Replace(importBlock, ")", "\t\"log/slog\"\n)", 1)
				}
				return importBlock
			})
		} else {
			// Check for single imports and add after package declaration
			packageRegex := regexp.MustCompile(`(package \w+\n)`)
			if packageRegex.MatchString(content) {
				content = packageRegex.ReplaceAllString(content, "$1\nimport \"log/slog\"\n")
			}
		}
	}

	// Ensure os import if we added os.Exit
	if strings.Contains(content, "os.Exit") && !strings.Contains(content, `"os"`) {
		importRegex := regexp.MustCompile(`import \(([\s\S]*?)\)`)
		if importRegex.MatchString(content) {
			content = importRegex.ReplaceAllStringFunc(content, func(importBlock string) string {
				if !strings.Contains(importBlock, `"os"`) {
					return strings.Replace(importBlock, ")", "\t\"os\"\n)", 1)
				}
				return importBlock
			})
		}
	}

	// Ensure fmt import if we added fmt.Sprintf
	if strings.Contains(content, "fmt.Sprintf") && !strings.Contains(content, `"fmt"`) {
		importRegex := regexp.MustCompile(`import \(([\s\S]*?)\)`)
		if importRegex.MatchString(content) {
			content = importRegex.ReplaceAllStringFunc(content, func(importBlock string) string {
				if !strings.Contains(importBlock, `"fmt"`) {
					return strings.Replace(importBlock, ")", "\t\"fmt\"\n)", 1)
				}
				return importBlock
			})
		}
	}

	return content
}