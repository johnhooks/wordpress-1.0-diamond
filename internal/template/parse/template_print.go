package parse

import (
	"fmt"
	"strings"
)

// Sprint returns an S-expression representation of the node tree.
// Useful for debugging, golden file tests, and tooling.
//
//	(document
//	  (element "html"
//	    (element "head")
//	    (element "body"
//	      (if (ident "commentsOpen")
//	        (element "p"
//	          (text "yes"))
//	        (else
//	          (element "p"
//	            (text "no")))))))
func Sprint(n *Node) string {
	var sb strings.Builder
	sprintNode(&sb, n, 0)
	return sb.String()
}

func sprintNode(sb *strings.Builder, n *Node, depth int) {
	indent := strings.Repeat("  ", depth)

	switch n.Type {
	case DocumentNode:
		fmt.Fprintf(sb, "%s(document", indent)
	case ElementNode:
		fmt.Fprintf(sb, "%s(element %q", indent, n.Data)
		for _, a := range n.Attr {
			if a.Parts != nil {
				fmt.Fprintf(sb, " (%s", a.Key)
				for _, p := range a.Parts {
					if p.Expr != nil {
						sb.WriteByte(' ')
						sprintExpr(sb, p.Expr)
					} else {
						fmt.Fprintf(sb, " %q", p.Text)
					}
				}
				sb.WriteByte(')')
			} else {
				fmt.Fprintf(sb, " (%s %q)", a.Key, a.Val)
			}
		}
	case TextNode:
		if strings.TrimSpace(n.Data) == "" {
			return // skip whitespace-only text nodes
		}
		fmt.Fprintf(sb, "%s(text %q)", indent, n.Data)
		sb.WriteByte('\n')
		return
	case CommentNode:
		fmt.Fprintf(sb, "%s(comment %q)", indent, n.Data)
		sb.WriteByte('\n')
		return
	case DoctypeNode:
		fmt.Fprintf(sb, "%s(doctype %q)", indent, n.Data)
		sb.WriteByte('\n')
		return
	case ExpressionNode:
		fmt.Fprintf(sb, "%s(expr ", indent)
		sprintTemplateExpr(sb, n)
		sb.WriteByte(')')
		sb.WriteByte('\n')
		return
	case IfNode:
		fmt.Fprintf(sb, "%s(if ", indent)
		sprintTemplateExpr(sb, n)
	case ElseNode:
		fmt.Fprintf(sb, "%s(else", indent)
	case EachNode:
		fmt.Fprintf(sb, "%s(each ", indent)
		sprintTemplateExpr(sb, n)
		if n.TemplateData != nil && n.TemplateData.Binding != "" {
			fmt.Fprintf(sb, " :as %q", n.TemplateData.Binding)
			if n.TemplateData.IndexBinding != "" {
				fmt.Fprintf(sb, " :index %q", n.TemplateData.IndexBinding)
			}
		}
	case SnippetNode:
		fmt.Fprintf(sb, "%s(snippet %q", indent, n.Data)
		if n.TemplateData != nil && n.TemplateData.SnippetParams != "" {
			fmt.Fprintf(sb, " :params %q", n.TemplateData.SnippetParams)
		}
	case ConstNode:
		name := ""
		if n.TemplateData != nil {
			name = n.TemplateData.ConstName
		}
		fmt.Fprintf(sb, "%s(const %q ", indent, name)
		sprintTemplateExpr(sb, n)
		sb.WriteByte(')')
		sb.WriteByte('\n')
		return
	default:
		fmt.Fprintf(sb, "%s(unknown:%d %q)", indent, n.Type, n.Data)
		sb.WriteByte('\n')
		return
	}

	// If this node has children, render them indented.
	if n.FirstChild != nil {
		sb.WriteByte('\n')
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sprintNode(sb, c, depth+1)
		}
		fmt.Fprintf(sb, "%s)", indent)
	} else {
		sb.WriteByte(')')
	}
	sb.WriteByte('\n')
}

// sprintTemplateExpr prints the compiled expression from a node's
// TemplateData, falling back to the raw Data string if not compiled.
func sprintTemplateExpr(sb *strings.Builder, n *Node) {
	if n.TemplateData != nil && n.TemplateData.Expr != nil {
		sprintExpr(sb, n.TemplateData.Expr)
	} else {
		fmt.Fprintf(sb, "%q", n.Data)
	}
}

// sprintExpr prints an expression AST as an S-expression.
func sprintExpr(sb *strings.Builder, expr Expr) {
	switch e := expr.(type) {
	case *Ident:
		fmt.Fprintf(sb, "(ident %q)", e.Name)
	case *MemberExpr:
		sb.WriteString("(. ")
		sprintExpr(sb, e.Object)
		fmt.Fprintf(sb, " %q)", e.Field)
	case *StringLit:
		fmt.Fprintf(sb, "(string %q)", e.Value)
	case *NumberLit:
		fmt.Fprintf(sb, "(number %v)", e.Value)
	case *BoolLit:
		fmt.Fprintf(sb, "(bool %v)", e.Value)
	case *BinaryExpr:
		fmt.Fprintf(sb, "(%s ", e.Op)
		sprintExpr(sb, e.Left)
		sb.WriteByte(' ')
		sprintExpr(sb, e.Right)
		sb.WriteByte(')')
	case *UnaryExpr:
		fmt.Fprintf(sb, "(%s ", e.Op)
		sprintExpr(sb, e.Operand)
		sb.WriteByte(')')
	default:
		fmt.Fprintf(sb, "(unknown %T)", expr)
	}
}
