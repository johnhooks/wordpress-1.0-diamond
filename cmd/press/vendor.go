package main

import (
	"encoding/json"
	"fmt"
	"os"

	"press/internal/importmap"

	"github.com/spf13/cobra"
)

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Short: "Fetch vendor dependencies from esm.sh",
	Long: `Reads pins.json, fetches each package and its transitive
dependencies from esm.sh, saves them to public/vendor/, and builds the manifest.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		publicDir := appConfig.PublicDir
		vendorDir := "vendor"
		urlPrefix := "/static/vendor/"
		pinsPath := "pins.json"

		data, err := os.ReadFile(pinsPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", pinsPath, err)
		}

		var pins map[string]string
		if err := json.Unmarshal(data, &pins); err != nil {
			return fmt.Errorf("invalid %s: %w", pinsPath, err)
		}

		m, err := importmap.Vendor(os.Stdout, publicDir, vendorDir, urlPrefix, pinsPath, pins)
		if err != nil {
			return err
		}

		fmt.Printf("Vendored %d modules\n", len(m.Imports))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vendorCmd)
}
