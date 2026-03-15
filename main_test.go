package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestFindInterfaces_NoGoFiles(t *testing.T) {
	tempDir := t.TempDir()
	writeTestFile(t, filepath.Join(tempDir, "readme.txt"), "not go code")

	finder := NewInterfaceFinder()
	interfaces, err := finder.FindInterfaces(tempDir)
	require.NoError(t, err)
	require.Empty(t, interfaces)
}

func TestFindInterfaces_FindsTopLevelInterfacesOnly(t *testing.T) {
	tempDir := t.TempDir()

	writeTestFile(t, filepath.Join(tempDir, "first.go"), `package test
type Reader interface { Read() }
`)
	writeTestFile(t, filepath.Join(tempDir, "second.go"), `package test
type Writer[T any] interface { Write(T) }
type Closer interface { Close() }
`)

	nestedDir := filepath.Join(tempDir, "nested")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))
	writeTestFile(t, filepath.Join(nestedDir, "ignored.go"), `package test
type ShouldNotBeFound interface { Noop() }
`)

	finder := NewInterfaceFinder()
	interfacesFound, err := finder.FindInterfaces(tempDir)
	require.NoError(t, err)

	sort.Strings(interfacesFound)
	require.Equal(t, []string{"Closer", "Reader", "Writer"}, interfacesFound)
}

func TestFindInterfaces_FindsGenericInterfacesWithoutSpaceAndMultilineParams(t *testing.T) {
	tempDir := t.TempDir()

	writeTestFile(t, filepath.Join(tempDir, "generic_compact.go"), `package test
type Compact[T any]interface { Use(T) }
`)

	writeTestFile(t, filepath.Join(tempDir, "generic_multiline.go"), `package test
type Multiline[
	P interface {
		~int | ~int64
	},
]interface {
	Use(P)
}
`)
	writeTestFile(t, filepath.Join(tempDir, "not_an_interface_multiline.go"), `package test
type HandlerFunc[T any] func(ctx context.Context, data T, headers map[string]any) error

type ProducerInterface[T map[string][]any] interface {
	Publish(ctx context.Context, routingKey string, message T) (err error)
	PublishWithHeaders(ctx context.Context, routingKey string, message T, headers map[string]any) (err error)
	RoutingKey() string
}`)

	writeTestFile(t, filepath.Join(tempDir, "уникод.go"), `package test
type ИнтерфейсНаРусском[T []byte]interface { Use(T) }
`)

	finder := NewInterfaceFinder()
	interfacesFound, err := finder.FindInterfaces(tempDir)
	require.NoError(t, err)

	sort.Strings(interfacesFound)
	require.Equal(t, []string{"Compact", "Multiline", "ProducerInterface", "ИнтерфейсНаРусском"}, interfacesFound)
}

func TestFindInterfaces_IgnoresInterfaceTextInCommentsAndStrings(t *testing.T) {
	tempDir := t.TempDir()

	writeTestFile(t, filepath.Join(tempDir, "input.go"), `package test

// type Fake interface { Ignored() }
const example = "type AlsoFake interface { Ignored() }"

type Real interface {
	Do()
}
`)

	finder := NewInterfaceFinder()
	interfacesFound, err := finder.FindInterfaces(tempDir)
	require.NoError(t, err)
	require.Equal(t, []string{"Real"}, interfacesFound)
}

func TestFindInterfaces_ContinuesWhenAFileHasParseErrors(t *testing.T) {
	tempDir := t.TempDir()

	writeTestFile(t, filepath.Join(tempDir, "valid.go"), `package test
type Reader interface { Read() }
`)
	writeTestFile(t, filepath.Join(tempDir, "broken.go"), `package test
func (
`)

	finder := NewInterfaceFinder()
	interfacesFound, err := finder.FindInterfaces(tempDir)
	require.NoError(t, err)
	require.Equal(t, []string{"Reader"}, interfacesFound)
}

func TestFindInterfaces_ReturnsErrorWhenAllGoFilesFailToParse(t *testing.T) {
	tempDir := t.TempDir()

	writeTestFile(t, filepath.Join(tempDir, "broken.go"), `package test
func (
`)

	finder := NewInterfaceFinder()
	interfacesFound, err := finder.FindInterfaces(tempDir)
	require.Error(t, err)
	require.Empty(t, interfacesFound)
}

func TestFindInterfaces_NonExistentDirectory(t *testing.T) {
	finder := NewInterfaceFinder()
	interfaces, err := finder.FindInterfaces(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
	require.Empty(t, interfaces)
}

func TestExtractInterfacesFromFile_RejectsPathOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()

	outsideFile := filepath.Join(outsideDir, "outside.go")
	writeTestFile(t, outsideFile, `package test
type Outside interface { Noop() }
`)

	finder := NewInterfaceFinder()
	finder.baseDir = baseDir

	interfaces, err := finder.extractInterfacesFromFile(outsideFile)
	require.Error(t, err)
	require.Empty(t, interfaces)
}

func TestExtractInterfacesFromFile_RejectsSiblingPathWithSharedPrefix(t *testing.T) {
	parentDir := t.TempDir()
	baseDir := filepath.Join(parentDir, "repo")
	sharedPrefixDir := filepath.Join(parentDir, "repo-other")

	require.NoError(t, os.MkdirAll(baseDir, 0o755))
	require.NoError(t, os.MkdirAll(sharedPrefixDir, 0o755))

	outsideFile := filepath.Join(sharedPrefixDir, "outside.go")
	writeTestFile(t, outsideFile, `package test
type Outside interface { Noop() }
`)

	finder := NewInterfaceFinder()
	finder.baseDir = baseDir

	interfaces, err := finder.extractInterfacesFromFile(outsideFile)
	require.Error(t, err)
	require.Empty(t, interfaces)
}

func TestIsGoFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Go file", "test.go", true},
		{"Uppercase extension", "test.GO", true},
		{"Non-Go file", "test.txt", false},
		{"File without extension", "test", false},
		{"Go file with path", "/path/to/test.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGoFile(tt.filename)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFileParseError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	parseErr := &fileParseError{path: "test.go", err: innerErr}

	require.Equal(t, innerErr, parseErr.Unwrap())
}

func TestExtractInterfacesFromDecl_NonTypeDeclaration(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", "package test; func Foo() {}", 0)
	require.NoError(t, err)

	interfaces := extractInterfacesFromAST(file)
	require.Empty(t, interfaces)
}

func TestExtractInterfaceName_NonInterfaceType(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", "package test; type MyInt int", 0)
	require.NoError(t, err)

	var found []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			if name, ok := extractInterfaceName(spec); ok {
				found = append(found, name)
			}
		}
	}

	require.Empty(t, found)
}

func TestAsFileParseError_Nil(t *testing.T) {
	result := asFileParseError(errors.New("some error"))
	require.Nil(t, result)
}

func TestAsFileParseError_Wrapped(t *testing.T) {
	innerErr := errors.New("inner")
	parseErr := &fileParseError{path: "test.go", err: innerErr}
	wrapped := fmt.Errorf("wrapped: %w", parseErr)

	result := asFileParseError(wrapped)
	require.Equal(t, parseErr, result)
}

func TestValidatePathWithinBase_RelError(t *testing.T) {
	err := validatePathWithinBase("/nonexistent/path/file.go", "/different/base")
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside the base directory")
}

func TestExtractInterfaceName_AliasType(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", "package test; type MyInt = int", 0)
	require.NoError(t, err)

	var found []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			if name, ok := extractInterfaceName(spec); ok {
				found = append(found, name)
			}
		}
	}

	require.Empty(t, found)
}

func TestExtractInterfaceName_StructType(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", "package test; type MyStruct struct{}", 0)
	require.NoError(t, err)

	var found []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			if name, ok := extractInterfaceName(spec); ok {
				found = append(found, name)
			}
		}
	}

	require.Empty(t, found)
}

func TestExtractInterfacesFromFile_ValidationError(t *testing.T) {
	baseDir := t.TempDir()

	// Create a file in baseDir
	filePath := filepath.Join(baseDir, "test.go")
	writeTestFile(t, filePath, `package test
type Reader interface { Read() }
`)

	// Now set baseDir to a different path to trigger validation error
	finder := NewInterfaceFinder()
	finder.baseDir = t.TempDir() // Different directory

	interfaces, err := finder.extractInterfacesFromFile(filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside the base directory")
	require.Empty(t, interfaces)
}

func TestCollectInterfaces_ReturnsNonFileParseError(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create a file outside baseDir
	outsideFile := filepath.Join(outsideDir, "outside.go")
	writeTestFile(t, outsideFile, `package test
type Outside interface { Noop() }
`)

	finder := NewInterfaceFinder()
	finder.baseDir = baseDir

	var interfaces []string
	var parseErrors []error

	// This should return an error that's not a fileParseError
	err := finder.collectInterfaces(outsideFile, &interfaces, &parseErrors)
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside the base directory")
}

func TestExtractInterfacesFromFile_PartialParseWithErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file with syntax errors but partial content
	filePath := filepath.Join(tempDir, "partial.go")
	writeTestFile(t, filePath, `package test

type Reader interface { Read() }

func broken(
`)

	finder := NewInterfaceFinder()
	finder.baseDir = tempDir

	interfaces, err := finder.extractInterfacesFromFile(filePath)
	require.Error(t, err) // Should return error due to parse error
	// But interfaces should still be found before the error
	require.Contains(t, interfaces, "Reader")
}

func TestExtractInterfaceName_NonTypeSpec(t *testing.T) {
	// Test with an import spec instead of type spec
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", "package test; import \"fmt\"", 0)
	require.NoError(t, err)

	var found []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for _, spec := range genDecl.Specs {
			if name, ok := extractInterfaceName(spec); ok {
				found = append(found, name)
			}
		}
	}

	require.Empty(t, found)
}
