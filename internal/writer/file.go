package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/tomoya-namekawa/terraform-file-organize/pkg/types"
)

type Writer struct {
	outputDir string
	dryRun    bool
}

func New(outputDir string, dryRun bool) *Writer {
	return &Writer{
		outputDir: outputDir,
		dryRun:    dryRun,
	}
}

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
		if i > 0 {
			rootBody.AppendNewline()
		}

		newBlock := rootBody.AppendNewBlock(block.Type, block.Labels)

		if err := w.copyBlockBody(block.Body, newBlock.Body()); err != nil {
			return fmt.Errorf("failed to copy block body: %w", err)
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
	// より直接的なアプローチを採用
	return w.copyBlockBodyGeneric(sourceBody, targetBody)
}

// setAttributeFromExpr は式から属性を設定
func (w *Writer) setAttributeFromExpr(targetBody *hclwrite.Body, name string, expr hcl.Expression) {
	// 式の種類に応じて処理
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		targetBody.SetAttributeValue(name, e.Val)
	case *hclsyntax.TemplateExpr:
		w.setTemplateAttribute(targetBody, name, e, expr)
	case *hclsyntax.TupleConsExpr:
		w.setTupleAttribute(targetBody, name, e)
	case *hclsyntax.ScopeTraversalExpr:
		// 変数参照の場合、参照をそのまま設定
		targetBody.SetAttributeTraversal(name, e.Traversal)
	case *hclsyntax.FunctionCallExpr:
		// 関数呼び出しの場合、空文字列
		targetBody.SetAttributeValue(name, cty.StringVal(""))
	case *hclsyntax.ObjectConsExpr:
		// オブジェクトの場合、空の構造体を設定
		targetBody.SetAttributeValue(name, cty.ObjectVal(map[string]cty.Value{}))
	default:
		// その他の場合は空の文字列として扱う
		targetBody.SetAttributeValue(name, cty.StringVal(""))
	}
}

// setTemplateAttribute はテンプレート式の属性を設定
func (w *Writer) setTemplateAttribute(targetBody *hclwrite.Body, name string, e *hclsyntax.TemplateExpr, expr hcl.Expression) {
	if len(e.Parts) == 1 {
		if literal, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
			targetBody.SetAttributeValue(name, literal.Val)
			return
		}
	}
	// 複雑なテンプレートの場合、元の式をそのまま保持
	if sourceRange := expr.Range(); sourceRange.Filename != "" {
		w.setComplexTemplateAttribute(targetBody, name, e)
	} else {
		// フォールバック: 空文字列
		targetBody.SetAttributeValue(name, cty.StringVal(""))
	}
}

// setComplexTemplateAttribute は複雑なテンプレート式の属性を設定
func (w *Writer) setComplexTemplateAttribute(targetBody *hclwrite.Body, name string, e *hclsyntax.TemplateExpr) {
	// 元のソースからトークンを再構築
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOQuote,
		Bytes: []byte(`"`),
	})

	// テンプレートの各部分を処理
	for _, part := range e.Parts {
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
			// 他の式タイプの場合はプレースホルダー
			tokens = append(tokens,
				&hclwrite.Token{
					Type:  hclsyntax.TokenTemplateInterp,
					Bytes: []byte("${"),
				},
				&hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("var.placeholder"),
				},
				&hclwrite.Token{
					Type:  hclsyntax.TokenTemplateSeqEnd,
					Bytes: []byte("}"),
				})
		}
	}

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCQuote,
		Bytes: []byte(`"`),
	})

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

// copyBlockBodyGeneric は汎用的なコピー方法
func (w *Writer) copyBlockBodyGeneric(sourceBody hcl.Body, targetBody *hclwrite.Body) error {
	// 包括的なschemaを定義して属性とブロックを取得
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			// AWS provider blocks
			{Type: "filter"},
			{Type: "ingress"},
			{Type: "egress"},
			{Type: "lifecycle"},
			{Type: "provisioner", LabelNames: []string{"type"}},
			{Type: "connection"},
			{Type: "dynamic", LabelNames: []string{"for_each"}},
			{Type: "route"},
			{Type: "versioning_configuration"},
			{Type: "required_providers"},

			// Google Cloud provider blocks
			{Type: "template"},
			{Type: "spec"},
			{Type: "containers"},
			{Type: "env"},
			{Type: "resources"},
			{Type: "ports"},
			{Type: "traffic"},
			{Type: "metadata"},
			{Type: "binding"},
			{Type: "limits"},
			{Type: "requests"},

			// Azure provider blocks
			{Type: "site_config"},
			{Type: "identity"},
			{Type: "auth_settings"},
			{Type: "backup"},
			{Type: "connection_string"},
			{Type: "logs"},

			// Kubernetes provider blocks
			{Type: "selector"},
			{Type: "volume"},
			{Type: "volume_mount"},
			{Type: "config_map"},
			{Type: "secret"},
			{Type: "service_account"},
			{Type: "security_context"},

			// Common blocks
			{Type: "tags"},
			{Type: "timeouts"},
			{Type: "depends_on"},
			{Type: "count"},
			{Type: "for_each"},
			{Type: "condition"},
			{Type: "precondition"},
			{Type: "postcondition"},
			{Type: "check"},
			{Type: "validation"},
			{Type: "sensitive"},
			{Type: "nullable"},
			{Type: "description"},

			// Network and security blocks
			{Type: "security_group_rule"},
			{Type: "subnet"},
			{Type: "vpc"},
			{Type: "network"},
			{Type: "firewall"},
			{Type: "load_balancer"},
			{Type: "target_group"},
			{Type: "listener"},
			{Type: "health_check"},

			// Storage blocks
			{Type: "disk"},
			{Type: "volume_attachment"},
			{Type: "snapshot"},
			{Type: "backup_policy"},
			{Type: "encryption"},

			// Database blocks
			{Type: "database"},
			{Type: "user"},
			{Type: "replica"},
			{Type: "parameter_group"},
			{Type: "subnet_group"},
			{Type: "option_group"},

			// Monitoring and logging
			{Type: "alarm"},
			{Type: "metric"},
			{Type: "dashboard"},
			{Type: "log_group"},
			{Type: "log_stream"},
			{Type: "notification"},

			// IAM and permissions
			{Type: "policy"},
			{Type: "role"},
			{Type: "user_policy_attachment"},
			{Type: "role_policy_attachment"},
			{Type: "group_policy_attachment"},
			{Type: "assume_role_policy"},
			{Type: "trust_policy"},

			// Generic blocks for unknown types
			{Type: "block"},
			{Type: "nested_block"},
			{Type: "configuration"},
			{Type: "settings"},
			{Type: "options"},
			{Type: "parameters"},
			{Type: "attributes"},
		},
	}

	// PartialContentで属性とブロックを取得
	content, remaining, diags := sourceBody.PartialContent(schema)
	if diags.HasErrors() {
		// エラーがあっても続行してベストエフォートで処理
		fmt.Printf("Warning: HCL parsing diagnostics: %v\n", diags)
	}

	// まず全ての属性を取得してアルファベット順でソートしてコピー
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

	// 既知のブロックをコピー
	for _, block := range content.Blocks {
		nestedBlock := targetBody.AppendNewBlock(block.Type, block.Labels)
		if err := w.copyBlockBody(block.Body, nestedBlock.Body()); err != nil {
			return fmt.Errorf("failed to copy nested block: %w", err)
		}
	}

	// 残りのブロック（未知のブロック）をコピー
	if remaining != nil {
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
