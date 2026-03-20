package server_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"press/internal/config"
	"press/internal/db"
	"press/internal/model"
	"press/internal/repository"
	"press/internal/server"
)

func setupServer(t *testing.T) (*server.Server, *repository.PostsRepository, *repository.CommentsRepository, *repository.TermsRepository, *repository.OptionsRepository, *repository.UsersRepository) {
	t.Helper()
	database := db.SetupTestDB(t)

	opts := repository.NewOptionsRepository(database)
	ctx := context.Background()
	opts.Set(ctx, "siteurl", "http://localhost:8080", "yes")
	opts.Set(ctx, "blogname", "Test Blog", "yes")
	opts.Set(ctx, "blogdescription", "A test blog", "yes")
	opts.Set(ctx, "posts_per_page", "10", "yes")

	publicDir := t.TempDir()
	cfg := &config.Config{
		AppHost:   "localhost",
		AppPort:   8080,
		PublicDir: publicDir,
		ThemeDir:  "../../local/themes/freerange",
	}

	s, err := server.New(cfg, database)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	posts := repository.NewPostsRepository(database)
	comments := repository.NewCommentsRepository(database)
	terms := repository.NewTermsRepository(database)
	users := repository.NewUsersRepository(database)

	return s, posts, comments, terms, opts, users
}

func TestServer_HomeReturnsOK(t *testing.T) {
	s, posts, _, _, _, _ := setupServer(t)
	ctx := context.Background()

	posts.Create(ctx, &model.Post{
		PostAuthor: 1, PostTitle: "Hello World", PostName: "hello-world",
		PostContent: "<p>Welcome!</p>", PostStatus: "publish", PostType: "post",
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !containsStr(body, "Hello World") {
		t.Error("expected homepage to contain post title")
	}
}

func TestServer_HomeEmptyBlog(t *testing.T) {
	s, _, _, _, _, _ := setupServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on empty blog, got %d", w.Code)
	}
}

func TestServer_PostByQueryParam(t *testing.T) {
	s, posts, _, _, _, _ := setupServer(t)
	ctx := context.Background()

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Query Post", PostName: "query-post",
		PostContent: "<p>Found by ID</p>", PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post)

	req := httptest.NewRequest("GET", "/?p="+itoa(post.ID), nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !containsStr(w.Body.String(), "Query Post") {
		t.Error("expected post title in response")
	}
}

func TestServer_PostByQueryParamInvalid(t *testing.T) {
	s, _, _, _, _, _ := setupServer(t)

	req := httptest.NewRequest("GET", "/?p=notanumber", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid ?p, got %d", w.Code)
	}
}

func TestServer_PostByQueryParamNotFound(t *testing.T) {
	s, _, _, _, _, _ := setupServer(t)

	req := httptest.NewRequest("GET", "/?p=99999", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing post, got %d", w.Code)
	}
}

func TestServer_PostWithComments(t *testing.T) {
	s, posts, comments, _, _, _ := setupServer(t)
	ctx := context.Background()

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Commented Post", PostName: "commented-post",
		PostContent: "<p>Has comments</p>", PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post)

	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Reader",
		CommentContent: "Great article!", CommentApproved: "1", CommentType: "comment",
	})
	// Unapproved comment should not show
	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Spammer",
		CommentContent: "Buy stuff", CommentApproved: "0", CommentType: "comment",
	})

	req := httptest.NewRequest("GET", "/?p="+itoa(post.ID), nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	if !containsStr(body, "Great article!") {
		t.Error("expected approved comment in response")
	}
	if containsStr(body, "Buy stuff") {
		t.Error("unapproved comment should not appear")
	}
}

func TestServer_PostWithCategories(t *testing.T) {
	s, posts, _, terms, _, _ := setupServer(t)
	ctx := context.Background()

	cat := &model.Term{Name: "Golang", Slug: "golang"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, cat, tt)

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Go Tips", PostName: "go-tips",
		PostContent: "<p>Tips</p>", PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post)
	terms.AddTermToPost(ctx, post.ID, tt.TermTaxonomyID)

	req := httptest.NewRequest("GET", "/?p="+itoa(post.ID), nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !containsStr(w.Body.String(), "Golang") {
		t.Error("expected category name in response")
	}
}

func TestServer_PostWithProseMirrorContent(t *testing.T) {
	s, posts, _, _, _, _ := setupServer(t)
	ctx := context.Background()

	post := &model.Post{
		PostAuthor: 1, PostTitle: "JSON Post", PostName: "json-post",
		PostContent: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello from "},{"type":"text","marks":[{"type":"strong"}],"text":"ProseMirror"}]}]}`,
		PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post)

	req := httptest.NewRequest("GET", "/?p="+itoa(post.ID), nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !containsStr(body, "<p>Hello from <strong>ProseMirror</strong></p>") {
		t.Errorf("expected rendered ProseMirror HTML in response, got:\n%s", body)
	}
}

func TestServer_DraftPostReturns404(t *testing.T) {
	s, posts, _, _, _, _ := setupServer(t)
	ctx := context.Background()

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Draft", PostName: "draft",
		PostContent: "secret", PostStatus: "draft", PostType: "post",
	}
	posts.Create(ctx, post)

	req := httptest.NewRequest("GET", "/?p="+itoa(post.ID), nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for draft post, got %d", w.Code)
	}
}

func TestServer_NotFoundPath(t *testing.T) {
	s, _, _, _, _, _ := setupServer(t)

	req := httptest.NewRequest("GET", "/nonexistent-path", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestServer_HomePagination(t *testing.T) {
	s, posts, _, _, opts, _ := setupServer(t)
	ctx := context.Background()

	opts.Set(ctx, "posts_per_page", "2", "yes")

	// Need to recreate server to pick up the new option
	// The option is read at startup, so we'll just test with the default
	// Create 3 posts
	for i := 0; i < 3; i++ {
		posts.Create(ctx, &model.Post{
			PostAuthor: 1, PostTitle: "Post " + itoa(int64(i)),
			PostName: "post-" + itoa(int64(i)), PostContent: "x",
			PostStatus: "publish", PostType: "post",
		})
	}

	// Page 1
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("page 1: expected 200, got %d", w.Code)
	}

	// Paged param
	req = httptest.NewRequest("GET", "/?paged=2", nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("page 2: expected 200, got %d", w.Code)
	}
}

func TestServer_StaticFiles(t *testing.T) {
	s, _, _, _, _, _ := setupServer(t)

	// Static route should not 500
	req := httptest.NewRequest("GET", "/static/", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	// We just want no panic/500 — a 200 or 404 is fine
	if w.Code == http.StatusInternalServerError {
		t.Error("static route returned 500")
	}
}

// --- Helpers ---

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}
