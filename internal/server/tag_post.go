package server

import (
	"context"
	"fmt"
	"net/http"

	"press/internal/errors"
	"press/internal/template"
)

// tagPost renders a <post /> vocabulary tag. It resolves the post from
// the current scope (typically bound by an {each posts as post} loop),
// pushes its view fields into a child scope, and injects the post ID
// as an engine attribute on the wrapper element.
func (s *Server) tagPost(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	postVal, ok := scope.Lookup("post")
	if !ok {
		return "", errors.New(errors.ErrTemplateRender, "<post />: no post in scope", http.StatusInternalServerError)
	}
	post, ok := postVal.(PostView)
	if !ok {
		return "", errors.New(errors.ErrTemplateRender, fmt.Sprintf("<post />: expected PostView, got %T", postVal), http.StatusInternalServerError)
	}

	engineAttrs := map[string]string{
		"id": fmt.Sprintf("post-%d", post.ID),
	}

	return s.theme.RenderTagScoped(ctx, scope, "post", attrs, engineAttrs, post)
}
