package parse

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestExpressionNode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantExprs []string
	}{
		{"simple variable", `<p>{name}</p>`, []string{"name"}},
		{"dot path", `<p>{post.Title}</p>`, []string{"post.Title"}},
		{"deep dot path", `<p>{post.Author.Name}</p>`, []string{"post.Author.Name"}},
		{"multiple expressions", `<h1>{post.TheTitle}</h1><p>{post.TheDate}</p>`, []string{"post.TheTitle", "post.TheDate"}},
		{"string literal", `<p>{"hello"}</p>`, []string{`"hello"`}},
		{"expression with spaces", `<p>{ name }</p>`, []string{"name"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseHTML(t, tt.input)

			var got []string
			collectExpressions(doc, &got)

			if len(got) != len(tt.wantExprs) {
				t.Fatalf("got %d expressions, want %d", len(got), len(tt.wantExprs))
			}
			for i, want := range tt.wantExprs {
				if got[i] != want {
					t.Errorf("expr[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestExpressionPreservesText(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantNodes  []nodeCheck
	}{
		{
			"text before and after",
			`<p>Hello {name}, welcome!</p>`,
			[]nodeCheck{
				{Type: TextNode, Data: "Hello "},
				{Type: ExpressionNode, Data: "name"},
				{Type: TextNode, Data: ", welcome!"},
			},
		},
		{
			"expression only",
			`<p>{title}</p>`,
			[]nodeCheck{
				{Type: ExpressionNode, Data: "title"},
			},
		},
		{
			"text then expression",
			`<p>Posted on {date}</p>`,
			[]nodeCheck{
				{Type: TextNode, Data: "Posted on "},
				{Type: ExpressionNode, Data: "date"},
			},
		},
		{
			"expression then text",
			`<p>{author} wrote this</p>`,
			[]nodeCheck{
				{Type: ExpressionNode, Data: "author"},
				{Type: TextNode, Data: " wrote this"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseHTML(t, tt.input)

			p := findElement(doc, "p")
			if p == nil {
				t.Fatal("expected a <p> element")
			}

			var children []nodeCheck
			for c := p.FirstChild; c != nil; c = c.NextSibling {
				children = append(children, nodeCheck{Type: c.Type, Data: c.Data})
			}

			if len(children) != len(tt.wantNodes) {
				t.Fatalf("got %d children, want %d: %v", len(children), len(tt.wantNodes), children)
			}
			for i, want := range tt.wantNodes {
				got := children[i]
				if got.Type != want.Type || got.Data != want.Data {
					t.Errorf("child[%d] = {%v %q}, want {%v %q}", i, got.Type, got.Data, want.Type, want.Data)
				}
			}
		})
	}
}

func TestExpressionRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantContains string
	}{
		{"simple", `<p>Hello {name}</p>`, "Hello {name}"},
		{"dot path", `<p>{post.Title}</p>`, "{post.Title}"},
		{"multiple", `<p>{a} and {b}</p>`, "{a} and {b}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseHTML(t, tt.input)

			var buf bytes.Buffer
			if err := Render(&buf, doc); err != nil {
				t.Fatalf("Render error: %v", err)
			}

			if !strings.Contains(buf.String(), tt.wantContains) {
				t.Errorf("rendered %q does not contain %q", buf.String(), tt.wantContains)
			}
		})
	}
}

func TestExpressionParent(t *testing.T) {
	doc := parseHTML(t, `<div><p>{post.Title}</p></div>`)

	expr := findNode(doc, ExpressionNode)
	if expr == nil {
		t.Fatal("expected an ExpressionNode")
	}
	if expr.Parent == nil || expr.Parent.Data != "p" {
		parent := "nil"
		if expr.Parent != nil {
			parent = expr.Parent.Data
		}
		t.Errorf("expression parent = %q, want \"p\"", parent)
	}
}

func TestSelfClosingVocabularyTag(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantChildren []nodeCheck // expected children of <body>
	}{
		{
			"self-closing has no children",
			`<div><comment-form /><p>sibling</p></div>`,
			// <div> should have two children: <comment-form> and <p>
			[]nodeCheck{
				{Type: ElementNode, Data: "comment-form"},
				{Type: ElementNode, Data: "p"},
			},
		},
		{
			"multiple self-closing siblings",
			`<div><site-header /><post /><site-footer /></div>`,
			[]nodeCheck{
				{Type: ElementNode, Data: "site-header"},
				{Type: ElementNode, Data: "post"},
				{Type: ElementNode, Data: "site-footer"},
			},
		},
		{
			"self-closing with attributes",
			`<div><comment-form post-id="42" /><p>after</p></div>`,
			[]nodeCheck{
				{Type: ElementNode, Data: "comment-form"},
				{Type: ElementNode, Data: "p"},
			},
		},
		{
			"self-closing preserves attributes",
			`<comment-form post-id="42" />`,
			nil, // checked separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseHTML(t, tt.input)

			if tt.wantChildren == nil {
				// Just check the tag has the attribute.
				cf := findElement(doc, "comment-form")
				if cf == nil {
					t.Fatal("expected <comment-form> element")
				}
				if len(cf.Attr) == 0 {
					t.Fatal("expected attributes on <comment-form>")
				}
				if cf.Attr[0].Key != "post-id" || cf.Attr[0].Val != "42" {
					t.Errorf("attr = %v, want post-id=42", cf.Attr[0])
				}
				return
			}

			div := findElement(doc, "div")
			if div == nil {
				t.Fatal("expected a <div> element")
			}

			var children []nodeCheck
			for c := div.FirstChild; c != nil; c = c.NextSibling {
				children = append(children, nodeCheck{Type: c.Type, Data: c.Data})
			}

			if len(children) != len(tt.wantChildren) {
				t.Fatalf("got %d children, want %d: %v", len(children), len(tt.wantChildren), children)
			}
			for i, want := range tt.wantChildren {
				got := children[i]
				if got.Type != want.Type || got.Data != want.Data {
					t.Errorf("child[%d] = {%v %q}, want {%v %q}", i, got.Type, got.Data, want.Type, want.Data)
				}
			}
		})
	}
}

func TestSelfClosingNoChildren(t *testing.T) {
	doc := parseHTML(t, `<comment-form /><p>not a child</p>`)

	cf := findElement(doc, "comment-form")
	if cf == nil {
		t.Fatal("expected <comment-form> element")
	}
	if cf.FirstChild != nil {
		t.Errorf("<comment-form> should have no children, got %v", cf.FirstChild.Data)
	}
}

func TestSelfClosingRendersWithCloseTag(t *testing.T) {
	doc := parseHTML(t, `<div><comment-form post-id="42" /><p>sibling</p></div>`)

	var buf bytes.Buffer
	if err := Render(&buf, doc); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `<comment-form post-id="42"></comment-form>`) {
		t.Errorf("expected closing tag in output, got: %s", got)
	}
	if !strings.Contains(got, `<p>sibling</p>`) {
		t.Errorf("expected sibling <p> in output, got: %s", got)
	}
}

func TestNonVocabularyTagNotSelfClosing(t *testing.T) {
	// A non-vocabulary custom tag should still be treated per HTML5 rules:
	// the self-closing is ignored and <p> becomes a child.
	doc := parseHTML(t, `<div><my-widget /><p>child</p></div>`)

	widget := findElement(doc, "my-widget")
	if widget == nil {
		t.Fatal("expected <my-widget> element")
	}

	// Per HTML5 rules, <p> should be inside <my-widget> (swallowed).
	p := findElement(widget, "p")
	if p == nil {
		t.Error("expected <p> to be swallowed as child of <my-widget> per HTML5 rules")
	}
}

func TestExpressionInAttribute(t *testing.T) {
	// Expressions in attribute values should be preserved as-is in the
	// attribute string. The HTML tokenizer reads attributes as text, so
	// {url} stays literal in the attribute value. The renderer is
	// responsible for resolving expressions in attributes at render time.
	doc := parseHTML(t, `<a href="{url}">link</a>`)

	a := findElement(doc, "a")
	if a == nil {
		t.Fatal("expected <a> element")
	}
	if len(a.Attr) == 0 {
		t.Fatal("expected attributes on <a>")
	}
	if a.Attr[0].Val != "{url}" {
		t.Errorf("attr val = %q, want %q", a.Attr[0].Val, "{url}")
	}
}

func TestExpressionInAttributeMixed(t *testing.T) {
	doc := parseHTML(t, `<div class="active {className}">text</div>`)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div> element")
	}
	if len(div.Attr) == 0 {
		t.Fatal("expected attributes on <div>")
	}
	if div.Attr[0].Val != "active {className}" {
		t.Errorf("attr val = %q, want %q", div.Attr[0].Val, "active {className}")
	}
}

func TestUnclosedTemplateExpression(t *testing.T) {
	_, err := Parse(strings.NewReader(`<p>{name</p>`))
	if err == nil {
		t.Fatal("expected error for unclosed template expression")
	}
	if !strings.Contains(err.Error(), "unclosed template expression") {
		t.Errorf("expected unclosed template expression error, got: %v", err)
	}
}

// --- Block node tests (using ParseTemplate) ---

func parseTemplate(t *testing.T, input string) *Node {
	t.Helper()
	doc, err := ParseTemplate(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTemplate error: %v", err)
	}
	return doc
}

func TestIfBlockStructure(t *testing.T) {
	doc := parseTemplate(t, `<div>{if commentsOpen}<p>open</p>{/if}</div>`)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	// div should have one child: the IfNode.
	ifN := div.FirstChild
	if ifN == nil || ifN.Type != IfNode {
		t.Fatalf("expected IfNode child, got %v", nodeDesc(ifN))
	}
	if ifN.Data != "commentsOpen" {
		t.Errorf("IfNode.Data = %q, want %q", ifN.Data, "commentsOpen")
	}

	// IfNode should contain the <p>.
	p := ifN.FirstChild
	if p == nil || p.Type != ElementNode || p.Data != "p" {
		t.Fatalf("expected <p> inside IfNode, got %v", nodeDesc(p))
	}

	// No close marker should remain.
	if ifN.NextSibling != nil {
		t.Errorf("expected no sibling after IfNode, got %v", nodeDesc(ifN.NextSibling))
	}
}

func TestIfElseBlockStructure(t *testing.T) {
	doc := parseTemplate(t, `<div>{if show}<p>yes</p>{else}<p>no</p>{/if}</div>`)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	ifN := div.FirstChild
	if ifN == nil || ifN.Type != IfNode {
		t.Fatalf("expected IfNode, got %v", nodeDesc(ifN))
	}

	// First child of IfNode is the "then" branch <p>.
	thenP := ifN.FirstChild
	if thenP == nil || thenP.Data != "p" {
		t.Fatalf("expected then-branch <p>, got %v", nodeDesc(thenP))
	}

	// Then an ElseNode.
	elseN := thenP.NextSibling
	if elseN == nil || elseN.Type != ElseNode {
		t.Fatalf("expected ElseNode, got %v", nodeDesc(elseN))
	}

	// ElseNode contains the "else" branch <p>.
	elseP := elseN.FirstChild
	if elseP == nil || elseP.Data != "p" {
		t.Fatalf("expected else-branch <p>, got %v", nodeDesc(elseP))
	}
}

func TestEachBlockStructure(t *testing.T) {
	doc := parseTemplate(t, `<ul>{each posts as post}<li>{post.Title}</li>{/each}</ul>`)

	ul := findElement(doc, "ul")
	if ul == nil {
		t.Fatal("expected <ul>")
	}

	eachN := ul.FirstChild
	if eachN == nil || eachN.Type != EachNode {
		t.Fatalf("expected EachNode, got %v", nodeDesc(eachN))
	}
	if eachN.TemplateData == nil || eachN.TemplateData.Binding != "post" {
		t.Errorf("expected Binding=post, got %+v", eachN.TemplateData)
	}

	// EachNode should contain the <li>.
	li := findElement(eachN, "li")
	if li == nil {
		t.Fatal("expected <li> inside EachNode")
	}
}

func TestBareEachIsError(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div>{each}<p>text</p>{/each}</div>`))
	if err == nil {
		t.Fatal("expected error for bare {each} with no binding")
	}
}

func TestEachElseBlockStructure(t *testing.T) {
	doc := parseTemplate(t, `<div>{each items as item}<p>item</p>{else}<p>empty</p>{/each}</div>`)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	eachN := div.FirstChild
	if eachN == nil || eachN.Type != EachNode {
		t.Fatalf("expected EachNode, got %v", nodeDesc(eachN))
	}

	// First child: the <p>item</p>.
	itemP := eachN.FirstChild
	if itemP == nil || itemP.Data != "p" {
		t.Fatalf("expected <p> in each body, got %v", nodeDesc(itemP))
	}

	// Then ElseNode.
	elseN := itemP.NextSibling
	if elseN == nil || elseN.Type != ElseNode {
		t.Fatalf("expected ElseNode, got %v", nodeDesc(elseN))
	}

	// ElseNode contains <p>empty</p>.
	emptyP := elseN.FirstChild
	if emptyP == nil || emptyP.Data != "p" {
		t.Fatalf("expected <p> in else branch, got %v", nodeDesc(emptyP))
	}
}

func TestSnippetBlockStructure(t *testing.T) {
	doc := parseTemplate(t, `<div>{snippet field(props)}<input type="text" />{/snippet}</div>`)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	snipN := div.FirstChild
	if snipN == nil || snipN.Type != SnippetNode {
		t.Fatalf("expected SnippetNode, got %v", nodeDesc(snipN))
	}
	if snipN.Data != "field" {
		t.Errorf("SnippetNode.Data = %q, want %q", snipN.Data, "field")
	}
	if snipN.TemplateData == nil || snipN.TemplateData.SnippetParams != "props" {
		t.Errorf("expected SnippetParams=props, got %+v", snipN.TemplateData)
	}

	// Should contain the <input>.
	input := findElement(snipN, "input")
	if input == nil {
		t.Fatal("expected <input> inside SnippetNode")
	}
}

func TestConstNode(t *testing.T) {
	doc := parseTemplate(t, `<div>{const count = 42}<p>{count}</p></div>`)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	constN := div.FirstChild
	if constN == nil || constN.Type != ConstNode {
		t.Fatalf("expected ConstNode, got %v", nodeDesc(constN))
	}
	if constN.TemplateData == nil || constN.TemplateData.ConstName != "count" {
		t.Errorf("expected ConstName=count, got %+v", constN.TemplateData)
	}
	if constN.Data != "42" {
		t.Errorf("ConstNode.Data = %q, want %q", constN.Data, "42")
	}
}

func TestNestedBlocks(t *testing.T) {
	input := `<div>{each posts as post}{if post.Featured}<p>featured</p>{/if}{/each}</div>`
	doc := parseTemplate(t, input)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	eachN := div.FirstChild
	if eachN == nil || eachN.Type != EachNode {
		t.Fatalf("expected EachNode, got %v", nodeDesc(eachN))
	}

	ifN := eachN.FirstChild
	if ifN == nil || ifN.Type != IfNode {
		t.Fatalf("expected IfNode inside EachNode, got %v", nodeDesc(ifN))
	}

	p := ifN.FirstChild
	if p == nil || p.Data != "p" {
		t.Fatalf("expected <p> inside IfNode, got %v", nodeDesc(p))
	}
}

func TestDeeplyNestedBlocks(t *testing.T) {
	input := `<div>{each categories as cat}{each cat.Posts as post}{if post.Featured}<p>{post.Title}</p>{else}<span>skip</span>{/if}{/each}{/each}</div>`
	doc := parseTemplate(t, input)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	// Level 1: each categories
	outerEach := div.FirstChild
	if outerEach == nil || outerEach.Type != EachNode {
		t.Fatalf("expected outer EachNode, got %v", nodeDesc(outerEach))
	}
	if outerEach.TemplateData == nil || outerEach.TemplateData.Binding != "cat" {
		t.Errorf("outer each binding = %+v, want cat", outerEach.TemplateData)
	}

	// Level 2: each cat.Posts
	innerEach := outerEach.FirstChild
	if innerEach == nil || innerEach.Type != EachNode {
		t.Fatalf("expected inner EachNode, got %v", nodeDesc(innerEach))
	}
	if innerEach.TemplateData == nil || innerEach.TemplateData.Binding != "post" {
		t.Errorf("inner each binding = %+v, want post", innerEach.TemplateData)
	}

	// Level 3: if post.Featured
	ifN := innerEach.FirstChild
	if ifN == nil || ifN.Type != IfNode {
		t.Fatalf("expected IfNode, got %v", nodeDesc(ifN))
	}
	if ifN.Data != "post.Featured" {
		t.Errorf("if condition = %q, want %q", ifN.Data, "post.Featured")
	}

	// Then branch: <p>
	thenP := ifN.FirstChild
	if thenP == nil || thenP.Type != ElementNode || thenP.Data != "p" {
		t.Fatalf("expected then-branch <p>, got %v", nodeDesc(thenP))
	}

	// Else branch
	elseN := thenP.NextSibling
	if elseN == nil || elseN.Type != ElseNode {
		t.Fatalf("expected ElseNode, got %v", nodeDesc(elseN))
	}
	elseSpan := elseN.FirstChild
	if elseSpan == nil || elseSpan.Data != "span" {
		t.Fatalf("expected else-branch <span>, got %v", nodeDesc(elseSpan))
	}

	// Nothing should leak outside the outer each.
	if outerEach.NextSibling != nil {
		t.Errorf("expected nothing after outer each, got %v", nodeDesc(outerEach.NextSibling))
	}
}

func TestIfInsideIfBlock(t *testing.T) {
	input := `<div>{if a}{if b}<p>both</p>{/if}{/if}</div>`
	doc := parseTemplate(t, input)

	div := findElement(doc, "div")
	if div == nil {
		t.Fatal("expected <div>")
	}

	outerIf := div.FirstChild
	if outerIf == nil || outerIf.Type != IfNode {
		t.Fatalf("expected outer IfNode, got %v", nodeDesc(outerIf))
	}

	innerIf := outerIf.FirstChild
	if innerIf == nil || innerIf.Type != IfNode {
		t.Fatalf("expected inner IfNode, got %v", nodeDesc(innerIf))
	}

	p := innerIf.FirstChild
	if p == nil || p.Data != "p" {
		t.Fatalf("expected <p> inside inner if, got %v", nodeDesc(p))
	}
}

func TestEachInsideEach(t *testing.T) {
	input := `<div>{each outer as o}{each o.Items as item}<p>item</p>{/each}{/each}</div>`
	doc := parseTemplate(t, input)

	div := findElement(doc, "div")
	outerEach := div.FirstChild
	if outerEach == nil || outerEach.Type != EachNode {
		t.Fatalf("expected outer EachNode, got %v", nodeDesc(outerEach))
	}

	innerEach := outerEach.FirstChild
	if innerEach == nil || innerEach.Type != EachNode {
		t.Fatalf("expected inner EachNode, got %v", nodeDesc(innerEach))
	}

	itemP := innerEach.FirstChild
	if itemP == nil || itemP.Data != "p" {
		t.Fatalf("expected <p> in inner each body, got %v", nodeDesc(itemP))
	}
}

func TestUnclosedIfBlock(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div>{if show}<p>text</p></div>`))
	if err == nil {
		t.Fatal("expected error for unclosed {if} block")
	}
	if !strings.Contains(err.Error(), "unclosed {if}") {
		t.Errorf("expected unclosed if error, got: %v", err)
	}
}

func TestUnclosedEachBlock(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div>{each items as item}<p>text</p></div>`))
	if err == nil {
		t.Fatal("expected error for unclosed {each} block")
	}
	if !strings.Contains(err.Error(), "unclosed {each}") {
		t.Errorf("expected unclosed each error, got: %v", err)
	}
}

func TestBlockBrokenByHTMLParsing(t *testing.T) {
	// A <div> inside a <p> causes the HTML5 parser to auto-close
	// the <p>, separating {if} from {/if}. The error message should
	// explain the HTML nesting issue, not just say "unclosed."
	_, err := ParseTemplate(strings.NewReader(`<p>{if show}<div>block</div>{/if}</p>`))
	if err == nil {
		t.Fatal("expected error for block broken by HTML parsing")
	}
	if !strings.Contains(err.Error(), "broken by HTML parsing") {
		t.Errorf("expected diagnostic about HTML parsing, got: %v", err)
	}
}

func TestMismatchedClose(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div>{if show}<p>text</p>{/each}</div>`))
	if err == nil {
		t.Fatal("expected error for mismatched close")
	}
}

func TestOrphanedClose(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div>{/if}</div>`))
	if err == nil {
		t.Fatal("expected error for orphaned {/if}")
	}
}

func TestOrphanedElse(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div>{else}</div>`))
	if err == nil {
		t.Fatal("expected error for orphaned {else}")
	}
}

func TestScriptRejected(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div><script>alert("xss")</script></div>`))
	if err == nil {
		t.Fatal("expected error for <script> in template")
	}
	if !strings.Contains(err.Error(), "<script>") {
		t.Errorf("expected script error, got: %v", err)
	}
}

func TestStyleRejected(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<div><style>body { color: red }</style></div>`))
	if err == nil {
		t.Fatal("expected error for <style> in template")
	}
	if !strings.Contains(err.Error(), "<style>") {
		t.Errorf("expected style error, got: %v", err)
	}
}

func TestAttrExprCompiled(t *testing.T) {
	doc := parseTemplate(t, `<a href="{URL}">link</a>`)
	a := findElement(doc, "a")
	if a == nil {
		t.Fatal("expected <a>")
	}
	if len(a.Attr) == 0 {
		t.Fatal("expected attributes on <a>")
	}
	attr := a.Attr[0]
	if attr.Key != "href" {
		t.Fatalf("expected href, got %q", attr.Key)
	}
	if attr.Parts == nil {
		t.Fatal("expected Parts to be compiled for attribute with expression")
	}
	if len(attr.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(attr.Parts))
	}
	if attr.Parts[0].Expr == nil {
		t.Fatal("expected expression part")
	}
}

func TestAttrExprMixed(t *testing.T) {
	doc := parseTemplate(t, `<a href="/search?q={Query}">link</a>`)
	a := findElement(doc, "a")
	if a == nil {
		t.Fatal("expected <a>")
	}
	attr := a.Attr[0]
	if attr.Parts == nil {
		t.Fatal("expected Parts to be compiled")
	}
	if len(attr.Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(attr.Parts))
	}
	if attr.Parts[0].Text != "/search?q=" {
		t.Errorf("expected static prefix, got %q", attr.Parts[0].Text)
	}
	if attr.Parts[1].Expr == nil {
		t.Fatal("expected expression part")
	}
}

func TestAttrStaticNoParts(t *testing.T) {
	doc := parseTemplate(t, `<a href="/about">link</a>`)
	a := findElement(doc, "a")
	if a == nil {
		t.Fatal("expected <a>")
	}
	attr := a.Attr[0]
	if attr.Parts != nil {
		t.Fatal("expected nil Parts for static attribute")
	}
}

func TestAttrExprInvalid(t *testing.T) {
	_, err := ParseTemplate(strings.NewReader(`<a href="{> invalid}">link</a>`))
	if err == nil {
		t.Fatal("expected error for invalid expression in attribute")
	}
}

func TestBlockRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantContains string
	}{
		{
			"if block",
			`<div>{if show}<p>yes</p>{/if}</div>`,
			"{if show}<p>yes</p>{/if}",
		},
		{
			"if-else block",
			`<div>{if show}<p>yes</p>{else}<p>no</p>{/if}</div>`,
			"{if show}<p>yes</p>{else}<p>no</p>{/if}",
		},
		{
			"each block",
			`<ul>{each posts as post}<li>item</li>{/each}</ul>`,
			"{each posts as post}<li>item</li>{/each}",
		},
		{
			"snippet block",
			`<div>{snippet field(props)}<span>text</span>{/snippet}</div>`,
			"{snippet field(props)}<span>text</span>{/snippet}",
		},
		{
			"const",
			`<div>{const count = 42}</div>`,
			"{const count = 42}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseTemplate(t, tt.input)

			var buf bytes.Buffer
			if err := Render(&buf, doc); err != nil {
				t.Fatalf("Render error: %v", err)
			}
			if !strings.Contains(buf.String(), tt.wantContains) {
				t.Errorf("rendered %q does not contain %q", buf.String(), tt.wantContains)
			}
		})
	}
}

func TestSprint(t *testing.T) {
	doc := parseTemplate(t, `<div>{if show}<p>yes</p>{else}<p>no</p>{/if}</div>`)

	got := Sprint(doc)
	want := `(document
  (element "html"
    (element "head")
    (element "body"
      (element "div"
        (if (ident "show")
          (element "p"
            (text "yes")
          )
          (else
            (element "p"
              (text "no")
            )
          )
        )
      )
    )
  )
)
`
	if got != want {
		t.Errorf("Sprint mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSprintAllNodeTypes(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{
			"expression",
			`<p>{name}</p>`,
			`(expr (ident "name"))`,
		},
		{
			"const",
			`<p>{const x = 42}</p>`,
			`(const "x" (number 42))`,
		},
		{
			"each with index",
			`<p>{each items as item, i}<span>x</span>{/each}</p>`,
			`(each (ident "items") :as "item" :index "i"`,
		},
		{
			"snippet with params",
			`<p>{snippet field(props)}<span>x</span>{/snippet}</p>`,
			`(snippet "field" :params "props"`,
		},
		{
			"vocabulary tag with attrs",
			`<comment-form post-id="42" />`,
			`(element "comment-form" (post-id "42"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseTemplate(t, tt.input)
			got := Sprint(doc)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Sprint output does not contain %q\ngot:\n%s", tt.want, got)
			}
		})
	}
}

func TestSprintExpr(t *testing.T) {
	var sb strings.Builder

	tests := []struct {
		name string
		expr Expr
		want string
	}{
		{"ident", &Ident{Name: "name"}, `(ident "name")`},
		{"member", &MemberExpr{Object: &Ident{Name: "post"}, Field: "Title"}, `(. (ident "post") "Title")`},
		{"nested member", &MemberExpr{
			Object: &MemberExpr{Object: &Ident{Name: "post"}, Field: "Author"},
			Field:  "Name",
		}, `(. (. (ident "post") "Author") "Name")`},
		{"string", &StringLit{Value: "hello"}, `(string "hello")`},
		{"number", &NumberLit{Value: 42}, `(number 42)`},
		{"float", &NumberLit{Value: 3.14}, `(number 3.14)`},
		{"bool true", &BoolLit{Value: true}, `(bool true)`},
		{"bool false", &BoolLit{Value: false}, `(bool false)`},
		{"binary ==", &BinaryExpr{
			Left: &Ident{Name: "x"}, Op: "==", Right: &NumberLit{Value: 0},
		}, `(== (ident "x") (number 0))`},
		{"binary >", &BinaryExpr{
			Left:  &MemberExpr{Object: &Ident{Name: "post"}, Field: "Count"},
			Op:    ">",
			Right: &NumberLit{Value: 0},
		}, `(> (. (ident "post") "Count") (number 0))`},
		{"binary and", &BinaryExpr{
			Left: &Ident{Name: "a"}, Op: "and", Right: &Ident{Name: "b"},
		}, `(and (ident "a") (ident "b"))`},
		{"binary or", &BinaryExpr{
			Left: &Ident{Name: "a"}, Op: "or", Right: &Ident{Name: "b"},
		}, `(or (ident "a") (ident "b"))`},
		{"unary not", &UnaryExpr{Op: "not", Operand: &Ident{Name: "show"}}, `(not (ident "show"))`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb.Reset()
			sprintExpr(&sb, tt.expr)
			got := sb.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSprintCompiledAttrs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"single expression attr",
			`<a href="{URL}">link</a>`,
			`(href (ident "URL"))`,
		},
		{
			"member expression attr",
			`<a href="{post.Permalink}">link</a>`,
			`(href (. (ident "post") "Permalink"))`,
		},
		{
			"mixed attr",
			`<a href="/search?q={Query}">link</a>`,
			`(href "/search?q=" (ident "Query"))`,
		},
		{
			"static attr unchanged",
			`<a href="/about">link</a>`,
			`(href "/about")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseTemplate(t, tt.input)
			got := Sprint(doc)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Sprint output does not contain %q\ngot:\n%s", tt.want, got)
			}
		})
	}
}

func TestSprintSkipsWhitespace(t *testing.T) {
	doc := parseTemplate(t, "<div>  \n  <p>text</p>  \n  </div>")
	got := Sprint(doc)
	if strings.Contains(got, "text \"") && strings.Count(got, "(text") != 1 {
		t.Errorf("expected whitespace-only text nodes to be skipped\n%s", got)
	}
	// Should have exactly one text node for "text"
	if strings.Count(got, "(text") != 1 {
		t.Errorf("expected exactly one text node, got:\n%s", got)
	}
}

func nodeDesc(n *Node) string {
	if n == nil {
		return "<nil>"
	}
	return fmt.Sprintf("{Type:%d Data:%q}", n.Type, n.Data)
}

// nodeCheck is a simplified node for test assertions.
type nodeCheck struct {
	Type NodeType
	Data string
}

// parseHTML parses input and fails the test on error.
func parseHTML(t *testing.T, input string) *Node {
	t.Helper()
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	return doc
}

// findNode does a depth-first search for a node of the given type.
func findNode(n *Node, nodeType NodeType) *Node {
	if n.Type == nodeType {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, nodeType); found != nil {
			return found
		}
	}
	return nil
}

// findElement finds the first element with the given tag name.
func findElement(n *Node, tag string) *Node {
	if n.Type == ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// collectExpressions collects all ExpressionNode Data values in tree order.
func collectExpressions(n *Node, out *[]string) {
	if n.Type == ExpressionNode {
		*out = append(*out, n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectExpressions(c, out)
	}
}
