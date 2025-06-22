# Development Guide

Detailed development guide for tf-file-organize.

## Development Environment Setup

### Tool Management

This project uses [mise](https://mise.jdx.dev/) for unified tool management.

```bash
# Install mise (first time only)
curl https://mise.run | sh

# Install project dependencies
mise install

# Check tool versions
mise list
```

### Initial Setup

```bash
# Clone repository
git clone https://github.com/tomoya-namekawa/tf-file-organize.git
cd tf-file-organize

# Install dependencies
go mod tidy

# Build
go build -o tf-file-organize
```

## Development Commands

### Build and Test

```bash
# Project build
go build -o tf-file-organize

# Run all tests
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Important: Golden file tests (regression detection)
go test -run TestGoldenFiles -v

# CLI functionality tests (all subcommands)
go test -run TestCLI -v

# Package-specific tests
go test ./internal/config -v
go test ./internal/parser -v
go test ./internal/splitter -v
go test ./internal/writer -v
go test ./internal/usecase -v

# Coverage goal: 60% or higher
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Code Quality Checks

```bash
# Linting
golangci-lint run

# Format
go mod tidy

# Workflow validation
actionlint
```

### Development Testing

```bash
# Preview mode
./tf-file-organize plan testdata/terraform/sample.tf

# Directory processing
./tf-file-organize plan testdata/terraform

# Test with config file
./tf-file-organize plan testdata/terraform --config testdata/configs/tf-file-organize.yaml

# Configuration file validation
./tf-file-organize validate-config testdata/configs/tf-file-organize.yaml
```

## Architecture

### Code Structure

This tool is designed following clean architecture principles with the following layers:

- **CLI Layer** (`cmd/`): Subcommand definitions and argument parsing
- **Usecase Layer** (`internal/usecase/`): Business logic orchestration and security validation
- **Domain Layer** (`internal/`): Core functionality (parser, splitter, writer, config)
- **Data Layer** (`pkg/types/`): Data structure definitions

### Subcommand Structure

- `run <input-path>`: Execute file organization
- `plan <input-path>`: Preview mode (formerly --dry-run)
- `validate-config <config-file>`: Configuration file validation
- `version`: Show version information

## Important Development Principles

### 1. Maintain Idempotency

The tool must guarantee consistent results across multiple runs:

- **Default Behavior**: Remove source files to prevent duplication
- **Backup Option**: `--backup` moves source files to 'backup' directory
- **Smart Conflict Resolution**: File removal logic considering configuration rules

### 2. Maintain Deterministic Output

Output must always be deterministic for CI/CD and version control compatibility:

- **Resource Ordering**: Sort alphabetically within groups
- **Attribute Ordering**: Sort HCL attributes alphabetically
- **Filename Ordering**: Sort output filenames alphabetically

### 3. Comment Preservation

Block-level comments must always be preserved:

- **Dual Parsing**: Standard HCL + `hclsyntax` parsing
- **RawBody Extraction**: Extract original source including comments
- **Raw Block Reconstruction**: Output with original content

### 4. Security First

- All file path operations use `filepath.Clean` and `filepath.Base`
- Input validation implemented in usecase layer
- Thorough path traversal attack prevention

### 5. Pattern Matching

Support complex pattern matching system:

- **Simple Patterns**: `aws_s3_*`
- **Sub-type Patterns**: `resource.aws_instance.web*`
- **Block Type Patterns**: `variable`, `output.debug_*`
- **Multiple Wildcards**: `*special*`

## Testing Strategy

### Test Structure

- **Unit Tests**: All `internal/` packages covered, using separate test packages
- **CLI Tests** (`cli_test.go`): Functionality testing via binary execution
- **Golden File Tests** (`golden_test.go`): **Most Important** - Regression detection

### Test Data Structure

```
testdata/
├── terraform/          # Basic sample files
├── configs/            # Configuration file examples
└── integration/        # Golden file test cases
    ├── case1/          # Basic blocks (default config)
    ├── case2/          # Multiple files basic grouping
    ├── case3/          # Custom grouping with config file
    ├── case4/          # Complex multi-cloud setup (25 blocks)
    └── case5/          # Template expressions and nested blocks
```

### Golden File Tests

**Most important tests**. Expected value files must be updated when output changes:

```bash
# Run golden file tests
go test -run TestGoldenFiles -v

# Update expected files (when output format changes)
cp tmp/integration-test/case*/* testdata/integration/case*/expected/
```

## Development Workflow

### Pre-commit Checklist

```bash
# 1. Format and lint
golangci-lint run

# 2. Test with coverage
go test -v -coverprofile=coverage.out ./...

# 3. Golden file verification
go test -run TestGoldenFiles -v

# 4. Build and integration test
go build -o tf-file-organize
./tf-file-organize plan testdata/terraform/sample.tf

# 5. Workflow validation
actionlint
```

### Output Change Considerations

When changing output format:

1. **Update golden file expected values**
2. **Verify deterministic output is maintained**
3. **Verify comment preservation functionality works correctly**

## CI/CD Configuration

### GitHub Actions

- **Main CI Pipeline** (`.github/workflows/ci.yml`): test, lint, build with security scanning
- **Workflow Lint Pipeline** (`.github/workflows/workflow-lint.yml`): actionlint and pinact checks

### Security Features

- GitHub Actions commit hash pinning
- gosec security scanning integration
- Path traversal attack protection

### Tool Versions (mise managed)

```toml
[tools]
go = "latest"
golangci-lint = "v2.1.6"
actionlint = "latest"
pinact = "latest"
"npm:@goreleaser/goreleaser" = "latest"
```

## Release Process

### Automated Release with Conventional Commits

This project adopts automatic releases using Conventional Commits and release-please.

#### Commit Message Format

```bash
# Add new feature
git commit -m "feat: add plan subcommand for preview mode"

# Bug fix
git commit -m "fix: resolve pattern matching for complex wildcards"

# Performance improvement
git commit -m "perf: optimize HCL parsing performance"

# Refactoring
git commit -m "refactor: simplify resource grouping logic"

# Documentation update
git commit -m "docs: update README for subcommand structure"

# Test addition/modification
git commit -m "test: add golden file tests for idempotency"

# CI/CD changes
git commit -m "ci: update GitHub Actions workflow"

# Build system changes
git commit -m "build: update go.mod dependencies"

# Other changes
git commit -m "chore: update development documentation"
```

#### Version Impact

- `feat:` → **minor** version bump (0.1.0 → 0.2.0)
- `fix:` → **patch** version bump (0.1.0 → 0.1.1)
- `BREAKING CHANGE:` footer → **major** version bump (0.1.0 → 1.0.0)
- Others → **patch** version bump

#### Automated Release Process

1. **Commit**: Commit to main branch with Conventional Commits
2. **PR Creation**: release-please automatically creates version bump PR
3. **Release**: Merging PR triggers automatic GoReleaser execution
4. **Artifacts**: Binaries and changelog published to GitHub Releases

### Configuration Files

- `.release-please-manifest.json`: Current version management
- `release-please-config.json`: Release configuration
- `.goreleaser.yaml`: Binary build configuration

### Manual Release Testing

```bash
# Test release build locally
make release-snapshot

# Check GoReleaser configuration
make release-check
```

## Common Issues and Solutions

### 1. Golden File Test Failures

When output format changes, expected value files need updating:

```bash
# Update expected values with new output
go test -run TestGoldenFiles -v
cp tmp/integration-test/case*/* testdata/integration/case*/expected/
```

### 2. Import Cycles

Create test files as separate packages (e.g., `package config_test`).

### 3. Non-deterministic Output

Sort collections (slices, maps) before output.

### 4. Pattern Matching Debugging

```bash
# Test specific pattern matching
go test ./internal/splitter -run TestGroupBlocksWithConfig -v

# Configuration file validation
./tf-file-organize validate-config testdata/configs/tf-file-organize.yaml
```

## Contribution Guidelines

### Pre-Pull Request Checklist

1. **Run and pass all tests**
2. **Run golden file tests**
3. **Run golangci-lint**
4. **Commit with Conventional Commits**
5. **Run actionlint (when modifying workflows)**

### Code Quality Standards

- **Test Coverage**: Maintain 60% or higher
- **Linting**: Pass golangci-lint checks
- **Golden Files**: Update expected files when output changes
- **Security**: Pass gosec security checks

Following this guide ensures quality and security while extending functionality.