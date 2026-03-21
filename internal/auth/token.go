// Package auth provides session token generation and verification.
//
// A session token is a random 32-byte value encoded as hex. The token
// is stored in the database and set as an HttpOnly cookie. There is no
// signing or encryption of the cookie value itself. The token is an
// opaque identifier that the server looks up in the sessions table.
//
// This is deliberately simpler than JWT. The server is the only consumer
// of the token. There is no need for the token to carry claims or be
// verifiable without a database lookup. Every authenticated request
// hits the sessions table anyway (to check revocation and expiry), so
// a signed token would add complexity without reducing database load.
package auth

import (
	"crypto/rand"
	"encoding/hex"

	"press/internal/errors"
)

// GenerateToken creates a cryptographically random session token.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.Internal(err, errors.ErrInternal)
	}
	return hex.EncodeToString(bytes), nil
}
