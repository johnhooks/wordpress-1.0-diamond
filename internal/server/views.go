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

// PostView adapts a model.Post for template rendering.
type PostView struct {
	ID            int64
	Title         string
	Content       template.HTML
	Excerpt       string
	Permalink     string
	Date          string
	AuthorName    string
	CategoryNames []string
	CommentCount  int
}

// CommentView adapts a model.Comment for template rendering.
type CommentView struct {
	ID      int64
	Author  string
	URL     string
	Date    string
	Content template.HTML
}

// HomeData is the template data for the homepage.
type HomeData struct {
	BlogName        string
	BlogDescription string
	PageTitle       string
	Posts           []PostView
	Page            int
	TotalPages      int
	HasPrev         bool
	HasNext         bool
	PrevPage        int
	NextPage        int
}

// SingleData is the template data for a single post.
type SingleData struct {
	BlogName        string
	BlogDescription string
	PageTitle       string
	Post            PostView
	Comments        []CommentView
}

// renderContent converts post content to HTML. ProseMirror JSON is
// rendered through the serializer; plain strings are passed through
// as-is for backward compatibility.
func renderContent(content string, s *prosemirror.Serializer) template.HTML {
	if len(content) > 0 && content[0] == '{' {
		html, err := s.Render(json.RawMessage(content))
		if err != nil {
			log.Printf("failed to render post content: %v", err)
			return ""
		}
		return html
	}
	return template.HTML(content)
}

// newPostView creates a PostView from a model.Post.
func newPostView(p *model.Post, authorName string, categories []model.Category, linker *permalink.Linker, s *prosemirror.Serializer) PostView {
	catNames := make([]string, len(categories))
	for i, c := range categories {
		catNames[i] = c.Name
	}

	return PostView{
		ID:            p.ID,
		Title:         p.PostTitle,
		Content:       renderContent(p.PostContent, s),
		Excerpt:       p.PostExcerpt,
		Permalink:     linker.PostPath(p.ID, p.PostDate, p.PostName),
		Date:          p.PostDate.Format("January 2, 2006"),
		AuthorName:    authorName,
		CategoryNames: catNames,
		CommentCount:  p.CommentCount,
	}
}

// newCommentView creates a CommentView from a model.Comment.
func newCommentView(c *model.Comment) CommentView {
	return CommentView{
		ID:      c.CommentID,
		Author:  c.CommentAuthor,
		URL:     c.CommentAuthorURL,
		Date:    c.CommentDate.Format("January 2, 2006 at 3:04 pm"),
		Content: template.HTML(c.CommentContent),
	}
}

// categoriesString joins category names with commas.
func categoriesString(names []string) string {
	if len(names) == 0 {
		return "Uncategorized"
	}
	return strings.Join(names, ", ")
}
