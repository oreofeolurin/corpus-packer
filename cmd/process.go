package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Summary holds processing statistics
type Summary struct {
	TotalFiles     int
	ProcessedFiles []string
	SkippedFiles   []string
	TotalBytes     int64
	StartTime      time.Time
	EndTime        time.Time
}

type fileProcessor struct {
	config         *Config
	outputFile     io.Writer
	contentBuffer  *bytes.Buffer
	processedFiles map[string]bool
	summary        *Summary
}

// ProcessDirectory processes files in the given directory according to the config
func ProcessDirectory(config Config) error {
	// Try to load default config file if it exists
	if autoConfig, err := tryLoadDefaultConfig(config.InputDir); err == nil {
		config = MergeConfig(config, autoConfig)
	}

	// Apply defaults for empty fields
	config = ApplyDefaults(config)

	// Get current working directory
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		return fmt.Errorf("error getting current working directory: %w", cwdErr)
	}

	// Make output file path relative to current working directory if not absolute
	if !filepath.IsAbs(config.OutputFile) {
		config.OutputFile = filepath.Join(cwd, config.OutputFile)
	}

	// Validate input directory first
	if err := validateConfig(&config); err != nil {
		return err
	}

	// Create output directory if needed
	outputDir := filepath.Dir(config.OutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		// If it's a read-only filesystem error, wrap it as an output file error
		if strings.Contains(err.Error(), "read-only file system") {
			return fmt.Errorf("error creating output file: %w", err)
		}
		return fmt.Errorf("error creating output directory: %w", err)
	}

	var (
		outputFile   *os.File
		gzipWriter   *gzip.Writer
		base64Writer io.WriteCloser
		writer       io.Writer
	)

	outputFile, err := os.Create(config.OutputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outputFile.Close()

	writer = outputFile

	// Create writer chain in correct order
	if config.Base64 {
		if !config.Gzip {
			return fmt.Errorf("--base64 requires --gzip")
		}
		base64Writer = base64.NewEncoder(base64.StdEncoding, outputFile)
		writer = base64Writer
	}

	if config.Gzip {
		gzipWriter = gzip.NewWriter(writer)
		writer = gzipWriter
	}

	// If verbose, write to buffer first
	var contentBuffer *bytes.Buffer
	if config.Verbose {
		contentBuffer = &bytes.Buffer{}
	}

	processor := &fileProcessor{
		config:         &config,
		outputFile:     writer,
		contentBuffer:  contentBuffer,
		processedFiles: make(map[string]bool),
		summary: &Summary{
			StartTime: time.Now(),
		},
	}

	err = filepath.Walk(config.InputDir, processor.processPath)
	if err != nil {
		return err
	}

	processor.summary.EndTime = time.Now()

	if config.Verbose {
		if err := processor.writeSummary(); err != nil {
			return err
		}

		if _, err := writer.Write(contentBuffer.Bytes()); err != nil {
			return fmt.Errorf("error writing file content: %w", err)
		}
	}

	// Close in reverse order
	if config.Gzip {
		if err := gzipWriter.Close(); err != nil {
			return fmt.Errorf("error closing gzip writer: %w", err)
		}
	}

	if config.Base64 {
		if err := base64Writer.Close(); err != nil {
			return fmt.Errorf("error closing base64 encoder: %w", err)
		}
	}

	return nil
}

// ProcessDirectoryWithConfigFile processes files using configuration from a file
func ProcessDirectoryWithConfigFile(configPath string, overrideConfig Config) error {
	// Load config from file
	fileConfig, err := LoadConfigFromFile(configPath)
	if err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current working directory: %w", err)
	}

	// Create a new config that will hold the merged values
	mergedConfig := *fileConfig

	// Handle input directory
	if overrideConfig.InputDir != "" {
		mergedConfig.InputDir = overrideConfig.InputDir
	} else if !filepath.IsAbs(mergedConfig.InputDir) {
		// Make input directory relative to current working directory
		mergedConfig.InputDir = filepath.Join(cwd, mergedConfig.InputDir)
	}

	// Handle output file path
	if overrideConfig.OutputFile != "" {
		mergedConfig.OutputFile = overrideConfig.OutputFile
	} else if !filepath.IsAbs(mergedConfig.OutputFile) {
		// Make output file relative to current working directory
		mergedConfig.OutputFile = filepath.Join(cwd, mergedConfig.OutputFile)
	}

	// Create output directory if needed
	outputDir := filepath.Dir(mergedConfig.OutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		// If it's a read-only filesystem error, wrap it as an output file error
		if strings.Contains(err.Error(), "read-only file system") {
			return fmt.Errorf("error creating output file: %w", err)
		}
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Handle include patterns - override takes precedence over file config
	if len(overrideConfig.IncludeGlobs) > 0 {
		mergedConfig.IncludeGlobs = overrideConfig.IncludeGlobs
	}

	// Handle exclude patterns - override takes precedence over file config
	if len(overrideConfig.ExcludeGlobs) > 0 {
		mergedConfig.ExcludeGlobs = overrideConfig.ExcludeGlobs
	}

	// Handle boolean flags - override takes precedence over file config
	if overrideConfig.Verbose {
		mergedConfig.Verbose = true
	}
	if overrideConfig.Compress {
		mergedConfig.Compress = true
	}
	if overrideConfig.MaxCompress {
		mergedConfig.MaxCompress = true
	}
	if overrideConfig.Gzip {
		mergedConfig.Gzip = true
	}
	if overrideConfig.Base64 {
		mergedConfig.Base64 = true
	}

	// Process with merged config
	return ProcessDirectory(mergedConfig)
}

func (p *fileProcessor) processPath(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %s: %v\n", path, err)
		return nil
	}

	// Get absolute path for the file
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path for %s: %v\n", path, err)
		return nil
	}

	// Get absolute path for input directory
	absInputDir, err := filepath.Abs(p.config.InputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path for input directory: %v\n", err)
		return nil
	}

	// Calculate relative path from input directory
	relPath, err := filepath.Rel(absInputDir, absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting relative path for %s: %v\n", path, err)
		return nil
	}

	if p.processedFiles[relPath] {
		return nil
	}

	if info.IsDir() {
		return p.processDirectory(relPath)
	}

	return p.processFile(relPath, absPath)
}

func (p *fileProcessor) processDirectory(relPath string) error {
	if p.shouldIgnoreDir(relPath) {
		return filepath.SkipDir
	}

	if !p.isValidDir(relPath) {
		return filepath.SkipDir
	}

	return nil
}

// matchGlobPattern checks if a path matches a glob pattern, properly handling ** patterns
func matchGlobPattern(pattern, path string) (bool, error) {
	// Convert pattern to regex
	pattern = filepath.Clean(pattern)
	path = filepath.Clean(path)

	// Make file extensions case insensitive by converting both to lowercase
	// Only do this for the extension part to preserve case sensitivity for directories
	patternExt := filepath.Ext(pattern)
	pathExt := filepath.Ext(path)
	if patternExt != "" && pathExt != "" {
		pattern = pattern[:len(pattern)-len(patternExt)] + strings.ToLower(patternExt)
		path = path[:len(path)-len(pathExt)] + strings.ToLower(pathExt)
	}

	// Escape special characters except * and ?
	regexPattern := regexp.QuoteMeta(pattern)

	// Handle special case where pattern starts with **/ or contains /**/ or ends with /**
	regexPattern = strings.ReplaceAll(regexPattern, "\\*\\*/", "(?:.*/)?")
	regexPattern = strings.ReplaceAll(regexPattern, "/\\*\\*/", "/(?:.*/)?")
	regexPattern = strings.ReplaceAll(regexPattern, "\\*\\*", ".*")

	// Replace * with non-separator match
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", "[^/]*")

	// Replace ? with single non-separator match
	regexPattern = strings.ReplaceAll(regexPattern, "\\?", "[^/]")

	// Ensure pattern matches the entire path
	regexPattern = "^" + regexPattern + "$"

	// Compile and match
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return false, fmt.Errorf("invalid pattern %s: %v", pattern, err)
	}

	return regex.MatchString(path), nil
}

func (p *fileProcessor) shouldIgnoreDir(relPath string) bool {
	for _, pattern := range p.config.ExcludeGlobs {
		matched, err := matchGlobPattern(pattern, relPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching directory pattern %s: %v\n", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func (p *fileProcessor) isValidDir(relPath string) bool {
	if len(p.config.IncludeGlobs) == 0 {
		return true
	}

	// Always allow the root directory (empty or "." path)
	if relPath == "" || relPath == "." {
		return true
	}

	// Clean the path
	relPathClean := filepath.Clean(relPath)

	// Check if this directory or any of its children could match any include pattern
	for _, pattern := range p.config.IncludeGlobs {
		// For patterns with **, check if this directory could be part of a valid path
		if strings.Contains(pattern, "**") {
			// Get the part before the first **
			parts := strings.Split(pattern, "**")
			prefix := parts[0]

			// If no prefix (pattern starts with **), allow the directory
			if prefix == "" {
				return true
			}

			// If there's a prefix, check if this directory matches or could contain matching files
			if strings.HasPrefix(relPathClean, prefix) || strings.HasPrefix(prefix, relPathClean) {
				return true
			}
			continue
		}

		// For non-** patterns, check if this directory is part of the pattern path
		patternDir := filepath.Dir(pattern)
		if patternDir == "." || strings.HasPrefix(relPathClean, patternDir) || strings.HasPrefix(patternDir, relPathClean) {
			return true
		}
	}

	return false
}

// Helper function to write string to io.Writer
func writeString(w io.Writer, s string) error {
	_, err := w.Write([]byte(s))
	return err
}

func (p *fileProcessor) processFile(relPath, path string) error {
	if !p.isValidFile(relPath, path) {
		p.summary.SkippedFiles = append(p.summary.SkippedFiles, relPath)
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
		p.summary.SkippedFiles = append(p.summary.SkippedFiles, relPath+" (read error)")
		return nil
	}

	p.summary.ProcessedFiles = append(p.summary.ProcessedFiles, relPath)
	p.summary.TotalBytes += int64(len(content))

	// Create separators
	startSeparator := fmt.Sprintf("--- START OF FILE: %s ---\n", relPath)
	endSeparator := fmt.Sprintf("\n--- END OF FILE: %s ---\n\n", relPath)

	// Apply compression if enabled
	if p.config.Compress {
		content = compressContent(content, p.config)
		// Also compress separators
		startSeparator = strings.TrimSpace(startSeparator) + " "
		endSeparator = " " + strings.TrimSpace(endSeparator) + " "
	}

	if p.config.Verbose {
		if _, err = p.contentBuffer.WriteString(startSeparator); err != nil {
			return fmt.Errorf("error writing separator to buffer: %w", err)
		}
		if _, err = p.contentBuffer.Write(content); err != nil {
			return fmt.Errorf("error writing content to buffer: %w", err)
		}
		if _, err = p.contentBuffer.WriteString(endSeparator); err != nil {
			return fmt.Errorf("error writing separator to buffer: %w", err)
		}
	} else {
		if err = writeString(p.outputFile, startSeparator); err != nil {
			return fmt.Errorf("error writing separator to output file: %w", err)
		}
		if _, err = p.outputFile.Write(content); err != nil {
			return fmt.Errorf("error writing content to output file: %w", err)
		}
		if err = writeString(p.outputFile, endSeparator); err != nil {
			return fmt.Errorf("error writing separator to output file: %w", err)
		}
	}

	p.processedFiles[relPath] = true
	return nil
}

func (p *fileProcessor) isValidFile(relPath, path string) bool {
	// First check if it matches any ignore patterns
	for _, pattern := range p.config.ExcludeGlobs {
		// For patterns without /, match against base name
		if !strings.Contains(pattern, "/") {
			matched, err := matchGlobPattern(pattern, filepath.Base(relPath))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error matching file pattern %s: %v\n", pattern, err)
				continue
			}
			if matched {
				return false
			}
			continue
		}

		// For patterns with /, match against full path
		matched, err := matchGlobPattern(pattern, relPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching file pattern %s: %v\n", pattern, err)
			continue
		}
		if matched {
			return false
		}
	}

	// Then check if it matches any include patterns
	if len(p.config.IncludeGlobs) == 0 {
		return true // If no include patterns specified, accept all files
	}

	for _, pattern := range p.config.IncludeGlobs {
		// For patterns without /, match against base name
		if !strings.Contains(pattern, "/") {
			matched, err := matchGlobPattern(pattern, filepath.Base(relPath))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error matching include pattern %s: %v\n", pattern, err)
				continue
			}
			if matched {
				return true
			}
			continue
		}

		// For patterns with /, match against full path
		matched, err := matchGlobPattern(pattern, relPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching include pattern %s: %v\n", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}

	return false
}

func (p *fileProcessor) writeSummary() error {
	duration := p.summary.EndTime.Sub(p.summary.StartTime)

	// Sort files for consistent output
	sort.Strings(p.summary.ProcessedFiles)
	sort.Strings(p.summary.SkippedFiles)

	summary := fmt.Sprintf(`--- CORPUS PACKER SUMMARY ---
Processing Time: %v
Total Files: %d
Total Files Processed: %d
Total Files Skipped: %d
Total Bytes Processed: %d

Processed Files:
%s

Skipped Files:
%s

--- END OF SUMMARY ---

`,
		duration,
		len(p.summary.ProcessedFiles)+len(p.summary.SkippedFiles),
		len(p.summary.ProcessedFiles),
		len(p.summary.SkippedFiles),
		p.summary.TotalBytes,
		strings.Join(p.summary.ProcessedFiles, "\n"),
		strings.Join(p.summary.SkippedFiles, "\n"),
	)

	// Apply compression if enabled
	if p.config.Compress {
		summary = string(compressContent([]byte(summary), p.config))
	}

	return writeString(p.outputFile, summary)
}

func validateConfig(config *Config) error {
	// Clean and validate input directory
	if !filepath.IsAbs(config.InputDir) {
		// Get absolute path relative to current working directory
		absPath, err := filepath.Abs(config.InputDir)
		if err != nil {
			return fmt.Errorf("error resolving input directory path: %w", err)
		}
		config.InputDir = absPath
	}

	if _, err := os.Stat(config.InputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", config.InputDir)
	}

	// Clean output file path
	if !filepath.IsAbs(config.OutputFile) {
		// Get absolute path relative to current working directory
		absPath, err := filepath.Abs(config.OutputFile)
		if err != nil {
			return fmt.Errorf("error resolving output file path: %w", err)
		}
		config.OutputFile = absPath
	}

	// Clean glob patterns
	for i, pattern := range config.IncludeGlobs {
		config.IncludeGlobs[i] = filepath.Clean(pattern)
	}
	for i, pattern := range config.ExcludeGlobs {
		config.ExcludeGlobs[i] = filepath.Clean(pattern)
	}

	return nil
}

func compressContent(content []byte, config *Config) []byte {
	str := string(content)

	// Remove comments if aggressive compression is enabled
	if config.MaxCompress {
		str = removeComments(str)
	}

	// Replace all whitespace with single space
	str = strings.Join(strings.Fields(strings.ReplaceAll(str, "\n", " ")), " ")

	// Remove spaces around more symbols
	symbols := []string{".", ",", ":", ";", ")", "(", "{", "}", "[", "]",
		"+", "-", "*", "/", "=", "<", ">", "&", "|", "!", "?"}
	for _, sym := range symbols {
		str = strings.ReplaceAll(str, " "+sym+" ", sym)
		str = strings.ReplaceAll(str, " "+sym, sym)
		str = strings.ReplaceAll(str, sym+" ", sym)
	}

	// Remove spaces between word and number
	wordNumRegex := regexp.MustCompile(`(\w+)\s+(\d+)`)
	str = wordNumRegex.ReplaceAllString(str, "$1$2")

	// Optional: Remove file separators in non-verbose mode
	if !config.Verbose {
		str = strings.ReplaceAll(str, "--- START OF FILE:", "")
		str = strings.ReplaceAll(str, "--- END OF FILE:", "")
	}

	return []byte(str)
}

// Helper function to remove comments
func removeComments(str string) string {
	// Remove single-line comments
	singleLine := regexp.MustCompile(`//.*`)
	str = singleLine.ReplaceAllString(str, "")

	// Remove multi-line comments
	multiLine := regexp.MustCompile(`(?s)/\*.*?\*/`)
	str = multiLine.ReplaceAllString(str, "")

	return str
}

// tryLoadDefaultConfig attempts to load a config file from the default locations
func tryLoadDefaultConfig(dir string) (*Config, error) {
	// Check for cpack.yml first
	ymlPath := filepath.Join(dir, "cpack.yml")
	if _, err := os.Stat(ymlPath); err == nil {
		return LoadConfigFromFile(ymlPath)
	}

	// Then check for cpack.yaml
	yamlPath := filepath.Join(dir, "cpack.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return LoadConfigFromFile(yamlPath)
	}

	// Finally check for cpack.json
	jsonPath := filepath.Join(dir, "cpack.json")
	if _, err := os.Stat(jsonPath); err == nil {
		return LoadConfigFromFile(jsonPath)
	}

	return nil, fmt.Errorf("no default config file found")
}
