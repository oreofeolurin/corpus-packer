# Corpus Packer (cpack)

Corpus Packer (cpack) is a powerful command-line tool built in Go for combining multiple files from directories into a single output file while preserving file structures, file extensions, and ignore patterns. It is perfect for tasks such as compiling training datasets, creating documentation bundles, or merging log files for analysis.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Command Line Options](#command-line-options)
- [Configuration File](#configuration-file)
- [Output Formats](#output-formats)
- [Examples](#examples)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features

- **File Aggregation**: Combine multiple files into a single output file while preserving content integrity.
- **Extension Based Filtering**: Process only files with specified extensions.
- **Ignore Patterns**: Exclude files and directories matching specific patterns.
- **Directory Traversal**: Recursively search through directories.
- **Smart Extension Handling**: Use file extensions with or without a leading dot effortlessly.
- **Custom Output**: Specify the destination file for the aggregated content.
- **Multiple Output Formats**: Support for compressed, gzipped, and base64 encoded output.
- **Flexible CLI**: Intuitive command-line interface with numerous options to tailor behavior to your needs.
- **Configuration Files**: Support for YAML and JSON configuration files.

## Installation

Corpus Packer requires Go (version 1.16 or higher) to build from source.

### Using go install

Install directly using the following command:

```
go install github.com/oreofeolurin/corpus-packer/cpack@latest
```

### Building from Source

Clone the repository and build it with:

```
git clone https://github.com/oreofeolurin/corpus-packer.git
cd corpus-packer
go build -o cpack
```

## Usage

Run Corpus Packer with minimal options to combine files from the current directory:

```
cpack
```

Alternatively, you can specify an input directory, output file, or filtering options as needed.

## Command Line Options

| Flag               | Short | Description                                           | Default           |
|-------------------|-------|-------------------------------------------------------|-------------------|
| `--dir`           | `-d`  | Input directory to process                            | Current directory |
| `--output`        | `-o`  | Output file path                                      | corpus-out.txt    |
| `--include`       | `-i`  | Glob patterns to include                              | All supported types |
| `--exclude`       | `-x`  | Glob patterns to exclude                              | Common test/vendor |
| `--compress`      | `-c`  | Compress output by removing whitespace                | false             |
| `--max-compress`  | `-m`  | Maximum compression (remove comments)                  | false             |
| `--gzip`          | `-z`  | Compress output file using gzip                       | false             |
| `--base64`        | `-b`  | Base64 encode the output (use with --gzip)           | false             |
| `--verbose`       | `-v`  | Include summary at start of output                    | false             |

## Configuration File

You can use a configuration file in either YAML or JSON format to specify your settings. This is particularly useful for complex configurations or when you want to reuse the same settings across multiple runs.

### YAML Configuration Example

```yaml
inputDir: ./src
outputFile: output.txt
includeGlobs:
  - "**/*.go"
  - "**/*.py"
  - "src/**/*.js"
excludeGlobs:
  - "**/*_test.go"
  - "**/vendor/**"
  - "**/.git/**"
  - "**/node_modules/**"
verbose: true
compress: false
maxCompress: false
gzip: false
base64: false
```

### JSON Configuration Example

```json
{
  "inputDir": "./src",
  "outputFile": "output.txt",
  "includeGlobs": ["**/*.go", "**/*.py", "src/**/*.js"],
  "excludeGlobs": ["**/*_test.go", "**/vendor/**", "**/.git/**", "**/node_modules/**"],
  "verbose": true,
  "compress": false,
  "maxCompress": false,
  "gzip": false,
  "base64": false
}
```

To use a configuration file:

```bash
cpack -c config.yaml -o custom-output.txt
```

Command line arguments take precedence over configuration file settings, allowing you to override specific values when needed.

## Output Formats

Corpus Packer supports multiple output formats to suit different needs:

1. **Standard Output (Default)**
   - Plain text output with original formatting preserved
   - File separators and content structure maintained

2. **Compressed Output** (`--compress`)
   - Removes unnecessary whitespace
   - Preserves essential formatting
   - Reduces file size while maintaining readability

3. **Maximum Compressed Output** (`--max-compress`)
   - Removes all comments and unnecessary whitespace
   - Minimal file size with reduced readability
   - Best for machine processing or size constraints

4. **Gzipped Output** (`--gzip`)
   - Compresses output using gzip
   - Automatically adds .gz extension if not present
   - Significant size reduction for text-based files

5. **Base64 Encoded Gzipped Output** (`--gzip --base64`)
   - Gzips the output and then base64 encodes it
   - Useful for systems that require base64 encoding
   - Must be used with gzip option

## Examples

1. Process only Go files in specific directories:

```bash
cpack -i "**/*.go" -i "src/**/*.go" -i "internal/**/*.go"
```

2. Process multiple file types with compression:

```bash
cpack -i "src/**/*.{go,py,js}" -c -o compressed.txt
```

3. Maximum compression with gzip:

```bash
cpack -m -z -o output.txt.gz
```

4. Gzip with base64 encoding:

```bash
cpack -z -b -o output.txt.gz.b64
```

5. Verbose output with compression:

```bash
cpack -v -c -o output.txt
```

6. Complete example with directory filtering and excluding patterns:

```bash
cpack \
  -d ./project \
  -o combined.txt \
  -i "src/**/*.go" \
  -i "internal/**/*.go" \
  -i "pkg/**/*.go" \
  -x "**/*_test.go" \
  -x "**/testdata/**" \
  -x "**/vendor/**" \
  -x "**/.git/**" \
  -c -v
```

7. Using a configuration file with overrides:

```bash
cpack -c config.yaml -o custom-output.txt -z
```

## Configuration

Corpus Packer provides flexibility through its command line flags and configuration files. Customize it to match your project structure and ignore patterns:

- **File Extensions**: Specify valid extensions with or without a leading dot.
- **Ignore Patterns**: Use glob patterns to exclude files or directories that should not be processed.
- **Configuration Files**: Use YAML or JSON files for complex configurations.
- **Output Formats**: Choose from multiple output formats based on your needs.

## Troubleshooting

- Ensure you are using Go version 1.16 or higher.
- Verify directory permissions if files are not being processed as expected.
- Double-check your use of the `--exclude` option in case required files are inadvertently excluded.
- When using configuration files, ensure they are properly formatted YAML or JSON.
- For gzipped output, ensure the target directory is writable.
- Base64 encoding requires the gzip option to be enabled.
- Refer back to the examples and command line options for guidance if issues arise.

## Contributing

Contributions are welcome! Please consult our [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to help improve Corpus Packer. Open issues or submit pull requests for bug fixes, feature enhancements, or other improvements.

## License

Corpus Packer is licensed under the [MIT License](LICENSE).