package prosemirror

import (
	"encoding/json"
	"html/template"
	"testing"
)

func TestSerializer(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want string
	}{
		// Basic nodes
		{
			"paragraph with text",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`,
			"<p>Hello world</p>",
		},
		{
			"multiple paragraphs",
			`{"type":"doc","content":[
				{"type":"paragraph","content":[{"type":"text","text":"First"}]},
				{"type":"paragraph","content":[{"type":"text","text":"Second"}]}
			]}`,
			"<p>First</p><p>Second</p>",
		},
		{
			"empty paragraph",
			`{"type":"doc","content":[{"type":"paragraph"}]}`,
			"<p></p>",
		},
		{
			"empty document",
			`{"type":"doc","content":[]}`,
			"",
		},
		{
			"null content",
			`{"type":"doc"}`,
			"",
		},

		// Headings
		{
			"heading h2 and paragraph",
			`{"type":"doc","content":[
				{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Section"}]},
				{"type":"paragraph","content":[{"type":"text","text":"Body text"}]}
			]}`,
			"<h2>Section</h2><p>Body text</p>",
		},
		{
			"multiple headings (h1 and h2)",
			`{"type":"doc","content":[
				{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"one"}]},
				{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"two"}]},
				{"type":"paragraph","content":[{"type":"text","text":"text"}]}
			]}`,
			"<h1>one</h1><h2>two</h2><p>text</p>",
		},

		// Inline formatting (marks)
		{
			"bold text",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","text":"Hello "},
				{"type":"text","marks":[{"type":"strong"}],"text":"world"}
			]}]}`,
			"<p>Hello <strong>world</strong></p>",
		},
		{
			"nested marks (em inside strong)",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","marks":[{"type":"em"},{"type":"strong"}],"text":"bold italic"}
			]}]}`,
			"<p><strong><em>bold italic</em></strong></p>",
		},
		{
			"triple marks (code+em+strong)",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","marks":[{"type":"code"},{"type":"em"},{"type":"strong"}],"text":"all three"}
			]}]}`,
			"<p><strong><em><code>all three</code></em></strong></p>",
		},
		{
			"link with bold",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","marks":[
					{"type":"strong"},
					{"type":"link","attrs":{"href":"https://example.com"}}
				],"text":"bold link"}
			]}]}`,
			`<p><a href="https://example.com"><strong>bold link</strong></a></p>`,
		},
		{
			"joining styles — marks at different boundaries",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","text":"one"},
				{"type":"text","marks":[{"type":"strong"}],"text":"two"},
				{"type":"text","marks":[{"type":"em"},{"type":"strong"}],"text":"three"},
				{"type":"text","marks":[{"type":"em"}],"text":"four"},
				{"type":"text","text":"five"}
			]}]}`,
			"<p>one<strong>two</strong><strong><em>three</em></strong><em>four</em>five</p>",
		},
		{
			"inline code with emphasis",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","text":"text and "},
				{"type":"text","marks":[{"type":"code"}],"text":"code that is "},
				{"type":"text","marks":[{"type":"code"},{"type":"em"}],"text":"emphasized"},
				{"type":"text","marks":[{"type":"code"}],"text":"..."}
			]}]}`,
			"<p>text and <code>code that is </code><em><code>emphasized</code></em><code>...</code></p>",
		},

		// Leaf nodes
		{
			"horizontal rule between paragraphs",
			`{"type":"doc","content":[
				{"type":"paragraph","content":[{"type":"text","text":"Before"}]},
				{"type":"horizontal_rule"},
				{"type":"paragraph","content":[{"type":"text","text":"After"}]}
			]}`,
			"<p>Before</p><hr><p>After</p>",
		},
		{
			"document with only leaf nodes",
			`{"type":"doc","content":[{"type":"horizontal_rule"},{"type":"horizontal_rule"}]}`,
			"<hr><hr>",
		},
		{
			"image inline with text",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","text":"hi"},
				{"type":"image","attrs":{"src":"img.png","alt":"x"}},
				{"type":"text","text":"there"}
			]}]}`,
			`<p>hi<img src="img.png" alt="x">there</p>`,
		},
		{
			"hard break in paragraph",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","text":"Line one"},
				{"type":"hard_break"},
				{"type":"text","text":"Line two"}
			]}]}`,
			"<p>Line one<br>Line two</p>",
		},
		{
			"leaf nodes in marks — hard break inside em",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","marks":[{"type":"em"}],"text":"hi"},
				{"type":"hard_break","marks":[{"type":"em"}]},
				{"type":"text","marks":[{"type":"em"}],"text":"x"}
			]}]}`,
			"<p><em>hi</em><em><br></em><em>x</em></p>",
		},

		// Block containers
		{
			"blockquote",
			`{"type":"doc","content":[
				{"type":"blockquote","content":[
					{"type":"paragraph","content":[{"type":"text","text":"Quoted"}]}
				]}
			]}`,
			"<blockquote><p>Quoted</p></blockquote>",
		},
		{
			"blockquote with multiple paragraphs",
			`{"type":"doc","content":[
				{"type":"blockquote","content":[
					{"type":"paragraph","content":[{"type":"text","text":"hello"}]},
					{"type":"paragraph","content":[{"type":"text","text":"bye"}]}
				]}
			]}`,
			"<blockquote><p>hello</p><p>bye</p></blockquote>",
		},
		{
			"deeply nested blockquotes",
			`{"type":"doc","content":[
				{"type":"blockquote","content":[
					{"type":"blockquote","content":[
						{"type":"blockquote","content":[
							{"type":"paragraph","content":[{"type":"text","text":"he said"}]}
						]},
						{"type":"paragraph","content":[{"type":"text","text":"i said"}]}
					]}
				]}
			]}`,
			"<blockquote><blockquote><blockquote><p>he said</p></blockquote><p>i said</p></blockquote></blockquote>",
		},
		{
			"code block inside blockquote",
			`{"type":"doc","content":[
				{"type":"blockquote","content":[
					{"type":"code_block","content":[{"type":"text","text":"some code"}]}
				]},
				{"type":"paragraph","content":[{"type":"text","text":"and"}]}
			]}`,
			"<blockquote><pre><code>some code</code></pre></blockquote><p>and</p>",
		},

		// Code blocks
		{
			"code block with language",
			`{"type":"doc","content":[
				{"type":"code_block","attrs":{"params":"go"},"content":[
					{"type":"text","text":"fmt.Println(\"hello\")"}
				]}
			]}`,
			`<pre><code class="language-go">fmt.Println(&#34;hello&#34;)</code></pre>`,
		},

		// Lists
		{
			"bullet list",
			`{"type":"doc","content":[
				{"type":"bullet_list","content":[
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Alpha"}]}]},
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Beta"}]}]}
				]}
			]}`,
			"<ul><li><p>Alpha</p></li><li><p>Beta</p></li></ul>",
		},
		{
			"ordered list with start",
			`{"type":"doc","content":[
				{"type":"ordered_list","attrs":{"order":3},"content":[
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Third"}]}]},
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Fourth"}]}]}
				]}
			]}`,
			`<ol start="3"><li><p>Third</p></li><li><p>Fourth</p></li></ol>`,
		},
		{
			"list with marks on last item",
			`{"type":"doc","content":[
				{"type":"bullet_list","content":[
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]},
					{"type":"list_item","content":[{"type":"paragraph","content":[
						{"type":"text","text":"three"},
						{"type":"text","marks":[{"type":"strong"}],"text":"!"}
					]}]}
				]},
				{"type":"paragraph","content":[{"type":"text","text":"after"}]}
			]}`,
			"<ul><li><p>one</p></li><li><p>two</p></li><li><p>three<strong>!</strong></p></li></ul><p>after</p>",
		},
		{
			"nested lists — bullet inside ordered",
			`{"type":"doc","content":[
				{"type":"ordered_list","content":[
					{"type":"list_item","content":[
						{"type":"paragraph","content":[{"type":"text","text":"First"}]},
						{"type":"bullet_list","content":[
							{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Sub A"}]}]},
							{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Sub B"}]}]}
						]}
					]},
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Second"}]}]}
				]}
			]}`,
			"<ol><li><p>First</p><ul><li><p>Sub A</p></li><li><p>Sub B</p></li></ul></li><li><p>Second</p></li></ol>",
		},
		{
			"blockquote containing list",
			`{"type":"doc","content":[
				{"type":"blockquote","content":[
					{"type":"bullet_list","content":[
						{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Nested"}]}]}
					]}
				]}
			]}`,
			"<blockquote><ul><li><p>Nested</p></li></ul></blockquote>",
		},

		// Unknown types
		{
			"unknown node renders children",
			`{"type":"doc","content":[
				{"type":"custom_widget","content":[
					{"type":"paragraph","content":[{"type":"text","text":"Inside"}]}
				]}
			]}`,
			"<p>Inside</p>",
		},
		{
			"unknown mark ignored — text preserved",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","marks":[{"type":"strikethrough"}],"text":"struck"}
			]}]}`,
			"<p>struck</p>",
		},

		// XSS
		{
			"escapes HTML in text nodes",
			`{"type":"doc","content":[{"type":"paragraph","content":[
				{"type":"text","text":"<img src=x onerror=alert(1)>"}
			]}]}`,
			"<p>&lt;img src=x onerror=alert(1)&gt;</p>",
		},

		// Full blog post integration
		{
			"full blog post",
			`{"type":"doc","content":[
				{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"My Post"}]},
				{"type":"paragraph","content":[
					{"type":"text","text":"This is "},
					{"type":"text","marks":[{"type":"strong"}],"text":"important"},
					{"type":"text","text":". Visit "},
					{"type":"text","marks":[{"type":"link","attrs":{"href":"https://example.com"}}],"text":"here"},
					{"type":"text","text":"."}
				]},
				{"type":"blockquote","content":[
					{"type":"paragraph","content":[{"type":"text","text":"A wise quote"}]}
				]},
				{"type":"code_block","attrs":{"params":"go"},"content":[
					{"type":"text","text":"fmt.Println()"}
				]},
				{"type":"bullet_list","content":[
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"One"}]}]},
					{"type":"list_item","content":[{"type":"paragraph","content":[{"type":"text","text":"Two"}]}]}
				]},
				{"type":"horizontal_rule"},
				{"type":"paragraph","content":[{"type":"text","text":"The end."}]}
			]}`,
			"<h1>My Post</h1>" +
				`<p>This is <strong>important</strong>. Visit <a href="https://example.com">here</a>.</p>` +
				"<blockquote><p>A wise quote</p></blockquote>" +
				`<pre><code class="language-go">fmt.Println()</code></pre>` +
				"<ul><li><p>One</p></li><li><p>Two</p></li></ul>" +
				"<hr>" +
				"<p>The end.</p>",
		},
	}

	s := DefaultSerializer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Render(json.RawMessage(tt.doc))
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestSerializerErrors(t *testing.T) {
	tests := []struct {
		name string
		doc  string
	}{
		{"invalid JSON", `not json`},
		{"wrong root type", `{"type":"paragraph"}`},
	}

	s := DefaultSerializer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Render(json.RawMessage(tt.doc))
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestSerializerOmitMark(t *testing.T) {
	s := DefaultSerializer()
	s.Marks["em"] = nil

	doc := `{"type":"doc","content":[{"type":"paragraph","content":[
		{"type":"text","text":"foo"},
		{"type":"text","marks":[{"type":"em"}],"text":"bar"},
		{"type":"text","marks":[{"type":"strong"}],"text":"baz"}
	]}]}`
	got, err := s.Render(json.RawMessage(doc))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := template.HTML("<p>foobar<strong>baz</strong></p>")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSerializerOverrideRenderers(t *testing.T) {
	t.Run("custom node", func(t *testing.T) {
		s := DefaultSerializer()
		s.Nodes["callout"] = func(_ Node, children func() template.HTML) template.HTML {
			return `<div class="callout">` + children() + "</div>"
		}
		doc := `{"type":"doc","content":[
			{"type":"callout","content":[
				{"type":"paragraph","content":[{"type":"text","text":"Note!"}]}
			]}
		]}`
		got, err := s.Render(json.RawMessage(doc))
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if string(got) != `<div class="callout"><p>Note!</p></div>` {
			t.Errorf("got %q", got)
		}
	})

	t.Run("override paragraph", func(t *testing.T) {
		s := DefaultSerializer()
		s.Nodes["paragraph"] = func(_ Node, children func() template.HTML) template.HTML {
			return `<p class="custom">` + children() + "</p>"
		}
		doc := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"styled"}]}]}`
		got, err := s.Render(json.RawMessage(doc))
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if string(got) != `<p class="custom">styled</p>` {
			t.Errorf("got %q", got)
		}
	})

	t.Run("override mark", func(t *testing.T) {
		s := DefaultSerializer()
		s.Marks["strong"] = func(_ Mark, content template.HTML) template.HTML {
			return "<b>" + content + "</b>"
		}
		doc := `{"type":"doc","content":[{"type":"paragraph","content":[
			{"type":"text","marks":[{"type":"strong"}],"text":"bold"}
		]}]}`
		got, err := s.Render(json.RawMessage(doc))
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if string(got) != "<p><b>bold</b></p>" {
			t.Errorf("got %q", got)
		}
	})
}
