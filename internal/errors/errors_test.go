package errors_test

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	"press/internal/errors"
)

func TestNew(t *testing.T) {
	err := errors.New(errors.ErrPostNotFound, "Post not found", http.StatusNotFound)

	if err.Code() != errors.ErrPostNotFound {
		t.Errorf("expected code %q, got %q", errors.ErrPostNotFound, err.Code())
	}
	if err.Message() != "Post not found" {
		t.Errorf("expected message 'Post not found', got %q", err.Message())
	}
	if err.Status() != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", err.Status())
	}
	if !err.IsSafe() {
		t.Error("expected safe=true for New()")
	}
	if err.Error() != "Post not found" {
		t.Errorf("Error() should return message for safe errors, got %q", err.Error())
	}
}

func TestWrap(t *testing.T) {
	dbErr := fmt.Errorf("sql: no rows in result set")
	err := errors.Wrap(dbErr, errors.ErrPostNotFound, "Post not found", http.StatusNotFound)

	// Error() returns user-safe message
	if err.Error() != "Post not found" {
		t.Errorf("expected 'Post not found', got %q", err.Error())
	}

	// Unwrap reveals the underlying error
	if err.Unwrap() != dbErr {
		t.Error("Unwrap should return the underlying error")
	}

	// errors.Is works through the chain
	if !stderrors.Is(err, dbErr) {
		t.Error("errors.Is should find the wrapped error")
	}
}

func TestInternal(t *testing.T) {
	dbErr := fmt.Errorf("UNIQUE constraint failed: wp_users.user_login")
	err := errors.Internal(dbErr, errors.ErrQueryFailed)

	// Error() returns the internal error for logging, NOT the safe message
	if err.Error() != dbErr.Error() {
		t.Errorf("Error() on unsafe should return underlying error for logging, got %q", err.Error())
	}
	if err.IsSafe() {
		t.Error("expected safe=false for Internal()")
	}
	if err.Message() != "An internal error occurred." {
		t.Errorf("Message() should return generic text, got %q", err.Message())
	}
}

func TestIs_ByCode(t *testing.T) {
	err := errors.New(errors.ErrPostNotFound, "Post not found", http.StatusNotFound)

	if !errors.IsCode(err, errors.ErrPostNotFound) {
		t.Error("IsCode should match CodePostNotFound")
	}
	if errors.IsCode(err, errors.ErrUserNotFound) {
		t.Error("IsCode should not match CodeUserNotFound")
	}
}

func TestIs_WrappedChain(t *testing.T) {
	inner := errors.New(errors.ErrPostNotFound, "Post not found", http.StatusNotFound)
	outer := errors.Wrap(inner, errors.ErrInternal, "Something went wrong", http.StatusInternalServerError)

	// Should find inner error by code through the chain
	if !errors.IsCode(outer, errors.ErrPostNotFound) {
		t.Error("IsCode should find CodePostNotFound through wrapping chain")
	}
	if !errors.IsCode(outer, errors.ErrInternal) {
		t.Error("IsCode should match outer code too")
	}
}

func TestIs_StandardErrorInChain(t *testing.T) {
	sqlErr := fmt.Errorf("connection refused")
	pressErr := errors.Wrap(sqlErr, errors.ErrQueryFailed, "Database error", http.StatusInternalServerError)

	if !stderrors.Is(pressErr, sqlErr) {
		t.Error("errors.Is should find standard error through chain")
	}
}

func TestToError(t *testing.T) {
	// Already a *Error
	original := errors.New(errors.ErrPostNotFound, "Post not found", http.StatusNotFound)
	converted := errors.ToError(original)
	if converted.Code() != errors.ErrPostNotFound {
		t.Error("ToError should return existing *Error unchanged")
	}

	// Standard error gets wrapped as internal
	stdErr := fmt.Errorf("something broke")
	converted = errors.ToError(stdErr)
	if converted.Code() != errors.ErrInternal {
		t.Errorf("expected CodeInternal, got %q", converted.Code())
	}
	if converted.IsSafe() {
		t.Error("standard error wrapped as internal should not be safe")
	}

	// nil returns nil
	if errors.ToError(nil) != nil {
		t.Error("ToError(nil) should return nil")
	}
}

func TestMarshalJSON_Safe(t *testing.T) {
	err := errors.New(errors.ErrPostNotFound, "Post not found", http.StatusNotFound)

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("MarshalJSON: %v", jsonErr)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["code"] != errors.ErrPostNotFound {
		t.Errorf("expected code %q, got %v", errors.ErrPostNotFound, result["code"])
	}
	if result["message"] != "Post not found" {
		t.Errorf("expected message 'Post not found', got %v", result["message"])
	}
}

func TestMarshalJSON_Unsafe(t *testing.T) {
	dbErr := fmt.Errorf("UNIQUE constraint failed: wp_users.user_login")
	err := errors.Internal(dbErr, errors.ErrQueryFailed)

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("MarshalJSON: %v", jsonErr)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	// The internal error message must NOT appear in JSON
	if result["message"] == dbErr.Error() {
		t.Error("internal error details should not leak in JSON output")
	}
	if result["message"] != "An internal error occurred." {
		t.Errorf("expected generic message, got %v", result["message"])
	}
}

func TestHelpers(t *testing.T) {
	tests := []struct {
		name   string
		err    *errors.Error
		code   string
		status int
	}{
		{"NotFound", errors.NotFound(errors.ErrPostNotFound, "nope"), errors.ErrPostNotFound, 404},
		{"BadRequest", errors.BadRequest(errors.ErrInvalidParam, "bad"), errors.ErrInvalidParam, 400},
		{"Unauthorized", errors.Unauthorized("login required"), errors.ErrUnauthorized, 401},
		{"Forbidden", errors.Forbidden("nope"), errors.ErrForbidden, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code() != tt.code {
				t.Errorf("expected code %q, got %q", tt.code, tt.err.Code())
			}
			if tt.err.Status() != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, tt.err.Status())
			}
			if !tt.err.IsSafe() {
				t.Error("helper errors should be safe")
			}
		})
	}
}
