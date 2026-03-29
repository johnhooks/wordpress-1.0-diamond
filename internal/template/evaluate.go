package template

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"press/internal/errors"
	"press/internal/template/parse"
)

func newDocumentNode() *parse.Node {
	return &parse.Node{Type: parse.DocumentNode}
}

func newTextNode(text string) *parse.Node {
	return &parse.Node{Type: parse.TextNode, Data: text}
}

func newRawNode(html string) *parse.Node {
	return &parse.Node{Type: parse.RawNode, Data: html}
}

func newElementNode(tag string, attrs []parse.Attribute) *parse.Node {
	return &parse.Node{
		Type: parse.ElementNode,
		Data: tag,
		Attr: attrs,
	}
}

func newCommentNode(data string) *parse.Node {
	return &parse.Node{Type: parse.CommentNode, Data: data}
}

func cloneShallow(n *parse.Node) *parse.Node {
	return n.CloneNode()
}

// BuildElement creates an element node with the given tag and attributes,
// then moves all children from body into it. Returns a document node
// wrapping the element. Tag handlers use this to build engine-owned
// wrapper elements (like <form>) around evaluated template content.
func BuildElement(tag string, attrs []parse.Attribute, body *parse.Node) *parse.Node {
	el := &parse.Node{
		Type: parse.ElementNode,
		Data: tag,
		Attr: attrs,
	}
	for c := body.FirstChild; c != nil; {
		next := c.NextSibling
		body.RemoveChild(c)
		el.AppendChild(c)
		c = next
	}
	doc := newDocumentNode()
	doc.AppendChild(el)
	return doc
}

// HiddenInput creates an <input type="hidden"> element node.
func HiddenInput(name, value string) *parse.Node {
	return &parse.Node{
		Type: parse.ElementNode,
		Data: "input",
		Attr: []parse.Attribute{
			{Key: "type", Val: "hidden"},
			{Key: "name", Val: name},
			{Key: "value", Val: value},
		},
	}
}

func appendChildren(parent *parse.Node, children []*parse.Node) {
	for _, c := range children {
		parent.AppendChild(c)
	}
}

// Evaluator transforms a template AST into a plain HTML AST.
// Tag handlers return *parse.Node subtrees instead of HTML strings.
// The output is rendered by parse.Render.
type Evaluator struct {
	ctx   context.Context
	scope *Scope
}

// NewEvaluator creates an Evaluator with data as the root scope.
// The context should carry a TagResolver (via WithTagResolver) so
// nested vocabulary tags resolve correctly. Used by RenderTag to
// invoke tag handlers outside the normal full-page evaluation path.
func NewEvaluator(ctx context.Context, data any) *Evaluator {
	return &Evaluator{ctx: ctx, scope: NewScope(data)}
}

// Evaluate evaluates a template AST against data and returns
// a plain HTML node tree suitable for parse.Render. Panics on
// evaluation errors; a validated theme with correct data should
// never produce errors.
func Evaluate(ctx context.Context, doc *parse.Node, data any) *parse.Node {
	ev := &Evaluator{
		ctx:   ctx,
		scope: NewScope(data),
	}
	out := newDocumentNode()
	if err := ev.evaluateChildren(out, doc); err != nil {
		panic(fmt.Sprintf("template: Evaluate: %v", err))
	}
	return out
}

// evaluateInto evaluates node n and appends zero or more output
// nodes to parent. This is the flattening mechanism: control flow
// nodes (if, each) produce zero or more children directly into
// the parent rather than wrapping them in a container.
func (ev *Evaluator) evaluateInto(parent *parse.Node, n *parse.Node) error {
	switch n.Type {
	case parse.DocumentNode:
		return ev.evaluateChildren(parent, n)

	case parse.TextNode:
		// Text is stored unescaped in the AST; parse.Render handles escaping.
		parent.AppendChild(newTextNode(n.Data))
		return nil

	case parse.CommentNode:
		parent.AppendChild(newCommentNode(n.Data))
		return nil

	case parse.DoctypeNode:
		c := cloneShallow(n)
		parent.AppendChild(c)
		return nil

	case parse.RawNode:
		parent.AppendChild(newRawNode(n.Data))
		return nil

	case parse.ElementNode:
		return ev.evaluateElement(parent, n)

	case parse.ExpressionNode:
		return ev.evaluateExpression(parent, n)

	case parse.IfNode:
		return ev.evaluateIf(parent, n)

	case parse.EachNode:
		return ev.evaluateEach(parent, n)

	case parse.ConstNode:
		return ev.evaluateConst(n)

	case parse.SnippetNode:
		// Snippet definitions are extracted at compile time,
		// not rendered inline.
		return nil

	default:
		return evaluateErr("unknown node type %d", n.Type)
	}
}

// evaluateChildren evaluates all children of src and appends them to parent.
func (ev *Evaluator) evaluateChildren(parent *parse.Node, src *parse.Node) error {
	for c := src.FirstChild; c != nil; c = c.NextSibling {
		if err := ev.evaluateInto(parent, c); err != nil {
			return err
		}
	}
	return nil
}

// evaluateElement handles a plain element node. If a TagResolver is
// on the context, it checks for a vocabulary tag handler first.
// Panics if the element is a known vocabulary tag without a handler.
func (ev *Evaluator) evaluateElement(parent *parse.Node, n *parse.Node) error {
	// Check for vocabulary tag handler.
	if resolver, ok := tagResolverFromContext(ev.ctx); ok {
		if handler := resolver.ResolveTag(n.Data); handler != nil {
			return ev.evaluateVocabularyTag(parent, n, handler)
		}
	}

	if parse.IsVocabularyTag(n.Data) {
		panic(fmt.Sprintf("template: vocabulary tag <%s> has no handler", n.Data))
	}

	// Clone the element (no children).
	el := cloneShallow(n)

	// Resolve attribute expressions.
	if err := ev.resolveAttrs(el); err != nil {
		return evaluateWrap(err, "resolving attributes on <%s>", n.Data)
	}

	parent.AppendChild(el)

	// Evaluate children into the new element.
	return ev.evaluateChildren(el, n)
}

// evaluateExpression evaluates an ExpressionNode and appends a TextNode
// (or RawNode for RawHTML) to parent.
func (ev *Evaluator) evaluateExpression(parent *parse.Node, n *parse.Node) error {
	expr := getExpr(n)
	if expr == nil {
		return evaluateErr("expression {%s} not compiled", n.Data)
	}

	val, err := Eval(expr, ev.scope)
	if err != nil {
		return evaluateWrap(err, "evaluating {%s}", n.Data)
	}

	// RawHTML values produce RawNode (no escaping by parse.Render).
	if raw, ok := val.(RawHTML); ok {
		parent.AppendChild(newRawNode(string(raw)))
		return nil
	}

	// Regular values produce TextNode (parse.Render escapes).
	parent.AppendChild(newTextNode(formatValue(val)))
	return nil
}

// resolveAttrs evaluates expression parts in element attributes,
// replacing Parts with resolved plain Val strings. Unlike EvalAttrParts
// (used by the walker), this does not HTML-escape expression values
// because parse.Render handles escaping when it writes the attribute.
func (ev *Evaluator) resolveAttrs(el *parse.Node) error {
	for i, attr := range el.Attr {
		if attr.Parts == nil {
			continue
		}
		val, err := evalAttrPartsUnescaped(attr.Parts, ev.scope)
		if err != nil {
			return evaluateWrap(err, "resolving attribute %q", attr.Key)
		}
		el.Attr[i].Val = val
		el.Attr[i].Parts = nil
	}
	return nil
}

// evalAttrPartsUnescaped evaluates attribute parts without HTML escaping.
// The evaluator stores raw values in the AST; parse.Render escapes them.
func evalAttrPartsUnescaped(parts []parse.AttrPart, ctx any) (string, error) {
	var sb strings.Builder
	for _, part := range parts {
		if part.Expr != nil {
			val, err := EvalString(part.Expr, ctx)
			if err != nil {
				return "", err
			}
			sb.WriteString(val)
		} else {
			sb.WriteString(part.Text)
		}
	}
	return sb.String(), nil
}

// evaluateIf handles an IfNode. Evaluates the condition and appends
// either the then-branch or else-branch children to parent.
func (ev *Evaluator) evaluateIf(parent *parse.Node, n *parse.Node) error {
	expr := getExpr(n)
	if expr == nil {
		return evaluateErr("if condition {%s} not compiled", n.Data)
	}

	truthy, err := EvalTruthy(expr, ev.scope)
	if err != nil {
		return evaluateWrap(err, "evaluating if condition {%s}", n.Data)
	}

	if truthy {
		// Evaluate then-branch: all children before ElseNode.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				break
			}
			if err := ev.evaluateInto(parent, c); err != nil {
				return err
			}
		}
	} else {
		// Evaluate else-branch: children of ElseNode.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				return ev.evaluateChildren(parent, c)
			}
		}
	}
	return nil
}

// evaluateEach handles an EachNode. Iterates over the list and appends
// body children for each item, or else-branch if empty.
func (ev *Evaluator) evaluateEach(parent *parse.Node, n *parse.Node) error {
	if n.TemplateData == nil {
		return evaluateErr("each node missing template data")
	}

	expr := getExpr(n)
	if expr == nil {
		return evaluateErr("each list {%s} not compiled", n.Data)
	}

	listVal, err := Eval(expr, ev.scope)
	if err != nil {
		return evaluateWrap(err, "evaluating each list {%s}", n.Data)
	}

	rv := reflect.ValueOf(listVal)
	empty := !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) || rv.Len() == 0

	if empty {
		// Evaluate the else branch if present.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				return ev.evaluateChildren(parent, c)
			}
		}
		return nil
	}

	binding := n.TemplateData.Binding
	indexBinding := n.TemplateData.IndexBinding

	for i := 0; i < rv.Len(); i++ {
		child := ev.scope.Push()
		if binding != "" {
			child.Set(binding, rv.Index(i).Interface())
		}
		if indexBinding != "" {
			child.Set(indexBinding, i)
		}

		oldScope := ev.scope
		ev.scope = child

		// Evaluate body children (before ElseNode).
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				break
			}
			if err := ev.evaluateInto(parent, c); err != nil {
				ev.scope = oldScope
				return err
			}
		}

		ev.scope = oldScope
	}
	return nil
}

// evaluateConst handles a ConstNode. Sets a variable in the current scope.
func (ev *Evaluator) evaluateConst(n *parse.Node) error {
	if n.TemplateData == nil {
		return evaluateErr("const node missing template data")
	}

	expr := getExpr(n)
	if expr == nil {
		return evaluateErr("const expression {%s} not compiled", n.Data)
	}

	val, err := Eval(expr, ev.scope)
	if err != nil {
		return evaluateWrap(err, "evaluating const {%s}", n.Data)
	}

	ev.scope.Set(n.TemplateData.ConstName, val)
	return nil
}

// TagNodeHandler is a vocabulary tag handler for the evaluator.
// It receives the element node and returns a replacement subtree.
// Handlers control descent by calling back into the evaluator.
// Handlers must not fail; panics indicate programmer errors.
type TagNodeHandler func(ctx context.Context, ev *Evaluator, el *parse.Node) *parse.Node

// TagResolver resolves vocabulary tag names to node-based handlers.
// Returns nil for elements that are not vocabulary tags.
type TagResolver interface {
	ResolveTag(name string) TagNodeHandler
}

type tagResolverKey struct{}

// WithTagResolver stores a TagResolver on the context.
func WithTagResolver(ctx context.Context, r TagResolver) context.Context {
	return context.WithValue(ctx, tagResolverKey{}, r)
}

func tagResolverFromContext(ctx context.Context) (TagResolver, bool) {
	r, ok := ctx.Value(tagResolverKey{}).(TagResolver)
	return r, ok
}

// evaluateVocabularyTag calls a TagNodeHandler and appends its output to parent.
// The handler receives a shallow clone of the element with resolved attributes
// and the original source node. The source node gives the handler access to the
// full child tree for evaluation via ev.EvaluateChildren without any cloning
// that could lose nested content. The handler must not mutate the source node.
func (ev *Evaluator) evaluateVocabularyTag(parent *parse.Node, n *parse.Node, handler TagNodeHandler) error {
	// Clone the element so the handler gets resolved attributes
	// without mutating the cached AST. The clone has no children;
	// handlers use the source node to evaluate children.
	el := cloneShallow(n)
	if err := ev.resolveAttrs(el); err != nil {
		return evaluateWrap(err, "resolving attributes on <%s>", n.Data)
	}

	result := handler(ev.ctx, ev, el)

	if result == nil {
		return nil
	}

	// If the handler returns a DocumentNode, append its children.
	// Otherwise append the node itself.
	if result.Type == parse.DocumentNode {
		// Collect children first to avoid mutation during iteration.
		var children []*parse.Node
		for c := result.FirstChild; c != nil; c = c.NextSibling {
			children = append(children, c)
		}
		for _, c := range children {
			result.RemoveChild(c)
			parent.AppendChild(c)
		}
	} else {
		parent.AppendChild(result)
	}
	return nil
}

// EvaluateInto is the public entry point for handlers to evaluate
// a template node and append results into a parent node.
func (ev *Evaluator) EvaluateInto(parent *parse.Node, n *parse.Node) error {
	return ev.evaluateInto(parent, n)
}

// EvaluateChildren is the public entry point for handlers to evaluate
// all children of a source node into a parent node.
func (ev *Evaluator) EvaluateChildren(parent *parse.Node, src *parse.Node) error {
	return ev.evaluateChildren(parent, src)
}

// EvaluateScoped creates a child scope with the given data, evaluates
// the template, and returns the result tree. Used by tag handlers to
// render sub-templates with their own data context.
func (ev *Evaluator) EvaluateScoped(doc *parse.Node, data any) *parse.Node {
	child := ev.scope.PushData(data)

	oldScope := ev.scope
	ev.scope = child

	out := newDocumentNode()
	if err := ev.evaluateChildren(out, doc); err != nil {
		panic(fmt.Sprintf("template: EvaluateScoped: %v", err))
	}

	ev.scope = oldScope
	return out
}

// Scope returns the current scope for handlers that need to inspect
// or extend it.
func (ev *Evaluator) Scope() *Scope {
	return ev.scope
}

func getExpr(n *parse.Node) parse.Expr {
	if n.TemplateData != nil {
		return n.TemplateData.Expr
	}
	return nil
}

func evaluateErr(format string, args ...any) *errors.Error {
	return errors.New(errors.ErrTemplateRender, fmt.Sprintf(format, args...), http.StatusInternalServerError)
}

func evaluateWrap(err error, format string, args ...any) *errors.Error {
	return errors.Wrap(err, errors.ErrTemplateRender, fmt.Sprintf(format, args...), http.StatusInternalServerError)
}
