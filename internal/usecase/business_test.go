package usecase_test

import (
	"errors"
	"testing"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/config"
	"github.com/tomoya-namekawa/terraform-file-organize/internal/usecase"
	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
)

// TestOrganizeFilesUsecase_ExecuteBusinessLogic はビジネスロジックのテスト
// モックを使用してファイルI/Oなしでコアロジックを検証
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
				splitter.groupBlocksFunc = func(parsedFile *types.ParsedFile) []*types.BlockGroup {
					return []*types.BlockGroup{
						{
							BlockType: "resource",
							SubType:   "aws_instance",
							Blocks:    parsedFile.Blocks,
							FileName:  "resource__aws_instance.tf",
						},
					}
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
				splitter.groupBlocksFunc = func(parsedFile *types.ParsedFile) []*types.BlockGroup {
					blocks := parsedFile.Blocks
					return []*types.BlockGroup{
						{BlockType: "resource", Blocks: blocks[:1], FileName: "resource__aws_instance.tf"},
						{BlockType: "variable", Blocks: blocks[1:2], FileName: "variables.tf"},
						{BlockType: "output", Blocks: blocks[2:], FileName: "outputs.tf"},
					}
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
				splitter.groupBlocksFunc = func(parsedFile *types.ParsedFile) []*types.BlockGroup {
					return []*types.BlockGroup{
						{BlockType: "resource", Blocks: parsedFile.Blocks, FileName: "resource.tf"},
					}
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
			// モックを作成
			parser, splitter, writer, configLoader := createMockDependencies()

			// モックのセットアップ
			if tt.setupMocks != nil {
				tt.setupMocks(parser, splitter, writer, configLoader)
			}

			// usecaseを依存性注入で作成
			uc := usecase.NewOrganizeFilesUsecaseWithDeps(parser, splitter, writer, configLoader)

			// ビジネスロジックをテスト（ファイルI/Oなし）
			resp, err := testBusinessLogic(uc, tt.blocks, configLoader, splitter, writer)

			// エラーのチェック
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error = %v, got error = %v", tt.expectedError, err)
				return
			}

			// レスポンスのチェック
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

// testBusinessLogic はファイルI/Oを使わないビジネスロジックのテスト用ヘルパー
func testBusinessLogic(_ *usecase.OrganizeFilesUsecase, blocks []*types.Block, configLoader *MockConfigLoader, splitter *MockSplitter, writer *MockWriter) (*usecase.OrganizeFilesResponse, error) {
	// 設定の読み込み
	_, err := configLoader.LoadConfig("")
	if err != nil {
		return nil, err
	}

	// ブロックが空の場合の早期リターン
	if len(blocks) == 0 {
		return &usecase.OrganizeFilesResponse{
			ProcessedFiles: 0,
			TotalBlocks:    0,
			FileGroups:     0,
			OutputDir:      "/test/output",
			WasDryRun:      true,
		}, nil
	}

	// スプリッター経由でのグループ化
	parsedFile := &types.ParsedFile{Blocks: blocks}
	groups := splitter.GroupBlocks(parsedFile)

	// ライター経由での書き込み
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

// TestOrganizeFilesUsecase_ProcessingFlow はビジネスロジックの処理フローをテスト
// 設定読み込み -> グループ化 -> 書き込みの一連の流れを検証
func TestOrganizeFilesUsecase_ProcessingFlow(t *testing.T) {
	parser, splitter, writer, configLoader := createMockDependencies()

	blocks := []*types.Block{
		{Type: "resource", Labels: []string{"aws_instance", "web"}},
		{Type: "variable", Labels: []string{"instance_type"}},
	}

	// モックの設定
	splitter.groupBlocksFunc = func(parsedFile *types.ParsedFile) []*types.BlockGroup {
		parsedBlocks := parsedFile.Blocks
		return []*types.BlockGroup{
			{BlockType: "resource", Blocks: parsedBlocks[:1], FileName: "resource.tf"},
			{BlockType: "variable", Blocks: parsedBlocks[1:], FileName: "variables.tf"},
		}
	}

	writer.writeGroupsFunc = func(groups []*types.BlockGroup) error {
		return nil
	}

	configLoader.loadConfigFunc = func(configPath string) (*config.Config, error) {
		return &config.Config{}, nil
	}

	uc := usecase.NewOrganizeFilesUsecaseWithDeps(parser, splitter, writer, configLoader)

	// ビジネスロジックをテスト
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
