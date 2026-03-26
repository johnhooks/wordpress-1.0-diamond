package model

import "time"

type Session struct {
	ID        int64      `db:"id"`
	Token     string     `db:"token"`
	UserID    int64      `db:"user_id"`
	CreatedAt time.Time  `db:"created_at"`
	LastUsed  time.Time  `db:"last_used"`
	ExpiresAt time.Time  `db:"expires_at"`
	IPAddress string     `db:"ip_address"`
	UserAgent string     `db:"user_agent"`
	RevokedAt *time.Time `db:"revoked_at"`
}

func (s *Session) IsActive() bool {
	return s.RevokedAt == nil && time.Now().Before(s.ExpiresAt)
}
