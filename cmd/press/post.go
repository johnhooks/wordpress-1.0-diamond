package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"press/internal/model"
	"press/internal/query"
	"press/internal/repository"
	"press/internal/slug"

	"github.com/spf13/cobra"
)

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
				defer f.Close()
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

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

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

		if porcelain {
			fmt.Println(post.ID)
			return nil
		}

		fmt.Printf("Success: Created post %d.\n", post.ID)
		return nil
	},
}

var postListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all posts",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		ctx := context.Background()
		posts := repository.NewPostsRepository(db)
		result, err := posts.List(ctx, query.Query{
			Filters: []query.Filter{{Field: "type", Operator: query.Is, Value: "post"}},
			PerPage: 1000,
		})
		if err != nil {
			return err
		}

		if len(result.Items) == 0 {
			fmt.Println("No posts found.")
			return nil
		}

		fmt.Printf("%-4s %-40s %-12s %s\n", "ID", "Title", "Status", "Date")
		fmt.Println(strings.Repeat("-", 80))
		for _, p := range result.Items {
			fmt.Printf("%-4d %-40s %-12s %s\n", p.ID, truncate(p.PostTitle, 38), p.PostStatus, p.PostDate.Format("2006-01-02"))
		}
		return nil
	},
}

var postDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a post by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var id int64
		if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
			return fmt.Errorf("invalid post ID: %s", args[0])
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		ctx := context.Background()
		posts := repository.NewPostsRepository(db)
		if _, err := posts.Delete(ctx, id); err != nil {
			return err
		}

		fmt.Printf("Post %d deleted.\n", id)
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
	postCreateCmd.Flags().Bool("porcelain", false, "output just the new post ID")

	postCmd.AddCommand(postCreateCmd, postListCmd, postDeleteCmd)
	rootCmd.AddCommand(postCmd)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
