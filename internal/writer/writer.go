// Package writer provides functionality to write grouped Terraform blocks
// to output files with comment preservation and formatting.
package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
)

// emptyBlockSchema is used when we don't need to parse internal structure and prefer RawBody.
var emptyBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{},
}

// Writer handles writing grouped blocks to output files.
type Writer struct {
	outputDir string // 出力ディレクトリ
	dryRun    bool   // ドライランモード
}

// New creates a new Writer with default settings.
func New(outputDir string, dryRun bool) *Writer {
	return &Writer{
		outputDir: outputDir,
		dryRun:    dryRun,
	}
}

// WriteGroups writes all block groups to their respective output files.
func (w *Writer) WriteGroups(groups []*types.BlockGroup) error {
	if !w.dryRun {
		if err := os.MkdirAll(w.outputDir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	for _, group := range groups {
		if err := w.writeGroup(group); err != nil {
			return fmt.Errorf("failed to write group %s: %w", group.FileName, err)
		}
	}

	return nil
}

func (w *Writer) writeGroup(group *types.BlockGroup) error {
	filePath := filepath.Join(w.outputDir, group.FileName)

	if w.dryRun {
		fmt.Printf("Would create file: %s\n", filePath)
		fmt.Printf("  Block type: %s\n", group.BlockType)
		if group.SubType != "" {
			fmt.Printf("  Sub type: %s\n", group.SubType)
		}
		fmt.Printf("  Number of blocks: %d\n", len(group.Blocks))
		fmt.Println()
		return nil
	}

	file := hclwrite.NewEmptyFile()
	rootBody := file.Body()

	for i, block := range group.Blocks {
		// ブロック前のLeadingCommentsを出力（常に有効）
		if block.LeadingComments != "" {
			// 最初のブロック以外でLeadingCommentsがある場合、前のブロックとの間に空行を追加
			if i > 0 {
				rootBody.AppendNewline()
			}

			// LeadingCommentsを行ごとに分割してトークンとして追加
			lines := strings.Split(block.LeadingComments, "\n")
			for _, line := range lines {
				if line != "" {
					rootBody.AppendUnstructuredTokens(hclwrite.Tokens{
						{Type: hclsyntax.TokenComment, Bytes: []byte(line)},
						{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
					})
				} else {
					rootBody.AppendNewline()
				}
			}
			// LeadingCommentsの後に空行を追加
			rootBody.AppendNewline()
		} else if i > 0 {
			// LeadingCommentsがない場合で最初のブロック以外では空行を追加
			rootBody.AppendNewline()
		}

		// 生のソースコードが利用可能な場合はそれを使用
		if block.RawBody != "" {
			w.appendRawBlock(rootBody, block)
		} else {
			newBlock := rootBody.AppendNewBlock(block.Type, block.Labels)
			if err := w.copyBlockBody(block.Body, newBlock.Body()); err != nil {
				return fmt.Errorf("failed to copy block body: %w", err)
			}
		}
	}

	content := file.Bytes()

	// hclwrite.Formatを使用してフォーマット
	formattedContent := hclwrite.Format(content)

	if err := os.WriteFile(filePath, formattedContent, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	fmt.Printf("Created file: %s\n", filePath)
	return nil
}

func (w *Writer) copyBlockBody(sourceBody hcl.Body, targetBody *hclwrite.Body) error {
	// RawBodyが優先されるため、この関数はフォールバック用として単純化
	return w.copyBlockBodyGeneric(sourceBody, targetBody)
}

// setAttributeFromExpr は式から属性を設定
func (w *Writer) setAttributeFromExpr(targetBody *hclwrite.Body, name string, expr hcl.Expression) {
	// 式の種類に応じて処理
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		targetBody.SetAttributeValue(name, e.Val)
	case *hclsyntax.TemplateExpr:
		w.setTemplateAttribute(targetBody, name, e)
	case *hclsyntax.TupleConsExpr:
		w.setTupleAttribute(targetBody, name, e)
	case *hclsyntax.ScopeTraversalExpr:
		// 変数参照の場合、参照をそのまま設定
		targetBody.SetAttributeTraversal(name, e.Traversal)
	case *hclsyntax.FunctionCallExpr:
		// 関数呼び出しの場合、空文字列
		targetBody.SetAttributeValue(name, cty.StringVal(""))
	case *hclsyntax.ObjectConsExpr:
		// オブジェクトの場合、より簡単な方法で処理
		w.setObjectAttributeSimple(targetBody, name, e)
	default:
		// その他の場合は空の文字列として扱う
		targetBody.SetAttributeValue(name, cty.StringVal(""))
	}
}

// setTemplateAttribute はテンプレート式の属性を設定
func (w *Writer) setTemplateAttribute(targetBody *hclwrite.Body, name string, e *hclsyntax.TemplateExpr) {
	// 単純なリテラル値の場合は直接設定
	if len(e.Parts) == 1 {
		if literal, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
			targetBody.SetAttributeValue(name, literal.Val)
			return
		}
	}

	// 複雑なテンプレートの場合は共通のtoken builder を使用
	tokens := w.buildTemplateTokens(e)
	targetBody.SetAttributeRaw(name, tokens)
}

// setTupleAttribute は配列式の属性を設定
func (w *Writer) setTupleAttribute(targetBody *hclwrite.Body, name string, e *hclsyntax.TupleConsExpr) {
	// 配列の場合、適切に処理
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})

	for i, subExpr := range e.Exprs {
		if i > 0 {
			tokens = append(tokens,
				&hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")},
				&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte(" ")})
		}

		switch se := subExpr.(type) {
		case *hclsyntax.ScopeTraversalExpr:
			tokens = append(tokens, hclwrite.TokensForTraversal(se.Traversal)...)
		case *hclsyntax.LiteralValueExpr:
			tokens = append(tokens, hclwrite.TokensForValue(se.Val)...)
		default:
			tokens = append(tokens, hclwrite.TokensForValue(cty.StringVal(""))...)
		}
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
	targetBody.SetAttributeRaw(name, tokens)
}

// setObjectAttributeSimple はオブジェクト式をRawトークンとして設定
func (w *Writer) setObjectAttributeSimple(targetBody *hclwrite.Body, name string, e *hclsyntax.ObjectConsExpr) {
	// オブジェクトのトークンを構築
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})

	for i, item := range e.Items {
		if i > 0 {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n    ")})

		// キーの処理 - より堅牢なアプローチ
		var keyTokens hclwrite.Tokens

		// まずキーを評価してみる
		if keyValue, keyDiags := item.KeyExpr.Value(nil); !keyDiags.HasErrors() && keyValue.Type() == cty.String {
			// 正常に評価できた文字列キー
			keyTokens = hclwrite.TokensForValue(keyValue)
		} else {
			// 評価できない場合は、式の型に応じて処理
			switch keyExpr := item.KeyExpr.(type) {
			case *hclsyntax.LiteralValueExpr:
				keyTokens = hclwrite.TokensForValue(keyExpr.Val)
			case *hclsyntax.TemplateExpr:
				if len(keyExpr.Parts) == 1 {
					if literal, ok := keyExpr.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
						keyTokens = hclwrite.TokensForValue(literal.Val)
					}
				}
			default:
				// フォールバック
				keyTokens = append(keyTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(`"unknown"`)})
			}
		}

		tokens = append(tokens, keyTokens...)

		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})

		// 値の処理
		switch valueExpr := item.ValueExpr.(type) {
		case *hclsyntax.LiteralValueExpr:
			tokens = append(tokens, hclwrite.TokensForValue(valueExpr.Val)...)
		case *hclsyntax.TemplateExpr:
			// テンプレート式を再構築
			templateTokens := w.buildTemplateTokens(valueExpr)
			tokens = append(tokens, templateTokens...)
		case *hclsyntax.ScopeTraversalExpr:
			tokens = append(tokens, hclwrite.TokensForTraversal(valueExpr.Traversal)...)
		default:
			tokens = append(tokens, hclwrite.TokensForValue(cty.StringVal(""))...)
		}
	}

	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n  ")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})

	targetBody.SetAttributeRaw(name, tokens)
}

// copyBlockBodyGeneric は汎用的なコピー方法（簡素化版）
func (w *Writer) copyBlockBodyGeneric(sourceBody hcl.Body, targetBody *hclwrite.Body) error {
	// 内部構造を詳細に解析せず、シンプルにコピー
	_, remaining, diags := sourceBody.PartialContent(emptyBlockSchema)
	if diags.HasErrors() {
		// エラーがあっても続行してベストエフォートで処理
		fmt.Printf("Warning: HCL parsing diagnostics: %v\n", diags)
	}

	// 属性をコピー
	w.copyAttributes(sourceBody, targetBody)

	// 全てのブロックを未知として処理（内部構造は気にしない）
	if err := w.copyUnknownBlocks(remaining, targetBody); err != nil {
		return fmt.Errorf("failed to copy blocks: %w", err)
	}

	return nil
}

// copyAttributes はソースボディから属性をコピー
func (w *Writer) copyAttributes(sourceBody hcl.Body, targetBody *hclwrite.Body) {
	allAttrs, _ := sourceBody.JustAttributes()

	// 属性名をソートして決定的な順序にする
	var attrNames []string
	for name := range allAttrs {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)

	// ソートされた順序で属性をコピー
	for _, name := range attrNames {
		attr := allAttrs[name]
		value, valueDiags := attr.Expr.Value(nil)
		if !valueDiags.HasErrors() {
			targetBody.SetAttributeValue(name, value)
		} else if syntaxBody, ok := sourceBody.(*hclsyntax.Body); ok {
			// syntax bodyから直接処理
			if syntaxAttr, exists := syntaxBody.Attributes[name]; exists {
				w.setAttributeFromExpr(targetBody, name, syntaxAttr.Expr)
			}
		}
	}
}

// copyUnknownBlocks は未知のブロックをコピー
func (w *Writer) copyUnknownBlocks(remaining hcl.Body, targetBody *hclwrite.Body) error {
	if remaining == nil {
		return nil
	}

	// remainingから直接すべてのブロックを取得
	if syntaxBody, ok := remaining.(*hclsyntax.Body); ok {
		// syntax bodyから直接ブロックを取得
		for _, block := range syntaxBody.Blocks {
			nestedBlock := targetBody.AppendNewBlock(block.Type, block.Labels)
			if err := w.copyBlockBody(block.Body, nestedBlock.Body()); err != nil {
				return fmt.Errorf("failed to copy nested block: %w", err)
			}
		}
	} else {
		// フォールバック: 従来の方法
		unknownContent, _, _ := remaining.PartialContent(&hcl.BodySchema{})
		for _, block := range unknownContent.Blocks {
			nestedBlock := targetBody.AppendNewBlock(block.Type, block.Labels)
			if err := w.copyBlockBody(block.Body, nestedBlock.Body()); err != nil {
				return fmt.Errorf("failed to copy nested block: %w", err)
			}
		}
	}
	return nil
}

// buildTemplateTokens はテンプレート式のトークンを構築する共通関数
func (w *Writer) buildTemplateTokens(valueExpr *hclsyntax.TemplateExpr) hclwrite.Tokens {
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)})
	for _, part := range valueExpr.Parts {
		switch p := part.(type) {
		case *hclsyntax.LiteralValueExpr:
			if p.Val.Type() == cty.String {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenQuotedLit,
					Bytes: []byte(p.Val.AsString()),
				})
			}
		case *hclsyntax.ScopeTraversalExpr:
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenTemplateInterp,
				Bytes: []byte("${"),
			})
			tokens = append(tokens, hclwrite.TokensForTraversal(p.Traversal)...)
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenTemplateSeqEnd,
				Bytes: []byte("}"),
			})
		default:
			// 未知のパートタイプの場合はスキップするのではなく、placeholderを出力
			tokens = append(tokens,
				&hclwrite.Token{
					Type:  hclsyntax.TokenTemplateInterp,
					Bytes: []byte("${"),
				},
				&hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("unknown"),
				},
				&hclwrite.Token{
					Type:  hclsyntax.TokenTemplateSeqEnd,
					Bytes: []byte("}"),
				})
		}
	}
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)})
	return tokens
}

// appendRawBlock は生のソースコードを使用してブロックを追加
func (w *Writer) appendRawBlock(targetBody *hclwrite.Body, block *types.Block) {
	// ブロックのヘッダーを構築
	var blockTokens hclwrite.Tokens

	// ブロックタイプを追加
	blockTokens = append(blockTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte(block.Type),
	})

	// ラベルを追加
	for _, label := range block.Labels {
		blockTokens = append(blockTokens,
			&hclwrite.Token{
				Type:  hclsyntax.TokenOQuote,
				Bytes: []byte(" \""),
			},
			&hclwrite.Token{
				Type:  hclsyntax.TokenQuotedLit,
				Bytes: []byte(label),
			},
			&hclwrite.Token{
				Type:  hclsyntax.TokenCQuote,
				Bytes: []byte("\""),
			})
	}

	// ブロック開始、ボディ、終了を追加
	blockTokens = append(blockTokens,
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte(" {"),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte(block.RawBody),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte("}"),
		})

	// ターゲットボディにトークンを追加
	targetBody.AppendUnstructuredTokens(blockTokens)
}
