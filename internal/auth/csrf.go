package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
)

// CSRFTokenLength is the number of hex characters in a CSRF token.
// 16 bytes of HMAC output = 32 hex characters.
const CSRFTokenLength = 32

// CSRFHelper generates action-scoped CSRF tokens for use in templates.
// The template calls .CSRF.For "action-name" to get a hidden input.
type CSRFHelper struct {
	sessionToken string
	secretKey    string
}

// NewCSRFHelper creates a CSRF helper bound to a session.
func NewCSRFHelper(sessionToken, secretKey string) CSRFHelper {
	return CSRFHelper{sessionToken: sessionToken, secretKey: secretKey}
}

// For returns a hidden input element with the CSRF token for the
// given action. Used in templates as {{.CSRF.For "action-name"}}.
func (c CSRFHelper) For(action string) template.HTML {
	token := generateCSRF(c.sessionToken, action, c.secretKey)
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="_csrf" value="%s">`, token))
}

// generateCSRF produces an action-scoped CSRF token. The token is
// deterministic: same session + same action = same token. No database
// storage, no expiry. The token is invalid once the session is revoked.
func generateCSRF(sessionToken, action, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(sessionToken))
	mac.Write([]byte(":"))
	mac.Write([]byte(action))
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)[:CSRFTokenLength]
}

// VerifyCSRF checks that the request contains a valid CSRF token for
// the given action. The token is read from the _csrf form field.
func VerifyCSRF(r *http.Request, action, secretKey string) bool {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	submitted := r.FormValue("_csrf")
	if submitted == "" {
		return false
	}
	expected := generateCSRF(cookie.Value, action, secretKey)
	return hmac.Equal([]byte(submitted), []byte(expected))
}
