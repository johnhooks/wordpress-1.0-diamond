package repository_test

import (
	"context"
	"testing"

	"press/internal/db"
	"press/internal/errors"
	"press/internal/model"
	"press/internal/query"
	"press/internal/repository"
)

func TestUsersRepository_CRUD(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	// Create
	user := &model.User{
		UserLogin:    "testuser",
		UserPass:     "hashed",
		UserNicename: "testuser",
		UserEmail:    "test@example.com",
		DisplayName:  "Test User",
	}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be set")
	}
	if user.UserRegistered.IsZero() {
		t.Error("expected UserRegistered to be auto-set")
	}

	// GetByID
	got, err := users.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserLogin != "testuser" {
		t.Errorf("expected login 'testuser', got %q", got.UserLogin)
	}

	// GetByLogin
	got, err = users.GetByLogin(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if got.ID != user.ID {
		t.Error("GetByLogin returned wrong user")
	}

	// GetByEmail
	got, err = users.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != user.ID {
		t.Error("GetByEmail returned wrong user")
	}

	// Update
	got.DisplayName = "Updated Name"
	updated, err := users.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DisplayName != "Updated Name" {
		t.Errorf("expected 'Updated Name', got %q", updated.DisplayName)
	}

	// Update non-existent
	ghost := &model.User{ID: 9999, UserLogin: "ghost", UserNicename: "ghost", UserEmail: "g@g.com", DisplayName: "g"}
	_, err = users.Update(ctx, ghost)
	if !errors.IsCode(err, errors.ErrUserNotFound) {
		t.Errorf("expected CodeUserNotFound, got %v", err)
	}

	// List
	result, err := users.List(ctx, query.Query{PerPage: 100})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 user, got %d", len(result.Items))
	}

	// Count
	count, err := users.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Delete
	deleted, err := users.Delete(ctx, user.ID, 0)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.UserLogin != "testuser" {
		t.Errorf("expected deleted user login 'testuser', got %q", deleted.UserLogin)
	}
	count, _ = users.Count(ctx)
	if count != 0 {
		t.Errorf("expected count 0 after delete, got %d", count)
	}

	// Delete non-existent
	_, err = users.Delete(ctx, 9999, 0)
	if !errors.IsCode(err, errors.ErrUserNotFound) {
		t.Errorf("expected CodeUserNotFound on delete, got %v", err)
	}
}

func TestUsersRepository_DuplicateLogin(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	if err := users.Create(ctx, &model.User{
		UserLogin: "admin", UserPass: "x", UserNicename: "admin",
		UserEmail: "admin@example.com", DisplayName: "Admin",
	}); err != nil {
		t.Fatal(err)
	}

	err := users.Create(ctx, &model.User{
		UserLogin: "admin", UserPass: "x", UserNicename: "admin2",
		UserEmail: "other@example.com", DisplayName: "Dup",
	})
	if !errors.IsCode(err, errors.ErrUserLoginExists) {
		t.Errorf("expected CodeUserLoginExists, got %v", err)
	}
}

func TestUsersRepository_DuplicateEmail(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	if err := users.Create(ctx, &model.User{
		UserLogin: "user1", UserPass: "x", UserNicename: "user1",
		UserEmail: "same@example.com", DisplayName: "User1",
	}); err != nil {
		t.Fatal(err)
	}

	err := users.Create(ctx, &model.User{
		UserLogin: "user2", UserPass: "x", UserNicename: "user2",
		UserEmail: "same@example.com", DisplayName: "User2",
	})
	if !errors.IsCode(err, errors.ErrUserEmailExists) {
		t.Errorf("expected CodeUserEmailExists, got %v", err)
	}
}

func TestUsersRepository_GetNotFound(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	_, err := users.GetByID(ctx, 9999)
	if !errors.IsCode(err, errors.ErrUserNotFound) {
		t.Errorf("expected CodeUserNotFound, got %v", err)
	}

	_, err = users.GetByLogin(ctx, "nobody")
	if !errors.IsCode(err, errors.ErrUserNotFound) {
		t.Errorf("expected CodeUserNotFound for login, got %v", err)
	}

	_, err = users.GetByEmail(ctx, "nobody@example.com")
	if !errors.IsCode(err, errors.ErrUserNotFound) {
		t.Errorf("expected CodeUserNotFound for email, got %v", err)
	}
}

func TestUsersRepository_DeleteCleansUpMeta(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	meta := repository.NewUserMeta(database)
	ctx := context.Background()

	user := &model.User{UserLogin: "metauser", UserPass: "x", UserNicename: "metauser", UserEmail: "m@t.com", DisplayName: "m"}
	if err := users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := meta.Set(ctx, user.ID, "wp_user_level", "10"); err != nil {
		t.Fatal(err)
	}
	if err := meta.Set(ctx, user.ID, "nickname", "admin"); err != nil {
		t.Fatal(err)
	}

	if _, err := users.Delete(ctx, user.ID, 0); err != nil {
		t.Fatal(err)
	}

	all, err := meta.GetAll(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetAll after delete: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 metas after user delete, got %d", len(all))
	}
}

func TestUsersRepository_NicenameAutoGeneration(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	user := &model.User{UserLogin: "John Doe", UserPass: "x", UserEmail: "john@example.com"}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.UserNicename != "john-doe" {
		t.Errorf("expected nicename 'john-doe', got %q", user.UserNicename)
	}

	if user.DisplayName != "John Doe" {
		t.Errorf("expected display name 'John Doe', got %q", user.DisplayName)
	}
}

func TestUsersRepository_DeleteReassignsPosts(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	u1 := &model.User{UserLogin: "author1", UserPass: "x", UserEmail: "a1@t.com"}
	u2 := &model.User{UserLogin: "author2", UserPass: "x", UserEmail: "a2@t.com"}
	if err := users.Create(ctx, u1); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, u2); err != nil {
		t.Fatal(err)
	}

	post := &model.Post{
		PostAuthor: u1.ID, PostTitle: "My Post", PostName: "my-post",
		PostContent: "x", PostStatus: "publish", PostType: "post",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	if _, err := users.Delete(ctx, u1.ID, u2.ID); err != nil {
		t.Fatal(err)
	}

	got, err := posts.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.PostAuthor != u2.ID {
		t.Errorf("expected post_author %d, got %d", u2.ID, got.PostAuthor)
	}
}

func TestUsersRepository_DeleteWithoutReassign(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	u := &model.User{UserLogin: "doomed", UserPass: "x", UserEmail: "d@t.com"}
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	if err := posts.Create(ctx, &model.Post{
		PostAuthor: u.ID, PostTitle: "Gone", PostName: "gone",
		PostContent: "x", PostStatus: "publish", PostType: "post",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := users.Delete(ctx, u.ID, 0); err != nil {
		t.Fatal(err)
	}

	result, _ := posts.List(ctx, query.Query{PerPage: 100})
	if result.Total != 0 {
		t.Errorf("expected 0 posts after user delete without reassign, got %d", result.Total)
	}
}

func TestUsersRepository_DeleteCascadesPostRelated(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	posts := repository.NewPostsRepository(database)
	comments := repository.NewCommentsRepository(database)
	terms := repository.NewTermsRepository(database)
	postMeta := repository.NewPostMeta(database)
	ctx := context.Background()

	u := &model.User{UserLogin: "cascade-user", UserPass: "x", UserEmail: "cu@t.com"}
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	post := &model.Post{
		PostAuthor: u.ID, PostTitle: "Cascade Post", PostName: "cascade-post",
		PostContent: "x", PostStatus: "publish", PostType: "post",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	// Add comments, meta, and terms to the post
	if err := comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Visitor",
		CommentContent: "hi", CommentApproved: "1", CommentType: "comment",
	}); err != nil {
		t.Fatal(err)
	}
	if err := postMeta.Set(ctx, post.ID, "custom", "value"); err != nil {
		t.Fatal(err)
	}

	cat := &model.Term{Name: "UserCat", Slug: "usercat"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	if err := terms.Create(ctx, cat, tt); err != nil {
		t.Fatal(err)
	}
	if err := terms.AddTermToPost(ctx, post.ID, tt.TermTaxonomyID); err != nil {
		t.Fatal(err)
	}

	// Delete without reassign
	if _, err := users.Delete(ctx, u.ID, 0); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Post should be gone
	_, err := posts.GetByID(ctx, post.ID)
	if !errors.IsCode(err, errors.ErrPostNotFound) {
		t.Errorf("expected post not found, got %v", err)
	}

	// Comments should be gone
	byPost, _ := comments.GetByPostID(ctx, post.ID)
	if len(byPost) != 0 {
		t.Errorf("expected 0 comments, got %d", len(byPost))
	}

	// Postmeta should be gone
	all, _ := postMeta.GetAll(ctx, post.ID)
	if len(all) != 0 {
		t.Errorf("expected 0 postmeta, got %d", len(all))
	}

	// Term relationships should be gone
	postTerms, _ := terms.GetPostTerms(ctx, post.ID, "category")
	if len(postTerms) != 0 {
		t.Errorf("expected 0 term relationships, got %d", len(postTerms))
	}
}

func TestUsersRepository_DeleteReassignsLinks(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	links := repository.NewLinksRepository(database)
	ctx := context.Background()

	u1 := &model.User{UserLogin: "linkowner", UserPass: "x", UserEmail: "lo@t.com"}
	u2 := &model.User{UserLogin: "linkreceiver", UserPass: "x", UserEmail: "lr@t.com"}
	if err := users.Create(ctx, u1); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, u2); err != nil {
		t.Fatal(err)
	}

	link := &model.Link{
		LinkURL: "https://example.com", LinkName: "Example",
		LinkVisible: "Y", LinkOwner: u1.ID,
	}
	if err := links.Create(ctx, link); err != nil {
		t.Fatal(err)
	}

	if _, err := users.Delete(ctx, u1.ID, u2.ID); err != nil {
		t.Fatal(err)
	}

	got, err := links.GetByID(ctx, link.LinkID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LinkOwner != u2.ID {
		t.Errorf("expected link_owner %d, got %d", u2.ID, got.LinkOwner)
	}
}

func TestUsersRepository_DeleteWithoutReassignDeletesLinks(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	links := repository.NewLinksRepository(database)
	ctx := context.Background()

	u := &model.User{UserLogin: "linkdoomed", UserPass: "x", UserEmail: "ld@t.com"}
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	if err := links.Create(ctx, &model.Link{
		LinkURL: "https://doomed.com", LinkName: "Doomed",
		LinkVisible: "Y", LinkOwner: u.ID,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := users.Delete(ctx, u.ID, 0); err != nil {
		t.Fatal(err)
	}

	result, _ := links.List(ctx, query.Query{PerPage: 100})
	if result.Total != 0 {
		t.Errorf("expected 0 links after user delete, got %d", result.Total)
	}
}

func TestUsersRepository_UsernameExists(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	u := &model.User{UserLogin: "checker", UserPass: "x", UserEmail: "c@t.com"}
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	id, err := users.UsernameExists(ctx, "checker")
	if err != nil {
		t.Fatalf("UsernameExists: %v", err)
	}
	if id != u.ID {
		t.Errorf("expected user ID %d, got %d", u.ID, id)
	}

	id, err = users.UsernameExists(ctx, "nobody")
	if err != nil {
		t.Fatalf("UsernameExists not found: %v", err)
	}
	if id != 0 {
		t.Errorf("expected 0 for nonexistent, got %d", id)
	}
}

func TestUsersRepository_EmailExists(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	u := &model.User{UserLogin: "emailcheck", UserPass: "x", UserEmail: "exists@t.com"}
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	id, err := users.EmailExists(ctx, "exists@t.com")
	if err != nil {
		t.Fatalf("EmailExists: %v", err)
	}
	if id != u.ID {
		t.Errorf("expected user ID %d, got %d", u.ID, id)
	}

	id, _ = users.EmailExists(ctx, "nope@t.com")
	if id != 0 {
		t.Errorf("expected 0, got %d", id)
	}
}

func TestUsersRepository_ListWithSearch(t *testing.T) {
	database := db.SetupTestDB(t)
	users := repository.NewUsersRepository(database)
	ctx := context.Background()

	if err := users.Create(ctx, &model.User{UserLogin: "alice", UserPass: "x", UserEmail: "alice@t.com", DisplayName: "Alice Smith"}); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, &model.User{UserLogin: "bob", UserPass: "x", UserEmail: "bob@t.com", DisplayName: "Bob Jones"}); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, &model.User{UserLogin: "charlie", UserPass: "x", UserEmail: "charlie@t.com", DisplayName: "Charlie Smith"}); err != nil {
		t.Fatal(err)
	}

	// Search by name
	result, err := users.List(ctx, query.Query{Search: "Smith", PerPage: 100})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 Smiths, got %d", len(result.Items))
	}

	// Search by login
	result, _ = users.List(ctx, query.Query{Search: "bob", PerPage: 100})
	if len(result.Items) != 1 {
		t.Errorf("expected 1 for 'bob', got %d", len(result.Items))
	}

	// Pagination
	result, _ = users.List(ctx, query.Query{PerPage: 2, Page: 1})
	if len(result.Items) != 2 {
		t.Errorf("expected 2 with limit, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}

	// No filters returns all
	result, _ = users.List(ctx, query.Query{PerPage: 100})
	if len(result.Items) != 3 {
		t.Errorf("expected 3 total, got %d", len(result.Items))
	}
}
