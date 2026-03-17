package prosemirror

import (
	"encoding/json"
	"html/template"
	"strings"
)

// NodeRenderer renders a single node type. It receives the node and a
// function that renders the node's children (the "content hole" in
// ProseMirror terms). Leaf nodes can ignore children.
type NodeRenderer func(node Node, children func() template.HTML) template.HTML

// MarkRenderer wraps already-rendered content with a mark's HTML.
// This mirrors ProseMirror's mark serialization: marks wrap content
// from outermost to innermost.
type MarkRenderer func(mark Mark, content template.HTML) template.HTML

// Serializer converts a ProseMirror document to HTML. It dispatches
// to per-type renderers for nodes and marks, mirroring ProseMirror's
// DOMSerializer architecture.
type Serializer struct {
	Nodes map[string]NodeRenderer
	Marks map[string]MarkRenderer
}

// Render converts a ProseMirror JSON document to HTML.
func (s *Serializer) Render(data json.RawMessage) (template.HTML, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	if doc.Type != "doc" {
		return "", &RenderError{Message: "expected document type 'doc', got '" + doc.Type + "'"}
	}
	return s.renderFragment(doc.Content), nil
}

// renderFragment renders a slice of child nodes, like ProseMirror's
// serializeFragment.
func (s *Serializer) renderFragment(nodes []Node) template.HTML {
	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(string(s.renderNode(node)))
	}
	return template.HTML(b.String())
}

// renderNode renders a single node with its marks applied.
func (s *Serializer) renderNode(node Node) template.HTML {
	// Render the node itself.
	inner := s.renderNodeInner(node)

	// Apply marks outermost to innermost. ProseMirror stores marks
	// innermost-first (index 0 = innermost), so we iterate forward:
	// each mark wraps the content accumulated so far, producing
	// outer(inner(content)).
	for i := 0; i < len(node.Marks); i++ {
		mark := node.Marks[i]
		if renderer, ok := s.Marks[mark.Type]; ok && renderer != nil {
			inner = renderer(mark, inner)
		}
	}
	return inner
}

// renderNodeInner renders a node without its marks.
func (s *Serializer) renderNodeInner(node Node) template.HTML {
	renderer, ok := s.Nodes[node.Type]
	if !ok {
		// Unknown node types render their children only, avoiding
		// data loss while not inventing markup.
		return s.renderFragment(node.Content)
	}
	return renderer(node, func() template.HTML {
		return s.renderFragment(node.Content)
	})
}

// RenderError is returned when the document structure is invalid.
type RenderError struct {
	Message string
}

func (e *RenderError) Error() string {
	return "prosemirror: " + e.Message
}
