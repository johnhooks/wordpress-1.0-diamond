// Command load reads fixture JSON files and populates a Press database.
// Run from the project root:
//
//	go run ./data/fixtures/ <database-path>
//
// The database must already exist with migrations applied.
// All existing data is deleted before loading.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// ── JSON shapes ───────────────────────────────────────────────

type userJSON struct {
	Login       string `json:"login"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	URL         string `json:"url"`
	Level       string `json:"level"`
	Group       string `json:"group"`
}

type categoryJSON struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type postJSON struct {
	Author        string   `json:"author"`
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	Categories    []string `json:"categories"`
	DaysAfterBase int      `json:"daysAfterBase"`
	Content       []string `json:"content"`
}

type commentJSON struct {
	Post           string  `json:"post"`
	Author         string  `json:"author"`
	Email          string  `json:"email"`
	URL            string  `json:"url"`
	HoursAfterPost float64 `json:"hoursAfterPost"`
	Content        string  `json:"content"`
}

// ── Helpers ───────────────────────────────────────────────────

func prosemirrorDoc(paragraphs []string) string {
	content := ""
	for i, p := range paragraphs {
		if i > 0 {
			content += ","
		}
		b, _ := json.Marshal(p)
		content += fmt.Sprintf(`{"type":"paragraph","content":[{"type":"text","text":%s}]}`, string(b))
	}
	return fmt.Sprintf(`{"type":"doc","content":[%s]}`, content)
}

func fixturesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func loadJSON(name string, v any) error {
	data, err := os.ReadFile(filepath.Join(fixturesDir(), name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run ./data/fixtures/ <database-path>\n")
		os.Exit(1)
	}
	dbPath := os.Args[1]

	db, err := sqlx.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := load(db); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func load(db *sqlx.DB) error {
	// Load JSON
	var users []userJSON
	var categories []categoryJSON
	var posts []postJSON
	var comments []commentJSON

	if err := loadJSON("users.json", &users); err != nil {
		return fmt.Errorf("loading users: %w", err)
	}
	if err := loadJSON("categories.json", &categories); err != nil {
		return fmt.Errorf("loading categories: %w", err)
	}
	if err := loadJSON("posts.json", &posts); err != nil {
		return fmt.Errorf("loading posts: %w", err)
	}
	if err := loadJSON("comments.json", &comments); err != nil {
		return fmt.Errorf("loading comments: %w", err)
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing data (order matters for foreign keys)
	for _, table := range []string{
		"wp_term_relationships", "wp_comments", "wp_commentmeta",
		"wp_postmeta", "wp_posts", "wp_usermeta", "wp_users",
		"wp_term_taxonomy", "wp_termmeta", "wp_terms",
		"wp_tuples", "wp_groups", "wp_share_tokens",
		"wp_links", "wp_options",
	} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("clearing %s: %w", table, err)
		}
	}

	// All users get "password"
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	base := time.Date(2004, 1, 3, 10, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	// ── Users ─────────────────────────────────────────────

	userIDs := make(map[string]int64) // login → id

	for _, u := range users {
		result, err := tx.Exec(`
			INSERT INTO wp_users (user_login, user_pass, user_nicename, user_email, user_url, user_registered, display_name)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			u.Login, string(hash), u.Login, u.Email, u.URL, base, u.DisplayName,
		)
		if err != nil {
			return fmt.Errorf("creating user %s: %w", u.Login, err)
		}
		id, _ := result.LastInsertId()
		userIDs[u.Login] = id

		for k, v := range map[string]string{
			"wp_user_level": u.Level,
			"nickname":      u.Login,
			"rich_editing":  "true",
		} {
			if _, err := tx.Exec("INSERT INTO wp_usermeta (user_id, meta_key, meta_value) VALUES (?, ?, ?)", id, k, v); err != nil {
				return fmt.Errorf("creating usermeta for %s: %w", u.Login, err)
			}
		}
	}

	adminID := userIDs["admin"]

	// ── Categories ────────────────────────────────────────

	catTTIDs := make(map[string]int64) // slug → term_taxonomy_id

	for _, c := range categories {
		result, err := tx.Exec("INSERT INTO wp_terms (name, slug) VALUES (?, ?)", c.Name, c.Slug)
		if err != nil {
			return fmt.Errorf("creating term %s: %w", c.Name, err)
		}
		termID, _ := result.LastInsertId()

		result, err = tx.Exec(
			"INSERT INTO wp_term_taxonomy (term_id, taxonomy, description, count) VALUES (?, 'category', '', 0)",
			termID,
		)
		if err != nil {
			return fmt.Errorf("creating taxonomy for %s: %w", c.Name, err)
		}
		catTTIDs[c.Slug], _ = result.LastInsertId()
	}

	// Blogroll link category
	result, err := tx.Exec("INSERT INTO wp_terms (name, slug) VALUES ('Blogroll', 'blogroll')")
	if err != nil {
		return fmt.Errorf("creating blogroll: %w", err)
	}
	blogrollID, _ := result.LastInsertId()
	if _, err := tx.Exec("INSERT INTO wp_term_taxonomy (term_id, taxonomy, description) VALUES (?, 'link_category', '')", blogrollID); err != nil {
		return fmt.Errorf("creating blogroll taxonomy: %w", err)
	}

	// ── Posts ──────────────────────────────────────────────

	postIDs := make(map[string]int64) // slug → id
	postDates := make(map[string]time.Time)
	postAuthors := make(map[string]int64)
	catCounts := make(map[string]int)

	for _, p := range posts {
		authorID, ok := userIDs[p.Author]
		if !ok {
			return fmt.Errorf("unknown author %q for post %q", p.Author, p.Title)
		}

		date := base.Add(time.Duration(p.DaysAfterBase) * day)
		content := prosemirrorDoc(p.Content)

		result, err := tx.Exec(`
			INSERT INTO wp_posts (
				post_author, post_date, post_date_gmt, post_content, post_title,
				post_excerpt, post_status, comment_status, post_name,
				post_modified, post_modified_gmt, post_type, guid
			) VALUES (?, ?, ?, ?, ?, '', 'publish', 'open', ?, ?, ?, 'post', '')`,
			authorID, date, date, content, p.Title, p.Slug, date, date,
		)
		if err != nil {
			return fmt.Errorf("creating post %q: %w", p.Title, err)
		}
		id, _ := result.LastInsertId()
		postIDs[p.Slug] = id
		postDates[p.Slug] = date
		postAuthors[p.Slug] = authorID

		for _, catSlug := range p.Categories {
			ttID, ok := catTTIDs[catSlug]
			if !ok {
				return fmt.Errorf("unknown category %q for post %q", catSlug, p.Title)
			}
			if _, err := tx.Exec("INSERT INTO wp_term_relationships (object_id, term_taxonomy_id) VALUES (?, ?)", id, ttID); err != nil {
				return fmt.Errorf("linking post %q to category %q: %w", p.Title, catSlug, err)
			}
			catCounts[catSlug]++
		}
	}

	for slug, count := range catCounts {
		if _, err := tx.Exec("UPDATE wp_term_taxonomy SET count = ? WHERE term_taxonomy_id = ?", count, catTTIDs[slug]); err != nil {
			return fmt.Errorf("updating count for %s: %w", slug, err)
		}
	}

	// ── Comments ──────────────────────────────────────────

	commentCounts := make(map[int64]int)

	for _, c := range comments {
		postID, ok := postIDs[c.Post]
		if !ok {
			return fmt.Errorf("unknown post %q for comment by %s", c.Post, c.Author)
		}
		postDate := postDates[c.Post]
		commentDate := postDate.Add(time.Duration(c.HoursAfterPost * float64(time.Hour)))

		if _, err := tx.Exec(`
			INSERT INTO wp_comments (
				comment_post_id, comment_author, comment_author_email, comment_author_url,
				comment_author_ip, comment_date, comment_date_gmt, comment_content,
				comment_approved, comment_agent, comment_type, user_id
			) VALUES (?, ?, ?, ?, '127.0.0.1', ?, ?, ?, '1', 'fixtures', 'comment', 0)`,
			postID, c.Author, c.Email, c.URL, commentDate, commentDate, c.Content,
		); err != nil {
			return fmt.Errorf("creating comment on %q: %w", c.Post, err)
		}
		commentCounts[postID]++
	}

	for postID, count := range commentCounts {
		if _, err := tx.Exec("UPDATE wp_posts SET comment_count = ? WHERE id = ?", count, postID); err != nil {
			return fmt.Errorf("updating comment count: %w", err)
		}
	}

	// ── Permission Groups ─────────────────────────────────

	for _, g := range []struct{ slug, name string }{
		{"public", "Public"},
		{"administrators", "Administrators"},
		{"editors", "Editors"},
		{"authors", "Authors"},
		{"subscribers", "Subscribers"},
	} {
		if _, err := tx.Exec("INSERT INTO wp_groups (slug, name, is_default, created_at, created_by) VALUES (?, ?, 1, ?, ?)", g.slug, g.name, base, adminID); err != nil {
			return fmt.Errorf("creating group %s: %w", g.slug, err)
		}
	}

	// ── Permission Tuples ─────────────────────────────────

	type tuple struct{ st, si, rel, ot, oi string }

	tuples := []tuple{
		{"group", "public", "viewer", "type", "post"},
		{"group", "public", "viewer", "type", "page"},
		{"group", "public", "commenter", "type", "post"},
		{"group", "administrators", "owner", "site", ""},
		{"group", "administrators", "editor", "type", "post"},
		{"group", "administrators", "editor", "type", "page"},
		{"group", "administrators", "editor", "type", "attachment"},
		{"group", "editors", "moderator", "site", ""},
		{"group", "editors", "editor", "type", "post"},
		{"group", "editors", "editor", "type", "page"},
		{"group", "editors", "writer", "type", "attachment"},
		{"group", "authors", "writer", "type", "post"},
		{"group", "authors", "writer", "type", "attachment"},
	}

	// User → group memberships
	for _, u := range users {
		tuples = append(tuples, tuple{"user", fmt.Sprintf("%d", userIDs[u.Login]), "member", "group", u.Group})
	}

	// Post authorship
	for slug, postID := range postIDs {
		tuples = append(tuples, tuple{"user", fmt.Sprintf("%d", postAuthors[slug]), "writer", "post", fmt.Sprintf("%d", postID)})
	}

	for _, t := range tuples {
		if _, err := tx.Exec(`
			INSERT INTO wp_tuples (subject_type, subject_id, relation, object_type, object_id, created_at, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			t.st, t.si, t.rel, t.ot, t.oi, base, adminID); err != nil {
			return fmt.Errorf("creating tuple: %w", err)
		}
	}

	// ── Options ───────────────────────────────────────────

	for name, value := range map[string]string{
		"siteurl":                   "http://localhost:3000",
		"blogname":                  "Press",
		"blogdescription":           "WordPress 1.6 — the release that never happened",
		"posts_per_page":            "10",
		"date_format":               "F j, Y",
		"time_format":               "g:i a",
		"use_smilies":               "1",
		"use_quicktags":             "1",
		"use_balanceTags":           "0",
		"comment_moderation":        "1",
		"default_category":          "1",
		"template":                  "freerange",
		"db_version":                "1",
		"start_of_week":             "1",
		"links_updated_date_format": "F j, Y g:i a",
	} {
		if _, err := tx.Exec("INSERT INTO wp_options (option_name, option_value, autoload) VALUES (?, ?, 'yes')", name, value); err != nil {
			return fmt.Errorf("creating option %s: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Loaded: %d users, %d posts, %d comments, %d categories\n",
		len(users), len(posts), len(comments), len(categories))
	fmt.Println("All passwords: password")
	return nil
}
