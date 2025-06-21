package writer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
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
	// まずschemaなしですべての内容を取得
	content, diags := sourceBody.Content(&hcl.BodySchema{})
	if diags.HasErrors() {
		// fallbackとして、より寛容な方法を試す
		return w.copyBlockBodyFallback(sourceBody, targetBody)
	}

	// 属性を取得
	attrs, attrDiags := sourceBody.JustAttributes()
	if !attrDiags.HasErrors() {
		for name, attr := range attrs {
			value, valueDiags := attr.Expr.Value(nil)
			if valueDiags.HasErrors() {
				// 評価できない場合は文字列でfallback
				continue
			}
			targetBody.SetAttributeValue(name, value)
		}
	}

	// 既知のブロックタイプを処理
	blocks := content.Blocks
	for _, block := range blocks {
		nestedBlock := targetBody.AppendNewBlock(block.Type, block.Labels)
		if err := w.copyBlockBody(block.Body, nestedBlock.Body()); err != nil {
			return fmt.Errorf("failed to copy nested block: %w", err)
		}
	}

	return nil
}

// fallback method for problematic cases
func (w *Writer) copyBlockBodyFallback(sourceBody hcl.Body, targetBody *hclwrite.Body) error {
	// より包括的なschemaを定義
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			// 基本属性
			{Name: "ami"}, {Name: "instance_type"}, {Name: "key_name"},
			{Name: "vpc_security_group_ids"}, {Name: "tags"}, {Name: "subnet_id"},
			{Name: "name_prefix"}, {Name: "description"}, {Name: "type"}, {Name: "default"},
			{Name: "region"}, {Name: "source"}, {Name: "name"}, {Name: "value"},
			{Name: "most_recent"}, {Name: "owners"}, {Name: "cidr_block"}, {Name: "vpc_id"},
			{Name: "cidr"}, {Name: "azs"}, {Name: "private_subnets"}, {Name: "public_subnets"},
			{Name: "enable_nat_gateway"}, {Name: "enable_vpn_gateway"},
			// random_string関連
			{Name: "length"}, {Name: "special"}, {Name: "upper"}, {Name: "lower"},
			// ロードバランサー関連
			{Name: "internal"}, {Name: "load_balancer_type"}, {Name: "security_groups"}, {Name: "subnets"},
			// データベース関連
			{Name: "identifier"}, {Name: "engine"}, {Name: "engine_version"}, {Name: "instance_class"},
			{Name: "allocated_storage"}, {Name: "storage_type"}, {Name: "db_name"}, {Name: "username"}, {Name: "password"},
			{Name: "db_subnet_group_name"}, {Name: "skip_final_snapshot"}, {Name: "subnet_ids"},
			// S3関連
			{Name: "bucket"}, {Name: "status"},
			// セキュリティグループ関連
			{Name: "from_port"}, {Name: "to_port"}, {Name: "protocol"}, {Name: "cidr_blocks"}, {Name: "gateway_id"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "required_providers"}, {Type: "ingress"}, {Type: "egress"}, {Type: "filter"},
			{Type: "lifecycle"}, {Type: "provisioner", LabelNames: []string{"type"}},
			{Type: "connection"}, {Type: "dynamic", LabelNames: []string{"for_each"}},
			{Type: "route"}, {Type: "versioning_configuration"},
		},
	}

	content, _, diags := sourceBody.PartialContent(schema)
	if diags.HasErrors() {
		return fmt.Errorf("failed to get content with fallback: %s", diags.Error())
	}

	// 属性をコピー
	for name, attr := range content.Attributes {
		value, valueDiags := attr.Expr.Value(nil)
		if valueDiags.HasErrors() {
			continue
		}
		targetBody.SetAttributeValue(name, value)
	}

	// ブロックをコピー
	for _, block := range content.Blocks {
		nestedBlock := targetBody.AppendNewBlock(block.Type, block.Labels)
		if err := w.copyBlockBody(block.Body, nestedBlock.Body()); err != nil {
			return fmt.Errorf("failed to copy nested block: %w", err)
		}
	}

	return nil
}