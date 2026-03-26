package server

import (
	"context"

	"press/internal/template"
	"press/internal/template/parse"
)

// tagSidebar renders a <sidebar /> vocabulary tag. Injects
// role="complementary" as an engine attribute. Sidebar molecules
// resolve their data from the parent scope chain.
func (s *Server) tagSidebar(ctx context.Context, ev *template.Evaluator, el *parse.Node) (*parse.Node, error) {
	engineAttrs := map[string]string{
		"role": "complementary",
	}
	return s.theme.EvalTagTemplate(ctx, ev, "sidebar", attrsFromNode(el), engineAttrs, nil)
}
