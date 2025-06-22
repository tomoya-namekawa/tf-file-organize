package validation_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tomoya-namekawa/tf-file-organize/internal/validation"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "current directory",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "testdata",
			wantErr: false,
		},
		{
			name:    "system directory",
			path:    "/etc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInputPath(t *testing.T) {
	// Create temporary file in current directory for testing
	tmpFile, err := os.CreateTemp(".", "test_*.tf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()
	_ = tmpFile.Close()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid file",
			path:    tmpFile.Name(),
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    "non-existent.tf",
			wantErr: true,
		},
		{
			name:    "current directory",
			path:    ".",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateInputPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInputPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOutputPath(t *testing.T) {
	// Create temporary directory in current directory for testing
	tmpDir, err := os.MkdirTemp(".", "test_output_")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create temporary file in current directory for testing
	tmpFile, err := os.CreateTemp(".", "test_*.tf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()
	_ = tmpFile.Close()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path (allowed)",
			path:    "",
			wantErr: false,
		},
		{
			name:    "valid directory",
			path:    tmpDir,
			wantErr: false,
		},
		{
			name:    "file instead of directory",
			path:    tmpFile.Name(),
			wantErr: true,
		},
		{
			name:    "non-existent path (allowed)",
			path:    filepath.Join(tmpDir, "new-dir"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateOutputPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	// Create temporary config file in current directory for testing
	tmpFile, err := os.CreateTemp(".", "test_config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()
	_ = tmpFile.Close()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path (allowed)",
			path:    "",
			wantErr: false,
		},
		{
			name:    "valid config file",
			path:    tmpFile.Name(),
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    "non-existent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateConfigPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFlagCombination(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		recursive bool
		wantErr   bool
	}{
		{
			name:      "no flags",
			outputDir: "",
			recursive: false,
			wantErr:   false,
		},
		{
			name:      "only output-dir",
			outputDir: "output",
			recursive: false,
			wantErr:   false,
		},
		{
			name:      "only recursive",
			outputDir: "",
			recursive: true,
			wantErr:   false,
		},
		{
			name:      "both flags (invalid)",
			outputDir: "output",
			recursive: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateFlagCombination(tt.outputDir, tt.recursive)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFlagCombination() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
