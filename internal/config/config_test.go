package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/namekawa/terraform-file-organize/internal/config"
)

func TestLoadConfig(t *testing.T) {
	// テスト用の一時設定ファイルを作成
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	
	configContent := `
groups:
  - name: "network"
    filename: "network.tf"
    patterns:
      - "aws_vpc"
      - "aws_subnet*"
overrides:
  variable: "vars.tf"
exclude:
  - "aws_instance_special*"
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	// 設定ファイルを読み込み
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	// Groups の検証
	if len(cfg.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(cfg.Groups))
	}
	
	group := cfg.Groups[0]
	if group.Name != "network" {
		t.Errorf("Expected group name 'network', got '%s'", group.Name)
	}
	if group.Filename != "network.tf" {
		t.Errorf("Expected filename 'network.tf', got '%s'", group.Filename)
	}
	if len(group.Patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(group.Patterns))
	}
	
	// Overrides の検証
	if cfg.Overrides["variable"] != "vars.tf" {
		t.Errorf("Expected override for variable to be 'vars.tf', got '%s'", cfg.Overrides["variable"])
	}
	
	// Exclude の検証
	if len(cfg.Exclude) != 1 {
		t.Errorf("Expected 1 exclude pattern, got %d", len(cfg.Exclude))
	}
}

func TestLoadConfigEmptyPath(t *testing.T) {
	cfg, err := config.LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig with empty path should not fail: %v", err)
	}
	
	if len(cfg.Groups) != 0 {
		t.Errorf("Expected empty config, got %d groups", len(cfg.Groups))
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := config.LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestFindGroupForResource(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.GroupConfig{
			{
				Name:     "network",
				Filename: "network.tf",
				Patterns: []string{"aws_vpc", "aws_subnet*", "aws_security_group*"},
			},
			{
				Name:     "compute",
				Filename: "compute.tf",
				Patterns: []string{"aws_instance", "aws_launch_*"},
			},
		},
	}
	
	testCases := []struct {
		resourceType string
		expectedName string
	}{
		{"aws_vpc", "network"},
		{"aws_subnet", "network"},
		{"aws_subnet_public", "network"},
		{"aws_security_group", "network"},
		{"aws_security_group_web", "network"},
		{"aws_instance", "compute"},
		{"aws_launch_template", "compute"},
		{"aws_s3_bucket", ""}, // no match
	}
	
	for _, tc := range testCases {
		t.Run(tc.resourceType, func(t *testing.T) {
			group := cfg.FindGroupForResource(tc.resourceType)
			if tc.expectedName == "" {
				if group != nil {
					t.Errorf("Expected no group for %s, got %s", tc.resourceType, group.Name)
				}
			} else {
				if group == nil {
					t.Errorf("Expected group %s for %s, got nil", tc.expectedName, tc.resourceType)
				} else if group.Name != tc.expectedName {
					t.Errorf("Expected group %s for %s, got %s", tc.expectedName, tc.resourceType, group.Name)
				}
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	cfg := &config.Config{
		Exclude: []string{"aws_instance_special*", "aws_db_dev_*"},
	}
	
	testCases := []struct {
		resourceType string
		expected     bool
	}{
		{"aws_instance_special", true},
		{"aws_instance_special_web", true},
		{"aws_instance", false},
		{"aws_db_dev_mysql", true},
		{"aws_db_prod_mysql", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.resourceType, func(t *testing.T) {
			result := cfg.IsExcluded(tc.resourceType)
			if result != tc.expected {
				t.Errorf("IsExcluded(%s) = %v, expected %v", tc.resourceType, result, tc.expected)
			}
		})
	}
}

func TestGetOverrideFilename(t *testing.T) {
	cfg := &config.Config{
		Overrides: map[string]string{
			"variable": "vars.tf",
			"locals":   "common.tf",
		},
	}
	
	testCases := []struct {
		blockType string
		expected  string
	}{
		{"variable", "vars.tf"},
		{"locals", "common.tf"},
		{"output", ""}, // no override
	}
	
	for _, tc := range testCases {
		t.Run(tc.blockType, func(t *testing.T) {
			result := cfg.GetOverrideFilename(tc.blockType)
			if result != tc.expected {
				t.Errorf("GetOverrideFilename(%s) = %s, expected %s", tc.blockType, result, tc.expected)
			}
		})
	}
}

func TestPatternMatching(t *testing.T) {
	// パターンマッチングの統合テスト
	cfg := &config.Config{
		Groups: []config.GroupConfig{
			{
				Name:     "s3",
				Filename: "s3.tf",
				Patterns: []string{"aws_s3_*", "*_bucket"},
			},
		},
		Exclude: []string{"aws_instance_special*", "*_test"},
	}
	
	// グループマッチングのテスト
	testCases := []struct {
		resourceType string
		shouldMatch  bool
	}{
		{"aws_s3_bucket", true},
		{"aws_s3_object", true},
		{"storage_bucket", true},
		{"aws_instance", false},
	}
	
	for _, tc := range testCases {
		t.Run("Group_"+tc.resourceType, func(t *testing.T) {
			group := cfg.FindGroupForResource(tc.resourceType)
			matched := group != nil
			if matched != tc.shouldMatch {
				t.Errorf("Pattern matching for %s: got %v, expected %v", tc.resourceType, matched, tc.shouldMatch)
			}
		})
	}
	
	// 除外パターンのテスト
	excludeTestCases := []struct {
		resourceType string
		shouldExclude bool
	}{
		{"aws_instance_special", true},
		{"aws_instance_special_web", true},
		{"database_test", true},
		{"aws_instance", false},
		{"database_prod", false},
	}
	
	for _, tc := range excludeTestCases {
		t.Run("Exclude_"+tc.resourceType, func(t *testing.T) {
			result := cfg.IsExcluded(tc.resourceType)
			if result != tc.shouldExclude {
				t.Errorf("Exclude pattern for %s: got %v, expected %v", tc.resourceType, result, tc.shouldExclude)
			}
		})
	}
}