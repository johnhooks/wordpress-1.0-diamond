package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the blog server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("press serve is not yet implemented.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
