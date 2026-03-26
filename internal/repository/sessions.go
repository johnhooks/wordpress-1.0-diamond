package repository

import (
	"context"
	"database/sql"
	"time"

	"press/internal/errors"
	"press/internal/model"

	"github.com/jmoiron/sqlx"
)

type SessionsRepository struct {
	db *sqlx.DB
}

func NewSessionsRepository(db *sqlx.DB) *SessionsRepository {
	return &SessionsRepository{db: db}
}

func (r *SessionsRepository) Create(ctx context.Context, session *model.Session) error {
	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastUsed.IsZero() {
		session.LastUsed = now
	}

	result, err := r.db.NamedExecContext(ctx, `
		INSERT INTO wp_sessions (token, user_id, created_at, last_used, expires_at, ip_address, user_agent)
		VALUES (:token, :user_id, :created_at, :last_used, :expires_at, :ip_address, :user_agent)`,
		session)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	id, _ := result.LastInsertId()
	session.ID = id
	return nil
}

func (r *SessionsRepository) GetByToken(ctx context.Context, token string) (*model.Session, error) {
	session := &model.Session{}
	err := r.db.GetContext(ctx, session, "SELECT * FROM wp_sessions WHERE token = ?", token)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrNotFound, "Session not found")
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return session, nil
}

func (r *SessionsRepository) GetByUserID(ctx context.Context, userID int64) ([]model.Session, error) {
	var sessions []model.Session
	err := r.db.SelectContext(ctx, &sessions,
		"SELECT * FROM wp_sessions WHERE user_id = ? AND revoked_at IS NULL ORDER BY last_used DESC", userID)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return sessions, nil
}

func (r *SessionsRepository) TouchLastUsed(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE wp_sessions SET last_used = ? WHERE token = ?",
		time.Now().UTC(), token)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}

func (r *SessionsRepository) Revoke(ctx context.Context, token string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE wp_sessions SET revoked_at = ? WHERE token = ? AND revoked_at IS NULL",
		time.Now().UTC(), token)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrNotFound, "Session not found or already revoked")
	}
	return nil
}

func (r *SessionsRepository) RevokeAllForUser(ctx context.Context, userID int64, exceptToken string) (int64, error) {
	var result sql.Result
	var err error
	if exceptToken != "" {
		result, err = r.db.ExecContext(ctx,
			"UPDATE wp_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL AND token != ?",
			time.Now().UTC(), userID, exceptToken)
	} else {
		result, err = r.db.ExecContext(ctx,
			"UPDATE wp_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL",
			time.Now().UTC(), userID)
	}
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (r *SessionsRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM wp_sessions WHERE expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)",
		time.Now().UTC(), time.Now().UTC().Add(-7*24*time.Hour))
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (r *SessionsRepository) CountActive(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM wp_sessions WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?",
		userID, time.Now().UTC())
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	return count, nil
}

// Verify checks if a token maps to an active session and returns the
// user ID. This is the hot path for every authenticated request.
func (r *SessionsRepository) Verify(ctx context.Context, token string) (int64, error) {
	var userID int64
	err := r.db.GetContext(ctx, &userID,
		"SELECT user_id FROM wp_sessions WHERE token = ? AND revoked_at IS NULL AND expires_at > ?",
		token, time.Now().UTC())
	if err == sql.ErrNoRows {
		return 0, errors.New(errors.ErrUnauthorized, "Invalid session", 401)
	}
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	return userID, nil
}
