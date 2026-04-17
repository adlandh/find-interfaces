package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

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

func TestFindInterfaces_IgnoresInterfaceLikeTextInInterfaceBody(t *testing.T) {
	tempDir := t.TempDir()

	writeTestFile(t, filepath.Join(tempDir, "input.go"), `package test

type Reader interface {
	Read() (data []byte, err error)
}
`)

	finder := NewInterfaceFinder()
	interfacesFound, err := finder.FindInterfaces(tempDir)
	require.NoError(t, err)
	require.Equal(t, []string{"Reader"}, interfacesFound)
}

func TestFindInterfaces_MultipleMixedDeclarationsInSingleFile(t *testing.T) {
	tempDir := t.TempDir()

	writeTestFile(t, filepath.Join(tempDir, "mixed.go"), `package test

type Alias = int
type Person struct { Name string }
type Reader interface { Read() }
const Max = 100
var global = 42
type Writer interface { Write([]byte) }
`)

	finder := NewInterfaceFinder()
	interfacesFound, err := finder.FindInterfaces(tempDir)
	require.NoError(t, err)
	sort.Strings(interfacesFound)
	require.Equal(t, []string{"Reader", "Writer"}, interfacesFound)
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

func TestFindInterfaces_WalkErrorPropagation(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o000))

	finder := NewInterfaceFinder()
	interfaces, err := finder.FindInterfaces(dir)

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

	interfaces, err := extractInterfacesFromFile(baseDir, outsideFile)
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

	interfaces, err := extractInterfacesFromFile(baseDir, outsideFile)
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

	// Now use a different baseDir to trigger validation error
	otherBaseDir := t.TempDir() // Different directory

	interfaces, err := extractInterfacesFromFile(otherBaseDir, filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside the base directory")
	require.Empty(t, interfaces)
}

func TestExtractInterfacesFromFile_ReturnsNonFileParseErrorForOutsidePath(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create a file outside baseDir
	outsideFile := filepath.Join(outsideDir, "outside.go")
	writeTestFile(t, outsideFile, `package test
type Outside interface { Noop() }
`)

	// This should return an error that's not a fileParseError
	_, err := extractInterfacesFromFile(baseDir, outsideFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside the base directory")
	require.Nil(t, asFileParseError(err), "error should not be a *fileParseError")
}

func TestExtractInterfacesFromFile_PartialParseWithErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file with syntax errors but partial content
	filePath := filepath.Join(tempDir, "partial.go")
	writeTestFile(t, filePath, `package test

type Reader interface { Read() }

func broken(
`)

	interfaces, err := extractInterfacesFromFile(tempDir, filePath)
	require.Error(t, err) // Should return error due to parse error
	// But interfaces should still be found before the error
	require.Contains(t, interfaces, "Reader")
}

func TestExtractInterfaceName_NonTypeSpec(t *testing.T) {
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

func TestShouldSkipDir(t *testing.T) {
	baseDir := t.TempDir()
	subDir := filepath.Join(baseDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	baseInfo, err := os.Lstat(baseDir)
	require.NoError(t, err)
	_, err = os.Lstat(subDir)
	require.NoError(t, err)

	require.False(t, shouldSkipDir(baseDir, osDirEntry{info: baseInfo}, baseDir), "base directory itself should not be skipped")

	nestedDir := filepath.Join(subDir, "nested")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))
	nestedInfo, err := os.Lstat(nestedDir)
	require.NoError(t, err)
	require.True(t, shouldSkipDir(nestedDir, osDirEntry{info: nestedInfo}, baseDir), "subdirectory should be skipped")
}

func TestShouldProcessFile(t *testing.T) {
	tests := []struct {
		name    string
		info    fs.FileInfo
		want    bool
		wantErr bool
	}{
		{
			name:    "regular go file",
			info:    fileInfo{name: "test.go", mode: 0o644},
			want:    true,
			wantErr: false,
		},
		{
			name:    "uppercase extension",
			info:    fileInfo{name: "test.GO", mode: 0o644},
			want:    true,
			wantErr: false,
		},
		{
			name:    "non go file",
			info:    fileInfo{name: "test.txt", mode: 0o644},
			want:    false,
			wantErr: false,
		},
		{
			name:    "no extension",
			info:    fileInfo{name: "Makefile", mode: 0o644},
			want:    false,
			wantErr: false,
		},
		{
			name:    "directory",
			info:    fileInfo{name: "mydir", mode: fs.ModeDir | 0o755},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldProcessFile(osDirEntry{info: tt.info})
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFileParseError_Error(t *testing.T) {
	innerErr := errors.New("syntax error")
	parseErr := &fileParseError{path: "test.go", err: innerErr}

	msg := parseErr.Error()
	require.Contains(t, msg, "test.go")
	require.Contains(t, msg, "syntax error")
}

func TestVisitPath_WalkError(t *testing.T) {
	walkErr := errors.New("permission denied")
	found, err := visitPath("", "/some/path", nil, walkErr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
	require.Empty(t, found)
}

func TestExtractInterfacesFromDecl_NonTypeTokens(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		declIndex int
	}{
		{"const declaration", "package test; const X = 1", 0},
		{"var declaration", "package test; var Y = 2", 0},
		{"import declaration", "package test; import \"fmt\"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.code, 0)
			require.NoError(t, err)
			require.Less(t, tt.declIndex, len(file.Decls))
			result := extractInterfacesFromDecl(file.Decls[tt.declIndex])
			require.Nil(t, result)
		})
	}
}

type osDirEntry struct {
	info fs.FileInfo
}

func (e osDirEntry) Name() string               { return e.info.Name() }
func (e osDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e osDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e osDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

type fileInfo struct {
	name string
	mode fs.FileMode
}

func (f fileInfo) Name() string       { return f.name }
func (f fileInfo) Size() int64        { return 0 }
func (f fileInfo) Mode() fs.FileMode  { return f.mode }
func (f fileInfo) ModTime() time.Time { return time.Time{} }
func (f fileInfo) IsDir() bool        { return f.mode&fs.ModeDir != 0 }
func (f fileInfo) Sys() any           { return nil }
