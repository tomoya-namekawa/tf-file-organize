package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
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
exclude_files:
  - "*special*.tf"
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

	// ExcludeFiles の検証
	if len(cfg.ExcludeFiles) != 1 {
		t.Errorf("Expected 1 exclude file pattern, got %d", len(cfg.ExcludeFiles))
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

func TestIsFileExcluded(t *testing.T) {
	cfg := &config.Config{
		ExcludeFiles: []string{"*special*.tf", "debug-*.tf"},
	}

	testCases := []struct {
		filename string
		expected bool
	}{
		{"resource-special.tf", true},
		{"special-config.tf", true},
		{"debug-output.tf", true},
		{"outputs.tf", false},
		{"main.tf", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := cfg.IsFileExcluded(tc.filename)
			if result != tc.expected {
				t.Errorf("IsFileExcluded(%s) = %v, expected %v", tc.filename, result, tc.expected)
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
		ExcludeFiles: []string{"*special*.tf", "*test*.tf"},
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
}

func TestValidateConfigFields(t *testing.T) {
	tests := []struct {
		name          string
		configYAML    string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid configuration",
			configYAML: `
groups:
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
exclude_files:
  - "*.backup"
`,
			expectError: false,
		},
		{
			name: "deprecated exclude field",
			configYAML: `
groups:
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
exclude:
  - "*.backup"
`,
			expectError:   true,
			errorContains: "deprecated fields found: 'exclude' (use 'exclude_files' instead)",
		},
		{
			name: "unknown top-level field",
			configYAML: `
groups:
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
unknown_field: "value"
`,
			expectError:   true,
			errorContains: "unknown fields found: 'unknown_field'",
		},
		{
			name: "invalid group field",
			configYAML: `
groups:
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
    priority: "high"
exclude_files:
  - "*.backup"
`,
			expectError:   true,
			errorContains: "unknown fields found: 'priority' in group 1",
		},
		{
			name: "multiple invalid fields",
			configYAML: `
groups:
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
    invalid1: "value"
    invalid2: "value"
exclude:
  - "*.backup"
unknown_top: "value"
`,
			expectError:   true,
			errorContains: "deprecated fields found:",
		},
		{
			name: "deprecated overrides field",
			configYAML: `
groups:
  - name: "compute"
    filename: "compute.tf"
    patterns:
      - "aws_instance"
overrides:
  variable: "vars.tf"
`,
			expectError:   true,
			errorContains: "deprecated fields found: 'overrides' (no longer supported)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test-config.yaml")

			err := os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			_, err = config.LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func containsString(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
