-- +goose Up

CREATE TABLE wp_posts (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    post_author           INTEGER NOT NULL DEFAULT 0,
    post_date             DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    post_date_gmt         DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    post_content          TEXT    NOT NULL DEFAULT '',
    post_title            TEXT    NOT NULL DEFAULT '',
    post_excerpt          TEXT    NOT NULL DEFAULT '',
    post_status           TEXT    NOT NULL DEFAULT 'publish',
    comment_status        TEXT    NOT NULL DEFAULT 'open',
    ping_status           TEXT    NOT NULL DEFAULT 'open',
    post_password         TEXT    NOT NULL DEFAULT '',
    post_name             TEXT    NOT NULL DEFAULT '',
    to_ping               TEXT    NOT NULL DEFAULT '',
    pinged                TEXT    NOT NULL DEFAULT '',
    post_modified         DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    post_modified_gmt     DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    post_content_filtered TEXT    NOT NULL DEFAULT '',
    post_parent           INTEGER NOT NULL DEFAULT 0,
    guid                  TEXT    NOT NULL DEFAULT '',
    menu_order            INTEGER NOT NULL DEFAULT 0,
    post_type             TEXT    NOT NULL DEFAULT 'post',
    post_mime_type        TEXT    NOT NULL DEFAULT '',
    comment_count         INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_posts_type_status_date ON wp_posts (post_type, post_status, post_date, id);
CREATE INDEX idx_posts_parent ON wp_posts (post_parent);
CREATE INDEX idx_posts_author ON wp_posts (post_author);
CREATE INDEX idx_posts_name ON wp_posts (post_name);

CREATE TABLE wp_postmeta (
    meta_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL DEFAULT 0,
    meta_key   TEXT    DEFAULT NULL,
    meta_value TEXT    DEFAULT NULL
);

CREATE INDEX idx_postmeta_post_id ON wp_postmeta (post_id);
CREATE INDEX idx_postmeta_meta_key ON wp_postmeta (meta_key);

CREATE TABLE wp_comments (
    comment_id           INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_post_id      INTEGER NOT NULL DEFAULT 0,
    comment_author       TEXT    NOT NULL DEFAULT '',
    comment_author_email TEXT    NOT NULL DEFAULT '',
    comment_author_url   TEXT    NOT NULL DEFAULT '',
    comment_author_ip    TEXT    NOT NULL DEFAULT '',
    comment_date         DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    comment_date_gmt     DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    comment_content      TEXT    NOT NULL DEFAULT '',
    comment_karma        INTEGER NOT NULL DEFAULT 0,
    comment_approved     TEXT    NOT NULL DEFAULT '1',
    comment_agent        TEXT    NOT NULL DEFAULT '',
    comment_type         TEXT    NOT NULL DEFAULT 'comment',
    comment_parent       INTEGER NOT NULL DEFAULT 0,
    user_id              INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_comments_post_id ON wp_comments (comment_post_id);
CREATE INDEX idx_comments_approved_date ON wp_comments (comment_approved, comment_date_gmt);
CREATE INDEX idx_comments_parent ON wp_comments (comment_parent);

CREATE TABLE wp_commentmeta (
    meta_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id   INTEGER NOT NULL DEFAULT 0,
    meta_key     TEXT    DEFAULT NULL,
    meta_value   TEXT    DEFAULT NULL
);

CREATE INDEX idx_commentmeta_comment_id ON wp_commentmeta (comment_id);
CREATE INDEX idx_commentmeta_meta_key ON wp_commentmeta (meta_key);

CREATE TABLE wp_users (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    user_login          TEXT    NOT NULL DEFAULT '',
    user_pass           TEXT    NOT NULL DEFAULT '',
    user_nicename       TEXT    NOT NULL DEFAULT '',
    user_email          TEXT    NOT NULL DEFAULT '',
    user_url            TEXT    NOT NULL DEFAULT '',
    user_registered     DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    user_activation_key TEXT    NOT NULL DEFAULT '',
    user_status         INTEGER NOT NULL DEFAULT 0,
    display_name        TEXT    NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_users_login ON wp_users (user_login);
CREATE INDEX idx_users_nicename ON wp_users (user_nicename);
CREATE INDEX idx_users_email ON wp_users (user_email);

CREATE TABLE wp_usermeta (
    meta_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 0,
    meta_key   TEXT    DEFAULT NULL,
    meta_value TEXT    DEFAULT NULL
);

CREATE INDEX idx_usermeta_user_id ON wp_usermeta (user_id);
CREATE INDEX idx_usermeta_meta_key ON wp_usermeta (meta_key);

CREATE TABLE wp_terms (
    term_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL DEFAULT '',
    slug       TEXT    NOT NULL DEFAULT '',
    term_group INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_terms_slug ON wp_terms (slug);
CREATE INDEX idx_terms_name ON wp_terms (name);

CREATE TABLE wp_term_taxonomy (
    term_taxonomy_id INTEGER PRIMARY KEY AUTOINCREMENT,
    term_id          INTEGER NOT NULL DEFAULT 0,
    taxonomy         TEXT    NOT NULL DEFAULT '',
    description      TEXT    NOT NULL DEFAULT '',
    parent           INTEGER NOT NULL DEFAULT 0,
    count            INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_term_taxonomy ON wp_term_taxonomy (term_id, taxonomy);
CREATE INDEX idx_term_taxonomy_taxonomy ON wp_term_taxonomy (taxonomy);

CREATE TABLE wp_term_relationships (
    object_id        INTEGER NOT NULL DEFAULT 0,
    term_taxonomy_id INTEGER NOT NULL DEFAULT 0,
    term_order       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (object_id, term_taxonomy_id)
);

CREATE INDEX idx_term_relationships_tt_id ON wp_term_relationships (term_taxonomy_id);

CREATE TABLE wp_termmeta (
    meta_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    term_id    INTEGER NOT NULL DEFAULT 0,
    meta_key   TEXT    DEFAULT NULL,
    meta_value TEXT    DEFAULT NULL
);

CREATE INDEX idx_termmeta_term_id ON wp_termmeta (term_id);
CREATE INDEX idx_termmeta_meta_key ON wp_termmeta (meta_key);

CREATE TABLE wp_links (
    link_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    link_url         TEXT    NOT NULL DEFAULT '',
    link_name        TEXT    NOT NULL DEFAULT '',
    link_image       TEXT    NOT NULL DEFAULT '',
    link_target      TEXT    NOT NULL DEFAULT '',
    link_description TEXT    NOT NULL DEFAULT '',
    link_visible     TEXT    NOT NULL DEFAULT 'Y',
    link_owner       INTEGER NOT NULL DEFAULT 1,
    link_rating      INTEGER NOT NULL DEFAULT 0,
    link_updated     DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
    link_rel         TEXT    NOT NULL DEFAULT '',
    link_notes       TEXT    NOT NULL DEFAULT '',
    link_rss         TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_links_visible ON wp_links (link_visible);

CREATE TABLE wp_options (
    option_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    option_name  TEXT    NOT NULL DEFAULT '',
    option_value TEXT    NOT NULL DEFAULT '',
    autoload     TEXT    NOT NULL DEFAULT 'yes'
);

CREATE UNIQUE INDEX idx_options_name ON wp_options (option_name);
CREATE INDEX idx_options_autoload ON wp_options (autoload);

-- +goose Down

DROP TABLE IF EXISTS wp_options;
DROP TABLE IF EXISTS wp_links;
DROP TABLE IF EXISTS wp_termmeta;
DROP TABLE IF EXISTS wp_term_relationships;
DROP TABLE IF EXISTS wp_term_taxonomy;
DROP TABLE IF EXISTS wp_terms;
DROP TABLE IF EXISTS wp_usermeta;
DROP TABLE IF EXISTS wp_users;
DROP TABLE IF EXISTS wp_commentmeta;
DROP TABLE IF EXISTS wp_comments;
DROP TABLE IF EXISTS wp_postmeta;
DROP TABLE IF EXISTS wp_posts;
