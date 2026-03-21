package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"press/internal/auth"
)

func TestCSRFHelper_For(t *testing.T) {
	helper := auth.NewCSRFHelper("session-token-123", "secret-key")

	field := string(helper.For("delete-post"))
	if !strings.Contains(field, `name="_csrf"`) {
		t.Error("expected hidden input with name _csrf")
	}
	if !strings.Contains(field, `value="`) {
		t.Error("expected value attribute")
	}
}

func TestCSRFHelper_Scoping(t *testing.T) {
	tests := []struct {
		name     string
		sessionA string
		actionA  string
		sessionB string
		actionB  string
		wantSame bool
	}{
		{"same inputs", "session-1", "delete", "session-1", "delete", true},
		{"different actions", "session-1", "delete", "session-1", "update", false},
		{"different sessions", "session-1", "delete", "session-2", "delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := string(auth.NewCSRFHelper(tt.sessionA, "secret").For(tt.actionA))
			b := string(auth.NewCSRFHelper(tt.sessionB, "secret").For(tt.actionB))

			if (a == b) != tt.wantSame {
				if tt.wantSame {
					t.Error("expected same tokens")
				} else {
					t.Error("expected different tokens")
				}
			}
		})
	}
}

func TestVerifyCSRF(t *testing.T) {
	const session = "test-session"
	const secret = "test-secret"

	// Generate a valid token for "save-post"
	helper := auth.NewCSRFHelper(session, secret)
	field := string(helper.For("save-post"))
	start := strings.Index(field, `value="`) + 7
	end := strings.Index(field[start:], `"`) + start
	validToken := field[start:end]

	tests := []struct {
		name      string
		cookie    string
		csrfField string
		action    string
		want      bool
	}{
		{"valid", session, validToken, "save-post", true},
		{"wrong action", session, validToken, "delete-post", false},
		{"wrong session", "other-session", validToken, "save-post", false},
		{"empty token", session, "", "save-post", false},
		{"no cookie", "", validToken, "save-post", false},
		{"garbage token", session, "not-a-real-token", "save-post", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := "_csrf=" + tt.csrfField
			req := httptest.NewRequest("POST", "/", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tt.cookie})
			}

			got := auth.VerifyCSRF(req, tt.action, secret)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
