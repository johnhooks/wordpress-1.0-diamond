package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
	if err != nil {
		perPage = 10
	}
	if perPage <= 0 {
		perPage = 10
	}

	result, err := s.posts.List(ctx, s.publishedPostQuery(page, perPage))
	if err != nil {
		log.Printf("failed to list posts: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	views := s.postViews(ctx, result.Items)

	data := HomeData{
		SiteData:    s.siteData(r),
		Posts:       views,
		HasPrev:     page > 1,
		HasNext:     page < result.TotalPages,
		PrevURL:     fmt.Sprintf("?paged=%d", page-1),
		NextURL:     fmt.Sprintf("?paged=%d", page+1),
		CurrentPage: page,
		TotalPages:  result.TotalPages,
	}

	s.renderSite(w, "home", data)
}
