// Package config provides configuration management for tf-file-organize.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Groups       []GroupConfig `yaml:"groups"`
	ExcludeFiles []string      `yaml:"exclude_files"`
}

type GroupConfig struct {
	Name     string   `yaml:"name"`
	Filename string   `yaml:"filename"`
	Patterns []string `yaml:"patterns"`
}

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

	stat, err := os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}

	const maxConfigSize = 1024 * 1024 // 1MB limit to prevent DoS
	if stat.Size() > maxConfigSize {
		return nil, fmt.Errorf("config file too large (max %d bytes): %d bytes", maxConfigSize, stat.Size())
	}

	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("config path must be a regular file: %s", configPath)
	}

	data, err := os.ReadFile(configPath) //nolint:gosec // configPath is validated for safety
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := validateConfigFields(data); err != nil {
		return nil, fmt.Errorf("invalid configuration fields: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func ValidateConfig(cfg *Config) error {
	groupNames := make(map[string]bool)
	for i, group := range cfg.Groups {
		if groupNames[group.Name] {
			return fmt.Errorf("duplicate group name '%s' at group %d", group.Name, i+1)
		}
		groupNames[group.Name] = true
	}

	filenames := make(map[string]string)
	for i, group := range cfg.Groups {
		if existingGroup, exists := filenames[group.Filename]; exists {
			return fmt.Errorf("duplicate filename '%s' in group '%s' (group %d) - already used by group '%s'", group.Filename, group.Name, i+1, existingGroup)
		}
		filenames[group.Filename] = group.Name
	}

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
		if !isValidPattern(pattern) {
			return fmt.Errorf("exclude file pattern %d ('%s') contains invalid characters", i+1, pattern)
		}
	}

	return nil
}

func isValidPattern(pattern string) bool {
	invalidChars := []string{"\x00", "\n", "\r", "\t"}
	for _, char := range invalidChars {
		if strings.Contains(pattern, char) {
			return false
		}
	}
	return true
}

func validateConfig(config *Config) error {
	if err := validateGroups(config.Groups); err != nil {
		return err
	}
	if err := validateExcludeFilePatterns(config.ExcludeFiles); err != nil {
		return err
	}
	return nil
}

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

// validateFilename ensures filename safety
func validateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename cannot contain '..'")
	}

	if strings.ContainsAny(filename, "/\\:*?\"<>|") {
		return fmt.Errorf("filename contains invalid characters")
	}

	if len(filename) > 255 {
		return fmt.Errorf("filename too long (max 255 chars)")
	}

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

	return c.matchWithWildcards(pattern, text)
}

// matchWithWildcards processes patterns with multiple wildcards
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

	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}

	return patternIndex == len(pattern)
}

func validateConfigFields(data []byte) error {
	var rawConfig map[string]any
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	validTopLevelFields := map[string]bool{
		"groups":        true,
		"exclude_files": true,
	}

	var invalidFields []string
	var deprecatedFields []string

	for field := range rawConfig {
		if !validTopLevelFields[field] {
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
