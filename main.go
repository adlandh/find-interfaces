// Package main provides a utility to find Go interface definitions in a directory.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

// InterfaceFinder defines the configuration for finding interfaces.
type InterfaceFinder struct {
	// baseDir is the absolute path of the directory being searched.
	baseDir string
}

// NewInterfaceFinder creates a new InterfaceFinder with default settings.
func NewInterfaceFinder() *InterfaceFinder {
	return &InterfaceFinder{}
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

	err = filepath.WalkDir(f.baseDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip subdirectories
		if info.IsDir() && path != f.baseDir {
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
	return strings.EqualFold(filepath.Ext(filename), ".go")
}

// extractInterfacesFromFile reads a file and extracts interface names.
// It validates that the file path is within the base directory to prevent directory traversal attacks.
func (f *InterfaceFinder) extractInterfacesFromFile(filePath string) ([]string, error) {
	cleanPath := filepath.Clean(filePath)

	if err := validatePathWithinBase(cleanPath, f.baseDir); err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, cleanPath, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %q: %w", cleanPath, err)
	}

	var interfaces []string

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				interfaces = append(interfaces, typeSpec.Name.Name)
			}
		}
	}

	return interfaces, nil
}

func validatePathWithinBase(path string, baseDir string) error {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q against base directory %q: %w", path, baseDir, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("file path %s is outside the base directory %s", path, baseDir)
	}

	return nil
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
