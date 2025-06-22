package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIBasicUsage(t *testing.T) {
	// バイナリをビルド
	binary := filepath.Join(t.TempDir(), "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// テスト用ファイルを作成
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	outputDir := filepath.Join(tmpDir, "split")

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

output "instance_id" {
  value = aws_instance.web.id
}
`

	err = os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// CLIを実行
	cmd = exec.Command(binary, inputFile, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// 出力ファイルが作成されたことを確認
	expectedFiles := []string{
		"terraform.tf",
		"variables.tf",
		"resource__aws_instance.tf",
		"outputs.tf",
	}

	for _, fileName := range expectedFiles {
		filePath := filepath.Join(outputDir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", fileName)
		}
	}
}

func TestCLIDryRun(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	outputDir := filepath.Join(tmpDir, "split")

	tfContent := `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`

	err = os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// dry runでCLIを実行
	cmd = exec.Command(binary, inputFile, "--output-dir", outputDir, "--dry-run")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// "Would create file" メッセージが含まれることを確認
	if !strings.Contains(string(output), "Would create file") {
		t.Errorf("Expected dry run output, got: %s", output)
	}

	// 実際のファイルは作成されていないことを確認
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("Output directory should not exist in dry run")
	}
}

func TestCLIDirectoryInput(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "terraform")
	outputDir := filepath.Join(tmpDir, "split")

	err = os.MkdirAll(inputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	// 複数のTerraformファイルを作成
	file1Content := `
variable "region" {
  type = string
}
`

	file2Content := `
resource "aws_instance" "web1" {
  ami = "ami-12345"
}

resource "aws_instance" "web2" {
  ami = "ami-67890"
}
`

	err = os.WriteFile(filepath.Join(inputDir, "variables.tf"), []byte(file1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create variables.tf: %v", err)
	}

	err = os.WriteFile(filepath.Join(inputDir, "instances.tf"), []byte(file2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create instances.tf: %v", err)
	}

	// ディレクトリ入力でCLIを実行
	cmd = exec.Command(binary, inputDir, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// 複数のリソースが統合されたファイルが作成されることを確認
	resourceFile := filepath.Join(outputDir, "resource__aws_instance.tf")
	if _, err := os.Stat(resourceFile); os.IsNotExist(err) {
		t.Errorf("Expected consolidated resource file was not created")
	}

	// ファイル内容確認（2つのaws_instanceリソースが含まれているはず）
	content, err := os.ReadFile(resourceFile)
	if err != nil {
		t.Fatalf("Failed to read resource file: %v", err)
	}

	contentStr := string(content)
	web1Count := strings.Count(contentStr, `"web1"`)
	web2Count := strings.Count(contentStr, `"web2"`)

	if web1Count != 1 || web2Count != 1 {
		t.Errorf("Expected both web1 and web2 resources in output, got web1:%d web2:%d", web1Count, web2Count)
	}
}

func TestCLIWithConfigFile(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	configFile := filepath.Join(tmpDir, "config.yaml")
	outputDir := filepath.Join(tmpDir, "split")

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

	err = os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// 設定ファイルを指定してCLIを実行
	cmd = exec.Command(binary, inputFile, "--config", configFile, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// 設定に従ったファイルが作成されることを確認
	infraFile := filepath.Join(outputDir, "infrastructure.tf")
	if _, err := os.Stat(infraFile); os.IsNotExist(err) {
		t.Errorf("Expected infrastructure.tf was not created")
	}

	// ファイル内容確認（3つのリソースがすべて含まれているはず）
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

func TestCLIAutoConfigDetection(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	configFile := filepath.Join(tmpDir, "terraform-file-organize.yaml")
	outputDir := filepath.Join(tmpDir, "split")

	tfContent := `
variable "name" {
  type = string
}
`

	configContent := `
overrides:
  variable: "custom-vars.tf"
`

	err = os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// 設定ファイルを明示的に指定せずにCLIを実行（自動検出をテスト）
	cmd = exec.Command(binary, inputFile, "--output-dir", outputDir)
	cmd.Dir = tmpDir // 作業ディレクトリを設定
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// 設定ファイルが自動検出されたことを確認
	if !strings.Contains(string(output), "Loading configuration from") {
		t.Errorf("Expected config auto-detection message, got: %s", output)
	}

	// カスタムファイル名が使用されたことを確認
	customVarsFile := filepath.Join(outputDir, "custom-vars.tf")
	if _, err := os.Stat(customVarsFile); os.IsNotExist(err) {
		t.Errorf("Expected custom-vars.tf was not created")
	}
}

func TestCLIErrorHandling(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// 存在しないファイルを指定
	cmd = exec.Command(binary, "/nonexistent/file.tf")
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

	// 無効な設定ファイルを指定
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "main.tf")
	invalidConfigFile := filepath.Join(tmpDir, "invalid.yaml")

	err = os.WriteFile(inputFile, []byte(`resource "aws_instance" "web" {}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(invalidConfigFile, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config: %v", err)
	}

	cmd = exec.Command(binary, inputFile, "--config", invalidConfigFile)
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for invalid config, got none")
	}

	if !strings.Contains(string(output), "failed to load config") {
		t.Errorf("Expected config error message, got: %s", output)
	}
}
