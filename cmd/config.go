package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the program's configuration
type Config struct {
	InputDir     string   `yaml:"inputDir" json:"inputDir"`
	OutputFile   string   `yaml:"outputFile" json:"outputFile"`
	IncludeGlobs []string `yaml:"includeGlobs" json:"includeGlobs"`
	ExcludeGlobs []string `yaml:"excludeGlobs" json:"excludeGlobs"`
	Verbose      bool     `yaml:"verbose" json:"verbose"`
	Compress     bool     `yaml:"compress" json:"compress"`
	MaxCompress  bool     `yaml:"maxCompress" json:"maxCompress"`
	Gzip         bool     `yaml:"gzip" json:"gzip"`
	Base64       bool     `yaml:"base64" json:"base64"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		InputDir:   ".", // Current directory
		OutputFile: "corpus-out.txt",
		Verbose:    false,
		Compress:   false,
		Gzip:       false,
		Base64:     false,
		IncludeGlobs: []string{
			"**/*.go",         // Go source files
			"**/*.js",         // JavaScript
			"**/*.ts",         // TypeScript
			"**/*.css",        // CSS
			"**/*.py",         // Python
			"**/*.java",       // Java
			"**/*.cpp",        // C++
			"**/*.c",          // C
			"**/*.h",          // Header files
			"**/*.hpp",        // C++ headers
			"**/*.rb",         // Ruby
			"**/*.php",        // PHP
			"**/*.cs",         // C#
			"**/*.swift",      // Swift
			"**/*.kt",         // Kotlin
			"**/*.md",         // Markdown
			"**/*.tsx",        // TypeScript React
			"**/*.jsx",        // JavaScript React
			"**/*.json",       // JSON
			"**/*.{yaml,yml}", // YAML
			"**/*.toml",       // TOML
			"**/*.txt",        // TXT
			"**/*.xml",        // XML
			"**/*.{doc,docx}", // Word documents
			"**/*.{ppt,pptx}", // PowerPoint
			"**/*.{xls,xlsx}", // Excel
			"**/*.pdf",        // PDF
		},
		ExcludeGlobs: []string{
			"**/vendor/**",       // Vendor directories
			"**/.git/**",         // Git directories
			"**/.github/**",      // Github directories
			"**/node_modules/**", // Node.js modules
			"**/__pycache__/**",  // Python cache
			"**/bin/**",          // Binary directories
			"**/obj/**",          // Object files
			"**/build/**",        // Build directories
			"**/dist/**",         // Distribution directories
			"**/.vitepress/**",   // Vitepress directories
			"**/.idea/**",        // IDE directories
			"**/.vscode/**",      // VS Code directories
			"**/*.min.*",         // Minified files
			"**/*.map",           // Source maps
			"**/*.generated.*",   // Generated files
		},
	}
}

// LoadConfigFromFile loads configuration from a YAML or JSON file
func LoadConfigFromFile(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config file path is empty")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(configPath))

	var config Config

	if len(data) == 0 {
		// For empty files, return default config
		defaultConfig := DefaultConfig()
		return &defaultConfig, nil
	}

	if ext == ".yml" || ext == ".yaml" {
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return nil, fmt.Errorf("error parsing YAML config: %w", err)
		}
		fmt.Println("YAML config loaded successfully", config)
	} else if ext == ".json" {
		err = json.Unmarshal(data, &config)
		if err != nil {
			return nil, fmt.Errorf("error parsing JSON config: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	return &config, nil
}

// Helper function to compare slices
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MergeConfig merges the provided config with an auto-loaded config, letting provided config take precedence
func MergeConfig(config Config, autoConfig *Config) Config {
	// If auto-config is nil, return the original config
	if autoConfig == nil {
		return config
	}

	// Create a copy of the config to work with
	mergedConfig := config

	// If the config is effectively empty (all zero values), use the auto-config
	if isEmptyConfig(config) {
		mergedConfig = *autoConfig
		// If user specified an input directory, use that instead
		if config.InputDir != "" {
			mergedConfig.InputDir = config.InputDir
		}
		return mergedConfig
	}

	// For non-empty configs, only fill in empty fields from auto-config
	if mergedConfig.InputDir == "" {
		mergedConfig.InputDir = autoConfig.InputDir
	}

	if mergedConfig.OutputFile == "" {
		mergedConfig.OutputFile = autoConfig.OutputFile
	}

	// For globs, if the config has patterns, use them as-is
	// Otherwise use the auto-config's patterns
	if len(mergedConfig.IncludeGlobs) == 0 {
		mergedConfig.IncludeGlobs = autoConfig.IncludeGlobs
	}

	if len(mergedConfig.ExcludeGlobs) == 0 {
		mergedConfig.ExcludeGlobs = autoConfig.ExcludeGlobs
	}

	return mergedConfig
}

// isEmptyConfig checks if a config is effectively empty (all zero values)
func isEmptyConfig(config Config) bool {
	return config.OutputFile == "" &&
		len(config.IncludeGlobs) == 0 &&
		len(config.ExcludeGlobs) == 0 &&
		!config.Verbose &&
		!config.Compress &&
		!config.MaxCompress &&
		!config.Gzip &&
		!config.Base64
}

// ApplyDefaults applies default values to empty fields in the config
func ApplyDefaults(config Config) Config {
	defaults := DefaultConfig()

	// Apply defaults for empty fields
	if config.InputDir == "" {
		config.InputDir = defaults.InputDir
	}

	// Handle output file name and gzip extension
	if config.OutputFile == "" {
		if config.Gzip {
			config.OutputFile = "corpus-out.txt.gz"
		} else {
			config.OutputFile = "corpus-out.txt"
		}
	} else if config.Gzip && !strings.HasSuffix(config.OutputFile, ".gz") &&
		!strings.Contains(config.OutputFile, ".gz.") {
		config.OutputFile += ".gz"
	}

	// Apply default globs if empty
	if config.IncludeGlobs == nil {
		config.IncludeGlobs = defaults.IncludeGlobs
	}
	if config.ExcludeGlobs == nil {
		config.ExcludeGlobs = defaults.ExcludeGlobs
	}

	return config
}
