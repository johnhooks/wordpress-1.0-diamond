package server

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"press/internal/auth"
	"press/internal/config"
	"press/internal/model"
	"press/internal/permalink"
	"press/internal/permission"
	"press/internal/prosemirror"
	"press/internal/query"
	"press/internal/repository"

	"github.com/jmoiron/sqlx"
)

// Server is the Press HTTP server.
type Server struct {
	cfg        *config.Config
	db         *sqlx.DB
	mux        *http.ServeMux
	adminTmpl  *template.Template
	theme      *Theme
	linker     *permalink.Linker
	router     *permalink.Router
	posts      *repository.PostsRepository
	comments   *repository.CommentsRepository
	terms      *repository.TermsRepository
	users      *repository.UsersRepository
	sessions   *repository.SessionsRepository
	options    *repository.OptionsRepository
	serializer *prosemirror.Serializer
	auth       *auth.Service
	perms      permission.Checker
}

// New creates a new Server with routes and templates configured.
func New(cfg *config.Config, db *sqlx.DB) (*Server, error) {
	funcMap := template.FuncMap{
		"categories": categoriesString,
		"add":        func(a, b int) int { return a + b },
		"sub":        func(a, b int) int { return a - b },
	}

	opts := repository.NewOptionsRepository(db)
	ctx := context.Background()

	siteURL, err := opts.GetValueDefault(ctx, "siteurl", cfg.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to load siteurl option: %w", err)
	}
	structure, err := opts.GetValueDefault(ctx, "permalink_structure", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load permalink_structure option: %w", err)
	}

	// Load admin templates from the built-in admin theme.
	adminTmpl, err := loadAdminTemplates(cfg.ThemeDir, funcMap)
	if err != nil {
		return nil, fmt.Errorf("failed to load admin templates: %w", err)
	}

	// Load the site theme (Press template engine).
	theme, err := LoadTheme(cfg.ThemeDir, cfg.IsDevelopment())
	if err != nil {
		return nil, fmt.Errorf("failed to load site theme: %w", err)
	}

	users := repository.NewUsersRepository(db)
	sessions := repository.NewSessionsRepository(db)
	permStore := permission.NewStore(db)
	secure := !cfg.IsDevelopment()

	s := &Server{
		cfg:       cfg,
		db:        db,
		mux:       http.NewServeMux(),
		adminTmpl: adminTmpl,
		theme:     theme,
		linker: &permalink.Linker{
			SiteURL:   siteURL,
			Structure: structure,
		},
		router:     permalink.NewRouter(structure),
		posts:      repository.NewPostsRepository(db),
		comments:   repository.NewCommentsRepository(db),
		terms:      repository.NewTermsRepository(db),
		users:      users,
		sessions:   sessions,
		options:    opts,
		serializer: prosemirror.DefaultSerializer(),
		auth:       auth.NewService(users, sessions, secure, cfg.SessionMaxAge),
		perms:      permission.NewChecker(permStore),
	}

	s.registerTagHandlers()
	s.setupRoutes()
	return s, nil
}

func (s *Server) registerTagHandlers() {
	// Molecules
	s.theme.RegisterHandler("post", s.tagPost)
	s.theme.RegisterHandler("comment", s.tagComment)
	s.theme.RegisterHandler("pagination", s.tagPagination)
	s.theme.RegisterHandler("post-navigation", s.tagPostNavigation)
	s.theme.RegisterHandler("post-list", s.tagPostList)
	s.theme.RegisterHandler("comment-form", s.tagCommentForm)
	s.theme.RegisterHandler("comment-list-empty", s.tagCommentListEmpty)
	s.theme.RegisterHandler("search-form", s.tagSearchForm)
	s.theme.RegisterHandler("category-list", s.tagCategoryList)
	s.theme.RegisterHandler("archive-list", s.tagArchiveList)
	s.theme.RegisterHandler("page-list", s.tagPageList)
	s.theme.RegisterHandler("meta-links", s.tagMetaLinks)
	s.theme.RegisterHandler("archive-header", s.tagArchiveHeader)

	// Organisms
	s.theme.RegisterHandler("sidebar", s.tagSidebar)
	s.theme.RegisterHandler("comment-list", s.tagCommentList)
}

// renderSite renders a site template with the given data. It ensures
// the visitor has a session and puts a CSRF helper on the context so
// tag handlers can generate tokens.
func (s *Server) renderSite(w http.ResponseWriter, r *http.Request, name string, data any) {
	sessionToken, err := s.auth.EnsureSession(w, r)
	if err != nil {
		log.Printf("failed to ensure session: %v", err)
	}

	ctx := auth.WithCSRF(r.Context(), auth.NewCSRFHelper(sessionToken, s.cfg.SecretKey))

	if err := s.theme.Render(w, ctx, name, data); err != nil {
		log.Printf("template render error: %v", err)
		http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
	}
}

// loadAdminTemplates loads admin templates from the theme directory.
// Admin molecules, organisms, then templates.
func loadAdminTemplates(themeDir string, funcMap template.FuncMap) (*template.Template, error) {
	tmpl := template.New("").Funcs(funcMap)

	dirs := []string{
		filepath.Join(themeDir, "admin", "molecules"),
		filepath.Join(themeDir, "admin", "organisms"),
		filepath.Join(themeDir, "admin", "templates"),
	}

	for _, dir := range dirs {
		pattern := filepath.Join(dir, "*.html")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob %s: %w", pattern, err)
		}
		if len(matches) == 0 {
			continue
		}
		tmpl, err = tmpl.ParseFiles(matches...)
		if err != nil {
			return nil, fmt.Errorf("failed to parse admin templates in %s: %w", dir, err)
		}
	}

	return tmpl, nil
}

// Handler returns the server's HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Start begins listening on the configured host and port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.AppHost, s.cfg.AppPort)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) setupRoutes() {
	// Static files served from the site's public directory
	publicFS := os.DirFS(s.cfg.PublicDir)
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(publicFS))))

	// Theme static files (style.css, images, etc.)
	themeFS := os.DirFS(s.cfg.ThemeDir)
	s.mux.Handle("GET /theme/", http.StripPrefix("/theme/", http.FileServer(http.FS(themeFS))))

	s.mux.HandleFunc("POST /comments", s.handleCommentSubmit)
	s.mux.HandleFunc("POST /comments/", s.handleCommentSubmit)

	// Admin routes — login is public, everything else requires auth
	s.mux.HandleFunc("GET /wp-admin/login", s.handleLogin)
	s.mux.HandleFunc("POST /wp-admin/login", s.handleLoginSubmit)
	s.mux.HandleFunc("GET /wp-admin/logout", s.handleLogout)

	s.mux.Handle("GET /wp-admin/post/new", s.auth.RequireAuth(http.HandlerFunc(s.handleWritePost)))
	s.mux.Handle("POST /wp-admin/post/new", s.auth.RequireAuth(http.HandlerFunc(s.handleWritePostSubmit)))
	s.mux.Handle("GET /wp-admin/post/{id}/edit", s.auth.RequireAuth(http.HandlerFunc(s.handleEditPost)))
	s.mux.Handle("POST /wp-admin/post/{id}/edit", s.auth.RequireAuth(http.HandlerFunc(s.handleEditPostSubmit)))
	s.mux.Handle("GET /wp-admin/post/{id}/delete", s.auth.RequireAuth(http.HandlerFunc(s.handleDeletePost)))
	s.mux.Handle("GET /wp-admin/posts", s.auth.RequireAuth(http.HandlerFunc(s.handleManagePosts)))
	s.mux.Handle("GET /wp-admin/", s.auth.RequireAuth(http.HandlerFunc(s.handleAdminDashboard)))

	s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// ?p=<id> — query-string post lookup
		if r.URL.Path == "/" && r.URL.Query().Get("p") != "" {
			s.handleByID(w, r)
			return
		}

		// Rewrite rules (pretty permalinks)
		if route := s.router.Resolve(r.URL.Path); route != nil {
			switch route.Type {
			case permalink.RoutePost:
				s.handlePost(w, r, route.Match)
			case permalink.RouteCategory:
				s.handleCategory(w, r, route.Match)
			case permalink.RouteDayArchive, permalink.RouteMonthArchive, permalink.RouteYearArchive:
				s.handleArchive(w, r, route)
			}
			return
		}

		// Homepage
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		s.handleHome(w, r)
	})
}

// httpError writes an error response. For HTML requests it sends a plain
// text error. The message should be safe for end users. Internal details
// are logged, not sent.
func (s *Server) httpError(w http.ResponseWriter, r *http.Request, message string, status int) {
	http.Error(w, message, status)
}

func (s *Server) blogInfo(ctx context.Context) (name, description string) {
	var err error
	name, err = s.options.GetValueDefault(ctx, "blogname", "My Weblog")
	if err != nil {
		log.Printf("failed to load blogname option: %v", err)
		name = "My Weblog"
	}
	description, err = s.options.GetValueDefault(ctx, "blogdescription", "Just another Press weblog")
	if err != nil {
		log.Printf("failed to load blogdescription option: %v", err)
		description = "Just another Press weblog"
	}
	return
}

// siteData builds the shared SiteData available on every page.
// Sidebar data (categories, archives, pages) and auth state will be
// populated here once the queries are wired up.
func (s *Server) siteData(r *http.Request) SiteData {
	name, desc := s.blogInfo(r.Context())
	data := SiteData{
		BlogName:        name,
		BlogDescription: desc,
		LoginURL:        "/wp-admin/login",
		LogoutURL:       "/wp-admin/logout",
		AdminURL:        "/wp-admin/",
		IsLoggedIn:      s.auth.Authenticate(r) != nil,
	}
	return data
}

func (s *Server) authorName(ctx context.Context, userID int64) string {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "Unknown"
	}
	return user.DisplayName
}

// postResult bundles a model.Post with its rendered PostView.
type postResult struct {
	*model.Post
	View PostView
}

// lookupPostByID fetches a published post by ID.
func (s *Server) lookupPostByID(ctx context.Context, id int64) (*postResult, error) {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post.PostStatus != "publish" || post.PostType != "post" {
		return nil, fmt.Errorf("not found")
	}
	author := s.authorName(ctx, post.PostAuthor)
	cats, err := s.terms.GetPostTerms(ctx, post.ID, "category")
	if err != nil {
		log.Printf("failed to load categories for post %d: %v", post.ID, err)
	}
	return &postResult{Post: post, View: newPostView(post, author, cats, s.linker, s.serializer)}, nil
}

// lookupPostBySlug fetches a published post by slug.
func (s *Server) lookupPostBySlug(ctx context.Context, slug string) (*postResult, error) {
	post, err := s.posts.GetBySlugAndType(ctx, slug, "post")
	if err != nil {
		return nil, err
	}
	if post.PostStatus != "publish" {
		return nil, fmt.Errorf("not found")
	}
	author := s.authorName(ctx, post.PostAuthor)
	cats, err := s.terms.GetPostTerms(ctx, post.ID, "category")
	if err != nil {
		log.Printf("failed to load categories for post %d: %v", post.ID, err)
	}
	return &postResult{Post: post, View: newPostView(post, author, cats, s.linker, s.serializer)}, nil
}

// publishedPostQuery builds a query for published posts.
func (s *Server) publishedPostQuery(page, perPage int) query.Query {
	return query.Query{
		Filters: []query.Filter{
			{Field: "status", Operator: query.Is, Value: "publish"},
			{Field: "type", Operator: query.Is, Value: "post"},
		},
		Sort:    &query.Sort{Field: "date", Direction: "desc"},
		Page:    page,
		PerPage: perPage,
	}
}

// postViews builds PostViews for a slice of posts.
func (s *Server) postViews(ctx context.Context, posts []model.Post) []PostView {
	views := make([]PostView, len(posts))
	for i := range posts {
		p := &posts[i]
		author := s.authorName(ctx, p.PostAuthor)
		cats, err := s.terms.GetPostTerms(ctx, p.ID, "category")
		if err != nil {
			log.Printf("failed to load categories for post %d: %v", p.ID, err)
		}
		views[i] = newPostView(p, author, cats, s.linker, s.serializer)
	}
	return views
}
