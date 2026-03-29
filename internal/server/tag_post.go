package server

import (
	"context"
	"fmt"

	"press/internal/template"
	"press/internal/template/parse"
)

// tagPost renders a <post /> vocabulary tag. It resolves the post from
// the current scope (typically bound by an {each posts as post} loop),
// pushes its view fields into a child scope, and injects the post ID
// as an engine attribute on the wrapper element.
func (s *Server) tagPost(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	postVal, ok := ev.Scope().Lookup("post")
	if !ok {
		panic("<post />: no post in scope")
	}
	post, ok := postVal.(PostView)
	if !ok {
		panic(fmt.Sprintf("<post />: expected PostView, got %T", postVal))
	}

	engineAttrs := map[string]string{
		"id": fmt.Sprintf("post-%d", post.ID),
	}

	return s.theme.EvalTagTemplate(ctx, ev, "post", attrsFromNode(el), engineAttrs, post)
}

// tagPostList renders a <post-list /> vocabulary tag.
// Posts are resolved from the parent scope chain.
func (s *Server) tagPostList(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "post-list", attrsFromNode(el), nil, nil)
}
