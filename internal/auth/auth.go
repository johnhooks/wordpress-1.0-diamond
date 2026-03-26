package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"press/internal/model"

	"golang.org/x/crypto/bcrypt"
)

// dummyHash is a bcrypt hash used to burn time when a login attempt
// targets a nonexistent user. This prevents timing attacks that could
// distinguish "user not found" from "wrong password."
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)

const CookieName = "press_session"

type contextKey string

const userContextKey contextKey = "user"
const sessionTokenKey contextKey = "session_token"

// UserLookup is the interface the auth service needs from the user store.
type UserLookup interface {
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

// SessionStore is the interface the auth service needs from the session store.
type SessionStore interface {
	Create(ctx context.Context, session *model.Session) error
	Verify(ctx context.Context, token string) (int64, error)
	Revoke(ctx context.Context, token string) error
	TouchLastUsed(ctx context.Context, token string) error
}

// Service handles authentication: login, logout, and request verification.
type Service struct {
	users    UserLookup
	sessions SessionStore
	secure   bool          // true if cookies should be Secure (HTTPS)
	maxAge   time.Duration // session lifetime
}

// NewService creates a new auth service. maxAgeDays is the session
// lifetime in days.
func NewService(users UserLookup, sessions SessionStore, secure bool, maxAgeDays int) *Service {
	return &Service{
		users:    users,
		sessions: sessions,
		secure:   secure,
		maxAge:   time.Duration(maxAgeDays) * 24 * time.Hour,
	}
}

// Login validates credentials and creates a session. Returns the session
// token on success.
func (s *Service) Login(ctx context.Context, login, password, ip, ua string) (string, error) {
	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		// Burn time to prevent timing attacks that reveal whether
		// a username exists.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.UserPass), []byte(password)); err != nil {
		return "", err
	}

	token, err := GenerateToken()
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	session := &model.Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: now,
		LastUsed:  now,
		ExpiresAt: now.Add(s.maxAge),
		IPAddress: ip,
		UserAgent: ua,
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return "", err
	}

	return token, nil
}

// Logout revokes a session by token.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Revoke(ctx, token)
}

// SetCookie writes the session cookie to the response.
func (s *Service) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.maxAge.Seconds()),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie removes the session cookie.
func (s *Service) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Authenticate checks the request for a valid session cookie and returns
// the user. Returns nil if not authenticated (no error — absence of auth
// is not an error, it is a state).
func (s *Service) Authenticate(r *http.Request) *model.User {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	ctx := r.Context()
	userID, err := s.sessions.Verify(ctx, cookie.Value)
	if err != nil {
		return nil
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil
	}

	// Touch last_used in the background. Do not block the request.
	go func() { _ = s.sessions.TouchLastUsed(context.Background(), cookie.Value) }()

	return user
}

// RequireAuth is middleware that redirects unauthenticated requests to
// the login page.
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := s.Authenticate(r)
		if user == nil {
			http.Redirect(w, r, "/wp-admin/login?redirect_to="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
			return
		}
		cookie, _ := r.Cookie(CookieName)
		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, sessionTokenKey, cookie.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SafeRedirect validates that a redirect target is a local path.
// Returns the path if safe, or the fallback if not.
func SafeRedirect(target, fallback string) string {
	if target == "" {
		return fallback
	}
	// Must start with / and not start with // (protocol-relative URL)
	if !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return fallback
	}
	// Parse to reject paths with a host component
	u, err := url.Parse(target)
	if err != nil || u.Host != "" || u.Scheme != "" {
		return fallback
	}
	return target
}

// UserFromContext returns the authenticated user from the request context,
// or nil if not authenticated.
func UserFromContext(ctx context.Context) *model.User {
	user, _ := ctx.Value(userContextKey).(*model.User)
	return user
}

// SessionTokenFromContext returns the session token from the request context.
func SessionTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(sessionTokenKey).(string)
	return token
}
