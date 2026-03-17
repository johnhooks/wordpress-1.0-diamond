package server

import (
	"net/http"
	"strconv"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	blogName, blogDesc := s.blogInfo(ctx)

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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	views := s.postViews(ctx, result.Items)

	data := HomeData{
		BlogName:        blogName,
		BlogDescription: blogDesc,
		Posts:           views,
		Page:            page,
		TotalPages:      result.TotalPages,
		HasPrev:         page > 1,
		HasNext:         page < result.TotalPages,
		PrevPage:        page - 1,
		NextPage:        page + 1,
	}

	s.render(w, r, "home.html", data)
}
