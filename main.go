// Package main provides a utility to find Go interface definitions in a directory.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Default regular expression to match Go interface definitions.
// Captures the interface name as the first submatch.
var defaultInterfaceRegex = regexp.MustCompile(`type\s+(\w+)\s+interface\s*\{`)

// InterfaceFinder defines the configuration for finding interfaces.
type InterfaceFinder struct {
	// Pattern is the regular expression used to find interface definitions.
	Pattern *regexp.Regexp
	// baseDir is the absolute path of the directory being searched
	baseDir string
}

// NewInterfaceFinder creates a new InterfaceFinder with default settings.
func NewInterfaceFinder() *InterfaceFinder {
	return &InterfaceFinder{
		Pattern: defaultInterfaceRegex,
	}
}

// FindInterfaces searches for interface definitions in the specified folder.
// It does not search in subfolders.
func (f *InterfaceFinder) FindInterfaces(folder string) ([]string, error) {
	var interfaces []string

	// Get absolute path of the folder to use as base directory for security validation
	absFolder, err := filepath.Abs(folder)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	f.baseDir = absFolder

	err = filepath.WalkDir(folder, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip subdirectories
		if info.IsDir() && path != folder {
			return filepath.SkipDir
		}

		// Process only Go files
		if !info.IsDir() && isGoFile(info.Name()) {
			foundInterfaces, err := f.extractInterfacesFromFile(path)
			if err != nil {
				return err
			}

			interfaces = append(interfaces, foundInterfaces...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return interfaces, nil
}

// isGoFile checks if a filename has a .go extension (case-insensitive).
func isGoFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".go")
}

// extractInterfacesFromFile reads a file and extracts interface names.
// It validates that the file path is within the base directory to prevent directory traversal attacks.
func (f *InterfaceFinder) extractInterfacesFromFile(filePath string) ([]string, error) {
	cleanPath := filepath.Clean(filePath)
	// Validate that the file path is within the base directory
	if !strings.HasPrefix(cleanPath, f.baseDir) {
		return nil, fmt.Errorf("file path %s (clean path: %s) is outside the base directory %s", filePath, cleanPath, f.baseDir)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	src := string(data)
	matches := f.Pattern.FindAllStringSubmatch(src, -1)

	var interfaces []string

	for _, match := range matches {
		if len(match) > 1 {
			interfaces = append(interfaces, match[1])
		}
	}

	return interfaces, nil
}

func main() {
	var flagPath string

	flag.StringVar(&flagPath, "path", ".", "path to search for interfaces")
	flag.Parse()

	finder := NewInterfaceFinder()

	interfaces, err := finder.FindInterfaces(flagPath)
	if err != nil {
		log.Fatalf("Error finding interfaces: %v", err)
	}

	fmt.Print(strings.Join(interfaces, " "))
}
