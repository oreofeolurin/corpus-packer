package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	config  Config
	rootCmd = &cobra.Command{
		Use:   "cpack [directory]",
		Short: "A tool for packing source code into a corpus file",
		Long: `Corpus Packer (cpack) is a tool that helps you create a corpus file from your source code.
It can process multiple file types and directories while respecting ignore patterns.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If directory argument is provided, use it
			if len(args) > 0 {
				config.InputDir = args[0]
			}
			return ProcessDirectory(config)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	defaults := DefaultConfig()

	// Input/Output flags
	rootCmd.Flags().StringVarP(&config.OutputFile, "output", "o", defaults.OutputFile,
		"Output file path (default: corpus-out.txt or corpus-out.txt.gz with --gzip)")
	rootCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", defaults.Verbose,
		"Include summary at the start of output file")
	rootCmd.Flags().BoolVarP(&config.Compress, "compress", "c", defaults.Compress,
		"Compress output by removing extra whitespace")
	rootCmd.Flags().BoolVarP(&config.AggressiveCompress, "max-compress", "m", defaults.AggressiveCompress,
		"Maximum compression: remove comments and all unnecessary whitespace")
	rootCmd.Flags().BoolVarP(&config.Gzip, "gzip", "z", defaults.Gzip,
		"Compress output file using gzip")
	rootCmd.Flags().BoolVarP(&config.Base64, "base64", "b", defaults.Base64,
		"Base64 encode the output (use with --gzip)")

	// File type flags
	rootCmd.Flags().StringSliceVarP(&config.ValidExtensions, "extensions", "e", defaults.ValidExtensions,
		"File extensions to process (e.g., .go,.js,.py)")
	rootCmd.Flags().StringSliceVarP(&config.ValidDirs, "include-glob", "g", defaults.ValidDirs,
		"Glob patterns for directories to include (e.g., 'src/**/pkg', 'internal/*')")

	// Ignore pattern flags
	rootCmd.Flags().StringSliceVarP(&config.IgnoreDirs, "exclude-glob", "x", defaults.IgnoreDirs,
		"Glob patterns for directories to exclude (e.g., '**/vendor', '**/.git', '**/node_modules')")
	rootCmd.Flags().StringSliceVarP(&config.IgnorePatterns, "ignore-patterns", "p", defaults.IgnorePatterns,
		"File patterns to ignore (e.g., '*_test.go', '*.min.js')")

	// Ensure paths are cleaned
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		config.InputDir = filepath.Clean(config.InputDir)
		config.OutputFile = filepath.Clean(config.OutputFile)
		return nil
	}
}
