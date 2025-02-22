package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Config holds the program's configuration
type Config struct {
	InputDir           string
	OutputFile         string
	ValidExtensions    []string
	ValidDirs          []string
	IgnorePatterns     []string
	IgnoreDirs         []string
	Verbose            bool
	Compress           bool
	AggressiveCompress bool
	Gzip               bool
	Base64             bool
}

// Summary holds processing statistics
type Summary struct {
	TotalFiles     int
	ProcessedFiles []string
	SkippedFiles   []string
	TotalBytes     int64
	StartTime      time.Time
	EndTime        time.Time
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
		ValidExtensions: []string{
			".go",    // Go source files
			".js",    // JavaScript
			".ts",    // TypeScript
			".css",   // CSS
			".py",    // Python
			".java",  // Java
			".cpp",   // C++
			".c",     // C
			".h",     // Header files
			".hpp",   // C++ headers
			".rb",    // Ruby
			".php",   // PHP
			".cs",    // C#
			".swift", // Swift
			".kt",    // Kotlin
			".md",    // Markdown
			".tsx",   // TypeScript React
			".jsx",   // JavaScript React
			".ts",    // TypeScript
			".tsx",   // TypeScript React
			".jsx",   // JavaScript React
			".json",  // JSON
			".yaml",  // YAML
			".yml",   // YAML
			".toml",  // TOML
			//".csv",   // CSV
			".txt", // TXT
			//	".tsv",   // TSV
			".xml",  // XML
			".docx", // Docx
			".pptx", // Pptx
			".xlsx", // Xlsx
			".xls",  // Xls
			".doc",  // Doc
			".ppt",  // Ppt
			".pdf",  // Pdf
		},
		IgnoreDirs: []string{
			"**/vendor",       // Vendor directories
			"**/.git",         // Git directories
			"**/.github",      // Github directories
			"**/node_modules", // Node.js modules
			"**/__pycache__",  // Python cache
			"**/bin",          // Binary directories
			"**/obj",          // Object files
			"**/build",        // Build directories
			"**/dist",         // Distribution directories
			//"**/dist",         // Distribution directories
			"**/.vitepress", // Vitepress directories
			"**/.idea",      // IDE directories
			"**/.vscode",    // VS Code directories
		},
		IgnorePatterns: []string{
			"*_test.go",     // Go test files
			"*.test.*",      // Test files
			"*.spec.*",      // Test specs
			"*.min.*",       // Minified files
			"*.map",         // Source maps
			"*.generated.*", // Generated files
		},
	}
}

// ProcessDirectory processes files in the given directory according to the config
func ProcessDirectory(config Config) error {
	// Apply defaults for empty fields
	if config.InputDir == "" {
		config.InputDir = "."
	}

	// Handle output file name and gzip extension
	if config.OutputFile == "" || config.OutputFile == "corpus-out.txt" {
		if config.Gzip {
			config.OutputFile = "corpus-out.txt.gz"
		} else {
			config.OutputFile = "corpus-out.txt"
		}
	} else if config.Gzip && !strings.HasSuffix(config.OutputFile, ".gz") &&
		!strings.Contains(config.OutputFile, ".gz.") {
		config.OutputFile += ".gz"
	}

	// Only apply default extensions if the field is nil, not if it's empty
	if config.ValidExtensions == nil {
		config.ValidExtensions = DefaultConfig().ValidExtensions
	}
	if len(config.IgnoreDirs) == 0 {
		config.IgnoreDirs = DefaultConfig().IgnoreDirs
	}
	if len(config.IgnorePatterns) == 0 {
		config.IgnorePatterns = DefaultConfig().IgnorePatterns
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

type fileProcessor struct {
	config         *Config
	outputFile     io.Writer
	contentBuffer  *bytes.Buffer
	processedFiles map[string]bool
	summary        *Summary
}

func (p *fileProcessor) processPath(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %s: %v\n", path, err)
		return nil
	}

	relPath, err := filepath.Rel(p.config.InputDir, path)
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

	return p.processFile(relPath, path)
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

func (p *fileProcessor) shouldIgnoreDir(relPath string) bool {
	for _, pattern := range p.config.IgnoreDirs {
		pattern = strings.TrimPrefix(pattern, "**/")
		matched, err := filepath.Match(pattern, filepath.Base(relPath))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching directory pattern %s: %v\n", pattern, err)
			continue
		}
		if matched {
			fmt.Printf("Skipping directory (pattern match): %s (pattern: %s)\n", relPath, pattern)
			return true
		}
	}
	return false
}

func (p *fileProcessor) isValidDir(relPath string) bool {
	if len(p.config.ValidDirs) == 0 {
		return true
	}

	relPathClean := filepath.Clean(relPath)
	if relPathClean == "." {
		return p.isValidRootDir()
	}

	return p.isValidSubDir(relPathClean)
}

func (p *fileProcessor) isValidRootDir() bool {
	for _, validDir := range p.config.ValidDirs {
		if strings.Contains(validDir, "/") {
			fmt.Printf("Root directory is parent of: %s\n", validDir)
			return true
		}
	}
	return false
}

func (p *fileProcessor) isValidSubDir(relPathClean string) bool {
	for _, validDir := range p.config.ValidDirs {
		validDirClean := filepath.Clean(validDir)
		fmt.Printf("Checking dir: %s against valid dir: %s\n", relPathClean, validDirClean)

		// Check parent directory relationship
		if strings.HasPrefix(validDirClean, relPathClean+"/") {
			fmt.Printf("Found parent directory match: %s is parent of %s\n", relPathClean, validDirClean)
			return true
		}

		// Check child directory relationship
		if strings.HasPrefix(relPathClean, validDirClean+"/") {
			fmt.Printf("Found child directory match: %s is child of %s\n", relPathClean, validDirClean)
			return true
		}

		// Check exact match
		if relPathClean == validDirClean {
			fmt.Printf("Found exact directory match: %s\n", relPathClean)
			return true
		}
	}
	fmt.Printf("Skipping invalid directory: %s\n", relPathClean)
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

	content, err := ioutil.ReadFile(path)
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
	ext := strings.ToLower(filepath.Ext(path))
	isValid := false
	for _, validExt := range p.config.ValidExtensions {
		if ext == validExt {
			isValid = true
			break
		}
	}

	if !isValid {
		return false
	}

	for _, pattern := range p.config.IgnorePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching file pattern %s: %v\n", pattern, err)
			continue
		}
		if matched {
			return false
		}
	}

	return true
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
	config.InputDir = filepath.Clean(config.InputDir)
	if _, err := os.Stat(config.InputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", config.InputDir)
	}

	// Clean output file path
	config.OutputFile = filepath.Clean(config.OutputFile)

	// Clean valid directories
	for i, dir := range config.ValidDirs {
		config.ValidDirs[i] = filepath.Clean(dir)
	}

	// Normalize extensions (convert to lowercase and remove duplicates)
	validExts := make(map[string]bool)
	for _, ext := range config.ValidExtensions {
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		validExts[ext] = true
	}

	// Convert back to slice in a deterministic order
	normalizedExts := make([]string, 0, len(validExts))
	for ext := range validExts {
		normalizedExts = append(normalizedExts, ext)
	}
	config.ValidExtensions = normalizedExts

	return nil
}

func compressContent(content []byte, config *Config) []byte {
	str := string(content)

	// Remove comments if aggressive compression is enabled
	if config.AggressiveCompress {
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
