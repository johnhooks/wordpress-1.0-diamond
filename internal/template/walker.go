package template

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"reflect"

	"press/internal/errors"
	"press/internal/template/parse"
)

// Walker walks a parsed template AST and renders HTML.
type Walker struct {
	w        io.Writer
	scope    *Scope
	handlers map[string]TagHandler
	snippets map[string]*Snippet
}

// TagHandler is a vocabulary tag handler. It receives a render
// context and resolved attributes, and returns an HTML fragment.
type TagHandler func(ctx *RenderContext) (string, error)

// RenderContext is passed to vocabulary tag handlers.
type RenderContext struct {
	Scope *Scope
	Attrs map[string]string
	Node  *parse.Node
}

// Snippet is a theme-defined reusable template fragment.
type Snippet struct {
	Name   string
	Params string
	Body   *parse.Node
}

// Walk renders a parsed template AST to the writer.
func Walk(w io.Writer, doc *parse.Node, data any, handlers map[string]TagHandler, snippets map[string]*Snippet) error {
	walker := &Walker{
		w:        w,
		scope:    NewScope(data),
		handlers: handlers,
		snippets: snippets,
	}
	return walker.walkChildren(doc)
}

// WalkWithScope renders a parsed template AST using an existing scope.
// Used by vocabulary tag handlers to render sub-templates while
// preserving the current scope chain (e.g. {each} bindings).
func WalkWithScope(w io.Writer, doc *parse.Node, scope *Scope, handlers map[string]TagHandler) error {
	walker := &Walker{
		w:        w,
		scope:    scope,
		handlers: handlers,
	}
	return walker.walkChildren(doc)
}

func (wk *Walker) walkNode(n *parse.Node) error {
	switch n.Type {
	case parse.DocumentNode:
		return wk.walkChildren(n)

	case parse.TextNode:
		_, err := io.WriteString(wk.w, html.EscapeString(n.Data))
		return err

	case parse.ElementNode:
		return wk.walkElement(n)

	case parse.ExpressionNode:
		return wk.walkExpression(n)

	case parse.IfNode:
		return wk.walkIf(n)

	case parse.EachNode:
		return wk.walkEach(n)

	case parse.ConstNode:
		return wk.walkConst(n)

	case parse.SnippetNode:
		// Snippet definitions are extracted at compile time,
		// not rendered inline.
		return nil

	case parse.CommentNode:
		_, err := fmt.Fprintf(wk.w, "<!--%s-->", n.Data)
		return err

	case parse.DoctypeNode:
		_, err := fmt.Fprintf(wk.w, "<!DOCTYPE %s>", n.Data)
		return err

	default:
		return renderErr("unknown node type %d", n.Type)
	}
}

func (wk *Walker) walkChildren(n *parse.Node) error {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := wk.walkNode(c); err != nil {
			return err
		}
	}
	return nil
}

func (wk *Walker) walkElement(n *parse.Node) error {
	// Check if this is a vocabulary tag with a handler.
	if handler, ok := wk.handlers[n.Data]; ok {
		return wk.walkVocabularyTag(n, handler)
	}

	// Write opening tag.
	if _, err := fmt.Fprintf(wk.w, "<%s", n.Data); err != nil {
		return err
	}

	// Write attributes with expression resolution.
	for _, attr := range n.Attr {
		val, err := wk.resolveAttr(attr)
		if err != nil {
			return renderWrap(err, "resolving attribute %q", attr.Key)
		}
		if _, err := fmt.Fprintf(wk.w, ` %s="%s"`, attr.Key, val); err != nil {
			return err
		}
	}

	// Void elements.
	if isVoidElement(n.Data) {
		_, err := io.WriteString(wk.w, "/>")
		return err
	}

	if _, err := io.WriteString(wk.w, ">"); err != nil {
		return err
	}

	// Walk children.
	if err := wk.walkChildren(n); err != nil {
		return err
	}

	// Write closing tag.
	_, err := fmt.Fprintf(wk.w, "</%s>", n.Data)
	return err
}

func (wk *Walker) walkVocabularyTag(n *parse.Node, handler TagHandler) error {
	// Resolve attributes.
	attrs := make(map[string]string, len(n.Attr))
	for _, attr := range n.Attr {
		val, err := wk.resolveAttr(attr)
		if err != nil {
			return renderWrap(err, "resolving attribute %q on <%s>", attr.Key, n.Data)
		}
		attrs[attr.Key] = val
	}

	ctx := &RenderContext{
		Scope: wk.scope,
		Attrs: attrs,
		Node:  n,
	}

	result, err := handler(ctx)
	if err != nil {
		return renderWrap(err, "handler for <%s>", n.Data)
	}

	_, err = io.WriteString(wk.w, result)
	return err
}

func (wk *Walker) walkExpression(n *parse.Node) error {
	expr := wk.getExpr(n)
	if expr == nil {
		return renderErr("expression {%s} not compiled", n.Data)
	}

	val, err := Eval(expr, wk.scopeContext())
	if err != nil {
		return renderWrap(err, "evaluating {%s}", n.Data)
	}

	// RawHTML values are pre-rendered and written without escaping.
	if raw, ok := val.(RawHTML); ok {
		_, err = io.WriteString(wk.w, string(raw))
		return err
	}

	_, err = io.WriteString(wk.w, html.EscapeString(formatValue(val)))
	return err
}

func (wk *Walker) walkIf(n *parse.Node) error {
	expr := wk.getExpr(n)
	if expr == nil {
		return renderErr("if condition {%s} not compiled", n.Data)
	}

	truthy, err := EvalTruthy(expr, wk.scopeContext())
	if err != nil {
		return renderWrap(err, "evaluating if condition {%s}", n.Data)
	}

	if truthy {
		// Walk then-branch: all children before ElseNode.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				break
			}
			if err := wk.walkNode(c); err != nil {
				return err
			}
		}
	} else {
		// Walk else-branch: children of ElseNode.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				return wk.walkChildren(c)
			}
		}
	}
	return nil
}

func (wk *Walker) walkEach(n *parse.Node) error {
	if n.TemplateData == nil {
		return renderErr("each node missing template data")
	}

	expr := wk.getExpr(n)
	if expr == nil {
		return renderErr("each list {%s} not compiled", n.Data)
	}

	listVal, err := Eval(expr, wk.scopeContext())
	if err != nil {
		return renderWrap(err, "evaluating each list {%s}", n.Data)
	}

	// Check if list is iterable.
	rv := reflect.ValueOf(listVal)
	empty := !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) || rv.Len() == 0

	if empty {
		// Walk the else branch if present.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				return wk.walkChildren(c)
			}
		}
		return nil
	}

	binding := n.TemplateData.Binding
	indexBinding := n.TemplateData.IndexBinding

	for i := 0; i < rv.Len(); i++ {
		child := wk.scope.Push()
		if binding != "" {
			child.Set(binding, rv.Index(i).Interface())
		}
		if indexBinding != "" {
			child.Set(indexBinding, i)
		}

		oldScope := wk.scope
		wk.scope = child

		// Walk body children (before ElseNode).
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == parse.ElseNode {
				break
			}
			if err := wk.walkNode(c); err != nil {
				wk.scope = oldScope
				return err
			}
		}

		wk.scope = oldScope
	}
	return nil
}

func (wk *Walker) walkConst(n *parse.Node) error {
	if n.TemplateData == nil {
		return renderErr("const node missing template data")
	}

	expr := wk.getExpr(n)
	if expr == nil {
		return renderErr("const expression {%s} not compiled", n.Data)
	}

	val, err := Eval(expr, wk.scopeContext())
	if err != nil {
		return renderWrap(err, "evaluating const {%s}", n.Data)
	}

	wk.scope.Set(n.TemplateData.ConstName, val)
	return nil
}

// getExpr returns the pre-compiled expression from a node's TemplateData.
func (wk *Walker) getExpr(n *parse.Node) parse.Expr {
	if n.TemplateData != nil {
		return n.TemplateData.Expr
	}
	return nil
}

// resolveAttr evaluates an attribute value. If the attribute was
// pre-compiled (Parts is set), it evaluates the parts directly.
// Otherwise it returns the static value as-is.
func (wk *Walker) resolveAttr(attr parse.Attribute) (string, error) {
	if attr.Parts == nil {
		return attr.Val, nil
	}
	return EvalAttrParts(attr.Parts, wk.scopeContext())
}

// scopeContext returns a context object that the expression evaluator
// can use. It wraps the scope so that identifier lookup checks scope
// bindings first, then falls through to the page data.
func (wk *Walker) scopeContext() *Scope {
	return wk.scope
}

// HTML void elements that should not have a closing tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

func isVoidElement(tag string) bool {
	return voidElements[tag]
}

// renderErr creates an *errors.Error with the template_render code.
func renderErr(format string, args ...any) *errors.Error {
	return errors.New(errors.ErrTemplateRender, fmt.Sprintf(format, args...), http.StatusInternalServerError)
}

// renderWrap wraps an existing error with the template_render code.
func renderWrap(err error, format string, args ...any) *errors.Error {
	return errors.Wrap(err, errors.ErrTemplateRender, fmt.Sprintf(format, args...), http.StatusInternalServerError)
}
