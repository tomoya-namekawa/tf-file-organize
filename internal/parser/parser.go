// Package parser provides functionality to parse Terraform configuration files
// and extract blocks with comment preservation.
package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
)

// Parser handles parsing of Terraform configuration files using HCL.
type Parser struct {
	parser *hclparse.Parser
}

// New creates a new Parser instance.
func New() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

// ParseFile parses a Terraform file and extracts all blocks with comment preservation.
func (p *Parser) ParseFile(filename string) (*types.ParsedFile, error) {
	content, err := os.ReadFile(filename) //nolint:gosec // filename is validated before use
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	file, diags := p.parser.ParseHCL(content, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	parsedFile := &types.ParsedFile{
		Blocks: make([]*types.Block, 0),
	}

	if file.Body == nil {
		return parsedFile, nil
	}

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "terraform"},
			{Type: "provider", LabelNames: []string{"name"}},
			{Type: "variable", LabelNames: []string{"name"}},
			{Type: "locals"},
			{Type: "data", LabelNames: []string{"type", "name"}},
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "module", LabelNames: []string{"name"}},
			{Type: "output", LabelNames: []string{"name"}},
		},
	}

	content_hcl, _, diags := file.Body.PartialContent(schema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to extract content: %s", diags.Error())
	}

	// HCL Syntaxファイルからもブロック情報を取得
	syntaxFile, syntaxDiags := hclsyntax.ParseConfig(content, filename, hcl.InitialPos)
	if syntaxDiags.HasErrors() {
		// Syntaxパースに失敗した場合は通常のパースで続行
		for _, block := range content_hcl.Blocks {
			parsedBlock := &types.Block{
				Type:            block.Type,
				Labels:          block.Labels,
				Body:            block.Body,
				DefRange:        block.DefRange,
				TypeRange:       block.TypeRange,
				RawBody:         "",
				LeadingComments: "",
			}
			parsedFile.Blocks = append(parsedFile.Blocks, parsedBlock)
		}
	} else {
		// Syntaxブロックから詳細情報を抽出
		for i, block := range content_hcl.Blocks {
			var rawBody, leadingComments string
			if i < len(syntaxFile.Body.(*hclsyntax.Body).Blocks) {
				syntaxBlock := syntaxFile.Body.(*hclsyntax.Body).Blocks[i]
				rawBody = p.extractRawBodyFromSyntax(content, syntaxBlock)
				leadingComments = p.extractLeadingComments(content, syntaxBlock, i, syntaxFile.Body.(*hclsyntax.Body).Blocks)
			}

			parsedBlock := &types.Block{
				Type:            block.Type,
				Labels:          block.Labels,
				Body:            block.Body,
				DefRange:        block.DefRange,
				TypeRange:       block.TypeRange,
				RawBody:         rawBody,
				LeadingComments: leadingComments,
			}
			parsedFile.Blocks = append(parsedFile.Blocks, parsedBlock)
		}
	}

	return parsedFile, nil
}

// extractRawBodyFromSyntax はSyntaxブロックから生ソースコードを抽出
func (p *Parser) extractRawBodyFromSyntax(content []byte, syntaxBlock *hclsyntax.Block) string {
	// OpenBraceRangeとCloseBraceRangeを使用してブロック本体を抽出
	openBraceRange := syntaxBlock.OpenBraceRange
	closeBraceRange := syntaxBlock.CloseBraceRange

	if openBraceRange.End.Byte >= len(content) || closeBraceRange.Start.Byte > len(content) {
		return ""
	}

	// '{' の後から '}' の前までの内容を抽出
	startByte := openBraceRange.End.Byte
	endByte := closeBraceRange.Start.Byte

	if startByte < endByte {
		rawContent := content[startByte:endByte]
		return string(rawContent)
	}

	return ""
}

// extractLeadingComments はブロックの前にあるコメントを抽出
func (p *Parser) extractLeadingComments(content []byte, currentBlock *hclsyntax.Block, blockIndex int, allBlocks []*hclsyntax.Block) string {
	// 現在のブロックの開始位置
	currentBlockStart := currentBlock.TypeRange.Start.Byte

	// 前のブロックの終了位置を取得
	var searchStartByte int
	if blockIndex == 0 {
		// 最初のブロックの場合は、ファイルの先頭から検索
		searchStartByte = 0
	} else {
		// 前のブロックの終了位置から検索
		prevBlock := allBlocks[blockIndex-1]
		searchStartByte = prevBlock.CloseBraceRange.End.Byte
	}

	// 検索範囲のコンテンツを取得
	if searchStartByte >= len(content) || currentBlockStart > len(content) || searchStartByte >= currentBlockStart {
		return ""
	}

	searchContent := string(content[searchStartByte:currentBlockStart])
	lines := strings.Split(searchContent, "\n")

	var comments []string
	var foundComment bool

	// 後ろから前に向かって処理し、コメント行を特定
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		if line == "" {
			// 空行の場合、コメントが見つかっている場合は続行、そうでなければスキップ
			if foundComment {
				comments = append([]string{""}, comments...)
			}
			continue
		}

		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			// コメント行を発見
			comments = append([]string{line}, comments...)
			foundComment = true
		} else {
			// コメント以外の行が見つかったら、そこで終了
			break
		}
	}

	if len(comments) == 0 {
		return ""
	}

	// コメントが見つかった場合、末尾の空行を削除
	for len(comments) > 0 && comments[len(comments)-1] == "" {
		comments = comments[:len(comments)-1]
	}

	if len(comments) == 0 {
		return ""
	}

	return strings.Join(comments, "\n")
}
