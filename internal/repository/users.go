package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"press/internal/errors"
	"press/internal/model"
	"press/internal/query"
	"press/internal/slug"

	"github.com/jmoiron/sqlx"
)

var userSortMap = map[string]string{
	"login": "user_login",
	"email": "user_email",
	"name":  "display_name",
	"id":    "id",
}

var userSearchColumns = []string{"user_login", "user_email", "display_name"}

type UsersRepository struct {
	db *sqlx.DB
}

func NewUsersRepository(db *sqlx.DB) *UsersRepository {
	return &UsersRepository{db: db}
}

func (r *UsersRepository) Create(ctx context.Context, user *model.User) error {
	if user.UserRegistered.IsZero() {
		user.UserRegistered = time.Now().UTC()
	}

	// Auto-generate nicename from login if empty
	if user.UserNicename == "" {
		nicename := slug.Generate(user.UserLogin)
		unique, err := slug.EnsureUnique(nicename, func(candidate string) (bool, error) {
			var count int
			err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM wp_users WHERE user_nicename = ?", candidate)
			return count > 0, err
		})
		if err != nil {
			return errors.Internal(err, errors.ErrQueryFailed)
		}
		user.UserNicename = unique
	}

	if user.DisplayName == "" {
		user.DisplayName = user.UserLogin
	}

	// Check for duplicate login
	var existing model.User
	err := r.db.GetContext(ctx, &existing, "SELECT id FROM wp_users WHERE user_login = ?", user.UserLogin)
	if err == nil {
		return errors.New(errors.ErrUserLoginExists, fmt.Sprintf("User login already exists: %s", user.UserLogin), http.StatusConflict)
	} else if err != sql.ErrNoRows {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	// Check for duplicate email
	err = r.db.GetContext(ctx, &existing, "SELECT id FROM wp_users WHERE user_email = ?", user.UserEmail)
	if err == nil {
		return errors.New(errors.ErrUserEmailExists, fmt.Sprintf("User email already exists: %s", user.UserEmail), http.StatusConflict)
	} else if err != sql.ErrNoRows {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	result, err := r.db.NamedExecContext(ctx, `
		INSERT INTO wp_users (
			user_login, user_pass, user_nicename, user_email, user_url,
			user_registered, user_activation_key, user_status, display_name
		) VALUES (
			:user_login, :user_pass, :user_nicename, :user_email, :user_url,
			:user_registered, :user_activation_key, :user_status, :display_name
		)`, user)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	user.ID = id
	return nil
}

func (r *UsersRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	user := &model.User{}
	err := r.db.GetContext(ctx, user, "SELECT * FROM wp_users WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrUserNotFound, fmt.Sprintf("User not found: %d", id))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return user, nil
}

func (r *UsersRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	user := &model.User{}
	err := r.db.GetContext(ctx, user, "SELECT * FROM wp_users WHERE user_login = ?", login)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrUserNotFound, fmt.Sprintf("User not found: %s", login))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return user, nil
}

func (r *UsersRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.GetContext(ctx, user, "SELECT * FROM wp_users WHERE user_email = ?", email)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrUserNotFound, fmt.Sprintf("User not found: %s", email))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return user, nil
}

func (r *UsersRepository) List(ctx context.Context, q query.Query) (*query.Result[model.User], error) {
	baseQuery := "SELECT * FROM wp_users"
	countQuery := "SELECT COUNT(*) FROM wp_users"
	args := []any{}
	wheres := []string{}

	if searchClause, searchArgs := query.ApplySearch(q.Search, userSearchColumns); searchClause != "" {
		wheres = append(wheres, searchClause)
		args = append(args, searchArgs...)
	}

	whereStr := ""
	if len(wheres) > 0 {
		whereStr = " WHERE " + strings.Join(wheres, " AND ")
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery+whereStr, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}

	orderBy := query.ApplySort(q.Sort, userSortMap, "user_login ASC")
	baseQuery += whereStr + " ORDER BY " + orderBy

	perPage := q.GetPerPage()
	offset := q.Offset()
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)

	var users []model.User
	if err := r.db.SelectContext(ctx, &users, baseQuery, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if users == nil {
		users = []model.User{}
	}

	return query.NewResult(users, total, q.GetPage(), perPage), nil
}

// UsernameExists returns the user ID if the login exists, 0 if not.
func (r *UsersRepository) UsernameExists(ctx context.Context, login string) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, "SELECT id FROM wp_users WHERE user_login = ?", login)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	return id, nil
}

// EmailExists returns the user ID if the email exists, 0 if not.
func (r *UsersRepository) EmailExists(ctx context.Context, email string) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, "SELECT id FROM wp_users WHERE user_email = ?", email)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	return id, nil
}

func (r *UsersRepository) Update(ctx context.Context, user *model.User) (*model.User, error) {
	result, err := r.db.NamedExecContext(ctx, `
		UPDATE wp_users SET
			user_login = :user_login, user_pass = :user_pass, user_nicename = :user_nicename,
			user_email = :user_email, user_url = :user_url, user_registered = :user_registered,
			user_activation_key = :user_activation_key, user_status = :user_status,
			display_name = :display_name
		WHERE id = :id`, user)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, errors.NotFound(errors.ErrUserNotFound, fmt.Sprintf("User not found: %d", user.ID))
	}
	return user, nil
}

// Delete removes a user. If reassignTo is non-zero, the user's posts and
// links are reassigned to that user. If zero, the user's posts (and their
// comments/term relationships) and links are deleted.
func (r *UsersRepository) Delete(ctx context.Context, id int64, reassignTo int64) (*model.User, error) {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	defer tx.Rollback()

	if reassignTo > 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE wp_posts SET post_author = ? WHERE post_author = ?", reassignTo, id); err != nil {
			return nil, errors.Internal(err, errors.ErrQueryFailed)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE wp_links SET link_owner = ? WHERE link_owner = ?", reassignTo, id); err != nil {
			return nil, errors.Internal(err, errors.ErrQueryFailed)
		}
	} else {
		var postIDs []int64
		if err := tx.SelectContext(ctx, &postIDs, "SELECT id FROM wp_posts WHERE post_author = ?", id); err != nil {
			return nil, errors.Internal(err, errors.ErrQueryFailed)
		}
		for _, pid := range postIDs {
			if _, err := tx.ExecContext(ctx, "DELETE FROM wp_postmeta WHERE post_id = ?", pid); err != nil {
				return nil, errors.Internal(err, errors.ErrQueryFailed)
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM wp_comments WHERE comment_post_id = ?", pid); err != nil {
				return nil, errors.Internal(err, errors.ErrQueryFailed)
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM wp_term_relationships WHERE object_id = ?", pid); err != nil {
				return nil, errors.Internal(err, errors.ErrQueryFailed)
			}
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM wp_posts WHERE post_author = ?", id); err != nil {
			return nil, errors.Internal(err, errors.ErrQueryFailed)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM wp_links WHERE link_owner = ?", id); err != nil {
			return nil, errors.Internal(err, errors.ErrQueryFailed)
		}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM wp_usermeta WHERE user_id = ?", id); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM wp_users WHERE id = ?", id); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return user, nil
}

func (r *UsersRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM wp_users"); err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	return count, nil
}
