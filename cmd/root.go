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
	inputFile  string
	outputDir  string
	configFile string
	dryRun     bool
	recursive  bool
	backup     bool
)

var rootCmd = &cobra.Command{
	Use:     "terraform-file-organize <input-path>",
	Short:   "Organize Terraform files by resource type",
	Version: version.GetVersion(),
	Long: `A CLI tool to split Terraform files into separate files organized by resource type.
Each resource type will be placed in its own file following naming conventions.

Input can be either a single .tf file or a directory containing .tf files.
By default, only files in the specified directory are processed. Use -r for recursive processing.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile = args[0]
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute runs the root command and handles CLI argument parsing.
func Execute() error {
	// Setup flags
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for split files (default: same as input path)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file for custom grouping rules")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Show what would be done without actually creating files")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Process directories recursively")
	rootCmd.Flags().BoolVar(&backup, "backup", false, "Backup original files to 'backup' subdirectory before organizing")

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

// validateFlagCombinations validates incompatible flag combinations
func validateFlagCombinations() error {
	// Prevent using -o (output-dir) and -r (recursive) together
	// because combining multiple directories into one output is unnatural
	if outputDir != "" && recursive {
		return fmt.Errorf("cannot use --output-dir (-o) with --recursive (-r): combining multiple directories into one output is not supported")
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

	// Validate flag combinations
	if err := validateFlagCombinations(); err != nil {
		return err
	}

	// Create usecase request
	req := &usecase.OrganizeFilesRequest{
		InputPath:  inputFile,
		OutputDir:  outputDir,
		ConfigFile: configFile,
		DryRun:     dryRun,
		Recursive:  recursive,
		Backup:     backup,
	}

	// Execute usecase
	uc := usecase.NewOrganizeFilesUsecase()
	_, err := uc.Execute(req)
	if err != nil {
		return err
	}

	return nil
}
