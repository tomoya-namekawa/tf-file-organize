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
go test ./...
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

4. **Writer** (`internal/writer/file.go`): Converts grouped blocks back to HCL format using `hclwrite`. Uses PartialContent with comprehensive attribute schemas to handle complex Terraform constructs.

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

### Testing

- `testdata/terraform/sample.tf`: Basic Terraform constructs
- `testdata/terraform/extended-sample.tf`: Complex multi-resource example
- `testdata/configs/terraform-file-organize.yaml`: Sample configuration with AWS resource grouping
- All test outputs go to `tmp/` directory (gitignored)