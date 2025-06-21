package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/config"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/parser"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/splitter"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/writer"
	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
)

// OrganizeFilesRequest は OrganizeFiles ユースケースのリクエスト
type OrganizeFilesRequest struct {
	InputPath  string
	OutputDir  string
	ConfigFile string
	DryRun     bool
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
type OrganizeFilesUsecase struct{}

// NewOrganizeFilesUsecase は新しい OrganizeFilesUsecase を作成
func NewOrganizeFilesUsecase() *OrganizeFilesUsecase {
	return &OrganizeFilesUsecase{}
}

// Execute は Terraform ファイルの整理を実行
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
	cfg, err := uc.loadConfig(req.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// ファイルの解析
	allBlocks, fileCount, err := uc.parseInput(req.InputPath, stat)
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
	combinedFile := &types.ParsedFile{Blocks: allBlocks}
	s := splitter.NewWithConfig(cfg)
	groups := s.GroupBlocks(combinedFile)

	fmt.Printf("Organized into %d file groups\n", len(groups))

	// ファイルの書き出し
	w := writer.New(outputDir, req.DryRun)
	if err := w.WriteGroups(groups); err != nil {
		return nil, fmt.Errorf("failed to write files: %w", err)
	}

	// 結果の表示
	if req.DryRun {
		fmt.Println("Dry run completed. Use --dry-run=false to actually create files.")
	} else {
		fmt.Printf("Successfully organized Terraform files into: %s\n", outputDir)
	}

	return &OrganizeFilesResponse{
		ProcessedFiles: fileCount,
		TotalBlocks:    len(allBlocks),
		FileGroups:     len(groups),
		OutputDir:      outputDir,
		WasDryRun:      req.DryRun,
	}, nil
}

// loadConfig は設定ファイルを読み込む
func (uc *OrganizeFilesUsecase) loadConfig(configFile string) (*config.Config, error) {
	if configFile != "" {
		fmt.Printf("Loading configuration from: %s\n", configFile)
		return config.LoadConfig(configFile)
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

// parseInput は入力パス（ファイルまたはディレクトリ）を解析
func (uc *OrganizeFilesUsecase) parseInput(inputPath string, stat os.FileInfo) ([]*types.Block, int, error) {
	var allBlocks []*types.Block
	var fileCount int

	if stat.IsDir() {
		fmt.Printf("Scanning directory for Terraform files: %s\n", inputPath)
		blocks, count, err := uc.parseDirectory(inputPath)
		if err != nil {
			return nil, 0, err
		}
		allBlocks = blocks
		fileCount = count
		fmt.Printf("Found %d .tf files with %d total blocks\n", fileCount, len(allBlocks))
	} else {
		fmt.Printf("Parsing Terraform file: %s\n", inputPath)
		p := parser.New()
		parsedFile, err := p.ParseFile(inputPath)
		if err != nil {
			return nil, 0, err
		}
		allBlocks = parsedFile.Blocks
		fileCount = 1
		fmt.Printf("Found %d blocks\n", len(allBlocks))
	}

	return allBlocks, fileCount, nil
}

// parseDirectory はディレクトリ内の.tfファイルを再帰的に解析
func (uc *OrganizeFilesUsecase) parseDirectory(dirPath string) ([]*types.Block, int, error) {
	var allBlocks []*types.Block
	fileCount := 0

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// シンボリックリンクをスキップ（セキュリティ上の理由）
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: skipping symbolic link: %s\n", path)
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tf") {
			// パスの安全性を確認
			if err := uc.validatePath(path); err != nil {
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