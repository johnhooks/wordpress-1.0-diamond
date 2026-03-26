package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"press/internal/model"
	"press/internal/permission"
	"press/internal/query"
	"press/internal/repository"
	"press/internal/slug"

	"github.com/spf13/cobra"
)

// wrapProseMirror wraps plain text content in ProseMirror document JSON.
// If the content already looks like ProseMirror JSON, it is returned as-is.
func wrapProseMirror(content string) string {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		// Check if it is valid ProseMirror JSON
		var doc map[string]any
		if json.Unmarshal([]byte(trimmed), &doc) == nil {
			if _, ok := doc["type"]; ok {
				return trimmed
			}
		}
	}

	// Split on double newlines into paragraphs
	paragraphs := strings.Split(trimmed, "\n\n")
	parts := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		escaped, _ := json.Marshal(p)
		parts = append(parts, fmt.Sprintf(`{"type":"paragraph","content":[{"type":"text","text":%s}]}`, string(escaped)))
	}
	return fmt.Sprintf(`{"type":"doc","content":[%s]}`, strings.Join(parts, ","))
}

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Manage posts",
}

var postCreateCmd = &cobra.Command{
	Use:   "create [file]",
	Short: "Create a new post",
	Long:  `Create a new post. Optionally read content from a file (use - for stdin).`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		content, _ := cmd.Flags().GetString("content")
		status, _ := cmd.Flags().GetString("status")
		postType, _ := cmd.Flags().GetString("type")
		author, _ := cmd.Flags().GetInt64("author")
		slugFlag, _ := cmd.Flags().GetString("slug")
		category, _ := cmd.Flags().GetString("category")
		porcelain, _ := cmd.Flags().GetBool("porcelain")

		// Read content from file or stdin if positional arg provided
		if len(args) > 0 {
			var r io.Reader
			if args[0] == "-" {
				r = os.Stdin
			} else {
				f, err := os.Open(args[0])
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer func() { _ = f.Close() }()
				r = f
			}
			data, err := io.ReadAll(r)
			if err != nil {
				return fmt.Errorf("failed to read content: %w", err)
			}
			content = string(data)
		}

		if title == "" {
			return fmt.Errorf("--title is required")
		}

		// Wrap content in ProseMirror JSON
		if content != "" {
			content = wrapProseMirror(content)
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		posts := repository.NewPostsRepository(db)

		// Generate slug from title if not provided
		postSlug := slugFlag
		if postSlug == "" {
			postSlug = slug.Generate(title)
		}
		postSlug, err = posts.EnsureUniqueSlug(ctx, postSlug, 0)
		if err != nil {
			return fmt.Errorf("failed to generate unique slug: %w", err)
		}

		post := &model.Post{
			PostAuthor:    author,
			PostContent:   content,
			PostTitle:     title,
			PostStatus:    status,
			PostName:      postSlug,
			PostType:      postType,
			CommentStatus: "open",
		}

		if err := posts.Create(ctx, post); err != nil {
			return err
		}

		// Assign category if provided
		if category != "" {
			terms := repository.NewTermsRepository(db)

			// Try by slug first, then by name
			term, err := terms.GetBySlug(ctx, category)
			if err != nil {
				term, err = terms.GetByName(ctx, category, "category")
			}
			if err != nil {
				return fmt.Errorf("category not found: %s", category)
			}

			tt, err := terms.GetTaxonomy(ctx, term.TermID, "category")
			if err != nil {
				return fmt.Errorf("category taxonomy not found: %w", err)
			}

			if err := terms.AddTermToPost(ctx, post.ID, tt.TermTaxonomyID); err != nil {
				return fmt.Errorf("failed to assign category: %w", err)
			}
		}

		// Create authorship tuple
		perms := permission.NewStore(db)
		if err := perms.CreateTuple(ctx, &permission.Tuple{
			SubjectType: "user",
			SubjectID:   fmt.Sprintf("%d", author),
			Relation:    "writer",
			ObjectType:  "post",
			ObjectID:    fmt.Sprintf("%d", post.ID),
			CreatedAt:   time.Now().UTC(),
			CreatedBy:   author,
		}); err != nil {
			return fmt.Errorf("failed to create authorship tuple: %w", err)
		}

		if porcelain {
			fmt.Println(post.ID)
			return nil
		}

		fmt.Printf("Success: Created post %d.\n", post.ID)
		return nil
	},
}

var postGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "Get a post",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		posts := repository.NewPostsRepository(db)

		var post *model.Post
		if id, s := resolveIDOrString(args[0]); id > 0 {
			post, err = posts.GetByID(ctx, id)
		} else {
			post, err = posts.GetBySlug(ctx, s)
		}
		if err != nil {
			return err
		}

		formatSingle(format, []kv{
			{"ID", fmt.Sprintf("%d", post.ID)},
			{"Title", post.PostTitle},
			{"Slug", post.PostName},
			{"Status", post.PostStatus},
			{"Type", post.PostType},
			{"Author", fmt.Sprintf("%d", post.PostAuthor)},
			{"Date", post.PostDate.Format("2006-01-02 15:04:05")},
			{"Comment Status", post.CommentStatus},
			{"Comment Count", fmt.Sprintf("%d", post.CommentCount)},
		})
		return nil
	},
}

var postListCmd = &cobra.Command{
	Use:   "list",
	Short: "List posts",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		status, _ := cmd.Flags().GetString("status")
		postType, _ := cmd.Flags().GetString("type")
		author, _ := cmd.Flags().GetInt64("author")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		posts := repository.NewPostsRepository(db)

		filters := []query.Filter{
			{Field: "type", Operator: query.Is, Value: postType},
		}
		if status != "" {
			filters = append(filters, query.Filter{Field: "status", Operator: query.Is, Value: status})
		}
		if author > 0 {
			filters = append(filters, query.Filter{Field: "author", Operator: query.Is, Value: author})
		}

		result, err := posts.List(ctx, query.Query{
			Filters: filters,
			PerPage: 1000,
		})
		if err != nil {
			return err
		}

		columns := []column{
			{"ID", 4},
			{"Title", 40},
			{"Status", 12},
			{"Date", 10},
		}

		var rows []map[string]any
		for _, p := range result.Items {
			rows = append(rows, map[string]any{
				"ID":     p.ID,
				"Title":  p.PostTitle,
				"Status": p.PostStatus,
				"Date":   p.PostDate.Format("2006-01-02"),
			})
		}

		if len(rows) == 0 {
			fmt.Println("No posts found.")
			return nil
		}

		formatList(format, "ID", columns, rows)
		return nil
	},
}

var postDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-slug>",
	Short: "Delete a post",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		posts := repository.NewPostsRepository(db)

		var id int64
		if numID, s := resolveIDOrString(args[0]); numID > 0 {
			id = numID
		} else {
			post, err := posts.GetBySlug(ctx, s)
			if err != nil {
				return err
			}
			id = post.ID
		}

		if _, err := posts.Delete(ctx, id); err != nil {
			return err
		}

		// Clean up permission tuples
		perms := permission.NewStore(db)
		if _, err := perms.DeleteTuplesForObject(ctx, "post", fmt.Sprintf("%d", id)); err != nil {
			return fmt.Errorf("failed to clean up permission tuples: %w", err)
		}

		fmt.Printf("Success: Deleted post %d.\n", id)
		return nil
	},
}

func init() {
	postCreateCmd.Flags().String("title", "", "post title (required)")
	postCreateCmd.Flags().String("content", "", "post content")
	postCreateCmd.Flags().String("status", "draft", "post status (draft, publish, pending, private)")
	postCreateCmd.Flags().String("type", "post", "post type")
	postCreateCmd.Flags().Int64("author", 1, "post author user ID")
	postCreateCmd.Flags().String("slug", "", "post slug (generated from title if omitted)")
	postCreateCmd.Flags().String("category", "", "category name or slug")
	postCreateCmd.Flags().Bool("porcelain", false, "output just the new post ID")

	postGetCmd.Flags().String("format", "table", "output format (table, json)")

	postListCmd.Flags().String("format", "table", "output format (table, json, csv, ids)")
	postListCmd.Flags().String("status", "", "filter by status")
	postListCmd.Flags().String("type", "post", "filter by post type")
	postListCmd.Flags().Int64("author", 0, "filter by author ID")

	postCmd.AddCommand(postCreateCmd, postGetCmd, postListCmd, postDeleteCmd)
	rootCmd.AddCommand(postCmd)
}
