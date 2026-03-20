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
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("comment_post_ID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil || postID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	author := strings.TrimSpace(r.FormValue("author"))
	email := strings.TrimSpace(r.FormValue("email"))
	commentText := strings.TrimSpace(r.FormValue("comment"))

	if author == "" || commentText == "" {
		http.Error(w, "Name and comment are required", http.StatusBadRequest)
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
		http.Error(w, "Failed to save comment", http.StatusInternalServerError)
		return
	}

	// htmx: return the comment fragment + OOB comment count update.
	// This hand-wired OOB pattern is scaffolding to prove the flow
	// between primary swap and secondary updates. It informs what
	// the theme compiler needs to automate.
	if r.Header.Get("HX-Request") != "" {
		s.render(w, r, "comment", newCommentView(comment))

		post, err := s.posts.GetByID(r.Context(), postID)
		if err != nil {
			log.Printf("failed to load post for comment count: %v", err)
		} else {
			s.renderOOB(w, "comment-count", post.CommentCount)
		}
		return
	}

	// No-JS fallback: redirect back to the post
	post, err := s.posts.GetByID(r.Context(), postID)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	redirect := s.linker.PostPath(post.ID, post.PostDate, post.PostName)
	http.Redirect(w, r, redirect+"#comment-"+strconv.FormatInt(comment.CommentID, 10), http.StatusSeeOther)
}
