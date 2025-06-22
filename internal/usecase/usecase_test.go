package usecase_test

import (
	"testing"

	"github.com/tomoya-namekawa/tf-file-organize/internal/usecase"
)

func TestDefaultConfigLoader_LoadConfig(t *testing.T) {
	loader := &usecase.DefaultConfigLoader{}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty config path",
			path:    "",
			wantErr: false, // デフォルト設定を返す
		},
		{
			name:    "nonexistent config file",
			path:    "nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loader.LoadConfig(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error = %v, got error = %v for path = %s", tt.wantErr, err != nil, tt.path)
				return
			}

			if !tt.wantErr && cfg == nil {
				t.Errorf("Expected config but got nil for path = %s", tt.path)
			}
		})
	}
}

func TestNewOrganizeFilesUsecase(t *testing.T) {
	uc := usecase.NewOrganizeFilesUsecase()

	if uc == nil {
		t.Error("Expected usecase instance but got nil")
	}
}

func TestNewOrganizeFilesUsecaseWithDeps(t *testing.T) {
	parser, splitter, writer, configLoader := createMockDependencies()

	uc := usecase.NewOrganizeFilesUsecaseWithDeps(parser, splitter, writer, configLoader)

	if uc == nil {
		t.Error("Expected usecase instance but got nil")
	}
}
