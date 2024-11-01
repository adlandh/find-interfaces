package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
)

func TestFindInterfaces_NoGoFiles(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", gofakeit.Word())
	require.NoError(t, err, "failed to create temporary directory")
	defer func(path string) {
		err := os.RemoveAll(path)
		require.NoError(t, err, "failed to remove temporary directory")
	}(tempDir)

	// Call the function to test
	interfaces, err := findInterfaces(tempDir)
	require.NoError(t, err)
	// Check the result
	require.Len(t, interfaces, 0)
}

func TestFindInterfaces(t *testing.T) {

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", gofakeit.Word())
	require.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		require.NoError(t, err, "failed to remove temporary directory")
	}(tempDir)

	count := gofakeit.Number(1, 5)

	interfaces := make([]string, count)

	for i := range count {
		// Generate a random interface name
		interfaceName := gofakeit.Word()
		interfaces[i] = interfaceName

		// Create a test file with an interface definition
		testFile := filepath.Join(tempDir, interfaceName+"Interface.go")

		interfaceDefinition := `
package main

type ` + interfaceName + ` interface {
    DoSomething()
}
`
		err = os.WriteFile(testFile, []byte(interfaceDefinition), 0644)
		require.NoError(t, err, "failed to create test file")

	}

	// Create a temporary subdirectory to check if it is included
	tempDirSub, err := os.MkdirTemp(tempDir, gofakeit.Word())
	require.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		require.NoError(t, err, "failed to remove temporary directory")
	}(tempDirSub)

	countSub := gofakeit.Number(1, 5)

	for range countSub {
		// Generate a random interface name
		interfaceName := gofakeit.Word()

		// Create a test file with an interface definition
		testFile := filepath.Join(tempDirSub, interfaceName+"Interface.go")

		interfaceDefinition := `
package main

type ` + interfaceName + ` interface {
    DoSomething()
}
`
		err = os.WriteFile(testFile, []byte(interfaceDefinition), 0644)
		require.NoError(t, err, "failed to create test file")

	}

	// Call the function to test
	interfacesFound, err := findInterfaces(tempDir)
	require.NoError(t, err, "findInterfaces returned an error")
	// Check the result
	require.Len(t, interfacesFound, count)
	for _, iface := range interfaces {
		require.Contains(t, interfacesFound, iface)
	}
}

func TestFindInterfaces_NonExistentDirectory(t *testing.T) {
	// Call the function to test
	interfaces, err := findInterfaces(gofakeit.Word())
	require.Error(t, err, "findInterfaces did not return an error for non-existent directory")

	// Check the result
	require.Len(t, interfaces, 0, "findInterfaces returned an unexpected result: %v", interfaces)
}
