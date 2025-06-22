// Package parser parses Terraform configuration files with comment preservation.
package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

type Parser struct {
	parser *hclparse.Parser
}

func New() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

// ParseFile parses a Terraform file with comment preservation
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
		FileName: filename,
		Blocks:   make([]*types.Block, 0),
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

	syntaxFile, syntaxDiags := hclsyntax.ParseConfig(content, filename, hcl.InitialPos)
	if syntaxDiags.HasErrors() {
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

func (p *Parser) extractRawBodyFromSyntax(content []byte, syntaxBlock *hclsyntax.Block) string {
	openBraceRange := syntaxBlock.OpenBraceRange
	closeBraceRange := syntaxBlock.CloseBraceRange

	if openBraceRange.End.Byte >= len(content) || closeBraceRange.Start.Byte > len(content) {
		return ""
	}

	startByte := openBraceRange.End.Byte
	endByte := closeBraceRange.Start.Byte

	if startByte < endByte {
		rawContent := content[startByte:endByte]
		return string(rawContent)
	}

	return ""
}

func (p *Parser) extractLeadingComments(content []byte, currentBlock *hclsyntax.Block, blockIndex int, allBlocks []*hclsyntax.Block) string {
	currentBlockStart := currentBlock.TypeRange.Start.Byte

	var searchStartByte int
	if blockIndex == 0 {
		searchStartByte = 0
	} else {
		prevBlock := allBlocks[blockIndex-1]
		searchStartByte = prevBlock.CloseBraceRange.End.Byte
	}

	if searchStartByte >= len(content) || currentBlockStart > len(content) || searchStartByte >= currentBlockStart {
		return ""
	}

	searchContent := string(content[searchStartByte:currentBlockStart])
	lines := strings.Split(searchContent, "\n")

	var comments []string
	var foundComment bool

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		if line == "" {
			if foundComment {
				comments = append([]string{""}, comments...)
			}
			continue
		}

		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			comments = append([]string{line}, comments...)
			foundComment = true
		} else {
			break
		}
	}

	if len(comments) == 0 {
		return ""
	}

	for len(comments) > 0 && comments[len(comments)-1] == "" {
		comments = comments[:len(comments)-1]
	}

	if len(comments) == 0 {
		return ""
	}

	return strings.Join(comments, "\n")
}
