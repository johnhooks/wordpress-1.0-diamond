package prosemirror

// Node renderers for ProseMirror's standard schema. Each function is
// a standalone NodeRenderer that can be registered with a Serializer.
//
// Attribute unmarshalling errors are intentionally ignored — a node
// with missing or corrupt attrs renders with defaults rather than
// failing the entire document.

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"strings"
)

// Paragraph renders a <p> element.
func Paragraph(_ Node, children func() template.HTML) template.HTML {
	return "<p>" + children() + "</p>"
}

// Heading renders <h1> through <h6>.
func Heading(node Node, children func() template.HTML) template.HTML {
	attrs := HeadingAttrs{Level: 1}
	if node.Attrs != nil {
		json.Unmarshal(node.Attrs, &attrs)
	}
	level := attrs.Level
	if level < 1 || level > 6 {
		level = 1
	}
	tag := fmt.Sprintf("h%d", level)
	return template.HTML("<" + tag + ">" + string(children()) + "</" + tag + ">")
}

// Blockquote renders a <blockquote> element.
func Blockquote(_ Node, children func() template.HTML) template.HTML {
	return "<blockquote>" + children() + "</blockquote>"
}

// CodeBlock renders <pre><code>. If a params attribute is set, it
// becomes a language class on the <code> element (matching the
// ProseMirror markdown schema convention).
func CodeBlock(node Node, children func() template.HTML) template.HTML {
	var attrs CodeBlockAttrs
	if node.Attrs != nil {
		json.Unmarshal(node.Attrs, &attrs)
	}
	if attrs.Params != "" {
		lang := html.EscapeString(attrs.Params)
		return template.HTML(`<pre><code class="language-` + lang + `">` + string(children()) + "</code></pre>")
	}
	return "<pre><code>" + children() + "</code></pre>"
}

// HorizontalRule renders an <hr> element.
func HorizontalRule(_ Node, _ func() template.HTML) template.HTML {
	return "<hr>"
}

// Image renders an <img> element. Self-closing, ignores children.
func Image(node Node, _ func() template.HTML) template.HTML {
	var attrs ImageAttrs
	if node.Attrs != nil {
		json.Unmarshal(node.Attrs, &attrs)
	}
	var b strings.Builder
	b.WriteString(`<img src="`)
	b.WriteString(html.EscapeString(attrs.Src))
	b.WriteByte('"')
	if attrs.Alt != "" {
		b.WriteString(` alt="`)
		b.WriteString(html.EscapeString(attrs.Alt))
		b.WriteByte('"')
	}
	if attrs.Title != "" {
		b.WriteString(` title="`)
		b.WriteString(html.EscapeString(attrs.Title))
		b.WriteByte('"')
	}
	b.WriteByte('>')
	return template.HTML(b.String())
}

// HardBreak renders a <br> element.
func HardBreak(_ Node, _ func() template.HTML) template.HTML {
	return "<br>"
}

// Text renders a text node. The text is HTML-escaped.
func Text(node Node, _ func() template.HTML) template.HTML {
	return template.HTML(html.EscapeString(node.Text))
}

// OrderedList renders an <ol> element. If the order attribute is not
// 1, a start attribute is included.
func OrderedList(node Node, children func() template.HTML) template.HTML {
	attrs := OrderedListAttrs{Order: 1}
	if node.Attrs != nil {
		json.Unmarshal(node.Attrs, &attrs)
	}
	if attrs.Order != 1 {
		return template.HTML(fmt.Sprintf(`<ol start="%d">`, attrs.Order)) + children() + "</ol>"
	}
	return "<ol>" + children() + "</ol>"
}

// BulletList renders a <ul> element.
func BulletList(_ Node, children func() template.HTML) template.HTML {
	return "<ul>" + children() + "</ul>"
}

// ListItem renders an <li> element.
func ListItem(_ Node, children func() template.HTML) template.HTML {
	return "<li>" + children() + "</li>"
}
