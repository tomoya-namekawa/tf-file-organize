package usecase_test

import (
	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

// MockParser はParserのモック実装
type MockParser struct {
	parseFileFunc func(filename string) (*types.ParsedFile, error)
}

func (m *MockParser) ParseFile(filename string) (*types.ParsedFile, error) {
	if m.parseFileFunc != nil {
		return m.parseFileFunc(filename)
	}
	// デフォルトの動作
	return &types.ParsedFile{
		Blocks: []*types.Block{
			{
				Type:   "resource",
				Labels: []string{"aws_instance", "test"},
			},
		},
	}, nil
}

// MockSplitter はSplitterのモック実装
type MockSplitter struct {
	groupBlocksFunc func(parsedFile *types.ParsedFile) []*types.BlockGroup
}

func (m *MockSplitter) GroupBlocks(parsedFile *types.ParsedFile) []*types.BlockGroup {
	if m.groupBlocksFunc != nil {
		return m.groupBlocksFunc(parsedFile)
	}
	// デフォルトの動作
	return []*types.BlockGroup{
		{
			BlockType: "resource",
			SubType:   "aws_instance",
			Blocks:    parsedFile.Blocks,
			FileName:  "resource__aws_instance.tf",
		},
	}
}

// MockWriter はWriterのモック実装
type MockWriter struct {
	writeGroupsFunc func(groups []*types.BlockGroup) error
}

func (m *MockWriter) WriteGroups(groups []*types.BlockGroup) error {
	if m.writeGroupsFunc != nil {
		return m.writeGroupsFunc(groups)
	}
	// デフォルトの動作（何もしない）
	return nil
}

// MockConfigLoader は設定読み込みのモック実装
type MockConfigLoader struct {
	loadConfigFunc func(configPath string) (*config.Config, error)
}

func (m *MockConfigLoader) LoadConfig(configPath string) (*config.Config, error) {
	if m.loadConfigFunc != nil {
		return m.loadConfigFunc(configPath)
	}
	// デフォルトの動作
	return &config.Config{}, nil
}

// createMockDependencies はテスト用のモック依存関係を作成
func createMockDependencies() (*MockParser, *MockSplitter, *MockWriter, *MockConfigLoader) {
	return &MockParser{}, &MockSplitter{}, &MockWriter{}, &MockConfigLoader{}
}
