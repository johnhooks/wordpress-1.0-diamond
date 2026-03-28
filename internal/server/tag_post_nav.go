package server

import (
	"context"

	"press/internal/template"
	"press/internal/template/parse"
)

func (s *Server) tagPostNavigation(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "post-navigation", attrsFromNode(el), nil, nil)
}
