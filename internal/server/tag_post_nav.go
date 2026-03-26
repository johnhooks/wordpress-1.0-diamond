package server

import (
	"context"

	"press/internal/template"
)

func (s *Server) tagPostNavigation(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	return s.theme.RenderTagScoped(ctx, scope, "post-navigation", attrs, nil, nil)
}
