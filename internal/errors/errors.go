// Package errors provides WordPress-inspired error handling for Press.
//
// Inspired by WP_Error (code + message + data), but improved:
//   - Go's native error wrapping replaces WP_Error's additional_errors array,
//     giving natural ordering through the error chain.
//   - Safe/unsafe separation ensures internal details never leak to users.
//   - errors.Is/As work through the chain for matching by code.
package errors

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
)

// Error codes. Go constants use Err prefix (convention); string values are
// bare codes suitable for JSON responses and log matching.
const (
	// Not found
	ErrPostNotFound    = "post_not_found"
	ErrCommentNotFound = "comment_not_found"
	ErrUserNotFound    = "user_not_found"
	ErrTermNotFound    = "term_not_found"
	ErrLinkNotFound    = "link_not_found"
	ErrOptionNotFound  = "option_not_found"
	ErrNotFound        = "not_found"

	// Already exists
	ErrPostSlugExists  = "post_slug_exists"
	ErrUserLoginExists = "user_login_exists"
	ErrUserEmailExists = "user_email_exists"
	ErrTermSlugExists  = "term_slug_exists"

	// Validation
	ErrInvalidParam = "invalid_param"
	ErrMissingParam = "missing_param"
	ErrInvalidJSON  = "invalid_json"

	// Permission
	ErrGroupNotFound = "group_not_found"
	ErrTupleNotFound = "tuple_not_found"
	ErrTokenNotFound = "token_not_found"
	ErrTokenExpired  = "token_expired"

	// Auth
	ErrUnauthorized = "unauthorized"
	ErrForbidden    = "forbidden"

	// Internal
	ErrInternal    = "internal_error"
	ErrQueryFailed = "query_failed"
)

// Error is the core error type for Press. It carries a code, a user-safe
// message, an HTTP status, and optionally wraps an underlying error.
//
// The safe flag controls what surfaces to users:
//   - safe=true: Message is shown to users (e.g., "Post not found")
//   - safe=false: A generic message is shown; the real error stays in logs
//
// Wrapping: Use Wrap() to chain errors. Go's errors.Is/As traverse the
// chain via Unwrap(), which is strictly ordered (newest → oldest) — an
// improvement over WP_Error's flat additional_errors array.
type Error struct {
	err     error  // wrapped error (nil for leaf errors)
	code    string // machine-readable code
	message string // user-facing message
	status  int    // HTTP status code
	safe    bool   // true if message is safe for end users
}

// New creates a user-safe error with no underlying cause.
func New(code, message string, status int) *Error {
	return &Error{
		code:    code,
		message: message,
		status:  status,
		safe:    true,
	}
}

// Wrap wraps an existing error with a Press error. The original error is
// preserved in the chain (accessible via Unwrap / errors.Is / errors.As)
// but hidden from users — only message is shown externally.
func Wrap(err error, code, message string, status int) *Error {
	return &Error{
		err:     err,
		code:    code,
		message: message,
		status:  status,
		safe:    true,
	}
}

// Internal wraps an error that should never be shown to users.
// The message field is set to a generic string; the real error is only
// available through Unwrap() for logging.
func Internal(err error, code string) *Error {
	return &Error{
		err:     err,
		code:    code,
		message: "An internal error occurred.",
		status:  http.StatusInternalServerError,
		safe:    false,
	}
}

// Error implements the error interface. For safe errors it returns the
// user-facing message. For unsafe errors it returns the underlying error
// string (for logging), falling back to the message.
func (e *Error) Error() string {
	if e.safe {
		return e.message
	}
	if e.err != nil {
		return e.err.Error()
	}
	return e.message
}

// Unwrap returns the underlying error for errors.Is/As chain traversal.
func (e *Error) Unwrap() error {
	return e.err
}

// Is enables errors.Is matching by code. Two *Error values match if their
// codes are equal. This lets you write:
//
//	errors.Is(err, &errors.Error{code: errors.ErrPostNotFound})
//
// or use the helper: errors.IsCode(err, errors.ErrPostNotFound)
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return t.code != "" && t.code == e.code
	}
	return false
}

// Code returns the machine-readable error code.
func (e *Error) Code() string { return e.code }

// Message returns the user-facing message.
func (e *Error) Message() string { return e.message }

// Status returns the HTTP status code.
func (e *Error) Status() int { return e.status }

// IsSafe returns true if the message is safe for end users.
func (e *Error) IsSafe() bool { return e.safe }

// MarshalJSON produces a WordPress-style JSON error response.
// Unsafe errors get a generic message; the real error never leaks.
func (e *Error) MarshalJSON() ([]byte, error) {
	msg := e.message
	if !e.safe {
		msg = "An internal error occurred."
	}

	return json.Marshal(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status int `json:"status"`
		} `json:"data"`
	}{
		Code:    e.code,
		Message: msg,
		Data: struct {
			Status int `json:"status"`
		}{Status: e.status},
	})
}

// --- Helpers for common patterns ---

// IsCode checks whether err (or anything in its chain) has the given code.
func IsCode(err error, code string) bool {
	return stderrors.Is(err, &Error{code: code})
}

// ToError converts any error to a *Error. If it's already a *Error, returns
// it. Otherwise wraps it as an internal error.
func ToError(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if stderrors.As(err, &e) {
		return e
	}
	return Internal(err, ErrInternal)
}

// --- Domain helpers ---

func NotFound(code, message string) *Error {
	return New(code, message, http.StatusNotFound)
}

func BadRequest(code, message string) *Error {
	return New(code, message, http.StatusBadRequest)
}

func Unauthorized(message string) *Error {
	return New(ErrUnauthorized, message, http.StatusUnauthorized)
}

func Forbidden(message string) *Error {
	return New(ErrForbidden, message, http.StatusForbidden)
}
