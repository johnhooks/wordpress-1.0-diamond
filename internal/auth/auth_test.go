package auth_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"press/internal/auth"
	"press/internal/model"

	"golang.org/x/crypto/bcrypt"
)

// --- Fakes ---

type fakeUsers struct {
	users map[string]*model.User // keyed by login
	byID  map[int64]*model.User
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{
		users: make(map[string]*model.User),
		byID:  make(map[int64]*model.User),
	}
}

func (f *fakeUsers) add(u *model.User) {
	f.users[u.UserLogin] = u
	f.byID[u.ID] = u
}

func (f *fakeUsers) GetByLogin(_ context.Context, login string) (*model.User, error) {
	u, ok := f.users[login]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", login)
	}
	return u, nil
}

func (f *fakeUsers) GetByID(_ context.Context, id int64) (*model.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %d", id)
	}
	return u, nil
}

type fakeSession struct {
	token   string
	userID  int64
	revoked bool
	expires time.Time
}

type fakeSessions struct {
	sessions map[string]*fakeSession
	touched  map[string]bool
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{
		sessions: make(map[string]*fakeSession),
		touched:  make(map[string]bool),
	}
}

func (f *fakeSessions) Create(_ context.Context, s *model.Session) error {
	f.sessions[s.Token] = &fakeSession{
		token:   s.Token,
		userID:  s.UserID,
		expires: s.ExpiresAt,
	}
	return nil
}

func (f *fakeSessions) Verify(_ context.Context, token string) (int64, error) {
	s, ok := f.sessions[token]
	if !ok || s.revoked || time.Now().After(s.expires) {
		return 0, fmt.Errorf("invalid session")
	}
	return s.userID, nil
}

func (f *fakeSessions) Revoke(_ context.Context, token string) error {
	s, ok := f.sessions[token]
	if !ok {
		return fmt.Errorf("session not found")
	}
	s.revoked = true
	return nil
}

func (f *fakeSessions) TouchLastUsed(_ context.Context, token string) error {
	f.touched[token] = true
	return nil
}

// --- Helpers ---

func hashPassword(t *testing.T, plain string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(hash)
}

func testUser(t *testing.T) *model.User {
	t.Helper()
	return &model.User{
		ID:        1,
		UserLogin: "admin",
		UserPass:  hashPassword(t, "password"),
		UserEmail: "admin@example.com",
	}
}

// --- Tests ---

func TestService_Login(t *testing.T) {
	users := newFakeUsers()
	users.add(testUser(t))
	sessions := newFakeSessions()
	svc := auth.NewService(users, sessions, false, 30)

	tests := []struct {
		name     string
		login    string
		password string
		wantErr  bool
	}{
		{"valid credentials", "admin", "password", false},
		{"wrong password", "admin", "wrong", true},
		{"unknown user", "nobody", "password", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.Login(context.Background(), tt.login, tt.password, "127.0.0.1", "test")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token == "" {
				t.Error("expected non-empty token")
			}
			if len(sessions.sessions) == 0 {
				t.Error("expected session to be created")
			}
		})
	}
}

func TestService_Logout(t *testing.T) {
	users := newFakeUsers()
	users.add(testUser(t))
	sessions := newFakeSessions()
	svc := auth.NewService(users, sessions, false, 30)

	token, _ := svc.Login(context.Background(), "admin", "password", "127.0.0.1", "test")

	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Session should be revoked
	if !sessions.sessions[token].revoked {
		t.Error("expected session to be revoked")
	}
}

func TestService_Authenticate(t *testing.T) {
	users := newFakeUsers()
	users.add(testUser(t))
	sessions := newFakeSessions()
	svc := auth.NewService(users, sessions, false, 30)

	token, _ := svc.Login(context.Background(), "admin", "password", "127.0.0.1", "test")

	tests := []struct {
		name     string
		cookie   string
		wantUser bool
	}{
		{"valid session", token, true},
		{"no cookie", "", false},
		{"invalid token", "bogus-token", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/wp-admin/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tt.cookie})
			}

			user := svc.Authenticate(req)
			if tt.wantUser && user == nil {
				t.Error("expected user, got nil")
			}
			if !tt.wantUser && user != nil {
				t.Error("expected nil user")
			}
		})
	}
}

func TestService_Authenticate_RevokedSession(t *testing.T) {
	users := newFakeUsers()
	users.add(testUser(t))
	sessions := newFakeSessions()
	svc := auth.NewService(users, sessions, false, 30)

	token, _ := svc.Login(context.Background(), "admin", "password", "127.0.0.1", "test")
	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})

	if user := svc.Authenticate(req); user != nil {
		t.Error("expected nil for revoked session")
	}
}

func TestService_RequireAuth(t *testing.T) {
	users := newFakeUsers()
	users.add(testUser(t))
	sessions := newFakeSessions()
	svc := auth.NewService(users, sessions, false, 30)

	token, _ := svc.Login(context.Background(), "admin", "password", "127.0.0.1", "test")

	// A handler that records whether it was called and checks context
	var called bool
	var ctxUser *model.User
	var ctxToken string
	handler := svc.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		ctxUser = auth.UserFromContext(r.Context())
		ctxToken = auth.SessionTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		cookie     string
		wantStatus int
		wantCalled bool
	}{
		{"authenticated", token, http.StatusOK, true},
		{"no cookie", "", http.StatusFound, false},
		{"bad token", "invalid", http.StatusFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called = false
			ctxUser = nil
			ctxToken = ""

			req := httptest.NewRequest("GET", "/wp-admin/posts", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tt.cookie})
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Errorf("handler called=%v, want %v", called, tt.wantCalled)
			}
			if tt.wantCalled {
				if ctxUser == nil {
					t.Error("expected user in context")
				}
				if ctxToken == "" {
					t.Error("expected session token in context")
				}
			}
		})
	}
}

func TestSafeRedirect(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		fallback string
		want     string
	}{
		{"valid path", "/wp-admin/posts", "/wp-admin/", "/wp-admin/posts"},
		{"valid path with query", "/wp-admin/posts?page=2", "/wp-admin/", "/wp-admin/posts?page=2"},
		{"empty", "", "/wp-admin/", "/wp-admin/"},
		{"absolute URL", "https://evil.com", "/wp-admin/", "/wp-admin/"},
		{"protocol-relative", "//evil.com", "/wp-admin/", "/wp-admin/"},
		{"no leading slash", "evil.com/path", "/wp-admin/", "/wp-admin/"},
		{"javascript", "javascript:alert(1)", "/wp-admin/", "/wp-admin/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.SafeRedirect(tt.target, tt.fallback)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestService_SetCookie(t *testing.T) {
	svc := auth.NewService(newFakeUsers(), newFakeSessions(), false, 30)
	w := httptest.NewRecorder()

	svc.SetCookie(w, "my-token")

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected press_session cookie")
	}
	if found.Value != "my-token" {
		t.Errorf("got value %q, want %q", found.Value, "my-token")
	}
	if !found.HttpOnly {
		t.Error("expected HttpOnly")
	}
}

func TestService_ClearCookie(t *testing.T) {
	svc := auth.NewService(newFakeUsers(), newFakeSessions(), false, 30)
	w := httptest.NewRecorder()

	svc.ClearCookie(w)

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected press_session cookie")
	}
	if found.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", found.MaxAge)
	}
}
