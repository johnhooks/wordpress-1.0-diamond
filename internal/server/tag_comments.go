package server

import (
	"context"
	"fmt"
	"net/http"

	"press/internal/auth"
	"press/internal/errors"
	"press/internal/template"
)

// tagComment renders a <comment /> vocabulary tag. It resolves the
// comment from the current scope (bound by {each comments as comment}),
// pushes its view fields into a child scope, and injects the comment
// ID as an engine attribute on the wrapper element.
func (s *Server) tagComment(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	val, ok := scope.Lookup("comment")
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

	return s.theme.RenderTagScoped(ctx, scope, "comment", attrs, engineAttrs, comment)
}

// tagCommentList renders a <comment-list /> vocabulary tag.
// Comments are resolved from the parent scope chain.
func (s *Server) tagCommentList(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "comment-list", attrs, nil, nil)
}

// tagCommentForm renders a <comment-form /> vocabulary tag. It resolves
// the post ID from scope and generates a CSRF token from the request
// context, pushing both into a child scope for the molecule template.
//
// TODO: The form element, htmx attributes, and hidden fields are
// currently in the molecule template. When the render pipeline moves
// to AST → AST → render, the handler should build the <form> node
// with engine concerns and graft the themed field layout as children.
func (s *Server) tagCommentForm(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	postIDVal, _ := scope.Lookup("post_id")
	postID, _ := postIDVal.(int64)

	csrf := auth.CSRFFromContext(ctx)

	data := commentFormData{
		PostID:    postID,
		CSRFToken: csrf.Token(fmt.Sprintf("comment-%d", postID)),
	}

	return s.theme.RenderTagScoped(ctx, scope, "comment-form", attrs, nil, data)
}

// commentFormData is pushed into scope for the comment-form molecule.
type commentFormData struct {
	PostID    int64  `view:"post_id"`
	CSRFToken string `view:"csrf_token"`
}
