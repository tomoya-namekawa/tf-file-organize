package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tomoya-namekawa/tf-file-organize/internal/parser"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

const (
	resourceBlockType = "resource"
	webLabel          = "web"
)

func TestParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	tfPath := filepath.Join(tmpDir, "test.tf")

	tfContent := `
terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-west-2"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

locals {
  common_tags = {
    Environment = "test"
  }
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = var.instance_type
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  name   = "test-vpc"
}

output "instance_id" {
  value = aws_instance.web.id
}
`

	err := os.WriteFile(tfPath, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test terraform file: %v", err)
	}

	p := parser.New()
	parsedFile, err := p.ParseFile(tfPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	expectedBlocks := 8 // terraform, provider, variable, locals, resource, data, module, output
	if len(parsedFile.Blocks) != expectedBlocks {
		t.Errorf("Expected %d blocks, got %d", expectedBlocks, len(parsedFile.Blocks))
	}

	blockTypes := make(map[string]int)
	for _, block := range parsedFile.Blocks {
		blockTypes[block.Type]++
	}

	expectedTypes := map[string]int{
		"terraform": 1,
		"provider":  1,
		"variable":  1,
		"locals":    1,
		"resource":  1,
		"data":      1,
		"module":    1,
		"output":    1,
	}

	for blockType, expectedCount := range expectedTypes {
		if count, exists := blockTypes[blockType]; !exists || count != expectedCount {
			t.Errorf("Expected %d %s blocks, got %d", expectedCount, blockType, count)
		}
	}

	var resourceBlock *types.Block
	for _, block := range parsedFile.Blocks {
		if block.Type == resourceBlockType {
			resourceBlock = block
			break
		}
	}

	if resourceBlock == nil {
		t.Fatal("Resource block not found")
	}

	if len(resourceBlock.Labels) != 2 {
		t.Errorf("Expected 2 labels for resource block, got %d", len(resourceBlock.Labels))
	}

	if resourceBlock.Labels[0] != "aws_instance" {
		t.Errorf("Expected first label 'aws_instance', got '%s'", resourceBlock.Labels[0])
	}

	if resourceBlock.Labels[1] != webLabel {
		t.Errorf("Expected second label 'web', got '%s'", resourceBlock.Labels[1])
	}
}

func TestParseFileNonExistent(t *testing.T) {
	p := parser.New()
	_, err := p.ParseFile("/nonexistent/file.tf")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestParseFileInvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	tfPath := filepath.Join(tmpDir, "invalid.tf")

	invalidContent := `
resource "aws_instance" "web" {
  ami = "ami-12345"
  // Missing closing brace
`

	err := os.WriteFile(tfPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	p := parser.New()
	_, err = p.ParseFile(tfPath)
	if err == nil {
		t.Error("Expected error for invalid HCL, got nil")
	}
}

func TestParseFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	tfPath := filepath.Join(tmpDir, "empty.tf")

	err := os.WriteFile(tfPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	p := parser.New()
	parsedFile, err := p.ParseFile(tfPath)
	if err != nil {
		t.Fatalf("ParseFile failed for empty file: %v", err)
	}

	if len(parsedFile.Blocks) != 0 {
		t.Errorf("Expected 0 blocks for empty file, got %d", len(parsedFile.Blocks))
	}
}

func TestParseFileComplexResource(t *testing.T) {
	tmpDir := t.TempDir()
	tfPath := filepath.Join(tmpDir, "complex.tf")

	complexContent := `
resource "aws_security_group" "web" {
  name_prefix = "web-"
  
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
`

	err := os.WriteFile(tfPath, []byte(complexContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create complex test file: %v", err)
	}

	p := parser.New()
	parsedFile, err := p.ParseFile(tfPath)
	if err != nil {
		t.Fatalf("ParseFile failed for complex resource: %v", err)
	}

	if len(parsedFile.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(parsedFile.Blocks))
	}

	block := parsedFile.Blocks[0]
	if block.Type != "resource" {
		t.Errorf("Expected block type 'resource', got '%s'", block.Type)
	}

	if len(block.Labels) != 2 || block.Labels[0] != "aws_security_group" || block.Labels[1] != "web" {
		t.Errorf("Expected labels [aws_security_group, web], got %v", block.Labels)
	}
}
