-- +goose Up
CREATE TABLE wp_sessions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    token       TEXT    NOT NULL UNIQUE,
    user_id     INTEGER NOT NULL,
    created_at  DATETIME NOT NULL,
    last_used   DATETIME NOT NULL,
    expires_at  DATETIME NOT NULL,
    ip_address  TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    revoked_at  DATETIME
);

CREATE INDEX idx_sessions_token ON wp_sessions (token);
CREATE INDEX idx_sessions_user_id ON wp_sessions (user_id);

-- +goose Down
DROP TABLE wp_sessions;
