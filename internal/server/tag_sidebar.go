package server

import (
	"press/internal/template"
)

// tagSidebar renders a <sidebar /> vocabulary tag. Injects
// role="complementary" as an engine attribute. Sidebar molecules
// resolve their data from the parent scope chain.
func (s *Server) tagSidebar(ctx *template.RenderContext) (string, error) {
	engineAttrs := map[string]string{
		"role": "complementary",
	}
	return s.theme.RenderTagScoped(ctx.Scope, "sidebar", ctx.Attrs, engineAttrs, nil)
}
