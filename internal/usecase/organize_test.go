package usecase_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/usecase"
)

func getProjectRoot() string {
	// テストファイルから見てプロジェクトルートを取得
	return "../../"
}

func TestOrganizeFilesUsecase_Execute(t *testing.T) {
	tests := []struct {
		name    string
		request *usecase.OrganizeFilesRequest
		wantErr bool
	}{
		{
			name: "single file dry run",
			request: &usecase.OrganizeFilesRequest{
				InputPath:  filepath.Join(getProjectRoot(), "testdata/terraform/sample.tf"),
				OutputDir:  "",
				ConfigFile: "",
				DryRun:     true,
			},
			wantErr: false,
		},
		{
			name: "directory with config",
			request: &usecase.OrganizeFilesRequest{
				InputPath:  filepath.Join(getProjectRoot(), "testdata/terraform"),
				OutputDir:  filepath.Join(getProjectRoot(), "tmp/usecase-test"),
				ConfigFile: filepath.Join(getProjectRoot(), "testdata/configs/terraform-file-organize.yaml"),
				DryRun:     true,
			},
			wantErr: false,
		},
		{
			name: "nonexistent file",
			request: &usecase.OrganizeFilesRequest{
				InputPath:  "nonexistent.tf",
				OutputDir:  "",
				ConfigFile: "",
				DryRun:     true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewOrganizeFilesUsecase()
			resp, err := uc.Execute(tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("OrganizeFilesUsecase.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Errorf("Expected response but got nil")
					return
				}

				if resp.WasDryRun != tt.request.DryRun {
					t.Errorf("Expected WasDryRun = %v, got %v", tt.request.DryRun, resp.WasDryRun)
				}
			}
		})
	}
}

func TestOrganizeFilesUsecase_ValidatePath(t *testing.T) {
	uc := usecase.NewOrganizeFilesUsecase()

	// セキュリティテストは相対パスの問題があるため、エラーケースのみテスト
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "path traversal attempt",
			path:    "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "system directory access",
			path:    "/etc/hosts",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// validatePathは非公開メソッドなので、実際の処理を通してテスト
			req := &usecase.OrganizeFilesRequest{
				InputPath:  tt.path,
				OutputDir:  "",
				ConfigFile: "",
				DryRun:     true,
			}

			_, err := uc.Execute(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error = %v, got error = %v", tt.wantErr, err != nil)
			}
		})
	}
}

func TestOrganizeFilesUsecase_LoadConfig(t *testing.T) {
	// 一時的な設定ファイルを作成
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `groups:
  - name: "test"
    filename: "test.tf"
    patterns:
      - "aws_instance"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	tests := []struct {
		name       string
		configFile string
		wantErr    bool
	}{
		{
			name:       "valid config file",
			configFile: configFile,
			wantErr:    false,
		},
		{
			name:       "empty config file (use default)",
			configFile: "",
			wantErr:    false,
		},
		{
			name:       "nonexistent config file",
			configFile: "nonexistent.yaml",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewOrganizeFilesUsecase()
			req := &usecase.OrganizeFilesRequest{
				InputPath:  filepath.Join(getProjectRoot(), "testdata/terraform/sample.tf"),
				OutputDir:  "",
				ConfigFile: tt.configFile,
				DryRun:     true,
			}

			_, err := uc.Execute(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error = %v, got error = %v", tt.wantErr, err != nil)
			}
		})
	}
}

func TestOrganizeFilesUsecase_Response(t *testing.T) {
	uc := usecase.NewOrganizeFilesUsecase()
	req := &usecase.OrganizeFilesRequest{
		InputPath:  filepath.Join(getProjectRoot(), "testdata/terraform/sample.tf"),
		OutputDir:  filepath.Join(getProjectRoot(), "tmp/response-test"),
		ConfigFile: "",
		DryRun:     true,
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// レスポンスの検証
	if resp.ProcessedFiles != 1 {
		t.Errorf("Expected ProcessedFiles = 1, got %d", resp.ProcessedFiles)
	}

	if resp.TotalBlocks == 0 {
		t.Errorf("Expected TotalBlocks > 0, got %d", resp.TotalBlocks)
	}

	if resp.FileGroups == 0 {
		t.Errorf("Expected FileGroups > 0, got %d", resp.FileGroups)
	}

	if resp.WasDryRun != true {
		t.Errorf("Expected WasDryRun = true, got %v", resp.WasDryRun)
	}

	expectedOutputDir := filepath.Join(getProjectRoot(), "tmp/response-test")
	if resp.OutputDir != expectedOutputDir {
		t.Errorf("Expected OutputDir = %s, got %s", expectedOutputDir, resp.OutputDir)
	}
}
