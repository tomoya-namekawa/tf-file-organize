# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

terraform-file-organize is a Go CLI tool that parses Terraform files and splits them into separate files organized by resource type. The tool uses HashiCorp's HCL parser to analyze Terraform configurations and reorganizes them according to specific naming conventions. It supports both single file and directory input, with optional YAML configuration for custom grouping rules.

## Build and Development Commands

This project uses [mise](https://mise.jdx.dev/) for tool management. All development tools are defined in `.mise.toml`.

```bash
# Install mise and project tools
mise install

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
go test ./internal/usecase

# Run integration tests
go test -v ./integration_test.go
go test -v ./integration_golden_test.go

# Run single test
go test -run TestGroupBlocks ./internal/splitter

# Run golden file tests (critical for regression detection)
go test -run TestGoldenFiles -v

# Linting (install locally)
golangci-lint run

# GitHub Actions linting
actionlint

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Architecture

The codebase follows clean architecture principles with strict layered separation:

### Core Components

1. **Usecase Layer** (`internal/usecase/organize.go`): Business logic orchestrator that coordinates all operations. Implements security validation, configuration loading, and error handling. This layer isolates business rules from CLI concerns and provides a clean interface for testing.

2. **Parser** (`internal/parser/terraform.go`): Uses `github.com/hashicorp/hcl/v2` to parse Terraform files into structured data. Extracts all Terraform block types using HCL's BodySchema. Supports both single files and recursive directory parsing.

3. **Config** (`internal/config/config.go`): Manages YAML configuration files for custom grouping rules. Supports wildcard pattern matching, resource exclusion, and filename overrides. Automatically searches for default config files.

4. **Splitter** (`internal/splitter/resource.go`): Groups parsed blocks by type and subtype, implementing both default and configuration-driven file naming logic. **Critical**: Implements deterministic sorting for stable output - resources are sorted alphabetically within groups and groups are sorted by filename.

5. **Writer** (`internal/writer/file.go`): Converts grouped blocks back to HCL format using `hclwrite`. **Key feature**: Implements deterministic attribute ordering by sorting attributes alphabetically before writing. Uses `hclwrite.Format` for consistent formatting without external terraform CLI dependency.

6. **Types** (`pkg/types/terraform.go`): Defines core data structures including Block, ParsedFile, and BlockGroup.

### CLI Interface

Built with Cobra framework (`cmd/root.go`). **Important**: The CLI layer has been refactored to be thin - it only handles argument parsing and delegates all business logic to the usecase layer. This enables better testing and separation of concerns.

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

### Key Implementation Details

**Deterministic Output**: Critical for CI/CD and version control. Both resource ordering (via `sortBlocksInGroup`) and attribute ordering (via alphabetical sorting in `copyBlockBodyGeneric`) are deterministic.

**Security Measures**: Path traversal protection implemented in the usecase layer with comprehensive validation. File paths are sanitized using `filepath.Clean` and `filepath.Base`.

**HCL Processing**: The writer handles complex Terraform expressions through multiple fallback mechanisms. Uses `hclsyntax.Body` for direct access to parsed syntax trees when standard evaluation fails.

**Testing Architecture**: Uses separate test packages (e.g., `config_test`) to avoid import cycles and ensure proper isolation. Golden file tests in `testdata/integration/` provide regression protection.

### Testing Strategy

The project implements comprehensive testing with multiple approaches:

**Unit Tests**: All packages in `internal/` have corresponding `*_test.go` files using separate test packages (e.g., `config_test`, `parser_test`, `usecase_test`) for proper isolation and to avoid import cycles.

**Integration Tests**: 
- `integration_test.go`: Binary-based CLI testing that avoids global variable issues
- `integration_golden_test.go`: **Critical** - Golden file testing that compares actual output against expected output files. These tests ensure deterministic output and catch regressions.

**Test Data Structure**:
- `testdata/terraform/`: Sample Terraform files for basic testing
- `testdata/integration/case*/`: Golden file test cases with input/expected output pairs
  - `case1/`: Basic Terraform blocks with default configuration
  - `case2/`: Multiple files with same resource types (basic grouping)  
  - `case3/`: Custom grouping rules with configuration file
- `testdata/configs/`: Configuration file examples
- `tmp/`: All test outputs (gitignored)

**Golden File Testing**: **Must be maintained** - Uses `testdata/integration/` structure where each case has `input/` and `expected/` directories. Tests verify exact file content matches including attribute ordering and formatting. When making changes that affect output, update expected files by running the tool and copying results.

**Test Execution Notes**:
- Golden file tests are enabled and critical for preventing regressions
- Tests assume deterministic output (attributes and resources sorted alphabetically)
- Failed golden file tests indicate either bugs or that expected outputs need updating

**Key Testing Commands**:
```bash
# Run golden file tests to check for regressions (most important)
go test -run TestGoldenFiles -v

# Test specific functionality
go test -run TestWildcardMatching ./internal/config

# Test usecase layer (business logic)
go test ./internal/usecase -v

# Update golden files when output format changes (manual process)
./terraform-file-organize testdata/integration/case1/input -o testdata/integration/case1/expected
```

## Development Principles

**HCL Processing**: Always use https://github.com/hashicorp/hcl for Terraform-related processing. This ensures compatibility and leverages HashiCorp's official parsing capabilities.

**Deterministic Output**: Any change to output generation must maintain deterministic behavior. Sort all collections (resources, attributes, filenames) alphabetically to ensure consistent results across runs.

**Security First**: All file path operations must use `filepath.Clean` and `filepath.Base` to prevent path traversal attacks. Validate all user inputs in the usecase layer.

**Testing Requirements**: 
- Golden file tests are mandatory for any output changes
- Use separate test packages to avoid import cycles
- All new functionality requires corresponding unit tests

## Critical Files for Output Changes

When modifying output behavior, these files are critical:
- `internal/splitter/resource.go` - Handles resource sorting and grouping
- `internal/writer/file.go` - Controls attribute ordering and HCL formatting
- `testdata/integration/case*/expected/` - Golden file test expectations

## CI/CD Configuration

The project uses GitHub Actions for continuous integration with comprehensive testing and security checks:

**CI Pipeline** (`.github/workflows/ci.yml`):
- **pinact-check**: Verifies all GitHub Actions are pinned to commit hashes for security
- **test**: Runs comprehensive test suite with race detection and coverage reporting
- **lint**: Executes golangci-lint via dedicated GitHub Action with multiple linters enabled (includes gosec)
- **build**: Builds binary and verifies it works with sample inputs

**Workflow Lint Pipeline** (`.github/workflows/workflow-lint.yml`):
Runs only when GitHub Actions workflows or mise configuration changes:
- **workflow-lint**: Lints GitHub Actions workflows using actionlint
- Triggered by changes to `.github/**` or `.mise.toml` files

**Linting Configuration** (`.golangci.yml`):
- Comprehensive linter configuration with security-focused rules
- Excludes test files and testdata from certain checks
- Timeout configured for 5 minutes to handle large codebases

**Security Features**:
- All GitHub Actions pinned to commit SHA hashes
- Automated pinact verification to prevent action tampering
- gosec security scanning integrated into CI
- renovatebot (not dependabot) used for dependency management