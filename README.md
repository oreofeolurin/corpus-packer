# cpack (Corpus Packer)

[![Go Version](https://img.shields.io/github/go-mod/go-version/oreofeolurin/corpus-packer)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A command-line tool that combines multiple files from a directory into a single output file while respecting file extensions and ignore patterns. Perfect for creating training datasets, documentation compilations, or any task requiring file content aggregation.

## Features

- 📁 Combine multiple files into a single output file
- 🔍 Filter files by extension
- 🚫 Ignore specific files and directories using patterns
- 📑 Clear file separation with start/end markers
- 🌳 Support for recursive directory traversal
- ⚡ Flexible command-line interface
- 🔒 Preserves file content integrity
- 🎯 Smart extension handling (with or without dots)

## Installation

### Prerequisites

- Go 1.16 or higher
- Git (for building from source)

### Using go install

```bash
go install github.com/oreofeolurin/corpus-packer/cpack@latest
```

### Building from source

```bash
git clone https://github.com/oreofeolurin/corpus-packer.git
cd corpus-packer
go build -o cpack
```

## Command Line Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--input` | `-i` | Input directory to process | Current directory |
| `--output` | `-o` | Output file path | `corpus-packer-out.txt` |
| `--valid` | `-v` | Valid file extensions | `.txt,.csv,.json,.js,.html,.go,.py` |
| `--dirs` | `-d` | Directories to include | All directories |
| `--ignore` | `-x` | Patterns to ignore (files) | `*.min.js,*.lock` |
| `--ignore-dirs` | `-D` | Directory patterns to ignore | `**/.*` |

### Examples

1. Process only specific directories:
```bash
cpack -d "src,lib,internal"
```

2. Process specific directories with specific file types:
```bash
cpack -d "src,pkg" -v "go,md"
```

3. Complete example with directory filtering:
```bash
cpack \
  -i ./project \
  -o combined.txt \
  -d "src,internal,pkg" \
  -v "go,md" \
  -x "*.test.go" \
  -D "**/testdata,**/vendor"
```
