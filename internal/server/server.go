package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"press/internal/config"
	"press/internal/model"
	"press/internal/permalink"
	"press/internal/prosemirror"
	"press/internal/query"
	"press/internal/repository"

	"github.com/jmoiron/sqlx"
)

//go:embed templates/*.html
var templateFS embed.FS

// Server is the Press HTTP server.
type Server struct {
	cfg      *config.Config
	db       *sqlx.DB
	mux      *http.ServeMux
	tmpl     *template.Template
	linker   *permalink.Linker
	router   *permalink.Router
	posts      *repository.PostsRepository
	comments   *repository.CommentsRepository
	terms      *repository.TermsRepository
	users      *repository.UsersRepository
	options    *repository.OptionsRepository
	serializer *prosemirror.Serializer
}

// New creates a new Server with routes and templates configured.
func New(cfg *config.Config, db *sqlx.DB) (*Server, error) {
	funcMap := template.FuncMap{
		"categories": categoriesString,
		"add":        func(a, b int) int { return a + b },
		"sub":        func(a, b int) int { return a - b },
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
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

	s := &Server{
		cfg:  cfg,
		db:   db,
		mux:  http.NewServeMux(),
		tmpl: tmpl,
		linker: &permalink.Linker{
			SiteURL:   siteURL,
			Structure: structure,
		},
		router:   permalink.NewRouter(structure),
		posts:    repository.NewPostsRepository(db),
		comments: repository.NewCommentsRepository(db),
		terms:    repository.NewTermsRepository(db),
		users:      repository.NewUsersRepository(db),
		options:    opts,
		serializer: prosemirror.DefaultSerializer(),
	}

	s.setupRoutes()
	return s, nil
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

// render executes a template and writes it to the response.
func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template error: %v", err)
	}
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
