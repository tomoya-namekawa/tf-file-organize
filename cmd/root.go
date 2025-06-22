package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/usecase"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/version"
)

var (
	inputFile   string
	outputDir   string
	configFile  string
	dryRun      bool
	addComments bool
)

var rootCmd = &cobra.Command{
	Use:     "terraform-file-organize <input-path>",
	Short:   "Organize Terraform files by resource type",
	Version: version.GetVersion(),
	Long: `A CLI tool to split Terraform files into separate files organized by resource type.
Each resource type will be placed in its own file following naming conventions.

Input can be either a single .tf file or a directory containing .tf files.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile = args[0]
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() error {
	// Setup flags
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for split files (default: same as input path)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file for custom grouping rules")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Show what would be done without actually creating files")
	rootCmd.Flags().BoolVar(&addComments, "add-comments", false, "Add descriptive comments to terraform blocks")

	// Enable version flag
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	return rootCmd.Execute()
}

// validatePath prevents path traversal attacks and ensures path safety
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Convert to absolute path to prevent ambiguity
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
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

// validateInputPath validates input file/directory with additional security checks
func validateInputPath(path string) error {
	if err := validatePath(path); err != nil {
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

// validateOutputPath validates output directory path
func validateOutputPath(path string) error {
	if path == "" {
		return nil // Will be set to default later
	}

	if err := validatePath(path); err != nil {
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

// validateConfigPath validates configuration file path
func validateConfigPath(path string) error {
	if path == "" {
		return nil // Optional
	}

	if err := validatePath(path); err != nil {
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

func run() error {
	// Validate all inputs first
	if err := validateInputPath(inputFile); err != nil {
		return fmt.Errorf("invalid input path: %w", err)
	}

	if err := validateOutputPath(outputDir); err != nil {
		return err
	}

	if err := validateConfigPath(configFile); err != nil {
		return err
	}

	// Create usecase request
	req := &usecase.OrganizeFilesRequest{
		InputPath:   inputFile,
		OutputDir:   outputDir,
		ConfigFile:  configFile,
		DryRun:      dryRun,
		AddComments: addComments,
	}

	// Execute usecase
	uc := usecase.NewOrganizeFilesUsecase()
	_, err := uc.Execute(req)
	if err != nil {
		return err
	}

	return nil
}
