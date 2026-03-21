package server

import (
	"log"
	"net/http"
	"strconv"

	"press/internal/permalink"
)

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request, m *permalink.Match) {
	ctx := r.Context()

	var post *postResult
	var err error

	switch {
	case m.PostID != "":
		id, err := strconv.ParseInt(m.PostID, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		post, err = s.lookupPostByID(ctx, id)
	case m.PostName != "":
		post, err = s.lookupPostBySlug(ctx, m.PostName)
	default:
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !m.ValidateDate(post.PostDate) {
		http.NotFound(w, r)
		return
	}

	s.renderPost(w, r, post)
}

// handleByID handles ?p=<id> requests.
// With pretty permalinks: redirects to the canonical URL.
// Without: renders the post directly (WP 1.0 default behavior).
func (s *Server) handleByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("p"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	post, err := s.lookupPostByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// With pretty permalinks, redirect to the canonical URL
	if s.linker.Structure != "" {
		canonical := s.linker.PostPath(post.ID, post.PostDate, post.PostName)
		http.Redirect(w, r, canonical, http.StatusMovedPermanently)
		return
	}

	s.renderPost(w, r, post)
}

func (s *Server) renderPost(w http.ResponseWriter, r *http.Request, post *postResult) {
	ctx := r.Context()

	comments, err := s.comments.GetByPostID(ctx, post.ID)
	if err != nil {
		log.Printf("failed to load comments for post %d: %v", post.ID, err)
	}
	var commentViews []CommentView
	for i := range comments {
		if comments[i].CommentApproved == "1" {
			commentViews = append(commentViews, newCommentView(&comments[i]))
		}
	}

	siteData := s.siteData(r)
	siteData.PageTitle = post.PostTitle

	data := SingleData{
		SiteData:     siteData,
		Post:         post.View,
		Comments:     commentViews,
		CommentsOpen: post.CommentStatus == "open",
	}

	s.render(w, r, "single.html", data)
}
