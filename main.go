// Package main provides a utility to find Go interface definitions in a directory.
package main

import (
	"errors"
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

type fileParseError struct {
	err  error
	path string
}

func (e *fileParseError) Error() string {
	return fmt.Sprintf("failed to parse Go file %q: %v", e.path, e.err)
}

func (e *fileParseError) Unwrap() error {
	return e.err
}

// NewInterfaceFinder creates a new InterfaceFinder with default settings.
func NewInterfaceFinder() *InterfaceFinder {
	return &InterfaceFinder{}
}

// FindInterfaces searches for interface definitions in the specified folder.
// It does not search in subfolders.
func (f *InterfaceFinder) FindInterfaces(folder string) ([]string, error) {
	var (
		interfaces  []string
		parseErrors []error
	)

	// Get absolute path of the folder to use as base directory for security validation
	absFolder, err := filepath.Abs(folder)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	f.baseDir = absFolder

	err = filepath.WalkDir(f.baseDir, func(path string, info fs.DirEntry, walkErr error) error {
		return f.visitPath(path, info, walkErr, &interfaces, &parseErrors)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(interfaces) == 0 && len(parseErrors) > 0 {
		return nil, fmt.Errorf("failed to parse any Go files successfully: %w", parseErrors[0])
	}

	return interfaces, nil
}

func (f *InterfaceFinder) visitPath(path string, info fs.DirEntry, walkErr error, interfaces *[]string, parseErrors *[]error) error {
	if walkErr != nil {
		return walkErr
	}

	if shouldSkipDir(path, info, f.baseDir) {
		return filepath.SkipDir
	}

	if !shouldProcessFile(info) {
		return nil
	}

	return f.collectInterfaces(path, interfaces, parseErrors)
}

func shouldSkipDir(path string, info fs.DirEntry, baseDir string) bool {
	return info.IsDir() && path != baseDir
}

func shouldProcessFile(info fs.DirEntry) bool {
	return !info.IsDir() && isGoFile(info.Name())
}

func (f *InterfaceFinder) collectInterfaces(path string, interfaces *[]string, parseErrors *[]error) error {
	foundInterfaces, err := f.extractInterfacesFromFile(path)
	*interfaces = append(*interfaces, foundInterfaces...)

	if err == nil {
		return nil
	}

	if parseErr := asFileParseError(err); parseErr != nil {
		*parseErrors = append(*parseErrors, parseErr)
		return nil
	}

	return err
}

func asFileParseError(err error) *fileParseError {
	var parseErr *fileParseError
	if errors.As(err, &parseErr) {
		return parseErr
	}

	return nil
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

	file, err := parser.ParseFile(fset, cleanPath, nil, parser.SkipObjectResolution|parser.AllErrors)
	if err != nil && file == nil {
		return nil, &fileParseError{path: cleanPath, err: err}
	}

	interfaces := extractInterfacesFromAST(file)

	if err != nil {
		return interfaces, &fileParseError{path: cleanPath, err: err}
	}

	return interfaces, nil
}

func extractInterfacesFromAST(file *ast.File) []string {
	interfaces := make([]string, 0, len(file.Decls))

	for _, decl := range file.Decls {
		interfaces = append(interfaces, extractInterfacesFromDecl(decl)...)
	}

	return interfaces
}

func extractInterfacesFromDecl(decl ast.Decl) []string {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.TYPE {
		return nil
	}

	var interfaces []string

	for _, spec := range genDecl.Specs {
		if name, ok := extractInterfaceName(spec); ok {
			interfaces = append(interfaces, name)
		}
	}

	return interfaces
}

func extractInterfaceName(spec ast.Spec) (string, bool) {
	typeSpec, ok := spec.(*ast.TypeSpec)
	if !ok {
		return "", false
	}

	if _, ok := typeSpec.Type.(*ast.InterfaceType); !ok {
		return "", false
	}

	return typeSpec.Name.Name, true
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
