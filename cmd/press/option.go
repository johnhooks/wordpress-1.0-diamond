package main

import (
	"context"
	"fmt"
	"strings"

	"press/internal/repository"

	"github.com/spf13/cobra"
)

var optionCmd = &cobra.Command{
	Use:   "option",
	Short: "Manage options",
}

var optionGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get an option value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		options := repository.NewOptionsRepository(db)
		value, err := options.GetValue(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Println(value)
		return nil
	},
}

var optionSetCmd = &cobra.Command{
	Use:   "set [name] [value]",
	Short: "Set an option value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		options := repository.NewOptionsRepository(db)
		if err := options.Set(ctx, args[0], args[1], "yes"); err != nil {
			return err
		}

		fmt.Printf("Option '%s' set.\n", args[0])
		return nil
	},
}

var optionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all options",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		options := repository.NewOptionsRepository(db)
		list, err := options.List(ctx)
		if err != nil {
			return err
		}

		if len(list) == 0 {
			fmt.Println("No options found.")
			return nil
		}

		fmt.Printf("%-30s %-10s %s\n", "Name", "Autoload", "Value")
		fmt.Println(strings.Repeat("-", 80))
		for _, o := range list {
			value := o.OptionValue
			if len(value) > 38 {
				value = value[:37] + "…"
			}
			fmt.Printf("%-30s %-10s %s\n", o.OptionName, o.Autoload, value)
		}
		return nil
	},
}

func init() {
	optionCmd.AddCommand(optionGetCmd, optionSetCmd, optionListCmd)
	rootCmd.AddCommand(optionCmd)
}
