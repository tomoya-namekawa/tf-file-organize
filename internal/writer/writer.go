// Package writer provides functionality to write grouped Terraform blocks
// to output files with comment preservation and formatting.
package writer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/tomoya-namekawa/tf-file-organize/pkg/types"
)

var emptyBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{},
}

// Writer handles writing grouped blocks to output files.
type Writer struct {
	outputDir string
	dryRun    bool
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
		if block.LeadingComments != "" {
			if i > 0 {
				rootBody.AppendNewline()
			}

			for line := range strings.SplitSeq(block.LeadingComments, "\n") {
				if line != "" {
					rootBody.AppendUnstructuredTokens(hclwrite.Tokens{
						{Type: hclsyntax.TokenComment, Bytes: []byte(line)},
						{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
					})
				} else {
					rootBody.AppendNewline()
				}
			}
			rootBody.AppendNewline()
		} else if i > 0 {
			rootBody.AppendNewline()
		}

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

	formattedContent := hclwrite.Format(content)

	// Check if file already exists with same content (for idempotency)
	if existingContent, err := os.ReadFile(filepath.Clean(filePath)); err == nil {
		if bytes.Equal(existingContent, formattedContent) {
			// File already exists with same content, skip writing
			return nil
		}
	}

	if err := os.WriteFile(filePath, formattedContent, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	fmt.Printf("Created file: %s\n", filePath)
	return nil
}

func (w *Writer) copyBlockBody(sourceBody hcl.Body, targetBody *hclwrite.Body) error {
	return w.copyBlockBodyGeneric(sourceBody, targetBody)
}

func (w *Writer) setAttributeFromExpr(targetBody *hclwrite.Body, name string, expr hcl.Expression) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		targetBody.SetAttributeValue(name, e.Val)
	case *hclsyntax.TemplateExpr:
		w.setTemplateAttribute(targetBody, name, e)
	case *hclsyntax.TupleConsExpr:
		w.setTupleAttribute(targetBody, name, e)
	case *hclsyntax.ScopeTraversalExpr:
		targetBody.SetAttributeTraversal(name, e.Traversal)
	case *hclsyntax.FunctionCallExpr:
		targetBody.SetAttributeValue(name, cty.StringVal(""))
	case *hclsyntax.ObjectConsExpr:
		w.setObjectAttributeSimple(targetBody, name, e)
	default:
		targetBody.SetAttributeValue(name, cty.StringVal(""))
	}
}

func (w *Writer) setTemplateAttribute(targetBody *hclwrite.Body, name string, e *hclsyntax.TemplateExpr) {
	if len(e.Parts) == 1 {
		if literal, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
			targetBody.SetAttributeValue(name, literal.Val)
			return
		}
	}

	tokens := w.buildTemplateTokens(e)
	targetBody.SetAttributeRaw(name, tokens)
}

func (w *Writer) setTupleAttribute(targetBody *hclwrite.Body, name string, e *hclsyntax.TupleConsExpr) {
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

func (w *Writer) setObjectAttributeSimple(targetBody *hclwrite.Body, name string, e *hclsyntax.ObjectConsExpr) {
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})

	for i, item := range e.Items {
		if i > 0 {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n    ")})

		var keyTokens hclwrite.Tokens

		if keyValue, keyDiags := item.KeyExpr.Value(nil); !keyDiags.HasErrors() && keyValue.Type() == cty.String {
			keyTokens = hclwrite.TokensForValue(keyValue)
		} else {
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
				keyTokens = append(keyTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(`"unknown"`)})
			}
		}

		tokens = append(tokens, keyTokens...)

		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})

		switch valueExpr := item.ValueExpr.(type) {
		case *hclsyntax.LiteralValueExpr:
			tokens = append(tokens, hclwrite.TokensForValue(valueExpr.Val)...)
		case *hclsyntax.TemplateExpr:
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

func (w *Writer) copyBlockBodyGeneric(sourceBody hcl.Body, targetBody *hclwrite.Body) error {
	_, remaining, diags := sourceBody.PartialContent(emptyBlockSchema)
	if diags.HasErrors() {
		fmt.Printf("Warning: HCL parsing diagnostics: %v\n", diags)
	}

	w.copyAttributes(sourceBody, targetBody)

	if err := w.copyUnknownBlocks(remaining, targetBody); err != nil {
		return fmt.Errorf("failed to copy blocks: %w", err)
	}

	return nil
}

func (w *Writer) copyAttributes(sourceBody hcl.Body, targetBody *hclwrite.Body) {
	allAttrs, _ := sourceBody.JustAttributes()

	var attrNames []string
	for name := range allAttrs {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)

	for _, name := range attrNames {
		attr := allAttrs[name]
		value, valueDiags := attr.Expr.Value(nil)
		if !valueDiags.HasErrors() {
			targetBody.SetAttributeValue(name, value)
		} else if syntaxBody, ok := sourceBody.(*hclsyntax.Body); ok {
			if syntaxAttr, exists := syntaxBody.Attributes[name]; exists {
				w.setAttributeFromExpr(targetBody, name, syntaxAttr.Expr)
			}
		}
	}
}

func (w *Writer) copyUnknownBlocks(remaining hcl.Body, targetBody *hclwrite.Body) error {
	if remaining == nil {
		return nil
	}

	if syntaxBody, ok := remaining.(*hclsyntax.Body); ok {
		for _, block := range syntaxBody.Blocks {
			nestedBlock := targetBody.AppendNewBlock(block.Type, block.Labels)
			if err := w.copyBlockBody(block.Body, nestedBlock.Body()); err != nil {
				return fmt.Errorf("failed to copy nested block: %w", err)
			}
		}
	} else {
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

func (w *Writer) appendRawBlock(targetBody *hclwrite.Body, block *types.Block) {
	var blockTokens hclwrite.Tokens

	blockTokens = append(blockTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte(block.Type),
	})

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

	blockTokens = append(blockTokens,
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte(" {"),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n" + strings.TrimSpace(block.RawBody) + "\n"),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte("}"),
		})

	targetBody.AppendUnstructuredTokens(blockTokens)
}
