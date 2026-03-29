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
	BlogName        string `view:"blog_name"`
	BlogDescription string `view:"blog_description"`
	PageTitle       string `view:"page_title"`

	// Sidebar
	Categories []CategoryView `view:"categories"`
	Archives   []ArchiveView  `view:"archives"`
	Pages      []PageLink     `view:"pages"`

	// Auth
	IsLoggedIn bool   `view:"is_logged_in"`
	LoginURL   string `view:"login_url"`
	LogoutURL  string `view:"logout_url"`
	AdminURL   string `view:"admin_url"`

	// Search
	SearchQuery string `view:"search_query"`

	// Sidebar context (archive/search pages)
	SidebarContext string `view:"sidebar_context"`
}

// CategoryView is a category for the sidebar.
type CategoryView struct {
	Name  string `view:"name"`
	Slug  string `view:"slug"`
	URL   string `view:"url"`
	Count int    `view:"count"`
}

// CategoryLink is a category as displayed in post metadata.
type CategoryLink struct {
	Name string `view:"name"`
	URL  string `view:"url"`
}

// ArchiveView is a monthly archive entry for the sidebar.
type ArchiveView struct {
	Label string `view:"label"`
	URL   string `view:"url"`
	Count int    `view:"count"`
}

// PageLink is a link to a page or post (sidebar, post navigation).
type PageLink struct {
	Title string `view:"title"`
	URL   string `view:"url"`
}

// PageView adapts a model.Post (type=page) for template rendering.
type PageView struct {
	ID         int64  `view:"id"`
	TheTitle   string `view:"the_title"`
	TheContent string `view:"the_content,raw"`
	EditURL    string `view:"edit_url"`
}

// PostView adapts a model.Post for template rendering.
//
// The "The" prefix on content fields (TheTitle, TheContent, TheDate,
// TheExcerpt, TheAuthor, TheCategories) is a deliberate nod to
// WordPress's template tag heritage — the_title(), the_content(),
// the_date(), the_author(), the_category(). This is probably a bad
// idea. But WordPress 1.6 never happened, and we're making it happen,
// and sometimes you just have to make bad decisions for the joy of it.
type PostView struct {
	ID            int64          `view:"id"`
	TheTitle      string         `view:"the_title"`
	TheContent    string         `view:"the_content,raw"`
	TheExcerpt    string         `view:"the_excerpt"`
	TheDate       string         `view:"the_date"`
	TheTime       string         `view:"the_time"`
	TheAuthor     string         `view:"the_author"`
	TheCategories []CategoryLink `view:"the_categories"`
	Permalink     string         `view:"permalink"`
	AuthorURL     string         `view:"author_url"`
	CommentCount  int            `view:"comment_count"`
	CommentsOpen  bool           `view:"comments_open"`
	EditURL       string         `view:"edit_url"`
}

// CommentView adapts a model.Comment for template rendering.
type CommentView struct {
	ID         int64  `view:"id"`
	TheAuthor  string `view:"the_author"`
	URL        string `view:"url"`
	TheDate    string `view:"the_date"`
	TheContent string `view:"the_content,raw"`
	Type       string `view:"type"`
	EditURL    string `view:"edit_url"`
}

// HomeData is the template data for the "home" page template.
type HomeData struct {
	SiteData
	Posts       []PostView `view:"posts"`
	HasPrev     bool       `view:"has_prev"`
	HasNext     bool       `view:"has_next"`
	PrevURL     string     `view:"prev_url"`
	NextURL     string     `view:"next_url"`
	CurrentPage int        `view:"current_page"`
	TotalPages  int        `view:"total_pages"`
}

// SingleData is the template data for the "single" page template.
type SingleData struct {
	SiteData
	Post         PostView      `view:"post"`
	PostID       int64         `view:"post_id"`
	Comments     []CommentView `view:"comments"`
	CommentsOpen bool          `view:"comments_open"`
	PrevPost     *PageLink     `view:"prev_post"`
	NextPost     *PageLink     `view:"next_post"`
}

// StaticPageData is the template data for the "page" page template.
type StaticPageData struct {
	SiteData
	Page PageView `view:"page"`
}

// ArchiveData is the template data for the "archive" page template.
type ArchiveData struct {
	SiteData
	ArchiveTitle       string     `view:"archive_title"`
	ArchiveDescription string     `view:"archive_description"`
	Posts              []PostView `view:"posts"`
	HasPrev            bool       `view:"has_prev"`
	HasNext            bool       `view:"has_next"`
	PrevURL            string     `view:"prev_url"`
	NextURL            string     `view:"next_url"`
	CurrentPage        int        `view:"current_page"`
	TotalPages         int        `view:"total_pages"`
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
// rendered through the serializer; plain strings are entity-escaped.
// Returns a plain string. Vocabulary tag handlers write this raw;
// the walker never sees it through an {expression}.
func renderContent(content string, s *prosemirror.Serializer) string {
	if len(content) > 0 && content[0] == '{' {
		html, err := s.Render(json.RawMessage(content))
		if err != nil {
			log.Printf("failed to render post content: %v", err)
			return ""
		}
		return string(html)
	}
	return template.HTMLEscapeString(content)
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
		CommentsOpen:  p.CommentStatus == "open",
	}
}

// newCommentView creates a CommentView from a model.Comment.
// Comment content is entity-escaped. The comment-list tag handler
// writes it raw, so the escaping happens here, not in the walker.
func newCommentView(c *model.Comment) CommentView {
	return CommentView{
		ID:         c.CommentID,
		TheAuthor:  c.CommentAuthor,
		URL:        c.CommentAuthorURL,
		TheDate:    c.CommentDate.Format("January 2, 2006 at 3:04 pm"),
		TheContent: template.HTMLEscapeString(c.CommentContent),
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
