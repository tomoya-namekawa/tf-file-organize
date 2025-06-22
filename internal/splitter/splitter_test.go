package splitter_test

import (
	"testing"

	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
	"github.com/tomoya-namekawa/tf-file-organize/internal/splitter"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

func createTestBlock(blockType string, labels []string) *types.Block {
	return &types.Block{
		Type:   blockType,
		Labels: labels,
	}
}

func TestGroupBlocksDefault(t *testing.T) {
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

	parsedFiles := &types.ParsedFiles{
		Files: []*types.ParsedFile{parsedFile},
	}
	groups, err := s.GroupBlocks(parsedFiles)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedGroups := 9
	if len(groups) != expectedGroups {
		t.Errorf("Expected %d groups, got %d", expectedGroups, len(groups))
	}

	groupsByFileName := make(map[string]*types.BlockGroup)
	for _, group := range groups {
		groupsByFileName[group.FileName] = group
	}
	if group, exists := groupsByFileName["variables.tf"]; exists {
		if len(group.Blocks) != 2 {
			t.Errorf("Expected 2 blocks in variables.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("variables.tf group not found")
	}

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
			{
				Name:     "variables",
				Filename: "vars.tf",
				Patterns: []string{"variable"},
			},
			{
				Name:     "locals",
				Filename: "common.tf",
				Patterns: []string{"locals"},
			},
			{
				Name:     "debug_outputs",
				Filename: "debug-outputs.tf",
				Patterns: []string{"output.debug_*"},
			},
			{
				Name:     "outputs",
				Filename: "outputs.tf",
				Patterns: []string{"output"},
			},
		},
		ExcludeFiles: []string{"*special*.tf", "debug-*.tf"},
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
			createTestBlock("output", []string{"debug_info"}),
			createTestBlock("output", []string{"api_key"}),
		},
	}

	parsedFiles := &types.ParsedFiles{
		Files: []*types.ParsedFile{parsedFile},
	}
	groups, err := s.GroupBlocks(parsedFiles)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	groupsByFileName := make(map[string]*types.BlockGroup)
	for _, group := range groups {
		groupsByFileName[group.FileName] = group
	}

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

	if group, exists := groupsByFileName["network.tf"]; exists {
		if len(group.Blocks) != 3 {
			t.Errorf("Expected 3 blocks in network.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("network.tf group not found")
	}

	if group, exists := groupsByFileName["compute.tf"]; exists {
		if len(group.Blocks) != 1 {
			t.Errorf("Expected 1 block in compute.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("compute.tf group not found")
	}

	if group, exists := groupsByFileName["resource__aws_instance_special.tf"]; exists {
		if len(group.Blocks) != 1 {
			t.Errorf("Expected 1 block in resource__aws_instance_special.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("resource__aws_instance_special.tf group not found")
	}

	if group, exists := groupsByFileName["output__debug_info.tf"]; exists {
		if len(group.Blocks) != 2 {
			t.Errorf("Expected 2 blocks in output__debug_info.tf, got %d", len(group.Blocks))
		}
	} else {
		t.Error("output__debug_info.tf group not found")
	}
}

func TestCheckForDuplicateResources(t *testing.T) {
	s := splitter.New()

	t.Run("duplicate resource names should return error", func(t *testing.T) {
		parsedFile := &types.ParsedFile{
			Blocks: []*types.Block{
				createTestBlock("resource", []string{"aws_instance", "web"}),
				createTestBlock("resource", []string{"aws_instance", "web"}), // duplicate
			},
		}
		parsedFiles := &types.ParsedFiles{
			Files: []*types.ParsedFile{parsedFile},
		}

		_, err := s.GroupBlocks(parsedFiles)
		if err == nil {
			t.Error("Expected error for duplicate resource names, got nil")
		}

		expectedMsg := "duplicate resource name 'aws_instance.web' found"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("duplicate data source names should return error", func(t *testing.T) {
		parsedFile := &types.ParsedFile{
			Blocks: []*types.Block{
				createTestBlock("data", []string{"aws_ami", "ubuntu"}),
				createTestBlock("data", []string{"aws_ami", "ubuntu"}), // duplicate
			},
		}
		parsedFiles := &types.ParsedFiles{
			Files: []*types.ParsedFile{parsedFile},
		}

		_, err := s.GroupBlocks(parsedFiles)
		if err == nil {
			t.Error("Expected error for duplicate data source names, got nil")
		}

		expectedMsg := "duplicate resource name 'data.aws_ami.ubuntu' found"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("unique resource names should not return error", func(t *testing.T) {
		parsedFile := &types.ParsedFile{
			Blocks: []*types.Block{
				createTestBlock("resource", []string{"aws_instance", "web1"}),
				createTestBlock("resource", []string{"aws_instance", "web2"}), // different name
				createTestBlock("data", []string{"aws_ami", "ubuntu"}),
			},
		}
		parsedFiles := &types.ParsedFiles{
			Files: []*types.ParsedFile{parsedFile},
		}

		_, err := s.GroupBlocks(parsedFiles)
		if err != nil {
			t.Errorf("Expected no error for unique resource names, got: %v", err)
		}
	})

	t.Run("non-resource blocks should not be checked", func(t *testing.T) {
		parsedFile := &types.ParsedFile{
			Blocks: []*types.Block{
				createTestBlock("variable", []string{"instance_type"}),
				createTestBlock("variable", []string{"instance_type"}), // duplicate variable names are allowed
				createTestBlock("output", []string{"id"}),
				createTestBlock("output", []string{"id"}), // duplicate output names are allowed
			},
		}
		parsedFiles := &types.ParsedFiles{
			Files: []*types.ParsedFile{parsedFile},
		}

		_, err := s.GroupBlocks(parsedFiles)
		if err != nil {
			t.Errorf("Expected no error for non-resource blocks, got: %v", err)
		}
	})
}
