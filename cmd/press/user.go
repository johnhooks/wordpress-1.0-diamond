package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"press/internal/model"
	"press/internal/query"
	"press/internal/repository"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userCreateCmd = &cobra.Command{
	Use:   "create <login> <email>",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		login := args[0]
		email := args[1]

		password, _ := cmd.Flags().GetString("pass")
		displayName, _ := cmd.Flags().GetString("display-name")
		porcelain, _ := cmd.Flags().GetBool("porcelain")

		// Generate password if not provided
		generatedPass := false
		if password == "" {
			bytes := make([]byte, 12)
			if _, err := rand.Read(bytes); err != nil {
				return fmt.Errorf("failed to generate password: %w", err)
			}
			password = hex.EncodeToString(bytes)
			generatedPass = true
		}

		if displayName == "" {
			displayName = login
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		ctx := context.Background()
		users := repository.NewUsersRepository(db)
		user := &model.User{
			UserLogin:   login,
			UserPass:    string(hashedPassword),
			UserEmail:   email,
			DisplayName: displayName,
		}

		if err := users.Create(ctx, user); err != nil {
			return err
		}

		// Set default user level
		meta := repository.NewUserMeta(db)
		if err := meta.Set(ctx, user.ID, "wp_user_level", "0"); err != nil {
			return err
		}

		if porcelain {
			fmt.Println(user.ID)
			return nil
		}

		fmt.Printf("Success: Created user %d.\n", user.ID)
		if generatedPass {
			fmt.Printf("Password: %s\n", password)
		}
		return nil
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		ctx := context.Background()
		users := repository.NewUsersRepository(db)
		result, err := users.List(ctx, query.Query{PerPage: 1000})
		if err != nil {
			return err
		}

		if len(result.Items) == 0 {
			fmt.Println("No users found.")
			return nil
		}

		fmt.Printf("%-4s %-20s %-30s %s\n", "ID", "Login", "Email", "Display Name")
		fmt.Println(strings.Repeat("-", 80))
		for _, u := range result.Items {
			fmt.Printf("%-4d %-20s %-30s %s\n", u.ID, u.UserLogin, u.UserEmail, u.DisplayName)
		}
		return nil
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a user by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var id int64
		if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
			return fmt.Errorf("invalid user ID: %s", args[0])
		}

		reassign, _ := cmd.Flags().GetInt64("reassign")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		ctx := context.Background()
		users := repository.NewUsersRepository(db)
		if _, err := users.Delete(ctx, id, reassign); err != nil {
			return err
		}

		if reassign > 0 {
			fmt.Printf("Success: Removed user %d. Posts and links reassigned to user %d.\n", id, reassign)
		} else {
			fmt.Printf("Success: Removed user %d.\n", id)
		}
		return nil
	},
}

func init() {
	userCreateCmd.Flags().String("pass", "", "user password (generated if omitted)")
	userCreateCmd.Flags().String("display-name", "", "display name (defaults to login)")
	userCreateCmd.Flags().Bool("porcelain", false, "output just the new user ID")

	userDeleteCmd.Flags().Int64("reassign", 0, "reassign posts and links to this user ID")

	userCmd.AddCommand(userCreateCmd, userListCmd, userDeleteCmd)
	rootCmd.AddCommand(userCmd)
}
