package server

import (
	"context"

	"press/internal/template"
	"press/internal/template/parse"
)

// tagCategoryList renders a <category-list /> vocabulary tag.
// Categories resolve from the parent scope chain (SiteData.Categories).
func (s *Server) tagCategoryList(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "category-list", attrsFromNode(el), nil, nil)
}

// tagArchiveList renders an <archive-list /> vocabulary tag.
// Archives resolve from the parent scope chain (SiteData.Archives).
func (s *Server) tagArchiveList(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "archive-list", attrsFromNode(el), nil, nil)
}

// tagPageList renders a <page-list /> vocabulary tag.
// Pages resolve from the parent scope chain (SiteData.Pages).
func (s *Server) tagPageList(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "page-list", attrsFromNode(el), nil, nil)
}

// tagMetaLinks renders a <meta-links /> vocabulary tag.
// Auth state (is_logged_in, login_url, etc.) resolves from the parent
// scope chain.
func (s *Server) tagMetaLinks(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "meta-links", attrsFromNode(el), nil, nil)
}

// tagSearchForm renders a <search-form /> vocabulary tag.
func (s *Server) tagSearchForm(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "search-form", attrsFromNode(el), nil, nil)
}

// tagArchiveHeader renders an <archive-header /> vocabulary tag.
func (s *Server) tagArchiveHeader(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "archive-header", attrsFromNode(el), nil, nil)
}
