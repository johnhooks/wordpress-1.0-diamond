package main

import (
	"fmt"
	"os"
	"path/filepath"

	"press/internal/importmap"

	"github.com/spf13/cobra"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Build the vendor asset manifest",
	Long:  `Hashes vendor JS files and writes vendor/manifest.json.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		publicDir := appConfig.PublicDir
		vendorDir := "vendor"
		urlPrefix := "/static/vendor/"

		m, err := importmap.Build(os.DirFS(publicDir), vendorDir, urlPrefix)
		if err != nil {
			return err
		}

		manifestPath := filepath.Join(publicDir, vendorDir, "manifest.json")
		if err := m.WriteManifest(manifestPath); err != nil {
			return err
		}

		fmt.Printf("Wrote %s (%d modules)\n", manifestPath, len(m.Imports))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(manifestCmd)
}
