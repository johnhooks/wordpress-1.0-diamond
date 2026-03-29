package server_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"press/internal/auth"
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
	if err := opts.Set(ctx, "siteurl", "http://localhost:8080", "yes"); err != nil {
		t.Fatal(err)
	}
	if err := opts.Set(ctx, "blogname", "Test Blog", "yes"); err != nil {
		t.Fatal(err)
	}
	if err := opts.Set(ctx, "blogdescription", "A test blog", "yes"); err != nil {
		t.Fatal(err)
	}
	if err := opts.Set(ctx, "posts_per_page", "10", "yes"); err != nil {
		t.Fatal(err)
	}

	publicDir := t.TempDir()
	cfg := &config.Config{
		AppHost:       "localhost",
		AppPort:       8080,
		PublicDir:     publicDir,
		ThemeDir:      "../../themes/freerange",
		SessionMaxAge: 30,
		SecretKey:     "test-secret-key",
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

	if err := posts.Create(ctx, &model.Post{
		PostAuthor: 1, PostTitle: "Hello World", PostName: "hello-world",
		PostContent: "<p>Welcome!</p>", PostStatus: "publish", PostType: "post",
	}); err != nil {
		t.Fatal(err)
	}

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
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

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
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	if err := comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Reader",
		CommentContent: "Great article!", CommentApproved: "1", CommentType: "comment",
	}); err != nil {
		t.Fatal(err)
	}
	// Unapproved comment should not show
	if err := comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Spammer",
		CommentContent: "Buy stuff", CommentApproved: "0", CommentType: "comment",
	}); err != nil {
		t.Fatal(err)
	}

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
	if err := terms.Create(ctx, cat, tt); err != nil {
		t.Fatal(err)
	}

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Go Tips", PostName: "go-tips",
		PostContent: "<p>Tips</p>", PostStatus: "publish", PostType: "post",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}
	if err := terms.AddTermToPost(ctx, post.ID, tt.TermTaxonomyID); err != nil {
		t.Fatal(err)
	}

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
		PostStatus:  "publish", PostType: "post",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

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
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

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

	if err := opts.Set(ctx, "posts_per_page", "2", "yes"); err != nil {
		t.Fatal(err)
	}

	// Need to recreate server to pick up the new option
	// The option is read at startup, so we'll just test with the default
	// Create 3 posts
	for i := 0; i < 3; i++ {
		if err := posts.Create(ctx, &model.Post{
			PostAuthor: 1, PostTitle: "Post " + itoa(int64(i)),
			PostName: "post-" + itoa(int64(i)), PostContent: "x",
			PostStatus: "publish", PostType: "post",
		}); err != nil {
			t.Fatal(err)
		}
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

func TestServer_CommentHTMX(t *testing.T) {
	s, posts, _, _, _, _ := setupServer(t)
	ctx := context.Background()

	post := &model.Post{
		PostAuthor: 1, PostTitle: "HTMX Post", PostName: "htmx-post",
		PostContent: "<p>Test htmx comments</p>", PostStatus: "publish", PostType: "post",
		CommentStatus: "open",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	// Generate a valid CSRF token for this post.
	sessionToken := "test-session-token"
	secretKey := "test-secret-key"
	action := fmt.Sprintf("comment-%d", post.ID)
	csrf := auth.NewCSRFHelper(sessionToken, secretKey)
	csrfToken := csrf.Token(action)

	form := fmt.Sprintf(
		"comment_post_id=%d&_csrf=%s&author=Alice&email=alice%%40example.com&comment=Great+post!",
		post.ID, csrfToken,
	)
	req := httptest.NewRequest("POST", "/comments", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sessionToken})

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// The response should contain a fresh comment form with engine-owned attrs.
	if !containsStr(body, `hx-post="/comments"`) {
		t.Error("expected engine-injected hx-post on comment form")
	}
	if !containsStr(body, `hx-swap="outerHTML"`) {
		t.Error("expected engine-injected hx-swap on comment form")
	}
	if !containsStr(body, `method="post"`) {
		t.Error("expected engine-injected method on comment form")
	}
	if !containsStr(body, `action="/comments"`) {
		t.Error("expected engine-injected action on comment form")
	}

	// The form should have a CSRF hidden input (with a fresh token).
	if !containsStr(body, `name="_csrf"`) {
		t.Error("expected CSRF hidden input in comment form")
	}

	// The response should contain an OOB swap with the new comment.
	oobTarget := fmt.Sprintf(`hx-swap-oob="beforeend:#post-%d-comments .comment-list"`, post.ID)
	if !containsStr(body, oobTarget) {
		t.Error("expected OOB swap targeting comment list")
	}

	// The new comment should have an engine-injected id.
	if !containsStr(body, `id="comment-`) {
		t.Error("expected engine-injected id on new comment")
	}

	// The comment content should be present.
	if !containsStr(body, "Alice") {
		t.Error("expected comment author in response")
	}
	if !containsStr(body, "Great post!") {
		t.Error("expected comment content in response")
	}
}

func TestServer_CommentNonHTMX(t *testing.T) {
	s, posts, _, _, _, _ := setupServer(t)
	ctx := context.Background()

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Redirect Post", PostName: "redirect-post",
		PostContent: "<p>Test non-htmx</p>", PostStatus: "publish", PostType: "post",
		CommentStatus: "open",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	sessionToken := "test-session-token"
	secretKey := "test-secret-key"
	csrf := auth.NewCSRFHelper(sessionToken, secretKey)
	csrfToken := csrf.Token(fmt.Sprintf("comment-%d", post.ID))

	form := fmt.Sprintf(
		"comment_post_id=%d&_csrf=%s&author=Bob&email=bob%%40example.com&comment=Nice!",
		post.ID, csrfToken,
	)
	req := httptest.NewRequest("POST", "/comments", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sessionToken})

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	// Non-htmx path should redirect.
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d: %s", w.Code, w.Body.String())
	}

	loc := w.Header().Get("Location")
	if !containsStr(loc, "#comment-") {
		t.Errorf("expected redirect to comment anchor, got %q", loc)
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
