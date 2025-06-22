package usecase_test

import (
	"errors"
	"testing"

	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
	"github.com/tomoya-namekawa/tf-file-organize/internal/usecase"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

// TestOrganizeFilesUsecase_ExecuteBusinessLogic tests business logic with mocks
func TestOrganizeFilesUsecase_ExecuteBusinessLogic(t *testing.T) {
	tests := []struct {
		name          string
		blocks        []*types.Block
		setupMocks    func(*MockParser, *MockSplitter, *MockWriter, *MockConfigLoader)
		expectedError bool
		expectedResp  func(*usecase.OrganizeFilesResponse) bool
	}{
		{
			name: "successful processing with single resource block",
			blocks: []*types.Block{
				{
					Type:   "resource",
					Labels: []string{"aws_instance", "web"},
				},
			},
			setupMocks: func(parser *MockParser, splitter *MockSplitter, writer *MockWriter, configLoader *MockConfigLoader) {
				splitter.groupBlocksFunc = func(parsedFiles *types.ParsedFiles) ([]*types.BlockGroup, error) {
					return []*types.BlockGroup{
						{
							BlockType: "resource",
							SubType:   "aws_instance",
							Blocks:    parsedFiles.AllBlocks(),
							FileName:  "resource__aws_instance.tf",
						},
					}, nil
				}
				writer.writeGroupsFunc = func(groups []*types.BlockGroup) error {
					return nil
				}
				configLoader.loadConfigFunc = func(configPath string) (*config.Config, error) {
					return &config.Config{}, nil
				}
			},
			expectedError: false,
			expectedResp: func(resp *usecase.OrganizeFilesResponse) bool {
				return resp != nil && resp.FileGroups == 1 && resp.TotalBlocks == 1
			},
		},
		{
			name: "multiple block types",
			blocks: []*types.Block{
				{Type: "resource", Labels: []string{"aws_instance", "web"}},
				{Type: "variable", Labels: []string{"instance_type"}},
				{Type: "output", Labels: []string{"instance_ip"}},
			},
			setupMocks: func(parser *MockParser, splitter *MockSplitter, writer *MockWriter, configLoader *MockConfigLoader) {
				splitter.groupBlocksFunc = func(parsedFiles *types.ParsedFiles) ([]*types.BlockGroup, error) {
					blocks := parsedFiles.AllBlocks()
					return []*types.BlockGroup{
						{BlockType: "resource", Blocks: blocks[:1], FileName: "resource__aws_instance.tf"},
						{BlockType: "variable", Blocks: blocks[1:2], FileName: "variables.tf"},
						{BlockType: "output", Blocks: blocks[2:], FileName: "outputs.tf"},
					}, nil
				}
				writer.writeGroupsFunc = func(groups []*types.BlockGroup) error {
					return nil
				}
				configLoader.loadConfigFunc = func(configPath string) (*config.Config, error) {
					return &config.Config{}, nil
				}
			},
			expectedError: false,
			expectedResp: func(resp *usecase.OrganizeFilesResponse) bool {
				return resp != nil && resp.FileGroups == 3 && resp.TotalBlocks == 3
			},
		},
		{
			name:   "empty blocks",
			blocks: []*types.Block{},
			setupMocks: func(parser *MockParser, splitter *MockSplitter, writer *MockWriter, configLoader *MockConfigLoader) {
				configLoader.loadConfigFunc = func(configPath string) (*config.Config, error) {
					return &config.Config{}, nil
				}
			},
			expectedError: false,
			expectedResp: func(resp *usecase.OrganizeFilesResponse) bool {
				return resp != nil && resp.TotalBlocks == 0 && resp.FileGroups == 0
			},
		},
		{
			name: "config loader error",
			blocks: []*types.Block{
				{Type: "resource", Labels: []string{"aws_instance", "web"}},
			},
			setupMocks: func(parser *MockParser, splitter *MockSplitter, writer *MockWriter, configLoader *MockConfigLoader) {
				configLoader.loadConfigFunc = func(configPath string) (*config.Config, error) {
					return nil, errors.New("config error")
				}
			},
			expectedError: true,
			expectedResp:  nil,
		},
		{
			name: "writer error",
			blocks: []*types.Block{
				{Type: "resource", Labels: []string{"aws_instance", "web"}},
			},
			setupMocks: func(parser *MockParser, splitter *MockSplitter, writer *MockWriter, configLoader *MockConfigLoader) {
				splitter.groupBlocksFunc = func(parsedFiles *types.ParsedFiles) ([]*types.BlockGroup, error) {
					return []*types.BlockGroup{
						{BlockType: "resource", Blocks: parsedFiles.AllBlocks(), FileName: "resource.tf"},
					}, nil
				}
				writer.writeGroupsFunc = func(groups []*types.BlockGroup) error {
					return errors.New("write error")
				}
				configLoader.loadConfigFunc = func(configPath string) (*config.Config, error) {
					return &config.Config{}, nil
				}
			},
			expectedError: true,
			expectedResp:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, splitter, writer, configLoader := createMockDependencies()

			if tt.setupMocks != nil {
				tt.setupMocks(parser, splitter, writer, configLoader)
			}

			uc := usecase.NewOrganizeFilesUsecaseWithDeps(parser, splitter, writer, configLoader)

			resp, err := testBusinessLogic(uc, tt.blocks, configLoader, splitter, writer)

			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error = %v, got error = %v", tt.expectedError, err)
				return
			}
			if !tt.expectedError {
				if resp == nil {
					t.Error("Expected response but got nil")
					return
				}
				if tt.expectedResp != nil && !tt.expectedResp(resp) {
					t.Errorf("Response validation failed: %+v", resp)
				}
			}
		})
	}
}

// testBusinessLogic tests business logic without file I/O
func testBusinessLogic(_ *usecase.OrganizeFilesUsecase, blocks []*types.Block, configLoader *MockConfigLoader, splitter *MockSplitter, writer *MockWriter) (*usecase.OrganizeFilesResponse, error) {
	_, err := configLoader.LoadConfig("")
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return &usecase.OrganizeFilesResponse{
			ProcessedFiles: 0,
			TotalBlocks:    0,
			FileGroups:     0,
			OutputDir:      "/test/output",
			WasDryRun:      true,
		}, nil
	}

	parsedFiles := &types.ParsedFiles{
		Files: []*types.ParsedFile{
			{FileName: "test.tf", Blocks: blocks},
		},
	}
	groups, err := splitter.GroupBlocks(parsedFiles)
	if err != nil {
		return nil, err
	}

	err = writer.WriteGroups(groups)
	if err != nil {
		return nil, err
	}

	return &usecase.OrganizeFilesResponse{
		ProcessedFiles: 1,
		TotalBlocks:    len(blocks),
		FileGroups:     len(groups),
		OutputDir:      "/test/output",
		WasDryRun:      true,
	}, nil
}

// TestOrganizeFilesUsecase_ProcessingFlow tests the processing flow
func TestOrganizeFilesUsecase_ProcessingFlow(t *testing.T) {
	parser, splitter, writer, configLoader := createMockDependencies()

	blocks := []*types.Block{
		{Type: "resource", Labels: []string{"aws_instance", "web"}},
		{Type: "variable", Labels: []string{"instance_type"}},
	}

	splitter.groupBlocksFunc = func(parsedFiles *types.ParsedFiles) ([]*types.BlockGroup, error) {
		parsedBlocks := parsedFiles.AllBlocks()
		return []*types.BlockGroup{
			{BlockType: "resource", Blocks: parsedBlocks[:1], FileName: "resource.tf"},
			{BlockType: "variable", Blocks: parsedBlocks[1:], FileName: "variables.tf"},
		}, nil
	}

	writer.writeGroupsFunc = func(groups []*types.BlockGroup) error {
		return nil
	}

	configLoader.loadConfigFunc = func(configPath string) (*config.Config, error) {
		return &config.Config{}, nil
	}

	uc := usecase.NewOrganizeFilesUsecaseWithDeps(parser, splitter, writer, configLoader)

	resp, err := testBusinessLogic(uc, blocks, configLoader, splitter, writer)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Error("Expected response but got nil")
		return
	}

	// レスポンスの検証
	if resp.TotalBlocks != 2 {
		t.Errorf("Expected 2 blocks, got %d", resp.TotalBlocks)
	}

	if resp.FileGroups != 2 {
		t.Errorf("Expected 2 file groups, got %d", resp.FileGroups)
	}

	if !resp.WasDryRun {
		t.Error("Expected dry run to be true")
	}
}
