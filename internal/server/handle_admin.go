package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"press/internal/auth"
	"press/internal/model"
	"press/internal/query"
	"press/internal/slug"
)

// ── View Data ─────────────────────────────────────────────────

// AdminShell is shared context for every authenticated admin page.
type AdminShell struct {
	SiteName    string
	SiteURL     string
	PageTitle   string
	CurrentPage string
	MenuItems   []MenuItem
	CurrentUser AdminUser
	Version     string
}

type AdminUser struct {
	ID          int64
	Login       string
	DisplayName string
	Level       int
}

type MenuItem struct {
	Label     string
	URL       string
	IsCurrent bool
}

type LoginData struct {
	SiteName    string
	SiteURL     string
	Error       string
	RedirectTo  string
	CanRegister bool
}

type CategoryOption struct {
	ID       int64
	Name     string
	Slug     string
	Selected bool
}

type WritePostData struct {
	Shell      AdminShell
	CSRF       auth.CSRFHelper
	Categories []CategoryOption
}

type EditPostData struct {
	Shell      AdminShell
	CSRF       auth.CSRFHelper
	Post       AdminPostEdit
	Categories []CategoryOption
	CanDelete  bool
}

type AdminPostEdit struct {
	ID            int64
	Title         string
	Content       string
	Excerpt       string
	Status        string
	CommentStatus string
	Slug          string
	Permalink     string
}

type AdminPostRow struct {
	ID           int64
	Title        string
	Status       string
	Date         string
	AuthorName   string
	Categories   string
	CommentCount int
	ViewURL      string
}

type ManagePostsData struct {
	Shell    AdminShell
	Posts    []AdminPostRow
	Drafts   []AdminPostRow
	HasPrev  bool
	HasNext  bool
	PrevURL  string
	NextURL  string
}

// ── Shell Builder ─────────────────────────────────────────────

func (s *Server) adminShell(r *http.Request, pageTitle, currentPage string) AdminShell {
	user := auth.UserFromContext(r.Context())
	name, _ := s.blogInfo(r.Context())

	return AdminShell{
		SiteName:    name,
		SiteURL:     s.linker.SiteURL,
		PageTitle:   pageTitle,
		CurrentPage: currentPage,
		MenuItems: []MenuItem{
			{Label: "Write", URL: "/wp-admin/post/new", IsCurrent: currentPage == "write"},
			{Label: "Manage", URL: "/wp-admin/posts", IsCurrent: currentPage == "posts"},
		},
		CurrentUser: AdminUser{
			ID:          user.ID,
			Login:       user.UserLogin,
			DisplayName: user.DisplayName,
		},
		Version: "1.6-dev",
	}
}

func (s *Server) adminCategories(r *http.Request, selectedIDs map[int64]bool) []CategoryOption {
	result, err := s.terms.List(r.Context(), query.Query{
		Filters: []query.Filter{{Field: "taxonomy", Operator: query.Is, Value: "category"}},
		PerPage: 1000,
	})
	if err != nil {
		log.Printf("failed to load categories: %v", err)
		return nil
	}

	opts := make([]CategoryOption, len(result.Items))
	for i, c := range result.Items {
		opts[i] = CategoryOption{
			ID:       c.TaxonomyID,
			Name:     c.Name,
			Slug:     c.Slug,
			Selected: selectedIDs[c.TaxonomyID],
		}
	}
	return opts
}

// ── Login ─────────────────────────────────────────────────────

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	name, _ := s.blogInfo(r.Context())
	redirectTo := auth.SafeRedirect(r.URL.Query().Get("redirect_to"), "/wp-admin/")

	s.renderAdmin(w, "admin-login", LoginData{
		SiteName:   name,
		SiteURL:    s.linker.SiteURL,
		RedirectTo: redirectTo,
	})
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.httpError(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	login := r.FormValue("log")
	password := r.FormValue("pwd")
	redirectTo := auth.SafeRedirect(r.FormValue("redirect_to"), "/wp-admin/")

	token, err := s.auth.Login(r.Context(), login, password, r.RemoteAddr, r.UserAgent())
	if err != nil {
		name, _ := s.blogInfo(r.Context())
		s.renderAdmin(w, "admin-login", LoginData{
			SiteName:   name,
			SiteURL:    s.linker.SiteURL,
			Error:      "Invalid login or password.",
			RedirectTo: redirectTo,
		})
		return
	}

	s.auth.SetCookie(w, token)
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		s.auth.Logout(r.Context(), cookie.Value)
	}
	s.auth.ClearCookie(w)
	http.Redirect(w, r, "/wp-admin/login", http.StatusFound)
}

// ── Dashboard ─────────────────────────────────────────────────

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/wp-admin/post/new", http.StatusFound)
}

// ── Write Post ────────────────────────────────────────────────

func (s *Server) handleWritePost(w http.ResponseWriter, r *http.Request) {
	s.renderAdmin(w, "admin-write", WritePostData{
		Shell:      s.adminShell(r, "Write Post", "write"),
		CSRF:       s.csrf(r),
		Categories: s.adminCategories(r, nil),
	})
}

func (s *Server) handleWritePostSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.httpError(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	if !auth.VerifyCSRF(r, "create-post", s.cfg.SecretKey) {
		s.httpError(w, r, "Invalid or expired form submission.", http.StatusForbidden)
		return
	}

	user := auth.UserFromContext(r.Context())
	ctx := r.Context()

	title := strings.TrimSpace(r.FormValue("post_title"))
	if title == "" {
		title = "Untitled"
	}

	content := r.FormValue("content")
	// Wrap plain text in ProseMirror JSON
	if content != "" && content[0] != '{' {
		content = wrapPlainText(content)
	}

	status := r.FormValue("post_status")
	if r.FormValue("draft") != "" {
		status = "draft"
	}
	if status == "" {
		status = "draft"
	}

	postSlug := slug.Generate(title)
	postSlug, _ = s.posts.EnsureUniqueSlug(ctx, postSlug, 0)

	post := &model.Post{
		PostAuthor:    user.ID,
		PostTitle:     title,
		PostContent:   content,
		PostExcerpt:   strings.TrimSpace(r.FormValue("excerpt")),
		PostStatus:    status,
		PostName:      postSlug,
		PostType:      "post",
		CommentStatus: r.FormValue("comment_status"),
	}
	if post.CommentStatus == "" {
		post.CommentStatus = "open"
	}

	if err := s.posts.Create(ctx, post); err != nil {
		log.Printf("failed to create post: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	// Assign categories
	catIDs := r.Form["post_category"]
	for _, idStr := range catIDs {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			s.terms.AddTermToPost(ctx, post.ID, id)
		}
	}

	http.Redirect(w, r, "/wp-admin/post/"+strconv.FormatInt(post.ID, 10)+"/edit", http.StatusFound)
}

// wrapPlainText wraps plain text content in ProseMirror document JSON.
func wrapPlainText(text string) string {
	// Reuse the CLI's wrapProseMirror logic concept but simplified
	// for the admin handler
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")
	parts := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// JSON-escape the text
		escaped := strings.ReplaceAll(p, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		parts = append(parts, `{"type":"paragraph","content":[{"type":"text","text":"`+escaped+`"}]}`)
	}
	return `{"type":"doc","content":[` + strings.Join(parts, ",") + `]}`
}

// ── Edit Post ─────────────────────────────────────────────────

func (s *Server) handleEditPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Get selected categories
	postCats, _ := s.terms.GetPostTerms(ctx, post.ID, "category")
	selectedIDs := make(map[int64]bool)
	for _, c := range postCats {
		selectedIDs[c.TaxonomyID] = true
	}

	permalink := s.linker.PostPath(post.ID, post.PostDate, post.PostName)

	s.renderAdmin(w, "admin-edit", EditPostData{
		Shell: s.adminShell(r, "Edit Post", "write"),
		CSRF:  s.csrf(r),
		Post: AdminPostEdit{
			ID:            post.ID,
			Title:         post.PostTitle,
			Content:       post.PostContent,
			Excerpt:       post.PostExcerpt,
			Status:        post.PostStatus,
			CommentStatus: post.CommentStatus,
			Slug:          post.PostName,
			Permalink:     permalink,
		},
		Categories: s.adminCategories(r, selectedIDs),
		CanDelete:  true, // TODO: check permissions
	})
}

func (s *Server) handleEditPostSubmit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.httpError(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	if !auth.VerifyCSRF(r, "edit-post", s.cfg.SecretKey) {
		s.httpError(w, r, "Invalid or expired form submission.", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	title := strings.TrimSpace(r.FormValue("post_title"))
	if title == "" {
		title = "Untitled"
	}

	content := r.FormValue("content")
	if content != "" && content[0] != '{' {
		content = wrapPlainText(content)
	}

	status := r.FormValue("post_status")
	if r.FormValue("publish") != "" {
		status = "publish"
	}
	if status == "" {
		status = post.PostStatus
	}

	post.PostTitle = title
	post.PostContent = content
	post.PostExcerpt = strings.TrimSpace(r.FormValue("excerpt"))
	post.PostStatus = status
	post.CommentStatus = r.FormValue("comment_status")
	if post.CommentStatus == "" {
		post.CommentStatus = "open"
	}

	if _, err := s.posts.Update(ctx, post); err != nil {
		log.Printf("failed to update post: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	// Update categories
	catIDs := r.Form["post_category"]
	var ttIDs []int64
	for _, idStr := range catIDs {
		if ttID, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			ttIDs = append(ttIDs, ttID)
		}
	}
	s.terms.SetPostTerms(ctx, post.ID, ttIDs)

	http.Redirect(w, r, "/wp-admin/post/"+strconv.FormatInt(post.ID, 10)+"/edit", http.StatusFound)
}

// ── Delete Post ───────────────────────────────────────────────

func (s *Server) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if _, err := s.posts.Delete(r.Context(), id); err != nil {
		log.Printf("failed to delete post: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/wp-admin/posts", http.StatusFound)
}

// ── Manage Posts ──────────────────────────────────────────────

func (s *Server) handleManagePosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserFromContext(ctx)

	page := 1
	if p := r.URL.Query().Get("paged"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	perPage := 20

	result, err := s.posts.List(ctx, query.Query{
		Filters: []query.Filter{
			{Field: "type", Operator: query.Is, Value: "post"},
		},
		Sort:    &query.Sort{Field: "date", Direction: "desc"},
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		log.Printf("failed to list posts: %v", err)
		s.httpError(w, r, "An internal error occurred.", http.StatusInternalServerError)
		return
	}

	posts := make([]AdminPostRow, len(result.Items))
	for i, p := range result.Items {
		authorName := s.authorName(ctx, p.PostAuthor)
		cats, _ := s.terms.GetPostTerms(ctx, p.ID, "category")
		catNames := make([]string, len(cats))
		for j, c := range cats {
			catNames[j] = c.Name
		}
		catStr := "Uncategorized"
		if len(catNames) > 0 {
			catStr = strings.Join(catNames, ", ")
		}

		posts[i] = AdminPostRow{
			ID:           p.ID,
			Title:        p.PostTitle,
			Status:       p.PostStatus,
			Date:         p.PostDate.Format("2006/01/02 15:04"),
			AuthorName:   authorName,
			Categories:   catStr,
			CommentCount: p.CommentCount,
			ViewURL:      s.linker.PostPath(p.ID, p.PostDate, p.PostName),
		}
	}

	// Load drafts for the current user
	var drafts []AdminPostRow
	draftResult, err := s.posts.List(ctx, query.Query{
		Filters: []query.Filter{
			{Field: "type", Operator: query.Is, Value: "post"},
			{Field: "status", Operator: query.Is, Value: "draft"},
			{Field: "author", Operator: query.Is, Value: user.ID},
		},
		PerPage: 10,
	})
	if err == nil {
		for _, p := range draftResult.Items {
			drafts = append(drafts, AdminPostRow{
				ID:    p.ID,
				Title: p.PostTitle,
			})
		}
	}

	totalPages := (result.Total + perPage - 1) / perPage
	data := ManagePostsData{
		Shell:  s.adminShell(r, "Manage Posts", "posts"),
		Posts:  posts,
		Drafts: drafts,
	}

	if page > 1 {
		data.HasPrev = true
		data.PrevURL = "/wp-admin/posts?paged=" + strconv.Itoa(page-1)
	}
	if page < totalPages {
		data.HasNext = true
		data.NextURL = "/wp-admin/posts?paged=" + strconv.Itoa(page+1)
	}

	s.renderAdmin(w, "admin-posts", data)
}

// ── Helpers ───────────────────────────────────────────────────

// csrf builds a CSRFHelper from the request context.
func (s *Server) csrf(r *http.Request) auth.CSRFHelper {
	token := auth.SessionTokenFromContext(r.Context())
	return auth.NewCSRFHelper(token, s.cfg.SecretKey)
}

// renderAdmin executes an admin template.
func (s *Server) renderAdmin(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.adminTmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("admin template error: %v", err)
		http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
	}
}
