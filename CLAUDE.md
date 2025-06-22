# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

terraform-file-organize is a Go CLI tool that parses Terraform files and splits them into separate files organized by resource type. The tool uses HashiCorp's HCL parser to analyze Terraform configurations and reorganizes them according to specific naming conventions. It supports both single file and directory input, with optional YAML configuration for custom grouping rules.

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
go build -o terraform-file-organize

# Basic usage examples
./terraform-file-organize main.tf --dry-run
./terraform-file-organize . --output-dir tmp/test --dry-run
./terraform-file-organize testdata/terraform --config testdata/configs/terraform-file-organize.yaml
```

### Essential Development Commands
```bash
# Code quality checks
go mod tidy
golangci-lint run
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Critical regression tests
go test -run TestGoldenFiles -v
go build && ./terraform-file-organize testdata/terraform/sample.tf --dry-run

# Workflow validation
actionlint
```

## Architecture

The codebase follows clean architecture principles with strict layered separation:

### Core Components

1. **Usecase Layer** (`internal/usecase/organize.go`): Business logic orchestrator that coordinates all operations. Implements security validation, configuration loading, and error handling.

2. **Parser** (`internal/parser/terraform.go`): Uses `github.com/hashicorp/hcl/v2` to parse Terraform files into structured data. **Key feature**: Extracts raw block content with comments preserved using dual parsing (standard HCL + syntax trees). Supports both single files and recursive directory parsing.

3. **Config** (`internal/config/config.go`): Manages YAML configuration files for custom grouping rules. Supports wildcard pattern matching, resource exclusion, and filename overrides.

4. **Splitter** (`internal/splitter/resource.go`): Groups parsed blocks by type and subtype. **Critical**: Implements deterministic sorting for stable output.

5. **Writer** (`internal/writer/file.go`): Converts grouped blocks back to HCL format. **Key features**: Comment preservation via raw source code reconstruction, optional descriptive comment generation, and deterministic attribute ordering.

6. **Types** (`pkg/types/terraform.go`): Defines core data structures including Block (with RawBody for comment preservation), ParsedFile, and BlockGroup.

7. **Version** (`internal/version/version.go`): Manages version information with fallback support for different build methods. Uses `runtime/debug.BuildInfo` for `go install` compatibility while maintaining GoReleaser ldflags injection priority.

### CLI Interface

Built with Cobra framework (`cmd/root.go`). The CLI layer is thin - it only handles argument parsing and delegates business logic to the usecase layer.

**Arguments:**
- Positional argument: Input path (file or directory, required)
- `--output-dir/-o`: Output directory (default: same as input path)
- `--config/-c`: Configuration file path (optional, auto-detects default files)
- `--dry-run/-d`: Preview mode without file creation
- `--add-comments`: Add descriptive comments to terraform blocks (disabled by default)

### Configuration System

Auto-searches for configuration files in this order:
1. `terraform-file-organize.yaml`
2. `terraform-file-organize.yml`
3. `.terraform-file-organize.yaml`
4. `.terraform-file-organize.yml`

**Configuration features:**
- **Groups**: Custom file grouping with wildcard patterns (e.g., `aws_s3_*` → `storage.tf`)
- **Overrides**: Custom filenames for block types (e.g., `variable` → `vars.tf`)
- **Exclude**: Patterns to keep as individual files

## Testing Strategy

### Test Structure
- **Unit Tests**: All `internal/` packages have `*_test.go` files using separate test packages for isolation
- **Integration Tests**: Binary-based CLI testing and golden file testing
- **Golden File Tests**: **Critical** - Compares actual output against expected files in `testdata/integration/`

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

# Package-specific tests
go test ./internal/config -v
go test ./internal/parser -v
go test ./internal/splitter -v
go test ./internal/writer -v
go test ./internal/usecase -v

# Coverage target: >60%
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Development Principles

### Core Requirements
- **HCL Processing**: Always use `github.com/hashicorp/hcl` for Terraform-related processing
- **Comment Preservation**: Block-level comments are ALWAYS preserved via RawBody extraction (no configuration needed)
- **Deterministic Output**: Sort all collections (resources, attributes, filenames) alphabetically
- **Security First**: Use `filepath.Clean` and `filepath.Base` to prevent path traversal attacks
- **Golden File Tests**: Mandatory for any output changes

### Code Standards
- Use constants for repeated strings (enforced by goconst linter)
- Terraform block types defined as constants in `internal/splitter/resource.go`
- Modern Go patterns: `slices.Contains()` instead of loops
- Named return values for complex functions

### Critical Files for Output Changes
When modifying output behavior, update these files:
- `internal/splitter/resource.go` - Resource sorting and grouping
- `internal/writer/file.go` - Attribute ordering and HCL formatting
- `testdata/integration/case*/expected/` - Golden file test expectations

## CI/CD Configuration

### GitHub Actions Pipelines

**Main CI Pipeline** (`.github/workflows/ci.yml`):
- **test**: Comprehensive test suite with race detection and coverage
- **lint**: golangci-lint with security-focused rules (gosec, govet, gocritic, etc.)
- **build**: Binary build and sample input verification

**Workflow Lint Pipeline** (`.github/workflows/workflow-lint.yml`):
- **workflow-lint**: actionlint for GitHub Actions
- **pinact-check**: Verifies all actions are pinned to commit hashes
- Triggered by changes to `.github/**` or `.mise.toml`

### Security Features
- All GitHub Actions pinned to commit SHA hashes
- gosec security scanning integrated into CI
- renovatebot for dependency management
- Path traversal protection in usecase layer

### Tool Versions (mise-managed)
```toml
[tools]
go = "latest"
golangci-lint = "v2.1.6"
actionlint = "latest"
pinact = "latest"
"npm:@goreleaser/goreleaser" = "latest"
```

## Release Process

### Automated Release with Release Please

The project uses Release Please for automated version management and GoReleaser for building:

1. **Conventional Commits**: Use conventional commit format for automatic versioning
   ```bash
   git commit -m "feat: add new feature"
   git commit -m "fix: fix critical bug"
   git commit -m "docs: update documentation"
   ```

2. **Automatic Process**: 
   - Release Please creates PR with version bump and changelog
   - Merging the PR triggers automatic release and GoReleaser build
   - Binaries are published to GitHub releases

3. **Configuration Files**:
   - `.release-please-manifest.json`: Version tracking
   - `release-please-config.json`: Release Please configuration
   - `.goreleaser.yaml`: GoReleaser build configuration

### Manual Release Testing

```bash
# Test release build locally
make release-snapshot

# Check GoReleaser configuration
make release-check
```

### Installation Methods

After release, users can install via:

```bash
# Go install (latest) - may require GOPRIVATE setting
GOPRIVATE=github.com/tomoya-namekawa/terraform-file-organize go install github.com/tomoya-namekawa/terraform-file-organize@latest

# Go install (specific version)
GOPRIVATE=github.com/tomoya-namekawa/terraform-file-organize go install github.com/tomoya-namekawa/terraform-file-organize@v0.1.1

# Download binary directly from GitHub releases
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
go build -o terraform-file-organize
./terraform-file-organize testdata/terraform/sample.tf --dry-run

# 5. Workflow validation
actionlint
```

### Key Implementation Details

**Comment Preservation Architecture**: Uses dual parsing approach - standard HCL parsing for structure analysis plus `hclsyntax` parsing for raw content extraction. The `RawBody` field in Block preserves original source including all comments, formatting, and attribute order.

**Deterministic Output**: Critical for CI/CD and version control. Both resource ordering (via `sortBlocksInGroup`) and original content preservation (via RawBody) ensure consistent output.

**Testing Architecture**: Uses separate test packages to avoid import cycles. Golden file tests provide regression protection with exact content matching including preserved comments.

**HCL Processing**: The writer prioritizes RawBody content when available (comment preservation), falling back to structured HCL processing for edge cases.

**Version Detection System**: Multi-tier version detection system that works across different build environments:
1. GoReleaser builds use ldflags-injected version information
2. `go install @version` builds use module version from BuildInfo
3. Development builds use VCS revision with dirty state detection
This ensures consistent version reporting regardless of installation method.

## Comment Preservation System

### How It Works
The tool ALWAYS preserves block-internal comments by default. This is achieved through:

1. **Dual Parsing**: Both standard HCL and syntax tree parsing to capture structure + raw content
2. **RawBody Extraction**: Using `OpenBraceRange` and `CloseBraceRange` to extract exact source between `{` and `}`
3. **Raw Block Reconstruction**: Writer uses `appendRawBlock()` to output original content with comments intact

### What Is Preserved
- **All block-internal comments**: `# comment` within blocks
- **Original formatting**: Spacing, indentation, and attribute order
- **Complex expressions**: Template strings, nested objects, and function calls
- **Multi-line structures**: Objects, arrays, and nested blocks

### What Is NOT Preserved
- **File-level comments**: Comments outside of blocks (by design - file organization changes structure)
- **Block header comments**: Comments above block declarations (optional via `--add-comments`)

### Example
```hcl
# Input
resource "aws_instance" "web" {
  # AMI configuration
  ami = "ami-12345"
  
  # Network settings  
  subnet_id = var.subnet_id
}

# Output (comments preserved exactly)
resource "aws_instance" "web" {
  # AMI configuration
  ami = "ami-12345"
  
  # Network settings  
  subnet_id = var.subnet_id
}
```

## Key Differences from Other Documentation

- **README.md**: User-focused documentation in Japanese with installation instructions and basic usage examples
- **DEVELOPMENT.md**: Comprehensive developer documentation in Japanese including detailed development workflow with Conventional Commits and release-please process
- **CLAUDE.md**: Technical reference for Claude Code with architecture details and critical implementation information

## Best Practices

### File Management
- テスト結果などの一時的なファイルはtmp dirに作って

### Test Case Details

**case5**: Specifically tests the fallback processing when RawBody is unavailable. This case validates:
- Complex template expressions like `"${var.subdomain}.${var.domain_name}"`
- Nested block preservation (`metadata`, `spec`) without RawBody
- Syntax body handling for unknown block types

### Golden File Test Updates
When modifying output behavior, golden files must be updated manually:
```bash
# Run tests to see differences
go test -run TestGoldenFiles -v

# Update golden files with new expected output
cp tmp/integration-test/case*/* testdata/integration/case*/expected/
```