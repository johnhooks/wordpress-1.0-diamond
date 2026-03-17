package prosemirror

// DefaultSerializer returns a Serializer configured with the standard
// blog node and mark types. This covers prosemirror-schema-basic,
// prosemirror-schema-list, and the code_block params extension from
// the prosemirror-markdown schema.
func DefaultSerializer() *Serializer {
	return &Serializer{
		Nodes: map[string]NodeRenderer{
			"paragraph":       Paragraph,
			"heading":         Heading,
			"blockquote":      Blockquote,
			"code_block":      CodeBlock,
			"horizontal_rule": HorizontalRule,
			"image":           Image,
			"hard_break":      HardBreak,
			"text":            Text,
			"ordered_list":    OrderedList,
			"bullet_list":     BulletList,
			"list_item":       ListItem,
		},
		Marks: map[string]MarkRenderer{
			"strong": Strong,
			"em":     Em,
			"code":   Code,
			"link":   Link,
		},
	}
}
