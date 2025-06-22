package splitter

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/config"
	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
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
)

type Splitter struct {
	config *config.Config
}

func New() *Splitter {
	return &Splitter{
		config: &config.Config{},
	}
}

func NewWithConfig(cfg *config.Config) *Splitter {
	return &Splitter{
		config: cfg,
	}
}

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

	// 設定ファイルでの除外チェック
	if s.config != nil && resourceType != "" && s.config.IsExcluded(resourceType) {
		// 除外対象は個別ファイルにする
		key := s.getDefaultGroupKey(block)
		fname := s.getDefaultFileName(block)
		return key, fname
	}

	// 設定ファイルでのグループ化チェック
	if s.config != nil && resourceType != "" {
		if group := s.config.FindGroupForResource(resourceType); group != nil {
			return group.Name, group.Filename
		}
	}

	// オーバーライド設定のチェック
	if s.config != nil {
		if overrideFilename := s.config.GetOverrideFilename(block.Type); overrideFilename != "" {
			key := s.getDefaultGroupKey(block)
			return key, overrideFilename
		}
	}

	// デフォルトの動作
	groupKey = s.getDefaultGroupKey(block)
	filename = s.getDefaultFileName(block)
	return
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
	}
	return ""
}

func (s *Splitter) getDefaultFileName(block *types.Block) string {
	switch block.Type {
	case blockTypeResource:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("resource__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return "resource.tf"
	case blockTypeData:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("data__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return "data.tf"
	case blockTypeModule:
		if len(block.Labels) > 0 {
			return fmt.Sprintf("module__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return "module.tf"
	case blockTypeProvider:
		return "providers.tf"
	case blockTypeVariable:
		return "variables.tf"
	case blockTypeOutput:
		return "outputs.tf"
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
