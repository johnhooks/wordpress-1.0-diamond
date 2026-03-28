package server

import (
	"context"
	"fmt"
	"net/http"

	"press/internal/auth"
	"press/internal/errors"
	"press/internal/template"
	"press/internal/template/parse"
)

// tagComment renders a <comment /> vocabulary tag. It resolves the
// comment from the current scope (bound by {each comments as comment})
// and pushes its view fields into a child scope.
func (s *Server) tagComment(ctx context.Context, ev *template.Evaluator, el *parse.Node) (*parse.Node, error) {
	val, ok := ev.Scope().Lookup("comment")
	if !ok {
		return nil, errors.New(errors.ErrTemplateRender, "<comment />: no comment in scope", http.StatusInternalServerError)
	}
	comment, ok := val.(CommentView)
	if !ok {
		return nil, errors.New(errors.ErrTemplateRender, fmt.Sprintf("<comment />: expected CommentView, got %T", val), http.StatusInternalServerError)
	}

	return s.theme.EvalTagTemplate(ctx, ev, "comment", attrsFromNode(el), nil, comment)
}

// tagCommentList renders a <comment-list /> vocabulary tag.
// Comments are resolved from the parent scope chain.
func (s *Server) tagCommentList(ctx context.Context, ev *template.Evaluator, el *parse.Node) (*parse.Node, error) {
	return s.theme.EvalTagTemplate(ctx, ev, "comment-list", attrsFromNode(el), nil, nil)
}

// tagCommentForm renders a <comment-form /> vocabulary tag. It resolves
// the post ID from scope and generates a CSRF token from the request
// context, pushing both into a child scope for the molecule template.
func (s *Server) tagCommentForm(ctx context.Context, ev *template.Evaluator, el *parse.Node) (*parse.Node, error) {
	postIDVal, _ := ev.Scope().Lookup("post_id")
	postID, _ := postIDVal.(int64)

	csrf := auth.CSRFFromContext(ctx)

	data := commentFormData{
		PostID:    postID,
		CSRFToken: csrf.Token(fmt.Sprintf("comment-%d", postID)),
	}

	return s.theme.EvalTagTemplate(ctx, ev, "comment-form", attrsFromNode(el), nil, data)
}

// commentFormData is pushed into scope for the comment-form molecule.
type commentFormData struct {
	PostID    int64  `view:"post_id"`
	CSRFToken string `view:"csrf_token"`
}
