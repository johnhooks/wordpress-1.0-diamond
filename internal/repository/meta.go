package repository

import (
	"context"
	"database/sql"
	"fmt"

	"press/internal/errors"

	"github.com/jmoiron/sqlx"
)

// Meta represents a single metadata row across all WordPress meta tables.
type Meta struct {
	MetaID    int64  `db:"meta_id"`
	ObjectID  int64  // mapped dynamically per-table (post_id, user_id, etc.)
	MetaKey   string `db:"meta_key"`
	MetaValue string `db:"meta_value"`
}

// MetaRepository provides CRUD operations for any WordPress meta table.
// All column references in queries are fully qualified (table.column) to avoid
// ambiguity.
type MetaRepository struct {
	db       *sqlx.DB
	table    string // wp_postmeta, wp_usermeta, etc.
	pkColumn string // meta_id (bare column name)
	fkColumn string // post_id, user_id, etc. (bare column name)
}

// col returns a fully qualified table.column reference.
func (r *MetaRepository) col(column string) string {
	return r.table + "." + column
}

// NewPostMeta creates a MetaRepository for wp_postmeta.
func NewPostMeta(db *sqlx.DB) *MetaRepository {
	return &MetaRepository{db: db, table: "wp_postmeta", pkColumn: "meta_id", fkColumn: "post_id"}
}

// NewUserMeta creates a MetaRepository for wp_usermeta.
func NewUserMeta(db *sqlx.DB) *MetaRepository {
	return &MetaRepository{db: db, table: "wp_usermeta", pkColumn: "meta_id", fkColumn: "user_id"}
}

// NewCommentMeta creates a MetaRepository for wp_commentmeta.
func NewCommentMeta(db *sqlx.DB) *MetaRepository {
	return &MetaRepository{db: db, table: "wp_commentmeta", pkColumn: "meta_id", fkColumn: "comment_id"}
}

// NewTermMeta creates a MetaRepository for wp_termmeta.
func NewTermMeta(db *sqlx.DB) *MetaRepository {
	return &MetaRepository{db: db, table: "wp_termmeta", pkColumn: "meta_id", fkColumn: "term_id"}
}

// Get returns the first meta value for a key on an object.
func (r *MetaRepository) Get(ctx context.Context, objectID int64, metaKey string) (*Meta, error) {
	q := fmt.Sprintf(
		"SELECT %s, %s, %s, %s FROM %s WHERE %s = ? AND %s = ? LIMIT 1",
		r.col(r.pkColumn), r.col(r.fkColumn), r.col("meta_key"), r.col("meta_value"),
		r.table, r.col(r.fkColumn), r.col("meta_key"),
	)
	row := r.db.QueryRowxContext(ctx, q, objectID, metaKey)

	meta := &Meta{}
	err := row.Scan(&meta.MetaID, &meta.ObjectID, &meta.MetaKey, &meta.MetaValue)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrNotFound, fmt.Sprintf("%s not found: %s", r.table, metaKey))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return meta, nil
}

// Set upserts a single-value meta key. If the key exists, its value is updated.
func (r *MetaRepository) Set(ctx context.Context, objectID int64, metaKey, metaValue string) error {
	_, err := r.Get(ctx, objectID, metaKey)
	if errors.IsCode(err, errors.ErrNotFound) {
		_, err = r.db.ExecContext(ctx,
			fmt.Sprintf("INSERT INTO %s (%s, meta_key, meta_value) VALUES (?, ?, ?)", r.table, r.fkColumn),
			objectID, metaKey, metaValue,
		)
	} else if err != nil {
		return err
	} else {
		_, err = r.db.ExecContext(ctx,
			fmt.Sprintf("UPDATE %s SET meta_value = ? WHERE %s = ? AND %s = ?",
				r.table, r.col(r.fkColumn), r.col("meta_key")),
			metaValue, objectID, metaKey,
		)
	}
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}

// Delete removes all meta rows matching the key for an object.
func (r *MetaRepository) Delete(ctx context.Context, objectID int64, metaKey string) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE %s = ? AND %s = ?",
		r.table, r.col(r.fkColumn), r.col("meta_key"))
	result, err := r.db.ExecContext(ctx, q, objectID, metaKey)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrNotFound, fmt.Sprintf("%s not found", r.table))
	}
	return nil
}

// GetAll returns all meta rows for an object.
func (r *MetaRepository) GetAll(ctx context.Context, objectID int64) ([]Meta, error) {
	q := fmt.Sprintf(
		"SELECT %s, %s, %s, %s FROM %s WHERE %s = ? ORDER BY %s",
		r.col(r.pkColumn), r.col(r.fkColumn), r.col("meta_key"), r.col("meta_value"),
		r.table, r.col(r.fkColumn), r.col("meta_key"),
	)
	rows, err := r.db.QueryxContext(ctx, q, objectID)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	defer func() { _ = rows.Close() }()

	var metas []Meta
	for rows.Next() {
		var m Meta
		if err := rows.Scan(&m.MetaID, &m.ObjectID, &m.MetaKey, &m.MetaValue); err != nil {
			return nil, errors.Internal(err, errors.ErrQueryFailed)
		}
		metas = append(metas, m)
	}
	return metas, nil
}

// DeleteAll removes all meta rows for an object.
func (r *MetaRepository) DeleteAll(ctx context.Context, objectID int64) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", r.table, r.col(r.fkColumn))
	_, err := r.db.ExecContext(ctx, q, objectID)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}
