package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/usecase"
)

// TestGoldenFiles は Golden File Testing を実行
// 実際の出力と期待される出力を比較してファイル分割の正確性を検証
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			caseDir := filepath.Join("testdata", "integration", tc.name)
			inputDir := filepath.Join(caseDir, "input")
			expectedDir := filepath.Join(caseDir, "expected")
			actualDir := filepath.Join("tmp", "integration-test", tc.name)

			// 出力ディレクトリを作成
			if err := os.MkdirAll(actualDir, 0755); err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}

			// ファイル分割の実行（usecaseを使用）
			uc := usecase.NewOrganizeFilesUsecase()

			// 設定ファイルのパスを決定
			configPath := filepath.Join(caseDir, "terraform-file-organize.yaml")
			var configFilePath string
			if _, statErr := os.Stat(configPath); statErr == nil {
				configFilePath = configPath
			}

			req := &usecase.OrganizeFilesRequest{
				InputPath:  inputDir,
				OutputDir:  actualDir,
				ConfigFile: configFilePath,
				DryRun:     false,
			}

			_, err := uc.Execute(req)
			if err != nil {
				t.Fatalf("Failed to process directory: %v", err)
			}

			// 期待される出力と実際の出力を比較
			if err := compareDirectories(t, expectedDir, actualDir); err != nil {
				t.Errorf("Golden file test failed for %s: %v", tc.name, err)
			}
		})
	}
}

// compareDirectories は2つのディレクトリの内容を比較
func compareDirectories(t *testing.T, expectedDir, actualDir string) error {
	expectedFiles, err := getFileList(expectedDir)
	if err != nil {
		return fmt.Errorf("failed to list expected files: %w", err)
	}

	actualFiles, err := getFileList(actualDir)
	if err != nil {
		return fmt.Errorf("failed to list actual files: %w", err)
	}

	// ファイル数とファイル名の比較
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

	// 各ファイルの内容を比較
	for _, filename := range expectedFiles {
		expectedPath := filepath.Join(expectedDir, filename)
		actualPath := filepath.Join(actualDir, filename)

		if err := compareFiles(t, expectedPath, actualPath); err != nil {
			return fmt.Errorf("file %s: %w", filename, err)
		}
	}

	return nil
}

// getFileList はディレクトリ内の.tfファイルリストを取得
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

// compareFiles は2つのファイルの内容を比較
func compareFiles(t *testing.T, expectedPath, actualPath string) error {
	expectedContent, err := os.ReadFile(expectedPath)
	if err != nil {
		return fmt.Errorf("failed to read expected file: %w", err)
	}

	actualContent, err := os.ReadFile(actualPath)
	if err != nil {
		return fmt.Errorf("failed to read actual file: %w", err)
	}

	// 正規化：改行コードを統一し、末尾の空白を削除
	expectedStr := strings.TrimSpace(normalizeContent(string(expectedContent)))
	actualStr := strings.TrimSpace(normalizeContent(string(actualContent)))

	if expectedStr != actualStr {
		return fmt.Errorf("content mismatch:\n--- Expected ---\n%s\n--- Actual ---\n%s\n--- End ---",
			expectedStr, actualStr)
	}

	return nil
}

// normalizeContent はファイル内容を正規化
func normalizeContent(content string) string {
	// 改行コードを統一
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// 連続する空行を単一にする
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

// Benchmark tests for performance regression detection
func BenchmarkFileProcessing(b *testing.B) {
	inputDir := "testdata/integration/case1/input"
	outputDir := filepath.Join("tmp", "benchmark-test")

	uc := usecase.NewOrganizeFilesUsecase()

	for i := 0; i < b.N; i++ {
		// 出力ディレクトリをクリア
		os.RemoveAll(outputDir)
		os.MkdirAll(outputDir, 0755)

		req := &usecase.OrganizeFilesRequest{
			InputPath:  inputDir,
			OutputDir:  outputDir,
			ConfigFile: "", // デフォルト設定を使用
			DryRun:     false,
		}

		_, err := uc.Execute(req)
		if err != nil {
			b.Fatalf("Failed to process directory: %v", err)
		}
	}
}
