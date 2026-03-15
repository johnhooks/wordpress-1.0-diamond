package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"press/internal/errors"
	"press/internal/model"
	"press/internal/query"
	"press/internal/slug"

	"github.com/jmoiron/sqlx"
)

var termFieldMap = map[string]string{
	"taxonomy": "tt.taxonomy",
	"parent":   "tt.parent",
}

var termSortMap = map[string]string{
	"name": "t.name",
	"id":   "t.term_id",
}

var termSearchColumns = []string{"t.name"}

type TermsRepository struct {
	db *sqlx.DB
}

func NewTermsRepository(db *sqlx.DB) *TermsRepository {
	return &TermsRepository{db: db}
}

func (r *TermsRepository) Create(ctx context.Context, term *model.Term, taxonomy *model.TermTaxonomy) error {
	if term.Slug == "" {
		generated := slug.Generate(term.Name)
		if generated == "" {
			generated = "term"
		}
		unique, err := slug.EnsureUnique(generated, func(candidate string) (bool, error) {
			var count int
			err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM wp_terms WHERE slug = ?", candidate)
			return count > 0, err
		})
		if err != nil {
			return errors.Internal(err, errors.ErrQueryFailed)
		}
		term.Slug = unique
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	defer tx.Rollback()

	result, err := tx.NamedExec(`
		INSERT INTO wp_terms (name, slug, term_group)
		VALUES (:name, :slug, :term_group)`, term)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	term.TermID = id
	taxonomy.TermID = id

	result, err = tx.NamedExec(`
		INSERT INTO wp_term_taxonomy (term_id, taxonomy, description, parent, count)
		VALUES (:term_id, :taxonomy, :description, :parent, :count)`, taxonomy)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	ttID, err := result.LastInsertId()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	taxonomy.TermTaxonomyID = ttID

	if err := tx.Commit(); err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}

func (r *TermsRepository) GetByID(ctx context.Context, id int64) (*model.Term, error) {
	term := &model.Term{}
	err := r.db.GetContext(ctx, term, "SELECT * FROM wp_terms WHERE term_id = ?", id)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrTermNotFound, fmt.Sprintf("Term not found: %d", id))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return term, nil
}

func (r *TermsRepository) GetBySlug(ctx context.Context, s string) (*model.Term, error) {
	term := &model.Term{}
	err := r.db.GetContext(ctx, term, "SELECT * FROM wp_terms WHERE slug = ?", s)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrTermNotFound, fmt.Sprintf("Term not found: %s", s))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return term, nil
}

// GetByName looks up a term by exact name within a taxonomy.
func (r *TermsRepository) GetByName(ctx context.Context, name, taxonomy string) (*model.Term, error) {
	term := &model.Term{}
	err := r.db.GetContext(ctx, term, "SELECT t.* FROM wp_terms t JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id WHERE t.name = ? AND tt.taxonomy = ?", name, taxonomy)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrTermNotFound, fmt.Sprintf("Term not found: %s", name))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return term, nil
}

func (r *TermsRepository) GetTaxonomy(ctx context.Context, termID int64, taxonomy string) (*model.TermTaxonomy, error) {
	tt := &model.TermTaxonomy{}
	err := r.db.GetContext(ctx, tt, "SELECT * FROM wp_term_taxonomy WHERE term_id = ? AND taxonomy = ?", termID, taxonomy)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrTermNotFound, "Term taxonomy not found")
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return tt, nil
}

func (r *TermsRepository) List(ctx context.Context, q query.Query) (*query.Result[model.Category], error) {
	selectClause := `SELECT t.term_id, t.name, t.slug, t.term_group,
		tt.term_taxonomy_id, tt.description, tt.parent, tt.count
		FROM wp_terms t
		JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id`
	countQuery := "SELECT COUNT(*) FROM wp_terms t JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id"
	args := []any{}
	wheres := []string{}

	clauses, filterArgs := query.ApplyFilters(q.Filters, termFieldMap)
	wheres = append(wheres, clauses...)
	args = append(args, filterArgs...)

	// Special: hide_empty filter
	for _, f := range q.Filters {
		if f.Field == "hide_empty" && f.Operator == query.Is {
			if v, ok := f.Value.(bool); ok && v {
				wheres = append(wheres, "tt.count > 0")
			}
		}
	}

	if searchClause, searchArgs := query.ApplySearch(q.Search, termSearchColumns); searchClause != "" {
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

	orderBy := query.ApplySort(q.Sort, termSortMap, "t.name ASC")
	selectClause += whereStr + " ORDER BY " + orderBy

	perPage := q.GetPerPage()
	offset := q.Offset()
	selectClause += " LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)

	var categories []model.Category
	if err := r.db.SelectContext(ctx, &categories, selectClause, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if categories == nil {
		categories = []model.Category{}
	}

	return query.NewResult(categories, total, q.GetPage(), perPage), nil
}

func (r *TermsRepository) Update(ctx context.Context, term *model.Term) (*model.Term, error) {
	result, err := r.db.NamedExecContext(ctx, `
		UPDATE wp_terms SET name = :name, slug = :slug, term_group = :term_group
		WHERE term_id = :term_id`, term)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, errors.NotFound(errors.ErrTermNotFound, fmt.Sprintf("Term not found: %d", term.TermID))
	}
	return term, nil
}

func (r *TermsRepository) UpdateTaxonomy(ctx context.Context, tt *model.TermTaxonomy) error {
	result, err := r.db.NamedExecContext(ctx, `
		UPDATE wp_term_taxonomy SET
			description = :description, parent = :parent, count = :count
		WHERE term_taxonomy_id = :term_taxonomy_id`, tt)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrTermNotFound, "Term taxonomy not found")
	}
	return nil
}

// ErrDefaultCategory is returned when attempting to delete the default category.
var ErrDefaultCategory = "cannot_delete_default_category"

func (r *TermsRepository) Delete(ctx context.Context, id int64) (*model.Term, error) {
	// Protect default category from deletion
	var defaultCat string
	err := r.db.GetContext(ctx, &defaultCat, "SELECT option_value FROM wp_options WHERE option_name = 'default_category'")
	if err == nil && defaultCat == fmt.Sprintf("%d", id) {
		return nil, errors.New(ErrDefaultCategory, "Cannot delete the default category", http.StatusForbidden)
	}

	term, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	defer tx.Rollback()

	// Reassign child terms to the deleted term's parent
	var ttRows []struct {
		TermTaxonomyID int64 `db:"term_taxonomy_id"`
		Parent         int64 `db:"parent"`
	}
	tx.Select(&ttRows, "SELECT term_taxonomy_id, parent FROM wp_term_taxonomy WHERE term_id = ?", id)
	for _, tt := range ttRows {
		tx.Exec("UPDATE wp_term_taxonomy SET parent = ? WHERE parent = ?", tt.Parent, tt.TermTaxonomyID)
	}

	tx.Exec("DELETE FROM wp_term_relationships WHERE term_taxonomy_id IN (SELECT term_taxonomy_id FROM wp_term_taxonomy WHERE term_id = ?)", id)
	tx.Exec("DELETE FROM wp_term_taxonomy WHERE term_id = ?", id)
	tx.Exec("DELETE FROM wp_termmeta WHERE term_id = ?", id)
	tx.Exec("DELETE FROM wp_terms WHERE term_id = ?", id)

	if err := tx.Commit(); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return term, nil
}

func (r *TermsRepository) GetPostTerms(ctx context.Context, postID int64, taxonomy string) ([]model.Category, error) {
	q := `SELECT t.term_id, t.name, t.slug, t.term_group,
		tt.term_taxonomy_id, tt.description, tt.parent, tt.count
		FROM wp_terms t
		JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id
		JOIN wp_term_relationships tr ON tt.term_taxonomy_id = tr.term_taxonomy_id
		WHERE tr.object_id = ?`
	args := []any{postID}

	if taxonomy != "" {
		q += " AND tt.taxonomy = ?"
		args = append(args, taxonomy)
	}
	q += " ORDER BY tr.term_order, t.name"

	var categories []model.Category
	if err := r.db.SelectContext(ctx, &categories, q, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return categories, nil
}

func (r *TermsRepository) SetPostTerms(ctx context.Context, postID int64, termTaxonomyIDs []int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	defer tx.Rollback()

	var oldTTIDs []int64
	tx.Select(&oldTTIDs, "SELECT term_taxonomy_id FROM wp_term_relationships WHERE object_id = ?", postID)
	tx.Exec("DELETE FROM wp_term_relationships WHERE object_id = ?", postID)

	for i, ttID := range termTaxonomyIDs {
		_, err = tx.Exec(
			"INSERT INTO wp_term_relationships (object_id, term_taxonomy_id, term_order) VALUES (?, ?, ?)",
			postID, ttID, i,
		)
		if err != nil {
			return errors.Internal(err, errors.ErrQueryFailed)
		}
	}

	for _, ttID := range oldTTIDs {
		tx.Exec("UPDATE wp_term_taxonomy SET count = (SELECT COUNT(*) FROM wp_term_relationships WHERE term_taxonomy_id = ?) WHERE term_taxonomy_id = ?", ttID, ttID)
	}
	for _, ttID := range termTaxonomyIDs {
		tx.Exec("UPDATE wp_term_taxonomy SET count = (SELECT COUNT(*) FROM wp_term_relationships WHERE term_taxonomy_id = ?) WHERE term_taxonomy_id = ?", ttID, ttID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}

func (r *TermsRepository) AddTermToPost(ctx context.Context, postID, termTaxonomyID int64) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO wp_term_relationships (object_id, term_taxonomy_id, term_order) VALUES (?, ?, 0)",
		postID, termTaxonomyID,
	)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	r.db.ExecContext(ctx, "UPDATE wp_term_taxonomy SET count = (SELECT COUNT(*) FROM wp_term_relationships WHERE term_taxonomy_id = ?) WHERE term_taxonomy_id = ?", termTaxonomyID, termTaxonomyID)
	return nil
}

func (r *TermsRepository) RemoveTermFromPost(ctx context.Context, postID, termTaxonomyID int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM wp_term_relationships WHERE object_id = ? AND term_taxonomy_id = ?", postID, termTaxonomyID)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrNotFound, "Relationship not found")
	}
	r.db.ExecContext(ctx, "UPDATE wp_term_taxonomy SET count = (SELECT COUNT(*) FROM wp_term_relationships WHERE term_taxonomy_id = ?) WHERE term_taxonomy_id = ?", termTaxonomyID, termTaxonomyID)
	return nil
}

func (r *TermsRepository) UpdateCount(ctx context.Context, termTaxonomyID int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE wp_term_taxonomy SET count = (SELECT COUNT(*) FROM wp_term_relationships WHERE term_taxonomy_id = ?) WHERE term_taxonomy_id = ?", termTaxonomyID, termTaxonomyID)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}
