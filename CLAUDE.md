# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tf-file-organize is a Go CLI tool that parses Terraform files and splits them into separate files organized by resource type. The tool uses HashiCorp's HCL parser to analyze Terraform configurations and reorganizes them according to specific naming conventions. It supports both single file and directory input, with optional YAML configuration for custom grouping rules.

## Quick Start

### Tool Management
This project uses [mise](https://mise.jdx.dev/) for unified tool management. All development tools are defined in `.mise.toml`.

```bash
# Install all development tools
mise install

# Check tool versions
mise list
```

### Build and Basic Usage
```bash
# Build the project
go build -o tf-file-organize

# Basic usage examples
./tf-file-organize plan main.tf
./tf-file-organize run . --output-dir tmp/test
./tf-file-organize run testdata/terraform --config testdata/configs/terraform-file-organize.yaml
./tf-file-organize validate-config testdata/configs/terraform-file-organize.yaml
```

### Essential Development Commands
```bash
# Code quality checks (run these before commits)
go mod tidy
golangci-lint run
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Critical regression tests (MUST pass before any output changes)
go test -run TestGoldenFiles -v
go build && ./tf-file-organize plan testdata/terraform/sample.tf

# Single test execution examples
go test -run TestSpecificFunction -v ./internal/packagename
go test ./internal/config -v
go test ./internal/parser -v

# Workflow validation
actionlint

# Alternative build using Makefile
make dev-build
make test-coverage
make check  # runs both lint and test
```

## Architecture

The codebase follows clean architecture principles with strict layered separation:

### Core Components

1. **Usecase Layer** (`internal/usecase/usecase.go`): Business logic orchestrator with recent refactoring for improved maintainability. The main `Execute` function is now split into focused sub-functions:
   - `prepareExecution()`: Input validation and setup 
   - `processBlocks()`: File parsing and block grouping
   - `handleOutput()`: File writing operations
   - `handleSourceFileCleanup()`: Backup and source file management
   - `displayResults()`: User feedback and result formatting

2. **Validation Layer** (`internal/validation/validation.go`): **New package** created to eliminate code duplication across CLI commands. Provides centralized path validation, security checks, and flag combination validation used by all cmd/ files.

3. **Parser** (`internal/parser/parser.go`): Uses `github.com/hashicorp/hcl/v2` to parse Terraform files into structured data. **Key feature**: Extracts raw block content with comments preserved using dual parsing (standard HCL + syntax trees).

4. **Config** (`internal/config/config.go`): Manages YAML configuration files for custom grouping rules. Supports complex pattern matching including sub-type patterns and file exclusion.

5. **Splitter** (`internal/splitter/splitter.go`): Groups parsed blocks by type and subtype with deterministic sorting for stable output. Recent refactoring split `sanitizeFileName` into focused functions:
   - `cleanUnsafeCharacters()`: Security-focused character cleaning
   - `applyLengthLimits()`: File length restrictions
   - `validateReservedNames()`: Windows reserved name handling

6. **Writer** (`internal/writer/writer.go`): Converts grouped blocks back to HCL format. Prioritizes RawBody content for comment preservation, falling back to structured HCL processing.

7. **Types** (`pkg/types/types.go`): Defines core data structures including Block (with RawBody for comment preservation), ParsedFile, and BlockGroup.

### CLI Interface

Built with Cobra framework using subcommand architecture (`cmd/`). Recent refactoring eliminated code duplication:

- **Common Logic** (`cmd/common.go`): Shared `executeOrganizeFiles()` function used by both run and plan commands
- **Command Separation**: Each subcommand has focused responsibility without duplicated validation logic

**Subcommands:**
- `run <input-path>`: Execute file organization with actual file creation/modification
  - `--output-dir/-o`: Output directory (default: same as input path)
  - `--config/-c`: Configuration file path (optional, auto-detects default files)
  - `--recursive/-r`: Process directories recursively
  - `--backup`: Backup original files to 'backup' subdirectory before organizing
- `plan <input-path>`: Preview mode showing what would be done without file creation
  - Same options as run (except `--backup` not applicable)
- `validate-config <config-file>`: Validate configuration file syntax and content
- `version`: Show version information

### Configuration System

Auto-searches for configuration files in this order:
1. `tf-file-organize.yaml`
2. `tf-file-organize.yml`
3. `.tf-file-organize.yaml`
4. `.tf-file-organize.yml`
5. `terraform-file-organize.yaml`
6. `terraform-file-organize.yml`

**Configuration features:**
- **Groups**: Custom file grouping with complex pattern matching:
  - Simple patterns: `aws_s3_*` → `storage.tf`
  - Sub-type patterns: `resource.aws_instance.web*` → `web-infrastructure.tf`
  - Block-type patterns: `variable`, `output.debug_*` → custom files
- **ExcludeFiles**: File name patterns to exclude from grouping (e.g., `*special*.tf`, `debug-*.tf`)

## Testing Strategy

### Test Structure
- **Unit Tests**: All `internal/` packages have `*_test.go` files using separate test packages (e.g., `main_test`) for isolation
- **Business Logic Tests** (`internal/usecase/business_test.go`): Business logic testing with mock implementations
- **Mock Helpers** (`internal/usecase/mocks_test.go`): Mock implementations and test utilities for usecase testing
- **CLI Tests** (`cli_test.go`): Command-line interface functionality testing via binary execution
- **Golden File Tests** (`golden_test.go`): **Critical** - End-to-end testing comparing actual output against expected files
- **Idempotency Tests** (`idempotency_test.go`): Ensures consistent results across multiple runs

### Test Data Organization
```
testdata/
├── terraform/           # Sample Terraform files for basic testing
├── integration/         # Golden file test cases
│   ├── case1/          # Basic blocks (default config)
│   ├── case2/          # Multiple files (basic grouping)
│   ├── case3/          # Custom grouping rules
│   ├── case4/          # Complex multi-cloud setup (25 blocks)
│   └── case5/          # Template expressions with nested blocks (fallback test)
├── configs/            # Configuration file examples
└── tmp/               # Test outputs (gitignored)
```

### Key Test Commands
```bash
# Regression detection (most important)
go test -run TestGoldenFiles -v

# CLI interface testing (includes all subcommands)
go test -run TestCLI -v

# Package-specific tests
go test ./internal/config -v
go test ./internal/parser -v
go test ./internal/splitter -v
go test ./internal/writer -v
go test ./internal/usecase -v
go test ./internal/validation -v  # New validation package

# Single test execution
go test -run TestSpecificFunction -v ./internal/packagename

# Coverage target: >60%
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Development Principles

### Core Requirements
- **HCL Processing**: Always use `github.com/hashicorp/hcl` for Terraform-related processing
- **Comment Preservation**: Block-level comments are ALWAYS preserved via RawBody extraction
- **Deterministic Output**: Sort all collections (resources, attributes, filenames) alphabetically
- **Security First**: Use `filepath.Clean` and `filepath.Base` to prevent path traversal attacks
- **Golden File Tests**: Mandatory for any output changes
- **Idempotency**: Multiple runs produce consistent results by default source file removal

### Code Standards
- Use constants for repeated strings (enforced by goconst linter)
- Terraform block types defined as constants in `internal/splitter/splitter.go` and `internal/usecase/usecase.go`
- Modern Go patterns: `slices.Contains()` instead of loops
- Named return values for complex functions
- **Function Length**: Keep functions under 50 lines; split complex functions into focused sub-functions
- **Single Responsibility**: Each function should have one clear purpose (recently enforced through refactoring)
- **Error Handling**: All defer statements in tests must handle errors appropriately (`_ = os.Remove()` pattern)
- **Test Packages**: Use separate test packages (e.g., `package main_test`) to avoid import cycles and ensure proper isolation

### Critical Files for Output Changes
When modifying output behavior, update these files:
- `internal/splitter/splitter.go` - Resource sorting, grouping, and complex pattern matching
- `internal/writer/writer.go` - Attribute ordering and HCL formatting
- `internal/config/config.go` - Configuration structure and validation
- `internal/validation/validation.go` - Path validation and security checks
- `testdata/integration/case*/expected/` - Golden file test expectations
- `cmd/*.go` - CLI subcommand definitions and flag handling

### Maintenance and Refactoring Guidelines
When improving code maintainability:
1. **Eliminate Duplication**: Use `internal/validation` package for shared validation logic
2. **Split Large Functions**: Break functions >50 lines into focused sub-functions
3. **Security First**: All file operations must use validation package functions
4. **Test Coverage**: Maintain >60% coverage, especially for new validation package (currently 80.8%)

## Key Implementation Details

### Comment Preservation Architecture
Uses dual parsing approach - standard HCL parsing for structure analysis plus `hclsyntax` parsing for raw content extraction. The `RawBody` field in Block preserves original source including all comments, formatting, and attribute order.

### Idempotency and File Management
The tool ensures idempotent operations by:
1. **Default Behavior**: Source files are removed after successful organization to prevent duplication
2. **Backup Option**: `--backup` flag moves source files to 'backup' subdirectory instead of deletion
3. **Smart Conflict Resolution**: Config-aware file removal prevents conflicts when grouped files would duplicate source content
4. **Source File Tracking**: Maintains list of original files for proper cleanup

### Pattern Matching System
Supports complex pattern matching including:
- Wildcard patterns: `aws_s3_*`, `*special*`
- Sub-type patterns: `resource.aws_instance.web*`
- Block-type patterns: `variable`, `output.debug_*`
- Multiple `*` wildcards in single pattern

### Golden File Test Updates
When modifying output behavior, golden files must be updated manually:
```bash
# Run tests to see differences
go test -run TestGoldenFiles -v

# Update golden files with new expected output (after verifying changes are correct)
for case in testdata/integration/case*/; do
  cp -r tmp/integration-test/$(basename "$case")/* "$case/expected/"
done

# Alternative: Update specific case
cp -r tmp/integration-test/case1/* testdata/integration/case1/expected/
```

## Development Workflow

### Pre-commit Checklist
```bash
# 1. Format and lint
golangci-lint run

# 2. Run tests with coverage
go test -v -coverprofile=coverage.out ./...

# 3. Verify golden files
go test -run TestGoldenFiles -v

# 4. Build and integration test
go build -o tf-file-organize
./tf-file-organize plan testdata/terraform/sample.tf

# 5. Workflow validation
actionlint
```

## CI/CD Configuration

### GitHub Actions Pipelines
- **Main CI Pipeline** (`.github/workflows/ci.yml`): test, lint, build with security scanning
- **Workflow Lint Pipeline** (`.github/workflows/workflow-lint.yml`): actionlint and pinact checks

### Security Features
- All GitHub Actions pinned to commit SHA hashes
- gosec security scanning integrated into CI
- Path traversal protection in usecase layer

### Tool Versions (mise-managed)
```toml
[tools]
go = "latest"
golangci-lint = "v2.1.6"
actionlint = "latest"
pinact = "latest"
"npm:@goreleaser/goreleaser" = "latest"
terraform = "latest"

[env]
GOPROXY = "https://proxy.golang.org,direct"
GOSUMDB = "sum.golang.org"
```

## Release Process

### Automated Release with Release Please
The project uses Release Please for automated version management and GoReleaser for building:

1. **Conventional Commits**: Use conventional commit format for automatic versioning
2. **Automatic Process**: Release Please creates PR with version bump, merging triggers release
3. **Configuration Files**: `.release-please-manifest.json`, `release-please-config.json`, `.goreleaser.yaml`

### Manual Release Testing
```bash
# Test release build locally
make release-snapshot

# Check GoReleaser configuration
make release-check
```

## Key Development Workflows

### When Making Output Changes
1. **Understand Impact**: Changes to parser, splitter, or writer affect golden files
2. **Run Golden Tests**: `go test -run TestGoldenFiles -v` to see current state
3. **Make Changes**: Implement your modifications
4. **Update Golden Files**: `cp tmp/integration-test/case*/* testdata/integration/case*/expected/`
5. **Verify**: Re-run golden tests to ensure they pass

### When Adding New Features
1. **Add Tests First**: Write unit tests and update golden files if needed
2. **Implement Feature**: Follow clean architecture principles
3. **Update Documentation**: Update README.md and DEVELOPMENT.md as needed
4. **Run Full Test Suite**: Ensure all tests pass, especially golden file tests

### When Debugging Test Failures
```bash
# For golden file test failures
go test -run TestGoldenFiles -v
# Check tmp/integration-test/ for actual output vs testdata/integration/*/expected/

# For specific package issues
go test ./internal/config -v -run TestSpecificFunction

# For CLI issues
go test -run TestCLI -v
```

## Development Guidelines

- **Documentation Guidelines**
  - All documentation and comments should be written in English
- **Comment Cleanup Policy**
  - Remove redundant comments that repeat the code
  - Keep important business logic and security-related comments
  - Maintain clean, readable code without excessive commenting
