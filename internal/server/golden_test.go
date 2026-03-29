package server

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"press/internal/auth"
)

var update = flag.Bool("update", false, "update golden files")

const goldenThemeDir = "../../local/themes/freerange"

// goldenTheme loads the freerange theme with real tag handlers registered.
func goldenTheme(t *testing.T) *Theme {
	t.Helper()
	th, err := LoadTheme(goldenThemeDir, false)
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	s := &Server{theme: th}
	s.registerTagHandlers()
	return th
}

// goldenCSRFContext returns a context with a CSRF helper so
// tagCommentForm can generate tokens.
func goldenCSRFContext() context.Context {
	ctx := context.Background()
	return auth.WithCSRF(ctx, auth.NewCSRFHelper("test-session", "test-secret"))
}

// --- Fixture data using real server view types ---

var gFixturePost = PostView{
	ID:         1,
	TheTitle:   "Hello World",
	TheContent: "<p>Welcome to Press.</p>",
	TheExcerpt: "Welcome to Press.",
	TheDate:    "January 1, 2004",
	TheAuthor:  "admin",
	TheCategories: []CategoryLink{
		{Name: "Uncategorized", URL: "/category/uncategorized"},
		{Name: "News", URL: "/category/news"},
	},
	Permalink:    "/2004/01/hello-world",
	CommentCount: 2,
	CommentsOpen: true,
}

var gFixturePost2 = PostView{
	ID:            2,
	TheTitle:      "Second Post",
	TheContent:    "<p>Another post.</p>",
	TheDate:       "January 2, 2004",
	TheAuthor:     "admin",
	TheCategories: []CategoryLink{},
	Permalink:     "/2004/01/second-post",
	CommentsOpen:  false,
}

var gFixtureComment = CommentView{
	ID:         10,
	TheAuthor:  "Alice",
	URL:        "https://alice.example.com",
	TheDate:    "January 2, 2004",
	TheContent: "<p>Great post!</p>",
}

var gFixtureComment2 = CommentView{
	ID:         11,
	TheAuthor:  "Bob",
	URL:        "",
	TheDate:    "January 3, 2004",
	TheContent: "<p>Thanks for sharing.</p>",
}

var gFixtureCategories = []CategoryView{
	{Name: "Uncategorized", Slug: "uncategorized", URL: "/category/uncategorized", Count: 5},
	{Name: "News", Slug: "news", URL: "/category/news", Count: 3},
}

var gFixtureArchives = []ArchiveView{
	{Label: "January 2004", URL: "/2004/01", Count: 2},
	{Label: "February 2004", URL: "/2004/02", Count: 1},
}

var gFixturePages = []PageLink{
	{Title: "About", URL: "/about"},
	{Title: "Contact", URL: "/contact"},
}

func gSiteData() SiteData {
	return SiteData{
		BlogName:        "Test Blog",
		BlogDescription: "Just another Press blog",
		Categories:      gFixtureCategories,
		Archives:        gFixtureArchives,
		Pages:           gFixturePages,
		IsLoggedIn:      true,
		LoginURL:        "/wp-login.php",
		LogoutURL:       "/wp-login.php?action=logout",
		AdminURL:        "/wp-admin/",
	}
}

// --- Page tests ---

func TestGoldenRenderedPages(t *testing.T) {
	th := goldenTheme(t)

	tests := []struct {
		name string
		ctx  context.Context
		data any
	}{
		{
			name: "single",
			ctx:  goldenCSRFContext(),
			data: SingleData{
				SiteData:     gSiteData(),
				Post:         gFixturePost,
				PostID:       1,
				Comments:     []CommentView{gFixtureComment, gFixtureComment2},
				CommentsOpen: true,
				PrevPost:     &PageLink{Title: "Earlier", URL: "/earlier"},
				NextPost:     nil,
			},
		},
		{
			name: "single-no-comments",
			ctx:  goldenCSRFContext(),
			data: SingleData{
				SiteData:     gSiteData(),
				Post:         gFixturePost2,
				PostID:       2,
				Comments:     []CommentView{},
				CommentsOpen: false,
			},
		},
		{
			name: "home",
			ctx:  context.Background(),
			data: HomeData{
				SiteData: gSiteData(),
				Posts:    []PostView{gFixturePost, gFixturePost2},
				HasNext:  true,
				NextURL:  "/page/2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Page templates share a file name with the test name,
			// except variants like single-no-comments use "single".
			tmplName := tt.name
			if mapped, ok := renderedPageFileMap[tmplName]; ok {
				tmplName = mapped
			}

			var buf bytes.Buffer
			if err := th.Render(&buf, tt.ctx, tmplName, tt.data); err != nil {
				t.Fatalf("Render %q: %v", tmplName, err)
			}
			goldenHTML(t, tt.name, buf.String())
		})
	}
}

var renderedPageFileMap = map[string]string{
	"single-no-comments": "single",
}

// --- Tag tests ---

func TestGoldenRenderedTags(t *testing.T) {
	th := goldenTheme(t)

	tests := []struct {
		name string
		tag  string
		ctx  context.Context
		data any
	}{
		{
			name: "comment-form",
			tag:  "comment-form",
			ctx:  goldenCSRFContext(),
			data: struct {
				PostID int64 `view:"post_id"`
			}{PostID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := th.RenderTag(&buf, tt.ctx, tt.tag, tt.data); err != nil {
				t.Fatalf("RenderTag %q: %v", tt.tag, err)
			}
			goldenHTML(t, tt.name, buf.String())
		})
	}
}

// --- Golden HTML comparison ---

func goldenHTML(t *testing.T, name, got string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", "rendered", name+".golden")

	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
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
		t.Fatalf("missing golden file (run with -args -update to create): %v", err)
	}

	if got != string(want) {
		// Show first difference line for easier debugging.
		t.Errorf("HTML mismatch for %s\n\ngot:\n%s\nwant:\n%s", name, abbreviate(got), abbreviate(string(want)))
	}
}

// abbreviate returns the first 200 lines of s for readable test output.
func abbreviate(s string) string {
	lines := 0
	for i, c := range s {
		if c == '\n' {
			lines++
			if lines >= 200 {
				return s[:i] + fmt.Sprintf("\n... (%d more bytes)", len(s)-i)
			}
		}
	}
	return s
}
