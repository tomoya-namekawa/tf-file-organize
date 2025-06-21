package splitter_test

import (
	"testing"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/config"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/splitter"
	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
)

func createTestBlock(blockType string, labels []string) *types.Block {
	return &types.Block{
		Type:   blockType,
		Labels: labels,
	}
}

func TestGroupBlocksDefault(t *testing.T) {
	// デフォルト設定でのテスト
	s := splitter.New()
	
	parsedFile := &types.ParsedFile{
		Blocks: []*types.Block{
			createTestBlock("terraform", nil),
			createTestBlock("provider", []string{"aws"}),
			createTestBlock("variable", []string{"instance_type"}),
			createTestBlock("variable", []string{"key_name"}),
			createTestBlock("locals", nil),
			createTestBlock("resource", []string{"aws_instance", "web"}),
			createTestBlock("resource", []string{"aws_s3_bucket", "data"}),
			createTestBlock("data", []string{"aws_ami", "ubuntu"}),
			createTestBlock("module", []string{"vpc"}),
			createTestBlock("output", []string{"instance_id"}),
		},
	}
	
	groups := s.GroupBlocks(parsedFile)
	
	// グループ数の検証
	expectedGroups := 9 // terraform, providers, variables, locals, resource__aws_instance, resource__aws_s3_bucket, data__aws_ami, module__vpc, outputs
	if len(groups) != expectedGroups {
		t.Errorf("Expected %d groups, got %d", expectedGroups, len(groups))
	}
	
	// 特定のグループの検証
	groupsByFileName := make(map[string]*types.BlockGroup)
	for _, group := range groups {
		groupsByFileName[group.FileName] = group
	}
	
	// variables.tf に2つのvariableブロックが含まれることを確認
	if group, exists := groupsByFileName["variables.tf"]; exists {
		if len(group.Blocks) != 2 {
			t.Errorf("Expected 2 blocks in variables.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("variables.tf group not found")
	}
	
	// resource__aws_instance.tf に1つのリソースブロックが含まれることを確認
	if group, exists := groupsByFileName["resource__aws_instance.tf"]; exists {
		if len(group.Blocks) != 1 {
			t.Errorf("Expected 1 block in resource__aws_instance.tf, got %d", len(group.Blocks))
		}
		if group.BlockType != "resource" {
			t.Errorf("Expected block type 'resource', got '%s'", group.BlockType)
		}
		if group.SubType != "aws_instance" {
			t.Errorf("Expected sub type 'aws_instance', got '%s'", group.SubType)
		}
	} else {
		t.Error("resource__aws_instance.tf group not found")
	}
}

func TestGroupBlocksWithConfig(t *testing.T) {
	// カスタム設定でのテスト
	cfg := &config.Config{
		Groups: []config.GroupConfig{
			{
				Name:     "network",
				Filename: "network.tf",
				Patterns: []string{"aws_vpc", "aws_subnet*", "aws_security_group*"},
			},
			{
				Name:     "compute",
				Filename: "compute.tf",
				Patterns: []string{"aws_instance", "aws_launch_*"},
			},
		},
		Overrides: map[string]string{
			"variable": "vars.tf",
			"locals":   "common.tf",
		},
		Exclude: []string{"aws_instance_special*"},
	}
	
	s := splitter.NewWithConfig(cfg)
	
	parsedFile := &types.ParsedFile{
		Blocks: []*types.Block{
			createTestBlock("variable", []string{"instance_type"}),
			createTestBlock("locals", nil),
			createTestBlock("resource", []string{"aws_vpc", "main"}),
			createTestBlock("resource", []string{"aws_subnet", "public"}),
			createTestBlock("resource", []string{"aws_security_group", "web"}),
			createTestBlock("resource", []string{"aws_instance", "web"}),
			createTestBlock("resource", []string{"aws_instance_special", "admin"}),
			createTestBlock("resource", []string{"aws_s3_bucket", "data"}),
		},
	}
	
	groups := s.GroupBlocks(parsedFile)
	
	groupsByFileName := make(map[string]*types.BlockGroup)
	for _, group := range groups {
		groupsByFileName[group.FileName] = group
	}
	
	// オーバーライドされたファイル名の確認
	if group, exists := groupsByFileName["vars.tf"]; exists {
		if len(group.Blocks) != 1 {
			t.Errorf("Expected 1 block in vars.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("vars.tf group not found")
	}
	
	if group, exists := groupsByFileName["common.tf"]; exists {
		if len(group.Blocks) != 1 {
			t.Errorf("Expected 1 block in common.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("common.tf group not found")
	}
	
	// ネットワークグループの確認
	if group, exists := groupsByFileName["network.tf"]; exists {
		if len(group.Blocks) != 3 { // aws_vpc, aws_subnet, aws_security_group
			t.Errorf("Expected 3 blocks in network.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("network.tf group not found")
	}
	
	// コンピュートグループの確認
	if group, exists := groupsByFileName["compute.tf"]; exists {
		if len(group.Blocks) != 1 { // aws_instance (excluding aws_instance_special)
			t.Errorf("Expected 1 block in compute.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("compute.tf group not found")
	}
	
	// 除外されたリソースが個別ファイルになることを確認
	if group, exists := groupsByFileName["resource__aws_instance_special.tf"]; exists {
		if len(group.Blocks) != 1 {
			t.Errorf("Expected 1 block in resource__aws_instance_special.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("resource__aws_instance_special.tf group not found")
	}
}

