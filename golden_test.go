package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGoldenFiles(t *testing.T) {
	testCases := []struct {
		name        string
		description string
	}{
		{
			name:        "case1",
			description: "Single file with basic Terraform blocks (default config)",
		},
		{
			name:        "case2",
			description: "Multiple files with same resource types (basic grouping)",
		},
		{
			name:        "case3",
			description: "Configuration file with custom grouping rules",
		},
		{
			name:        "case4",
			description: "Complex nested blocks and template expressions",
		},
		{
			name:        "case5",
			description: "Template expressions with nested blocks (fallback processing test)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			caseDir := filepath.Join("testdata", "integration", tc.name)
			inputDir := filepath.Join(caseDir, "input")
			expectedDir := filepath.Join(caseDir, "expected")
			actualDir := filepath.Join("tmp", "integration-test", tc.name)

			if err := os.RemoveAll(actualDir); err != nil {
				t.Fatalf("Failed to remove existing output directory: %v", err)
			}
			if err := os.MkdirAll(actualDir, 0755); err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}

			binary := filepath.Join(t.TempDir(), "tf-file-organize")
			buildCmd := exec.Command("go", "build", "-o", binary)
			err := buildCmd.Run()
			if err != nil {
				t.Fatalf("Failed to build binary: %v", err)
			}

			configPath := filepath.Join(caseDir, "tf-file-organize.yaml")
			var args []string
			args = append(args, "run", inputDir, "--output-dir", actualDir)
			if _, statErr := os.Stat(configPath); statErr == nil {
				args = append(args, "--config", configPath)
			}

			cmd := exec.Command(binary, args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Failed to process directory: %v\nOutput: %s", err, output)
			}

			if err := compareDirectories(t, expectedDir, actualDir); err != nil {
				t.Errorf("Golden file test failed for %s: %v", tc.name, err)
			}
		})
	}
}

func compareDirectories(t *testing.T, expectedDir, actualDir string) error {
	expectedFiles, err := getFileList(expectedDir)
	if err != nil {
		return fmt.Errorf("failed to list expected files: %w", err)
	}

	actualFiles, err := getFileList(actualDir)
	if err != nil {
		return fmt.Errorf("failed to list actual files: %w", err)
	}

	if len(expectedFiles) != len(actualFiles) {
		return fmt.Errorf("file count mismatch: expected %d, got %d\nExpected: %v\nActual: %v",
			len(expectedFiles), len(actualFiles), expectedFiles, actualFiles)
	}

	for i, expectedFile := range expectedFiles {
		if actualFiles[i] != expectedFile {
			return fmt.Errorf("file name mismatch at index %d: expected %s, got %s",
				i, expectedFile, actualFiles[i])
		}
	}

	for _, filename := range expectedFiles {
		expectedPath := filepath.Join(expectedDir, filename)
		actualPath := filepath.Join(actualDir, filename)

		if err := compareFiles(t, expectedPath, actualPath); err != nil {
			return fmt.Errorf("file %s: %w", filename, err)
		}
	}

	return nil
}

func getFileList(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)
	return files, nil
}

func compareFiles(_ *testing.T, expectedPath, actualPath string) error {
	expectedContent, err := os.ReadFile(expectedPath)
	if err != nil {
		return fmt.Errorf("failed to read expected file: %w", err)
	}

	actualContent, err := os.ReadFile(actualPath)
	if err != nil {
		return fmt.Errorf("failed to read actual file: %w", err)
	}

	expectedStr := strings.TrimSpace(normalizeContent(string(expectedContent)))
	actualStr := strings.TrimSpace(normalizeContent(string(actualContent)))

	if expectedStr != actualStr {
		return fmt.Errorf("content mismatch:\n--- Expected ---\n%s\n--- Actual ---\n%s\n--- End ---",
			expectedStr, actualStr)
	}

	return nil
}

func normalizeContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	var normalizedLines []string
	var lastLineEmpty bool

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !lastLineEmpty {
				normalizedLines = append(normalizedLines, "")
			}
			lastLineEmpty = true
		} else {
			normalizedLines = append(normalizedLines, line)
			lastLineEmpty = false
		}
	}

	return strings.Join(normalizedLines, "\n")
}

func BenchmarkFileProcessing(b *testing.B) {
	inputDir := "testdata/integration/case1/input"
	outputDir := filepath.Join("tmp", "benchmark-test")

	binary := filepath.Join(b.TempDir(), "tf-file-organize")
	buildCmd := exec.Command("go", "build", "-o", binary)
	err := buildCmd.Run()
	if err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}

	for b.Loop() {
		_ = os.RemoveAll(outputDir)
		_ = os.MkdirAll(outputDir, 0755)

		cmd := exec.Command(binary, "run", inputDir, "--output-dir", outputDir)
		_, err := cmd.CombinedOutput()
		if err != nil {
			b.Fatalf("Failed to process directory: %v", err)
		}
	}
}
