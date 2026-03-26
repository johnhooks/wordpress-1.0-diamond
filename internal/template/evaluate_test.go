package template

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"press/internal/template/parse"
)

func evaluateTemplate(t *testing.T, input string, data any) string {
	t.Helper()
	doc, err := parse.ParseTemplate(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTemplate error: %v", err)
	}

	result, err := Evaluate(context.Background(), doc, data)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	var buf bytes.Buffer
	if err := parse.Render(&buf, result); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	return buf.String()
}

func TestBuilders_DocumentNode(t *testing.T) {
	n := newDocumentNode()
	if n.Type != parse.DocumentNode {
		t.Errorf("expected DocumentNode, got %v", n.Type)
	}
}

func TestBuilders_TextNode(t *testing.T) {
	n := newTextNode("hello")
	if n.Type != parse.TextNode || n.Data != "hello" {
		t.Errorf("expected TextNode with 'hello', got %v %q", n.Type, n.Data)
	}
}

func TestBuilders_RawNode(t *testing.T) {
	n := newRawNode("<b>bold</b>")
	if n.Type != parse.RawNode || n.Data != "<b>bold</b>" {
		t.Errorf("expected RawNode, got %v %q", n.Type, n.Data)
	}
}

func TestBuilders_ElementNode(t *testing.T) {
	attrs := []parse.Attribute{{Key: "class", Val: "test"}}
	n := newElementNode("div", attrs)
	if n.Type != parse.ElementNode || n.Data != "div" || len(n.Attr) != 1 {
		t.Errorf("expected ElementNode div with 1 attr, got %v %q %d", n.Type, n.Data, len(n.Attr))
	}
}

func TestBuilders_CommentNode(t *testing.T) {
	n := newCommentNode(" a comment ")
	if n.Type != parse.CommentNode || n.Data != " a comment " {
		t.Errorf("expected CommentNode, got %v %q", n.Type, n.Data)
	}
}

func TestBuilders_CloneShallow(t *testing.T) {
	parent := newElementNode("div", []parse.Attribute{{Key: "id", Val: "x"}})
	child := newTextNode("inner")
	parent.AppendChild(child)

	cloned := cloneShallow(parent)
	if cloned.Type != parse.ElementNode || cloned.Data != "div" {
		t.Errorf("clone type/data mismatch")
	}
	if len(cloned.Attr) != 1 || cloned.Attr[0].Val != "x" {
		t.Errorf("clone attrs mismatch")
	}
	if cloned.FirstChild != nil {
		t.Errorf("cloneShallow should not copy children")
	}
	if cloned.Parent != nil {
		t.Errorf("cloneShallow should not have parent")
	}
}

func TestBuilders_AppendChildren(t *testing.T) {
	parent := newDocumentNode()
	children := []*parse.Node{
		newTextNode("a"),
		newTextNode("b"),
		newTextNode("c"),
	}
	appendChildren(parent, children)

	// Verify linked list integrity.
	if parent.FirstChild != children[0] {
		t.Error("FirstChild wrong")
	}
	if parent.LastChild != children[2] {
		t.Error("LastChild wrong")
	}
	if children[0].NextSibling != children[1] {
		t.Error("first→second link wrong")
	}
	if children[1].NextSibling != children[2] {
		t.Error("second→third link wrong")
	}
	if children[2].PrevSibling != children[1] {
		t.Error("third←second link wrong")
	}
	if children[1].PrevSibling != children[0] {
		t.Error("second←first link wrong")
	}
	for _, c := range children {
		if c.Parent != parent {
			t.Error("child parent not set")
		}
	}
}

func TestEvaluate_StaticHTML(t *testing.T) {
	got := evaluateTemplate(t, `<div><p>hello</p></div>`, nil)
	if !strings.Contains(got, "<div><p>hello</p></div>") {
		t.Errorf("expected static HTML preserved, got: %s", got)
	}
}

func TestEvaluate_TextEscaping(t *testing.T) {
	got := evaluateTemplate(t, `<p>5 &gt; 3 &amp; done</p>`, nil)
	if !strings.Contains(got, "5 &gt; 3 &amp; done") {
		t.Errorf("expected entities preserved, got: %s", got)
	}
}

func TestEvaluate_Comment(t *testing.T) {
	got := evaluateTemplate(t, `<div><!-- hello --></div>`, nil)
	if !strings.Contains(got, "<!-- hello -->") {
		t.Errorf("expected comment preserved, got: %s", got)
	}
}

func TestEvaluate_VoidElements(t *testing.T) {
	got := evaluateTemplate(t, `<div><br><hr><img src="test.png"></div>`, nil)
	if !strings.Contains(got, "<br/>") || !strings.Contains(got, "<hr/>") {
		t.Errorf("expected void elements, got: %s", got)
	}
}

func TestEvaluate_Expression(t *testing.T) {
	data := struct{ BlogName string }{BlogName: "My Blog"}
	got := evaluateTemplate(t, `<h1>{BlogName}</h1>`, data)
	if !strings.Contains(got, "<h1>My Blog</h1>") {
		t.Errorf("expected expression resolved, got: %s", got)
	}
}

func TestEvaluate_ExpressionEscaped(t *testing.T) {
	data := struct{ Title string }{Title: `<script>alert("xss")</script>`}
	got := evaluateTemplate(t, `<h1>{Title}</h1>`, data)
	if strings.Contains(got, "<script>") {
		t.Errorf("expected HTML escaping, got: %s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped output, got: %s", got)
	}
}

func TestEvaluate_MemberExpression(t *testing.T) {
	data := struct {
		Post struct{ TheTitle string }
	}{Post: struct{ TheTitle string }{TheTitle: "Hello World"}}
	got := evaluateTemplate(t, `<h1>{Post.TheTitle}</h1>`, data)
	if !strings.Contains(got, "<h1>Hello World</h1>") {
		t.Errorf("expected member access, got: %s", got)
	}
}

func TestEvaluate_AttributeExpression(t *testing.T) {
	data := struct{ URL string }{URL: "/posts/1"}
	got := evaluateTemplate(t, `<a href="{URL}">link</a>`, data)
	if !strings.Contains(got, `href="/posts/1"`) {
		t.Errorf("expected attribute resolved, got: %s", got)
	}
}

func TestEvaluate_MixedAttribute(t *testing.T) {
	data := struct{ Query string }{Query: "a&b"}
	got := evaluateTemplate(t, `<a href="/search?q={Query}">link</a>`, data)
	if !strings.Contains(got, "a&amp;b") {
		t.Errorf("expected single escape, got: %s", got)
	}
	if strings.Contains(got, "a&amp;amp;b") {
		t.Errorf("double-escaped, got: %s", got)
	}
}

func TestEvaluate_RawField(t *testing.T) {
	type Page struct {
		Content string `view:"the_content,raw"`
	}
	data := Page{Content: "<p>Hello <strong>world</strong></p>"}
	got := evaluateTemplate(t, `<div>{the_content}</div>`, data)
	if !strings.Contains(got, "<strong>world</strong>") {
		t.Errorf("expected raw HTML preserved, got: %s", got)
	}
}

func TestEvaluate_HTMLContentEscaped(t *testing.T) {
	data := struct{ Content string }{Content: "<p>Hello <strong>world</strong></p>"}
	got := evaluateTemplate(t, `<div>{Content}</div>`, data)
	if strings.Contains(got, "<strong>") {
		t.Errorf("expected HTML to be escaped, got: %s", got)
	}
}

func TestEvaluate_IfTrue(t *testing.T) {
	data := struct{ Show bool }{Show: true}
	got := evaluateTemplate(t, `<div>{if Show}<p>yes</p>{/if}</div>`, data)
	if !strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected then-branch, got: %s", got)
	}
}

func TestEvaluate_IfFalse(t *testing.T) {
	data := struct{ Show bool }{Show: false}
	got := evaluateTemplate(t, `<div>{if Show}<p>yes</p>{/if}</div>`, data)
	if strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected then-branch NOT rendered, got: %s", got)
	}
}

func TestEvaluate_IfElse(t *testing.T) {
	data := struct{ Show bool }{Show: false}
	got := evaluateTemplate(t, `<div>{if Show}<p>yes</p>{else}<p>no</p>{/if}</div>`, data)
	if strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected then-branch NOT rendered, got: %s", got)
	}
	if !strings.Contains(got, "<p>no</p>") {
		t.Errorf("expected else-branch rendered, got: %s", got)
	}
}

func TestEvaluate_Each(t *testing.T) {
	data := struct {
		Items []struct{ Name string }
	}{Items: []struct{ Name string }{
		{Name: "Alice"},
		{Name: "Bob"},
		{Name: "Charlie"},
	}}
	got := evaluateTemplate(t, `<ul>{each Items as item}<li>{item.Name}</li>{/each}</ul>`, data)
	if !strings.Contains(got, "<li>Alice</li>") || !strings.Contains(got, "<li>Bob</li>") || !strings.Contains(got, "<li>Charlie</li>") {
		t.Errorf("expected all items, got: %s", got)
	}
}

func TestEvaluate_EachEmpty(t *testing.T) {
	data := struct{ Items []string }{Items: []string{}}
	got := evaluateTemplate(t, `<div>{each Items as item}<p>item</p>{/each}</div>`, data)
	if strings.Contains(got, "<p>item</p>") {
		t.Errorf("expected no items, got: %s", got)
	}
}

func TestEvaluate_EachElse(t *testing.T) {
	data := struct{ Items []string }{Items: []string{}}
	got := evaluateTemplate(t, `<div>{each Items as item}<p>{item}</p>{else}<p>empty</p>{/each}</div>`, data)
	if !strings.Contains(got, "<p>empty</p>") {
		t.Errorf("expected else branch, got: %s", got)
	}
}

func TestEvaluate_EachElseNotRenderedWhenFull(t *testing.T) {
	data := struct{ Items []string }{Items: []string{"a", "b"}}
	got := evaluateTemplate(t, `<div>{each Items as item}<p>{item}</p>{else}<p>empty</p>{/each}</div>`, data)
	if strings.Contains(got, "<p>empty</p>") {
		t.Errorf("else branch should not render, got: %s", got)
	}
	if !strings.Contains(got, "<p>a</p>") || !strings.Contains(got, "<p>b</p>") {
		t.Errorf("expected items, got: %s", got)
	}
}

func TestEvaluate_EachWithIndex(t *testing.T) {
	data := struct{ Items []string }{Items: []string{"a", "b", "c"}}
	got := evaluateTemplate(t, `<ul>{each Items as item, i}<li>{i}</li>{/each}</ul>`, data)
	if !strings.Contains(got, "<li>0</li>") || !strings.Contains(got, "<li>2</li>") {
		t.Errorf("expected indices, got: %s", got)
	}
}

func TestEvaluate_Const(t *testing.T) {
	data := struct{ Count int }{Count: 5}
	got := evaluateTemplate(t, `<div>{const hasItems = Count > 0}{if hasItems}<p>yes</p>{/if}</div>`, data)
	if !strings.Contains(got, "<p>yes</p>") {
		t.Errorf("expected const with if, got: %s", got)
	}
}

func TestEvaluate_NestedEachIf(t *testing.T) {
	type Post struct {
		Title    string
		Featured bool
	}
	data := struct{ Posts []Post }{Posts: []Post{
		{Title: "First", Featured: true},
		{Title: "Second", Featured: false},
		{Title: "Third", Featured: true},
	}}
	got := evaluateTemplate(t, `<div>{each Posts as post}{if post.Featured}<p>{post.Title}</p>{/if}{/each}</div>`, data)
	if !strings.Contains(got, "<p>First</p>") || strings.Contains(got, "<p>Second</p>") || !strings.Contains(got, "<p>Third</p>") {
		t.Errorf("expected filtered posts, got: %s", got)
	}
}

func TestEvaluate_ConstScopedToEachIteration(t *testing.T) {
	data := struct{ Items []int }{Items: []int{1, 2, 3}}
	got := evaluateTemplate(t, `<ul>{each Items as item}{const label = item}<li>{label}</li>{/each}</ul>`, data)
	if !strings.Contains(got, "<li>1</li>") || !strings.Contains(got, "<li>2</li>") || !strings.Contains(got, "<li>3</li>") {
		t.Errorf("expected scoped consts, got: %s", got)
	}
}

func TestEvaluate_ConstInsideEachNotVisibleOutside(t *testing.T) {
	data := struct {
		Items []int
		X     string
	}{Items: []int{1}, X: "outer"}
	got := evaluateTemplate(t, `<div>{each Items as item}{const X = item}{/each}<p>{X}</p></div>`, data)
	if !strings.Contains(got, "<p>outer</p>") {
		t.Errorf("expected outer X preserved, got: %s", got)
	}
}

func TestEvaluate_TagNodeHandler(t *testing.T) {
	handler := func(_ context.Context, ev *Evaluator, el *parse.Node) (*parse.Node, error) {
		div := newElementNode("div", []parse.Attribute{{Key: "class", Val: "custom"}})
		div.AppendChild(newTextNode("handled"))
		return div, nil
	}

	doc, err := parse.ParseTemplate(strings.NewReader(`<div><my-tag/></div>`))
	if err != nil {
		t.Fatal(err)
	}

	resolver := &testResolver{handlers: map[string]TagNodeHandler{"my-tag": handler}}
	ctx := WithTagResolver(context.Background(), resolver)

	result, err := Evaluate(ctx, doc, nil)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := parse.Render(&buf, result); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, `<div class="custom">handled</div>`) {
		t.Errorf("expected handler output, got: %s", got)
	}
}

func TestEvaluate_TagHandlerWithSubTemplate(t *testing.T) {
	handler := func(_ context.Context, ev *Evaluator, el *parse.Node) (*parse.Node, error) {
		wrapper := newElementNode("section", nil)
		name := ""
		for _, a := range el.Attr {
			if a.Key == "name" {
				name = a.Val
			}
		}
		wrapper.AppendChild(newTextNode("Hello, " + name))
		return wrapper, nil
	}

	doc, err := parse.ParseTemplate(strings.NewReader(`<div><greeting name="{Name}"/></div>`))
	if err != nil {
		t.Fatal(err)
	}

	resolver := &testResolver{handlers: map[string]TagNodeHandler{"greeting": handler}}
	ctx := WithTagResolver(context.Background(), resolver)

	data := struct{ Name string }{Name: "World"}
	result, err := Evaluate(ctx, doc, data)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := parse.Render(&buf, result); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "<section>Hello, World</section>") {
		t.Errorf("expected resolved attr in handler, got: %s", got)
	}
}

func TestEvaluate_UnknownElementPassesThrough(t *testing.T) {
	got := evaluateTemplate(t, `<div><custom-el class="x">content</custom-el></div>`, nil)
	if !strings.Contains(got, `<custom-el class="x">content</custom-el>`) {
		t.Errorf("expected unknown element to pass through, got: %s", got)
	}
}

func TestEvaluate_TitleExpression(t *testing.T) {
	data := struct{ BlogName string }{BlogName: "My Blog"}
	got := evaluateTemplate(t, `<title>{BlogName}</title>`, data)
	if !strings.Contains(got, "<title>My Blog</title>") {
		t.Errorf("expected expression resolved inside <title>, got: %s", got)
	}
}

func TestEvaluate_TextareaExpression(t *testing.T) {
	data := struct{ Value string }{Value: "saved text"}
	got := evaluateTemplate(t, `<textarea>{Value}</textarea>`, data)
	if !strings.Contains(got, "saved text") {
		t.Errorf("expected expression resolved inside <textarea>, got: %s", got)
	}
}

func TestEvaluate_EntityRoundTrip(t *testing.T) {
	got := evaluateTemplate(t, `<p>5 &gt; 3 &amp; done</p>`, struct{}{})
	if !strings.Contains(got, "5 &gt; 3 &amp; done") {
		t.Errorf("expected entities preserved, got: %s", got)
	}
}

// testResolver is a simple TagResolver for testing.
type testResolver struct {
	handlers map[string]TagNodeHandler
}

func (r *testResolver) ResolveTag(name string) (TagNodeHandler, bool) {
	h, ok := r.handlers[name]
	return h, ok
}
