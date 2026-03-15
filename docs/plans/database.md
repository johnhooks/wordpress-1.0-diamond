# Database Schema

## Approach

Use the modern WordPress schema (from `wordpress-develop/src/wp-admin/includes/schema.php`) as our foundation. The original WP 1.0 had 10+ tables with some awkward designs (4 tables for options, hardcoded category tables, 15+ inline user profile columns). Modern WordPress cleaned all of this up over 20 years. We take advantage of that.

The surface still looks like WP 1.0. The user sees categories, not "taxonomies." The admin says user levels, not "capabilities." But underneath, it is the modern table structure.

## Engine

SQLite only. Single file, zero configuration. The database file lives next to the binary by default, configurable via environment variable or config file.

The SQLite driver is `modernc.org/sqlite`, a pure Go implementation that requires no CGO.

## Tables

12 tables total. Based on modern WordPress schema, adapted for SQLite syntax and our needs.

### wp_posts

Posts and attachments. Faithful to WP 1.0 post model but with modern schema improvements.

```sql
CREATE TABLE wp_posts (
    ID              INTEGER PRIMARY KEY AUTOINCREMENT,
    post_author     INTEGER NOT NULL DEFAULT 0,
    post_date       TEXT    NOT NULL DEFAULT '0000-00-00 00:00:00',
    post_date_gmt   TEXT    NOT NULL DEFAULT '0000-00-00 00:00:00',
    post_content    TEXT    NOT NULL DEFAULT '',
    post_title      TEXT    NOT NULL DEFAULT '',
    post_excerpt    TEXT    NOT NULL DEFAULT '',
    post_status     TEXT    NOT NULL DEFAULT 'publish',
    comment_status  TEXT    NOT NULL DEFAULT 'open',
    ping_status     TEXT    NOT NULL DEFAULT 'open',
    post_password   TEXT    NOT NULL DEFAULT '',
    post_name       TEXT    NOT NULL DEFAULT '',
    to_ping         TEXT    NOT NULL DEFAULT '',
    pinged          TEXT    NOT NULL DEFAULT '',
    post_modified       TEXT NOT NULL DEFAULT '0000-00-00 00:00:00',
    post_modified_gmt   TEXT NOT NULL DEFAULT '0000-00-00 00:00:00',
    post_content_filtered TEXT NOT NULL DEFAULT '',
    post_parent     INTEGER NOT NULL DEFAULT 0,
    guid            TEXT    NOT NULL DEFAULT '',
    menu_order      INTEGER NOT NULL DEFAULT 0,
    post_type       TEXT    NOT NULL DEFAULT 'post',
    post_mime_type  TEXT    NOT NULL DEFAULT '',
    comment_count   INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_posts_type_status_date ON wp_posts (post_type, post_status, post_date, ID);
CREATE INDEX idx_posts_parent ON wp_posts (post_parent);
CREATE INDEX idx_posts_author ON wp_posts (post_author);
CREATE INDEX idx_posts_name ON wp_posts (post_name);
```

**Changes from WP 1.0:** Added post_name (slug), post_modified, post_parent, post_type, guid, menu_order, comment_count, and GMT timestamps. Dropped post_lat/post_lon because GeoURL is a dead feature. Dropped the direct post_category column in favor of the taxonomy system.

**Changes from modern WP:** Using TEXT instead of MySQL's longtext/varchar. Using TEXT instead of ENUM (SQLite doesn't have ENUM). Datetimes stored as TEXT in ISO-ish format (SQLite convention).

### wp_postmeta

Extensible key-value metadata for posts. Mostly unused in the WP 1.0 surface but included for completeness since it costs nothing.

```sql
CREATE TABLE wp_postmeta (
    meta_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL DEFAULT 0,
    meta_key   TEXT    DEFAULT NULL,
    meta_value TEXT    DEFAULT NULL
);

CREATE INDEX idx_postmeta_post_id ON wp_postmeta (post_id);
CREATE INDEX idx_postmeta_meta_key ON wp_postmeta (meta_key);
```

### wp_comments

```sql
CREATE TABLE wp_comments (
    comment_ID           INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_post_ID      INTEGER NOT NULL DEFAULT 0,
    comment_author       TEXT    NOT NULL DEFAULT '',
    comment_author_email TEXT    NOT NULL DEFAULT '',
    comment_author_url   TEXT    NOT NULL DEFAULT '',
    comment_author_IP    TEXT    NOT NULL DEFAULT '',
    comment_date         TEXT    NOT NULL DEFAULT '0000-00-00 00:00:00',
    comment_date_gmt     TEXT    NOT NULL DEFAULT '0000-00-00 00:00:00',
    comment_content      TEXT    NOT NULL DEFAULT '',
    comment_karma        INTEGER NOT NULL DEFAULT 0,
    comment_approved     TEXT    NOT NULL DEFAULT '1',
    comment_agent        TEXT    NOT NULL DEFAULT '',
    comment_type         TEXT    NOT NULL DEFAULT 'comment',
    comment_parent       INTEGER NOT NULL DEFAULT 0,
    user_id              INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_comments_post_id ON wp_comments (comment_post_ID);
CREATE INDEX idx_comments_approved_date ON wp_comments (comment_approved, comment_date_gmt);
CREATE INDEX idx_comments_parent ON wp_comments (comment_parent);
```

**Changes from WP 1.0:** Added comment_agent, comment_type, comment_parent (threading), user_id, GMT timestamp. Changed comment_approved from ENUM('0','1') to TEXT (allows 'spam', 'trash' states if we ever want them).

### wp_commentmeta

```sql
CREATE TABLE wp_commentmeta (
    meta_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id   INTEGER NOT NULL DEFAULT 0,
    meta_key     TEXT    DEFAULT NULL,
    meta_value   TEXT    DEFAULT NULL
);

CREATE INDEX idx_commentmeta_comment_id ON wp_commentmeta (comment_id);
CREATE INDEX idx_commentmeta_meta_key ON wp_commentmeta (meta_key);
```

### wp_users

Slim modern version. All the old profile fields (AIM, ICQ, MSN, YIM, etc.) live in usermeta.

```sql
CREATE TABLE wp_users (
    ID                  INTEGER PRIMARY KEY AUTOINCREMENT,
    user_login          TEXT    NOT NULL DEFAULT '',
    user_pass           TEXT    NOT NULL DEFAULT '',
    user_nicename       TEXT    NOT NULL DEFAULT '',
    user_email          TEXT    NOT NULL DEFAULT '',
    user_url            TEXT    NOT NULL DEFAULT '',
    user_registered     TEXT    NOT NULL DEFAULT '0000-00-00 00:00:00',
    user_activation_key TEXT    NOT NULL DEFAULT '',
    user_status         INTEGER NOT NULL DEFAULT 0,
    display_name        TEXT    NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_users_login ON wp_users (user_login);
CREATE INDEX idx_users_nicename ON wp_users (user_nicename);
CREATE INDEX idx_users_email ON wp_users (user_email);
```

**Changes from WP 1.0:** Dropped 15+ inline columns (user_firstname, user_lastname, user_nickname, user_icq, user_aim, user_msn, user_yim, user_ip, user_domain, user_browser, user_level, user_idmode, user_description). These move to usermeta. Added user_nicename (slug), display_name, user_activation_key, user_status. Password field stores bcrypt hashes.

### wp_usermeta

All user profile fields and capabilities stored as key-value pairs.

```sql
CREATE TABLE wp_usermeta (
    umeta_id   INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 0,
    meta_key   TEXT    DEFAULT NULL,
    meta_value TEXT    DEFAULT NULL
);

CREATE INDEX idx_usermeta_user_id ON wp_usermeta (user_id);
CREATE INDEX idx_usermeta_meta_key ON wp_usermeta (meta_key);
```

**Standard meta keys we'll use (matching WP 1.0 user fields):**
- `first_name`, `last_name`, `nickname`, `description`
- `aim`, `yim`, `jabber` (was MSN)
- `wp_user_level` is a numeric value from 0-10, same as the original.
- `rich_editing` stores the quicktags preference.

### wp_terms

The base vocabulary. Each category name is a term.

```sql
CREATE TABLE wp_terms (
    term_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL DEFAULT '',
    slug       TEXT    NOT NULL DEFAULT '',
    term_group INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_terms_slug ON wp_terms (slug);
CREATE INDEX idx_terms_name ON wp_terms (name);
```

### wp_term_taxonomy

Links terms to their taxonomy type (category, link_category) and adds hierarchy.

```sql
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
```

**Taxonomies we use:**
- `category` for post categories, replacing wp_categories.
- `link_category` for blogroll link categories, replacing wp_linkcategories.

### wp_term_relationships

Junction table linking posts (or links) to their taxonomy terms.

```sql
CREATE TABLE wp_term_relationships (
    object_id        INTEGER NOT NULL DEFAULT 0,
    term_taxonomy_id INTEGER NOT NULL DEFAULT 0,
    term_order       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (object_id, term_taxonomy_id)
);

CREATE INDEX idx_term_relationships_tt_id ON wp_term_relationships (term_taxonomy_id);
```

**object_id** is the post ID or link ID depending on the taxonomy.

### wp_termmeta

```sql
CREATE TABLE wp_termmeta (
    meta_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    term_id    INTEGER NOT NULL DEFAULT 0,
    meta_key   TEXT    DEFAULT NULL,
    meta_value TEXT    DEFAULT NULL
);

CREATE INDEX idx_termmeta_term_id ON wp_termmeta (term_id);
CREATE INDEX idx_termmeta_meta_key ON wp_termmeta (meta_key);
```

**Used for link category display options** that the original stored as columns on wp_linkcategories: show_images, show_description, show_rating, show_updated, sort_order, sort_desc, text_before_link, text_after_link, text_after_all, list_limit.

### wp_links

Blogroll. Nearly unchanged from the original.

```sql
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
    link_updated     TEXT    NOT NULL DEFAULT '0000-00-00 00:00:00',
    link_rel         TEXT    NOT NULL DEFAULT '',
    link_notes       TEXT    NOT NULL DEFAULT '',
    link_rss         TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_links_visible ON wp_links (link_visible);
```

**Note:** link_category column is gone. Link-to-category relationships go through wp_term_relationships with taxonomy='link_category'.

### wp_options

Simple key-value store. The modern version, not the 4-table monstrosity from 1.0.

```sql
CREATE TABLE wp_options (
    option_id    INTEGER PRIMARY KEY AUTOINCREMENT,
    option_name  TEXT    NOT NULL DEFAULT '',
    option_value TEXT    NOT NULL DEFAULT '',
    autoload     TEXT    NOT NULL DEFAULT 'yes'
);

CREATE UNIQUE INDEX idx_options_name ON wp_options (option_name);
CREATE INDEX idx_options_autoload ON wp_options (autoload);
```

**Replaces:** wp_options (old version), wp_optiontypes, wp_optiongroups, wp_optiongroup_options, wp_optionvalues. All 5 tables collapsed into 1.

## Taxonomy Mapping

How WP 1.0 category operations map to the modern taxonomy tables:

| WP 1.0 Operation | Old Tables | Modern Equivalent |
|---|---|---|
| List categories | `SELECT * FROM wp_categories` | `SELECT t.*, tt.* FROM wp_terms t JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id WHERE tt.taxonomy = 'category'` |
| Get post categories | `SELECT * FROM wp_post2cat WHERE post_id = ?` | `SELECT t.* FROM wp_terms t JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id JOIN wp_term_relationships tr ON tt.term_taxonomy_id = tr.term_taxonomy_id WHERE tr.object_id = ? AND tt.taxonomy = 'category'` |
| Add post to category | `INSERT INTO wp_post2cat (post_id, category_id)` | `INSERT INTO wp_term_relationships (object_id, term_taxonomy_id)` + update count |
| Category hierarchy | `category_parent` column | `parent` column on `wp_term_taxonomy` |
| Link categories | `SELECT * FROM wp_linkcategories` | Same as categories but `WHERE tt.taxonomy = 'link_category'` |
| Link category display options | Columns on `wp_linkcategories` | Key-value pairs in `wp_termmeta` |

## Table Summary

| Table | Purpose | WP 1.0 Equivalent |
|-------|---------|-------------------|
| wp_posts | Posts | wp_posts (expanded) |
| wp_postmeta | Post metadata | (new) |
| wp_comments | Comments | wp_comments (expanded) |
| wp_commentmeta | Comment metadata | (new) |
| wp_users | User accounts | wp_users (slimmed) |
| wp_usermeta | User profile data | (new, replaces inline columns) |
| wp_terms | Category/tag names | wp_categories (generalized) |
| wp_term_taxonomy | Taxonomy assignments | wp_categories (split out) |
| wp_term_relationships | Object-to-term links | wp_post2cat (generalized) |
| wp_termmeta | Term metadata | wp_linkcategories display options |
| wp_links | Blogroll | wp_links (minor changes) |
| wp_options | Settings | wp_options + 4 helper tables (collapsed) |

## Seed Data

On first run, the database is created and seeded with:

- **Admin user** with login `admin`, a generated password, and user_level 10.
- **Default category** called "Uncategorized" stored as a term with term_taxonomy taxonomy='category'.
- **Sample post** titled "Hello world!" in Uncategorized, matching the original.
- **Default link category** called "Blogroll" stored as a term with term_taxonomy taxonomy='link_category'.
- **Default options** including blogname, blogdescription, siteurl, posts_per_page, date_format, time_format, comment_moderation, use_smilies, use_bbcode, use_quicktags, use_htmltrans, use_balanceTags, and others matching WP 1.0 defaults.

## Options Caching Strategy

Modern WordPress evolved a brilliant multi-layer caching system for options over 20 years. We implement this from day one.

### Three-Layer Lookup

```
1. Check in-memory map (autoloaded options)     → O(1) map lookup
2. Check "notoptions" set (known non-existent)   → O(1) map lookup
3. Query database                                → parameterized SELECT
   → cache result OR add to notoptions set
```

### Autoload System

Options with `autoload = 'yes'` are bulk-loaded into an in-memory `map[string]string` on startup via a single query. Most settings reads hit this map and never touch the database.

### Notoptions Cache

When we query for an option that doesn't exist, we record that fact. This prevents repeated database queries for non-existent options.

### Cache Invalidation

On update:
- If the option is in the autoload map → update it in place
- If moving from non-autoload to autoload → add to map
- If moving from autoload to non-autoload → remove from map
- Clear from notoptions set if present

On delete:
- Remove from whichever cache layer it's in
- Add to notoptions set

## Migrations

Schema versioning via a `db_version` option in wp_options. On startup, the app checks the current version and runs any pending migrations. For v1 there's just one migration: create all tables and seed data.
