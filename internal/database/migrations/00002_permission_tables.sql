-- +goose Up

CREATE TABLE wp_groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    slug        TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    is_default  INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE wp_tuples (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    subject_type TEXT NOT NULL,
    subject_id   TEXT NOT NULL,
    relation     TEXT NOT NULL,
    object_type  TEXT NOT NULL,
    object_id    TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by   INTEGER NOT NULL DEFAULT 0,
    UNIQUE(subject_type, subject_id, relation, object_type, object_id)
);

CREATE INDEX idx_tuples_subject ON wp_tuples (subject_type, subject_id, object_type, object_id);
CREATE INDEX idx_tuples_object ON wp_tuples (object_type, object_id, subject_type, subject_id);

CREATE TABLE wp_share_tokens (
    token      TEXT PRIMARY KEY,
    created_by INTEGER  NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME
);

-- +goose Down

DROP TABLE IF EXISTS wp_share_tokens;
DROP TABLE IF EXISTS wp_tuples;
DROP TABLE IF EXISTS wp_groups;
