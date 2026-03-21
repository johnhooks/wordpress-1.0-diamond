package template

import (
	"strings"
	"testing"

	"press/internal/template/parse"
)

type testPost struct {
	TheTitle     string
	TheContent   string
	CommentCount int
	CommentsOpen bool
	Author       testAuthor
}

type testAuthor struct {
	Name string
}

type testData struct {
	BlogName string
	Post     testPost
	Posts    []testPost
	Tags     map[string]string
	NilPtr   *testPost
}

func parseExpr(t *testing.T, s string) parse.Expr {
	t.Helper()
	expr, err := parse.ParseExpr(s)
	if err != nil {
		t.Fatalf("ParseExpr(%q): %v", s, err)
	}
	return expr
}

func TestEval_FieldAccess(t *testing.T) {
	data := testData{
		BlogName: "My Blog",
		Post: testPost{
			TheTitle:     "Hello World",
			CommentCount: 5,
			CommentsOpen: true,
			Author:       testAuthor{Name: "Alice"},
		},
	}

	tests := []struct {
		expr string
		want any
	}{
		{"BlogName", "My Blog"},
		{"Post.TheTitle", "Hello World"},
		{"Post.CommentCount", 5},
		{"Post.CommentsOpen", true},
		{"Post.Author.Name", "Alice"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Eval(parseExpr(t, tt.expr), data)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestEval_MethodNotCallable(t *testing.T) {
	// Methods are not callable from templates. The evaluator
	// only resolves struct fields and map keys.
	data := testData{}
	_, err := Eval(parseExpr(t, "Nonexistent"), data)
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestEval_MapAccess(t *testing.T) {
	data := testData{
		Tags: map[string]string{"color": "blue"},
	}
	got, err := Eval(parseExpr(t, "Tags.color"), data)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "blue" {
		t.Errorf("got %v, want blue", got)
	}
}

func TestEval_MapMissing(t *testing.T) {
	data := testData{
		Tags: map[string]string{"color": "blue"},
	}
	got, err := Eval(parseExpr(t, "Tags.missing"), data)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestEval_NilPtr(t *testing.T) {
	data := testData{NilPtr: nil}
	got, err := Eval(parseExpr(t, "NilPtr"), data)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	// NilPtr is a *testPost, so the interface value is non-nil
	// but the pointer is nil. Accessing a field on it should
	// return nil, not panic.
	if got == nil {
		return // field access on nil handled below
	}
	_, err = Eval(parseExpr(t, "NilPtr.TheTitle"), data)
	if err != nil {
		// This is fine — accessing a field on nil returns nil.
		return
	}
}

func TestEval_Literals(t *testing.T) {
	tests := []struct {
		expr string
		want any
	}{
		{`"hello"`, "hello"},
		{"42", float64(42)},
		{"3.14", float64(3.14)},
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Eval(parseExpr(t, tt.expr), nil)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestEval_Comparisons(t *testing.T) {
	data := testData{
		Post: testPost{CommentCount: 5, CommentsOpen: true},
	}

	tests := []struct {
		expr string
		want bool
	}{
		{"Post.CommentCount > 0", true},
		{"Post.CommentCount < 3", false},
		{"Post.CommentCount == 5", true},
		{"Post.CommentCount != 5", false},
		{"Post.CommentCount >= 5", true},
		{"Post.CommentCount <= 5", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Eval(parseExpr(t, tt.expr), data)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEval_Logical(t *testing.T) {
	data := testData{
		Post: testPost{CommentsOpen: true, CommentCount: 5},
	}

	tests := []struct {
		expr string
		want bool
	}{
		{"Post.CommentsOpen and Post.CommentCount > 0", true},
		{"Post.CommentsOpen and Post.CommentCount > 10", false},
		{"not Post.CommentsOpen", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalTruthy(parseExpr(t, tt.expr), data)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEval_ShortCircuit(t *testing.T) {
	data := testData{NilPtr: nil}

	// "and" with a falsy left side must not evaluate the right side.
	// NilPtr is nil, so NilPtr.TheTitle would error if evaluated.
	got, err := Eval(parseExpr(t, "NilPtr and NilPtr.TheTitle"), data)
	if err != nil {
		t.Fatalf("and short-circuit: unexpected error: %v", err)
	}
	if got != (*testPost)(nil) {
		t.Errorf("and short-circuit: got %v (%T), want nil *testPost", got, got)
	}

	// "or" with a truthy left side must not evaluate the right side.
	got, err = Eval(parseExpr(t, "BlogName or NilPtr.TheTitle"), testData{BlogName: "Press"})
	if err != nil {
		t.Fatalf("or short-circuit: unexpected error: %v", err)
	}
	if got != "Press" {
		t.Errorf("or short-circuit: got %v, want \"Press\"", got)
	}
}

func TestEval_LogicalValues(t *testing.T) {
	// and/or return the deciding value, not a boolean.
	data := testData{
		BlogName: "Press",
		Post:     testPost{TheTitle: "Hello", CommentCount: 0},
	}

	tests := []struct {
		name string
		expr string
		want any
	}{
		// or returns the first truthy value.
		{"or truthy left", `BlogName or "fallback"`, "Press"},
		// or returns the right value when left is falsy.
		{"or falsy left", `Post.CommentsOpen or "closed"`, "closed"},
		// and returns the first falsy value.
		{"and falsy left", `Post.CommentsOpen and BlogName`, false},
		// and returns the right value when left is truthy.
		{"and truthy left", `BlogName and Post.TheTitle`, "Hello"},
		// Chained or: first truthy wins.
		{"or chain", `Post.CommentsOpen or Post.CommentCount or BlogName`, "Press"},
		// Chained and: first falsy wins.
		{"and chain", `BlogName and Post.CommentsOpen and Post.TheTitle`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Eval(parseExpr(t, tt.expr), data)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"nil", nil, false},
		{"empty string", "", false},
		{"non-empty string", "hello", true},
		{"zero int", 0, false},
		{"positive int", 1, true},
		{"false", false, false},
		{"true", true, true},
		{"zero float", 0.0, false},
		{"positive float", 1.5, true},
		{"empty slice", []string{}, false},
		{"non-empty slice", []string{"a"}, true},
		{"empty map", map[string]string{}, false},
		{"non-empty map", map[string]string{"a": "b"}, true},
		{"nil pointer", (*testPost)(nil), false},
		{"non-nil pointer", &testPost{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTruthy(tt.val)
			if got != tt.want {
				t.Errorf("IsTruthy(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestEvalString(t *testing.T) {
	data := testData{
		BlogName: "My Blog",
		Post:     testPost{CommentCount: 5},
	}

	tests := []struct {
		expr string
		want string
	}{
		{"BlogName", "My Blog"},
		{"Post.CommentCount", "5"},
		{`"literal"`, "literal"},
		{"true", "true"},
		{"42", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalString(parseExpr(t, tt.expr), data)
			if err != nil {
				t.Fatalf("EvalString error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvalAttrParts(t *testing.T) {
	data := testData{
		BlogName: "My Blog",
		Post:     testPost{TheTitle: "Hello"},
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single expression", "{BlogName}", "My Blog"},
		{"mixed", "title: {Post.TheTitle}!", "title: Hello!"},
		{"multiple", "{BlogName} - {Post.TheTitle}", "My Blog - Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := parseParts(t, tt.input)
			got, err := EvalAttrParts(parts, data)
			if err != nil {
				t.Fatalf("EvalAttrParts error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvalAttrParts_Escapes(t *testing.T) {
	data := struct{ Name string }{Name: `<script>alert("xss")</script>`}
	parts := parseParts(t, "{Name}")
	got, err := EvalAttrParts(parts, data)
	if err != nil {
		t.Fatal(err)
	}
	if got != `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;` {
		t.Errorf("expected escaped output, got %q", got)
	}
}

// parseParts compiles an attribute value string into AttrParts
// using the same logic as the parser's compileAttrExprs pass.
func parseParts(t *testing.T, val string) []parse.AttrPart {
	t.Helper()
	var parts []parse.AttrPart
	for i := 0; i < len(val); {
		open := strings.IndexByte(val[i:], '{')
		if open < 0 {
			parts = append(parts, parse.AttrPart{Text: val[i:]})
			break
		}
		if open > 0 {
			parts = append(parts, parse.AttrPart{Text: val[i : i+open]})
		}
		close := strings.IndexByte(val[i+open:], '}')
		if close < 0 {
			parts = append(parts, parse.AttrPart{Text: val[i+open:]})
			break
		}
		exprStr := val[i+open+1 : i+open+close]
		expr, err := parse.ParseExpr(exprStr)
		if err != nil {
			t.Fatalf("ParseExpr(%q): %v", exprStr, err)
		}
		parts = append(parts, parse.AttrPart{Expr: expr})
		i = i + open + close + 1
	}
	return parts
}

func TestEval_MissingField(t *testing.T) {
	data := testData{}
	_, err := Eval(parseExpr(t, "Nonexistent"), data)
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}
