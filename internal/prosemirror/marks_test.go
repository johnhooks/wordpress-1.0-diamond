package prosemirror

import (
	"encoding/json"
	"testing"
)

func TestStrong(t *testing.T) {
	got := Strong(Mark{}, "bold text")
	if string(got) != "<strong>bold text</strong>" {
		t.Errorf("got %q", got)
	}
}

func TestEm(t *testing.T) {
	got := Em(Mark{}, "italic text")
	if string(got) != "<em>italic text</em>" {
		t.Errorf("got %q", got)
	}
}

func TestCode(t *testing.T) {
	got := Code(Mark{}, "inline code")
	if string(got) != "<code>inline code</code>" {
		t.Errorf("got %q", got)
	}
}

func TestLink(t *testing.T) {
	tests := []struct {
		name  string
		attrs *LinkAttrs
		want  string
	}{
		{"with title", &LinkAttrs{Href: "https://example.com", Title: "Example"}, `<a href="https://example.com" title="Example">click</a>`},
		{"no title", &LinkAttrs{Href: "https://example.com"}, `<a href="https://example.com">click</a>`},
		{"escapes href", &LinkAttrs{Href: `https://example.com/a"b`}, `<a href="https://example.com/a&#34;b">click</a>`},
		{"no attrs", nil, `<a href="">click</a>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mark := Mark{}
			if tt.attrs != nil {
				mark.Attrs, _ = json.Marshal(tt.attrs)
			}
			got := Link(mark, "click")
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
