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
	GroupBlocks(parsedFile *types.ParsedFile) []*types.BlockGroup
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
	preparationResult, err := uc.prepareExecution(req)
	if err != nil {
		return nil, err
	}

	processingResult, err := uc.processBlocks(req, preparationResult)
	if err != nil {
		return nil, err
	}

	if len(processingResult.allBlocks) == 0 {
		fmt.Println("No Terraform blocks found to organize")
		return &OrganizeFilesResponse{
			ProcessedFiles: processingResult.fileCount,
			TotalBlocks:    0,
			FileGroups:     0,
			OutputDir:      preparationResult.outputDir,
			WasDryRun:      req.DryRun,
		}, nil
	}

	if err := uc.handleOutput(req, preparationResult, processingResult); err != nil {
		return nil, err
	}

	if err := uc.handleSourceFileCleanup(req, preparationResult, processingResult); err != nil {
		return nil, err
	}

	uc.displayResults(req, preparationResult, processingResult)

	return &OrganizeFilesResponse{
		ProcessedFiles: processingResult.fileCount,
		TotalBlocks:    len(processingResult.allBlocks),
		FileGroups:     len(processingResult.groups),
		OutputDir:      preparationResult.outputDir,
		WasDryRun:      req.DryRun,
	}, nil
}

type preparationResult struct {
	stat      os.FileInfo
	outputDir string
	cfg       *config.Config
}

type processingResult struct {
	allBlocks     []*types.Block
	fileCount     int
	sourceFiles   []string
	groups        []*types.BlockGroup
	filesToRemove []string
}

func (uc *OrganizeFilesUsecase) prepareExecution(req *OrganizeFilesRequest) (*preparationResult, error) {
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

	return &preparationResult{
		stat:      stat,
		outputDir: outputDir,
		cfg:       cfg,
	}, nil
}

func (uc *OrganizeFilesUsecase) processBlocks(req *OrganizeFilesRequest, prep *preparationResult) (*processingResult, error) {
	allBlocks, fileCount, sourceFiles, err := uc.parseInput(req.InputPath, prep.stat, req.Recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if len(allBlocks) == 0 {
		return &processingResult{
			allBlocks:   allBlocks,
			fileCount:   fileCount,
			sourceFiles: sourceFiles,
		}, nil
	}

	groups := uc.groupBlocks(allBlocks, prep.cfg)
	fmt.Printf("Organized into %d file groups\n", len(groups))

	filesToRemove := uc.getFilesToRemove(sourceFiles, groups, prep.cfg)

	return &processingResult{
		allBlocks:     allBlocks,
		fileCount:     fileCount,
		sourceFiles:   sourceFiles,
		groups:        groups,
		filesToRemove: filesToRemove,
	}, nil
}

func (uc *OrganizeFilesUsecase) groupBlocks(allBlocks []*types.Block, cfg *config.Config) []*types.BlockGroup {
	parsedFile := &types.ParsedFile{Blocks: allBlocks}
	if uc.splitter != nil {
		return uc.splitter.GroupBlocks(parsedFile)
	}
	s := splitter.NewWithConfig(cfg)
	return s.GroupBlocks(parsedFile)
}

func (uc *OrganizeFilesUsecase) handleOutput(req *OrganizeFilesRequest, prep *preparationResult, proc *processingResult) error {
	if uc.writer != nil {
		if err := uc.writer.WriteGroups(proc.groups); err != nil {
			return fmt.Errorf("failed to write files: %w", err)
		}
	} else {
		w := writer.New(prep.outputDir, req.DryRun)
		if err := w.WriteGroups(proc.groups); err != nil {
			return fmt.Errorf("failed to write files: %w", err)
		}
	}
	return nil
}

func (uc *OrganizeFilesUsecase) handleSourceFileCleanup(req *OrganizeFilesRequest, prep *preparationResult, proc *processingResult) error {
	inputDir := req.InputPath
	if !prep.stat.IsDir() {
		inputDir = filepath.Dir(req.InputPath)
	}
	sameDirectory := (prep.outputDir == inputDir)

	shouldProcessSourceFiles := !req.DryRun && len(proc.filesToRemove) > 0 && sameDirectory

	if shouldProcessSourceFiles {
		if req.Backup {
			if err := uc.backupSourceFiles(proc.filesToRemove, prep.outputDir); err != nil {
				return fmt.Errorf("failed to backup source files: %w", err)
			}
		} else {
			if err := uc.removeSourceFiles(proc.filesToRemove); err != nil {
				return fmt.Errorf("failed to remove source files: %w", err)
			}
		}
	}

	return nil
}

func (uc *OrganizeFilesUsecase) displayResults(req *OrganizeFilesRequest, prep *preparationResult, proc *processingResult) {
	inputDir := req.InputPath
	if !prep.stat.IsDir() {
		inputDir = filepath.Dir(req.InputPath)
	}
	sameDirectory := (prep.outputDir == inputDir)
	shouldProcessSourceFiles := !req.DryRun && len(proc.filesToRemove) > 0 && sameDirectory

	if req.DryRun {
		if sameDirectory && len(proc.filesToRemove) > 0 {
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
				fmt.Printf("Successfully organized Terraform files into: %s (backed up %d source files)\n", prep.outputDir, len(proc.filesToRemove))
			} else {
				fmt.Printf("Successfully organized Terraform files into: %s (removed %d source files)\n", prep.outputDir, len(proc.filesToRemove))
			}
		} else {
			fmt.Printf("Successfully organized Terraform files into: %s\n", prep.outputDir)
		}
	}
}

func (uc *OrganizeFilesUsecase) parseInput(inputPath string, stat os.FileInfo, recursive bool) (blocks []*types.Block, fileCount int, sourceFiles []string, err error) {
	if stat.IsDir() {
		if recursive {
			fmt.Printf("Scanning directory recursively for Terraform files: %s\n", inputPath)
		} else {
			fmt.Printf("Scanning directory for Terraform files: %s\n", inputPath)
		}
		blocks, fileCount, sourceFiles, err = uc.parseDirectory(inputPath, recursive)
		if err != nil {
			return
		}
		fmt.Printf("Found %d .tf files with %d total blocks\n", fileCount, len(blocks))
	} else {
		fmt.Printf("Parsing Terraform file: %s\n", inputPath)
		parsedFile, parseErr := uc.parser.ParseFile(inputPath)
		if parseErr != nil {
			err = parseErr
			return
		}
		blocks = parsedFile.Blocks
		fileCount = 1
		sourceFiles = []string{inputPath}
		fmt.Printf("Found %d blocks\n", len(blocks))
	}

	return blocks, fileCount, sourceFiles, nil
}

func (uc *OrganizeFilesUsecase) parseDirectory(dirPath string, recursive bool) (blocks []*types.Block, fileCount int, sourceFiles []string, err error) {
	if recursive {
		return uc.parseDirectoryRecursive(dirPath)
	}
	return uc.parseDirectoryNonRecursive(dirPath)
}

func (uc *OrganizeFilesUsecase) parseDirectoryRecursive(dirPath string) (blocks []*types.Block, fileCount int, sourceFiles []string, err error) {
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip symbolic links for security
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: skipping symbolic link: %s\n", path)
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tf") {
			fileBlocks, parseErr := uc.processFile(path)
			if parseErr != nil {
				return nil // Continue with warning only for file errors
			}
			blocks = append(blocks, fileBlocks...)
			sourceFiles = append(sourceFiles, path)
			fileCount++
		}

		return nil
	})

	return blocks, fileCount, sourceFiles, err
}

func (uc *OrganizeFilesUsecase) parseDirectoryNonRecursive(dirPath string) (blocks []*types.Block, fileCount int, sourceFiles []string, err error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to read directory: %w", err)
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

		fileBlocks, parseErr := uc.processFile(path)
		if parseErr != nil {
			continue // Continue with warning only for file errors
		}
		blocks = append(blocks, fileBlocks...)
		sourceFiles = append(sourceFiles, path)
		fileCount++
	}

	return blocks, fileCount, sourceFiles, nil
}

func (uc *OrganizeFilesUsecase) processFile(path string) ([]*types.Block, error) {
	if err := uc.validatePath(path); err != nil {
		fmt.Printf("Warning: skipping unsafe path %s: %v\n", path, err)
		return nil, err
	}

	parsedFile, parseErr := uc.parser.ParseFile(path)
	if parseErr != nil {
		fmt.Printf("Warning: failed to parse %s: %v\n", path, parseErr)
		return nil, parseErr
	}

	fmt.Printf("  Processed: %s (%d blocks)\n", path, len(parsedFile.Blocks))
	return parsedFile.Blocks, nil
}

// validatePath prevents path traversal attacks
func (uc *OrganizeFilesUsecase) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	systemDirs := []string{"/etc", "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/sys", "/proc"}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(absPath, sysDir) {
			return fmt.Errorf("access to system directory not allowed: %s", path)
		}
	}

	return nil
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
