package main

import (
	"context"
	"fmt"

	"press/internal/model"
	"press/internal/query"
	"press/internal/repository"

	"github.com/spf13/cobra"
)

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage categories",
}

var categoryCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		slugFlag, _ := cmd.Flags().GetString("slug")
		description, _ := cmd.Flags().GetString("description")
		parent, _ := cmd.Flags().GetString("parent")
		porcelain, _ := cmd.Flags().GetBool("porcelain")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		terms := repository.NewTermsRepository(db)

		// Resolve parent
		var parentID int64
		if parent != "" {
			if id, slug := resolveIDOrString(parent); id > 0 {
				tt, err := terms.GetTaxonomy(ctx, id, "category")
				if err != nil {
					return fmt.Errorf("parent category not found: %s", parent)
				}
				parentID = tt.TermTaxonomyID
			} else {
				term, err := terms.GetBySlug(ctx, slug)
				if err != nil {
					return fmt.Errorf("parent category not found: %s", parent)
				}
				tt, err := terms.GetTaxonomy(ctx, term.TermID, "category")
				if err != nil {
					return fmt.Errorf("parent category taxonomy not found: %w", err)
				}
				parentID = tt.TermTaxonomyID
			}
		}

		term := &model.Term{
			Name: name,
			Slug: slugFlag,
		}
		taxonomy := &model.TermTaxonomy{
			Taxonomy:    "category",
			Description: description,
			Parent:      parentID,
		}

		if err := terms.Create(ctx, term, taxonomy); err != nil {
			return err
		}

		if porcelain {
			fmt.Println(term.TermID)
			return nil
		}

		fmt.Printf("Success: Created category %d.\n", term.TermID)
		return nil
	},
}

var categoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		terms := repository.NewTermsRepository(db)

		result, err := terms.List(ctx, query.Query{
			Filters: []query.Filter{
				{Field: "taxonomy", Operator: query.Is, Value: "category"},
			},
			PerPage: 1000,
		})
		if err != nil {
			return err
		}

		columns := []column{
			{"ID", 4},
			{"Name", 25},
			{"Slug", 25},
			{"Count", 6},
		}

		var rows []map[string]any
		for _, c := range result.Items {
			rows = append(rows, map[string]any{
				"ID":    c.TermID,
				"Name":  c.Name,
				"Slug":  c.Slug,
				"Count": c.Count,
			})
		}

		if len(rows) == 0 {
			fmt.Println("No categories found.")
			return nil
		}

		formatList(format, "ID", columns, rows)
		return nil
	},
}

var categoryDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-slug>",
	Short: "Delete a category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		terms := repository.NewTermsRepository(db)

		var id int64
		if numID, slug := resolveIDOrString(args[0]); numID > 0 {
			id = numID
		} else {
			term, err := terms.GetBySlug(ctx, slug)
			if err != nil {
				return err
			}
			id = term.TermID
		}

		if _, err := terms.Delete(ctx, id); err != nil {
			return err
		}

		fmt.Printf("Success: Deleted category %d.\n", id)
		return nil
	},
}

func init() {
	categoryCreateCmd.Flags().String("slug", "", "category slug (generated from name if omitted)")
	categoryCreateCmd.Flags().String("description", "", "category description")
	categoryCreateCmd.Flags().String("parent", "", "parent category ID or slug")
	categoryCreateCmd.Flags().Bool("porcelain", false, "output just the new category ID")

	categoryListCmd.Flags().String("format", "table", "output format (table, json, csv, ids)")

	categoryCmd.AddCommand(categoryCreateCmd, categoryListCmd, categoryDeleteCmd)
	rootCmd.AddCommand(categoryCmd)
}
