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
}

// ParsedFile represents a parsed Terraform file containing a collection of blocks.
type ParsedFile struct {
	Blocks []*Block // List of parsed blocks
}

// BlockGroup represents a group of blocks that will be written to the same output file.
type BlockGroup struct {
	BlockType string   // Block type (basis for grouping)
	SubType   string   // Sub-type (resource type, etc.)
	Blocks    []*Block // Blocks included in the group
	FileName  string   // Output file name
}
