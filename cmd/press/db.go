package main

import (
	"fmt"
	"os"

	"press/internal/database"
	"press/internal/database/seed"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
}

var dbSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with default data",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		return seed.Run(db)
	},
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Drop database, re-run migrations, and seed",
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
		fmt.Println("Migrations complete.")

		return seed.Run(db)
	},
}

func init() {
	dbCmd.AddCommand(dbSeedCmd, dbResetCmd)
	rootCmd.AddCommand(dbCmd)
}
