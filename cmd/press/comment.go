package main

import (
	"context"
	"fmt"

	"press/internal/query"
	"press/internal/model"
	"press/internal/repository"

	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage comments",
}

var commentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		postID, _ := cmd.Flags().GetInt64("post")
		content, _ := cmd.Flags().GetString("content")
		author, _ := cmd.Flags().GetString("author")
		authorEmail, _ := cmd.Flags().GetString("author-email")
		authorURL, _ := cmd.Flags().GetString("author-url")
		status, _ := cmd.Flags().GetString("status")
		porcelain, _ := cmd.Flags().GetBool("porcelain")

		if postID == 0 {
			return fmt.Errorf("--post is required")
		}
		if content == "" {
			return fmt.Errorf("--content is required")
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		comments := repository.NewCommentsRepository(db)

		comment := &model.Comment{
			CommentPostID:      postID,
			CommentAuthor:      author,
			CommentAuthorEmail: authorEmail,
			CommentAuthorURL:   authorURL,
			CommentContent:     content,
			CommentApproved:    status,
			CommentAgent:       "press-cli",
			CommentType:        "comment",
		}

		if err := comments.Create(ctx, comment); err != nil {
			return err
		}

		if porcelain {
			fmt.Println(comment.CommentID)
			return nil
		}

		fmt.Printf("Success: Created comment %d.\n", comment.CommentID)
		return nil
	},
}

var commentGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a comment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		id, _ := resolveIDOrString(args[0])
		if id == 0 {
			return fmt.Errorf("invalid comment ID: %s", args[0])
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		comments := repository.NewCommentsRepository(db)

		comment, err := comments.GetByID(ctx, id)
		if err != nil {
			return err
		}

		formatSingle(format, []kv{
			{"ID", fmt.Sprintf("%d", comment.CommentID)},
			{"Post", fmt.Sprintf("%d", comment.CommentPostID)},
			{"Author", comment.CommentAuthor},
			{"Email", comment.CommentAuthorEmail},
			{"URL", comment.CommentAuthorURL},
			{"Date", comment.CommentDate.Format("2006-01-02 15:04:05")},
			{"Status", comment.CommentApproved},
			{"Type", comment.CommentType},
			{"Content", comment.CommentContent},
		})
		return nil
	},
}

var commentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List comments",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		postID, _ := cmd.Flags().GetInt64("post")
		status, _ := cmd.Flags().GetString("status")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		comments := repository.NewCommentsRepository(db)

		var filters []query.Filter
		if postID > 0 {
			filters = append(filters, query.Filter{Field: "post", Operator: query.Is, Value: postID})
		}
		if status != "" {
			filters = append(filters, query.Filter{Field: "approved", Operator: query.Is, Value: status})
		}

		result, err := comments.List(ctx, query.Query{
			Filters: filters,
			PerPage: 1000,
		})
		if err != nil {
			return err
		}

		columns := []column{
			{"ID", 4},
			{"Post", 6},
			{"Author", 20},
			{"Date", 19},
			{"Content", 40},
		}

		var rows []map[string]any
		for _, c := range result.Items {
			rows = append(rows, map[string]any{
				"ID":      c.CommentID,
				"Post":    c.CommentPostID,
				"Author":  c.CommentAuthor,
				"Date":    c.CommentDate.Format("2006-01-02 15:04"),
				"Content": c.CommentContent,
			})
		}

		if len(rows) == 0 {
			fmt.Println("No comments found.")
			return nil
		}

		formatList(format, "ID", columns, rows)
		return nil
	},
}

var commentDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a comment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := resolveIDOrString(args[0])
		if id == 0 {
			return fmt.Errorf("invalid comment ID: %s", args[0])
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		comments := repository.NewCommentsRepository(db)

		if _, err := comments.Delete(ctx, id); err != nil {
			return err
		}

		fmt.Printf("Success: Deleted comment %d.\n", id)
		return nil
	},
}

func init() {
	commentCreateCmd.Flags().Int64("post", 0, "post ID (required)")
	commentCreateCmd.Flags().String("content", "", "comment content (required)")
	commentCreateCmd.Flags().String("author", "", "commenter name")
	commentCreateCmd.Flags().String("author-email", "", "commenter email")
	commentCreateCmd.Flags().String("author-url", "", "commenter URL")
	commentCreateCmd.Flags().String("status", "1", "approval status (1=approved, 0=pending)")
	commentCreateCmd.Flags().Bool("porcelain", false, "output just the new comment ID")

	commentGetCmd.Flags().String("format", "table", "output format (table, json)")

	commentListCmd.Flags().String("format", "table", "output format (table, json, csv, ids)")
	commentListCmd.Flags().Int64("post", 0, "filter by post ID")
	commentListCmd.Flags().String("status", "", "filter by status (1=approved, 0=pending)")

	commentCmd.AddCommand(commentCreateCmd, commentGetCmd, commentListCmd, commentDeleteCmd)
	rootCmd.AddCommand(commentCmd)
}
