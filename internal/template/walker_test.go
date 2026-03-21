package template

import (
	"bytes"
	"strings"
	"testing"

	"press/internal/template/parse"
)

func renderTemplate(t *testing.T, input string, data any) string {
	t.Helper()
	doc, err := parse.ParseTemplate(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTemplate error: %v", err)
	}

	var buf bytes.Buffer
	if err := Walk(&buf, doc, data, nil, nil); err != nil {
		t.Fatalf("Walk error: %v", err)
	}
	return buf.String()
}

func renderWithHandlers(t *testing.T, input string, data any, handlers map[string]TagHandler) string {
	t.Helper()
	doc, err := parse.ParseTemplate(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTemplate error: %v", err)
	}

	var buf bytes.Buffer
	if err := Walk(&buf, doc, data, handlers, nil); err != nil {
		t.Fatalf("Walk error: %v", err)
	}
	return buf.String()
}

func TestWalk_Expression(t *testing.T) {
	data := struct {
		BlogName string
	}{BlogName: "My Blog"}

	got := renderTemplate(t, `<h1>{BlogName}</h1>`, data)
	if !strings.Contains(got, "<h1>My Blog</h1>") {
		t.Errorf("expected expression resolved, got: %s", got)
	}
}

func TestWalk_ExpressionEscaped(t *testing.T) {
	data := struct {
		Title string
	}{Title: `<script>alert("xss")</script>`}

	got := renderTemplate(t, `<h1>{Title}</h1>`, data)
	if strings.Contains(got, "<script>") {
		t.Errorf("expected HTML escaping, got: %s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped output, got: %s", got)
	}
}

func TestWalk_MemberExpression(t *testing.T) {
	data := struct {
		Post struct {
			TheTitle string
		}
	}{Post: struct{ TheTitle string }{TheTitle: "Hello World"}}

	got := renderTemplate(t, `<h1>{Post.TheTitle}</h1>`, data)
	if !strings.Contains(got, "<h1>Hello World</h1>") {
		t.Errorf("expected member access, got: %s", got)
	}
}

func TestWalk_AttributeExpression(t *testing.T) {
	data := struct {
		URL string
	}{URL: "/posts/1"}

	got := renderTemplate(t, `<a href="{URL}">link</a>`, data)
	if !strings.Contains(got, `href="/posts/1"`) {
		t.Errorf("expected attribute resolved, got: %s", got)
	}
}

func TestWalk_IfTrue(t *testing.T) {
	data := struct {
		Show bool
	}{Show: true}

	got := renderTemplate(t, `<div>{if Show}<p>yes</p>{/if}</div>`, data)
	if !strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected then-branch rendered, got: %s", got)
	}
}

func TestWalk_IfFalse(t *testing.T) {
	data := struct {
		Show bool
	}{Show: false}

	got := renderTemplate(t, `<div>{if Show}<p>yes</p>{/if}</div>`, data)
	if strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected then-branch NOT rendered, got: %s", got)
	}
}

func TestWalk_IfElse(t *testing.T) {
	data := struct {
		Show bool
	}{Show: false}

	got := renderTemplate(t, `<div>{if Show}<p>yes</p>{else}<p>no</p>{/if}</div>`, data)
	if strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected then-branch NOT rendered, got: %s", got)
	}
	if !strings.Contains(got, "<p>no</p>") {
		t.Errorf("expected else-branch rendered, got: %s", got)
	}
}

func TestWalk_IfTruthy(t *testing.T) {
	data := struct {
		Items []string
	}{Items: []string{"a", "b"}}

	got := renderTemplate(t, `<div>{if Items}<p>has items</p>{/if}</div>`, data)
	if !strings.Contains(got, "has items") {
		t.Errorf("expected truthy slice, got: %s", got)
	}
}

func TestWalk_IfTruthyEmpty(t *testing.T) {
	data := struct {
		Items []string
	}{Items: []string{}}

	got := renderTemplate(t, `<div>{if Items}<p>has items</p>{else}<p>empty</p>{/if}</div>`, data)
	if strings.Contains(got, "has items") {
		t.Errorf("expected empty slice falsy, got: %s", got)
	}
	if !strings.Contains(got, "empty") {
		t.Errorf("expected else-branch, got: %s", got)
	}
}

func TestWalk_Each(t *testing.T) {
	data := struct {
		Items []struct{ Name string }
	}{Items: []struct{ Name string }{
		{Name: "Alice"},
		{Name: "Bob"},
		{Name: "Charlie"},
	}}

	got := renderTemplate(t, `<ul>{each Items as item}<li>{item.Name}</li>{/each}</ul>`, data)
	if !strings.Contains(got, "<li>Alice</li>") {
		t.Errorf("expected Alice, got: %s", got)
	}
	if !strings.Contains(got, "<li>Bob</li>") {
		t.Errorf("expected Bob, got: %s", got)
	}
	if !strings.Contains(got, "<li>Charlie</li>") {
		t.Errorf("expected Charlie, got: %s", got)
	}
}

func TestWalk_EachEmpty(t *testing.T) {
	data := struct {
		Items []string
	}{Items: []string{}}

	got := renderTemplate(t, `<div>{each Items as item}<p>item</p>{/each}</div>`, data)
	if strings.Contains(got, "<p>item</p>") {
		t.Errorf("expected no items rendered, got: %s", got)
	}
	if !strings.Contains(got, "<div></div>") {
		t.Errorf("expected empty div, got: %s", got)
	}
}

func TestWalk_EachElse(t *testing.T) {
	data := struct {
		Items []string
	}{Items: []string{}}

	got := renderTemplate(t, `<div>{each Items as item}<p>{item}</p>{else}<p>empty</p>{/each}</div>`, data)
	if strings.Contains(got, "<p>item</p>") {
		t.Errorf("expected no items rendered, got: %s", got)
	}
	if !strings.Contains(got, "<p>empty</p>") {
		t.Errorf("expected else branch rendered, got: %s", got)
	}
}

func TestWalk_EachElseNotRenderedWhenFull(t *testing.T) {
	data := struct {
		Items []string
	}{Items: []string{"a", "b"}}

	got := renderTemplate(t, `<div>{each Items as item}<p>{item}</p>{else}<p>empty</p>{/each}</div>`, data)
	if strings.Contains(got, "<p>empty</p>") {
		t.Errorf("else branch should not render when list has items, got: %s", got)
	}
	if !strings.Contains(got, "<p>a</p>") || !strings.Contains(got, "<p>b</p>") {
		t.Errorf("expected items rendered, got: %s", got)
	}
}

func TestWalk_EachWithIndex(t *testing.T) {
	data := struct {
		Items []string
	}{Items: []string{"a", "b", "c"}}

	got := renderTemplate(t, `<ul>{each Items as item, i}<li>{i}</li>{/each}</ul>`, data)
	if !strings.Contains(got, "<li>0</li>") {
		t.Errorf("expected index 0, got: %s", got)
	}
	if !strings.Contains(got, "<li>2</li>") {
		t.Errorf("expected index 2, got: %s", got)
	}
}

func TestWalk_Const(t *testing.T) {
	data := struct {
		Count int
	}{Count: 5}

	got := renderTemplate(t, `<div>{const hasItems = Count > 0}{if hasItems}<p>yes</p>{/if}</div>`, data)
	if !strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected const to work with if, got: %s", got)
	}
}

func TestWalk_ConstScopedToEachIteration(t *testing.T) {
	data := struct {
		Items []int
	}{Items: []int{1, 2, 3}}

	// Each iteration gets its own scope. A const defined inside
	// the loop should not carry over between iterations.
	// "prev" is set to item on each pass. If scopes leaked,
	// the second iteration would see prev=1 from the first.
	got := renderTemplate(t, `<ul>{each Items as item}{const label = item}<li>{label}</li>{/each}</ul>`, data)
	if !strings.Contains(got, "<li>1</li>") {
		t.Errorf("expected 1, got: %s", got)
	}
	if !strings.Contains(got, "<li>2</li>") {
		t.Errorf("expected 2, got: %s", got)
	}
	if !strings.Contains(got, "<li>3</li>") {
		t.Errorf("expected 3, got: %s", got)
	}
}

func TestWalk_ConstInsideEachNotVisibleOutside(t *testing.T) {
	data := struct {
		Items []int
		X     string
	}{Items: []int{1}, X: "outer"}

	// A const "X" inside {each} should not overwrite the outer
	// field "X". After the loop, {X} should resolve to the
	// struct field, not the const from inside the loop.
	got := renderTemplate(t, `<div>{each Items as item}{const X = item}{/each}<p>{X}</p></div>`, data)
	if !strings.Contains(got, "<p>outer</p>") {
		t.Errorf("expected outer X preserved, got: %s", got)
	}
}

func TestWalk_HTMLContentEscaped(t *testing.T) {
	data := struct {
		Content string
	}{Content: "<p>Hello <strong>world</strong></p>"}

	got := renderTemplate(t, `<div>{Content}</div>`, data)
	// HTML content through a regular expression must be escaped.
	if strings.Contains(got, "<strong>") {
		t.Errorf("expected HTML to be escaped, got raw tags: %s", got)
	}
	if !strings.Contains(got, "&lt;p&gt;Hello &lt;strong&gt;world&lt;/strong&gt;&lt;/p&gt;") {
		t.Errorf("expected escaped HTML content, got: %s", got)
	}
}

func TestWalk_VocabularyTag(t *testing.T) {
	data := struct {
		PostID int
	}{PostID: 42}

	handlers := map[string]TagHandler{
		"comment-form": func(ctx *RenderContext) (string, error) {
			id := ctx.Attrs["post-id"]
			return `<form data-post="` + id + `"><textarea></textarea></form>`, nil
		},
	}

	got := renderWithHandlers(t,
		`<div><comment-form post-id="{PostID}" /></div>`,
		data, handlers)

	if !strings.Contains(got, `<form data-post="42">`) {
		t.Errorf("expected handler output with resolved attr, got: %s", got)
	}
}

func TestWalk_NestedEachIf(t *testing.T) {
	type Post struct {
		Title    string
		Featured bool
	}
	data := struct {
		Posts []Post
	}{Posts: []Post{
		{Title: "First", Featured: true},
		{Title: "Second", Featured: false},
		{Title: "Third", Featured: true},
	}}

	got := renderTemplate(t, `<div>{each Posts as post}{if post.Featured}<p>{post.Title}</p>{/if}{/each}</div>`, data)
	if !strings.Contains(got, "<p>First</p>") {
		t.Errorf("expected First, got: %s", got)
	}
	if strings.Contains(got, "<p>Second</p>") {
		t.Errorf("expected Second filtered out, got: %s", got)
	}
	if !strings.Contains(got, "<p>Third</p>") {
		t.Errorf("expected Third, got: %s", got)
	}
}

func TestWalk_AttributeNoDoubleEscape(t *testing.T) {
	data := struct {
		Query string
	}{Query: "a&b"}

	got := renderTemplate(t, `<a href="/search?q={Query}">link</a>`, data)
	// Should be escaped once, not double-escaped.
	if !strings.Contains(got, "a&amp;b") {
		t.Errorf("expected single escape, got: %s", got)
	}
	if strings.Contains(got, "a&amp;amp;b") {
		t.Errorf("double-escaped, got: %s", got)
	}
}

func TestWalk_TitleExpression(t *testing.T) {

	data := struct {
		BlogName string
	}{BlogName: "My Blog"}

	got := renderTemplate(t, `<title>{BlogName}</title>`, data)
	if !strings.Contains(got, "<title>My Blog</title>") {
		t.Errorf("expected expression resolved inside <title>, got: %s", got)
	}
}

func TestWalk_TextareaExpression(t *testing.T) {

	data := struct {
		Value string
	}{Value: "saved text"}

	got := renderTemplate(t, `<textarea>{Value}</textarea>`, data)
	if !strings.Contains(got, "saved text") {
		t.Errorf("expected expression resolved inside <textarea>, got: %s", got)
	}
}

func TestWalk_EntityRoundTrip(t *testing.T) {
	data := struct{}{}

	got := renderTemplate(t, `<p>5 &gt; 3 &amp; done</p>`, data)
	if !strings.Contains(got, "5 &gt; 3 &amp; done") {
		t.Errorf("expected entities preserved in round-trip, got: %s", got)
	}
	if strings.Contains(got, "&amp;gt;") {
		t.Errorf("entities were double-escaped, got: %s", got)
	}
}

func TestWalk_VoidElements(t *testing.T) {
	data := struct{}{}
	got := renderTemplate(t, `<div><br><hr><img src="test.png"></div>`, data)
	if !strings.Contains(got, "<br/>") || !strings.Contains(got, "<hr/>") {
		t.Errorf("expected void elements self-closing, got: %s", got)
	}
}
