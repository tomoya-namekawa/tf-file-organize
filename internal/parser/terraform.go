package parser

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
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
	content, err := os.ReadFile(filename)
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

	for _, block := range content_hcl.Blocks {
		parsedBlock := &types.Block{
			Type:      block.Type,
			Labels:    block.Labels,
			Body:      block.Body,
			DefRange:  block.DefRange,
			TypeRange: block.TypeRange,
		}
		parsedFile.Blocks = append(parsedFile.Blocks, parsedBlock)
	}

	return parsedFile, nil
}