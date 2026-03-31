package server

import (
	"encoding/json"
	"html/template"
	"log"
	"strings"

	"press/internal/model"
	"press/internal/permalink"
	"press/internal/prosemirror"
	"press/view"
)

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

// newPost creates a view.Post from a model.Post.
func newPost(p *model.Post, authorName string, categories []model.Category, linker *permalink.Linker, s *prosemirror.Serializer) view.Post {
	catLinks := make([]view.CategoryLink, len(categories))
	for i, c := range categories {
		catLinks[i] = view.CategoryLink{
			Name: c.Name,
			URL:  "/category/" + c.Slug + "/",
		}
	}

	return view.Post{
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

// newComment creates a view.Comment from a model.Comment.
// Comment content is entity-escaped. The comment-list tag handler
// writes it raw, so the escaping happens here, not in the walker.
func newComment(c *model.Comment) view.Comment {
	return view.Comment{
		ID:         c.CommentID,
		TheAuthor:  c.CommentAuthor,
		URL:        c.CommentAuthorURL,
		TheDate:    c.CommentDate.Format("January 2, 2006 at 3:04 pm"),
		TheContent: template.HTMLEscapeString(c.CommentContent),
		Type:       c.CommentType,
	}
}

// categoriesString joins category link names with commas.
func categoriesString(links []view.CategoryLink) string {
	if len(links) == 0 {
		return "Uncategorized"
	}
	names := make([]string, len(links))
	for i, l := range links {
		names[i] = l.Name
	}
	return strings.Join(names, ", ")
}
