# tf-file-organize

A Go CLI tool that parses and splits Terraform files by resource type for better organization.

## Overview

Splits large Terraform files into manageable smaller files organized by resource type. Each block type is placed in separate files following specific naming conventions.

## Installation

### Using go install

```bash
go install github.com/tomoya-namekawa/tf-file-organize@latest
```

### Download Binary

Download platform-specific binaries from [GitHub Releases](https://github.com/tomoya-namekawa/tf-file-organize/releases).

### Build from Source

```bash
git clone https://github.com/tomoya-namekawa/tf-file-organize.git
cd tf-file-organize
go build -o tf-file-organize
```

## Usage

### Basic Commands

```bash
# Preview mode (no actual file creation)
tf-file-organize plan .

# Actually organize files (source files are removed)
tf-file-organize run .

# Organize with backup (source files moved to backup directory)
tf-file-organize run . --backup

# Organize entire directory
tf-file-organize run ./terraform-configs

# Specify custom output directory
tf-file-organize run . --output-dir ./organized

# Use config file for custom grouping
tf-file-organize run . --config tf-file-organize.yaml

# Validate configuration file
tf-file-organize validate-config tf-file-organize.yaml
```

### Subcommands

| Command | Description | Example |
|---------|-------------|---------|
| `run` | Actually organize and create files | `tf-file-organize run .` |
| `plan` | Preview mode (dry-run) | `tf-file-organize plan .` |
| `validate-config` | Validate configuration file | `tf-file-organize validate-config config.yaml` |
| `version` | Show version information | `tf-file-organize version` |

### Options

#### run command
- `<input-path>`: Input Terraform file or directory (required positional argument)
- `-o, --output-dir`: Output directory (default: same as input path)
- `-c, --config`: Configuration file path (default: auto-detect)
- `-r, --recursive`: Process directories recursively
- `--backup`: Move original files to backup directory

#### plan command
- Same options (except `--backup`)

## File Naming Convention

| Block Type | Naming Convention | Example |
|------------|-------------------|---------|
| resource | `resource__{resource_type}.tf` | `resource__aws_instance.tf` |
| data | `data__{data_source_type}.tf` | `data__aws_ami.tf` |
| variable | `variables.tf` | `variables.tf` |
| output | `outputs.tf` | `outputs.tf` |
| provider | `providers.tf` | `providers.tf` |
| terraform | `terraform.tf` | `terraform.tf` |
| locals | `locals.tf` | `locals.tf` |
| module | `module__{module_name}.tf` | `module__vpc.tf` |

## Configuration File

### Auto-detection

The tool automatically detects configuration files in the following order:

1. `tf-file-organize.yaml`
2. `tf-file-organize.yml`
3. `.tf-file-organize.yaml`
4. `.tf-file-organize.yml`

### Configuration Example

```yaml
# tf-file-organize.yaml
groups:
  # Group AWS network resources
  - name: "network"
    filename: "network.tf"
    patterns:
      - "aws_vpc"
      - "aws_subnet"
      - "aws_security_group*"

  # Group AWS compute resources
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
      - "aws_lb*"

  # Sub-type pattern example
  - name: "web_infrastructure"
    filename: "web-infra.tf"
    patterns:
      - "resource.aws_instance.web*"
      - "resource.aws_lb.web*"
      - "resource.aws_security_group.web"

  # Customize variables and outputs
  - name: "variables"
    filename: "vars.tf"
    patterns:
      - "variable"

  - name: "debug_outputs"
    filename: "debug-outputs.tf"
    patterns:
      - "output.debug_*"

# Exclude files by name pattern (keep as individual files)
exclude_files:
  - "*special*.tf"
  - "debug-*.tf"
```

### Pattern Matching Features

- **Simple Patterns**: `aws_s3_*` to match all S3-related resources
- **Sub-type Patterns**: `resource.aws_instance.web*` to match specific resource name patterns
- **Block Type Patterns**: `variable`, `output.debug_*` to match block types
- **Multiple Wildcards**: Multiple `*` wildcards allowed like `*special*`

## Examples

### Basic Usage

```bash
# Step 1: Preview what will happen (safe to run)
tf-file-organize plan .

# Step 2: Actually organize the files
tf-file-organize run .
```

### Before and After

**Before** - One large file (`main.tf`):
```hcl
terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-west-2"
}

variable "instance_type" {
  type    = string
  default = "t3.micro"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = var.instance_type
}

data "aws_ami" "ubuntu" {
  most_recent = true
}
```

**After** - Multiple organized files:

`terraform.tf`:
```hcl
terraform {
  required_version = ">= 1.0"
}
```

`providers.tf`:
```hcl
provider "aws" {
  region = "us-west-2"
}
```

`variables.tf`:
```hcl
variable "instance_type" {
  type    = string
  default = "t3.micro"
}
```

`resource__aws_instance.tf`:
```hcl
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = var.instance_type
}
```

`data__aws_ami.tf`:
```hcl
data "aws_ami" "ubuntu" {
  most_recent = true
}
```

## Development & Contributing

For development information, technical specifications, and contribution guidelines, see [DEVELOPMENT.md](DEVELOPMENT.md).

## License

This project is released under the MIT License.