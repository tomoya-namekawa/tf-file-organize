package writer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"

	"github.com/tomoya-namekawa/tf-file-organize/internal/writer"
	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

func createTestBlockGroup(filename, blockType string, blocks []*types.Block) *types.BlockGroup {
	return &types.BlockGroup{
		FileName:  filename,
		BlockType: blockType,
		Blocks:    blocks,
	}
}

func parseHCLBlock(t *testing.T, content string) *types.Block {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(content), "test.tf")
	if diags.HasErrors() {
		t.Fatalf("Failed to parse HCL: %v", diags)
	}

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "variable", LabelNames: []string{"name"}},
			{Type: "output", LabelNames: []string{"name"}},
		},
	}

	contentHCL, _, diags := file.Body.PartialContent(schema)
	if diags.HasErrors() {
		t.Fatalf("Failed to extract content: %v", diags)
	}

	if len(contentHCL.Blocks) == 0 {
		t.Fatal("No blocks found in test content")
	}

	block := contentHCL.Blocks[0]
	return &types.Block{
		Type:   block.Type,
		Labels: block.Labels,
		Body:   block.Body,
	}
}

func TestWriteGroupsDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	w := writer.New(tmpDir, true)

	block := parseHCLBlock(t, `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t3.micro"
}
`)

	groups := []*types.BlockGroup{
		createTestBlockGroup("resource__aws_instance.tf", "resource", []*types.Block{block}),
	}

	err := w.WriteGroups(groups)
	if err != nil {
		t.Fatalf("WriteGroups failed: %v", err)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected no files in dry run, got %d files", len(files))
	}
}

func TestWriteGroupsActual(t *testing.T) {
	tmpDir := t.TempDir()
	w := writer.New(tmpDir, false)

	variableBlock := parseHCLBlock(t, `
variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}
`)

	resourceBlock := parseHCLBlock(t, `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t3.micro"
}
`)

	groups := []*types.BlockGroup{
		createTestBlockGroup("variables.tf", "variable", []*types.Block{variableBlock}),
		createTestBlockGroup("resource__aws_instance.tf", "resource", []*types.Block{resourceBlock}),
	}

	err := w.WriteGroups(groups)
	if err != nil {
		t.Fatalf("WriteGroups failed: %v", err)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	variablesPath := filepath.Join(tmpDir, "variables.tf")
	variablesContent, err := os.ReadFile(variablesPath)
	if err != nil {
		t.Fatalf("Failed to read variables.tf: %v", err)
	}

	variablesStr := string(variablesContent)
	if !strings.Contains(variablesStr, `variable "instance_type"`) {
		t.Errorf("variables.tf does not contain expected variable block")
	}

	resourcePath := filepath.Join(tmpDir, "resource__aws_instance.tf")
	resourceContent, err := os.ReadFile(resourcePath)
	if err != nil {
		t.Fatalf("Failed to read resource__aws_instance.tf: %v", err)
	}

	resourceStr := string(resourceContent)
	if !strings.Contains(resourceStr, `resource "aws_instance" "web"`) {
		t.Errorf("resource__aws_instance.tf does not contain expected resource block")
	}
}

func TestWriteGroupsMultipleBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	w := writer.New(tmpDir, false)

	variable1 := parseHCLBlock(t, `
variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}
`)

	variable2 := parseHCLBlock(t, `
variable "key_name" {
  description = "Key pair name"
  type        = string
}
`)

	groups := []*types.BlockGroup{
		createTestBlockGroup("variables.tf", "variable", []*types.Block{variable1, variable2}),
	}

	err := w.WriteGroups(groups)
	if err != nil {
		t.Fatalf("WriteGroups failed: %v", err)
	}

	variablesPath := filepath.Join(tmpDir, "variables.tf")
	content, err := os.ReadFile(variablesPath)
	if err != nil {
		t.Fatalf("Failed to read variables.tf: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `variable "instance_type"`) {
		t.Errorf("variables.tf does not contain instance_type variable")
	}
	if !strings.Contains(contentStr, `variable "key_name"`) {
		t.Errorf("variables.tf does not contain key_name variable")
	}

	variableCount := strings.Count(contentStr, "variable ")
	if variableCount != 2 {
		t.Errorf("Expected 2 variable blocks, found %d", variableCount)
	}
}

func TestWriteGroupsOutputDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "custom", "output")
	w := writer.New(outputDir, false)

	block := parseHCLBlock(t, `
output "instance_id" {
  value = "i-12345"
}
`)

	groups := []*types.BlockGroup{
		createTestBlockGroup("outputs.tf", "output", []*types.Block{block}),
	}

	err := w.WriteGroups(groups)
	if err != nil {
		t.Fatalf("WriteGroups failed: %v", err)
	}

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("Output directory was not created: %s", outputDir)
	}

	outputPath := filepath.Join(outputDir, "outputs.tf")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file was not created: %s", outputPath)
	}
}

func TestNewWriter(t *testing.T) {
	outputDir := "/tmp/test"
	dryRun := true

	w := writer.New(outputDir, dryRun)

	if w == nil {
		t.Error("Expected writer instance, got nil")
	}
	tmpDir := t.TempDir()
	testWriter := writer.New(tmpDir, false)

	block := parseHCLBlock(t, `variable "test" { type = string }`)
	groups := []*types.BlockGroup{
		createTestBlockGroup("test.tf", "variable", []*types.Block{block}),
	}

	err := testWriter.WriteGroups(groups)
	if err != nil {
		t.Errorf("Writer should work correctly: %v", err)
	}
}
