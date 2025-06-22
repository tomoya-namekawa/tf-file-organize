package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// CLIテストはバイナリを使って実行する
func buildTestBinary(t *testing.T) string {
	binary := filepath.Join(t.TempDir(), "terraform-file-organize-test")
	cmd := exec.Command("go", "build", "-o", binary, "../")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	return binary
}

func TestCLISingleFile(t *testing.T) {
	binary := buildTestBinary(t)

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	outputDir := filepath.Join(tmpDir, "output")

	tfContent := `
terraform {
  required_version = ">= 1.0"
}

variable "instance_type" {
  type    = string
  default = "t3.micro"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = var.instance_type
}
`

	err := os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(binary, inputFile, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	expectedFiles := []string{
		"terraform.tf",
		"variables.tf",
		"resource__aws_instance.tf",
	}

	for _, fileName := range expectedFiles {
		filePath := filepath.Join(outputDir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", fileName)
		}
	}
}

func TestCLIDirectory(t *testing.T) {
	binary := buildTestBinary(t)

	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "terraform")
	outputDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(inputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	file1Content := `
variable "region" {
  type = string
}

provider "aws" {
  region = var.region
}
`

	file2Content := `
resource "aws_instance" "web" {
  ami = "ami-12345"
}

resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"
}
`

	err = os.WriteFile(filepath.Join(inputDir, "vars.tf"), []byte(file1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create vars.tf: %v", err)
	}

	err = os.WriteFile(filepath.Join(inputDir, "resources.tf"), []byte(file2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create resources.tf: %v", err)
	}

	cmd := exec.Command(binary, inputDir, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	expectedFiles := []string{
		"variables.tf",
		"providers.tf",
		"resource__aws_instance.tf",
		"resource__aws_s3_bucket.tf",
	}

	for _, fileName := range expectedFiles {
		filePath := filepath.Join(outputDir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", fileName)
		}
	}
}

func TestCLIDryRun(t *testing.T) {
	binary := buildTestBinary(t)

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	outputDir := filepath.Join(tmpDir, "output")

	tfContent := `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`

	err := os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(binary, inputFile, "--output-dir", outputDir, "--dry-run")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Would create file") {
		t.Errorf("Expected dry run output, got: %s", output)
	}

	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("Output directory should not exist in dry run")
	}
}

func TestCLIWithConfig(t *testing.T) {
	binary := buildTestBinary(t)

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	configFile := filepath.Join(tmpDir, "config.yaml")
	outputDir := filepath.Join(tmpDir, "output")

	tfContent := `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id = aws_vpc.main.id
}

resource "aws_instance" "web" {
  ami = "ami-12345"
}
`

	configContent := `
groups:
  - name: "infrastructure"
    filename: "infrastructure.tf"
    patterns:
      - "aws_vpc"
      - "aws_subnet*"
      - "aws_instance"
`

	err := os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	cmd := exec.Command(binary, inputFile, "--config", configFile, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	infraFile := filepath.Join(outputDir, "infrastructure.tf")
	if _, statErr := os.Stat(infraFile); os.IsNotExist(statErr) {
		t.Errorf("Expected infrastructure.tf was not created")
	}

	content, err := os.ReadFile(infraFile)
	if err != nil {
		t.Fatalf("Failed to read infrastructure file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"aws_vpc"`) {
		t.Errorf("infrastructure.tf does not contain aws_vpc")
	}
	if !strings.Contains(contentStr, `"aws_subnet"`) {
		t.Errorf("infrastructure.tf does not contain aws_subnet")
	}
	if !strings.Contains(contentStr, `"aws_instance"`) {
		t.Errorf("infrastructure.tf does not contain aws_instance")
	}
}

func TestCLIErrorHandling(t *testing.T) {
	binary := buildTestBinary(t)

	// 存在しないファイルを指定
	cmd := exec.Command(binary, "/nonexistent/file.tf")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for nonexistent file, got none")
	}

	if !strings.Contains(string(output), "does not exist") {
		t.Errorf("Expected 'does not exist' error message, got: %s", output)
	}

	// 引数なしで実行
	cmd = exec.Command(binary)
	_, err = cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for missing arguments, got none")
	}
}
