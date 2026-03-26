package parse

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseExpr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Identifiers
		{"bare ident", "name", "name", false},
		{"underscore ident", "_private", "_private", false},

		// Member expressions
		{"single dot", "post.Title", "post.Title", false},
		{"double dot", "post.Author.Name", "post.Author.Name", false},
		{"triple dot", "a.b.c.d", "a.b.c.d", false},

		// Literals
		{"string literal", `"hello"`, `"hello"`, false},
		{"string with escape", `"say \"hi\""`, `"say \"hi\""`, false},
		{"integer", "42", "42", false},
		{"float", "3.14", "3.14", false},
		{"true", "true", "true", false},
		{"false", "false", "false", false},

		// Comparisons
		{"equals", `post.Type == "post"`, `post.Type == "post"`, false},
		{"not equals", "count != 0", "count != 0", false},
		{"greater than", "count > 0", "count > 0", false},
		{"less than", "count < 10", "count < 10", false},
		{"greater or equal", "count >= 1", "count >= 1", false},
		{"less or equal", "count <= 100", "count <= 100", false},

		// Logical
		{"negation bang", "!commentsOpen", "not commentsOpen", false},
		{"negation keyword", "not commentsOpen", "not commentsOpen", false},
		{"double negation", "!!value", "not not value", false},
		{"and symbols", "a && b", "a and b", false},
		{"and keyword", "a and b", "a and b", false},
		{"or symbols", "a || b", "a or b", false},
		{"or keyword", "a or b", "a or b", false},
		{"and chain", "a && b && c", "a and b and c", false},
		{"and chain keyword", "a and b and c", "a and b and c", false},

		// Combined
		{"negation with member", "!post.EditURL", "not post.EditURL", false},
		{"comparison with member", "post.CommentCount > 0", "post.CommentCount > 0", false},

		// Whitespace
		{"leading space", "  name", "name", false},
		{"trailing space", "name  ", "name", false},
		{"spaces around op", "a  ==  b", "a == b", false},

		// Keyword boundaries
		{"not prefix ident", "nothing", "nothing", false},
		{"and prefix ident", "android", "android", false},
		{"or prefix ident", "order", "order", false},

		// Numbers
		{"negative-looking number", "0", "0", false},
		{"large integer", "9999", "9999", false},

		// Errors
		{"empty", "", "", true},
		{"trailing dot", "post.", "", true},
		{"double dot no field", "post..", "", true},
		{"unterminated string", `"hello`, "", true},
		{"chained comparison", "a < b < c", "", true},
		{"multiple dots in number", "3.14.15", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpr(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q, got %v", tt.input, expr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := expr.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseExpr_Types(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{"ident", "name", "*parse.Ident"},
		{"member", "post.Title", "*parse.MemberExpr"},
		{"deep member", "a.b.c", "*parse.MemberExpr"},
		{"string", `"hello"`, "*parse.StringLit"},
		{"number", "42", "*parse.NumberLit"},
		{"bool true", "true", "*parse.BoolLit"},
		{"bool false", "false", "*parse.BoolLit"},
		{"binary", "a == b", "*parse.BinaryExpr"},
		{"unary", "!a", "*parse.UnaryExpr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpr(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := typeName(expr)
			if got != tt.wantType {
				t.Errorf("got type %s, want %s", got, tt.wantType)
			}
		})
	}
}

func TestParseExpr_MemberStructure(t *testing.T) {
	expr, err := ParseExpr("post.Author.Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be MemberExpr{ Object: MemberExpr{ Object: Ident{post}, Field: Author }, Field: Name }
	outer, ok := expr.(*MemberExpr)
	if !ok {
		t.Fatalf("expected MemberExpr, got %T", expr)
	}
	if outer.Field != "Name" {
		t.Errorf("outer.Field = %q, want \"Name\"", outer.Field)
	}

	inner, ok := outer.Object.(*MemberExpr)
	if !ok {
		t.Fatalf("expected inner MemberExpr, got %T", outer.Object)
	}
	if inner.Field != "Author" {
		t.Errorf("inner.Field = %q, want \"Author\"", inner.Field)
	}

	root, ok := inner.Object.(*Ident)
	if !ok {
		t.Fatalf("expected root Ident, got %T", inner.Object)
	}
	if root.Name != "post" {
		t.Errorf("root.Name = %q, want \"post\"", root.Name)
	}
}

func TestParseExpr_DepthLimit(t *testing.T) {
	// Build a deeply nested grouped expression that exceeds maxExprDepth.
	input := strings.Repeat("(", 40) + "x" + strings.Repeat(")", 40)
	_, err := ParseExpr(input)
	if err == nil {
		t.Error("expected depth limit error for deeply nested expression")
	}
	if err != nil && !strings.Contains(err.Error(), "nesting too deep") {
		t.Errorf("expected nesting error, got: %v", err)
	}
}

func TestClassifyTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKind TemplateTokenKind
		wantRest string
	}{
		{"expression", "post.Title", ExprToken, "post.Title"},
		{"if", "if commentsOpen", IfOpenToken, "commentsOpen"},
		{"if with comparison", "if count > 0", IfOpenToken, "count > 0"},
		{"else", "else", ElseToken, ""},
		{"close if", "/if", IfCloseToken, ""},
		{"each", "each posts as post", EachOpenToken, "posts as post"},
		{"each with index", "each posts as post, i", EachOpenToken, "posts as post, i"},
		{"close each", "/each", EachCloseToken, ""},
		{"snippet", "snippet field(props)", SnippetOpenToken, "field(props)"},
		{"close snippet", "/snippet", SnippetCloseToken, ""},
		{"const", "const area = width * height", ConstToken, "area = width * height"},
		{"whitespace", "  post.Title  ", ExprToken, "post.Title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, rest := ClassifyTemplate(tt.input)
			if kind != tt.wantKind {
				t.Errorf("kind = %d, want %d", kind, tt.wantKind)
			}
			if rest != tt.wantRest {
				t.Errorf("rest = %q, want %q", rest, tt.wantRest)
			}
		})
	}
}

func TestParseEachBinding(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantList    string
		wantBinding string
		wantIndex   string
	}{
		{"simple", "posts as post", "posts", "post", ""},
		{"with index", "posts as post, i", "posts", "post", "i"},
		{"member list", "site.Categories as cat", "site.Categories", "cat", ""},
		{"no as", "posts", "posts", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, binding, index := ParseEachBinding(tt.input)
			if list != tt.wantList {
				t.Errorf("list = %q, want %q", list, tt.wantList)
			}
			if binding != tt.wantBinding {
				t.Errorf("binding = %q, want %q", binding, tt.wantBinding)
			}
			if index != tt.wantIndex {
				t.Errorf("index = %q, want %q", index, tt.wantIndex)
			}
		})
	}
}

func TestParseSnippetDecl(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantName   string
		wantParams string
	}{
		{"with params", "field(props)", "field", "props"},
		{"no params", "footer", "footer", ""},
		{"empty parens", "header()", "header", ""},
		{"multi params", "row(item, index)", "row", "item, index"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, params := ParseSnippetDecl(tt.input)
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if params != tt.wantParams {
				t.Errorf("params = %q, want %q", params, tt.wantParams)
			}
		})
	}
}

func TestParseConstDecl(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantExpr string
	}{
		{"simple", "count = 42", "count", "42"},
		{"expression", "hasComments = post.CommentCount > 0", "hasComments", "post.CommentCount > 0"},
		{"no value", "orphan", "orphan", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, expr := ParseConstDecl(tt.input)
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if expr != tt.wantExpr {
				t.Errorf("expr = %q, want %q", expr, tt.wantExpr)
			}
		})
	}
}

func typeName(v interface{}) string {
	return fmt.Sprintf("%T", v)
}
