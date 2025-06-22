package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createTestDir creates a test directory within the project's tmp folder
func createTestDir(t *testing.T, testName string) string {
	tmpDir := filepath.Join("tmp", "cli-test", testName+"-"+t.Name()+"-"+time.Now().Format("20060102-150405"))
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})
	return tmpDir
}

func TestCLIBasicUsage(t *testing.T) {
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "basic")

	// バイナリをビルド
	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// テスト用ファイルを作成
	inputFile := filepath.Join(testDir, "main.tf")
	outputDir := filepath.Join(testDir, "split")

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
	cmd = exec.Command(binary, "run", inputFile, "--output-dir", outputDir)
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

func TestCLIPlan(t *testing.T) {
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "plan")

	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputFile := filepath.Join(testDir, "main.tf")
	outputDir := filepath.Join(testDir, "split")

	tfContent := `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`

	err = os.WriteFile(inputFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// planサブコマンドでCLIを実行
	cmd = exec.Command(binary, "plan", inputFile, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// "Would create file" メッセージが含まれることを確認
	if !strings.Contains(string(output), "Would create file") {
		t.Errorf("Expected plan output, got: %s", output)
	}

	// 実際のファイルは作成されていないことを確認
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("Output directory should not exist in plan mode")
	}
}

func TestCLIDirectoryInput(t *testing.T) {
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "directory")

	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputDir := filepath.Join(testDir, "terraform")
	outputDir := filepath.Join(testDir, "split")

	err = os.MkdirAll(inputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	// 複数のTerraformファイルを作成
	const (
		file1Content = `
variable "region" {
  type = string
}
`
		file2Content = `
resource "aws_instance" "web1" {
  ami = "ami-12345"
}

resource "aws_instance" "web2" {
  ami = "ami-67890"
}
`
	)

	err = os.WriteFile(filepath.Join(inputDir, "variables.tf"), []byte(file1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create variables.tf: %v", err)
	}

	err = os.WriteFile(filepath.Join(inputDir, "instances.tf"), []byte(file2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create instances.tf: %v", err)
	}

	// ディレクトリ入力でCLIを実行（非再帰的）
	cmd = exec.Command(binary, "run", inputDir, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// 複数のリソースが統合されたファイルが作成されることを確認
	resourceFile := filepath.Join(outputDir, "resource__aws_instance.tf")
	if _, statErr := os.Stat(resourceFile); os.IsNotExist(statErr) {
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

func TestCLIRecursiveDirectoryInput(t *testing.T) {
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "recursive")

	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputDir := filepath.Join(testDir, "terraform")
	subDir := filepath.Join(inputDir, "modules", "compute")

	// ディレクトリ構造を作成
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// ルートディレクトリのファイル
	const rootFileContent = `
variable "region" {
  type = string
}
`

	// サブディレクトリのファイル
	subFileContent := `
resource "aws_instance" "sub_web" {
  ami = "ami-sub123"
}
`

	err = os.WriteFile(filepath.Join(inputDir, "variables.tf"), []byte(rootFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create root variables.tf: %v", err)
	}

	err = os.WriteFile(filepath.Join(subDir, "instances.tf"), []byte(subFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create sub instances.tf: %v", err)
	}

	// 再帰的フラグなしでCLIを実行（inputDirに直接出力）
	cmd = exec.Command(binary, "plan", inputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// サブディレクトリのファイルは処理されないはず（planなので出力メッセージで確認）
	outputStr := string(output)
	if strings.Contains(outputStr, "sub_web") {
		t.Errorf("Sub-directory files should not be processed without recursive flag")
	}

	// variables.tfは処理されるはず
	if !strings.Contains(outputStr, "variables.tf") {
		t.Errorf("Variables file should be processed in root directory")
	}

	// 再帰的フラグありでCLIを実行
	cmd = exec.Command(binary, "plan", inputDir, "--recursive")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed with recursive flag: %v\nOutput: %s", err, output)
	}

	// 今度はサブディレクトリのファイルも処理されるはず
	outputStr = string(output)
	if !strings.Contains(outputStr, "resource__aws_instance.tf") {
		t.Errorf("Resource file should be created with recursive flag")
	}

	// サブディレクトリのファイルが処理されることを確認（Processedメッセージで確認）
	if !strings.Contains(outputStr, "modules/compute/instances.tf") {
		t.Errorf("Sub-directory file should be processed with recursive flag")
	}
}

func TestCLIWithConfigFile(t *testing.T) {
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "config")

	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputFile := filepath.Join(testDir, "main.tf")
	configFile := filepath.Join(testDir, "config.yaml")
	outputDir := filepath.Join(testDir, "split")

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
	cmd = exec.Command(binary, "run", inputFile, "--config", configFile, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI execution failed: %v\nOutput: %s", err, output)
	}

	// 設定に従ったファイルが作成されることを確認
	infraFile := filepath.Join(outputDir, "infrastructure.tf")
	if _, statErr := os.Stat(infraFile); os.IsNotExist(statErr) {
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
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "autoconfig")

	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputFile := filepath.Join(testDir, "main.tf")
	configFile := filepath.Join(testDir, "terraform-file-organize.yaml")
	outputDir := filepath.Join(testDir, "split")

	tfContent := `
variable "name" {
  type = string
}
`

	configContent := `
groups:
  - name: "variables"
    filename: "custom-vars.tf"
    patterns:
      - "variable"
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
	// 絶対パスを使用
	absInputFile, err := filepath.Abs(inputFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path for input file: %v", err)
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path for output dir: %v", err)
	}

	absBinary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatalf("Failed to get absolute path for binary: %v", err)
	}

	cmd = exec.Command(absBinary, "run", absInputFile, "--output-dir", absOutputDir)
	cmd.Dir = testDir // 作業ディレクトリを設定
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
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "error")

	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// 存在しないファイルを指定（プロジェクト内の存在しないパス）
	nonexistentFile := filepath.Join(testDir, "nonexistent", "file.tf")
	cmd = exec.Command(binary, "run", nonexistentFile)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for nonexistent file, got none")
	}

	if !strings.Contains(string(output), "does not exist") {
		t.Errorf("Expected 'does not exist' error message, got: %s", output)
	}

	// 引数なしで実行（runサブコマンドのみ）
	cmd = exec.Command(binary, "run")
	_, err = cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for missing arguments, got none")
	}

	// 無効な設定ファイルを指定
	inputFile := filepath.Join(testDir, "main.tf")
	invalidConfigFile := filepath.Join(testDir, "invalid.yaml")

	err = os.WriteFile(inputFile, []byte(`resource "aws_instance" "web" {}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(invalidConfigFile, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config: %v", err)
	}

	cmd = exec.Command(binary, "run", inputFile, "--config", invalidConfigFile)
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error for invalid config, got none")
	}

	if !strings.Contains(string(output), "failed to load config") {
		t.Errorf("Expected config error message, got: %s", output)
	}
}

func TestCLIIncompatibleFlags(t *testing.T) {
	// テスト用ディレクトリを作成
	testDir := createTestDir(t, "incompatible")

	binary := filepath.Join(testDir, "terraform-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputDir := filepath.Join(testDir, "terraform")
	outputDir := filepath.Join(testDir, "output")

	// テスト用ディレクトリとファイルを作成
	err = os.MkdirAll(inputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	tfContent := `
variable "test" {
  type = string
}
`
	err = os.WriteFile(filepath.Join(inputDir, "test.tf"), []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// -o と -r を同時に指定してエラーになることを確認
	cmd = exec.Command(binary, "run", inputDir, "--output-dir", outputDir, "--recursive")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected error when using -o and -r together, got none")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "cannot use --output-dir") {
		t.Errorf("Expected incompatible flags error message, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "combining multiple directories") {
		t.Errorf("Expected explanation about combining directories, got: %s", outputStr)
	}
}
