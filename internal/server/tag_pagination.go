package server

import (
	"context"

	"press/internal/template"
	"press/internal/template/parse"
)

func (s *Server) tagPagination(ctx context.Context, ev *template.Evaluator, el *parse.Node) *parse.Node {
	return s.theme.EvalTagTemplate(ctx, ev, "pagination", attrsFromNode(el), nil, nil)
}
