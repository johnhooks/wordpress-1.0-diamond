// Package view defines the data types passed from the server to
// templates. These are pure data structs with view tags for the
// template engine. Constructor functions that depend on server
// internals live in internal/server.
package view

// Site is shared context available on every page.
type Site struct {
	BlogName        string `view:"blog_name"`
	BlogDescription string `view:"blog_description"`
	PageTitle       string `view:"page_title"`

	// Sidebar
	Categories []Category `view:"categories"`
	Archives   []Archive  `view:"archives"`
	Pages      []PageLink `view:"pages"`

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

// Category is a category for the sidebar.
type Category struct {
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

// Archive is a monthly archive entry for the sidebar.
type Archive struct {
	Label string `view:"label"`
	URL   string `view:"url"`
	Count int    `view:"count"`
}

// PageLink is a link to a page or post (sidebar, post navigation).
type PageLink struct {
	Title string `view:"title"`
	URL   string `view:"url"`
}

// Page adapts a model.Post (type=page) for template rendering.
type Page struct {
	ID         int64  `view:"id"`
	TheTitle   string `view:"the_title"`
	TheContent string `view:"the_content,raw"`
	EditURL    string `view:"edit_url"`
}

// Post adapts a model.Post for template rendering.
//
// The "The" prefix on content fields (TheTitle, TheContent, TheDate,
// TheExcerpt, TheAuthor, TheCategories) is a deliberate nod to
// WordPress's template tag heritage — the_title(), the_content(),
// the_date(), the_author(), the_category(). This is probably a bad
// idea. But WordPress 1.6 never happened, and we're making it happen,
// and sometimes you just have to make bad decisions for the joy of it.
type Post struct {
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

// Comment adapts a model.Comment for template rendering.
type Comment struct {
	ID         int64  `view:"id"`
	TheAuthor  string `view:"the_author"`
	URL        string `view:"url"`
	TheDate    string `view:"the_date"`
	TheContent string `view:"the_content,raw"`
	Type       string `view:"type"`
	EditURL    string `view:"edit_url"`
}

// HomePage is the template data for the "home" page template.
type HomePage struct {
	Site
	Posts       []Post `view:"posts"`
	HasPrev     bool   `view:"has_prev"`
	HasNext     bool   `view:"has_next"`
	PrevURL     string `view:"prev_url"`
	NextURL     string `view:"next_url"`
	CurrentPage int    `view:"current_page"`
	TotalPages  int    `view:"total_pages"`
}

// SinglePage is the template data for the "single" page template.
type SinglePage struct {
	Site
	Post         Post      `view:"post"`
	PostID       int64     `view:"post_id"`
	Comments     []Comment `view:"comments"`
	CommentsOpen bool      `view:"comments_open"`
	CanComment   bool      `view:"can_comment"`
	PrevPost     *PageLink `view:"prev_post"`
	NextPost     *PageLink `view:"next_post"`
}

// StaticPage is the template data for the "page" page template.
type StaticPage struct {
	Site
	Page Page `view:"page"`
}

// ArchivePage is the template data for the "archive" page template.
type ArchivePage struct {
	Site
	ArchiveTitle       string `view:"archive_title"`
	ArchiveDescription string `view:"archive_description"`
	Posts              []Post `view:"posts"`
	HasPrev            bool   `view:"has_prev"`
	HasNext            bool   `view:"has_next"`
	PrevURL            string `view:"prev_url"`
	NextURL            string `view:"next_url"`
	CurrentPage        int    `view:"current_page"`
	TotalPages         int    `view:"total_pages"`
}

// NotFoundPage is the template data for the "404" page template.
type NotFoundPage struct {
	Site
}
