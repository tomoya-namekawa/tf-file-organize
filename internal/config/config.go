// Package config provides configuration management for terraform-file-organize,
// including custom grouping rules, overrides, and exclusion patterns.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

// Config represents the main configuration structure for file organization rules.
type Config struct {
	Groups       []GroupConfig `yaml:"groups"`        // カスタムグループ化ルール
	ExcludeFiles []string      `yaml:"exclude_files"` // 除外ファイルパターン
}

// GroupConfig defines a custom grouping rule for specific resource patterns.
type GroupConfig struct {
	Name     string   `yaml:"name"`     // グループ名
	Filename string   `yaml:"filename"` // 出力ファイル名
	Patterns []string `yaml:"patterns"` // マッチするパターンのリスト
}

// LoadConfig loads and validates a configuration file from the specified path.
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return &Config{}, nil
	}

	if !filepath.IsAbs(configPath) {
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		configPath = abs
	}

	// セキュリティチェック: ファイル情報を検証
	stat, err := os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}

	// ファイルサイズ制限（DoS攻撃対策）
	const maxConfigSize = 1024 * 1024 // 1MB
	if stat.Size() > maxConfigSize {
		return nil, fmt.Errorf("config file too large (max %d bytes): %d bytes", maxConfigSize, stat.Size())
	}

	// 通常のファイルかチェック
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("config path must be a regular file: %s", configPath)
	}

	data, err := os.ReadFile(configPath) //nolint:gosec // configPath is validated for safety
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 無効なフィールドの検出
	if err := validateConfigFields(data); err != nil {
		return nil, fmt.Errorf("invalid configuration fields: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 設定内容の検証
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// ValidateConfig performs comprehensive validation of a configuration
func ValidateConfig(cfg *Config) error {
	// Check for duplicate group names
	groupNames := make(map[string]bool)
	for i, group := range cfg.Groups {
		if groupNames[group.Name] {
			return fmt.Errorf("duplicate group name '%s' at group %d", group.Name, i+1)
		}
		groupNames[group.Name] = true
	}

	// Check for duplicate filenames
	filenames := make(map[string]string)
	for i, group := range cfg.Groups {
		if existingGroup, exists := filenames[group.Filename]; exists {
			return fmt.Errorf("duplicate filename '%s' in group '%s' (group %d) - already used by group '%s'", group.Filename, group.Name, i+1, existingGroup)
		}
		filenames[group.Filename] = group.Name
	}

	// Check for pattern conflicts (same pattern in multiple groups)
	patternGroups := make(map[string]string)
	for _, group := range cfg.Groups {
		for _, pattern := range group.Patterns {
			if existingGroup, exists := patternGroups[pattern]; exists {
				return fmt.Errorf("pattern '%s' appears in multiple groups: '%s' and '%s'", pattern, existingGroup, group.Name)
			}
			patternGroups[pattern] = group.Name
		}
	}

	// Validate exclude file patterns
	for i, pattern := range cfg.ExcludeFiles {
		if pattern == "" {
			return fmt.Errorf("exclude file pattern %d is empty", i+1)
		}
		// Test pattern validity by attempting to match against a test string
		if !isValidPattern(pattern) {
			return fmt.Errorf("exclude file pattern %d ('%s') contains invalid characters", i+1, pattern)
		}
	}

	return nil
}

// isValidPattern checks if a pattern is valid for file matching
func isValidPattern(pattern string) bool {
	// Check for basic invalid characters that could cause issues
	invalidChars := []string{"\x00", "\n", "\r", "\t"}
	for _, char := range invalidChars {
		if strings.Contains(pattern, char) {
			return false
		}
	}
	return true
}

// validateConfig validates the loaded configuration for security and correctness
func validateConfig(config *Config) error {
	if err := validateGroups(config.Groups); err != nil {
		return err
	}
	if err := validateExcludeFilePatterns(config.ExcludeFiles); err != nil {
		return err
	}
	return nil
}

// validateGroups validates group configurations
func validateGroups(groups []GroupConfig) error {
	for i, group := range groups {
		if group.Name == "" {
			return fmt.Errorf("group %d: name cannot be empty", i)
		}

		if group.Filename == "" {
			return fmt.Errorf("group %d (%s): filename cannot be empty", i, group.Name)
		}

		if err := validateFilename(group.Filename); err != nil {
			return fmt.Errorf("group %d (%s): invalid filename: %w", i, group.Name, err)
		}

		if len(group.Patterns) == 0 {
			return fmt.Errorf("group %d (%s): at least one pattern is required", i, group.Name)
		}

		if err := validatePatterns(group.Patterns, i, group.Name); err != nil {
			return err
		}
	}
	return nil
}

// validatePatterns validates pattern configurations
func validatePatterns(patterns []string, groupIndex int, groupName string) error {
	for j, pattern := range patterns {
		if pattern == "" {
			return fmt.Errorf("group %d (%s), pattern %d: pattern cannot be empty", groupIndex, groupName, j)
		}
		if len(pattern) > 100 {
			return fmt.Errorf("group %d (%s), pattern %d: pattern too long (max 100 chars)", groupIndex, groupName, j)
		}
	}
	return nil
}

// validateExcludeFilePatterns validates exclude file pattern configurations
func validateExcludeFilePatterns(patterns []string) error {
	for i, pattern := range patterns {
		if pattern == "" {
			return fmt.Errorf("exclude file pattern %d: pattern cannot be empty", i)
		}
		if len(pattern) > 100 {
			return fmt.Errorf("exclude file pattern %d: pattern too long (max 100 chars)", i)
		}
	}
	return nil
}

// validateFilename ensures filename is safe and doesn't contain dangerous characters
func validateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// 基本的な文字チェック
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename cannot contain '..'")
	}

	if strings.ContainsAny(filename, "/\\:*?\"<>|") {
		return fmt.Errorf("filename contains invalid characters")
	}

	// 長さ制限
	if len(filename) > 255 {
		return fmt.Errorf("filename too long (max 255 chars)")
	}

	// システムファイル名のチェック
	systemNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4",
		"COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2",
		"LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

	nameWithoutExt := strings.ToUpper(filename)
	if idx := strings.LastIndex(nameWithoutExt, "."); idx != -1 {
		nameWithoutExt = nameWithoutExt[:idx]
	}

	if slices.Contains(systemNames, nameWithoutExt) {
		return fmt.Errorf("filename cannot be a system reserved name: %s", filename)
	}

	return nil
}

func (c *Config) FindGroupForResource(resourceType string) *GroupConfig {
	for _, group := range c.Groups {
		for _, pattern := range group.Patterns {
			if c.matchPattern(pattern, resourceType) {
				return &group
			}
		}
	}
	return nil
}

func (c *Config) IsFileExcluded(filename string) bool {
	for _, pattern := range c.ExcludeFiles {
		if c.matchPattern(pattern, filename) {
			return true
		}
	}
	return false
}

func (c *Config) matchPattern(pattern, text string) bool {
	if strings.Contains(pattern, "*") {
		return c.wildcardMatch(pattern, text)
	}

	return pattern == text
}

func (c *Config) wildcardMatch(pattern, text string) bool {
	if pattern == "*" {
		return true
	}

	// filepath.Matchと同様のロジックを使用
	return c.matchWithWildcards(pattern, text)
}

// matchWithWildcards は複数の*を含むパターンを処理
func (c *Config) matchWithWildcards(pattern, text string) bool {
	patternIndex := 0
	textIndex := 0
	starIdx := -1
	match := 0

	for textIndex < len(text) {
		if patternIndex < len(pattern) && (pattern[patternIndex] == text[textIndex] || pattern[patternIndex] == '?') {
			patternIndex++
			textIndex++
		} else if patternIndex < len(pattern) && pattern[patternIndex] == '*' {
			starIdx = patternIndex
			match = textIndex
			patternIndex++
		} else if starIdx != -1 {
			patternIndex = starIdx + 1
			match++
			textIndex = match
		} else {
			return false
		}
	}

	// パターンの残りの*を処理
	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}

	return patternIndex == len(pattern)
}

// validateConfigFields は設定ファイル内の無効なフィールドを検出
func validateConfigFields(data []byte) error {
	// 生のYAMLを map[string]any にパース
	var rawConfig map[string]any
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// 有効なトップレベルフィールドを定義
	validTopLevelFields := map[string]bool{
		"groups":        true,
		"exclude_files": true,
	}

	// 古い形式や無効なフィールドを検出
	var invalidFields []string
	var deprecatedFields []string

	for field := range rawConfig {
		if !validTopLevelFields[field] {
			// 既知の古いフィールドかチェック
			switch field {
			case "exclude":
				deprecatedFields = append(deprecatedFields, fmt.Sprintf("'%s' (use 'exclude_files' instead)", field))
			case "overrides":
				deprecatedFields = append(deprecatedFields, fmt.Sprintf("'%s' (no longer supported)", field))
			default:
				invalidFields = append(invalidFields, fmt.Sprintf("'%s'", field))
			}
		}
	}

	// グループ内の無効なフィールドも検証
	if groupsInterface, exists := rawConfig["groups"]; exists {
		if groups, ok := groupsInterface.([]any); ok {
			validGroupFields := map[string]bool{
				"name":     true,
				"filename": true,
				"patterns": true,
			}

			for i, groupInterface := range groups {
				if group, ok := groupInterface.(map[string]any); ok {
					for field := range group {
						if !validGroupFields[field] {
							invalidFields = append(invalidFields, fmt.Sprintf("'%s' in group %d", field, i+1))
						}
					}
				}
			}
		}
	}

	// エラーメッセージの構築
	var errorMessages []string

	if len(deprecatedFields) > 0 {
		errorMessages = append(errorMessages, fmt.Sprintf("deprecated fields found: %s", strings.Join(deprecatedFields, ", ")))
	}

	if len(invalidFields) > 0 {
		errorMessages = append(errorMessages, fmt.Sprintf("unknown fields found: %s", strings.Join(invalidFields, ", ")))
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("%s", strings.Join(errorMessages, "; "))
	}

	return nil
}
