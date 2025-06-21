package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/namekawa/terraform-file-organize/internal/config"
	"github.com/namekawa/terraform-file-organize/internal/parser"
	"github.com/namekawa/terraform-file-organize/internal/splitter"
	"github.com/namekawa/terraform-file-organize/internal/writer"
	"github.com/namekawa/terraform-file-organize/pkg/types"
)

var (
	inputFile  string
	outputDir  string
	configFile string
	dryRun     bool
)

var rootCmd = &cobra.Command{
	Use:   "terraform-file-organize <input-path>",
	Short: "Organize Terraform files by resource type",
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

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for split files (default: same as input path)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file for custom grouping rules")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Show what would be done without actually creating files")
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
	
	stat, err := os.Stat(inputFile)
	if err != nil {
		return fmt.Errorf("failed to access input path: %w", err)
	}

	// 出力ディレクトリのデフォルト設定
	if outputDir == "" {
		if stat.IsDir() {
			outputDir = inputFile
		} else {
			outputDir = filepath.Dir(inputFile)
		}
	}

	var cfg *config.Config
	
	// デフォルト設定ファイルを探す
	if configFile == "" {
		defaultConfigs := []string{
			"terraform-file-organize.yaml",
			"terraform-file-organize.yml",
			".terraform-file-organize.yaml",
			".terraform-file-organize.yml",
		}
		
		for _, defaultConfig := range defaultConfigs {
			// セキュリティ検証を追加
			if err := validateConfigPath(defaultConfig); err == nil {
				configFile = defaultConfig
				break
			}
		}
	}
	
	if configFile != "" {
		fmt.Printf("Loading configuration from: %s\n", configFile)
		cfg, err = config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		cfg = &config.Config{}
	}

	var allBlocks []*types.Block
	var fileCount int

	if stat.IsDir() {
		fmt.Printf("Scanning directory for Terraform files: %s\n", inputFile)
		allBlocks, fileCount, err = parseDirectory(inputFile)
		if err != nil {
			return fmt.Errorf("failed to parse directory: %w", err)
		}
		fmt.Printf("Found %d .tf files with %d total blocks\n", fileCount, len(allBlocks))
	} else {
		fmt.Printf("Parsing Terraform file: %s\n", inputFile)
		p := parser.New()
		parsedFile, err := p.ParseFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}
		allBlocks = parsedFile.Blocks
		fileCount = 1
		fmt.Printf("Found %d blocks\n", len(allBlocks))
	}

	if len(allBlocks) == 0 {
		fmt.Println("No Terraform blocks found to organize")
		return nil
	}

	combinedFile := &types.ParsedFile{Blocks: allBlocks}
	s := splitter.NewWithConfig(cfg)
	groups := s.GroupBlocks(combinedFile)
	
	fmt.Printf("Organized into %d file groups\n", len(groups))

	w := writer.New(outputDir, dryRun)
	if err := w.WriteGroups(groups); err != nil {
		return fmt.Errorf("failed to write files: %w", err)
	}

	if dryRun {
		fmt.Println("Dry run completed. Use --dry-run=false to actually create files.")
	} else {
		fmt.Printf("Successfully organized Terraform files into: %s\n", outputDir)
	}

	return nil
}

func parseDirectory(dirPath string) ([]*types.Block, int, error) {
	var allBlocks []*types.Block
	fileCount := 0
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// セキュリティチェック: シンボリックリンクをスキップ
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: skipping symbolic link: %s\n", path)
			return nil
		}
		
		if !info.IsDir() && strings.HasSuffix(path, ".tf") {
			// 各ファイルパスも検証
			if err := validatePath(path); err != nil {
				fmt.Printf("Warning: skipping unsafe path %s: %v\n", path, err)
				return nil
			}
			
			p := parser.New()
			parsedFile, parseErr := p.ParseFile(path)
			if parseErr != nil {
				fmt.Printf("Warning: failed to parse %s: %v\n", path, parseErr)
				return nil
			}
			
			allBlocks = append(allBlocks, parsedFile.Blocks...)
			fileCount++
			fmt.Printf("  Processed: %s (%d blocks)\n", path, len(parsedFile.Blocks))
		}
		
		return nil
	})
	
	return allBlocks, fileCount, err
}