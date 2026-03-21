package server

import (
	"encoding/json"
	"html/template"
	"log"
	"strings"

	"press/internal/model"
	"press/internal/permalink"
	"press/internal/prosemirror"
)

// SiteData is shared context available on every page.
type SiteData struct {
	BlogName        string
	BlogDescription string
	PageTitle       string

	// Sidebar
	Categories []CategoryView
	Archives   []ArchiveView
	Pages      []PageLink

	// Auth
	IsLoggedIn bool
	LoginURL   string
	LogoutURL  string
	AdminURL   string

	// Search
	SearchQuery string

	// Sidebar context (archive/search pages)
	SidebarContext string
}

// CategoryView is a category for the sidebar.
type CategoryView struct {
	Name  string
	Slug  string
	URL   string
	Count int
}

// CategoryLink is a category as displayed in post metadata.
type CategoryLink struct {
	Name string
	URL  string
}

// ArchiveView is a monthly archive entry for the sidebar.
type ArchiveView struct {
	Label string
	URL   string
	Count int
}

// PageLink is a link to a page or post (sidebar, post navigation).
type PageLink struct {
	Title string
	URL   string
}

// PageView adapts a model.Post (type=page) for template rendering.
type PageView struct {
	ID         int64
	TheTitle   string
	TheContent template.HTML
	EditURL    string
}

// PostView adapts a model.Post for template rendering.
// Used as the data context for the "post" fragment template and
// inside {{range .Posts}} in page templates.
//
// The "The" prefix on content fields (TheTitle, TheContent, TheDate,
// TheExcerpt, TheAuthor, TheCategories) is a deliberate nod to
// WordPress's template tag heritage — the_title(), the_content(),
// the_date(), the_author(), the_category(). This is probably a bad
// idea. But WordPress 1.6 never happened, and we're making it happen,
// and sometimes you just have to make bad decisions for the joy of it.
type PostView struct {
	ID            int64
	TheTitle      string
	TheContent    template.HTML
	TheExcerpt    string
	TheDate       string
	TheTime       string
	TheAuthor     string
	TheCategories []CategoryLink
	Permalink     string
	AuthorURL     string
	CommentCount  int
	CommentsOpen  bool
	EditURL       string
}

// CommentView adapts a model.Comment for template rendering.
// Used as the data context for the "comment" fragment template and
// inside {{range .Comments}} in page templates.
type CommentView struct {
	ID        int64
	TheAuthor string
	URL       string
	TheDate   string
	TheContent template.HTML
	Type      string
	EditURL   string
}

// CommentFormData is the data context for the "comment-form" fragment.
type CommentFormData struct {
	Post         PostView
	CommentsOpen bool
	SavedAuthor  string
	SavedEmail   string
	SavedURL     string
}

// HomeData is the template data for the "home" page template.
type HomeData struct {
	SiteData
	Posts      []PostView
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
	CurrentPage int
	TotalPages  int
}

// SingleData is the template data for the "single" page template.
type SingleData struct {
	SiteData
	Post         PostView
	Comments     []CommentView
	CommentsOpen bool
	SavedAuthor  string
	SavedEmail   string
	SavedURL     string
	PrevPost     *PageLink
	NextPost     *PageLink
}

// StaticPageData is the template data for the "page" page template.
type StaticPageData struct {
	SiteData
	Page PageView
}

// ArchiveData is the template data for the "archive" page template.
type ArchiveData struct {
	SiteData
	ArchiveTitle       string
	ArchiveDescription string
	Posts              []PostView
	HasPrev            bool
	HasNext            bool
	PrevURL            string
	NextURL            string
	CurrentPage        int
	TotalPages         int
}

// SearchData is the template data for the "search" page template.
type SearchData struct {
	ArchiveData
}

// NotFoundData is the template data for the "404" page template.
type NotFoundData struct {
	SiteData
}

// renderContent converts post content to HTML. ProseMirror JSON is
// rendered through the serializer; plain strings are passed through
// as-is for backward compatibility.
//
// Stopgap: plain text content is entity-escaped before wrapping in
// template.HTML. This prevents XSS but means plain text posts cannot
// contain formatting. Once all content is ProseMirror JSON, this
// fallback branch and the template.HTML cast disappear entirely.
// The new template engine escapes at the walker level; views will
// hold plain strings with no template.HTML wrapper.
func renderContent(content string, s *prosemirror.Serializer) template.HTML {
	if len(content) > 0 && content[0] == '{' {
		html, err := s.Render(json.RawMessage(content))
		if err != nil {
			log.Printf("failed to render post content: %v", err)
			return ""
		}
		return html
	}
	return template.HTML(template.HTMLEscapeString(content))
}

// newPostView creates a PostView from a model.Post.
func newPostView(p *model.Post, authorName string, categories []model.Category, linker *permalink.Linker, s *prosemirror.Serializer) PostView {
	catLinks := make([]CategoryLink, len(categories))
	for i, c := range categories {
		catLinks[i] = CategoryLink{
			Name: c.Name,
			URL:  "/category/" + c.Slug + "/",
		}
	}

	return PostView{
		ID:            p.ID,
		TheTitle:      p.PostTitle,
		TheContent:    renderContent(p.PostContent, s),
		TheExcerpt:    p.PostExcerpt,
		Permalink:     linker.PostPath(p.ID, p.PostDate, p.PostName),
		TheDate:       p.PostDate.Format("January 2, 2006"),
		TheTime:       p.PostDate.Format("3:04 pm"),
		TheAuthor:     authorName,
		TheCategories: catLinks,
		CommentCount:  p.CommentCount,
	}
}

// newCommentView creates a CommentView from a model.Comment.
//
// Stopgap: comment content is entity-escaped before wrapping in
// template.HTML. Comments are plain text only until ProseMirror
// handles comment input with a restrictive schema. The new template
// engine will escape at the walker level; this cast disappears.
func newCommentView(c *model.Comment) CommentView {
	return CommentView{
		ID:         c.CommentID,
		TheAuthor:  c.CommentAuthor,
		URL:        c.CommentAuthorURL,
		TheDate:    c.CommentDate.Format("January 2, 2006 at 3:04 pm"),
		TheContent: template.HTML(template.HTMLEscapeString(c.CommentContent)),
		Type:       c.CommentType,
	}
}

// categoriesString joins category link names with commas.
func categoriesString(links []CategoryLink) string {
	if len(links) == 0 {
		return "Uncategorized"
	}
	names := make([]string, len(links))
	for i, l := range links {
		names[i] = l.Name
	}
	return strings.Join(names, ", ")
}

