package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestFiles creates a temporary directory structure for testing
func createTestFiles(t *testing.T) (string, func()) {
	t.Helper()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "corpus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Ensure the directory is created with proper permissions
	if err := os.Chmod(tempDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set directory permissions: %v", err)
	}

	// Create test file structure
	dirs := []string{
		"src/pkg1",
		"src/pkg1/subdir",
		"src/pkg2",
		"internal",
		".git",
		"vendor",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"src/pkg1/file1.go":      "package pkg1\n\nfunc Test() {}\n",
		"src/pkg1/file1_test.go": "package pkg1_test\n",
		"src/pkg2/file2.go":      "package pkg2\n",
		"src/pkg2/README.md":     "# Package 2\n",
		".git/config":            "[core]\n",
		"vendor/vendor.json":     "{}",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// assertFileContains checks if a file contains expected content
func assertFileContains(t *testing.T, path string, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !strings.Contains(string(content), expected) {
		t.Errorf("Expected file to contain:\n%s\nGot:\n%s", expected, string(content))
	}
}

// assertFileExists checks if a file exists
func assertFileExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", path)
	}
}

// assertFileNotContains checks if a file does not contain expected content
func assertFileNotContains(t *testing.T, path string, unexpected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if strings.Contains(string(content), unexpected) {
		t.Errorf("File should not contain:\n%s", unexpected)
	}
}

// assertFileNotExists checks if a file does not exist
func assertFileNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		t.Errorf("File %s should not exist", path)
	}
}

// Helper function to check if a slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// Helper function to check if strings are sorted
func isSorted(strs []string) bool {
	for i := 1; i < len(strs); i++ {
		if strs[i] < strs[i-1] {
			return false
		}
	}
	return true
}
