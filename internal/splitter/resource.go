package splitter

import (
	"fmt"
	"strings"

	"github.com/namekawa/terraform-file-organize/internal/config"
	"github.com/namekawa/terraform-file-organize/pkg/types"
)

type Splitter struct{
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
		result = append(result, group)
	}
	
	return result
}

func (s *Splitter) getGroupKeyAndFilename(block *types.Block) (string, string) {
	resourceType := s.getSubType(block)
	
	// 設定ファイルでの除外チェック
	if s.config != nil && resourceType != "" && s.config.IsExcluded(resourceType) {
		// 除外対象は個別ファイルにする
		key := s.getDefaultGroupKey(block)
		filename := s.getDefaultFileName(block)
		return key, filename
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
	key := s.getDefaultGroupKey(block)
	filename := s.getDefaultFileName(block)
	return key, filename
}

func (s *Splitter) getDefaultGroupKey(block *types.Block) string {
	switch block.Type {
	case "resource", "data":
		if len(block.Labels) > 0 {
			return fmt.Sprintf("%s_%s", block.Type, block.Labels[0])
		}
		return block.Type
	case "module":
		if len(block.Labels) > 0 {
			return fmt.Sprintf("%s_%s", block.Type, block.Labels[0])
		}
		return block.Type
	case "provider":
		if len(block.Labels) > 0 {
			return fmt.Sprintf("%s_%s", block.Type, block.Labels[0])
		}
		return "providers"
	case "variable":
		return "variables"
	case "output":
		return "outputs"
	case "locals":
		return "locals"
	case "terraform":
		return "terraform"
	default:
		return block.Type
	}
}

func (s *Splitter) getSubType(block *types.Block) string {
	switch block.Type {
	case "resource", "data":
		if len(block.Labels) > 0 {
			return block.Labels[0]
		}
	case "module":
		if len(block.Labels) > 0 {
			return block.Labels[0]
		}
	case "provider":
		if len(block.Labels) > 0 {
			return block.Labels[0]
		}
	}
	return ""
}

func (s *Splitter) getDefaultFileName(block *types.Block) string {
	switch block.Type {
	case "resource":
		if len(block.Labels) > 0 {
			return fmt.Sprintf("resource__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return "resource.tf"
	case "data":
		if len(block.Labels) > 0 {
			return fmt.Sprintf("data__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return "data.tf"
	case "module":
		if len(block.Labels) > 0 {
			return fmt.Sprintf("module__%s.tf", s.sanitizeFileName(block.Labels[0]))
		}
		return "module.tf"
	case "provider":
		return "providers.tf"
	case "variable":
		return "variables.tf"
	case "output":
		return "outputs.tf"
	case "locals":
		return "locals.tf"
	case "terraform":
		return "terraform.tf"
	default:
		return fmt.Sprintf("%s.tf", s.sanitizeFileName(block.Type))
	}
}

func (s *Splitter) sanitizeFileName(name string) string {
	if name == "" {
		return "unnamed"
	}
	
	// まず危険な文字を置換
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
		"..", "_", // パストラバーサル対策
		"\x00", "_", // ヌル文字
	)
	sanitized := replacer.Replace(name)
	
	// 制御文字を除去
	var result strings.Builder
	for _, r := range sanitized {
		if r >= 32 && r <= 126 { // 印刷可能ASCII文字のみ
			result.WriteRune(r)
		} else {
			result.WriteString("_")
		}
	}
	
	cleaned := result.String()
	
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