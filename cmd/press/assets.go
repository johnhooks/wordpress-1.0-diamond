package main

import (
	"fmt"
	"os"

	"press/assets"

	"github.com/spf13/cobra"
)

var assetsCmd = &cobra.Command{
	Use:   "assets",
	Short: "Fetch vendor dependencies and build the asset manifest",
	Long: `Writes engine JS, fetches vendor packages from esm.sh, copies
theme JS (if present), hashes all asset files, and writes the manifest.

Existing vendor files are skipped. Use --fresh to re-fetch everything.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fresh, _ := cmd.Flags().GetBool("fresh")
		publicDir := appConfig.PublicDir

		pipe := assets.Default()
		pipe.ThemeDir = appConfig.ThemeDir

		if fresh {
			if err := assets.CleanVendor(os.Stdout, publicDir); err != nil {
				return err
			}
		}

		m, err := pipe.Run(os.Stdout, publicDir)
		if err != nil {
			return err
		}

		fmt.Printf("Manifested %d modules\n", len(m.Imports))
		return nil
	},
}

func init() {
	assetsCmd.Flags().Bool("fresh", false, "Delete vendor JS files and re-fetch")
	rootCmd.AddCommand(assetsCmd)
}
