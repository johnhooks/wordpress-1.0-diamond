package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"press/internal/permalink"
	"press/internal/query"
)

func (s *Server) handleCategory(w http.ResponseWriter, r *http.Request, m *permalink.Match) {
	ctx := r.Context()

	cat, err := s.terms.GetBySlug(ctx, m.Category)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	page := 1
	if p := r.URL.Query().Get("paged"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	perPageStr, err := s.options.GetValueDefault(ctx, "posts_per_page", "10")
	if err != nil {
		perPageStr = "10"
	}
	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage <= 0 {
		perPage = 10
	}

	q := query.Query{
		Filters: []query.Filter{
			{Field: "status", Operator: query.Is, Value: "publish"},
			{Field: "type", Operator: query.Is, Value: "post"},
			{Field: "category", Operator: query.Is, Value: cat.TermID},
		},
		Sort:    &query.Sort{Field: "date", Direction: "desc"},
		Page:    page,
		PerPage: perPage,
	}

	result, err := s.posts.List(ctx, q)
	if err != nil {
		log.Printf("failed to list category posts: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	views := s.postViews(ctx, result.Items)
	siteData := s.siteData(r)
	siteData.PageTitle = "Category: " + cat.Name

	data := ArchiveData{
		SiteData:     siteData,
		ArchiveTitle: "Category: " + cat.Name,
		Posts:        views,
		HasPrev:      page > 1,
		HasNext:      page < result.TotalPages,
		PrevURL:      fmt.Sprintf("%s?paged=%d", r.URL.Path, page-1),
		NextURL:      fmt.Sprintf("%s?paged=%d", r.URL.Path, page+1),
		CurrentPage:  page,
		TotalPages:   result.TotalPages,
	}

	s.renderSite(w, "archive", data)
}
