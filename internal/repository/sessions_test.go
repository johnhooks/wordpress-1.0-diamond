package repository_test

import (
	"context"
	"testing"
	"time"

	"press/internal/db"
	"press/internal/model"
	"press/internal/repository"
)

func TestSessionsRepository_CreateAndVerify(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	session := &model.Session{
		Token:     "test-token-abc123",
		UserID:    1,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	if err := sessions.Create(ctx, session); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if session.ID == 0 {
		t.Fatal("expected session ID to be set")
	}
	if session.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be auto-set")
	}

	userID, err := sessions.Verify(ctx, "test-token-abc123")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if userID != 1 {
		t.Errorf("expected user ID 1, got %d", userID)
	}
}

func TestSessionsRepository_VerifyExpired(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	if err := sessions.Create(ctx, &model.Session{
		Token:     "expired-token",
		UserID:    1,
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	_, err := sessions.Verify(ctx, "expired-token")
	if err == nil {
		t.Error("expected error for expired session")
	}
}

func TestSessionsRepository_VerifyRevoked(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	if err := sessions.Create(ctx, &model.Session{
		Token:     "revoked-token",
		UserID:    1,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	if err := sessions.Revoke(ctx, "revoked-token"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	_, err := sessions.Verify(ctx, "revoked-token")
	if err == nil {
		t.Error("expected error for revoked session")
	}
}

func TestSessionsRepository_VerifyNotFound(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	_, err := sessions.Verify(ctx, "nonexistent-token")
	if err == nil {
		t.Error("expected error for nonexistent token")
	}
}

func TestSessionsRepository_GetByUserID(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	expires := time.Now().UTC().Add(24 * time.Hour)
	if err := sessions.Create(ctx, &model.Session{Token: "token-a", UserID: 1, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, &model.Session{Token: "token-b", UserID: 1, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, &model.Session{Token: "token-c", UserID: 2, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}

	got, err := sessions.GetByUserID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 sessions for user 1, got %d", len(got))
	}
}

func TestSessionsRepository_RevokeAllForUser(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	expires := time.Now().UTC().Add(24 * time.Hour)
	if err := sessions.Create(ctx, &model.Session{Token: "keep-this", UserID: 1, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, &model.Session{Token: "revoke-this", UserID: 1, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, &model.Session{Token: "also-revoke", UserID: 1, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}

	n, err := sessions.RevokeAllForUser(ctx, 1, "keep-this")
	if err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 revoked, got %d", n)
	}

	// The kept token should still verify
	userID, err := sessions.Verify(ctx, "keep-this")
	if err != nil {
		t.Fatalf("expected kept token to verify: %v", err)
	}
	if userID != 1 {
		t.Errorf("expected user ID 1, got %d", userID)
	}

	// The revoked tokens should not verify
	_, err = sessions.Verify(ctx, "revoke-this")
	if err == nil {
		t.Error("expected revoked token to fail verification")
	}
}

func TestSessionsRepository_TouchLastUsed(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	if err := sessions.Create(ctx, &model.Session{
		Token:     "touch-me",
		UserID:    1,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	before, _ := sessions.GetByToken(ctx, "touch-me")
	time.Sleep(10 * time.Millisecond)

	if err := sessions.TouchLastUsed(ctx, "touch-me"); err != nil {
		t.Fatal(err)
	}

	after, _ := sessions.GetByToken(ctx, "touch-me")
	if !after.LastUsed.After(before.LastUsed) {
		t.Error("expected LastUsed to be updated")
	}
}

func TestSessionsRepository_CountActive(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	expires := time.Now().UTC().Add(24 * time.Hour)
	if err := sessions.Create(ctx, &model.Session{Token: "active-1", UserID: 1, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, &model.Session{Token: "active-2", UserID: 1, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, &model.Session{Token: "expired", UserID: 1, ExpiresAt: time.Now().UTC().Add(-1 * time.Hour)}); err != nil {
		t.Fatal(err)
	}

	count, err := sessions.CountActive(ctx, 1)
	if err != nil {
		t.Fatalf("CountActive: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 active, got %d", count)
	}
}

func TestSessionsRepository_DeleteExpired(t *testing.T) {
	database := db.SetupTestDB(t)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()

	// Expired session
	if err := sessions.Create(ctx, &model.Session{Token: "old", UserID: 1, ExpiresAt: time.Now().UTC().Add(-1 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	// Active session
	if err := sessions.Create(ctx, &model.Session{Token: "fresh", UserID: 1, ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}

	n, err := sessions.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	// Fresh should still exist
	_, err = sessions.Verify(ctx, "fresh")
	if err != nil {
		t.Error("expected fresh session to survive cleanup")
	}
}
