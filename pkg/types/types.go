// Package types defines the core data structures used throughout the tf-file-organize application.
package types

import "github.com/hashicorp/hcl/v2"

// Block represents a Terraform configuration block with its metadata and source content.
type Block struct {
	Type            string    // ブロックタイプ (resource, variable, output, etc.)
	Labels          []string  // ブロックラベル (resource type, variable name, etc.)
	Body            hcl.Body  // HCLボディ（構造化された内容）
	DefRange        hcl.Range // ブロック定義の範囲
	TypeRange       hcl.Range // ブロックタイプの範囲
	RawBody         string    // ブロック内の生のソースコード（コメント付き）
	LeadingComments string    // ブロック前のコメント（ファイルレベルコメント）
}

// ParsedFile represents a parsed Terraform file containing a collection of blocks.
type ParsedFile struct {
	Blocks []*Block // 解析されたブロックのリスト
}

// BlockGroup represents a group of blocks that will be written to the same output file.
type BlockGroup struct {
	BlockType string   // ブロックタイプ（グループ化の基準）
	SubType   string   // サブタイプ（リソースタイプなど）
	Blocks    []*Block // グループに含まれるブロック
	FileName  string   // 出力ファイル名
}
