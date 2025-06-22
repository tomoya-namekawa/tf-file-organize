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
	Groups    []GroupConfig     `yaml:"groups"`    // カスタムグループ化ルール
	Overrides map[string]string `yaml:"overrides"` // ブロックタイプ別ファイル名オーバーライド
	Exclude   []string          `yaml:"exclude"`   // 除外パターン
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

// validateConfig validates the loaded configuration for security and correctness
func validateConfig(config *Config) error {
	if err := validateGroups(config.Groups); err != nil {
		return err
	}
	if err := validateOverrides(config.Overrides); err != nil {
		return err
	}
	if err := validateExcludePatterns(config.Exclude); err != nil {
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

// validateOverrides validates override configurations
func validateOverrides(overrides map[string]string) error {
	for blockType, filename := range overrides {
		if blockType == "" {
			return fmt.Errorf("override: block type cannot be empty")
		}
		if filename == "" {
			return fmt.Errorf("override for %s: filename cannot be empty", blockType)
		}
		if err := validateFilename(filename); err != nil {
			return fmt.Errorf("override for %s: invalid filename: %w", blockType, err)
		}
	}
	return nil
}

// validateExcludePatterns validates exclude pattern configurations
func validateExcludePatterns(patterns []string) error {
	for i, pattern := range patterns {
		if pattern == "" {
			return fmt.Errorf("exclude pattern %d: pattern cannot be empty", i)
		}
		if len(pattern) > 100 {
			return fmt.Errorf("exclude pattern %d: pattern too long (max 100 chars)", i)
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

func (c *Config) IsExcluded(resourceType string) bool {
	for _, pattern := range c.Exclude {
		if c.matchPattern(pattern, resourceType) {
			return true
		}
	}
	return false
}

func (c *Config) GetOverrideFilename(blockType string) string {
	if filename, exists := c.Overrides[blockType]; exists {
		return filename
	}
	return ""
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

	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix, suffix := parts[0], parts[1]
		return strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix)
	}

	return false
}
