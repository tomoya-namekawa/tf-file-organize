package parser

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
)

type Parser struct {
	parser *hclparse.Parser
}

func New() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

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
				Type:      block.Type,
				Labels:    block.Labels,
				Body:      block.Body,
				DefRange:  block.DefRange,
				TypeRange: block.TypeRange,
				RawBody:   "",
			}
			parsedFile.Blocks = append(parsedFile.Blocks, parsedBlock)
		}
	} else {
		// Syntaxブロックから詳細情報を抽出
		for i, block := range content_hcl.Blocks {
			var rawBody string
			if i < len(syntaxFile.Body.(*hclsyntax.Body).Blocks) {
				syntaxBlock := syntaxFile.Body.(*hclsyntax.Body).Blocks[i]
				rawBody = p.extractRawBodyFromSyntax(content, syntaxBlock)
			}

			parsedBlock := &types.Block{
				Type:      block.Type,
				Labels:    block.Labels,
				Body:      block.Body,
				DefRange:  block.DefRange,
				TypeRange: block.TypeRange,
				RawBody:   rawBody,
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
