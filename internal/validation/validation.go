package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath prevents path traversal attacks and ensures path safety
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(path)

	// Convert to absolute path first to properly validate
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Get current working directory to validate relative paths
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Check if the absolute path is within current working directory or its subdirectories
	// This allows relative paths like ../config.yaml but prevents access to system directories
	if !strings.HasPrefix(absPath, cwd) {
		// Allow if it's still within a reasonable project scope
		projectRoot := filepath.Dir(cwd)
		if !strings.HasPrefix(absPath, projectRoot) {
			return fmt.Errorf("path outside allowed directory scope: %s", path)
		}
	}

	// Ensure the path doesn't access system directories (additional protection)
	systemDirs := []string{"/etc", "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/sys", "/proc"}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(absPath, sysDir) {
			return fmt.Errorf("access to system directory not allowed: %s", path)
		}
	}

	return nil
}

// ValidateInputPath validates input file/directory with additional security checks
func ValidateInputPath(path string) error {
	if err := ValidatePath(path); err != nil {
		return err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist or is not accessible: %s", path)
	}

	// Check for symbolic links to prevent symlink attacks
	if stat.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symbolic links are not allowed for security reasons: %s", path)
	}

	// Additional check: ensure it's a regular file or directory
	if !stat.IsDir() && !stat.Mode().IsRegular() {
		return fmt.Errorf("path must be a regular file or directory: %s", path)
	}

	return nil
}

// ValidateOutputPath validates output directory path
func ValidateOutputPath(path string) error {
	if path == "" {
		return nil // Will be set to default later
	}

	if err := ValidatePath(path); err != nil {
		return fmt.Errorf("invalid output directory: %w", err)
	}

	// If directory exists, check if it's actually a directory
	if stat, err := os.Stat(path); err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("output path exists but is not a directory: %s", path)
		}
	}

	return nil
}

// ValidateConfigPath validates configuration file path
func ValidateConfigPath(path string) error {
	if path == "" {
		return nil // Optional
	}

	if err := ValidatePath(path); err != nil {
		return fmt.Errorf("invalid config file path: %w", err)
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("config file does not exist: %s", path)
	}

	// Ensure it's a regular file
	if !stat.Mode().IsRegular() {
		return fmt.Errorf("config path must be a regular file: %s", path)
	}

	// Check file size to prevent DoS attacks
	const maxConfigSize = 1024 * 1024 // 1MB
	if stat.Size() > maxConfigSize {
		return fmt.Errorf("config file too large (max %d bytes): %s", maxConfigSize, path)
	}

	return nil
}

// ValidateFlagCombination validates that output-dir and recursive flags are not used together
func ValidateFlagCombination(outputDir string, recursive bool) error {
	// Prevent using -o (output-dir) and -r (recursive) together
	// because combining multiple directories into one output is unnatural
	if outputDir != "" && recursive {
		return fmt.Errorf("cannot use --output-dir (-o) with --recursive (-r): combining multiple directories into one output is not supported")
	}

	return nil
}
