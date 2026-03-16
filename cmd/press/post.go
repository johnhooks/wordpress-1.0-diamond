package main

import (
	"context"
	"fmt"
	"strings"

	"press/internal/query"
	"press/internal/repository"

	"github.com/spf13/cobra"
)

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Manage posts",
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
	postCmd.AddCommand(postListCmd, postDeleteCmd)
	rootCmd.AddCommand(postCmd)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
