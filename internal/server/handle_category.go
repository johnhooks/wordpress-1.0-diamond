package server

import (
	"net/http"

	"press/internal/permalink"
)

func (s *Server) handleCategory(w http.ResponseWriter, r *http.Request, m *permalink.Match) {
	// TODO: category archive pages
	http.NotFound(w, r)
}
