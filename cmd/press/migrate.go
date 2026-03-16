package main

import (
	"fmt"
	"os"

	"press/internal/database"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetBaseFS(database.MigrationsFS)
		if err := goose.SetDialect("sqlite"); err != nil {
			return err
		}

		if err := goose.Up(db.DB, "migrations"); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		fmt.Println("Migrations complete.")
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetBaseFS(database.MigrationsFS)
		if err := goose.SetDialect("sqlite"); err != nil {
			return err
		}

		if err := goose.Down(db.DB, "migrations"); err != nil {
			return fmt.Errorf("failed to rollback: %w", err)
		}
		fmt.Println("Rollback complete.")
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetBaseFS(database.MigrationsFS)
		if err := goose.SetDialect("sqlite"); err != nil {
			return err
		}

		return goose.Status(db.DB, "migrations")
	},
}

var migrateFreshCmd = &cobra.Command{
	Use:   "fresh",
	Short: "Drop database and re-run all migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Delete the database file
		if err := os.Remove(appConfig.DBPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove database: %w", err)
		}
		fmt.Println("Database dropped.")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetBaseFS(database.MigrationsFS)
		if err := goose.SetDialect("sqlite"); err != nil {
			return err
		}

		if err := goose.Up(db.DB, "migrations"); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		fmt.Println("Fresh migration complete.")
		return nil
	},
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		if err := goose.SetDialect("sqlite"); err != nil {
			return err
		}

		if err := goose.Create(db.DB, "internal/database/migrations", args[0], "sql"); err != nil {
			return fmt.Errorf("failed to create migration: %w", err)
		}
		fmt.Printf("Created migration: %s\n", args[0])
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateFreshCmd, migrateCreateCmd)
	rootCmd.AddCommand(migrateCmd)
}
