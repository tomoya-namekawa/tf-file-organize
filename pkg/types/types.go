// Package types defines the core data structures used throughout the tf-file-organize application.
package types

import "github.com/hashicorp/hcl/v2"

// Block represents a Terraform configuration block with its metadata and source content.
type Block struct {
	Type            string    // Block type (resource, variable, output, etc.)
	Labels          []string  // Block labels (resource type, variable name, etc.)
	Body            hcl.Body  // HCL body (structured content)
	DefRange        hcl.Range // Block definition range
	TypeRange       hcl.Range // Block type range
	RawBody         string    // Raw source code within the block (with comments)
	LeadingComments string    // Comments before the block (file-level comments)
	SourceFile      string    // Source file path where this block was parsed from
}

// ParsedFile represents a parsed Terraform file containing a collection of blocks.
type ParsedFile struct {
	FileName string   // Source file name
	Blocks   []*Block // List of parsed blocks
}

// ParsedFiles represents a collection of parsed Terraform files.
type ParsedFiles struct {
	Files []*ParsedFile // List of parsed files
}

// AllBlocks returns all blocks from all parsed files.
func (pf *ParsedFiles) AllBlocks() []*Block {
	var allBlocks []*Block
	for _, file := range pf.Files {
		allBlocks = append(allBlocks, file.Blocks...)
	}
	return allBlocks
}

// FileNames returns list of all source file names.
func (pf *ParsedFiles) FileNames() []string {
	var names []string
	for _, file := range pf.Files {
		names = append(names, file.FileName)
	}
	return names
}

// TotalBlocks returns the total number of blocks across all files.
func (pf *ParsedFiles) TotalBlocks() int {
	total := 0
	for _, file := range pf.Files {
		total += len(file.Blocks)
	}
	return total
}

// BlockGroup represents a group of blocks that will be written to the same output file.
type BlockGroup struct {
	BlockType string   // Block type (basis for grouping)
	SubType   string   // Sub-type (resource type, etc.)
	Blocks    []*Block // Blocks included in the group
	FileName  string   // Output file name
}
