package prosemirror

// Mark renderers for ProseMirror's standard schema. Attribute
// unmarshalling errors are intentionally ignored — a mark with
// missing attrs renders with defaults.

import (
	"encoding/json"
	"html"
	"html/template"
	"strings"
)

// Strong renders a <strong> element.
func Strong(_ Mark, content template.HTML) template.HTML {
	return "<strong>" + content + "</strong>"
}

// Em renders an <em> element.
func Em(_ Mark, content template.HTML) template.HTML {
	return "<em>" + content + "</em>"
}

// Code renders an inline <code> element.
func Code(_ Mark, content template.HTML) template.HTML {
	return "<code>" + content + "</code>"
}

// Link renders an <a> element.
func Link(mark Mark, content template.HTML) template.HTML {
	var attrs LinkAttrs
	if mark.Attrs != nil {
		json.Unmarshal(mark.Attrs, &attrs)
	}
	var b strings.Builder
	b.WriteString(`<a href="`)
	b.WriteString(html.EscapeString(attrs.Href))
	b.WriteByte('"')
	if attrs.Title != "" {
		b.WriteString(` title="`)
		b.WriteString(html.EscapeString(attrs.Title))
		b.WriteByte('"')
	}
	b.WriteByte('>')
	b.WriteString(string(content))
	b.WriteString("</a>")
	return template.HTML(b.String())
}
