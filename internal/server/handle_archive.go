package server

import (
	"net/http"

	"press/internal/permalink"
)

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request, route *permalink.Route) {
	// TODO: date archive pages (year, month, day)
	http.NotFound(w, r)
}
