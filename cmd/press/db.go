package main

import (
	"fmt"
	"os"

	"press/internal/database"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Drop database and re-run migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.Remove(appConfig.DBPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove database: %w", err)
		}
		fmt.Println("Database dropped.")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

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

func init() {
	dbCmd.AddCommand(dbResetCmd)
	rootCmd.AddCommand(dbCmd)
}
