package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"

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

	// TODO: htmx comment fragment rendering via template engine.
	// For now, always redirect.

	// Redirect back to the post
	post, err := s.posts.GetByID(r.Context(), postID)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	redirect := s.linker.PostPath(post.ID, post.PostDate, post.PostName)
	http.Redirect(w, r, redirect+"#comment-"+strconv.FormatInt(comment.CommentID, 10), http.StatusSeeOther)
}
