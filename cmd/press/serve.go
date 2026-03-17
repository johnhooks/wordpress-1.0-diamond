package main

import (
	"fmt"

	"press/internal/server"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the blog server",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		srv, err := server.New(appConfig, db)
		if err != nil {
			return err
		}

		fmt.Printf("Press is running at %s\n", appConfig.BaseURL())
		return srv.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
