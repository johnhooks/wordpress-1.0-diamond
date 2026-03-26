package server

import (
	"press/internal/template"
)

// tagCategoryList renders a <category-list /> vocabulary tag.
// Categories resolve from the parent scope chain (SiteData.Categories).
func (s *Server) tagCategoryList(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "category-list", ctx.Attrs, nil, nil)
}

// tagArchiveList renders an <archive-list /> vocabulary tag.
// Archives resolve from the parent scope chain (SiteData.Archives).
func (s *Server) tagArchiveList(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "archive-list", ctx.Attrs, nil, nil)
}

// tagPageList renders a <page-list /> vocabulary tag.
// Pages resolve from the parent scope chain (SiteData.Pages).
func (s *Server) tagPageList(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "page-list", ctx.Attrs, nil, nil)
}

// tagMetaLinks renders a <meta-links /> vocabulary tag.
// Auth state (is_logged_in, login_url, etc.) resolves from the parent
// scope chain.
func (s *Server) tagMetaLinks(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "meta-links", ctx.Attrs, nil, nil)
}

// tagSearchForm renders a <search-form /> vocabulary tag.
func (s *Server) tagSearchForm(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "search-form", ctx.Attrs, nil, nil)
}

// tagArchiveHeader renders an <archive-header /> vocabulary tag.
func (s *Server) tagArchiveHeader(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "archive-header", ctx.Attrs, nil, nil)
}
