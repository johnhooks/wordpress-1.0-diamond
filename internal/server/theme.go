package server

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"press/internal/errors"
	"press/internal/template"
	"press/internal/template/parse"
)

// Theme holds parsed template ASTs and vocabulary tag handlers.
type Theme struct {
	dir       string
	devMode   bool
	templates map[string]*parse.Node
	handlers  map[string]template.TagNodeHandler
}

// themeDirs is the search order for template directories.
// Molecules are loaded first so vocabulary tags resolve to component
// templates. Organisms compose molecules. Page templates come last.
var themeDirs = []string{"molecules", "organisms", "templates"}

// LoadTheme loads and parses all .html templates from molecules/,
// organisms/, and templates/ directories within dir. All templates
// are parsed as fragments (no document scaffolding). The engine owns
// the document shell and writes it in Render.
func LoadTheme(dir string, devMode bool) (*Theme, error) {
	th := &Theme{
		dir:      dir,
		devMode:  devMode,
		handlers: make(map[string]template.TagNodeHandler),
	}

	all := make(map[string]*parse.Node)
	for _, sub := range themeDirs {
		subdir := filepath.Join(dir, sub)
		if _, err := os.Stat(subdir); os.IsNotExist(err) {
			continue
		}
		templates, err := parseTemplates(subdir)
		if err != nil {
			return nil, err
		}
		for name, doc := range templates {
			if _, exists := all[name]; !exists {
				all[name] = doc
			}
		}
	}
	th.templates = all

	return th, nil
}

// RegisterHandler registers a vocabulary tag handler.
func (th *Theme) RegisterHandler(name string, handler template.TagNodeHandler) {
	th.handlers[name] = handler
}

// ResolveTag implements template.TagResolver.
func (th *Theme) ResolveTag(name string) template.TagNodeHandler {
	return th.handlers[name]
}

// Render writes the document shell and renders the named page template
// within it. The engine owns the <head> (charset, viewport, title,
// theme stylesheet, htmx). Page templates are body-content fragments.
func (th *Theme) Render(w io.Writer, ctx context.Context, name string, data any) error {
	doc := th.GetTemplate(name)

	// Extract title fields from the view data.
	blogName := lookupViewString(data, "blog_name")
	pageTitle := lookupViewString(data, "page_title")

	title := blogName
	if pageTitle != "" {
		title = pageTitle + " - " + blogName
	}

	// Write document shell.
	if _, err := fmt.Fprintf(w,
		"<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n"+
			"    <meta charset=\"utf-8\"/>\n"+
			"    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"/>\n"+
			"    <title>%s</title>\n"+
			"    <link rel=\"stylesheet\" href=\"/theme/style.css\"/>\n"+
			"    <script src=\"/static/vendor/htmx.org.js\" defer></script>\n"+
			"</head>\n<body>\n",
		html.EscapeString(title),
	); err != nil {
		return err
	}

	// Put the tag resolver on the context so the evaluator can
	// dispatch vocabulary tags to handlers.
	ctx = template.WithTagResolver(ctx, th)

	// Evaluate the page template into a plain HTML AST, then render.
	result := template.Evaluate(ctx, doc, data)
	if err := parse.Render(w, result); err != nil {
		return err
	}

	// Close the document.
	_, err := io.WriteString(w, "\n</body>\n</html>\n")
	return err
}

// EvalTagTemplate loads a sub-template by name, evaluates it with a
// child scope from data, merges engine and caller attributes onto the
// output wrapper element, and returns the result node tree.
func (th *Theme) EvalTagTemplate(
	ctx context.Context,
	ev *template.Evaluator,
	name string,
	callerAttrs map[string]string,
	engineAttrs map[string]string,
	data any,
) *parse.Node {
	// The engine owns "id" on vocabulary tags. Theme authors must not
	// set id in caller attributes or molecule templates. The engine
	// uses id for htmx OOB swap targeting.
	// TODO: add this rule to the theme validator so it reports a clear
	// error at load time instead of panicking at request time.
	if _, ok := callerAttrs["id"]; ok {
		panic(fmt.Sprintf("theme: vocabulary tag <%s> must not have an id attribute; id is reserved for the engine", name))
	}

	doc := th.GetTemplate(name)
	result := ev.EvaluateScoped(doc, data)

	// Reject id on the molecule's own wrapper element for the same
	// reason. The engine is the sole source of id on vocabulary output.
	wrapper := firstElementChild(result)
	if wrapper != nil {
		for _, a := range wrapper.Attr {
			if a.Key == "id" {
				panic(fmt.Sprintf("theme: molecule template %q must not have an id attribute on its wrapper element; id is reserved for the engine", name))
			}
		}
		if len(engineAttrs) > 0 {
			mergeAttrs(wrapper, engineAttrs)
		}
		if len(callerAttrs) > 0 {
			mergeAttrs(wrapper, callerAttrs)
		}
	}

	return result
}

// RenderTag renders a vocabulary tag by name for htmx partial responses.
// It creates an evaluator with data as the root scope, calls the
// registered tag handler (which injects engine attributes), and
// serializes the result. Both the full-page path and the partial path
// go through the same tag handler, so both get the same engine attrs.
func (th *Theme) RenderTag(w io.Writer, ctx context.Context, name string, data any) error {
	handler := th.ResolveTag(name)
	if handler == nil {
		panic(fmt.Sprintf("theme: RenderTag: no handler for %q", name))
	}

	ctx = template.WithTagResolver(ctx, th)
	ev := template.NewEvaluator(ctx, data)

	// Pass a nil element node since there are no caller attributes
	// on a direct invocation.
	result := handler(ctx, ev, &parse.Node{Type: parse.ElementNode, Data: name})
	if result == nil {
		return nil
	}
	return parse.Render(w, result)
}

// attrsFromNode extracts a map of attribute key-value pairs from a
// parsed element node. Used by tag handlers to pass caller attributes
// through to sub-templates.
func attrsFromNode(el *parse.Node) map[string]string {
	if len(el.Attr) == 0 {
		return nil
	}
	attrs := make(map[string]string, len(el.Attr))
	for _, a := range el.Attr {
		attrs[a.Key] = a.Val
	}
	return attrs
}

// GetTemplate returns the parsed AST for a template. Panics if the
// template does not exist. Templates are validated at load time; a
// missing template at request time is a programmer error.
func (th *Theme) GetTemplate(name string) *parse.Node {
	if th.devMode {
		// Search molecules → organisms → templates in order.
		for _, sub := range themeDirs {
			path := filepath.Join(th.dir, sub, name+".html")
			if _, err := os.Stat(path); err == nil {
				doc, err := parseFile(path)
				if err != nil {
					panic(fmt.Sprintf("theme: failed to parse template %q: %v", name, err))
				}
				return doc
			}
		}
		panic(fmt.Sprintf("theme: template %q not found", name))
	}

	doc, ok := th.templates[name]
	if !ok {
		panic(fmt.Sprintf("theme: template %q not found", name))
	}
	return doc
}

// lookupViewString resolves a view-tagged string field from a struct.
// Returns empty string if the field is not found or not a string.
func lookupViewString(data any, name string) string {
	val, err := template.LookupField(data, name)
	if err != nil {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// firstElementChild returns the first ElementNode child of a node,
// skipping text nodes, doctypes, and whitespace.
func firstElementChild(n *parse.Node) *parse.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == parse.ElementNode {
			return c
		}
	}
	return nil
}

// mergeAttrs merges caller attributes into an element node.
// The "class" attribute is concatenated (theme classes first,
// caller classes appended). All other attributes are set directly,
// with caller values overriding existing ones.
func mergeAttrs(n *parse.Node, attrs map[string]string) {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		val := attrs[key]
		if key == "class" {
			mergeClass(n, val)
			continue
		}
		setAttr(n, key, val)
	}
}

func mergeClass(n *parse.Node, callerClass string) {
	for i, attr := range n.Attr {
		if attr.Key == "class" {
			n.Attr[i].Val = attr.Val + " " + callerClass
			return
		}
	}
	n.Attr = append(n.Attr, parse.Attribute{Key: "class", Val: callerClass})
}

func setAttr(n *parse.Node, key, val string) {
	for i, attr := range n.Attr {
		if attr.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, parse.Attribute{Key: key, Val: val})
}

func parseTemplates(dir string) (map[string]*parse.Node, error) {
	pattern := filepath.Join(dir, "*.html")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrTemplateSyntax, fmt.Sprintf("globbing %s", pattern), http.StatusInternalServerError)
	}

	templates := make(map[string]*parse.Node, len(matches))
	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), ".html")
		doc, err := parseFile(path)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrTemplateSyntax, fmt.Sprintf("parsing %s", path), http.StatusInternalServerError)
		}
		templates[name] = doc
	}
	return templates, nil
}

func parseFile(path string) (*parse.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return parse.ParseTemplateFragment(f)
}
