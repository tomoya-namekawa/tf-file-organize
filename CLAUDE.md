# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

terraform-file-organize is a Go CLI tool that parses Terraform files and splits them into separate files organized by resource type. The tool uses HashiCorp's HCL parser to analyze Terraform configurations and reorganizes them according to specific naming conventions. It supports both single file and directory input, with optional YAML configuration for custom grouping rules.

## Build and Development Commands

```bash
# Build the project
go build -o terraform-file-organize

# Basic usage examples
./terraform-file-organize main.tf --dry-run
./terraform-file-organize . --output-dir tmp/test --dry-run
./terraform-file-organize testdata/terraform --config testdata/configs/terraform-file-organize.yaml

# Development commands
go mod tidy

# Run all tests
go test ./...

# Run specific test packages
go test ./internal/config
go test ./internal/parser
go test ./internal/splitter  
go test ./internal/writer

# Run integration tests
go test -v ./integration_test.go
go test -v ./integration_golden_test.go

# Run single test
go test -run TestGroupBlocks ./internal/splitter
```

## Architecture

The codebase follows a layered architecture with clear separation of concerns:

### Core Components

1. **Parser** (`internal/parser/terraform.go`): Uses `github.com/hashicorp/hcl/v2` to parse Terraform files into structured data. Extracts all Terraform block types using HCL's BodySchema. Supports both single files and recursive directory parsing.

2. **Config** (`internal/config/config.go`): Manages YAML configuration files for custom grouping rules. Supports wildcard pattern matching, resource exclusion, and filename overrides. Automatically searches for default config files.

3. **Splitter** (`internal/splitter/resource.go`): Groups parsed blocks by type and subtype, implementing both default and configuration-driven file naming logic. Supports:
   - Default grouping by resource type
   - Custom grouping via configuration patterns
   - Resource exclusion patterns
   - Filename overrides for block types

4. **Writer** (`internal/writer/file.go`): Converts grouped blocks back to HCL format using `hclwrite`. Features sophisticated attribute handling with `hclsyntax.Body` parsing and comprehensive fallback mechanisms for complex Terraform expressions.

5. **Types** (`pkg/types/terraform.go`): Defines core data structures including Block, ParsedFile, and BlockGroup.

### CLI Interface

Built with Cobra framework (`cmd/root.go`), supporting:
- Positional argument: Input path (file or directory, required)
- `--output-dir/-o`: Output directory (default: same as input path)
- `--config/-c`: Configuration file path (optional, auto-detects default files)
- `--dry-run/-d`: Preview mode without file creation

### Configuration System

The tool automatically searches for configuration files in this order:
1. `terraform-file-organize.yaml`
2. `terraform-file-organize.yml`
3. `.terraform-file-organize.yaml`
4. `.terraform-file-organize.yml`

Configuration supports:
- **Groups**: Custom file grouping with wildcard patterns (e.g., `aws_s3_*` → `storage.tf`)
- **Overrides**: Custom filenames for block types (e.g., `variable` → `vars.tf`)
- **Exclude**: Patterns to keep as individual files

### Default Output Behavior

- Single file input: Output to same directory as input file
- Directory input: Output to input directory
- `--output-dir` specified: Override default behavior

### Key Implementation Details

**Directory Processing**: Uses `filepath.Walk` to recursively find `.tf` files, combining all blocks before splitting.

**Wildcard Matching**: Simple pattern matching supporting prefix (`aws_s3_*`), suffix (`*_bucket`), and contains patterns.

**HCL Complexity Handling**: The writer uses a comprehensive attribute schema to handle various Terraform constructs, with fallback parsing for unknown attributes.

### Testing Strategy

The project implements comprehensive testing with multiple approaches:

**Unit Tests**: All packages in `internal/` have corresponding `*_test.go` files using separate test packages (e.g., `config_test`, `parser_test`) for proper isolation.

**Integration Tests**: 
- `integration_test.go`: Binary-based CLI testing that avoids global variable issues
- `integration_golden_test.go`: Golden file testing that compares actual output against expected output files

**Test Data Structure**:
- `testdata/terraform/`: Sample Terraform files for basic testing
- `testdata/integration/case*/`: Golden file test cases with input/expected output pairs
- `testdata/configs/`: Configuration file examples
- `tmp/`: All test outputs (gitignored)

**Golden File Testing**: Uses `testdata/integration/` structure where each case has `input/` and `expected/` directories. Tests verify exact file content matches to prevent regression issues.

**Key Testing Commands**:
```bash
# Run golden file tests to check for regressions
go test -v ./integration_golden_test.go

# Test specific functionality
go test -run TestWildcardMatching ./internal/config
```