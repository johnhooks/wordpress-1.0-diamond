package server

import (
	"context"

	"press/internal/template"
)

// tagCategoryList renders a <category-list /> vocabulary tag.
// Categories resolve from the parent scope chain (SiteData.Categories).
func (s *Server) tagCategoryList(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "category-list", attrs, nil, nil)
}

// tagArchiveList renders an <archive-list /> vocabulary tag.
// Archives resolve from the parent scope chain (SiteData.Archives).
func (s *Server) tagArchiveList(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "archive-list", attrs, nil, nil)
}

// tagPageList renders a <page-list /> vocabulary tag.
// Pages resolve from the parent scope chain (SiteData.Pages).
func (s *Server) tagPageList(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "page-list", attrs, nil, nil)
}

// tagMetaLinks renders a <meta-links /> vocabulary tag.
// Auth state (is_logged_in, login_url, etc.) resolves from the parent
// scope chain.
func (s *Server) tagMetaLinks(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "meta-links", attrs, nil, nil)
}

// tagSearchForm renders a <search-form /> vocabulary tag.
func (s *Server) tagSearchForm(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "search-form", attrs, nil, nil)
}

// tagArchiveHeader renders an <archive-header /> vocabulary tag.
func (s *Server) tagArchiveHeader(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "archive-header", attrs, nil, nil)
}
