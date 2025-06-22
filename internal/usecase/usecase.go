package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/config"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/parser"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/splitter"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/writer"
	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
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
		"terraform-file-organize.yaml",
		"terraform-file-organize.yml",
		".terraform-file-organize.yaml",
		".terraform-file-organize.yml",
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
//
//nolint:gocyclo // Complex business logic with multiple validation steps
func (uc *OrganizeFilesUsecase) Execute(req *OrganizeFilesRequest) (*OrganizeFilesResponse, error) {
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

	// ファイルの解析
	allBlocks, fileCount, sourceFiles, err := uc.parseInput(req.InputPath, stat, req.Recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if len(allBlocks) == 0 {
		fmt.Println("No Terraform blocks found to organize")
		return &OrganizeFilesResponse{
			ProcessedFiles: fileCount,
			TotalBlocks:    0,
			FileGroups:     0,
			OutputDir:      outputDir,
			WasDryRun:      req.DryRun,
		}, nil
	}

	// ブロックのグループ化
	// 依存性注入されたsplitterを使用するか、設定ファイル付きのデフォルトを作成
	var groups []*types.BlockGroup
	parsedFile := &types.ParsedFile{Blocks: allBlocks}
	if uc.splitter != nil {
		groups = uc.splitter.GroupBlocks(parsedFile)
	} else {
		s := splitter.NewWithConfig(cfg)
		groups = s.GroupBlocks(parsedFile)
	}

	fmt.Printf("Organized into %d file groups\n", len(groups))

	// ファイルの書き出し
	// 依存性注入されたwriterを使用するか、デフォルトを作成
	if uc.writer != nil {
		if err := uc.writer.WriteGroups(groups); err != nil {
			return nil, fmt.Errorf("failed to write files: %w", err)
		}
	} else {
		w := writer.New(outputDir, req.DryRun)
		if err := w.WriteGroups(groups); err != nil {
			return nil, fmt.Errorf("failed to write files: %w", err)
		}
	}

	// バックアップとソースファイルの処理
	// 入力と出力が同じディレクトリの場合のみ処理（冪等性確保のため）
	inputDir := req.InputPath
	if !stat.IsDir() {
		inputDir = filepath.Dir(req.InputPath)
	}
	sameDirectory := (outputDir == inputDir)
	shouldProcessSourceFiles := !req.DryRun && len(sourceFiles) > 0 && sameDirectory

	if shouldProcessSourceFiles {
		if req.Backup {
			// バックアップディレクトリにファイルを移動
			if err := uc.backupSourceFiles(sourceFiles, outputDir); err != nil {
				return nil, fmt.Errorf("failed to backup source files: %w", err)
			}
		} else {
			// デフォルトでソースファイルを削除（冪等性のため）
			if err := uc.removeSourceFiles(sourceFiles); err != nil {
				return nil, fmt.Errorf("failed to remove source files: %w", err)
			}
		}
	}

	// 結果の表示
	if req.DryRun {
		if sameDirectory {
			if req.Backup {
				fmt.Println("Dry run completed. Use --dry-run=false to actually create files and backup source files.")
			} else {
				fmt.Println("Dry run completed. Use --dry-run=false to actually create files and remove source files.")
			}
		} else {
			fmt.Println("Dry run completed. Use --dry-run=false to actually create files.")
		}
	} else {
		if shouldProcessSourceFiles {
			if req.Backup {
				fmt.Printf("Successfully organized Terraform files into: %s (backed up %d source files)\n", outputDir, len(sourceFiles))
			} else {
				fmt.Printf("Successfully organized Terraform files into: %s (removed %d source files)\n", outputDir, len(sourceFiles))
			}
		} else {
			fmt.Printf("Successfully organized Terraform files into: %s\n", outputDir)
		}
	}

	return &OrganizeFilesResponse{
		ProcessedFiles: fileCount,
		TotalBlocks:    len(allBlocks),
		FileGroups:     len(groups),
		OutputDir:      outputDir,
		WasDryRun:      req.DryRun,
	}, nil
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

		if !info.IsDir() && strings.HasSuffix(path, ".tf") && !uc.isGeneratedFile(filepath.Base(path), filepath.Dir(path)) {
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

		// .tfファイルのみ処理（生成済みファイルはスキップ）
		if !strings.HasSuffix(entry.Name(), ".tf") || uc.isGeneratedFile(entry.Name(), dirPath) {
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

// isGeneratedFile は生成済みファイルかどうかを判定
func (uc *OrganizeFilesUsecase) isGeneratedFile(filename, dirPath string) bool {
	// terraform-file-organizeで生成されるファイル名パターンをチェック
	generatedPrefixes := []string{
		"data__",
		"resource__",
		"module__",
	}

	// プレフィックスマッチングによる判定（常に生成済み扱い）
	for _, prefix := range generatedPrefixes {
		if strings.HasPrefix(filename, prefix) {
			return true
		}
	}

	// 標準ファイル名（locals.tf, variables.tf等）は、ディレクトリに生成済みファイルが複数存在する場合のみ判定
	standardNames := []string{
		"locals.tf",
		"outputs.tf",
		"providers.tf",
		"terraform.tf",
		"variables.tf",
	}

	if slices.Contains(standardNames, filename) {
		// ディレクトリ内に他の生成済みファイルが存在するかチェック
		return uc.hasOtherGeneratedFiles(dirPath, filename)
	}

	return false
}

// hasOtherGeneratedFiles は指定したファイル以外に生成済みファイルが存在するかチェック
func (uc *OrganizeFilesUsecase) hasOtherGeneratedFiles(dirPath, excludeFile string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	generatedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") || entry.Name() == excludeFile {
			continue
		}

		// data__, resource__, module__ プレフィックスのファイルが存在するかチェック
		if strings.HasPrefix(entry.Name(), "data__") ||
			strings.HasPrefix(entry.Name(), "resource__") ||
			strings.HasPrefix(entry.Name(), "module__") {
			generatedCount++
		}
	}

	// 生成済みファイルが2個以上存在する場合、分割済みディレクトリと判定
	return generatedCount >= 2
}
