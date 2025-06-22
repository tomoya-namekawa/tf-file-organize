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

// Default Terraform file names
const (
	localsFile    = "locals.tf"
	outputsFile   = "outputs.tf"
	providersFile = "providers.tf"
	terraformFile = "terraform.tf"
	variablesFile = "variables.tf"
)

// ParserInterface はParserの抽象化
type ParserInterface interface {
	ParseFile(filename string) (*types.ParsedFile, error)
}

// SplitterInterface はSplitterの抽象化
type SplitterInterface interface {
	GroupBlocks(parsedFile *types.ParsedFile) []*types.BlockGroup
}

// WriterInterface はWriterの抽象化
type WriterInterface interface {
	WriteGroups(groups []*types.BlockGroup) error
}

// ConfigLoaderInterface は設定読み込みの抽象化
type ConfigLoaderInterface interface {
	LoadConfig(configPath string) (*config.Config, error)
}

// OrganizeFilesRequest は OrganizeFiles ユースケースのリクエスト
type OrganizeFilesRequest struct {
	InputPath  string
	OutputDir  string
	ConfigFile string
	DryRun     bool
	Recursive  bool
	Backup     bool
}

// OrganizeFilesResponse は OrganizeFiles ユースケースのレスポンス
type OrganizeFilesResponse struct {
	ProcessedFiles int
	TotalBlocks    int
	FileGroups     int
	OutputDir      string
	WasDryRun      bool
}

// OrganizeFilesUsecase は Terraform ファイル整理のユースケース
type OrganizeFilesUsecase struct {
	parser       ParserInterface
	splitter     SplitterInterface
	writer       WriterInterface
	configLoader ConfigLoaderInterface
}

// NewOrganizeFilesUsecase は新しい OrganizeFilesUsecase を作成
func NewOrganizeFilesUsecase() *OrganizeFilesUsecase {
	return &OrganizeFilesUsecase{
		parser:       parser.New(),
		splitter:     nil, // Executeで設定付きで初期化
		writer:       nil, // Executeで初期化
		configLoader: &DefaultConfigLoader{},
	}
}

// NewOrganizeFilesUsecaseWithDeps は依存関係を注入して OrganizeFilesUsecase を作成
func NewOrganizeFilesUsecaseWithDeps(p ParserInterface, s SplitterInterface, w WriterInterface, c ConfigLoaderInterface) *OrganizeFilesUsecase {
	return &OrganizeFilesUsecase{
		parser:       p,
		splitter:     s,
		writer:       w,
		configLoader: c,
	}
}

// DefaultConfigLoader wraps config.LoadConfig for dependency injection.
type DefaultConfigLoader struct{}

// LoadConfig loads configuration using the standard config loader.
func (d *DefaultConfigLoader) LoadConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		fmt.Printf("Loading configuration from: %s\n", configPath)
		return config.LoadConfig(configPath)
	}

	// 設定ファイルが指定されていない場合はデフォルトを探す
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
	// 前処理: 入力検証と設定準備
	preparationResult, err := uc.prepareExecution(req)
	if err != nil {
		return nil, err
	}

	// ファイル解析とブロック処理
	processingResult, err := uc.processBlocks(req, preparationResult)
	if err != nil {
		return nil, err
	}

	// ブロックが見つからない場合の早期終了
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

	// ファイル出力処理
	if err := uc.handleOutput(req, preparationResult, processingResult); err != nil {
		return nil, err
	}

	// ソースファイル処理とクリーンアップ
	if err := uc.handleSourceFileCleanup(req, preparationResult, processingResult); err != nil {
		return nil, err
	}

	// 結果表示とレスポンス作成
	uc.displayResults(req, preparationResult, processingResult)

	return &OrganizeFilesResponse{
		ProcessedFiles: processingResult.fileCount,
		TotalBlocks:    len(processingResult.allBlocks),
		FileGroups:     len(processingResult.groups),
		OutputDir:      preparationResult.outputDir,
		WasDryRun:      req.DryRun,
	}, nil
}

// preparationResult holds the result of request preparation
type preparationResult struct {
	stat      os.FileInfo
	outputDir string
	cfg       *config.Config
}

// processingResult holds the result of block processing
type processingResult struct {
	allBlocks     []*types.Block
	fileCount     int
	sourceFiles   []string
	groups        []*types.BlockGroup
	filesToRemove []string
}

// prepareExecution validates and prepares the execution environment
func (uc *OrganizeFilesUsecase) prepareExecution(req *OrganizeFilesRequest) (*preparationResult, error) {
	// 入力パスの情報を取得
	stat, err := os.Stat(req.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access input path: %w", err)
	}

	// 出力ディレクトリのデフォルト設定
	outputDir := req.OutputDir
	if outputDir == "" {
		if stat.IsDir() {
			outputDir = req.InputPath
		} else {
			outputDir = filepath.Dir(req.InputPath)
		}
	}

	// 設定ファイルの処理
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

// processBlocks parses input files and groups blocks
func (uc *OrganizeFilesUsecase) processBlocks(req *OrganizeFilesRequest, prep *preparationResult) (*processingResult, error) {
	// ファイルの解析
	allBlocks, fileCount, sourceFiles, err := uc.parseInput(req.InputPath, prep.stat, req.Recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// ブロックが見つからない場合は早期終了
	if len(allBlocks) == 0 {
		return &processingResult{
			allBlocks:   allBlocks,
			fileCount:   fileCount,
			sourceFiles: sourceFiles,
		}, nil
	}

	// ブロックのグループ化
	groups := uc.groupBlocks(allBlocks, prep.cfg)
	fmt.Printf("Organized into %d file groups\n", len(groups))

	// 削除対象ファイルの特定
	filesToRemove := uc.getFilesToRemove(sourceFiles, groups, prep.cfg)

	return &processingResult{
		allBlocks:     allBlocks,
		fileCount:     fileCount,
		sourceFiles:   sourceFiles,
		groups:        groups,
		filesToRemove: filesToRemove,
	}, nil
}

// groupBlocks groups blocks using either injected splitter or default with config
func (uc *OrganizeFilesUsecase) groupBlocks(allBlocks []*types.Block, cfg *config.Config) []*types.BlockGroup {
	parsedFile := &types.ParsedFile{Blocks: allBlocks}
	if uc.splitter != nil {
		return uc.splitter.GroupBlocks(parsedFile)
	}
	s := splitter.NewWithConfig(cfg)
	return s.GroupBlocks(parsedFile)
}

// handleOutput writes the grouped blocks to files
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

// handleSourceFileCleanup manages backup and removal of source files
func (uc *OrganizeFilesUsecase) handleSourceFileCleanup(req *OrganizeFilesRequest, prep *preparationResult, proc *processingResult) error {
	// 入力と出力が同じディレクトリかチェック
	inputDir := req.InputPath
	if !prep.stat.IsDir() {
		inputDir = filepath.Dir(req.InputPath)
	}
	sameDirectory := (prep.outputDir == inputDir)

	// ソースファイル処理が必要かチェック
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

// displayResults shows the execution results to the user
func (uc *OrganizeFilesUsecase) displayResults(req *OrganizeFilesRequest, prep *preparationResult, proc *processingResult) {
	// 入力と出力が同じディレクトリかチェック
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

// parseInput は入力パス（ファイルまたはディレクトリ）を解析
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

// parseDirectory はディレクトリ内の.tfファイルを解析（再帰可能）
func (uc *OrganizeFilesUsecase) parseDirectory(dirPath string, recursive bool) (blocks []*types.Block, fileCount int, sourceFiles []string, err error) {
	if recursive {
		return uc.parseDirectoryRecursive(dirPath)
	}
	return uc.parseDirectoryNonRecursive(dirPath)
}

// parseDirectoryRecursive はディレクトリを再帰的に解析
func (uc *OrganizeFilesUsecase) parseDirectoryRecursive(dirPath string) (blocks []*types.Block, fileCount int, sourceFiles []string, err error) {
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// シンボリックリンクをスキップ（セキュリティ上の理由）
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: skipping symbolic link: %s\n", path)
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tf") {
			fileBlocks, parseErr := uc.processFile(path)
			if parseErr != nil {
				return nil // ファイルエラーは警告のみで継続
			}
			blocks = append(blocks, fileBlocks...)
			sourceFiles = append(sourceFiles, path)
			fileCount++
		}

		return nil
	})

	return blocks, fileCount, sourceFiles, err
}

// parseDirectoryNonRecursive は指定されたディレクトリのみを解析
func (uc *OrganizeFilesUsecase) parseDirectoryNonRecursive(dirPath string) (blocks []*types.Block, fileCount int, sourceFiles []string, err error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		// ディレクトリはスキップ
		if entry.IsDir() {
			continue
		}

		// .tfファイルのみ処理
		if !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		path := filepath.Join(dirPath, entry.Name())

		// シンボリックリンクをスキップ（セキュリティ上の理由）
		if info, infoErr := entry.Info(); infoErr == nil && info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: skipping symbolic link: %s\n", path)
			continue
		}

		fileBlocks, parseErr := uc.processFile(path)
		if parseErr != nil {
			continue // ファイルエラーは警告のみで継続
		}
		blocks = append(blocks, fileBlocks...)
		sourceFiles = append(sourceFiles, path)
		fileCount++
	}

	return blocks, fileCount, sourceFiles, nil
}

// processFile は単一ファイルを処理
func (uc *OrganizeFilesUsecase) processFile(path string) ([]*types.Block, error) {
	// パスの安全性を確認
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

// validatePath はパスの安全性を検証（パストラバーサル攻撃を防ぐ）
func (uc *OrganizeFilesUsecase) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// パスを正規化
	cleanPath := filepath.Clean(path)

	// パストラバーサル攻撃をチェック
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// 絶対パスに変換して曖昧さを排除
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// システムディレクトリへのアクセスを防ぐ
	systemDirs := []string{"/etc", "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/sys", "/proc"}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(absPath, sysDir) {
			return fmt.Errorf("access to system directory not allowed: %s", path)
		}
	}

	return nil
}

// backupSourceFiles はソースファイルをbackupディレクトリに移動
func (uc *OrganizeFilesUsecase) backupSourceFiles(sourceFiles []string, outputDir string) error {
	backupDir := filepath.Join(outputDir, "backup")
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	for _, sourceFile := range sourceFiles {
		fileName := filepath.Base(sourceFile)
		backupPath := filepath.Join(backupDir, fileName)

		// ファイルが既に存在する場合は上書き
		if err := os.Rename(sourceFile, backupPath); err != nil {
			return fmt.Errorf("failed to backup file %s: %w", sourceFile, err)
		}
		fmt.Printf("  Backed up: %s -> %s\n", sourceFile, backupPath)
	}

	return nil
}

// removeSourceFiles はソースファイルを削除
func (uc *OrganizeFilesUsecase) removeSourceFiles(sourceFiles []string) error {
	for _, sourceFile := range sourceFiles {
		if err := os.Remove(sourceFile); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", sourceFile, err)
		}
		fmt.Printf("  Removed: %s\n", sourceFile)
	}

	return nil
}

// getFilesToRemove は削除すべきソースファイルを特定
func (uc *OrganizeFilesUsecase) getFilesToRemove(sourceFiles []string, groups []*types.BlockGroup, _ *config.Config) []string {
	// 生成される予定のファイル名を収集
	generatedFiles := make(map[string]bool)
	for _, group := range groups {
		generatedFiles[group.FileName] = true
	}

	var filesToRemove []string
	for _, sourceFile := range sourceFiles {
		fileName := filepath.Base(sourceFile)

		// 生成済みファイルパターンは削除対象から除外（ツールが生成したファイル）
		if strings.HasPrefix(fileName, "data__") ||
			strings.HasPrefix(fileName, "resource__") ||
			strings.HasPrefix(fileName, "module__") {
			continue
		}

		// 新しく生成されるファイルは削除対象から除外（冪等性のため）
		// 既に生成されたファイルを削除してしまうと、次回実行時にファイルが無くなってしまう
		if generatedFiles[fileName] {
			continue
		}

		// デフォルト生成ファイルかチェック
		isDefaultGenerated := fileName == localsFile ||
			fileName == outputsFile ||
			fileName == providersFile ||
			fileName == terraformFile ||
			fileName == variablesFile

		if isDefaultGenerated {
			// デフォルト生成ファイルは削除対象から除外（これらも生成される可能性がある）
			continue
		}

		// それ以外のファイル（main.tf等のユーザー作成ファイル）のみを削除対象とする
		filesToRemove = append(filesToRemove, sourceFile)
	}

	return filesToRemove
}
