package parse

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html/atom"
)

// ParseTemplate parses HTML with template syntax and restructures
// block nodes ({if}...{/if}, {each}...{/each}, {snippet}...{/snippet})
// into a proper tree. Call this instead of Parse for template files.
func ParseTemplate(r io.Reader) (*Node, error) {
	doc, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return templatePasses(doc)
}

// ParseTemplateFragment parses an HTML fragment (no <html>, <head>,
// <body> scaffolding) with template syntax. Used for molecules and
// organisms that are rendered as sub-templates within a page.
func ParseTemplateFragment(r io.Reader) (*Node, error) {
	context := &Node{
		Type:     ElementNode,
		DataAtom: atom.Lookup([]byte("body")),
		Data:     "body",
	}
	nodes, err := ParseFragment(r, context)
	if err != nil {
		return nil, err
	}

	// Wrap the fragment nodes in a DocumentNode so the walker
	// can treat it the same as a full template.
	doc := &Node{Type: DocumentNode}
	for _, n := range nodes {
		doc.AppendChild(n)
	}

	return templatePasses(doc)
}

func templatePasses(doc *Node) (*Node, error) {
	if err := splitRawTextExprs(doc); err != nil {
		return nil, err
	}
	if err := restructureBlocks(doc); err != nil {
		return nil, err
	}
	if err := compileAttrExprs(doc); err != nil {
		return nil, err
	}
	if err := validatePass(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// compileAttrExprs walks the tree and pre-compiles {expr} patterns
// in element attribute values. After this pass, attributes containing
// expressions have their Parts field populated with pre-compiled
// AttrPart slices. Pure static attributes are left unchanged.
func compileAttrExprs(n *Node) error {
	if n.Type == ElementNode {
		for i := range n.Attr {
			if !strings.Contains(n.Attr[i].Val, "{") {
				continue
			}
			parts, err := parseAttrParts(n.Attr[i].Val)
			if err != nil {
				return fmt.Errorf("attribute %q on <%s>: %w", n.Attr[i].Key, n.Data, err)
			}
			n.Attr[i].Parts = parts
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := compileAttrExprs(c); err != nil {
			return err
		}
	}
	return nil
}

// parseAttrParts splits an attribute value into static text and
// compiled expression segments.
func parseAttrParts(val string) ([]AttrPart, error) {
	var parts []AttrPart
	for i := 0; i < len(val); {
		open := strings.IndexByte(val[i:], '{')
		if open < 0 {
			parts = append(parts, AttrPart{Text: val[i:]})
			break
		}
		if open > 0 {
			parts = append(parts, AttrPart{Text: val[i : i+open]})
		}
		close := strings.IndexByte(val[i+open:], '}')
		if close < 0 {
			return nil, fmt.Errorf("unclosed expression in attribute value: %s", val[i+open:])
		}
		exprStr := val[i+open+1 : i+open+close]
		expr, err := ParseExpr(exprStr)
		if err != nil {
			return nil, fmt.Errorf("invalid expression {%s}: %w", exprStr, err)
		}
		parts = append(parts, AttrPart{Expr: expr})
		i = i + open + close + 1
	}
	return parts, nil
}

// rawTextElements are HTML elements where the tokenizer reads content
// as raw text, bypassing template expression recognition. The
// tokenizer's readRawOrRCDATA function only looks for the closing tag
// and ignores '{'. We fix this in a post-parse pass rather than
// patching the raw text state machine.
var rawTextElements = map[string]bool{
	"title":    true,
	"textarea": true,
	// style and script are rejected by validatePass, but handle
	// them here too in case validation order changes.
	"style":  true,
	"script": true,
}

// splitRawTextExprs walks the tree and splits text nodes inside raw
// text elements (title, textarea) that contain template expressions.
//
// The HTML5 tokenizer reads content inside these elements as raw
// text, so {expression} is not recognized during tokenization. This
// pass finds those text nodes, parses expressions out of them, and
// replaces the single text node with a sequence of text and
// expression nodes.
//
// This is a workaround. A cleaner solution would patch the
// tokenizer's readRawOrRCDATA to recognize '{', but that function
// has complex buffering and end-tag detection that we do not want
// to modify in the fork.
func splitRawTextExprs(n *Node) error {
	if n.Type == ElementNode && rawTextElements[n.Data] {
		// Collect children to avoid mutating during iteration.
		var children []*Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			children = append(children, c)
		}
		for _, c := range children {
			if c.Type != TextNode || !strings.Contains(c.Data, "{") {
				continue
			}
			nodes, err := splitTextIntoExprs(c.Data)
			if err != nil {
				return fmt.Errorf("in <%s>: %w", n.Data, err)
			}
			if len(nodes) == 1 && nodes[0].Type == TextNode {
				continue // no expressions found, leave as-is
			}
			for _, newNode := range nodes {
				n.InsertBefore(newNode, c)
			}
			n.RemoveChild(c)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := splitRawTextExprs(c); err != nil {
			return err
		}
	}
	return nil
}

// splitTextIntoExprs splits a raw text string into a sequence of
// TextNode and ExpressionNode nodes by finding {expression} patterns.
func splitTextIntoExprs(text string) ([]*Node, error) {
	var nodes []*Node
	for i := 0; i < len(text); {
		open := strings.IndexByte(text[i:], '{')
		if open < 0 {
			nodes = append(nodes, &Node{Type: TextNode, Data: text[i:]})
			break
		}
		if open > 0 {
			nodes = append(nodes, &Node{Type: TextNode, Data: text[i : i+open]})
		}

		close := strings.IndexByte(text[i+open:], '}')
		if close < 0 {
			return nil, fmt.Errorf("unclosed expression in text: %s", text[i+open:])
		}

		exprStr := text[i+open+1 : i+open+close]
		expr, err := ParseExpr(exprStr)
		if err != nil {
			return nil, fmt.Errorf("invalid expression {%s}: %w", exprStr, err)
		}
		nodes = append(nodes, &Node{
			Type:         ExpressionNode,
			Data:         exprStr,
			TemplateData: &TemplateData{Expr: expr},
		})
		i = i + open + close + 1
	}
	return nodes, nil
}

// validatePass walks the restructured tree and rejects elements
// that are not allowed in templates.
func validatePass(n *Node) error {
	if n.Type == ElementNode {
		if n.Data == "script" || n.Data == "style" {
			return fmt.Errorf("<%s> elements are not allowed in templates", n.Data)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := validatePass(c); err != nil {
			return err
		}
	}
	return nil
}

// restructureBlocks walks the tree and converts flat block markers
// into nested parent-child structures using a stack, mirroring how
// the HTML5 tree builder uses an open elements stack.
//
// Before: parent → [IfNode, <p>, ElseNode, <p>, ifCloseNode]
// After:  parent → [IfNode → [<p>, ElseNode → [<p>]]]
//
// For each parent node, we walk its children with a stack of open
// block nodes. Block opens push onto the stack, content is moved
// to the top of stack, and block closes pop. Close markers are
// removed from the tree.
func restructureBlocks(n *Node) error {
	// Collect children into a slice so we can detach and reattach
	// without fighting the linked list during iteration.
	var children []*Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, c)
	}
	if len(children) == 0 {
		return nil
	}

	// Check if any restructuring is needed before detaching.
	// Close markers are the signal: they only exist in the flat
	// tree before restructuring. After restructuring they are
	// consumed, so their presence means this level still needs
	// a restructuring pass.
	hasCloses := false
	for _, c := range children {
		if isBlockClose(c.Type) {
			hasCloses = true
			break
		}
	}

	if !hasCloses {
		for _, c := range children {
			// Block opens without any close markers at this level.
			// Either already restructured (has children) or unclosed.
			if isBlockOpen(c.Type) && c.FirstChild == nil {
				return blockError(c, n)
			}
			// Else without a parent block is an error.
			if isBlockMiddle(c.Type) && (c.Parent == nil || !isBlockOpen(c.Parent.Type)) {
				return fmt.Errorf("{%s} without matching block", blockName(c.Type))
			}
		}
		// No restructuring needed. Recurse into children.
		for _, c := range children {
			if err := restructureBlocks(c); err != nil {
				return err
			}
		}
		return nil
	}

	// Detach all children from parent.
	for _, c := range children {
		n.RemoveChild(c)
	}

	// Walk children with a block stack. The stack tracks open block
	// nodes. Content goes to the top of the stack, or back to the
	// parent if the stack is empty.
	var stack []*Node

	target := func() *Node {
		if len(stack) > 0 {
			return stack[len(stack)-1]
		}
		return n
	}

	for _, c := range children {
		switch {
		case isBlockOpen(c.Type):
			// Attach the block node to the current target,
			// then push it so subsequent nodes become its children.
			target().AppendChild(c)
			stack = append(stack, c)

		case c.Type == ElseNode:
			// {else} is a child of the current IfNode or EachNode.
			// Subsequent nodes go into the ElseNode.
			if len(stack) == 0 || (stack[len(stack)-1].Type != IfNode && stack[len(stack)-1].Type != EachNode) {
				return fmt.Errorf("{else} without matching {if} or {each}")
			}
			stack[len(stack)-1].AppendChild(c)
			// Push ElseNode so content goes into it.
			stack = append(stack, c)

		case isBlockClose(c.Type):
			// Pop the stack back to the matching open.
			if err := popBlock(&stack, c.Type); err != nil {
				return err
			}
			// Close marker is consumed; not added to the tree.

		default:
			target().AppendChild(c)
		}
	}

	// Any remaining open blocks are unclosed.
	if len(stack) > 0 {
		top := stack[len(stack)-1]
		return blockError(top, n)
	}

	// Recurse into all children (now properly nested).
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := restructureBlocks(c); err != nil {
			return err
		}
	}
	return nil
}

// popBlock pops the stack back to the matching open for a close marker.
// ElseNode is an intermediate node that sits between the open and
// close, so we pop past it to reach the block open.
func popBlock(stack *[]*Node, closeType NodeType) error {
	wantOpen := closeToOpen(closeType)

	for len(*stack) > 0 {
		top := (*stack)[len(*stack)-1]
		*stack = (*stack)[:len(*stack)-1]

		if top.Type == wantOpen {
			return nil
		}

		// ElseNode is intermediate — pop through it.
		if top.Type == ElseNode {
			continue
		}

		return fmt.Errorf("{/%s} closed a {%s} block", closeName(closeType), blockName(top.Type))
	}

	return fmt.Errorf(
		"{/%s} without matching open; "+
			"this usually means the {%s} block was broken by HTML parsing, "+
			"where the HTML5 parser rearranged elements between {%s} and {/%s} "+
			"because a block-level element appeared where only inline content "+
			"is allowed (like a <div> inside a <p>)",
		closeName(closeType), closeName(closeType),
		closeName(closeType), closeName(closeType),
	)
}

// blockError produces a diagnostic for a template block whose open
// marker has no matching close at the same level. If a matching close
// exists elsewhere in the tree, the HTML5 parser split them apart,
// and we produce a diagnostic explaining the nesting issue. Otherwise
// it is a genuinely unclosed block.
func blockError(block, parent *Node) error {
	name := blockName(block.Type)
	wantClose := openToClose(block.Type)
	if wantClose != 0 && hasCloseElsewhere(parent, wantClose) {
		hint := ""
		if parent.Type == ElementNode {
			hint = fmt.Sprintf(
				"; move the {%s} block outside the <%s> or use only "+
					"inline elements inside it", name, parent.Data)
		}
		return fmt.Errorf(
			"{%s} block was broken by HTML parsing; "+
				"the HTML5 parser rearranged elements between {%s} and {/%s}, "+
				"most likely because a block-level element (like <div>) appeared "+
				"where only inline content is allowed (like inside <p>)%s",
			name, name, name, hint,
		)
	}
	return fmt.Errorf("unclosed {%s} block", name)
}

// hasCloseElsewhere checks whether a close marker of the given type
// exists anywhere in the tree rooted at n or its ancestors' siblings.
func hasCloseElsewhere(n *Node, closeType NodeType) bool {
	// Walk up to the root, checking siblings at each level.
	for cur := n; cur != nil; cur = cur.Parent {
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			if findNodeType(c, closeType) {
				return true
			}
		}
	}
	return false
}

// findNodeType does a depth-first search for a node of the given type.
func findNodeType(n *Node, t NodeType) bool {
	if n.Type == t {
		return true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if findNodeType(c, t) {
			return true
		}
	}
	return false
}

func openToClose(t NodeType) NodeType {
	switch t {
	case IfNode:
		return ifCloseNode
	case EachNode:
		return eachCloseNode
	case SnippetNode:
		return snippetCloseNode
	}
	return 0
}

func isBlockOpen(t NodeType) bool {
	return t == IfNode || t == EachNode || t == SnippetNode
}

func isBlockClose(t NodeType) bool {
	return t == ifCloseNode || t == eachCloseNode || t == snippetCloseNode
}

func isBlockMiddle(t NodeType) bool {
	return t == ElseNode
}

func closeToOpen(t NodeType) NodeType {
	switch t {
	case ifCloseNode:
		return IfNode
	case eachCloseNode:
		return EachNode
	case snippetCloseNode:
		return SnippetNode
	}
	return 0
}

func blockName(t NodeType) string {
	switch t {
	case IfNode:
		return "if"
	case EachNode:
		return "each"
	case SnippetNode:
		return "snippet"
	case ElseNode:
		return "else"
	}
	return "unknown"
}

func closeName(t NodeType) string {
	switch t {
	case ifCloseNode:
		return "if"
	case eachCloseNode:
		return "each"
	case snippetCloseNode:
		return "snippet"
	}
	return "unknown"
}
