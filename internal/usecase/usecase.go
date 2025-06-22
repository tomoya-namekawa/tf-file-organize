package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
	"github.com/tomoya-namekawa/tf-file-organize/internal/parser"
	"github.com/tomoya-namekawa/tf-file-organize/internal/splitter"
	"github.com/tomoya-namekawa/tf-file-organize/internal/writer"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

const (
	localsFile    = "locals.tf"
	outputsFile   = "outputs.tf"
	providersFile = "providers.tf"
	terraformFile = "terraform.tf"
	variablesFile = "variables.tf"
)

type ParserInterface interface {
	ParseFile(filename string) (*types.ParsedFile, error)
}

type SplitterInterface interface {
	GroupBlocks(parsedFiles *types.ParsedFiles) ([]*types.BlockGroup, error)
}

type WriterInterface interface {
	WriteGroups(groups []*types.BlockGroup) error
}

type ConfigLoaderInterface interface {
	LoadConfig(configPath string) (*config.Config, error)
}

type OrganizeFilesRequest struct {
	InputPath  string
	OutputDir  string
	ConfigFile string
	DryRun     bool
	Recursive  bool
	Backup     bool
}

type OrganizeFilesResponse struct {
	ProcessedFiles int
	TotalBlocks    int
	FileGroups     int
	OutputDir      string
	WasDryRun      bool
}

type OrganizeFilesUsecase struct {
	parser       ParserInterface
	splitter     SplitterInterface
	writer       WriterInterface
	configLoader ConfigLoaderInterface
}

func NewOrganizeFilesUsecase() *OrganizeFilesUsecase {
	return &OrganizeFilesUsecase{
		parser:       parser.New(),
		splitter:     nil, // Initialized with configuration in Execute
		writer:       nil, // Initialized in Execute
		configLoader: &DefaultConfigLoader{},
	}
}

func NewOrganizeFilesUsecaseWithDeps(p ParserInterface, s SplitterInterface, w WriterInterface, c ConfigLoaderInterface) *OrganizeFilesUsecase {
	return &OrganizeFilesUsecase{
		parser:       p,
		splitter:     s,
		writer:       w,
		configLoader: c,
	}
}

type DefaultConfigLoader struct{}

func (d *DefaultConfigLoader) LoadConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		fmt.Printf("Loading configuration from: %s\n", configPath)
		return config.LoadConfig(configPath)
	}

	defaultConfigs := []string{
		"tf-file-organize.yaml",
		"tf-file-organize.yml",
		".tf-file-organize.yaml",
		".tf-file-organize.yml",
	}

	for _, defaultConfig := range defaultConfigs {
		if _, err := os.Stat(defaultConfig); err == nil {
			fmt.Printf("Loading configuration from: %s\n", defaultConfig)
			return config.LoadConfig(defaultConfig)
		}
	}

	return &config.Config{}, nil
}

// Execute performs the main business logic for organizing Terraform files.
func (uc *OrganizeFilesUsecase) Execute(req *OrganizeFilesRequest) (*OrganizeFilesResponse, error) {
	// 1. Prepare: input validation and config loading
	stat, err := os.Stat(req.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access input path: %w", err)
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		if stat.IsDir() {
			outputDir = req.InputPath
		} else {
			outputDir = filepath.Dir(req.InputPath)
		}
	}

	cfg, err := uc.configLoader.LoadConfig(req.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Parse: extract blocks from files
	parsedFiles, err := uc.parseInput(req.InputPath, stat, req.Recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if parsedFiles.TotalBlocks() == 0 {
		fmt.Println("No Terraform blocks found to organize")
		return &OrganizeFilesResponse{
			ProcessedFiles: len(parsedFiles.Files),
			TotalBlocks:    0,
			FileGroups:     0,
			OutputDir:      outputDir,
			WasDryRun:      req.DryRun,
		}, nil
	}

	// 3. Group: organize blocks by type and config
	groups, err := uc.getSplitter(cfg).GroupBlocks(parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to group blocks: %w", err)
	}
	fmt.Printf("Organized into %d file groups\n", len(groups))

	// 4. Write: output organized files
	if err := uc.getWriter(outputDir, req.DryRun).WriteGroups(groups); err != nil {
		return nil, fmt.Errorf("failed to write files: %w", err)
	}

	// 5. Cleanup: handle source files if needed
	filesToRemove := uc.getFilesToRemove(parsedFiles.FileNames(), groups, cfg)
	if err := uc.handleSourceFileCleanup(req, stat, outputDir, filesToRemove); err != nil {
		return nil, err
	}

	// 6. Display results
	uc.displayResults(req, stat, outputDir, filesToRemove)

	return &OrganizeFilesResponse{
		ProcessedFiles: len(parsedFiles.Files),
		TotalBlocks:    parsedFiles.TotalBlocks(),
		FileGroups:     len(groups),
		OutputDir:      outputDir,
		WasDryRun:      req.DryRun,
	}, nil
}

func (uc *OrganizeFilesUsecase) getSplitter(cfg *config.Config) SplitterInterface {
	if uc.splitter != nil {
		return uc.splitter
	}
	return splitter.NewWithConfig(cfg)
}

func (uc *OrganizeFilesUsecase) getWriter(outputDir string, dryRun bool) WriterInterface {
	if uc.writer != nil {
		return uc.writer
	}
	return writer.New(outputDir, dryRun)
}

func (uc *OrganizeFilesUsecase) handleSourceFileCleanup(req *OrganizeFilesRequest, stat os.FileInfo, outputDir string, filesToRemove []string) error {
	inputDir := req.InputPath
	if !stat.IsDir() {
		inputDir = filepath.Dir(req.InputPath)
	}
	sameDirectory := (outputDir == inputDir)

	shouldProcessSourceFiles := !req.DryRun && len(filesToRemove) > 0 && sameDirectory

	if shouldProcessSourceFiles {
		if req.Backup {
			if err := uc.backupSourceFiles(filesToRemove, outputDir); err != nil {
				return fmt.Errorf("failed to backup source files: %w", err)
			}
		} else {
			if err := uc.removeSourceFiles(filesToRemove); err != nil {
				return fmt.Errorf("failed to remove source files: %w", err)
			}
		}
	}

	return nil
}

func (uc *OrganizeFilesUsecase) displayResults(req *OrganizeFilesRequest, stat os.FileInfo, outputDir string, filesToRemove []string) {
	inputDir := req.InputPath
	if !stat.IsDir() {
		inputDir = filepath.Dir(req.InputPath)
	}
	sameDirectory := (outputDir == inputDir)
	shouldProcessSourceFiles := !req.DryRun && len(filesToRemove) > 0 && sameDirectory

	if req.DryRun {
		if sameDirectory && len(filesToRemove) > 0 {
			if req.Backup {
				fmt.Println("Plan completed. Use 'run --backup' to actually create files and backup source files.")
			} else {
				fmt.Println("Plan completed. Use 'run' to actually create files and remove source files.")
			}
		} else {
			fmt.Println("Plan completed. Use 'run' to actually create files.")
		}
	} else {
		if shouldProcessSourceFiles {
			if req.Backup {
				fmt.Printf("Successfully organized Terraform files into: %s (backed up %d source files)\n", outputDir, len(filesToRemove))
			} else {
				fmt.Printf("Successfully organized Terraform files into: %s (removed %d source files)\n", outputDir, len(filesToRemove))
			}
		} else {
			fmt.Printf("Successfully organized Terraform files into: %s\n", outputDir)
		}
	}
}

func (uc *OrganizeFilesUsecase) parseInput(inputPath string, stat os.FileInfo, recursive bool) (*types.ParsedFiles, error) {
	if stat.IsDir() {
		if recursive {
			fmt.Printf("Scanning directory recursively for Terraform files: %s\n", inputPath)
		} else {
			fmt.Printf("Scanning directory for Terraform files: %s\n", inputPath)
		}
		parsedFiles, err := uc.parseDirectory(inputPath, recursive)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Found %d .tf files with %d total blocks\n", len(parsedFiles.Files), parsedFiles.TotalBlocks())
		return parsedFiles, nil
	} else {
		fmt.Printf("Parsing Terraform file: %s\n", inputPath)
		parsedFile, err := uc.parser.ParseFile(inputPath)
		if err != nil {
			return nil, err
		}
		parsedFiles := &types.ParsedFiles{
			Files: []*types.ParsedFile{parsedFile},
		}
		fmt.Printf("Found %d blocks\n", parsedFiles.TotalBlocks())
		return parsedFiles, nil
	}
}

func (uc *OrganizeFilesUsecase) parseDirectory(dirPath string, recursive bool) (*types.ParsedFiles, error) {
	if recursive {
		return uc.parseDirectoryRecursive(dirPath)
	}
	return uc.parseDirectoryNonRecursive(dirPath)
}

func (uc *OrganizeFilesUsecase) parseDirectoryRecursive(dirPath string) (*types.ParsedFiles, error) {
	parsedFiles := &types.ParsedFiles{
		Files: make([]*types.ParsedFile, 0),
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip symbolic links for security
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: skipping symbolic link: %s\n", path)
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tf") {
			parsedFile, parseErr := uc.parser.ParseFile(path)
			if parseErr != nil {
				fmt.Printf("Warning: failed to parse file %s: %v\n", path, parseErr)
				return nil // Continue with warning only for file errors
			}
			parsedFiles.Files = append(parsedFiles.Files, parsedFile)
		}

		return nil
	})

	return parsedFiles, err
}

func (uc *OrganizeFilesUsecase) parseDirectoryNonRecursive(dirPath string) (*types.ParsedFiles, error) {
	parsedFiles := &types.ParsedFiles{
		Files: make([]*types.ParsedFile, 0),
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		path := filepath.Join(dirPath, entry.Name())

		// Skip symbolic links for security
		if info, infoErr := entry.Info(); infoErr == nil && info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: skipping symbolic link: %s\n", path)
			continue
		}

		parsedFile, parseErr := uc.parser.ParseFile(path)
		if parseErr != nil {
			fmt.Printf("Warning: failed to parse file %s: %v\n", path, parseErr)
			continue // Continue with warning only for file errors
		}
		parsedFiles.Files = append(parsedFiles.Files, parsedFile)
	}

	return parsedFiles, nil
}

func (uc *OrganizeFilesUsecase) backupSourceFiles(sourceFiles []string, outputDir string) error {
	backupDir := filepath.Join(outputDir, "backup")
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	for _, sourceFile := range sourceFiles {
		fileName := filepath.Base(sourceFile)
		backupPath := filepath.Join(backupDir, fileName)

		if err := os.Rename(sourceFile, backupPath); err != nil {
			return fmt.Errorf("failed to backup file %s: %w", sourceFile, err)
		}
		fmt.Printf("  Backed up: %s -> %s\n", sourceFile, backupPath)
	}

	return nil
}

func (uc *OrganizeFilesUsecase) removeSourceFiles(sourceFiles []string) error {
	for _, sourceFile := range sourceFiles {
		if err := os.Remove(sourceFile); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", sourceFile, err)
		}
		fmt.Printf("  Removed: %s\n", sourceFile)
	}

	return nil
}

// getFilesToRemove identifies source files that should be removed for idempotency
func (uc *OrganizeFilesUsecase) getFilesToRemove(sourceFiles []string, groups []*types.BlockGroup, _ *config.Config) []string {
	generatedFiles := make(map[string]bool)
	for _, group := range groups {
		generatedFiles[group.FileName] = true
	}

	var filesToRemove []string
	for _, sourceFile := range sourceFiles {
		fileName := filepath.Base(sourceFile)

		// Exclude tool-generated files
		if strings.HasPrefix(fileName, "data__") ||
			strings.HasPrefix(fileName, "resource__") ||
			strings.HasPrefix(fileName, "module__") {
			continue
		}

		// Exclude newly generated files for idempotency
		if generatedFiles[fileName] {
			continue
		}

		isDefaultGenerated := fileName == localsFile ||
			fileName == outputsFile ||
			fileName == providersFile ||
			fileName == terraformFile ||
			fileName == variablesFile

		if isDefaultGenerated {
			// Exclude default generated files
			continue
		}

		// Target user-created files for removal
		filesToRemove = append(filesToRemove, sourceFile)
	}

	return filesToRemove
}
