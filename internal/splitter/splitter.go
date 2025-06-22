// Package splitter groups Terraform blocks by type with custom rules.
package splitter

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

const (
	blockTypeResource  = "resource"
	blockTypeData      = "data"
	blockTypeModule    = "module"
	blockTypeProvider  = "provider"
	blockTypeVariable  = "variable"
	blockTypeOutput    = "output"
	blockTypeLocals    = "locals"
	blockTypeTerraform = "terraform"

	defaultResourceFile  = "resource.tf"
	defaultDataFile      = "data.tf"
	defaultModuleFile    = "module.tf"
	defaultOutputsFile   = "outputs.tf"
	defaultVariablesFile = "variables.tf"
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
		s.sortBlocksInGroup(group)
		result = append(result, group)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].FileName < result[j].FileName
	})

	return result
}

func (s *Splitter) getGroupKeyAndFilename(block *types.Block) (groupKey, filename string) {
	resourceType := s.getSubType(block)

	candidates := s.getMatchCandidates(block, resourceType)

	if s.config != nil {
		for _, candidate := range candidates {
			if group := s.config.FindGroupForResource(candidate); group != nil {
				if s.config.IsFileExcluded(group.Filename) {
					key := s.getDefaultGroupKey(block)
					fname := s.getExcludedFileName(block)
					return key, fname
				}
				return group.Name, group.Filename
			}
		}
	}

	groupKey = s.getDefaultGroupKey(block)
	filename = s.getDefaultFileName(block)
	return
}

// getMatchCandidates generates matching candidates for blocks in priority order:
// 1. block_type.sub_type.name 2. block_type.sub_type 3. sub_type 4. block_type
func (s *Splitter) getMatchCandidates(block *types.Block, resourceType string) []string {
	var candidates []string

	var blockName string
	if len(block.Labels) > 1 {
		blockName = block.Labels[1]
	}

	if resourceType != "" && blockName != "" {
		candidates = append(candidates, fmt.Sprintf("%s.%s.%s", block.Type, resourceType, blockName))
	}

	if resourceType != "" {
		candidates = append(candidates, fmt.Sprintf("%s.%s", block.Type, resourceType))
	}

	if resourceType != "" {
		candidates = append(candidates, resourceType)
	}

	candidates = append(candidates, block.Type)

	return candidates
}

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

	cleaned := s.cleanUnsafeCharacters(name)

	cleaned = s.applyLengthLimits(cleaned)

	cleaned = s.validateReservedNames(cleaned)

	return cleaned
}

// cleanUnsafeCharacters removes dangerous characters
func (s *Splitter) cleanUnsafeCharacters(name string) string {
	cleaned := filepath.Clean(name)

	cleaned = filepath.Base(cleaned)

	replacer := strings.NewReplacer(
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
		"\x00", "_",
	)
	cleaned = replacer.Replace(cleaned)

	var result strings.Builder
	for _, r := range cleaned {
		if r >= 32 && r <= 126 {
			result.WriteRune(r)
		} else {
			result.WriteString("_")
		}
	}

	cleaned = result.String()

	for strings.Contains(cleaned, "__") {
		cleaned = strings.ReplaceAll(cleaned, "__", "_")
	}

	cleaned = strings.Trim(cleaned, "_")

	return cleaned
}

func (s *Splitter) applyLengthLimits(cleaned string) string {
	const maxLength = 200
	if len(cleaned) > maxLength {
		cleaned = cleaned[:maxLength]
		cleaned = strings.TrimSuffix(cleaned, "_")
	}

	if cleaned == "" {
		cleaned = "unnamed"
	}

	return cleaned
}

func (s *Splitter) validateReservedNames(cleaned string) string {
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

func (s *Splitter) sortBlocksInGroup(group *types.BlockGroup) {
	sort.Slice(group.Blocks, func(i, j int) bool {
		return s.getBlockSortKey(group.Blocks[i]) < s.getBlockSortKey(group.Blocks[j])
	})
}

func (s *Splitter) getBlockSortKey(block *types.Block) string {
	key := block.Type
	for _, label := range block.Labels {
		key += "_" + label
	}
	return key
}
