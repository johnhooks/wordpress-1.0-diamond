package server

import (
	"context"
	"fmt"

	"press/internal/auth"
	"press/internal/template"
	"press/internal/template/parse"
)

// tagComment renders a <comment /> vocabulary tag. It resolves the
// comment from the current scope (bound by {each comments as comment})
// and pushes its view fields into a child scope.
func (s *Server) tagComment(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	val, ok := ev.Scope().Lookup("comment")
	if !ok {
		panic("<comment />: no comment in scope")
	}
	comment, ok := val.(CommentView)
	if !ok {
		panic(fmt.Sprintf("<comment />: expected CommentView, got %T", val))
	}

	engineAttrs := map[string]string{
		"id": fmt.Sprintf("comment-%d", comment.ID),
	}
	return s.theme.EvalTagTemplate(ctx, ev, "comment", attrsFromNode(el), engineAttrs, comment)
}

// tagCommentList renders a <comment-list /> vocabulary tag.
// Comments are resolved from the parent scope chain. The engine
// injects an id scoped to the post so multiple comment sections
// on the same page each have a unique swap target.
func (s *Server) tagCommentList(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	postIDVal, _ := ev.Scope().Lookup("post_id")
	postID, _ := postIDVal.(int64)

	engineAttrs := map[string]string{
		"id": fmt.Sprintf("post-%d-comments", postID),
	}
	return s.theme.EvalTagTemplate(ctx, ev, "comment-list", attrsFromNode(el), engineAttrs, nil)
}

// tagCommentListEmpty renders a <comment-list-empty /> vocabulary tag.
// It injects the data-empty engine attribute so the engine can hide the
// element when sibling comments exist.
func (s *Server) tagCommentListEmpty(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	engineAttrs := map[string]string{"data-empty": ""}
	return s.theme.EvalTagTemplate(ctx, ev, "comment-list-empty", attrsFromNode(el), engineAttrs, nil)
}

// tagCommentForm renders a <comment-form /> vocabulary tag. It resolves
// the post ID from scope and generates a CSRF token from the request
// context, pushing both into a child scope for the molecule template.
func (s *Server) tagCommentForm(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
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
	Author    string `view:"author"`
	Email     string `view:"email"`
	URL       string `view:"url"`
}
