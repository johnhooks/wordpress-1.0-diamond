package seed

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func Run(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Generate random password
	passwordBytes := make([]byte, 8)
	if _, err := rand.Read(passwordBytes); err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}
	plainPassword := hex.EncodeToString(passwordBytes)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 1. Admin user
	result, err := tx.Exec(`
		INSERT INTO wp_users (user_login, user_pass, user_nicename, user_email, user_url, user_registered, display_name)
		VALUES ('admin', ?, 'admin', 'admin@example.com', '', ?, 'admin')`,
		string(hashedPassword), now,
	)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	adminID, _ := result.LastInsertId()

	// 2. Usermeta
	metas := map[string]string{
		"wp_user_level": "10",
		"first_name":    "",
		"last_name":     "",
		"nickname":      "admin",
		"rich_editing":  "true",
	}
	for key, value := range metas {
		_, err := tx.Exec("INSERT INTO wp_usermeta (user_id, meta_key, meta_value) VALUES (?, ?, ?)", adminID, key, value)
		if err != nil {
			return fmt.Errorf("failed to create usermeta %s: %w", key, err)
		}
	}

	// 3. "Uncategorized" term + term_taxonomy
	result, err = tx.Exec("INSERT INTO wp_terms (name, slug) VALUES ('Uncategorized', 'uncategorized')")
	if err != nil {
		return fmt.Errorf("failed to create uncategorized term: %w", err)
	}
	uncatTermID, _ := result.LastInsertId()

	result, err = tx.Exec(
		"INSERT INTO wp_term_taxonomy (term_id, taxonomy, description, count) VALUES (?, 'category', '', 1)",
		uncatTermID,
	)
	if err != nil {
		return fmt.Errorf("failed to create uncategorized taxonomy: %w", err)
	}
	uncatTTID, _ := result.LastInsertId()

	// 4. "Blogroll" term + term_taxonomy
	result, err = tx.Exec("INSERT INTO wp_terms (name, slug) VALUES ('Blogroll', 'blogroll')")
	if err != nil {
		return fmt.Errorf("failed to create blogroll term: %w", err)
	}
	blogrollTermID, _ := result.LastInsertId()

	_, err = tx.Exec(
		"INSERT INTO wp_term_taxonomy (term_id, taxonomy, description) VALUES (?, 'link_category', '')",
		blogrollTermID,
	)
	if err != nil {
		return fmt.Errorf("failed to create blogroll taxonomy: %w", err)
	}

	// 5. "Hello world!" post
	result, err = tx.Exec(`
		INSERT INTO wp_posts (
			post_author, post_date, post_date_gmt, post_content, post_title,
			post_excerpt, post_status, comment_status, post_name,
			post_modified, post_modified_gmt, post_type, guid
		) VALUES (?, ?, ?, ?, 'Hello world!', '', 'publish', 'open', 'hello-world', ?, ?, 'post', '')`,
		adminID, now, now,
		`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Welcome to Press. This is your first post. Edit or delete it, then start writing!"}]}]}`,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create hello world post: %w", err)
	}
	postID, _ := result.LastInsertId()

	// Link post to Uncategorized
	_, err = tx.Exec(
		"INSERT INTO wp_term_relationships (object_id, term_taxonomy_id) VALUES (?, ?)",
		postID, uncatTTID,
	)
	if err != nil {
		return fmt.Errorf("failed to link post to category: %w", err)
	}

	// 6. Permission groups
	groups := []struct {
		slug string
		name string
	}{
		{"public", "Public"},
		{"administrators", "Administrators"},
		{"editors", "Editors"},
		{"authors", "Authors"},
		{"subscribers", "Subscribers"},
	}
	for _, g := range groups {
		_, err := tx.Exec(`
			INSERT INTO wp_groups (slug, name, is_default, created_at, created_by)
			VALUES (?, ?, 1, ?, ?)`, g.slug, g.name, now, adminID)
		if err != nil {
			return fmt.Errorf("failed to create group %s: %w", g.slug, err)
		}
	}

	// 7. Permission tuples
	tuples := []struct {
		subjectType string
		subjectID   string
		relation    string
		objectType  string
		objectID    string
	}{
		// Public (applies to everyone, including anonymous)
		{"group", "public", "viewer", "type", "post"},
		{"group", "public", "viewer", "type", "page"},
		{"group", "public", "commenter", "type", "post"},
		// Administrators — site administration + content control
		{"group", "administrators", "owner", "site", ""},
		{"group", "administrators", "editor", "type", "post"},
		{"group", "administrators", "editor", "type", "page"},
		{"group", "administrators", "editor", "type", "attachment"},
		// Editors — content control + moderation
		{"group", "editors", "moderator", "site", ""},
		{"group", "editors", "editor", "type", "post"},
		{"group", "editors", "editor", "type", "page"},
		{"group", "editors", "writer", "type", "attachment"},
		// Authors — create and manage own content
		{"group", "authors", "writer", "type", "post"},
		{"group", "authors", "writer", "type", "attachment"},
		// Admin user membership
		{"user", fmt.Sprintf("%d", adminID), "member", "group", "administrators"},
		// Hello World post: authorship
		{"user", fmt.Sprintf("%d", adminID), "writer", "post", fmt.Sprintf("%d", postID)},
	}
	for _, t := range tuples {
		_, err := tx.Exec(`
			INSERT INTO wp_tuples (subject_type, subject_id, relation, object_type, object_id, created_at, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			t.subjectType, t.subjectID, t.relation, t.objectType, t.objectID, now, adminID)
		if err != nil {
			return fmt.Errorf("failed to create tuple %s:%s→%s→%s:%s: %w",
				t.subjectType, t.subjectID, t.relation, t.objectType, t.objectID, err)
		}
	}

	// 8. Default options
	options := map[string]string{
		"siteurl":            "http://localhost:3000",
		"blogname":           "My Weblog",
		"blogdescription":    "Just another Press weblog",
		"posts_per_page":     "10",
		"date_format":        "F j, Y",
		"time_format":        "g:i a",
		"use_smilies":        "1",
		"use_quicktags":      "1",
		"use_balanceTags":    "0",
		"comment_moderation": "1",
		"default_category":   fmt.Sprintf("%d", uncatTermID),
		"template":           "default",
		"db_version":         "1",
		"start_of_week":      "1",
		"links_updated_date_format": "F j, Y g:i a",
	}

	for name, value := range options {
		_, err := tx.Exec(`
			INSERT INTO wp_options (option_name, option_value, autoload)
			VALUES (?, ?, 'yes')`, name, value,
		)
		if err != nil {
			return fmt.Errorf("failed to create option %s: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seed data: %w", err)
	}

	fmt.Printf("Admin user created.\n  Login: admin\n  Password: %s\n", plainPassword)
	return nil
}
