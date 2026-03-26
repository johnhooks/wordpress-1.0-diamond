package server

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	handlers  map[string]template.TagHandler
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
		handlers: make(map[string]template.TagHandler),
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
func (th *Theme) RegisterHandler(name string, handler template.TagHandler) {
	th.handlers[name] = handler
}

// Render writes the document shell and renders the named page template
// within it. The engine owns the <head> (charset, viewport, title,
// theme stylesheet, htmx). Page templates are body-content fragments.
func (th *Theme) Render(w io.Writer, ctx context.Context, name string, data any) error {
	doc, err := th.getTemplate(name)
	if err != nil {
		return err
	}

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

	// Render the page template as body content.
	if err := template.Walk(w, ctx, doc, data, th.handlers, nil); err != nil {
		return err
	}

	// Close the document.
	_, err = io.WriteString(w, "\n</body>\n</html>\n")
	return err
}

// RenderTagScoped loads a sub-template by name, creates a child scope
// with data fields flattened as variables, merges engine and caller
// attributes into the wrapper element, and renders within the shaped
// scope. This is the primary rendering path for vocabulary tag handlers.
func (th *Theme) RenderTagScoped(
	ctx context.Context,
	parentScope *template.Scope,
	name string,
	callerAttrs map[string]string,
	engineAttrs map[string]string,
	data any,
) (string, error) {
	doc, err := th.getTemplate(name)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrTemplateRender, fmt.Sprintf("tag template %q", name), http.StatusInternalServerError)
	}

	// Find the wrapper element (first element child of the document).
	wrapper := firstElementChild(doc)
	if wrapper != nil {
		// Clone the attribute slice to prevent mutation of cached ASTs.
		cloned := make([]parse.Attribute, len(wrapper.Attr))
		copy(cloned, wrapper.Attr)
		wrapper.Attr = cloned

		// Engine attributes first (id, role, hx-*), then caller overrides.
		if len(engineAttrs) > 0 {
			mergeAttrs(wrapper, engineAttrs)
		}
		if len(callerAttrs) > 0 {
			mergeAttrs(wrapper, callerAttrs)
		}
	}

	child := parentScope.PushData(data)

	var buf strings.Builder
	if err := template.WalkWithScope(&buf, ctx, doc, child, th.handlers); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (th *Theme) getTemplate(name string) (*parse.Node, error) {
	if th.devMode {
		// Search molecules → organisms → templates in order.
		for _, sub := range themeDirs {
			path := filepath.Join(th.dir, sub, name+".html")
			if _, err := os.Stat(path); err == nil {
				return parseFile(path)
			}
		}
		return nil, errors.NotFound(errors.ErrNotFound, fmt.Sprintf("template %q not found", name))
	}

	doc, ok := th.templates[name]
	if !ok {
		return nil, errors.NotFound(errors.ErrNotFound, fmt.Sprintf("template %q not found", name))
	}
	return doc, nil
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
	for key, val := range attrs {
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
