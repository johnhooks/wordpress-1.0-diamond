package server

import (
	"context"
	"fmt"

	"press/internal/auth"
	"press/internal/template"
	"press/internal/template/parse"
	"press/view"
)

// tagComment renders a <comment /> vocabulary tag. It resolves the
// comment from the current scope (bound by {each comments as comment})
// and pushes its view fields into a child scope.
func (s *Server) tagComment(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	val, ok := ev.Scope().Lookup("comment")
	if !ok {
		panic("<comment />: no comment in scope")
	}
	comment, ok := val.(view.Comment)
	if !ok {
		panic(fmt.Sprintf("<comment />: expected view.Comment, got %T", val))
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

// tagCommentForm renders a <comment-form /> vocabulary tag. The engine
// owns the <form> element, its routing attributes, and the hidden
// inputs (post ID, CSRF token). The theme template provides only the
// visible field markup, which is evaluated into the form as children.
func (s *Server) tagCommentForm(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	postIDVal, _ := ev.Scope().Lookup("post_id")
	postID, _ := postIDVal.(int64)

	csrf := auth.CSRFFromContext(ctx)
	csrfToken := csrf.Token(fmt.Sprintf("comment-%d", postID))

	// Evaluate the theme template (visible fields only).
	doc := s.theme.GetTemplate("comment-form")
	body := ev.EvaluateScoped(doc, commentFormData{})

	// Build the engine-owned <form> around the evaluated fields.
	// Hidden inputs go first, then the theme content.
	fields := &parse.Node{Type: parse.DocumentNode}
	fields.AppendChild(template.HiddenInput("comment_post_id", fmt.Sprintf("%d", postID)))
	fields.AppendChild(template.HiddenInput("_csrf", csrfToken))
	for c := body.FirstChild; c != nil; {
		next := c.NextSibling
		body.RemoveChild(c)
		fields.AppendChild(c)
		c = next
	}

	return template.BuildElement("form", []parse.Attribute{
		{Key: "method", Val: "post"},
		{Key: "action", Val: "/comments"},
		{Key: "hx-post", Val: "/comments"},
		{Key: "hx-target", Val: "this"},
		{Key: "hx-swap", Val: "outerHTML"},
		{Key: "hx-disabled-elt", Val: "this, find [type='submit']"},
	}, fields)
}

// commentFormData is pushed into scope for the comment-form molecule.
type commentFormData struct {
	Author string `view:"author"`
	Email  string `view:"email"`
	URL    string `view:"url"`
}
