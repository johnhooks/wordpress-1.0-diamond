package main

import (
	"fmt"
	"os"

	"press/internal/config"
	"press/internal/db"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
)

var (
	envFile   string
	appConfig *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "press",
	Short: "Press - A blog engine continuing WordPress 1.0",
	Long:  `Press is what happens when you fork WordPress 1.0 and keep evolving it as a blogging engine.`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&envFile, "env", "", "path to .env file")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(envFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		appConfig = cfg
		return nil
	}
}

func openDB() (*sqlx.DB, error) {
	return db.Connect(appConfig.DBPath)
}
