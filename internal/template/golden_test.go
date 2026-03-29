package template

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"press/internal/template/parse"
)

var update = flag.Bool("update", false, "update golden files")

// themeDir is the path to the freerange theme relative to the module root.
const themeDir = "../../local/themes/freerange"

// --- Fixture data types (mirror server view structs with view tags) ---

type categoryLink struct {
	Name string `view:"name"`
	URL  string `view:"url"`
}

type categoryView struct {
	Name  string `view:"name"`
	Slug  string `view:"slug"`
	URL   string `view:"url"`
	Count int    `view:"count"`
}

type archiveView struct {
	Label string `view:"label"`
	URL   string `view:"url"`
	Count int    `view:"count"`
}

type pageLink struct {
	Title string `view:"title"`
	URL   string `view:"url"`
}

type postView struct {
	ID            int64          `view:"id"`
	TheTitle      string         `view:"the_title"`
	TheContent    string         `view:"the_content,raw"`
	TheExcerpt    string         `view:"the_excerpt"`
	TheDate       string         `view:"the_date"`
	TheAuthor     string         `view:"the_author"`
	TheCategories []categoryLink `view:"the_categories"`
	Permalink     string         `view:"permalink"`
	CommentCount  int            `view:"comment_count"`
	CommentsOpen  bool           `view:"comments_open"`
}

type commentView struct {
	ID         int64  `view:"id"`
	TheAuthor  string `view:"the_author"`
	URL        string `view:"url"`
	TheDate    string `view:"the_date"`
	TheContent string `view:"the_content,raw"`
}

// --- Fixture data ---

var fixturePost = postView{
	ID:         1,
	TheTitle:   "Hello World",
	TheContent: "<p>Welcome to Press.</p>",
	TheExcerpt: "Welcome to Press.",
	TheDate:    "January 1, 2004",
	TheAuthor:  "admin",
	TheCategories: []categoryLink{
		{Name: "Uncategorized", URL: "/category/uncategorized"},
		{Name: "News", URL: "/category/news"},
	},
	Permalink:    "/2004/01/hello-world",
	CommentCount: 2,
	CommentsOpen: true,
}

var fixturePost2 = postView{
	ID:            2,
	TheTitle:      "Second Post",
	TheContent:    "<p>Another post.</p>",
	TheDate:       "January 2, 2004",
	TheAuthor:     "admin",
	TheCategories: []categoryLink{},
	Permalink:     "/2004/01/second-post",
	CommentsOpen:  false,
}

var fixtureComment = commentView{
	ID:         10,
	TheAuthor:  "Alice",
	URL:        "https://alice.example.com",
	TheDate:    "January 2, 2004",
	TheContent: "<p>Great post!</p>",
}

var fixtureComment2 = commentView{
	ID:         11,
	TheAuthor:  "Bob",
	URL:        "",
	TheDate:    "January 3, 2004",
	TheContent: "<p>Thanks for sharing.</p>",
}

var fixtureCategories = []categoryView{
	{Name: "Uncategorized", Slug: "uncategorized", URL: "/category/uncategorized", Count: 5},
	{Name: "News", Slug: "news", URL: "/category/news", Count: 3},
}

var fixtureArchives = []archiveView{
	{Label: "January 2004", URL: "/2004/01", Count: 2},
	{Label: "February 2004", URL: "/2004/02", Count: 1},
}

var fixturePages = []pageLink{
	{Title: "About", URL: "/about"},
	{Title: "Contact", URL: "/contact"},
}

// siteData returns a fixture struct matching SiteData fields.
func siteData() siteFixture {
	return siteFixture{
		BlogName:        "Test Blog",
		BlogDescription: "Just another Press blog",
		Categories:      fixtureCategories,
		Archives:        fixtureArchives,
		Pages:           fixturePages,
		IsLoggedIn:      true,
		LoginURL:        "/wp-login.php",
		LogoutURL:       "/wp-login.php?action=logout",
		AdminURL:        "/wp-admin/",
	}
}

type siteFixture struct {
	BlogName        string         `view:"blog_name"`
	BlogDescription string         `view:"blog_description"`
	PageTitle       string         `view:"page_title"`
	Categories      []categoryView `view:"categories"`
	Archives        []archiveView  `view:"archives"`
	Pages           []pageLink     `view:"pages"`
	IsLoggedIn      bool           `view:"is_logged_in"`
	LoginURL        string         `view:"login_url"`
	LogoutURL       string         `view:"logout_url"`
	AdminURL        string         `view:"admin_url"`
	SearchQuery     string         `view:"search_query"`
}

// --- Molecule tests (no handlers needed) ---

type moleculeTest struct {
	name string // template file name (without .html)
	dir  string // subdirectory under theme (molecules, organisms, templates)
	data any    // fixture data for the root scope
}

var moleculeTests = []moleculeTest{
	{
		name: "post",
		dir:  "molecules",
		data: fixturePost,
	},
	{
		name: "post-with-no-categories",
		dir:  "molecules",
		data: postView{
			ID:            3,
			TheTitle:      "No Categories",
			TheContent:    "<p>A post with no categories.</p>",
			TheDate:       "January 5, 2004",
			TheAuthor:     "admin",
			TheCategories: []categoryLink{},
			Permalink:     "/2004/01/no-categories",
		},
	},
	{
		name: "comment-with-url",
		dir:  "molecules",
		data: fixtureComment,
	},
	{
		name: "comment-without-url",
		dir:  "molecules",
		data: commentView{
			ID:         12,
			TheAuthor:  "Charlie",
			URL:        "",
			TheDate:    "January 4, 2004",
			TheContent: "<p>No website.</p>",
		},
	},
	{
		name: "pagination-both",
		dir:  "molecules",
		data: struct {
			HasPrev bool   `view:"has_prev"`
			HasNext bool   `view:"has_next"`
			PrevURL string `view:"prev_url"`
			NextURL string `view:"next_url"`
		}{
			HasPrev: true,
			HasNext: true,
			PrevURL: "/page/1",
			NextURL: "/page/3",
		},
	},
	{
		name: "pagination-next-only",
		dir:  "molecules",
		data: struct {
			HasPrev bool   `view:"has_prev"`
			HasNext bool   `view:"has_next"`
			PrevURL string `view:"prev_url"`
			NextURL string `view:"next_url"`
		}{
			HasPrev: false,
			HasNext: true,
			NextURL: "/page/2",
		},
	},
	{
		name: "category-list",
		dir:  "molecules",
		data: siteData(),
	},
	{
		name: "archive-list",
		dir:  "molecules",
		data: siteData(),
	},
	{
		name: "page-list",
		dir:  "molecules",
		data: siteData(),
	},
	{
		name: "search-form",
		dir:  "molecules",
		data: struct {
			SearchQuery string `view:"search_query"`
		}{SearchQuery: "hello"},
	},
	{
		name: "meta-links-logged-in",
		dir:  "molecules",
		data: siteData(),
	},
	{
		name: "meta-links-logged-out",
		dir:  "molecules",
		data: func() siteFixture {
			s := siteData()
			s.IsLoggedIn = false
			return s
		}(),
	},
	{
		name: "archive-header-with-description",
		dir:  "molecules",
		data: struct {
			ArchiveTitle       string `view:"archive_title"`
			ArchiveDescription string `view:"archive_description"`
		}{
			ArchiveTitle:       "January 2004",
			ArchiveDescription: "Posts from January.",
		},
	},
	{
		name: "archive-header-no-description",
		dir:  "molecules",
		data: struct {
			ArchiveTitle       string `view:"archive_title"`
			ArchiveDescription string `view:"archive_description"`
		}{
			ArchiveTitle: "January 2004",
		},
	},
	{
		name: "comment-form",
		dir:  "molecules",
		data: struct {
			Author string `view:"author"`
			Email  string `view:"email"`
			URL    string `view:"url"`
		}{},
	},
	{
		name: "post-navigation-both",
		dir:  "molecules",
		data: struct {
			PrevPost *pageLink `view:"prev_post"`
			NextPost *pageLink `view:"next_post"`
		}{
			PrevPost: &pageLink{Title: "Earlier Post", URL: "/earlier"},
			NextPost: &pageLink{Title: "Later Post", URL: "/later"},
		},
	},
	{
		name: "post-navigation-none",
		dir:  "molecules",
		data: struct {
			PrevPost *pageLink `view:"prev_post"`
			NextPost *pageLink `view:"next_post"`
		}{},
	},
}

// templateFileForTest maps test names to template files. Tests that
// share a template but use different data have the suffix stripped.
var templateFileMap = map[string]string{
	"post-with-no-categories":         "post",
	"comment-with-url":                "comment",
	"comment-without-url":             "comment",
	"pagination-both":                 "pagination",
	"pagination-next-only":            "pagination",
	"meta-links-logged-in":            "meta-links",
	"meta-links-logged-out":           "meta-links",
	"archive-header-with-description": "archive-header",
	"archive-header-no-description":   "archive-header",
	"post-navigation-both":            "post-navigation",
	"post-navigation-none":            "post-navigation",
}

func templateFile(name string) string {
	if mapped, ok := templateFileMap[name]; ok {
		return mapped
	}
	return name
}

func TestGoldenMolecules(t *testing.T) {
	for _, tt := range moleculeTests {
		t.Run(tt.name, func(t *testing.T) {
			tmplName := templateFile(tt.name)
			path := filepath.Join(themeDir, tt.dir, tmplName+".html")
			runGoldenEval(t, tt.name, path, tt.data, nil)
		})
	}
}

// --- Organism tests (need handlers to resolve vocabulary tags) ---

// nodeTemplateHandler returns a TagNodeHandler that loads and evaluates
// a sub-template from the theme, returning the result as a node tree.
func nodeTemplateHandler(dir, name string) TagNodeHandler {
	return func(ctx context.Context, ev *Evaluator, el *parse.Node) *parse.Node {
		path := filepath.Join(themeDir, dir, name+".html")
		return evalSubTemplate(ev, path, nil)
	}
}

// engineAttrHandler returns a TagNodeHandler that loads a sub-template
// and injects engine attributes onto the wrapper element, mirroring
// what EvalTagTemplate does with engineAttrs in production.
func engineAttrHandler(dir, name string, attrs map[string]string) TagNodeHandler {
	return func(ctx context.Context, ev *Evaluator, el *parse.Node) *parse.Node {
		path := filepath.Join(themeDir, dir, name+".html")
		result := evalSubTemplate(ev, path, nil)
		injectAttrs(result, attrs)
		return result
	}
}

// injectAttrs adds attributes to the first element child of a node
// in sorted key order for deterministic golden output.
func injectAttrs(n *parse.Node, attrs map[string]string) {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == parse.ElementNode {
			for _, k := range keys {
				c.Attr = append(c.Attr, parse.Attribute{Key: k, Val: attrs[k]})
			}
			break
		}
	}
}

// commentFormHandler returns a TagNodeHandler that mirrors the real
// tagCommentForm handler: the engine builds the <form> element with
// routing and htmx attributes, adds hidden inputs for post ID and
// CSRF token, and evaluates the theme template (visible fields) into
// the form as children.
func commentFormHandler() TagNodeHandler {
	return func(ctx context.Context, ev *Evaluator, el *parse.Node) *parse.Node {
		postIDVal, _ := ev.Scope().Lookup("post_id")
		postID, _ := postIDVal.(int64)
		csrfVal, _ := ev.Scope().Lookup("csrf_token")
		csrf, _ := csrfVal.(string)

		data := struct {
			Author string `view:"author"`
			Email  string `view:"email"`
			URL    string `view:"url"`
		}{}
		path := filepath.Join(themeDir, "molecules", "comment-form.html")
		body := evalSubTemplate(ev, path, data)

		fields := &parse.Node{Type: parse.DocumentNode}
		fields.AppendChild(HiddenInput("comment_post_id", fmt.Sprintf("%d", postID)))
		fields.AppendChild(HiddenInput("_csrf", csrf))
		for c := body.FirstChild; c != nil; {
			next := c.NextSibling
			body.RemoveChild(c)
			fields.AppendChild(c)
			c = next
		}

		return BuildElement("form", []parse.Attribute{
			{Key: "method", Val: "post"},
			{Key: "action", Val: "/comments"},
			{Key: "hx-post", Val: "/comments"},
			{Key: "hx-target", Val: "this"},
			{Key: "hx-swap", Val: "outerHTML"},
			{Key: "hx-disabled-elt", Val: "this, find [type='submit']"},
		}, fields)
	}
}

// scopedNodeHandler returns a TagNodeHandler that looks up a binding
// from scope, pushes its view fields into a child scope, and evaluates
// a sub-template. This mirrors what real tag handlers like tagPost and
// tagComment do.
func scopedNodeHandler(dir, tmpl, binding string) TagNodeHandler {
	return func(ctx context.Context, ev *Evaluator, el *parse.Node) *parse.Node {
		val, ok := ev.Scope().Lookup(binding)
		if !ok {
			return nil
		}
		path := filepath.Join(themeDir, dir, tmpl+".html")
		return evalSubTemplate(ev, path, val)
	}
}

func evalSubTemplate(ev *Evaluator, path string, data any) *parse.Node {
	f, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("test: open template %s: %v", path, err))
	}
	defer func() { _ = f.Close() }()

	doc, err := parse.ParseTemplateFragment(f)
	if err != nil {
		panic(fmt.Sprintf("test: parse template %s: %v", path, err))
	}

	if data != nil {
		return ev.EvaluateScoped(doc, data)
	}

	out := newDocumentNode()
	if err := ev.EvaluateChildren(out, doc); err != nil {
		panic(fmt.Sprintf("test: evaluate template %s: %v", path, err))
	}
	return out
}

func themeResolver() *testResolver {
	return &testResolver{handlers: map[string]TagNodeHandler{
		"post":               scopedNodeHandler("molecules", "post", "post"),
		"comment":            scopedNodeHandler("molecules", "comment", "comment"),
		"pagination":         nodeTemplateHandler("molecules", "pagination"),
		"post-navigation":    nodeTemplateHandler("molecules", "post-navigation"),
		"comment-form":       commentFormHandler(),
		"comment-list":       nodeTemplateHandler("organisms", "comment-list"),
		"post-list":          nodeTemplateHandler("organisms", "post-list"),
		"sidebar":            nodeTemplateHandler("organisms", "sidebar"),
		"search-form":        nodeTemplateHandler("molecules", "search-form"),
		"category-list":      nodeTemplateHandler("molecules", "category-list"),
		"archive-list":       nodeTemplateHandler("molecules", "archive-list"),
		"page-list":          nodeTemplateHandler("molecules", "page-list"),
		"meta-links":         nodeTemplateHandler("molecules", "meta-links"),
		"archive-header":     nodeTemplateHandler("molecules", "archive-header"),
		"comment-list-empty": engineAttrHandler("molecules", "comment-list-empty", map[string]string{"data-empty": ""}),
	}}
}

type organismTest struct {
	name string
	dir  string
	data any
}

var organismTests = []organismTest{
	{
		name: "comment-list",
		dir:  "organisms",
		data: struct {
			Comments []commentView `view:"comments"`
		}{
			Comments: []commentView{fixtureComment, fixtureComment2},
		},
	},
	{
		name: "comment-list-empty",
		dir:  "organisms",
		data: struct {
			Comments []commentView `view:"comments"`
		}{},
	},
	{
		name: "post-list",
		dir:  "organisms",
		data: struct {
			Posts []postView `view:"posts"`
		}{
			Posts: []postView{fixturePost, fixturePost2},
		},
	},
	{
		name: "post-list-empty",
		dir:  "organisms",
		data: struct {
			Posts []postView `view:"posts"`
		}{},
	},
	{
		name: "sidebar",
		dir:  "organisms",
		data: siteData(),
	},
}

var organismFileMap = map[string]string{
	"comment-list-empty": "comment-list",
	"post-list-empty":    "post-list",
}

func TestGoldenOrganisms(t *testing.T) {
	resolver := themeResolver()
	for _, tt := range organismTests {
		t.Run(tt.name, func(t *testing.T) {
			tmplName := tt.name
			if mapped, ok := organismFileMap[tmplName]; ok {
				tmplName = mapped
			}
			path := filepath.Join(themeDir, tt.dir, tmplName+".html")
			runGoldenEval(t, tt.name, path, tt.data, resolver)
		})
	}
}

// --- Page template tests ---

type homeData struct {
	siteFixture
	Posts   []postView `view:"posts"`
	HasPrev bool       `view:"has_prev"`
	HasNext bool       `view:"has_next"`
	PrevURL string     `view:"prev_url"`
	NextURL string     `view:"next_url"`
}

type singleData struct {
	siteFixture
	Post         postView      `view:"post"`
	PostID       int64         `view:"post_id"`
	Comments     []commentView `view:"comments"`
	CommentsOpen bool          `view:"comments_open"`
	CSRFToken    string        `view:"csrf_token"`
	PrevPost     *pageLink     `view:"prev_post"`
	NextPost     *pageLink     `view:"next_post"`
}

type pageData struct {
	siteFixture
	Page struct {
		TheTitle   string `view:"the_title"`
		TheContent string `view:"the_content,raw"`
	} `view:"page"`
}

type archiveData struct {
	siteFixture
	ArchiveTitle       string     `view:"archive_title"`
	ArchiveDescription string     `view:"archive_description"`
	Posts              []postView `view:"posts"`
	HasPrev            bool       `view:"has_prev"`
	HasNext            bool       `view:"has_next"`
	PrevURL            string     `view:"prev_url"`
	NextURL            string     `view:"next_url"`
}

type searchData struct {
	siteFixture
	Posts   []postView `view:"posts"`
	HasPrev bool       `view:"has_prev"`
	HasNext bool       `view:"has_next"`
	PrevURL string     `view:"prev_url"`
	NextURL string     `view:"next_url"`
}

var pageTemplateTests = []organismTest{
	{
		name: "home",
		dir:  "templates",
		data: homeData{
			siteFixture: siteData(),
			Posts:       []postView{fixturePost, fixturePost2},
			HasNext:     true,
			NextURL:     "/page/2",
		},
	},
	{
		name: "home-empty",
		dir:  "templates",
		data: homeData{
			siteFixture: siteData(),
			Posts:       []postView{},
		},
	},
	{
		name: "single",
		dir:  "templates",
		data: singleData{
			siteFixture:  siteData(),
			Post:         fixturePost,
			PostID:       1,
			Comments:     []commentView{fixtureComment, fixtureComment2},
			CommentsOpen: true,
			CSRFToken:    "test-csrf-token",
			PrevPost:     &pageLink{Title: "Earlier", URL: "/earlier"},
			NextPost:     nil,
		},
	},
	{
		name: "page",
		dir:  "templates",
		data: func() pageData {
			d := pageData{siteFixture: siteData()}
			d.Page.TheTitle = "About"
			d.Page.TheContent = "<p>About this blog.</p>"
			return d
		}(),
	},
	{
		name: "archive",
		dir:  "templates",
		data: archiveData{
			siteFixture:        siteData(),
			ArchiveTitle:       "January 2004",
			ArchiveDescription: "Posts from January.",
			Posts:              []postView{fixturePost},
			HasNext:            true,
			NextURL:            "/2004/01/page/2",
		},
	},
	{
		name: "search",
		dir:  "templates",
		data: func() searchData {
			s := siteData()
			s.SearchQuery = "hello"
			return searchData{
				siteFixture: s,
				Posts:       []postView{fixturePost},
			}
		}(),
	},
	{
		name: "404",
		dir:  "templates",
		data: siteData(),
	},
}

var pageFileMap = map[string]string{
	"home-empty": "home",
}

func TestGoldenPages(t *testing.T) {
	resolver := themeResolver()
	for _, tt := range pageTemplateTests {
		t.Run(tt.name, func(t *testing.T) {
			tmplName := tt.name
			if mapped, ok := pageFileMap[tmplName]; ok {
				tmplName = mapped
			}
			path := filepath.Join(themeDir, tt.dir, tmplName+".html")
			runGoldenEval(t, tt.name, path, tt.data, resolver)
		})
	}
}

// --- Golden test runner ---

func runGoldenEval(t *testing.T, name, templatePath string, data any, resolver TagResolver) {
	t.Helper()
	runGoldenEvalDir(t, name, templatePath, data, resolver, "testdata")
}

func runGoldenEvalDir(t *testing.T, name, templatePath string, data any, resolver TagResolver, goldenDir string) {
	t.Helper()

	f, err := os.Open(templatePath)
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	defer func() { _ = f.Close() }()

	doc, err := parse.ParseTemplateFragment(f)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	ctx := context.Background()
	if resolver != nil {
		ctx = WithTagResolver(ctx, resolver)
	}
	result := Evaluate(ctx, doc, data)

	goldenCompare(t, name, goldenDir, result)
}

func goldenCompare(t *testing.T, name, goldenDir string, result *parse.Node) {
	t.Helper()

	got := parse.Sprint(result)
	goldenPath := filepath.Join(goldenDir, name+".golden")

	if *update {
		if err := os.MkdirAll(goldenDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("missing golden file (run with -update to create): %v", err)
	}

	if got != string(want) {
		t.Errorf("AST mismatch for %s\n\ngot:\n%s\nwant:\n%s", name, got, string(want))
	}
}
