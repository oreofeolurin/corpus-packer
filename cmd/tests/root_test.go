package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oreofeolurin/corpus-packer/cpack/cmd"
)

func TestExecute(t *testing.T) {
	// Save original args and restore them after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Create test directory
	tempDir, cleanup := createTestFiles(t)
	defer cleanup()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "use current directory",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "use directory argument",
			args:    []string{tempDir},
			wantErr: false,
		},
		{
			name:    "use directory with output flag",
			args:    []string{tempDir, "-o", filepath.Join(tempDir, "out.txt")},
			wantErr: false,
		},
		{
			name:    "nonexistent directory",
			args:    []string{"/nonexistent/path"},
			wantErr: true,
		},
		{
			name:    "too many arguments",
			args:    []string{tempDir, "extra-arg"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset args for each test
			os.Args = append([]string{"cpack"}, tt.args...)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Cleanup any test output files "corpus-out.txt"
			files, err := filepath.Glob("corpus-out.txt")
			if err == nil {
				for _, f := range files {
					os.Remove(f)
				}
			}
		})
	}
}
