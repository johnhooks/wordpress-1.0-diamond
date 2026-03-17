package prosemirror

import "encoding/json"

// Document represents a ProseMirror document root node.
type Document struct {
	Type    string `json:"type"`
	Content []Node `json:"content,omitempty"`
}

// Node represents any node in a ProseMirror document tree.
type Node struct {
	Type    string          `json:"type"`
	Content []Node          `json:"content,omitempty"`
	Attrs   json.RawMessage `json:"attrs,omitempty"`
	Marks   []Mark          `json:"marks,omitempty"`
	Text    string          `json:"text,omitempty"`
}

// Mark represents a ProseMirror mark (inline formatting).
type Mark struct {
	Type  string          `json:"type"`
	Attrs json.RawMessage `json:"attrs,omitempty"`
}

// HeadingAttrs holds attributes for a heading node.
type HeadingAttrs struct {
	Level int `json:"level"`
}

// ImageAttrs holds attributes for an image node.
type ImageAttrs struct {
	Src   string `json:"src"`
	Alt   string `json:"alt,omitempty"`
	Title string `json:"title,omitempty"`
}

// LinkAttrs holds attributes for a link mark.
type LinkAttrs struct {
	Href  string `json:"href"`
	Title string `json:"title,omitempty"`
}

// OrderedListAttrs holds attributes for an ordered list node.
type OrderedListAttrs struct {
	Order int `json:"order"`
}

// CodeBlockAttrs holds attributes for a code block node.
type CodeBlockAttrs struct {
	Params string `json:"params,omitempty"`
}
