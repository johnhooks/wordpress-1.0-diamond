package permission_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"press/internal/db"
	"press/internal/permission"
)

// can is a test helper that calls Can and fails on error.
func can(t *testing.T, chk permission.Checker, ctx context.Context, userID int64, action permission.Action, object permission.Object) bool {
	t.Helper()
	ok, err := chk.Can(ctx, userID, action, object)
	if err != nil {
		t.Fatalf("Can(%d, %s, %s:%s) returned unexpected error: %v", userID, action, object.Type, object.ID, err)
	}
	return ok
}

// canToken is a test helper that calls CanToken and fails on error.
func canToken(t *testing.T, chk permission.Checker, ctx context.Context, token string, action permission.Action, object permission.Object) bool {
	t.Helper()
	ok, err := chk.CanToken(ctx, token, action, object)
	if err != nil {
		t.Fatalf("CanToken(%s, %s, %s:%s) returned unexpected error: %v", token, action, object.Type, object.ID, err)
	}
	return ok
}

// setup creates a test DB, runs migrations, seeds permission data,
// and returns a checker ready for assertions.
func setup(t *testing.T) (permission.Checker, *permission.Store) {
	t.Helper()

	database := db.SetupTestDB(t)
	store := permission.NewStore(database)
	ctx := context.Background()
	now := time.Now().UTC()

	for _, g := range []permission.Group{
		{Slug: "public", Name: "Public", IsDefault: true, CreatedAt: now},
		{Slug: "administrators", Name: "Administrators", IsDefault: true, CreatedAt: now},
		{Slug: "editors", Name: "Editors", IsDefault: true, CreatedAt: now},
		{Slug: "authors", Name: "Authors", IsDefault: true, CreatedAt: now},
		{Slug: "subscribers", Name: "Subscribers", IsDefault: true, CreatedAt: now},
	} {
		if err := store.CreateGroup(ctx, &g); err != nil {
			t.Fatalf("create group %s: %v", g.Slug, err)
		}
	}

	tuples := []permission.Tuple{
		{SubjectType: "group", SubjectID: "public", Relation: permission.Viewer, ObjectType: "type", ObjectID: "post", CreatedAt: now},
		{SubjectType: "group", SubjectID: "public", Relation: permission.Viewer, ObjectType: "type", ObjectID: "page", CreatedAt: now},
		{SubjectType: "group", SubjectID: "public", Relation: permission.Commenter, ObjectType: "type", ObjectID: "post", CreatedAt: now},
		{SubjectType: "group", SubjectID: "administrators", Relation: permission.Owner, ObjectType: "site", ObjectID: "", CreatedAt: now},
		{SubjectType: "group", SubjectID: "administrators", Relation: permission.Editor, ObjectType: "type", ObjectID: "post", CreatedAt: now},
		{SubjectType: "group", SubjectID: "administrators", Relation: permission.Editor, ObjectType: "type", ObjectID: "page", CreatedAt: now},
		{SubjectType: "group", SubjectID: "administrators", Relation: permission.Editor, ObjectType: "type", ObjectID: "attachment", CreatedAt: now},
		{SubjectType: "group", SubjectID: "editors", Relation: permission.Editor, ObjectType: "type", ObjectID: "post", CreatedAt: now},
		{SubjectType: "group", SubjectID: "editors", Relation: permission.Editor, ObjectType: "type", ObjectID: "page", CreatedAt: now},
		{SubjectType: "group", SubjectID: "editors", Relation: permission.Moderator, ObjectType: "site", ObjectID: "", CreatedAt: now},
		{SubjectType: "group", SubjectID: "editors", Relation: permission.Writer, ObjectType: "type", ObjectID: "attachment", CreatedAt: now},
		{SubjectType: "group", SubjectID: "authors", Relation: permission.Writer, ObjectType: "type", ObjectID: "post", CreatedAt: now},
		{SubjectType: "group", SubjectID: "authors", Relation: permission.Writer, ObjectType: "type", ObjectID: "attachment", CreatedAt: now},
		{SubjectType: "user", SubjectID: "1", Relation: permission.Member, ObjectType: "group", ObjectID: "administrators", CreatedAt: now},
		{SubjectType: "user", SubjectID: "2", Relation: permission.Member, ObjectType: "group", ObjectID: "editors", CreatedAt: now},
		{SubjectType: "user", SubjectID: "3", Relation: permission.Member, ObjectType: "group", ObjectID: "authors", CreatedAt: now},
		{SubjectType: "user", SubjectID: "4", Relation: permission.Member, ObjectType: "group", ObjectID: "subscribers", CreatedAt: now},
	}

	for _, tup := range tuples {
		if err := store.CreateTuple(ctx, &tup); err != nil {
			t.Fatalf("create tuple: %v", err)
		}
	}

	return permission.NewChecker(store), store
}

func createAuthorship(t *testing.T, store *permission.Store, userID int64, postType, postID string) {
	t.Helper()
	tup := permission.Tuple{
		SubjectType: "user",
		SubjectID:   fmt.Sprintf("%d", userID),
		Relation:    permission.Writer,
		ObjectType:  postType,
		ObjectID:    postID,
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   userID,
	}
	if err := store.CreateTuple(context.Background(), &tup); err != nil {
		t.Fatalf("create authorship tuple: %v", err)
	}
}

// --- Relation hierarchy ---

func TestRelationSatisfies(t *testing.T) {
	tests := []struct {
		held     permission.Relation
		required permission.Relation
		want     bool
	}{
		{permission.Owner, permission.Editor, true},
		{permission.Owner, permission.Viewer, true},
		{permission.Owner, permission.Moderator, true},
		{permission.Editor, permission.Writer, true},
		{permission.Editor, permission.Viewer, true},
		{permission.Writer, permission.Commenter, true},
		{permission.Commenter, permission.Viewer, true},
		{permission.Writer, permission.Editor, false},
		{permission.Viewer, permission.Writer, false},
		{permission.Moderator, permission.Editor, false},
		{permission.Member, permission.Viewer, false},
		{permission.Viewer, permission.Viewer, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s→%s", tt.held, tt.required), func(t *testing.T) {
			got := tt.held.Satisfies(tt.required)
			if got != tt.want {
				t.Errorf("(%s).Satisfies(%s) = %v, want %v", tt.held, tt.required, got, tt.want)
			}
		})
	}
}

// --- Group-level permission checks (seed data only, no extra setup) ---

func TestGroupLevelPermissions(t *testing.T) {
	chk, _ := setup(t)
	ctx := context.Background()

	const (
		admin      int64 = 1
		editor     int64 = 2
		author     int64 = 3
		subscriber int64 = 4
		anonymous  int64 = 0
	)

	tests := []struct {
		name   string
		user   int64
		action permission.Action
		object permission.Object
		want   bool
	}{
		// Admin — site administration
		{"admin can manage options", admin, permission.ActionManageOptions, permission.Site(), true},
		{"admin can manage users", admin, permission.ActionManageUsers, permission.Site(), true},
		{"admin can moderate", admin, permission.ActionModerate, permission.Site(), true},
		{"admin can view site (hierarchy)", admin, permission.ActionView, permission.Site(), true},

		// Admin — content (editor on all types)
		{"admin can edit type:post", admin, permission.ActionEdit, permission.TypeObject("post"), true},
		{"admin can edit type:page", admin, permission.ActionEdit, permission.TypeObject("page"), true},
		{"admin can create type:post", admin, permission.ActionCreate, permission.TypeObject("post"), true},
		{"admin can delete type:post", admin, permission.ActionDelete, permission.TypeObject("post"), true},
		{"admin can publish type:post", admin, permission.ActionPublish, permission.TypeObject("post"), true},
		{"admin can upload type:attachment", admin, permission.ActionUpload, permission.TypeObject("attachment"), true},
		{"admin can edit specific post", admin, permission.ActionEdit, permission.PostObject("42"), true},
		{"admin can edit specific page", admin, permission.ActionEdit, permission.PageObject("17"), true},
		{"admin can edit specific attachment", admin, permission.ActionEdit, permission.AttachmentObject("5"), true},

		// Editor — content (editor on post/page, writer on attachment)
		{"editor can edit type:post", editor, permission.ActionEdit, permission.TypeObject("post"), true},
		{"editor can delete type:post", editor, permission.ActionDelete, permission.TypeObject("post"), true},
		{"editor can edit specific post", editor, permission.ActionEdit, permission.PostObject("99"), true},
		{"editor can edit specific page", editor, permission.ActionEdit, permission.PageObject("17"), true},
		{"editor can upload attachments", editor, permission.ActionUpload, permission.TypeObject("attachment"), true},
		{"editor can view posts (hierarchy)", editor, permission.ActionView, permission.TypeObject("post"), true},
		{"editor can comment on posts (hierarchy)", editor, permission.ActionComment, permission.TypeObject("post"), true},

		// Editor — site (moderator)
		{"editor can moderate", editor, permission.ActionModerate, permission.Site(), true},
		{"editor cannot manage options", editor, permission.ActionManageOptions, permission.Site(), false},
		{"editor cannot manage users", editor, permission.ActionManageUsers, permission.Site(), false},

		// Author — content (writer on post/attachment)
		{"author can create posts", author, permission.ActionCreate, permission.TypeObject("post"), true},
		{"author can view posts", author, permission.ActionView, permission.TypeObject("post"), true},
		{"author can comment on posts", author, permission.ActionComment, permission.TypeObject("post"), true},
		{"author can upload attachments", author, permission.ActionUpload, permission.TypeObject("attachment"), true},

		// Author — ownership-sensitive actions denied at type level
		{"author cannot edit at type level", author, permission.ActionEdit, permission.TypeObject("post"), false},
		{"author cannot delete at type level", author, permission.ActionDelete, permission.TypeObject("post"), false},
		{"author cannot publish at type level", author, permission.ActionPublish, permission.TypeObject("post"), false},

		// Author — no page access
		{"author cannot create pages", author, permission.ActionCreate, permission.TypeObject("page"), false},
		{"author cannot create specific page", author, permission.ActionCreate, permission.PageObject("17"), false},

		// Subscriber — no grants beyond public
		{"subscriber cannot create posts", subscriber, permission.ActionCreate, permission.TypeObject("post"), false},
		{"subscriber cannot edit posts", subscriber, permission.ActionEdit, permission.TypeObject("post"), false},
		{"subscriber cannot moderate", subscriber, permission.ActionModerate, permission.Site(), false},
		{"subscriber can view posts (via public)", subscriber, permission.ActionView, permission.TypeObject("post"), true},
		{"subscriber can comment on posts (via public)", subscriber, permission.ActionComment, permission.TypeObject("post"), true},

		// Anonymous — public group only
		{"anonymous can view type:post", anonymous, permission.ActionView, permission.TypeObject("post"), true},
		{"anonymous can view type:page", anonymous, permission.ActionView, permission.TypeObject("page"), true},
		{"anonymous can view specific post", anonymous, permission.ActionView, permission.PostObject("42"), true},
		{"anonymous can comment on posts", anonymous, permission.ActionComment, permission.TypeObject("post"), true},
		{"anonymous cannot comment on pages", anonymous, permission.ActionComment, permission.TypeObject("page"), false},
		{"anonymous cannot create posts", anonymous, permission.ActionCreate, permission.TypeObject("post"), false},
		{"anonymous cannot edit posts", anonymous, permission.ActionEdit, permission.TypeObject("post"), false},
		{"anonymous cannot moderate", anonymous, permission.ActionModerate, permission.Site(), false},

		// Unknown action
		{"unknown action denied for admin", admin, permission.Action("nonexistent"), permission.Site(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := can(t, chk, ctx, tt.user, tt.action, tt.object)
			if got != tt.want {
				t.Errorf("Can(%d, %s, %s:%s) = %v, want %v", tt.user, tt.action, tt.object.Type, tt.object.ID, got, tt.want)
			}
		})
	}
}

// --- Direct grants and authorship ---

func TestDirectGrantPermissions(t *testing.T) {
	chk, store := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Author (3) owns post:50 and attachment:20
	createAuthorship(t, store, 3, "post", "50")
	createAuthorship(t, store, 3, "attachment", "20")

	// Someone else (99) owns post:60 and attachment:10
	createAuthorship(t, store, 99, "post", "60")
	createAuthorship(t, store, 99, "attachment", "10")

	// Subscriber (4) gets direct editor grant on post:42
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "user", SubjectID: "4",
		Relation:  permission.Editor,
		ObjectType: "post", ObjectID: "42",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create direct grant: %v", err)
	}

	tests := []struct {
		name   string
		user   int64
		action permission.Action
		object permission.Object
		want   bool
	}{
		// Author can edit/delete/publish own post via authorship tuple
		{"author can edit own post", 3, permission.ActionEdit, permission.PostObject("50"), true},
		{"author can delete own post", 3, permission.ActionDelete, permission.PostObject("50"), true},
		{"author can publish own post", 3, permission.ActionPublish, permission.PostObject("50"), true},

		// Author cannot edit others' post
		{"author cannot edit others' post", 3, permission.ActionEdit, permission.PostObject("60"), false},

		// Author can edit own attachment
		{"author can edit own attachment", 3, permission.ActionEdit, permission.AttachmentObject("20"), true},

		// Editor (writer on type:attachment) cannot edit others' attachment
		{"editor cannot edit others' attachment", 2, permission.ActionEdit, permission.AttachmentObject("10"), false},

		// Admin (editor on type:attachment) can edit anyone's attachment
		{"admin can edit any attachment", 1, permission.ActionEdit, permission.AttachmentObject("10"), true},

		// Subscriber with direct grant
		{"subscriber can edit post:42 via direct grant", 4, permission.ActionEdit, permission.PostObject("42"), true},
		{"subscriber cannot edit post:43", 4, permission.ActionEdit, permission.PostObject("43"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := can(t, chk, ctx, tt.user, tt.action, tt.object)
			if got != tt.want {
				t.Errorf("Can(%d, %s, %s:%s) = %v, want %v", tt.user, tt.action, tt.object.Type, tt.object.ID, got, tt.want)
			}
		})
	}
}

// --- Per-object group grants ---

func TestPerObjectGroupGrants(t *testing.T) {
	chk, store := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Give public group viewer access to specific post:99
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "group", SubjectID: "public",
		Relation:  permission.Viewer,
		ObjectType: "post", ObjectID: "99",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create per-object group grant: %v", err)
	}

	// Give editors group a writer grant on specific post:200
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "group", SubjectID: "editors",
		Relation:  permission.Writer,
		ObjectType: "post", ObjectID: "200",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create grant: %v", err)
	}

	tests := []struct {
		name   string
		user   int64
		action permission.Action
		object permission.Object
		want   bool
	}{
		{"anonymous can view post:99 via per-object group grant", 0, permission.ActionView, permission.PostObject("99"), true},
		{"editor can edit post:200", 2, permission.ActionEdit, permission.PostObject("200"), true},
		{"editor can still edit post:201 via type-level grant", 2, permission.ActionEdit, permission.PostObject("201"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := can(t, chk, ctx, tt.user, tt.action, tt.object)
			if got != tt.want {
				t.Errorf("Can(%d, %s, %s:%s) = %v, want %v", tt.user, tt.action, tt.object.Type, tt.object.ID, got, tt.want)
			}
		})
	}
}

// --- Multiple group membership ---

func TestMultipleGroupMembership(t *testing.T) {
	chk, store := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for _, group := range []string{"editors", "authors"} {
		if err := store.CreateTuple(ctx, &permission.Tuple{
			SubjectType: "user", SubjectID: "10",
			Relation:  permission.Member,
			ObjectType: "group", ObjectID: group,
			CreatedAt: now,
		}); err != nil {
			t.Fatalf("add to %s: %v", group, err)
		}
	}

	tests := []struct {
		name   string
		action permission.Action
		object permission.Object
		want   bool
	}{
		{"can edit posts (from editors)", permission.ActionEdit, permission.TypeObject("post"), true},
		{"can moderate (from editors)", permission.ActionModerate, permission.Site(), true},
		{"cannot manage options (neither group)", permission.ActionManageOptions, permission.Site(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := can(t, chk, ctx, 10, tt.action, tt.object)
			if got != tt.want {
				t.Errorf("Can(10, %s, %s:%s) = %v, want %v", tt.action, tt.object.Type, tt.object.ID, got, tt.want)
			}
		})
	}
}

// --- Workflow tests (multi-step, can't be table-driven) ---

func TestAuthorshipTransfer(t *testing.T) {
	chk, store := setup(t)
	ctx := context.Background()

	createAuthorship(t, store, 3, "post", "70")

	if !can(t, chk, ctx, 3, permission.ActionEdit, permission.PostObject("70")) {
		t.Error("original author should be able to edit")
	}

	if err := store.DeleteTuple(ctx, "user", "3", permission.Writer, "post", "70"); err != nil {
		t.Fatalf("delete old authorship: %v", err)
	}
	createAuthorship(t, store, 5, "post", "70")

	if can(t, chk, ctx, 3, permission.ActionEdit, permission.PostObject("70")) {
		t.Error("old author should NOT be able to edit after transfer")
	}
	if !can(t, chk, ctx, 5, permission.ActionEdit, permission.PostObject("70")) {
		t.Error("new author should be able to edit after transfer")
	}
}

func TestCollaborativeAuthorship(t *testing.T) {
	chk, store := setup(t)
	ctx := context.Background()

	createAuthorship(t, store, 3, "post", "80")
	createAuthorship(t, store, 5, "post", "80")

	if !can(t, chk, ctx, 3, permission.ActionEdit, permission.PostObject("80")) {
		t.Error("first co-author should be able to edit")
	}
	if !can(t, chk, ctx, 5, permission.ActionEdit, permission.PostObject("80")) {
		t.Error("second co-author should be able to edit")
	}
}

func TestMembershipTakesEffectImmediately(t *testing.T) {
	chk, store := setup(t)
	ctx := context.Background()

	if can(t, chk, ctx, 5, permission.ActionCreate, permission.TypeObject("post")) {
		t.Error("user 5 should NOT be able to create posts before membership")
	}

	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "user", SubjectID: "5",
		Relation:  permission.Member,
		ObjectType: "group", ObjectID: "authors",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("add membership: %v", err)
	}

	if !can(t, chk, ctx, 5, permission.ActionCreate, permission.TypeObject("post")) {
		t.Error("user 5 should be able to create posts immediately after membership")
	}
}

// --- Share tokens ---

func TestTokenAccess(t *testing.T) {
	chk, store := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Viewer token on post:42
	if err := store.CreateShareToken(ctx, &permission.ShareToken{
		Token: "viewer-token", CreatedBy: 1, CreatedAt: now,
	}, permission.Viewer, permission.PostObject("42")); err != nil {
		t.Fatalf("create viewer token: %v", err)
	}

	// Writer token on post:42
	if err := store.CreateShareToken(ctx, &permission.ShareToken{
		Token: "writer-token", CreatedBy: 1, CreatedAt: now,
	}, permission.Writer, permission.PostObject("42")); err != nil {
		t.Fatalf("create writer token: %v", err)
	}

	// Expired token
	past := now.Add(-1 * time.Hour)
	if err := store.CreateShareToken(ctx, &permission.ShareToken{
		Token: "expired-token", CreatedBy: 1, CreatedAt: now, ExpiresAt: &past,
	}, permission.Viewer, permission.PostObject("42")); err != nil {
		t.Fatalf("create expired token: %v", err)
	}

	tests := []struct {
		name   string
		token  string
		action permission.Action
		object permission.Object
		want   bool
	}{
		// Viewer token
		{"viewer token grants view", "viewer-token", permission.ActionView, permission.PostObject("42"), true},
		{"viewer token denies edit", "viewer-token", permission.ActionEdit, permission.PostObject("42"), false},
		{"viewer token scoped to post:42", "viewer-token", permission.ActionView, permission.PostObject("99"), false},

		// Writer token (hierarchy: writer implies viewer)
		{"writer token grants view", "writer-token", permission.ActionView, permission.PostObject("42"), true},
		{"writer token grants edit", "writer-token", permission.ActionEdit, permission.PostObject("42"), true},
		{"writer token denies moderate", "writer-token", permission.ActionModerate, permission.PostObject("42"), false},

		// Expired token
		{"expired token denied", "expired-token", permission.ActionView, permission.PostObject("42"), false},

		// Nonexistent token
		{"nonexistent token denied", "nonexistent", permission.ActionView, permission.PostObject("42"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canToken(t, chk, ctx, tt.token, tt.action, tt.object)
			if got != tt.want {
				t.Errorf("CanToken(%s, %s, %s:%s) = %v, want %v", tt.token, tt.action, tt.object.Type, tt.object.ID, got, tt.want)
			}
		})
	}
}

// --- Store: deletion cascades ---

func TestDeleteGroupCascade(t *testing.T) {
	database := db.SetupTestDB(t)
	store := permission.NewStore(database)
	chk := permission.NewChecker(store)
	ctx := context.Background()
	now := time.Now().UTC()

	g := permission.Group{Slug: "testers", Name: "Testers", CreatedAt: now}
	if err := store.CreateGroup(ctx, &g); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "group", SubjectID: "testers",
		Relation: permission.Writer, ObjectType: "type", ObjectID: "post",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create grant: %v", err)
	}
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "user", SubjectID: "50",
		Relation: permission.Member, ObjectType: "group", ObjectID: "testers",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("add membership: %v", err)
	}

	if !can(t, chk, ctx, 50, permission.ActionCreate, permission.TypeObject("post")) {
		t.Fatal("user 50 should be able to create posts via testers group")
	}

	if err := store.DeleteGroup(ctx, "testers"); err != nil {
		t.Fatalf("delete group: %v", err)
	}

	if can(t, chk, ctx, 50, permission.ActionCreate, permission.TypeObject("post")) {
		t.Error("user 50 should NOT be able to create posts after group deletion")
	}
}

func TestDeleteTuplesForObject(t *testing.T) {
	database := db.SetupTestDB(t)
	store := permission.NewStore(database)
	chk := permission.NewChecker(store)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "user", SubjectID: "7",
		Relation: permission.Writer, ObjectType: "post", ObjectID: "42",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create authorship: %v", err)
	}

	if !can(t, chk, ctx, 7, permission.ActionEdit, permission.PostObject("42")) {
		t.Fatal("user 7 should be able to edit post:42 before deletion")
	}

	n, err := store.DeleteTuplesForObject(ctx, "post", "42")
	if err != nil {
		t.Fatalf("delete tuples for object: %v", err)
	}
	if n == 0 {
		t.Error("should have deleted at least one tuple")
	}

	if can(t, chk, ctx, 7, permission.ActionEdit, permission.PostObject("42")) {
		t.Error("user 7 should NOT be able to edit post:42 after tuple cleanup")
	}
}

func TestDeleteTuplesForSubject(t *testing.T) {
	database := db.SetupTestDB(t)
	store := permission.NewStore(database)
	ctx := context.Background()
	now := time.Now().UTC()

	g := permission.Group{Slug: "writers", Name: "Writers", CreatedAt: now}
	if err := store.CreateGroup(ctx, &g); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "group", SubjectID: "writers",
		Relation: permission.Writer, ObjectType: "type", ObjectID: "post",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create grant: %v", err)
	}
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "user", SubjectID: "8",
		Relation: permission.Member, ObjectType: "group", ObjectID: "writers",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("add membership: %v", err)
	}
	if err := store.CreateTuple(ctx, &permission.Tuple{
		SubjectType: "user", SubjectID: "8",
		Relation: permission.Writer, ObjectType: "post", ObjectID: "100",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create authorship: %v", err)
	}

	chk := permission.NewChecker(store)

	if !can(t, chk, ctx, 8, permission.ActionCreate, permission.TypeObject("post")) {
		t.Fatal("user 8 should be able to create posts before deletion")
	}
	if !can(t, chk, ctx, 8, permission.ActionEdit, permission.PostObject("100")) {
		t.Fatal("user 8 should be able to edit post:100 before deletion")
	}

	n, err := store.DeleteTuplesForSubject(ctx, "user", "8")
	if err != nil {
		t.Fatalf("delete tuples for subject: %v", err)
	}
	if n < 2 {
		t.Errorf("should have deleted at least 2 tuples, got %d", n)
	}

	if can(t, chk, ctx, 8, permission.ActionCreate, permission.TypeObject("post")) {
		t.Error("user 8 should NOT be able to create posts after subject cleanup")
	}
	if can(t, chk, ctx, 8, permission.ActionEdit, permission.PostObject("100")) {
		t.Error("user 8 should NOT be able to edit post:100 after subject cleanup")
	}
}

func TestDeleteShareTokenCascade(t *testing.T) {
	database := db.SetupTestDB(t)
	store := permission.NewStore(database)
	chk := permission.NewChecker(store)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := store.CreateShareToken(ctx, &permission.ShareToken{
		Token: "delete-me", CreatedBy: 1, CreatedAt: now,
	}, permission.Viewer, permission.PostObject("42")); err != nil {
		t.Fatalf("create share token: %v", err)
	}

	if !canToken(t, chk, ctx, "delete-me", permission.ActionView, permission.PostObject("42")) {
		t.Fatal("token should work before deletion")
	}

	if err := store.DeleteShareToken(ctx, "delete-me"); err != nil {
		t.Fatalf("delete share token: %v", err)
	}

	if canToken(t, chk, ctx, "delete-me", permission.ActionView, permission.PostObject("42")) {
		t.Error("deleted token should NOT grant access")
	}
}

// --- Validation ---

func TestInvalidTupleValidation(t *testing.T) {
	database := db.SetupTestDB(t)
	store := permission.NewStore(database)
	ctx := context.Background()
	now := time.Now().UTC()

	tests := []struct {
		name string
		tup  permission.Tuple
	}{
		{"invalid relation", permission.Tuple{
			SubjectType: "user", SubjectID: "1",
			Relation:  "superadmin",
			ObjectType: "site", ObjectID: "",
			CreatedAt: now,
		}},
		{"invalid subject type", permission.Tuple{
			SubjectType: "robot", SubjectID: "1",
			Relation:  permission.Viewer,
			ObjectType: "post", ObjectID: "1",
			CreatedAt: now,
		}},
		{"invalid object type", permission.Tuple{
			SubjectType: "user", SubjectID: "1",
			Relation:  permission.Viewer,
			ObjectType: "widget", ObjectID: "1",
			CreatedAt: now,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateTuple(ctx, &tt.tup)
			if err == nil {
				t.Errorf("should reject tuple with %s", tt.name)
			}
		})
	}
}
