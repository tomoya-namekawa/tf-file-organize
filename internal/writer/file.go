package writer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/namekawa/terraform-file-organize/pkg/types"
)

type Writer struct {
	outputDir string
	dryRun    bool
}

func New(outputDir string, dryRun bool) *Writer {
	return &Writer{
		outputDir: outputDir,
		dryRun:    dryRun,
	}
}

func (w *Writer) WriteGroups(groups []*types.BlockGroup) error {
	if !w.dryRun {
		if err := os.MkdirAll(w.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	for _, group := range groups {
		if err := w.writeGroup(group); err != nil {
			return fmt.Errorf("failed to write group %s: %w", group.FileName, err)
		}
	}

	return nil
}

func (w *Writer) writeGroup(group *types.BlockGroup) error {
	filepath := filepath.Join(w.outputDir, group.FileName)
	
	if w.dryRun {
		fmt.Printf("Would create file: %s\n", filepath)
		fmt.Printf("  Block type: %s\n", group.BlockType)
		if group.SubType != "" {
			fmt.Printf("  Sub type: %s\n", group.SubType)
		}
		fmt.Printf("  Number of blocks: %d\n", len(group.Blocks))
		fmt.Println()
		return nil
	}

	file := hclwrite.NewEmptyFile()
	rootBody := file.Body()

	for i, block := range group.Blocks {
		if i > 0 {
			rootBody.AppendNewline()
		}

		newBlock := rootBody.AppendNewBlock(block.Type, block.Labels)
		
		if err := w.copyBlockBody(block.Body, newBlock.Body()); err != nil {
			return fmt.Errorf("failed to copy block body: %w", err)
		}
	}

	content := file.Bytes()
	
	if err := os.WriteFile(filepath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath, err)
	}

	fmt.Printf("Created file: %s\n", filepath)
	return nil
}

func (w *Writer) copyBlockBody(sourceBody hcl.Body, targetBody *hclwrite.Body) error {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "required_version"},
			{Name: "region"},
			{Name: "description"},
			{Name: "type"},
			{Name: "default"},
			{Name: "ami"},
			{Name: "instance_type"},
			{Name: "key_name"},
			{Name: "name_prefix"},
			{Name: "from_port"},
			{Name: "to_port"},
			{Name: "protocol"},
			{Name: "cidr_blocks"},
			{Name: "vpc_security_group_ids"},
			{Name: "tags"},
			{Name: "source"},
			{Name: "name"},
			{Name: "cidr"},
			{Name: "azs"},
			{Name: "private_subnets"},
			{Name: "public_subnets"},
			{Name: "enable_nat_gateway"},
			{Name: "enable_vpn_gateway"},
			{Name: "value"},
			{Name: "most_recent"},
			{Name: "owners"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "required_providers"},
			{Type: "ingress"},
			{Type: "egress"},
			{Type: "filter"},
			{Type: "lifecycle"},
			{Type: "provisioner", LabelNames: []string{"type"}},
			{Type: "connection"},
			{Type: "dynamic", LabelNames: []string{"for_each"}},
		},
	}
	
	content, remain, diags := sourceBody.PartialContent(schema)
	if diags.HasErrors() {
		return fmt.Errorf("failed to get content: %s", diags.Error())
	}

	for name, attr := range content.Attributes {
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			continue
		}
		targetBody.SetAttributeValue(name, value)
	}

	for _, block := range content.Blocks {
		nestedBlock := targetBody.AppendNewBlock(block.Type, block.Labels)
		if err := w.copyBlockBody(block.Body, nestedBlock.Body()); err != nil {
			return fmt.Errorf("failed to copy nested block: %w", err)
		}
	}

	remainAttrs, diags := remain.JustAttributes()
	if !diags.HasErrors() {
		for name, attr := range remainAttrs {
			value, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				continue
			}
			targetBody.SetAttributeValue(name, value)
		}
	}

	return nil
}