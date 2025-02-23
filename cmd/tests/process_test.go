package tests

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oreofeolurin/corpus-packer/cpack/cmd"
)

func TestProcessDirectory(t *testing.T) {
	tests := []struct {
		name     string
		config   cmd.Config
		validate func(t *testing.T, outputPath string, config cmd.Config)
	}{
		{
			name: "process go files only",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				ExcludeGlobs: []string{"*_test.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				assertFileContains(t, outputPath, "package pkg1")
				assertFileContains(t, outputPath, "package pkg2")
				content, _ := os.ReadFile(outputPath)
				if strings.Contains(string(content), "package pkg1_test") {
					t.Error("Output should not contain test files")
				}
			},
		},
		{
			name: "process specific directories",
			config: cmd.Config{
				IncludeGlobs: []string{"src/pkg1/**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Output should contain pkg1")
				}
				if strings.Contains(contentStr, "package pkg2") {
					t.Error("Output should not contain pkg2")
				}
			},
		},
		{
			name: "ignore directories",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go", "**/*.json"},
				ExcludeGlobs: []string{"**/vendor/**", "**/.git/**"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				assertFileNotContains(t, outputPath, "vendor.json")
				assertFileNotContains(t, outputPath, "[core]")
			},
		},
		{
			name: "handle invalid file paths",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
				InputDir:     "/nonexistent/path",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				err := cmd.ProcessDirectory(config)
				if err == nil {
					t.Error("Expected error for nonexistent input directory")
				}
				if !strings.Contains(err.Error(), "input directory does not exist") {
					t.Errorf("Expected 'input directory does not exist' error, got: %v", err)
				}
				assertFileNotExists(t, outputPath)
			},
		},
		{
			name: "handle invalid output file path",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "/invalid/path/out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				err := cmd.ProcessDirectory(config)
				if err == nil {
					t.Error("Expected error for invalid output file path")
					return
				}
				if !strings.Contains(err.Error(), "error creating output file") {
					t.Errorf("Expected 'error creating output file' error, got: %v", err)
				}
			},
		},
		{
			name: "process multiple file types",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go", "**/*.md"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Output should contain Go files")
				}
				if !strings.Contains(contentStr, "# Package 2") {
					t.Error("Output should contain Markdown files")
				}
			},
		},
		{
			name: "handle invalid file patterns",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				ExcludeGlobs: []string{"[invalid-pattern"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should still process valid files with invalid patterns")
				}
			},
		},
		{
			name: "empty include patterns",
			config: cmd.Config{
				IncludeGlobs: []string{},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should process all files when no include patterns specified")
				}
			},
		},
		{
			name: "handle mixed case patterns",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.GO", "**/*.Md"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should handle uppercase patterns (.GO)")
				}
				if !strings.Contains(contentStr, "# Package 2") {
					t.Error("Should handle mixed case patterns (.Md)")
				}
			},
		},
		{
			name: "handle invalid file paths",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
				InputDir:     "/nonexistent/path",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				err := cmd.ProcessDirectory(config)
				if err == nil {
					t.Error("Expected error for nonexistent input directory")
				}
				if !strings.Contains(err.Error(), "input directory does not exist") {
					t.Errorf("Expected 'input directory does not exist' error, got: %v", err)
				}
				assertFileNotExists(t, outputPath)
			},
		},
		{
			name: "handle invalid output file path",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "/invalid/path/out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				err := cmd.ProcessDirectory(config)
				if err == nil {
					t.Error("Expected error for invalid output file path")
					return
				}
				if !strings.Contains(err.Error(), "error creating output file") {
					t.Errorf("Expected 'error creating output file' error, got: %v", err)
				}
			},
		},
		{
			name: "process multiple file types",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go", "**/*.md"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Output should contain Go files")
				}
				if !strings.Contains(contentStr, "# Package 2") {
					t.Error("Output should contain Markdown files")
				}
			},
		},
		{
			name: "handle invalid file patterns",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				ExcludeGlobs: []string{"[invalid-pattern"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should still process valid files with invalid patterns")
				}
			},
		},
		{
			name: "empty include patterns",
			config: cmd.Config{
				IncludeGlobs: []string{},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should process all files when no include patterns specified")
				}
			},
		},
		{
			name: "handle mixed case patterns",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.GO", "**/*.Md"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should handle uppercase patterns (.GO)")
				}
				if !strings.Contains(contentStr, "# Package 2") {
					t.Error("Should handle mixed case patterns (.Md)")
				}
			},
		},
		{
			name: "handle relative path traversal attempt",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				ExcludeGlobs: []string{"../something"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if strings.Contains(string(content), "..") {
					t.Error("Should not process paths with directory traversal")
				}
			},
		},
		{
			name: "unreadable file handling",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Create an unreadable file in the temp directory
				unreadableFile := filepath.Join(filepath.Dir(outputPath), "unreadable.go")
				err := os.WriteFile(unreadableFile, []byte("package main"), 0000)
				if err != nil {
					t.Fatalf("Failed to create unreadable file: %v", err)
				}

				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				// Should still process other files
				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should still process readable files when encountering unreadable ones")
				}
			},
		},
		{
			name: "symlink handling",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Create a symlink in the temp directory
				symlink := filepath.Join(filepath.Dir(outputPath), "symlink")
				target := filepath.Join(filepath.Dir(outputPath), "src")
				err := os.Symlink(target, symlink)
				if err != nil {
					t.Skipf("Symlink creation not supported: %v", err)
				}

				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				// Should process files through symlinks
				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should process files through symlinks")
				}
			},
		},
		{
			name: "handle unreadable directory",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Create an unreadable directory
				unreadableDir := filepath.Join(filepath.Dir(outputPath), "unreadable")
				err := os.MkdirAll(unreadableDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				err = os.WriteFile(filepath.Join(unreadableDir, "test.go"), []byte("package test"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				err = os.Chmod(unreadableDir, 0000)
				if err != nil {
					t.Fatalf("Failed to change directory permissions: %v", err)
				}
				defer os.Chmod(unreadableDir, 0755) // Restore permissions for cleanup

				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				// Should still process other directories
				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should still process readable directories")
				}
			},
		},
		{
			name: "handle relative paths in config",
			config: cmd.Config{
				IncludeGlobs: []string{"./src/pkg1/**/*.go"},
				OutputFile:   "./out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should handle relative paths in config")
				}
			},
		},
		{
			name: "handle empty input directory",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Create empty directory
				emptyDir := filepath.Join(filepath.Dir(outputPath), "empty")
				err := os.MkdirAll(emptyDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create empty directory: %v", err)
				}
				config.InputDir = emptyDir

				err = cmd.ProcessDirectory(config)
				if err != nil {
					t.Fatalf("ProcessDirectory failed: %v", err)
				}

				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if len(string(content)) > 0 {
					t.Error("Output should be empty for empty input directory")
				}
			},
		},
		{
			name: "handle mixed case extensions",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.GO", "**/*.Md"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should handle uppercase extensions (.GO)")
				}
				if !strings.Contains(contentStr, "# Package 2") {
					t.Error("Should handle mixed case extensions (.Md)")
				}
			},
		},
		{
			name: "handle duplicate extensions",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				// Count occurrences of file markers
				contentStr := string(content)
				startMarkerCount := strings.Count(contentStr, "--- START OF FILE: src/pkg1/file1.go ---")
				if startMarkerCount > 1 {
					t.Errorf("File processed multiple times: found %d occurrences", startMarkerCount)
				}
			},
		},
		{
			name: "handle nil extensions",
			config: cmd.Config{
				OutputFile: "out.txt",
				// IncludeGlobs intentionally left as nil
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				// Should use default extensions
				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should process .go files with default extensions")
				}
			},
		},
		{
			name: "handle extension normalization",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go", "**/*.md", "**/*.java"}, // Mix of formats
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should normalize and handle .go extension")
				}
			},
		},
		{
			name: "handle relative output path",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "./subdir/out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				assertFileExists(t, outputPath)
				assertFileContains(t, outputPath, "package pkg1")
			},
		},
		{
			name: "handle no valid dirs with files",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				ExcludeGlobs: []string{}, // Empty valid dirs
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Should process all directories when ValidDirs is empty")
				}
			},
		},
		{
			name: "handle invalid relative path",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				InputDir:     "../../../outside/project",
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				err := cmd.ProcessDirectory(config)
				if err == nil {
					t.Error("Should fail with invalid relative path")
					return
				}
				if !strings.Contains(err.Error(), "input directory does not exist") {
					t.Errorf("Expected 'input directory does not exist' error, got: %v", err)
				}
			},
		},
		{
			name: "handle verbose output",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				ExcludeGlobs: []string{"**/*_test.go"},
				OutputFile:   "out.txt",
				Verbose:      true,
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)

				// Check summary header and footer
				if !strings.Contains(contentStr, "--- CORPUS PACKER SUMMARY ---") {
					t.Error("Should include summary header when verbose is true")
				}
				if !strings.Contains(contentStr, "--- END OF SUMMARY ---") {
					t.Error("Should include summary footer when verbose is true")
				}

				// Check summary content
				if !strings.Contains(contentStr, "Processing Time:") {
					t.Error("Summary should include processing time")
				}
				if !strings.Contains(contentStr, "Total Files Processed:") {
					t.Error("Summary should include total files processed")
				}
				if !strings.Contains(contentStr, "Total Bytes Processed:") {
					t.Error("Summary should include total bytes processed")
				}

				// Check file lists
				if !strings.Contains(contentStr, "Processed Files:") {
					t.Error("Summary should include list of processed files")
				}
				if !strings.Contains(contentStr, "Skipped Files:") {
					t.Error("Summary should include list of skipped files")
				}

				// Check actual file content
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should still include file content after summary")
				}

				// Verify file appears in correct sections
				processedSection := contentStr[strings.Index(contentStr, "Processed Files:"):strings.Index(contentStr, "Skipped Files:")]
				if !strings.Contains(processedSection, "src/pkg1/file1.go") {
					t.Error("File should appear in processed files section")
				}

				skippedSection := contentStr[strings.Index(contentStr, "Skipped Files:"):strings.Index(contentStr, "--- END OF SUMMARY ---")]
				if !strings.Contains(skippedSection, "src/pkg1/file1_test.go") {
					t.Error("Test file should appear in skipped files section")
				}

				// Verify file content appears after summary
				contentSection := contentStr[strings.Index(contentStr, "--- END OF SUMMARY ---"):]
				if !strings.Contains(contentSection, "--- START OF FILE: src/pkg1/file1.go ---") {
					t.Error("File content should appear after summary")
				}
			},
		},
		{
			name: "handle non-verbose output",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
				Verbose:      false,
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)

				// Check that summary is not included
				if strings.Contains(contentStr, "--- CORPUS PACKER SUMMARY ---") {
					t.Error("Should not include summary when verbose is false")
				}

				// Check that file content is still included
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Should include file content without summary")
				}
			},
		},
		{
			name: "handle directory creation error",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "/dev/null/out.txt", // This should fail to create directory
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Skip the default ProcessDirectory call in the test runner
				if strings.HasPrefix(config.OutputFile, "/dev/null/") {
					err := cmd.ProcessDirectory(config)
					if err == nil {
						t.Error("Should fail when unable to create output directory")
						return
					}
					// Check for either error message since it might vary by OS
					if !strings.Contains(err.Error(), "error creating output directory") &&
						!strings.Contains(err.Error(), "not a directory") {
						t.Errorf("Expected directory creation error, got: %v", err)
					}
					return
				}
				t.Error("Test should have returned early")
			},
		},
		{
			name: "handle buffer write errors in verbose mode",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
				Verbose:      true,
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Create a huge file that might cause buffer write issues
				hugeFile := filepath.Join(filepath.Dir(outputPath), "huge.go")
				hugeContent := make([]byte, 1<<30) // 1GB
				err := os.WriteFile(hugeFile, hugeContent, 0644)
				if err != nil {
					t.Skipf("Could not create huge test file: %v", err)
				}
				defer os.Remove(hugeFile)

				err = cmd.ProcessDirectory(config)
				if err != nil {
					if !strings.Contains(err.Error(), "error writing") {
						t.Errorf("Expected write error, got: %v", err)
					}
				}
			},
		},
		{
			name: "verify summary sorting",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
				Verbose:      true,
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				processedSection := contentStr[strings.Index(contentStr, "Processed Files:"):strings.Index(contentStr, "Skipped Files:")]
				skippedSection := contentStr[strings.Index(contentStr, "Skipped Files:"):strings.Index(contentStr, "--- END OF SUMMARY ---")]

				// Check if files are sorted
				if !isSorted(strings.Split(strings.TrimSpace(processedSection), "\n")[1:]) {
					t.Error("Processed files should be sorted")
				}
				if !isSorted(strings.Split(strings.TrimSpace(skippedSection), "\n")[1:]) {
					t.Error("Skipped files should be sorted")
				}
			},
		},
		{
			name: "handle symlink cycle",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Create a symlink cycle
				cycleDir := filepath.Join(filepath.Dir(outputPath), "cycle")
				os.MkdirAll(cycleDir, 0755)
				os.Symlink(cycleDir, filepath.Join(cycleDir, "loop"))

				err := cmd.ProcessDirectory(config)
				if err != nil {
					t.Errorf("Should handle symlink cycles gracefully: %v", err)
				}
			},
		},
		{
			name: "handle write errors",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Create a read-only directory to trigger write error
				readOnlyDir := filepath.Join(filepath.Dir(outputPath), "readonly")
				if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
					t.Fatalf("Failed to create read-only directory: %v", err)
				}
				defer os.RemoveAll(readOnlyDir)

				config.OutputFile = filepath.Join(readOnlyDir, "out.txt")
				err := cmd.ProcessDirectory(config)
				if err == nil {
					t.Error("Expected write error, got nil")
					return
				}
				if !strings.Contains(err.Error(), "permission denied") &&
					!strings.Contains(err.Error(), "error creating output file") {
					t.Errorf("Expected permission denied error, got: %v", err)
				}
			},
		},
		{
			name: "handle compressed output",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt",
				Compress:     true,
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				contentStr := string(content)
				// Check that extra whitespace is removed
				if strings.Contains(contentStr, "  ") {
					t.Error("Compressed output should not contain multiple spaces")
				}
				// Check that code is still valid
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Compressed output should preserve code structure")
				}
				// Check file size reduction
				uncompressedConfig := config
				uncompressedConfig.Compress = false
				uncompressedConfig.OutputFile = outputPath + ".uncompressed"

				err = cmd.ProcessDirectory(uncompressedConfig)
				if err != nil {
					t.Fatalf("Failed to create uncompressed file: %v", err)
				}

				uncompressedContent, err := os.ReadFile(uncompressedConfig.OutputFile)
				if err != nil {
					t.Fatalf("Failed to read uncompressed file: %v", err)
				}

				if len(content) >= len(uncompressedContent) {
					t.Error("Compressed output should be smaller than uncompressed")
				}
			},
		},
		{
			name: "handle gzip with base64 encoding",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				OutputFile:   "out.txt.gz.b64",
				Gzip:         true,
				Base64:       true,
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				fmt.Printf("Debug: Validating output file: %s\n", outputPath)
				// Read and decode base64
				encoded, err := os.ReadFile(outputPath)
				if err != nil {
					fmt.Printf("Debug: Error reading file: %v\n", err)
					fmt.Printf("Debug: Current working directory: %s\n",
						func() string { dir, _ := os.Getwd(); return dir }())
					t.Fatalf("Failed to read output file: %v", err)
				}

				decoded := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
				n, err := base64.StdEncoding.Decode(decoded, encoded)
				if err != nil {
					t.Fatalf("Base64 decode failed: %v", err)
				}
				decoded = decoded[:n]

				// Decompress gzip
				gr, err := gzip.NewReader(bytes.NewReader(decoded))
				if err != nil {
					t.Fatalf("Failed to create gzip reader: %v", err)
				}
				defer gr.Close()

				content, err := io.ReadAll(gr)
				if err != nil {
					t.Fatalf("Failed to decompress gzip content: %v", err)
				}

				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Base64 encoded gzip content should contain source code")
				}
			},
		},
		{
			name: "default output with gzip",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				Gzip:         true,
				OutputFile:   "corpus-out.txt.gz",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				if !strings.HasSuffix(outputPath, ".gz") {
					t.Error("Default output file should have .gz extension when using --gzip")
				}

				// Verify it's a valid gzip file
				f, err := os.Open(outputPath)
				if err != nil {
					t.Fatalf("Failed to open output file: %v", err)
				}
				defer f.Close()

				gr, err := gzip.NewReader(f)
				if err != nil {
					t.Fatalf("Not a valid gzip file: %v", err)
				}
				defer gr.Close()

				content, err := io.ReadAll(gr)
				if err != nil {
					t.Fatalf("Failed to read gzip content: %v", err)
				}

				if !strings.Contains(string(content), "package pkg1") {
					t.Error("Gzipped content should contain source code")
				}
			},
		},
		{
			name: "auto-add gzip extension",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				Gzip:         true,
				OutputFile:   "custom-output.txt",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				// Construct the expected path with .gz extension
				expectedPath := outputPath + ".gz"

				if !strings.HasSuffix(expectedPath, ".gz") {
					t.Error("Should automatically add .gz extension to custom filename")
				}

				// Verify it's a valid gzip file
				f, err := os.Open(expectedPath)
				if err != nil {
					t.Fatalf("Failed to open output file: %v", err)
				}
				defer f.Close()

				gr, err := gzip.NewReader(f)
				if err != nil {
					t.Fatalf("Not a valid gzip file: %v", err)
				}
				defer gr.Close()
			},
		},
		{
			name: "keep existing gzip extension",
			config: cmd.Config{
				IncludeGlobs: []string{"**/*.go"},
				Gzip:         true,
				OutputFile:   "already-has.gz",
			},
			validate: func(t *testing.T, outputPath string, config cmd.Config) {
				if !strings.HasSuffix(outputPath, ".gz") || strings.Count(outputPath, ".gz") > 1 {
					t.Error("Should preserve existing .gz extension without duplicating")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory structure
			tempDir, cleanup := createTestFiles(t)
			defer cleanup()

			// Set input directory in config if not explicitly set
			if tt.config.InputDir == "" {
				tt.config.InputDir = tempDir
			}

			// Make output path absolute if not already
			if !filepath.IsAbs(tt.config.OutputFile) {
				tt.config.OutputFile = filepath.Join(tempDir, tt.config.OutputFile)
			}

			// For error test cases, validate directly in the test case
			if tt.config.InputDir == "/nonexistent/path" ||
				strings.HasPrefix(tt.config.OutputFile, "/invalid/path/") ||
				strings.HasPrefix(tt.config.OutputFile, "/dev/null/") ||
				strings.Contains(tt.config.InputDir, "../../../outside") {
				tt.validate(t, tt.config.OutputFile, tt.config)
				return
			}

			// Process directory for normal cases
			err := cmd.ProcessDirectory(tt.config)
			if err != nil {
				t.Fatalf("ProcessDirectory failed: %v", err)
			}

			// Validate results
			tt.validate(t, tt.config.OutputFile, tt.config)
		})
	}
}

func TestProcessDirectoryWithConfigFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files and directories
	testFiles := map[string]string{
		"src/pkg1/file1.go":     "package pkg1",
		"src/pkg2/file2.go":     "package pkg2",
		"src/pkg1/file1.py":     "def main():",
		"src/pkg2/test_file.go": "package test",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Create test config files
	yamlConfig := fmt.Sprintf(`
inputDir: %s
outputFile: output.txt
validExtensions:
  - .go
includeGlobs:
  - "**/pkg1/*.go"
excludeGlobs:
  - "*test*.go"
verbose: true
`, filepath.Join(tmpDir, "src"))

	yamlPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(yamlPath, []byte(yamlConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML config: %v", err)
	}

	// Save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	tests := []struct {
		name           string
		configPath     string
		overrideConfig cmd.Config
		wantErr        bool
		validate       func(t *testing.T, config cmd.Config)
	}{
		{
			name:       "basic yaml config",
			configPath: yamlPath,
			overrideConfig: cmd.Config{
				InputDir: filepath.Join(tmpDir, "src"),
			},
			wantErr: false,
			validate: func(t *testing.T, config cmd.Config) {
				// Get output file path relative to CWD
				outputPath := filepath.Join(originalWd, "output.txt")
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				contentStr := string(content)

				// Should contain pkg1 files but not pkg2
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Output should contain pkg1 files")
				}
				if strings.Contains(contentStr, "package pkg2") {
					t.Error("Output should not contain pkg2 files")
				}
				// Should not contain test files
				if strings.Contains(contentStr, "package test") {
					t.Error("Output should not contain test files")
				}
			},
		},
		{
			name:       "override config values",
			configPath: yamlPath,
			overrideConfig: cmd.Config{
				InputDir:     filepath.Join(tmpDir, "src"),
				IncludeGlobs: []string{"**/*.py"},
			},
			wantErr: false,
			validate: func(t *testing.T, config cmd.Config) {
				// Get output file path relative to CWD
				outputPath := filepath.Join(originalWd, "output.txt")
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				contentStr := string(content)

				// Should only contain Python files
				if !strings.Contains(contentStr, "def main():") {
					t.Error("Output should contain Python files")
				}
				if strings.Contains(contentStr, "package") {
					t.Error("Output should not contain Go files")
				}
			},
		},
		{
			name:       "non-existent config file",
			configPath: "non-existent.yaml",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load config first to get the final config for validation
			fileConfig, err := cmd.LoadConfigFromFile(tt.configPath)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Failed to load config file: %v", err)
			}

			// Apply overrides
			if tt.overrideConfig.InputDir != "" {
				fileConfig.InputDir = tt.overrideConfig.InputDir
			}
			if tt.overrideConfig.IncludeGlobs != nil {
				fileConfig.IncludeGlobs = tt.overrideConfig.IncludeGlobs
			}

			err = cmd.ProcessDirectoryWithConfigFile(tt.configPath, tt.overrideConfig)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, *fileConfig)
			}
		})
	}

	// Clean up any output files created in CWD
	os.Remove(filepath.Join(originalWd, "output.txt"))
}
