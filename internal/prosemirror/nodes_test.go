package prosemirror

import (
	"encoding/json"
	"html/template"
	"testing"
)

func children(s string) func() template.HTML {
	return func() template.HTML { return template.HTML(s) }
}

func TestParagraph(t *testing.T) {
	tests := []struct {
		name     string
		children string
		want     string
	}{
		{"with text", "Hello world", "<p>Hello world</p>"},
		{"empty", "", "<p></p>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Paragraph(Node{}, children(tt.children))
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHeading(t *testing.T) {
	tests := []struct {
		name  string
		attrs *HeadingAttrs
		want  string
	}{
		{"level 1", &HeadingAttrs{Level: 1}, "<h1>Title</h1>"},
		{"level 2", &HeadingAttrs{Level: 2}, "<h2>Title</h2>"},
		{"level 3", &HeadingAttrs{Level: 3}, "<h3>Title</h3>"},
		{"level 4", &HeadingAttrs{Level: 4}, "<h4>Title</h4>"},
		{"level 5", &HeadingAttrs{Level: 5}, "<h5>Title</h5>"},
		{"level 6", &HeadingAttrs{Level: 6}, "<h6>Title</h6>"},
		{"no attrs defaults to h1", nil, "<h1>Title</h1>"},
		{"invalid level defaults to h1", &HeadingAttrs{Level: 99}, "<h1>Title</h1>"},
		{"zero level defaults to h1", &HeadingAttrs{Level: 0}, "<h1>Title</h1>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{}
			if tt.attrs != nil {
				node.Attrs, _ = json.Marshal(tt.attrs)
			}
			got := Heading(node, children("Title"))
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBlockquote(t *testing.T) {
	got := Blockquote(Node{}, children("<p>Quoted text</p>"))
	want := "<blockquote><p>Quoted text</p></blockquote>"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		attrs    *CodeBlockAttrs
		children string
		want     string
	}{
		{"no language", nil, "fmt.Println()", "<pre><code>fmt.Println()</code></pre>"},
		{"with language", &CodeBlockAttrs{Params: "go"}, "fmt.Println()", `<pre><code class="language-go">fmt.Println()</code></pre>`},
		{"language escaped", &CodeBlockAttrs{Params: `a"b`}, "x", `<pre><code class="language-a&#34;b">x</code></pre>`},
		{"empty", nil, "", "<pre><code></code></pre>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{}
			if tt.attrs != nil {
				node.Attrs, _ = json.Marshal(tt.attrs)
			}
			got := CodeBlock(node, children(tt.children))
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHorizontalRule(t *testing.T) {
	got := HorizontalRule(Node{}, nil)
	if string(got) != "<hr>" {
		t.Errorf("got %q, want %q", got, "<hr>")
	}
}

func TestImage(t *testing.T) {
	tests := []struct {
		name string
		attrs ImageAttrs
		want  string
	}{
		{"all attributes", ImageAttrs{Src: "photo.jpg", Alt: "A photo", Title: "My photo"}, `<img src="photo.jpg" alt="A photo" title="My photo">`},
		{"src only", ImageAttrs{Src: "photo.jpg"}, `<img src="photo.jpg">`},
		{"escapes src", ImageAttrs{Src: `x" onerror="alert(1)`}, `<img src="x&#34; onerror=&#34;alert(1)">`},
		{"escapes alt", ImageAttrs{Src: "x.jpg", Alt: `<script>`}, `<img src="x.jpg" alt="&lt;script&gt;">`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs, _ := json.Marshal(tt.attrs)
			got := Image(Node{Attrs: attrs}, nil)
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHardBreak(t *testing.T) {
	got := HardBreak(Node{}, nil)
	if string(got) != "<br>" {
		t.Errorf("got %q, want %q", got, "<br>")
	}
}

func TestText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"plain text", "Hello world", "Hello world"},
		{"escapes HTML tags", `<script>alert("xss")</script>`, `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`},
		{"escapes ampersand", "Tom & Jerry", "Tom &amp; Jerry"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Text(Node{Text: tt.text}, nil)
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrderedList(t *testing.T) {
	tests := []struct {
		name     string
		attrs    *OrderedListAttrs
		children string
		want     string
	}{
		{"default start", nil, "<li>one</li>", "<ol><li>one</li></ol>"},
		{"start at 1 omits attr", &OrderedListAttrs{Order: 1}, "<li>one</li>", "<ol><li>one</li></ol>"},
		{"start at 5", &OrderedListAttrs{Order: 5}, "<li>five</li>", `<ol start="5"><li>five</li></ol>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{}
			if tt.attrs != nil {
				node.Attrs, _ = json.Marshal(tt.attrs)
			}
			got := OrderedList(node, children(tt.children))
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBulletList(t *testing.T) {
	got := BulletList(Node{}, children("<li>a</li><li>b</li>"))
	want := "<ul><li>a</li><li>b</li></ul>"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestListItem(t *testing.T) {
	got := ListItem(Node{}, children("<p>Item</p>"))
	want := "<li><p>Item</p></li>"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
