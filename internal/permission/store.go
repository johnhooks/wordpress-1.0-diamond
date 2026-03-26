package permission

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"press/internal/errors"
)

// Store handles persistence for groups, tuples, and share tokens.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new permission store.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// --- Groups ---

// CreateGroup inserts a new group.
func (s *Store) CreateGroup(ctx context.Context, g *Group) error {
	result, err := s.db.NamedExecContext(ctx, `
		INSERT INTO wp_groups (slug, name, description, is_default, created_at, created_by)
		VALUES (:slug, :name, :description, :is_default, :created_at, :created_by)`, g)
	if err != nil {
		return errors.Wrap(err, errors.ErrQueryFailed, "Failed to create group", 500)
	}
	id, _ := result.LastInsertId()
	g.ID = id
	return nil
}

// GetGroup returns a group by slug.
func (s *Store) GetGroup(ctx context.Context, slug string) (*Group, error) {
	var g Group
	err := s.db.GetContext(ctx, &g, "SELECT * FROM wp_groups WHERE slug = ?", slug)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrGroupNotFound, fmt.Sprintf("Group %q not found", slug))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return &g, nil
}

// ListGroups returns all groups.
func (s *Store) ListGroups(ctx context.Context) ([]Group, error) {
	var groups []Group
	err := s.db.SelectContext(ctx, &groups, "SELECT * FROM wp_groups ORDER BY name")
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return groups, nil
}

// DeleteGroup removes a group and all its tuples.
func (s *Store) DeleteGroup(ctx context.Context, slug string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	defer func() { _ = tx.Rollback() }()

	// Delete tuples where this group is subject or object.
	_, err = tx.ExecContext(ctx,
		"DELETE FROM wp_tuples WHERE (subject_type = 'group' AND subject_id = ?) OR (object_type = 'group' AND object_id = ?)",
		slug, slug)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM wp_groups WHERE slug = ?", slug)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrGroupNotFound, fmt.Sprintf("Group %q not found", slug))
	}

	return tx.Commit()
}

// --- Tuples ---

// CreateTuple inserts a new tuple after validating its fields.
func (s *Store) CreateTuple(ctx context.Context, t *Tuple) error {
	if !ValidRelation(t.Relation) {
		return errors.BadRequest(errors.ErrInvalidParam, fmt.Sprintf("invalid relation %q", t.Relation))
	}
	if !ValidSubjectType(t.SubjectType) {
		return errors.BadRequest(errors.ErrInvalidParam, fmt.Sprintf("invalid subject type %q", t.SubjectType))
	}
	if !ValidObjectType(t.ObjectType) {
		return errors.BadRequest(errors.ErrInvalidParam, fmt.Sprintf("invalid object type %q", t.ObjectType))
	}

	result, err := s.db.NamedExecContext(ctx, `
		INSERT INTO wp_tuples (subject_type, subject_id, relation, object_type, object_id, created_at, created_by)
		VALUES (:subject_type, :subject_id, :relation, :object_type, :object_id, :created_at, :created_by)`, t)
	if err != nil {
		return errors.Wrap(err, errors.ErrQueryFailed, "Failed to create tuple", 500)
	}
	id, _ := result.LastInsertId()
	t.ID = id
	return nil
}

// DeleteTuple removes a specific tuple.
func (s *Store) DeleteTuple(ctx context.Context, subjectType, subjectID string, relation Relation, objectType, objectID string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM wp_tuples
		WHERE subject_type = ? AND subject_id = ? AND relation = ? AND object_type = ? AND object_id = ?`,
		subjectType, subjectID, string(relation), objectType, objectID)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrTupleNotFound, "Tuple not found")
	}
	return nil
}

// GetUserGroups returns the group slugs a user belongs to.
func (s *Store) GetUserGroups(ctx context.Context, userID int64) ([]string, error) {
	var slugs []string
	err := s.db.SelectContext(ctx, &slugs, `
		SELECT object_id FROM wp_tuples
		WHERE subject_type = 'user' AND subject_id = ? AND relation = 'member' AND object_type = 'group'`,
		fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return slugs, nil
}

// GetGroupTuples returns all non-member tuples where the given group is the subject.
func (s *Store) GetGroupTuples(ctx context.Context, groupSlug string) ([]Tuple, error) {
	var tuples []Tuple
	err := s.db.SelectContext(ctx, &tuples, `
		SELECT * FROM wp_tuples
		WHERE subject_type = 'group' AND subject_id = ? AND relation != 'member'`,
		groupSlug)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return tuples, nil
}

// GetDirectGrants returns tuples where a specific user is granted access
// to a specific object.
func (s *Store) GetDirectGrants(ctx context.Context, userID int64, objectType, objectID string) ([]Tuple, error) {
	var tuples []Tuple
	err := s.db.SelectContext(ctx, &tuples, `
		SELECT * FROM wp_tuples
		WHERE subject_type = 'user' AND subject_id = ? AND object_type = ? AND object_id = ?`,
		fmt.Sprintf("%d", userID), objectType, objectID)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return tuples, nil
}

// GetTokenGrants returns tuples where a token is the subject for a specific object.
func (s *Store) GetTokenGrants(ctx context.Context, token, objectType, objectID string) ([]Tuple, error) {
	var tuples []Tuple
	err := s.db.SelectContext(ctx, &tuples, `
		SELECT * FROM wp_tuples
		WHERE subject_type = 'token' AND subject_id = ? AND object_type = ? AND object_id = ?`,
		token, objectType, objectID)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return tuples, nil
}

// DeleteTuplesForObject removes all tuples referencing a specific object.
// Call this when deleting a post, page, or attachment.
func (s *Store) DeleteTuplesForObject(ctx context.Context, objectType, objectID string) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM wp_tuples WHERE object_type = ? AND object_id = ?",
		objectType, objectID)
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// DeleteTuplesForSubject removes all tuples where the given subject is the actor.
// Call this when deleting a user.
func (s *Store) DeleteTuplesForSubject(ctx context.Context, subjectType, subjectID string) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM wp_tuples WHERE subject_type = ? AND subject_id = ?",
		subjectType, subjectID)
	if err != nil {
		return 0, errors.Internal(err, errors.ErrQueryFailed)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// GetGroupGrantsForObject returns group-level tuples relevant to checking
// access on a given object. It queries for both exact object matches and
// type-level grants in a single query.
func (s *Store) GetGroupGrantsForObject(ctx context.Context, groups []string, object Object) ([]Tuple, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		SELECT * FROM wp_tuples
		WHERE subject_type = 'group' AND subject_id IN (?)
		AND (
			(object_type = ? AND object_id = ?)
			OR (object_type = 'type' AND object_id = ?)
		)`, groups, object.Type, object.ID, object.Type)
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	query = s.db.Rebind(query)

	var tuples []Tuple
	if err := s.db.SelectContext(ctx, &tuples, query, args...); err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return tuples, nil
}

// --- Share Tokens ---

// CreateShareToken inserts a share token and its corresponding tuple.
func (s *Store) CreateShareToken(ctx context.Context, st *ShareToken, relation Relation, object Object) error {
	if !ValidRelation(relation) {
		return errors.BadRequest(errors.ErrInvalidParam, fmt.Sprintf("invalid relation %q", relation))
	}
	if !ValidObjectType(object.Type) {
		return errors.BadRequest(errors.ErrInvalidParam, fmt.Sprintf("invalid object type %q", object.Type))
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO wp_share_tokens (token, created_by, created_at, expires_at)
		VALUES (?, ?, ?, ?)`,
		st.Token, st.CreatedBy, st.CreatedAt, st.ExpiresAt)
	if err != nil {
		return errors.Wrap(err, errors.ErrQueryFailed, "Failed to create share token", 500)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO wp_tuples (subject_type, subject_id, relation, object_type, object_id, created_at, created_by)
		VALUES ('token', ?, ?, ?, ?, ?, ?)`,
		st.Token, string(relation), object.Type, object.ID, st.CreatedAt, st.CreatedBy)
	if err != nil {
		return errors.Wrap(err, errors.ErrQueryFailed, "Failed to create token tuple", 500)
	}

	return tx.Commit()
}

// ValidateToken checks if a token exists and is not expired.
func (s *Store) ValidateToken(ctx context.Context, token string) (*ShareToken, error) {
	var st ShareToken
	err := s.db.GetContext(ctx, &st, "SELECT * FROM wp_share_tokens WHERE token = ?", token)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrTokenNotFound, "Share token not found")
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	if st.ExpiresAt != nil && st.ExpiresAt.Before(time.Now().UTC()) {
		return nil, errors.New(errors.ErrTokenExpired, "Share token has expired", 403)
	}
	return &st, nil
}

// DeleteShareToken removes a token and its tuples.
func (s *Store) DeleteShareToken(ctx context.Context, token string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, "DELETE FROM wp_tuples WHERE subject_type = 'token' AND subject_id = ?", token)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM wp_share_tokens WHERE token = ?", token)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrTokenNotFound, "Share token not found")
	}

	return tx.Commit()
}

