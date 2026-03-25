package server

import (
	"fmt"
	"net/http"

	"press/internal/errors"
	"press/internal/template"
)

// tagComment renders a <comment /> vocabulary tag. It resolves the
// comment from the current scope (bound by {each comments as comment}),
// pushes its view fields into a child scope, and injects the comment
// ID as an engine attribute on the wrapper element.
func (s *Server) tagComment(ctx *template.RenderContext) (string, error) {
	val, ok := ctx.Scope.Lookup("comment")
	if !ok {
		return "", errors.New(errors.ErrTemplateRender, "<comment />: no comment in scope", http.StatusInternalServerError)
	}
	comment, ok := val.(CommentView)
	if !ok {
		return "", errors.New(errors.ErrTemplateRender, fmt.Sprintf("<comment />: expected CommentView, got %T", val), http.StatusInternalServerError)
	}

	engineAttrs := map[string]string{
		"id": fmt.Sprintf("comment-%d", comment.ID),
	}

	return s.theme.RenderTagScoped(ctx.Scope, "comment", ctx.Attrs, engineAttrs, comment)
}

// tagCommentList renders a <comment-list /> vocabulary tag.
// Comments are resolved from the parent scope chain.
func (s *Server) tagCommentList(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "comment-list", ctx.Attrs, nil, nil)
}

// tagCommentForm renders a <comment-form /> vocabulary tag.
// Form data (comments_open, post_id, saved fields) resolves from
// the parent scope chain.
func (s *Server) tagCommentForm(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "comment-form", ctx.Attrs, nil, nil)
}
