package parse

import "io"

// RenderExpression writes an ExpressionNode to w as {expression}.
// Called from the patched render1 when it encounters ExpressionNode.
func RenderExpression(w writer, n *Node) error {
	if err := w.WriteByte('{'); err != nil {
		return err
	}
	if _, err := io.WriteString(w, n.Data); err != nil {
		return err
	}
	return w.WriteByte('}')
}

// RenderTemplateBlock writes a block template node and its children.
// This reconstructs the template source from the structured AST.
func RenderTemplateBlock(w writer, n *Node) error {
	// Write the opening tag.
	switch n.Type {
	case IfNode:
		if _, err := io.WriteString(w, "{if "); err != nil {
			return err
		}
		if _, err := io.WriteString(w, n.Data); err != nil {
			return err
		}
		if err := w.WriteByte('}'); err != nil {
			return err
		}
	case ElseNode:
		if _, err := io.WriteString(w, "{else}"); err != nil {
			return err
		}
	case EachNode:
		if _, err := io.WriteString(w, "{each "); err != nil {
			return err
		}
		if _, err := io.WriteString(w, n.Data); err != nil {
			return err
		}
		if err := w.WriteByte('}'); err != nil {
			return err
		}
	case SnippetNode:
		if _, err := io.WriteString(w, "{snippet "); err != nil {
			return err
		}
		if _, err := io.WriteString(w, n.Data); err != nil {
			return err
		}
		if n.TemplateData != nil && n.TemplateData.SnippetParams != "" {
			if err := w.WriteByte('('); err != nil {
				return err
			}
			if _, err := io.WriteString(w, n.TemplateData.SnippetParams); err != nil {
				return err
			}
			if err := w.WriteByte(')'); err != nil {
				return err
			}
		}
		if err := w.WriteByte('}'); err != nil {
			return err
		}
	case ConstNode:
		if _, err := io.WriteString(w, "{const "); err != nil {
			return err
		}
		if n.TemplateData != nil {
			if _, err := io.WriteString(w, n.TemplateData.ConstName); err != nil {
				return err
			}
			if _, err := io.WriteString(w, " = "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, n.Data); err != nil {
			return err
		}
		if err := w.WriteByte('}'); err != nil {
			return err
		}
	}

	// Render children.
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := render1(w, c); err != nil {
			return err
		}
	}

	// Write the closing tag.
	switch n.Type {
	case IfNode:
		if _, err := io.WriteString(w, "{/if}"); err != nil {
			return err
		}
	case EachNode:
		if _, err := io.WriteString(w, "{/each}"); err != nil {
			return err
		}
	case SnippetNode:
		if _, err := io.WriteString(w, "{/snippet}"); err != nil {
			return err
		}
	}

	return nil
}
