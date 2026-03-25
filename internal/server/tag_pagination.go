package server

import (
	"press/internal/template"
)

// tagPagination renders a <pagination /> vocabulary tag.
// Pagination data (has_prev, has_next, etc.) resolves from the
// parent scope chain.
func (s *Server) tagPagination(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "pagination", ctx.Attrs, nil, nil)
}
