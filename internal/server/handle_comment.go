package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"press/internal/auth"
	"press/internal/model"
	"press/internal/permission"
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

	// Check that the post exists, is published, and accepts comments.
	post, err := s.posts.GetByID(r.Context(), postID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if post.PostStatus != "publish" {
		http.NotFound(w, r)
		return
	}
	if post.CommentStatus != "open" {
		s.httpError(w, r, "Comments are closed.", http.StatusForbidden)
		return
	}

	// Check that the caller has permission to comment.
	var userID int64
	user := s.auth.Authenticate(r)
	if user != nil {
		userID = user.ID
	}
	obj := permission.ObjectForPostType(post.PostType, fmt.Sprintf("%d", postID))
	allowed, err := s.perms.Can(r.Context(), userID, permission.ActionComment, obj)
	if err != nil {
		log.Printf("permission check failed: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}
	if !allowed {
		s.httpError(w, r, "You do not have permission to comment.", http.StatusForbidden)
		return
	}

	author := strings.TrimSpace(r.FormValue("author"))
	email := strings.TrimSpace(r.FormValue("email"))
	url := strings.TrimSpace(r.FormValue("url"))
	commentText := strings.TrimSpace(r.FormValue("comment"))

	// Logged-in users: fill author fields from their account.
	if user != nil {
		author = user.DisplayName
		email = user.UserEmail
		url = user.UserURL
	}

	if author == "" || commentText == "" {
		s.httpError(w, r, "Name and comment are required", http.StatusBadRequest)
		return
	}

	comment := &model.Comment{
		CommentPostID:      postID,
		CommentAuthor:      author,
		CommentAuthorEmail: email,
		CommentAuthorURL:   url,
		CommentAuthorIP:    r.RemoteAddr,
		CommentContent:     commentText,
		CommentApproved:    "1",
		CommentAgent:       r.UserAgent(),
		CommentType:        "comment",
		UserID:             userID,
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
	// The tag handler owns the <form> element, hidden inputs, and htmx
	// attrs. It reads post_id from scope and CSRF from context.
	formScope := struct {
		PostID     int64 `view:"post_id"`
		IsLoggedIn bool  `view:"is_logged_in"`
	}{PostID: postID, IsLoggedIn: s.auth.Authenticate(r) != nil}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.theme.RenderTag(w, ctx, "comment-form", formScope); err != nil {
		log.Printf("render comment-form fragment: %v", err)
		return
	}

	// OOB swap: append the new comment to the comment list.
	// Uses RenderTag so the tag handler injects the engine id attr.
	// The tagComment handler looks up "comment" from scope, so wrap
	// the view in a struct with the expected binding.
	cv := newCommentView(comment)
	commentScope := struct {
		Comment CommentView `view:"comment"`
	}{Comment: cv}
	if _, err := fmt.Fprintf(w, `<div hx-swap-oob="beforeend:#post-%d-comments .comment-list">`, postID); err != nil {
		return
	}
	if err := s.theme.RenderTag(w, ctx, "comment", commentScope); err != nil {
		log.Printf("render comment fragment: %v", err)
		return
	}
	_, _ = fmt.Fprintf(w, `</div>`)
}
