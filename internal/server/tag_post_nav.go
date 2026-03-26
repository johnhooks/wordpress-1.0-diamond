package server

import (
	"press/internal/template"
)

// tagPostNavigation renders a <post-navigation /> vocabulary tag.
// Navigation data (prev_post, next_post) resolves from the parent
// scope chain.
func (s *Server) tagPostNavigation(ctx *template.RenderContext) (string, error) {
	return s.theme.RenderTagScoped(ctx.Scope, "post-navigation", ctx.Attrs, nil, nil)
}
