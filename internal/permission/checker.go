package permission

import (
	"context"
	"fmt"

	"press/internal/errors"
)

// Checker determines whether a subject can perform an action on an object.
type Checker interface {
	Can(ctx context.Context, userID int64, action Action, object Object) (bool, error)
	CanToken(ctx context.Context, token string, action Action, object Object) (bool, error)
}

type checker struct {
	store *Store
}

// NewChecker creates a Checker backed by the given store.
func NewChecker(store *Store) Checker {
	return &checker{store: store}
}

// Can checks whether userID can perform action on object.
//
// Every check hits the database. There is no in-memory cache — the
// system is deliberately simple for now. Caching is an application-level
// concern that will be added later.
func (c *checker) Can(ctx context.Context, userID int64, action Action, object Object) (bool, error) {
	required, ok := RequiredRelation(action)
	if !ok {
		return false, nil
	}

	// Step 1: Determine the user's groups. Always include public.
	groups, err := c.store.GetUserGroups(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get user groups: %w", err)
	}
	hasPublic := false
	for _, g := range groups {
		if g == PublicGroup {
			hasPublic = true
			break
		}
	}
	if !hasPublic {
		groups = append(groups, PublicGroup)
	}

	// Step 2: Group-level check (DB query).
	ok, err = c.checkGroupGrants(ctx, groups, required, action, object)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}

	// Step 3: Direct user grant (DB query, specific objects only).
	if object.Type != "site" && object.Type != "type" {
		ok, err = c.checkDirectGrant(ctx, userID, required, object)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

// CanToken checks whether a share token grants access.
func (c *checker) CanToken(ctx context.Context, token string, action Action, object Object) (bool, error) {
	required, ok := RequiredRelation(action)
	if !ok {
		return false, nil
	}

	// Validate the token (exists, not expired).
	_, err := c.store.ValidateToken(ctx, token)
	if err != nil {
		// Token not found or expired is a normal denial, not an error.
		if isExpectedTokenError(err) {
			return false, nil
		}
		return false, fmt.Errorf("validate token: %w", err)
	}

	// Check token grants on this specific object.
	tuples, err := c.store.GetTokenGrants(ctx, token, object.Type, object.ID)
	if err != nil {
		return false, fmt.Errorf("get token grants: %w", err)
	}
	for _, t := range tuples {
		if t.Relation.Satisfies(required) {
			return true, nil
		}
	}

	return false, nil
}

// isExpectedTokenError returns true for token errors that represent
// normal denial (not found, expired) rather than infrastructure failure.
func isExpectedTokenError(err error) bool {
	return errors.IsCode(err, errors.ErrTokenNotFound) ||
		errors.IsCode(err, errors.ErrTokenExpired)
}

// checkGroupGrants queries the DB for group-level grants relevant to
// this object and evaluates them against the required relation.
func (c *checker) checkGroupGrants(ctx context.Context, groups []string, required Relation, action Action, object Object) (bool, error) {
	tuples, err := c.store.GetGroupGrantsForObject(ctx, groups, object)
	if err != nil {
		return false, fmt.Errorf("get group grants: %w", err)
	}

	for _, t := range tuples {
		if !t.Relation.Satisfies(required) {
			continue
		}

		switch {
		// Site-level match: grant on site:"" only covers checks
		// against the site object itself.
		case t.ObjectType == "site" && object.Type == "site":
			return true, nil

		// Type-level match: grant on type:X covers objects of that
		// type, but writer only covers non-ownership actions.
		case t.ObjectType == "type":
			if typeGrantCovers(t.Relation, action) {
				return true, nil
			}

		// Specific object match from a group grant (e.g.,
		// group:public → viewer → post:42).
		default:
			if t.ObjectType == object.Type && t.ObjectID == object.ID {
				return true, nil
			}
		}
	}

	return false, nil
}

// typeGrantCovers decides whether a type-level grant covers a given action.
//
// Editor or higher at the type level covers all actions on any object of
// that type. Writer at the type level only covers non-ownership actions
// (create, view, comment, upload). Ownership-sensitive actions (edit,
// delete, publish) on specific objects require a direct grant — which is
// how authorship works: creating a post produces user:N → writer → post:ID.
func typeGrantCovers(grantRelation Relation, action Action) bool {
	if grantRelation.Satisfies(Editor) {
		return true
	}
	return !ownershipActions[action]
}

// checkDirectGrant queries the DB for user-specific grants on a specific object.
func (c *checker) checkDirectGrant(ctx context.Context, userID int64, required Relation, object Object) (bool, error) {
	tuples, err := c.store.GetDirectGrants(ctx, userID, object.Type, object.ID)
	if err != nil {
		return false, fmt.Errorf("get direct grants: %w", err)
	}
	for _, t := range tuples {
		if t.Relation.Satisfies(required) {
			return true, nil
		}
	}
	return false, nil
}
