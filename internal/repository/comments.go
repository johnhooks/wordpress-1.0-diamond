package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"press/internal/errors"
	"press/internal/model"
	"press/internal/query"

	"github.com/jmoiron/sqlx"
)

var commentFieldMap = map[string]string{
	"post":     "comment_post_id",
	"approved": "comment_approved",
	"type":     "comment_type",
	"email":    "comment_author_email",
	"parent":   "comment_parent",
}

var commentSortMap = map[string]string{
	"date": "comment_date",
	"id":   "comment_id",
}

var commentSearchColumns = []string{"comment_content", "comment_author"}

type CommentsRepository struct {
	db *sqlx.DB
}

func NewCommentsRepository(db *sqlx.DB) *CommentsRepository {
	return &CommentsRepository{db: db}
}

func (r *CommentsRepository) Create(ctx context.Context, comment *model.Comment) error {
	now := time.Now().UTC()
	if comment.CommentDate.IsZero() {
		comment.CommentDate = now
		comment.CommentDateGmt = now
	}

	result, err := r.db.NamedExecContext(ctx, `
		INSERT INTO wp_comments (
			comment_post_id, comment_author, comment_author_email, comment_author_url,
			comment_author_ip, comment_date, comment_date_gmt, comment_content,
			comment_karma, comment_approved, comment_agent, comment_type,
			comment_parent, user_id
		) VALUES (
			:comment_post_id, :comment_author, :comment_author_email, :comment_author_url,
			:comment_author_ip, :comment_date, :comment_date_gmt, :comment_content,
			:comment_karma, :comment_approved, :comment_agent, :comment_type,
			:comment_parent, :user_id
		)`, comment)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	comment.CommentID = id

	if comment.CommentApproved == "1" {
		if _, err := r.db.ExecContext(ctx, "UPDATE wp_posts SET comment_count = comment_count + 1 WHERE id = ?", comment.CommentPostID); err != nil {
			return errors.Internal(err, errors.ErrQueryFailed)
		}
	}
	return nil
}

func (r *CommentsRepository) GetByID(ctx context.Context, id int64) (*model.Comment, error) {
	comment := &model.Comment{}
	err := r.db.GetContext(ctx, comment, "SELECT * FROM wp_comments WHERE comment_id = ?", id)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrCommentNotFound, fmt.Sprintf("Comment not found: %d", id))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return comment, nil
}

func (r *CommentsRepository) GetByPostID(ctx context.Context, postID int64) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.SelectContext(ctx, &comments, "SELECT * FROM wp_comments WHERE comment_post_id = ? ORDER BY comment_date ASC", postID)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return comments, nil
}

func (r *CommentsRepository) List(ctx context.Context, q query.Query) (*query.Result[model.Comment], error) {
	baseQuery := "SELECT * FROM wp_comments"
	countQuery := "SELECT COUNT(*) FROM wp_comments"
	args := []any{}
	wheres := []string{}

	clauses, filterArgs := query.ApplyFilters(q.Filters, commentFieldMap)
	wheres = append(wheres, clauses...)
	args = append(args, filterArgs...)

	if searchClause, searchArgs := query.ApplySearch(q.Search, commentSearchColumns); searchClause != "" {
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

	orderBy := query.ApplySort(q.Sort, commentSortMap, "comment_date DESC")
	baseQuery += whereStr + " ORDER BY " + orderBy

	perPage := q.GetPerPage()
	offset := q.Offset()
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)

	var comments []model.Comment
	if err := r.db.SelectContext(ctx, &comments, baseQuery, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if comments == nil {
		comments = []model.Comment{}
	}

	return query.NewResult(comments, total, q.GetPage(), perPage), nil
}

func (r *CommentsRepository) Update(ctx context.Context, comment *model.Comment) (*model.Comment, error) {
	old, err := r.GetByID(ctx, comment.CommentID)
	if err != nil {
		return nil, err
	}

	result, err := r.db.NamedExecContext(ctx, `
		UPDATE wp_comments SET
			comment_post_id = :comment_post_id, comment_author = :comment_author,
			comment_author_email = :comment_author_email, comment_author_url = :comment_author_url,
			comment_author_ip = :comment_author_ip, comment_content = :comment_content,
			comment_karma = :comment_karma, comment_approved = :comment_approved,
			comment_agent = :comment_agent, comment_type = :comment_type,
			comment_parent = :comment_parent, user_id = :user_id
		WHERE comment_id = :comment_id`, comment)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, errors.NotFound(errors.ErrCommentNotFound, fmt.Sprintf("Comment not found: %d", comment.CommentID))
	}

	if old.CommentApproved != comment.CommentApproved {
		if comment.CommentApproved == "1" {
			if _, err := r.db.ExecContext(ctx, "UPDATE wp_posts SET comment_count = comment_count + 1 WHERE id = ?", comment.CommentPostID); err != nil {
				return nil, errors.Internal(err, errors.ErrQueryFailed)
			}
		} else if old.CommentApproved == "1" {
			if _, err := r.db.ExecContext(ctx, "UPDATE wp_posts SET comment_count = comment_count - 1 WHERE id = ? AND comment_count > 0", comment.CommentPostID); err != nil {
				return nil, errors.Internal(err, errors.ErrQueryFailed)
			}
		}
	}
	return comment, nil
}

func (r *CommentsRepository) Delete(ctx context.Context, id int64) (*model.Comment, error) {
	comment, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	defer func() { _ = tx.Rollback() }()

	// Re-parent child comments
	if _, err := tx.ExecContext(ctx, "UPDATE wp_comments SET comment_parent = ? WHERE comment_parent = ?", comment.CommentParent, id); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}

	// Delete comment metadata
	if _, err := tx.ExecContext(ctx, "DELETE FROM wp_commentmeta WHERE comment_id = ?", id); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM wp_comments WHERE comment_id = ?", id); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}

	if comment.CommentApproved == "1" {
		if _, err := tx.ExecContext(ctx, "UPDATE wp_posts SET comment_count = comment_count - 1 WHERE id = ? AND comment_count > 0", comment.CommentPostID); err != nil {
			return nil, errors.Internal(err, errors.ErrQueryFailed)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return comment, nil
}

func (r *CommentsRepository) Approve(ctx context.Context, id int64) error {
	comment, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if comment.CommentApproved == "1" {
		return nil
	}

	_, err = r.db.ExecContext(ctx, "UPDATE wp_comments SET comment_approved = '1' WHERE comment_id = ?", id)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	if _, err := r.db.ExecContext(ctx, "UPDATE wp_posts SET comment_count = comment_count + 1 WHERE id = ?", comment.CommentPostID); err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}

func (r *CommentsRepository) Unapprove(ctx context.Context, id int64) error {
	comment, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if comment.CommentApproved != "1" {
		return nil
	}

	_, err = r.db.ExecContext(ctx, "UPDATE wp_comments SET comment_approved = '0' WHERE comment_id = ?", id)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	if _, err := r.db.ExecContext(ctx, "UPDATE wp_posts SET comment_count = comment_count - 1 WHERE id = ? AND comment_count > 0", comment.CommentPostID); err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}
