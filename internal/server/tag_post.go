package server

import (
	"context"
	"fmt"
	"net/http"

	"press/internal/errors"
	"press/internal/template"
	"press/internal/template/parse"
)

// tagPost renders a <post /> vocabulary tag. It resolves the post from
// the current scope (typically bound by an {each posts as post} loop),
// pushes its view fields into a child scope, and injects the post ID
// as an engine attribute on the wrapper element.
func (s *Server) tagPost(ctx context.Context, ev *template.Evaluator, el *parse.Node) (*parse.Node, error) {
	postVal, ok := ev.Scope().Lookup("post")
	if !ok {
		return nil, errors.New(errors.ErrTemplateRender, "<post />: no post in scope", http.StatusInternalServerError)
	}
	post, ok := postVal.(PostView)
	if !ok {
		return nil, errors.New(errors.ErrTemplateRender, fmt.Sprintf("<post />: expected PostView, got %T", postVal), http.StatusInternalServerError)
	}

	engineAttrs := map[string]string{
		"id": fmt.Sprintf("post-%d", post.ID),
	}

	return s.theme.EvalTagTemplate(ctx, ev, "post", attrsFromNode(el), engineAttrs, post)
}
