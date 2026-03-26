package auth_test

import (
	"testing"

	"press/internal/auth"
)

func TestGenerateToken(t *testing.T) {
	token, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if len(token) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(token))
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	a, _ := auth.GenerateToken()
	b, _ := auth.GenerateToken()
	if a == b {
		t.Error("expected unique tokens")
	}
}
