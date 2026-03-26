package server

import (
	"context"

	"press/internal/template"
)

func (s *Server) tagPagination(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "pagination", attrs, nil, nil)
}
