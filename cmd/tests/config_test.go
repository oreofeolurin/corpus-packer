package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oreofeolurin/corpus-packer/cpack/cmd"
)

func TestLoadConfigFromFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		configFile string
		content    string
		wantErr    bool
		validate   func(t *testing.T, config *cmd.Config)
	}{
		{
			name:       "valid yaml config",
			configFile: "config.yaml",
			content: `
inputDir: ./src
outputFile: output.txt
includeGlobs:
  - "**/*.go"
  - "**/*.py"
excludeGlobs:
  - "**/*_test.go"
  - "**/vendor/**"
verbose: true
`,
			wantErr: false,
			validate: func(t *testing.T, config *cmd.Config) {
				if config.InputDir != "./src" {
					t.Errorf("Expected InputDir to be './src', got %s", config.InputDir)
				}
				if config.OutputFile != "output.txt" {
					t.Errorf("Expected OutputFile to be 'output.txt', got %s", config.OutputFile)
				}
				if len(config.IncludeGlobs) != 2 {
					t.Errorf("Expected 2 include patterns, got %d", len(config.IncludeGlobs))
				}
				if !config.Verbose {
					t.Error("Expected Verbose to be true")
				}
			},
		},
		{
			name:       "valid json config",
			configFile: "config.json",
			content: `{
				"inputDir": "./src",
				"outputFile": "output.txt",
				"includeGlobs": ["**/*.go", "**/*.py"],
				"excludeGlobs": ["**/*_test.go", "**/vendor/**"],
				"verbose": true
			}`,
			wantErr: false,
			validate: func(t *testing.T, config *cmd.Config) {
				if config.InputDir != "./src" {
					t.Errorf("Expected InputDir to be './src', got %s", config.InputDir)
				}
				if config.OutputFile != "output.txt" {
					t.Errorf("Expected OutputFile to be 'output.txt', got %s", config.OutputFile)
				}
				if len(config.IncludeGlobs) != 2 {
					t.Errorf("Expected 2 include patterns, got %d", len(config.IncludeGlobs))
				}
				if !config.Verbose {
					t.Error("Expected Verbose to be true")
				}
			},
		},
		{
			name:       "invalid yaml syntax",
			configFile: "invalid.yaml",
			content:    "invalid: [yaml: syntax",
			wantErr:    true,
		},
		{
			name:       "invalid json syntax",
			configFile: "invalid.json",
			content:    "{invalid: json}",
			wantErr:    true,
		},
		{
			name:       "unsupported file format",
			configFile: "config.txt",
			content:    "some content",
			wantErr:    true,
		},
		{
			name:       "empty config file",
			configFile: "empty.yaml",
			content:    "",
			wantErr:    false,
			validate: func(t *testing.T, config *cmd.Config) {
				defaultConfig := cmd.DefaultConfig()
				if config.InputDir != defaultConfig.InputDir {
					t.Error("Empty config should use default InputDir")
				}
				if config.OutputFile != defaultConfig.OutputFile {
					t.Error("Empty config should use default OutputFile")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.configFile)
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			config, err := cmd.LoadConfigFromFile(configPath)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := cmd.LoadConfigFromFile("non-existent.yaml")
		if err == nil {
			t.Error("Expected error for non-existent file but got none")
		}
	})

	// Test empty file path
	t.Run("empty file path", func(t *testing.T) {
		_, err := cmd.LoadConfigFromFile("")
		if err == nil {
			t.Error("Expected error for empty file path but got none")
		}
	})
}

func TestAutoLoadConfig(t *testing.T) {
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

	tests := []struct {
		name       string
		configFile string
		content    string
		config     cmd.Config
		validate   func(t *testing.T, outputPath string)
	}{
		{
			name:       "auto load yaml config",
			configFile: "cpack.yaml",
			content: `
inputDir: src
outputFile: output.txt
includeGlobs:
  - "**/*.go"
excludeGlobs:
  - "*test*.go"
verbose: true
`,
			config: cmd.Config{
				InputDir: tmpDir,
			},
			validate: func(t *testing.T, outputPath string) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				contentStr := string(content)

				// Should contain pkg1 files but not pkg2
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Output should contain pkg1 files")
				}
				if !strings.Contains(contentStr, "package pkg2") {
					t.Error("Output should not contain pkg2 files")
				}
				// Should not contain test files
				if strings.Contains(contentStr, "package test") {
					t.Error("Output should not contain test files")
				}
				// Should not contain python files
				if strings.Contains(contentStr, "def main():") {
					t.Error("Output should not contain python files")
				}
			},
		},
		{
			name:       "auto load json config",
			configFile: "cpack.json",
			content: `{
				"inputDir": "src",
				"outputFile": "output.txt",
				"includeGlobs": ["**/*.go"],
				"excludeGlobs": ["*test*.go"],
				"verbose": true
			}`,
			config: cmd.Config{
				InputDir: tmpDir,
			},
			validate: func(t *testing.T, outputPath string) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				contentStr := string(content)

				// Should only contain Go files
				if !strings.Contains(contentStr, "package pkg1") {
					t.Error("Output should contain Go files")
				}
				if !strings.Contains(contentStr, "package pkg2") {
					t.Error("Output should not contain Go files")
				}
				// Should not contain test files
				if strings.Contains(contentStr, "package test") {
					t.Error("Output should not contain test files")
				}
				// Should not contain python files
				if strings.Contains(contentStr, "def main():") {
					t.Error("Output should not contain python files")
				}
			},
		},
		{
			name:       "command line overrides auto config",
			configFile: "cpack.yml",
			content: `
inputDir: src
outputFile: output.txt
includeGlobs:
  - "**/*.go"
excludeGlobs:
  - "*test*.go"
verbose: true
`,
			config: cmd.Config{
				InputDir:     tmpDir,
				IncludeGlobs: []string{"**/*.py"},
			},
			validate: func(t *testing.T, outputPath string) {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				contentStr := string(content)

				// Should only contain Python files due to command line override
				if !strings.Contains(contentStr, "def main():") {
					t.Error("Output should contain Python files")
				}
				if strings.Contains(contentStr, "package") {
					t.Error("Output should not contain Go files")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file
			configPath := filepath.Join(tmpDir, tt.configFile)
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Process directory using the config file
			err = cmd.ProcessDirectoryWithConfigFile(configPath, tt.config)
			if err != nil {
				t.Fatalf("ProcessDirectoryWithConfigFile failed: %v", err)
			}

			// Get output file path - it should be relative to the current working directory
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current working directory: %v", err)
			}
			outputPath := filepath.Join(cwd, "output.txt")
			if tt.validate != nil {
				tt.validate(t, outputPath)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := cmd.DefaultConfig()

	// Test default values
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{
			name:     "default input directory",
			got:      config.InputDir,
			expected: ".",
		},
		{
			name:     "default output file",
			got:      config.OutputFile,
			expected: "corpus-out.txt",
		},
		{
			name:     "default include patterns include .go",
			got:      contains(config.IncludeGlobs, "**/*.go"),
			expected: true,
		},
		{
			name:     "default exclude patterns include .git",
			got:      contains(config.ExcludeGlobs, "**/.git/**"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %v, want %v", tt.got, tt.expected)
			}
		})
	}

	t.Run("default verbose setting", func(t *testing.T) {
		if config.Verbose != false {
			t.Error("Verbose should be false by default")
		}
	})

	// Test using default config with ProcessDirectory
	t.Run("process with default config", func(t *testing.T) {
		tempDir, cleanup := createTestFiles(t)
		defer cleanup()

		config := cmd.DefaultConfig()
		config.InputDir = tempDir
		config.OutputFile = filepath.Join(tempDir, "out.txt")

		err := cmd.ProcessDirectory(config)
		if err != nil {
			t.Fatalf("ProcessDirectory failed: %v", err)
		}

		content, err := os.ReadFile(config.OutputFile)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "package pkg1") {
			t.Error("Should process .go files with default config")
		}
		if strings.Contains(contentStr, "[core]") {
			t.Error("Should ignore .git directory with default config")
		}
	})
}

func TestMergeConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     cmd.Config
		autoConfig *cmd.Config
		want       cmd.Config
	}{
		{
			name: "nil auto config",
			config: cmd.Config{
				InputDir:   ".",
				OutputFile: "out.txt",
			},
			autoConfig: nil,
			want: cmd.Config{
				InputDir:   ".",
				OutputFile: "out.txt",
			},
		},
		{
			name: "merge with empty config",
			config: cmd.Config{
				InputDir: ".",
			},
			autoConfig: &cmd.Config{
				OutputFile:   "auto.txt",
				IncludeGlobs: []string{"**/*.go"},
				Verbose:      true,
			},
			want: cmd.Config{
				InputDir:     ".",
				OutputFile:   "auto.txt",
				IncludeGlobs: []string{"**/*.go"},
				Verbose:      true,
			},
		},
		{
			name: "config takes precedence",
			config: cmd.Config{
				InputDir:     "/custom",
				OutputFile:   "custom.txt",
				IncludeGlobs: []string{"**/*.py"},
				Verbose:      false,
			},
			autoConfig: &cmd.Config{
				InputDir:     "/auto",
				OutputFile:   "auto.txt",
				IncludeGlobs: []string{"**/*.go"},
				Verbose:      true,
			},
			want: cmd.Config{
				InputDir:     "/custom",
				OutputFile:   "custom.txt",
				IncludeGlobs: []string{"**/*.py"},
				Verbose:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println("config", tt.config)
			fmt.Println("autoConfig", tt.autoConfig)
			got := cmd.MergeConfig(tt.config, tt.autoConfig)

			fmt.Println("got", got)

			if got.InputDir != tt.want.InputDir {
				t.Errorf("InputDir = %v, want %v", got.InputDir, tt.want.InputDir)
			}
			if got.OutputFile != tt.want.OutputFile {
				t.Errorf("OutputFile = %v, want %v", got.OutputFile, tt.want.OutputFile)
			}
			if !sliceEqual(got.IncludeGlobs, tt.want.IncludeGlobs) {
				t.Errorf("IncludeGlobs = %v, want %v", got.IncludeGlobs, tt.want.IncludeGlobs)
			}
			if got.Verbose != tt.want.Verbose {
				t.Errorf("Verbose = %v, want %v", got.Verbose, tt.want.Verbose)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config cmd.Config
		want   cmd.Config
	}{
		{
			name:   "empty config",
			config: cmd.Config{},
			want: cmd.Config{
				InputDir:     ".",
				OutputFile:   filepath.Join(".", "corpus-out.txt"),
				IncludeGlobs: cmd.DefaultConfig().IncludeGlobs,
				ExcludeGlobs: cmd.DefaultConfig().ExcludeGlobs,
			},
		},
		{
			name: "gzip output file",
			config: cmd.Config{
				InputDir: "src",
				Gzip:     true,
			},
			want: cmd.Config{
				InputDir:     "src",
				OutputFile:   filepath.Join(".", "corpus-out.txt.gz"),
				IncludeGlobs: cmd.DefaultConfig().IncludeGlobs,
				ExcludeGlobs: cmd.DefaultConfig().ExcludeGlobs,
				Gzip:         true,
			},
		},
		{
			name: "custom values preserved",
			config: cmd.Config{
				InputDir:     "/custom",
				OutputFile:   "out.txt",
				IncludeGlobs: []string{"**/*.py"},
			},
			want: cmd.Config{
				InputDir:     "/custom",
				OutputFile:   filepath.Join(".", "out.txt"),
				IncludeGlobs: []string{"**/*.py"},
				ExcludeGlobs: cmd.DefaultConfig().ExcludeGlobs,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cmd.ApplyDefaults(tt.config)

			if got.InputDir != tt.want.InputDir {
				t.Errorf("InputDir = %v, want %v", got.InputDir, tt.want.InputDir)
			}
			if got.OutputFile != tt.want.OutputFile {
				t.Errorf("OutputFile = %v, want %v", got.OutputFile, tt.want.OutputFile)
			}
			if !sliceEqual(got.IncludeGlobs, tt.want.IncludeGlobs) {
				t.Errorf("IncludeGlobs = %v, want %v", got.IncludeGlobs, tt.want.IncludeGlobs)
			}
			if !sliceEqual(got.ExcludeGlobs, tt.want.ExcludeGlobs) {
				t.Errorf("ExcludeGlobs = %v, want %v", got.ExcludeGlobs, tt.want.ExcludeGlobs)
			}
			if got.Gzip != tt.want.Gzip {
				t.Errorf("Gzip = %v, want %v", got.Gzip, tt.want.Gzip)
			}
		})
	}
}

func TestRelativePathsInConfig(t *testing.T) {
	// Create test directory structure using helper
	tmpDir, cleanup := createTestFiles(t)
	defer cleanup()

	// Test cases for different relative path configurations
	tests := []struct {
		name          string
		configContent string
		workDir       string   // Directory to run the command from
		wantFiles     []string // Files that should be included
		unwantFiles   []string // Files that should be excluded
	}{
		{
			name:    "paths relative to root directory",
			workDir: tmpDir,
			configContent: `
inputDir: src
outputFile: output/result.txt
includeGlobs:
  - "pkg1/**/*.go"
excludeGlobs:
  - "**/vendor/**"
  - "**/*_test.go"
verbose: true
`,
			wantFiles: []string{
				"package pkg1", // from file1.go
			},
			unwantFiles: []string{
				"package pkg1_test", // from file1_test.go
				"package pkg2",      // from pkg2/file2.go
				"def main()",        // from main.py
				"def helper()",      // from utils.py
			},
		},
		{
			name:    "paths relative to src directory",
			workDir: filepath.Join(tmpDir, "src"),
			configContent: `
inputDir: .
outputFile: ../output/result2.txt
includeGlobs:
  - "**/*.go"
excludeGlobs:
  - "**/vendor/**"
  - "**/*_test.go"
verbose: true
`,
			wantFiles: []string{
				"package pkg1", // from file1.go
				"package pkg2", // from file2.go
			},
			unwantFiles: []string{
				"package pkg1_test", // from file1_test.go
				"def main()",        // from main.py
				"def helper()",      // from utils.py
			},
		},
		{
			name:    "paths relative to config directory",
			workDir: filepath.Join(tmpDir, "config"),
			configContent: `
inputDir: ../src
outputFile: ../output/result3.txt
includeGlobs:
  - "pkg1/**/*.go"
excludeGlobs:
  - "**/vendor/**"
  - "**/*_test.go"
verbose: true
`,
			wantFiles: []string{
				"package pkg1", // from file1.go
			},
			unwantFiles: []string{
				"package pkg1_test", // from file1_test.go
				"package pkg2",      // from pkg2/file2.go
				"def main()",        // from main.py
				"def helper()",      // from utils.py
			},
		},
	}

	// Save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file in the config directory
			configPath := filepath.Join(tmpDir, "config", "cpack.yaml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Change to test working directory
			if err := os.Chdir(tt.workDir); err != nil {
				t.Fatalf("Failed to change working directory: %v", err)
			}
			// Restore original working directory after test
			defer func() {
				if err := os.Chdir(originalWd); err != nil {
					t.Errorf("Failed to restore working directory: %v", err)
				}
			}()

			// Process directory using the config file
			err := cmd.ProcessDirectoryWithConfigFile(configPath, cmd.Config{})
			if err != nil {
				t.Fatalf("ProcessDirectoryWithConfigFile failed: %v", err)
			}

			// Get output file path from the config content
			var outputFile string
			if strings.Contains(tt.configContent, "result2.txt") {
				outputFile = "result2.txt"
			} else if strings.Contains(tt.configContent, "result3.txt") {
				outputFile = "result3.txt"
			} else {
				outputFile = "result.txt"
			}
			outputPath := filepath.Join(tmpDir, "output", outputFile)

			// Verify output file exists and is not empty
			outputInfo, err := os.Stat(outputPath)
			if err != nil {
				t.Fatalf("Output file not created in correct location: %v", err)
			}
			if outputInfo.Size() == 0 {
				t.Error("Output file is empty")
			}

			// Read output content
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}
			contentStr := string(content)

			// Verify wanted files are included
			for _, wantFile := range tt.wantFiles {
				if !strings.Contains(contentStr, wantFile) {
					t.Errorf("Output should contain: %s", wantFile)
				}
			}

			// Verify unwanted files are excluded
			for _, unwantFile := range tt.unwantFiles {
				if strings.Contains(contentStr, unwantFile) {
					t.Errorf("Output should not contain: %s", unwantFile)
				}
			}
		})
	}
}
