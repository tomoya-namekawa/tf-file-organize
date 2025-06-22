// Package splitter provides functionality to group Terraform blocks by type
// and apply custom grouping rules based on configuration.
package splitter

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

// Terraform block type constants
const (
	blockTypeResource  = "resource"
	blockTypeData      = "data"
	blockTypeModule    = "module"
	blockTypeProvider  = "provider"
	blockTypeVariable  = "variable"
	blockTypeOutput    = "output"
	blockTypeLocals    = "locals"
	blockTypeTerraform = "terraform"

	// Default file names
	defaultResourceFile  = "resource.tf"
	defaultDataFile      = "data.tf"
	defaultModuleFile    = "module.tf"
	defaultOutputsFile   = "outputs.tf"
	defaultVariablesFile = "variables.tf"
)

// Splitter groups Terraform blocks according to configuration rules.
type Splitter struct {
	config *config.Config
}

// New creates a new Splitter with default configuration.
func New() *Splitter {
	return &Splitter{
		config: &config.Config{},
	}
}

// NewWithConfig creates a new Splitter with the provided configuration.
func NewWithConfig(cfg *config.Config) *Splitter {
	return &Splitter{
		config: cfg,
	}
}

// GroupBlocks groups the parsed blocks according to configuration rules and returns block groups.
func (s *Splitter) GroupBlocks(parsedFile *types.ParsedFile) []*types.BlockGroup {
	groups := make(map[string]*types.BlockGroup)

	for _, block := range parsedFile.Blocks {
		key, filename := s.getGroupKeyAndFilename(block)

		if group, exists := groups[key]; exists {
			group.Blocks = append(group.Blocks, block)
		} else {
			groups[key] = &types.BlockGroup{
				BlockType: block.Type,
				SubType:   s.getSubType(block),
				Blocks:    []*types.Block{block},
				FileName:  filename,
			}
		}
	}

	result := make([]*types.BlockGroup, 0, len(groups))
	for _, group := range groups {
		// グループ内のブロックをアルファベット順でソート
		s.sortBlocksInGroup(group)
		result = append(result, group)
	}

	// グループ自体もファイル名でソート
	sort.Slice(result, func(i, j int) bool {
		return result[i].FileName < result[j].FileName
	})

	return result
}

func (s *Splitter) getGroupKeyAndFilename(block *types.Block) (groupKey, filename string) {
	resourceType := s.getSubType(block)

	// パターンマッチング用の候補文字列を作成
	candidates := s.getMatchCandidates(block, resourceType)

	// 設定ファイルでのグループ化チェック
	if s.config != nil {
		for _, candidate := range candidates {
			if group := s.config.FindGroupForResource(candidate); group != nil {
				// ファイル除外チェック
				if s.config.IsFileExcluded(group.Filename) {
					// 除外対象は個別ファイルにする
					key := s.getDefaultGroupKey(block)
					fname := s.getExcludedFileName(block)
					return key, fname
				}
				return group.Name, group.Filename
			}
		}
	}

	// デフォルトの動作
	groupKey = s.getDefaultGroupKey(block)
	filename = s.getDefaultFileName(block)
	return
}

// getMatchCandidates はブロックに対するマッチング候補を生成
// 優先度順に以下のパターンを生成：
// 1. block_type.sub_type.name (例: output.instance_ip.web)
// 2. block_type.sub_type (例: resource.aws_instance)
// 3. sub_type (例: aws_instance)
// 4. block_type (例: resource)
func (s *Splitter) getMatchCandidates(block *types.Block, resourceType string) []string {
	var candidates []string

	// ブロック名（第2ラベル）を取得
	var blockName string
	if len(block.Labels) > 1 {
		blockName = block.Labels[1]
	}

	// 1. block_type.sub_type.name パターン
	if resourceType != "" && blockName != "" {
		candidates = append(candidates, fmt.Sprintf("%s.%s.%s", block.Type, resourceType, blockName))
	}

	// 2. block_type.sub_type パターン
	if resourceType != "" {
		candidates = append(candidates, fmt.Sprintf("%s.%s", block.Type, resourceType))
	}

	// 3. sub_type パターン
	if resourceType != "" {
		candidates = append(candidates, resourceType)
	}

	// 4. block_type パターン
	candidates = append(candidates, block.Type)

	return candidates
}

// getExcludedFileName は除外されたブロックの個別ファイル名を生成
func (s *Splitter) getExcludedFileName(block *types.Block) string {
	switch block.Type {
	case blockTypeResource:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("resource__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultResourceFile
	case blockTypeData:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("data__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultDataFile
	case blockTypeModule:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("module__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultModuleFile
	case blockTypeOutput:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("output__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultOutputsFile
	case blockTypeVariable:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("variable__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultVariablesFile
	default:
		// その他のブロックタイプはデフォルトファイル名
		return s.getDefaultFileName(block)
	}
}

func (s *Splitter) getDefaultGroupKey(block *types.Block) string {
	switch block.Type {
	case blockTypeResource, blockTypeData:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("%s_%s", block.Type, block.Labels[0])
		}
		return block.Type
	case blockTypeModule:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("%s_%s", block.Type, block.Labels[0])
		}
		return block.Type
	case blockTypeProvider:
		// すべてのproviderを同じグループにまとめる
		return "providers"
	case blockTypeVariable:
		return "variables"
	case blockTypeOutput:
		return "outputs"
	case blockTypeLocals:
		return blockTypeLocals
	case blockTypeTerraform:
		return blockTypeTerraform
	default:
		return block.Type
	}
}

func (s *Splitter) getSubType(block *types.Block) string {
	switch block.Type {
	case blockTypeResource, blockTypeData:
		if len(block.Labels) > 0 {
			return block.Labels[0]
		}
	case blockTypeModule:
		if len(block.Labels) > 0 {
			return block.Labels[0]
		}
	case blockTypeProvider:
		if len(block.Labels) > 0 {
			return block.Labels[0]
		}
	case blockTypeOutput, blockTypeVariable:
		// outputやvariableブロックの場合、第1ラベルがname
		if len(block.Labels) > 0 {
			return block.Labels[0]
		}
	}
	return ""
}

func (s *Splitter) getDefaultFileName(block *types.Block) string {
	switch block.Type {
	case blockTypeResource:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("resource__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultResourceFile
	case blockTypeData:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("data__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultDataFile
	case blockTypeModule:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("module__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return defaultModuleFile
	case blockTypeProvider:
		return "providers.tf"
	case blockTypeVariable:
		return defaultVariablesFile
	case blockTypeOutput:
		return defaultOutputsFile
	case blockTypeLocals:
		return "locals.tf"
	case blockTypeTerraform:
		return "terraform.tf"
	default:
		return fmt.Sprintf("%s.tf", s.sanitizeFileName(block.Type))
	}
}

const unnamedFile = "unnamed"

func (s *Splitter) sanitizeFileName(name string) string {
	if name == "" {
		return unnamedFile
	}

	// セキュリティクリーニング
	cleaned := s.cleanUnsafeCharacters(name)

	// 長さ制限とフォーマット正規化
	cleaned = s.applyLengthLimits(cleaned)

	// Windows予約名検証
	cleaned = s.validateReservedNames(cleaned)

	return cleaned
}

// cleanUnsafeCharacters removes dangerous characters and path traversal elements
func (s *Splitter) cleanUnsafeCharacters(name string) string {
	// filepath.Cleanを使用してパストラバーサルを防ぐ
	cleaned := filepath.Clean(name)

	// filepath.Baseを使用してディレクトリ区切り文字を除去
	cleaned = filepath.Base(cleaned)

	// 残りの危険な文字を置換
	replacer := strings.NewReplacer(
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
		"\x00", "_", // ヌル文字
	)
	cleaned = replacer.Replace(cleaned)

	// 制御文字を除去
	var result strings.Builder
	for _, r := range cleaned {
		if r >= 32 && r <= 126 { // 印刷可能ASCII文字のみ
			result.WriteRune(r)
		} else {
			result.WriteString("_")
		}
	}

	cleaned = result.String()

	// 連続するアンダースコアを単一に
	for strings.Contains(cleaned, "__") {
		cleaned = strings.ReplaceAll(cleaned, "__", "_")
	}

	// 先頭・末尾のアンダースコアを除去
	cleaned = strings.Trim(cleaned, "_")

	return cleaned
}

// applyLengthLimits applies length restrictions and handles empty results
func (s *Splitter) applyLengthLimits(cleaned string) string {
	// 長さ制限（Windows互換性のため）
	const maxLength = 200 // .tfを考慮して200文字
	if len(cleaned) > maxLength {
		cleaned = cleaned[:maxLength]
		cleaned = strings.TrimSuffix(cleaned, "_")
	}

	// 空になった場合のフォールバック
	if cleaned == "" {
		cleaned = "unnamed"
	}

	return cleaned
}

// validateReservedNames checks and handles Windows reserved names
func (s *Splitter) validateReservedNames(cleaned string) string {
	// Windowsの予約名チェック
	reservedNames := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true,
		"COM5": true, "COM6": true, "COM7": true, "COM8": true,
		"COM9": true, "LPT1": true, "LPT2": true, "LPT3": true,
		"LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true,
		"LPT8": true, "LPT9": true,
	}

	if reservedNames[strings.ToUpper(cleaned)] {
		cleaned = "tf_" + cleaned
	}

	return cleaned
}

// sortBlocksInGroup はグループ内のブロックをアルファベット順でソート
func (s *Splitter) sortBlocksInGroup(group *types.BlockGroup) {
	sort.Slice(group.Blocks, func(i, j int) bool {
		return s.getBlockSortKey(group.Blocks[i]) < s.getBlockSortKey(group.Blocks[j])
	})
}

// getBlockSortKey はブロックのソートキーを生成
func (s *Splitter) getBlockSortKey(block *types.Block) string {
	// ブロックタイプ + ラベルでソートキーを作成
	key := block.Type
	for _, label := range block.Labels {
		key += "_" + label
	}
	return key
}
