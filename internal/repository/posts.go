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
	"press/internal/slug"

	"github.com/jmoiron/sqlx"
)

var postFieldMap = map[string]string{
	"status": "p.post_status",
	"type":   "p.post_type",
	"author": "p.post_author",
}

var postSortMap = map[string]string{
	"date":  "p.post_date",
	"title": "p.post_title",
	"id":    "p.id",
}

var postSearchColumns = []string{"p.post_title", "p.post_content"}

type PostsRepository struct {
	db *sqlx.DB
}

func NewPostsRepository(db *sqlx.DB) *PostsRepository {
	return &PostsRepository{db: db}
}

func (r *PostsRepository) Create(ctx context.Context, post *model.Post) error {
	now := time.Now().UTC()
	if post.PostDate.IsZero() {
		post.PostDate = now
		post.PostDateGmt = now
	}
	post.PostModified = now
	post.PostModifiedGmt = now

	result, err := r.db.NamedExecContext(ctx, `
		INSERT INTO wp_posts (
			post_author, post_date, post_date_gmt, post_content, post_title,
			post_excerpt, post_status, comment_status, ping_status, post_password,
			post_name, to_ping, pinged, post_modified, post_modified_gmt,
			post_content_filtered, post_parent, guid, menu_order, post_type,
			post_mime_type, comment_count
		) VALUES (
			:post_author, :post_date, :post_date_gmt, :post_content, :post_title,
			:post_excerpt, :post_status, :comment_status, :ping_status, :post_password,
			:post_name, :to_ping, :pinged, :post_modified, :post_modified_gmt,
			:post_content_filtered, :post_parent, :guid, :menu_order, :post_type,
			:post_mime_type, :comment_count
		)`, post)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	post.ID = id
	return nil
}

func (r *PostsRepository) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	post := &model.Post{}
	err := r.db.GetContext(ctx, post, "SELECT * FROM wp_posts WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrPostNotFound, fmt.Sprintf("Post not found: %d", id))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return post, nil
}

func (r *PostsRepository) GetBySlug(ctx context.Context, s string) (*model.Post, error) {
	post := &model.Post{}
	err := r.db.GetContext(ctx, post, "SELECT * FROM wp_posts WHERE post_name = ?", s)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrPostNotFound, fmt.Sprintf("Post not found: %s", s))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return post, nil
}

func (r *PostsRepository) GetBySlugAndType(ctx context.Context, s, postType string) (*model.Post, error) {
	post := &model.Post{}
	err := r.db.GetContext(ctx, post, "SELECT * FROM wp_posts WHERE post_name = ? AND post_type = ?", s, postType)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrPostNotFound, fmt.Sprintf("Post not found: %s", s))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return post, nil
}

// EnsureUniqueSlug generates a unique post_name by appending -2, -3, etc. if needed.
func (r *PostsRepository) EnsureUniqueSlug(ctx context.Context, s string, excludeID int64) (string, error) {
	return slug.EnsureUnique(s, func(candidate string) (bool, error) {
		var count int
		err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM wp_posts WHERE post_name = ? AND id != ?", candidate, excludeID)
		return count > 0, err
	})
}

func (r *PostsRepository) List(ctx context.Context, q query.Query) (*query.Result[model.Post], error) {
	baseQuery := "SELECT p.* FROM wp_posts p"
	countQuery := "SELECT COUNT(*) FROM wp_posts p"
	args := []any{}
	wheres := []string{}

	// Special: category filter adds JOINs
	join := ""
	for _, f := range q.Filters {
		if f.Field == "category" && f.Operator == query.Is {
			join = `
				JOIN wp_term_relationships tr ON p.id = tr.object_id
				JOIN wp_term_taxonomy tt ON tr.term_taxonomy_id = tt.term_taxonomy_id`
			wheres = append(wheres, "tt.term_id = ? AND tt.taxonomy = 'category'")
			args = append(args, f.Value)
		}
	}
	baseQuery += join
	countQuery += join

	// Generic filters
	clauses, filterArgs := query.ApplyFilters(q.Filters, postFieldMap)
	wheres = append(wheres, clauses...)
	args = append(args, filterArgs...)

	// Search
	if searchClause, searchArgs := query.ApplySearch(q.Search, postSearchColumns); searchClause != "" {
		wheres = append(wheres, searchClause)
		args = append(args, searchArgs...)
	}

	whereStr := ""
	if len(wheres) > 0 {
		whereStr = " WHERE " + strings.Join(wheres, " AND ")
	}

	// Count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery+whereStr, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}

	// Sort
	orderBy := query.ApplySort(q.Sort, postSortMap, "p.post_date DESC")
	baseQuery += whereStr + " ORDER BY " + orderBy

	// Pagination
	perPage := q.GetPerPage()
	offset := q.Offset()
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)

	var posts []model.Post
	if err := r.db.SelectContext(ctx, &posts, baseQuery, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if posts == nil {
		posts = []model.Post{}
	}

	return query.NewResult(posts, total, q.GetPage(), perPage), nil
}

func (r *PostsRepository) Update(ctx context.Context, post *model.Post) (*model.Post, error) {
	now := time.Now().UTC()
	post.PostModified = now
	post.PostModifiedGmt = now

	result, err := r.db.NamedExecContext(ctx, `
		UPDATE wp_posts SET
			post_author = :post_author, post_date = :post_date, post_date_gmt = :post_date_gmt,
			post_content = :post_content, post_title = :post_title, post_excerpt = :post_excerpt,
			post_status = :post_status, comment_status = :comment_status, ping_status = :ping_status,
			post_password = :post_password, post_name = :post_name, to_ping = :to_ping,
			pinged = :pinged, post_modified = :post_modified, post_modified_gmt = :post_modified_gmt,
			post_content_filtered = :post_content_filtered, post_parent = :post_parent,
			guid = :guid, menu_order = :menu_order, post_type = :post_type,
			post_mime_type = :post_mime_type, comment_count = :comment_count
		WHERE id = :id`, post)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, errors.NotFound(errors.ErrPostNotFound, fmt.Sprintf("Post not found: %d", post.ID))
	}
	return post, nil
}

func (r *PostsRepository) Delete(ctx context.Context, id int64) (*model.Post, error) {
	post, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx, "DELETE FROM wp_posts WHERE id = ?", id)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}

	// Clean up related data
	r.db.ExecContext(ctx, "DELETE FROM wp_postmeta WHERE post_id = ?", id)
	r.db.ExecContext(ctx, "DELETE FROM wp_comments WHERE comment_post_id = ?", id)
	r.db.ExecContext(ctx, "DELETE FROM wp_term_relationships WHERE object_id = ?", id)
	return post, nil
}
