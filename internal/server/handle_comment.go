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

	// BUG: The form has hx-post with hx-swap="outerHTML", so htmx
	// expects an HTML fragment back. But we redirect, which htmx
	// follows and swaps the entire page into the form element.
	// The fix: detect HX-Request header, return a fresh empty form
	// (via the comment-form tag handler) plus an OOB swap to append
	// the new comment to the comment list. Fall back to redirect
	// for non-htmx requests. Fix in the AST render pipeline refactor.

	// Redirect back to the post
	post, err := s.posts.GetByID(r.Context(), postID)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	redirect := s.linker.PostPath(post.ID, post.PostDate, post.PostName)
	http.Redirect(w, r, redirect+"#comment-"+strconv.FormatInt(comment.CommentID, 10), http.StatusSeeOther)
}
