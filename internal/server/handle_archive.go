package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"press/internal/permalink"
	"press/internal/query"
)

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request, route *permalink.Route) {
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
	if err != nil || perPage <= 0 {
		perPage = 10
	}

	q := query.Query{
		Filters: []query.Filter{
			{Field: "status", Operator: query.Is, Value: "publish"},
			{Field: "type", Operator: query.Is, Value: "post"},
		},
		Sort:    &query.Sort{Field: "date", Direction: "desc"},
		Page:    page,
		PerPage: perPage,
	}

	var title string

	switch route.Type {
	case permalink.RouteYearArchive:
		q.Filters = append(q.Filters, query.Filter{Field: "year", Operator: query.Is, Value: route.Match.Year})
		title = route.Match.Year
	case permalink.RouteMonthArchive:
		q.Filters = append(q.Filters, query.Filter{Field: "year", Operator: query.Is, Value: route.Match.Year})
		q.Filters = append(q.Filters, query.Filter{Field: "month", Operator: query.Is, Value: route.Match.Month})
		title = fmt.Sprintf("%s/%s", route.Match.Year, route.Match.Month)
	case permalink.RouteDayArchive:
		q.Filters = append(q.Filters, query.Filter{Field: "year", Operator: query.Is, Value: route.Match.Year})
		q.Filters = append(q.Filters, query.Filter{Field: "month", Operator: query.Is, Value: route.Match.Month})
		q.Filters = append(q.Filters, query.Filter{Field: "day", Operator: query.Is, Value: route.Match.Day})
		title = fmt.Sprintf("%s/%s/%s", route.Match.Year, route.Match.Month, route.Match.Day)
	}

	result, err := s.posts.List(ctx, q)
	if err != nil {
		log.Printf("failed to list archive posts: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	views := s.postViews(ctx, result.Items)
	siteData := s.siteData(r)
	siteData.PageTitle = "Archive: " + title

	data := ArchiveData{
		SiteData:     siteData,
		ArchiveTitle: "Archive: " + title,
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
