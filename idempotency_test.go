package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdempotency(t *testing.T) {
	testDir := createTestDir(t, "idempotency")

	binary := filepath.Join(testDir, "tf-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputFile := filepath.Join(testDir, "main.tf")
	const tfContent = `
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

	cmd = exec.Command(binary, "run", testDir)
	output1, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("First execution failed: %v\nOutput: %s", err, output1)
	}

	files1, err := getCreatedFiles(testDir)
	if err != nil {
		t.Fatalf("Failed to get files after first run: %v", err)
	}

	contents1, err := getFileContents(files1)
	if err != nil {
		t.Fatalf("Failed to read file contents after first run: %v", err)
	}

	cmd = exec.Command(binary, "run", testDir)
	output2, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Second execution failed: %v\nOutput: %s", err, output2)
	}

	files2, err := getCreatedFiles(testDir)
	if err != nil {
		t.Fatalf("Failed to get files after second run: %v", err)
	}

	contents2, err := getFileContents(files2)
	if err != nil {
		t.Fatalf("Failed to read file contents after second run: %v", err)
	}

	cmd = exec.Command(binary, "run", testDir)
	output3, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Third execution failed: %v\nOutput: %s", err, output3)
	}

	files3, err := getCreatedFiles(testDir)
	if err != nil {
		t.Fatalf("Failed to get files after third run: %v", err)
	}

	contents3, err := getFileContents(files3)
	if err != nil {
		t.Fatalf("Failed to read file contents after third run: %v", err)
	}

	if len(files1) != len(files2) || len(files2) != len(files3) {
		t.Errorf("File count differs between runs: %d, %d, %d", len(files1), len(files2), len(files3))
	}

	if !compareFileLists(files1, files2) || !compareFileLists(files2, files3) {
		t.Errorf("File lists differ between runs")
		t.Logf("Run 1 files: %v", files1)
		t.Logf("Run 2 files: %v", files2)
		t.Logf("Run 3 files: %v", files3)
	}

	if !compareFileContents(contents1, contents2) || !compareFileContents(contents2, contents3) {
		t.Errorf("File contents differ between runs")
		logContentDifferences(t, contents1, contents2, contents3)
	}
}

func TestIdempotencyWithConfig(t *testing.T) {
	testDir := createTestDir(t, "idempotency-config")

	binary := filepath.Join(testDir, "tf-file-organize")
	cmd := exec.Command("go", "build", "-o", binary)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	inputFile := filepath.Join(testDir, "main.tf")
	const tfConfigContent = `
terraform {
  required_version = ">= 1.0"
}

variable "instance_type" {
  type    = string
  default = "t3.micro"
}

resource "google_cloud_run_service" "app" {
  name     = "app"
  location = "us-central1"
}

data "google_iam_policy" "noauth" {
  binding {
    role = "roles/run.invoker"
    members = ["allUsers"]
  }
}

output "service_url" {
  value = google_cloud_run_service.app.status[0].url
}
`

	err = os.WriteFile(inputFile, []byte(tfConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	configFile := filepath.Join(testDir, "tf-file-organize.yaml")
	configContent := `
groups:
  - name: "data_blocks"
    filename: "data.tf"
    patterns:
      - "google_iam_*"
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "google_cloud_run_*"

exclude_files:
  - "*special*.tf"
`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	var allFiles [][]string
	var allContents []map[string]string

	for i := range 3 {
		cmd = exec.Command(binary, "run", testDir, "--config", configFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Execution %d failed: %v\nOutput: %s", i+1, err, output)
		}

		files, err := getCreatedFiles(testDir)
		if err != nil {
			t.Fatalf("Failed to get files after run %d: %v", i+1, err)
		}

		contents, err := getFileContents(files)
		if err != nil {
			t.Fatalf("Failed to read file contents after run %d: %v", i+1, err)
		}

		allFiles = append(allFiles, files)
		allContents = append(allContents, contents)
	}

	for i := 1; i < len(allFiles); i++ {
		if !compareFileLists(allFiles[0], allFiles[i]) {
			t.Errorf("File lists differ between run 1 and run %d", i+1)
			t.Logf("Run 1 files: %v", allFiles[0])
			t.Logf("Run %d files: %v", i+1, allFiles[i])
		}

		if !compareFileContents(allContents[0], allContents[i]) {
			t.Errorf("File contents differ between run 1 and run %d", i+1)
			logContentDifferences(t, allContents[0], allContents[i], nil)
		}
	}
}

func getCreatedFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") &&
			entry.Name() != "tf-file-organize.yaml" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func getFileContents(files []string) (map[string]string, error) {
	contents := make(map[string]string)

	for _, filename := range files {
		content, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		contents[filepath.Base(filename)] = string(content)
	}

	return contents, nil
}

func compareFileLists(files1, files2 []string) bool {
	if len(files1) != len(files2) {
		return false
	}

	sorted1 := make([]string, len(files1))
	sorted2 := make([]string, len(files2))
	copy(sorted1, files1)
	copy(sorted2, files2)

	for i := range len(sorted1) {
		for j := i + 1; j < len(sorted1); j++ {
			if sorted1[i] > sorted1[j] {
				sorted1[i], sorted1[j] = sorted1[j], sorted1[i]
			}
		}
	}
	for i := range len(sorted2) {
		for j := i + 1; j < len(sorted2); j++ {
			if sorted2[i] > sorted2[j] {
				sorted2[i], sorted2[j] = sorted2[j], sorted2[i]
			}
		}
	}

	for i := range sorted1 {
		if sorted1[i] != sorted2[i] {
			return false
		}
	}

	return true
}

func compareFileContents(contents1, contents2 map[string]string) bool {
	if len(contents1) != len(contents2) {
		return false
	}

	for filename, content1 := range contents1 {
		content2, exists := contents2[filename]
		if !exists {
			return false
		}

		normalized1 := strings.TrimSpace(content1)
		normalized2 := strings.TrimSpace(content2)

		if normalized1 != normalized2 {
			return false
		}
	}

	return true
}

func logContentDifferences(t *testing.T, contents1, contents2, contents3 map[string]string) {
	for filename := range contents1 {
		content1 := contents1[filename]
		content2 := contents2[filename]

		if content1 != content2 {
			t.Logf("Content difference in %s:", filename)
			t.Logf("Run 1 length: %d", len(content1))
			t.Logf("Run 2 length: %d", len(content2))

			if len(content1) > 200 {
				t.Logf("Run 1 preview: %s...", content1[:200])
			} else {
				t.Logf("Run 1 content: %s", content1)
			}

			if len(content2) > 200 {
				t.Logf("Run 2 preview: %s...", content2[:200])
			} else {
				t.Logf("Run 2 content: %s", content2)
			}
		}

		if contents3 != nil {
			content3 := contents3[filename]
			if content1 != content3 {
				t.Logf("Content difference in %s (run 1 vs 3):", filename)
				t.Logf("Run 1 length: %d", len(content1))
				t.Logf("Run 3 length: %d", len(content3))
			}
		}
	}
}
