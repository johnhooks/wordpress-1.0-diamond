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

var linkFieldMap = map[string]string{
	"visible": "l.link_visible",
}

var linkSortMap = map[string]string{
	"name": "l.link_name",
	"id":   "l.link_id",
}

var linkSearchColumns = []string{"l.link_name", "l.link_url"}

type LinksRepository struct {
	db *sqlx.DB
}

func NewLinksRepository(db *sqlx.DB) *LinksRepository {
	return &LinksRepository{db: db}
}

func (r *LinksRepository) Create(ctx context.Context, link *model.Link) error {
	result, err := r.db.NamedExecContext(ctx, `
		INSERT INTO wp_links (
			link_url, link_name, link_image, link_target, link_description,
			link_visible, link_owner, link_rating, link_updated, link_rel,
			link_notes, link_rss
		) VALUES (
			:link_url, :link_name, :link_image, :link_target, :link_description,
			:link_visible, :link_owner, :link_rating, :link_updated, :link_rel,
			:link_notes, :link_rss
		)`, link)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	link.LinkID = id
	return nil
}

func (r *LinksRepository) GetByID(ctx context.Context, id int64) (*model.Link, error) {
	link := &model.Link{}
	err := r.db.GetContext(ctx, link, "SELECT * FROM wp_links WHERE link_id = ?", id)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrLinkNotFound, fmt.Sprintf("Link not found: %d", id))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return link, nil
}

func (r *LinksRepository) List(ctx context.Context, q query.Query) (*query.Result[model.Link], error) {
	baseQuery := "SELECT l.* FROM wp_links l"
	countQuery := "SELECT COUNT(*) FROM wp_links l"
	args := []any{}
	wheres := []string{}

	// Special: category filter adds JOINs
	join := ""
	for _, f := range q.Filters {
		if f.Field == "category" && f.Operator == query.Is {
			join = `
				JOIN wp_term_relationships tr ON l.link_id = tr.object_id
				JOIN wp_term_taxonomy tt ON tr.term_taxonomy_id = tt.term_taxonomy_id`
			wheres = append(wheres, "tt.term_id = ? AND tt.taxonomy = 'link_category'")
			args = append(args, f.Value)
		}
	}
	baseQuery += join
	countQuery += join

	clauses, filterArgs := query.ApplyFilters(q.Filters, linkFieldMap)
	wheres = append(wheres, clauses...)
	args = append(args, filterArgs...)

	if searchClause, searchArgs := query.ApplySearch(q.Search, linkSearchColumns); searchClause != "" {
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

	orderBy := query.ApplySort(q.Sort, linkSortMap, "l.link_name ASC")
	baseQuery += whereStr + " ORDER BY " + orderBy

	perPage := q.GetPerPage()
	offset := q.Offset()
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)

	var links []model.Link
	if err := r.db.SelectContext(ctx, &links, baseQuery, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if links == nil {
		links = []model.Link{}
	}

	return query.NewResult(links, total, q.GetPage(), perPage), nil
}

func (r *LinksRepository) Update(ctx context.Context, link *model.Link) (*model.Link, error) {
	link.LinkUpdated = time.Now().UTC()

	result, err := r.db.NamedExecContext(ctx, `
		UPDATE wp_links SET
			link_url = :link_url, link_name = :link_name, link_image = :link_image,
			link_target = :link_target, link_description = :link_description,
			link_visible = :link_visible, link_owner = :link_owner, link_rating = :link_rating,
			link_updated = :link_updated, link_rel = :link_rel, link_notes = :link_notes,
			link_rss = :link_rss
		WHERE link_id = :link_id`, link)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, errors.NotFound(errors.ErrLinkNotFound, fmt.Sprintf("Link not found: %d", link.LinkID))
	}
	return link, nil
}

func (r *LinksRepository) Delete(ctx context.Context, id int64) (*model.Link, error) {
	link, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx, "DELETE FROM wp_links WHERE link_id = ?", id)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM wp_term_relationships WHERE object_id = ?", id); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return link, nil
}
