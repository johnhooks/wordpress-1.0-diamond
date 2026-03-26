package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"press/internal/permission"
	"press/internal/query"
	"press/internal/model"
	"press/internal/repository"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

var roleLevels = map[string]string{
	"administrators": "10",
	"editors":        "7",
	"authors":        "2",
	"subscribers":    "0",
}

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
		role, _ := cmd.Flags().GetString("role")
		userURL, _ := cmd.Flags().GetString("url")
		porcelain, _ := cmd.Flags().GetBool("porcelain")

		// Validate role
		level, ok := roleLevels[role]
		if !ok {
			return fmt.Errorf("invalid role %q (use: administrators, editors, authors, subscribers)", role)
		}

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
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		users := repository.NewUsersRepository(db)
		user := &model.User{
			UserLogin:   login,
			UserPass:    string(hashedPassword),
			UserEmail:   email,
			UserURL:     userURL,
			DisplayName: displayName,
		}

		if err := users.Create(ctx, user); err != nil {
			return err
		}

		// Set user level meta
		meta := repository.NewUserMeta(db)
		if err := meta.Set(ctx, user.ID, "wp_user_level", level); err != nil {
			return err
		}
		if err := meta.Set(ctx, user.ID, "nickname", login); err != nil {
			return err
		}

		// Create group membership tuple
		perms := permission.NewStore(db)
		if err := perms.CreateTuple(ctx, &permission.Tuple{
			SubjectType: "user",
			SubjectID:   fmt.Sprintf("%d", user.ID),
			Relation:    "member",
			ObjectType:  "group",
			ObjectID:    role,
			CreatedAt:   time.Now().UTC(),
			CreatedBy:   user.ID,
		}); err != nil {
			return fmt.Errorf("failed to set role: %w", err)
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

var userGetCmd = &cobra.Command{
	Use:   "get <id-or-login>",
	Short: "Get a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		users := repository.NewUsersRepository(db)

		var user *model.User
		if id, login := resolveIDOrString(args[0]); id > 0 {
			user, err = users.GetByID(ctx, id)
		} else {
			user, err = users.GetByLogin(ctx, login)
		}
		if err != nil {
			return err
		}

		// Get role
		perms := permission.NewStore(db)
		groups, _ := perms.GetUserGroups(ctx, user.ID)
		role := ""
		if len(groups) > 0 {
			role = groups[0]
		}

		formatSingle(format, []kv{
			{"ID", fmt.Sprintf("%d", user.ID)},
			{"Login", user.UserLogin},
			{"Email", user.UserEmail},
			{"Display Name", user.DisplayName},
			{"URL", user.UserURL},
			{"Role", role},
			{"Registered", user.UserRegistered.Format("2006-01-02 15:04:05")},
		})
		return nil
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		role, _ := cmd.Flags().GetString("role")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		users := repository.NewUsersRepository(db)
		result, err := users.List(ctx, query.Query{PerPage: 1000})
		if err != nil {
			return err
		}

		// If filtering by role, get members of that group
		var roleFilter map[int64]bool
		if role != "" {
			perms := permission.NewStore(db)
			// Get all users in this group by querying tuples
			var memberIDs []string
			dbx := db
			if err := dbx.SelectContext(ctx, &memberIDs,
				"SELECT subject_id FROM wp_tuples WHERE subject_type = 'user' AND relation = 'member' AND object_type = 'group' AND object_id = ?",
				role); err != nil {
				return err
			}
			roleFilter = make(map[int64]bool)
			for _, idStr := range memberIDs {
				if id, _ := resolveIDOrString(idStr); id > 0 {
					roleFilter[id] = true
				}
			}
			_ = perms // used above via db directly
		}

		columns := []column{
			{"ID", 4},
			{"Login", 20},
			{"Email", 30},
			{"Display Name", 20},
		}

		var rows []map[string]any
		for _, u := range result.Items {
			if roleFilter != nil && !roleFilter[u.ID] {
				continue
			}
			rows = append(rows, map[string]any{
				"ID":           u.ID,
				"Login":        u.UserLogin,
				"Email":        u.UserEmail,
				"Display Name": u.DisplayName,
			})
		}

		if len(rows) == 0 {
			fmt.Println("No users found.")
			return nil
		}

		formatList(format, "ID", columns, rows)
		return nil
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-login>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reassign, _ := cmd.Flags().GetInt64("reassign")

		db, err := openDB()
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		ctx := context.Background()
		users := repository.NewUsersRepository(db)

		// Resolve ID or login
		var id int64
		if numID, login := resolveIDOrString(args[0]); numID > 0 {
			id = numID
		} else {
			user, err := users.GetByLogin(ctx, login)
			if err != nil {
				return err
			}
			id = user.ID
		}

		if _, err := users.Delete(ctx, id, reassign); err != nil {
			return err
		}

		// Clean up permission tuples
		perms := permission.NewStore(db)
		if _, err := perms.DeleteTuplesForSubject(ctx, "user", fmt.Sprintf("%d", id)); err != nil {
			return fmt.Errorf("failed to clean up permission tuples: %w", err)
		}

		if reassign > 0 {
			fmt.Printf("Success: Removed user %d. Posts reassigned to user %d.\n", id, reassign)
		} else {
			fmt.Printf("Success: Removed user %d.\n", id)
		}
		return nil
	},
}

func init() {
	userCreateCmd.Flags().String("pass", "", "user password (generated if omitted)")
	userCreateCmd.Flags().String("display-name", "", "display name (defaults to login)")
	userCreateCmd.Flags().String("role", "subscribers", "user role (administrators, editors, authors, subscribers)")
	userCreateCmd.Flags().String("url", "", "user URL")
	userCreateCmd.Flags().Bool("porcelain", false, "output just the new user ID")

	userGetCmd.Flags().String("format", "table", "output format (table, json)")

	userListCmd.Flags().String("format", "table", "output format (table, json, csv, ids)")
	userListCmd.Flags().String("role", "", "filter by role")

	userDeleteCmd.Flags().Int64("reassign", 0, "reassign posts to this user ID")

	userCmd.AddCommand(userCreateCmd, userGetCmd, userListCmd, userDeleteCmd)
	rootCmd.AddCommand(userCmd)
}
