package types

import "github.com/hashicorp/hcl/v2"

type Block struct {
	Type      string
	Labels    []string
	Body      hcl.Body
	DefRange  hcl.Range
	TypeRange hcl.Range
	RawBody   string // ブロック内の生のソースコード（コメント付き）
}

type ParsedFile struct {
	Blocks []*Block
}

type BlockGroup struct {
	BlockType string
	SubType   string
	Blocks    []*Block
	FileName  string
}
