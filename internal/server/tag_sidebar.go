package server

import (
	"context"

	"press/internal/template"
)

// tagSidebar renders a <sidebar /> vocabulary tag. Injects
// role="complementary" as an engine attribute. Sidebar molecules
// resolve their data from the parent scope chain.
func (s *Server) tagSidebar(ctx context.Context, scope *template.Scope, attrs map[string]string) (string, error) {
	engineAttrs := map[string]string{
		"role": "complementary",
	}
	return s.theme.RenderTagScoped(ctx, scope, "sidebar", attrs, engineAttrs, nil)
}
