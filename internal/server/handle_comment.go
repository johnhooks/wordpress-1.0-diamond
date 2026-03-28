package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"press/internal/auth"
	"press/internal/model"
)

func (s *Server) handleCommentSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.httpError(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("comment_post_id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil || postID == 0 {
		s.httpError(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	if !auth.VerifyCSRF(r, fmt.Sprintf("comment-%d", postID), s.cfg.SecretKey) {
		s.httpError(w, r, "Invalid or expired form. Please reload and try again.", http.StatusForbidden)
		return
	}

	author := strings.TrimSpace(r.FormValue("author"))
	email := strings.TrimSpace(r.FormValue("email"))
	commentText := strings.TrimSpace(r.FormValue("comment"))

	if author == "" || commentText == "" {
		s.httpError(w, r, "Name and comment are required", http.StatusBadRequest)
		return
	}

	comment := &model.Comment{
		CommentPostID:      postID,
		CommentAuthor:      author,
		CommentAuthorEmail: email,
		CommentAuthorURL:   strings.TrimSpace(r.FormValue("url")),
		CommentAuthorIP:    r.RemoteAddr,
		CommentContent:     commentText,
		CommentApproved:    "1",
		CommentAgent:       r.UserAgent(),
		CommentType:        "comment",
	}

	if err := s.comments.Create(r.Context(), comment); err != nil {
		log.Printf("failed to create comment: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		s.handleCommentHTMX(w, r, postID, comment)
		return
	}

	// Non-htmx: redirect back to the post.
	post, err := s.posts.GetByID(r.Context(), postID)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	redirect := s.linker.PostPath(post.ID, post.PostDate, post.PostName)
	http.Redirect(w, r, redirect+"#comment-"+strconv.FormatInt(comment.CommentID, 10), http.StatusSeeOther)
}

// handleCommentHTMX renders an htmx response: a fresh empty comment
// form (swapped in place via hx-swap="outerHTML") plus an OOB swap
// that appends the new comment to the comment list.
func (s *Server) handleCommentHTMX(w http.ResponseWriter, r *http.Request, postID int64, comment *model.Comment) {
	cookie, _ := r.Cookie(auth.CookieName)
	sessionToken := ""
	if cookie != nil {
		sessionToken = cookie.Value
	}
	ctx := auth.WithCSRF(r.Context(), auth.NewCSRFHelper(sessionToken, s.cfg.SecretKey))

	// Render a fresh comment form (replaces the submitted form).
	formData := commentFormData{
		PostID:    postID,
		CSRFToken: auth.CSRFFromContext(ctx).Token(fmt.Sprintf("comment-%d", postID)),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.theme.RenderFragment(w, ctx, "comment-form", formData)

	// OOB swap: append the new comment to the comment list.
	cv := newCommentView(comment)
	fmt.Fprintf(w, `<div hx-swap-oob="beforeend:.comment-list">`)
	s.theme.RenderFragment(w, ctx, "comment", cv)
	fmt.Fprintf(w, `</div>`)
}
