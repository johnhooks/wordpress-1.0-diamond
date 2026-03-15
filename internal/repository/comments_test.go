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

func createTestPost(t *testing.T, ctx context.Context, posts *repository.PostsRepository) *model.Post {
	t.Helper()
	post := &model.Post{
		PostAuthor: 1, PostTitle: "Comment Test", PostName: "comment-test-" + t.Name(),
		PostContent: "test", PostStatus: "publish", PostType: "post",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatalf("create test post: %v", err)
	}
	return post
}

func TestCommentsRepository_CRUD(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post := createTestPost(t, ctx, posts)

	// Create approved comment
	comment := &model.Comment{
		CommentPostID:      post.ID,
		CommentAuthor:      "Jane",
		CommentAuthorEmail: "jane@example.com",
		CommentContent:     "Great post!",
		CommentApproved:    "1",
		CommentType:        "comment",
	}
	if err := comments.Create(ctx, comment); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if comment.CommentID == 0 {
		t.Fatal("expected comment ID to be set")
	}
	if comment.CommentDate.IsZero() {
		t.Error("expected CommentDate to be auto-set")
	}

	// Verify post comment_count updated
	updatedPost, _ := posts.GetByID(ctx, post.ID)
	if updatedPost.CommentCount != 1 {
		t.Errorf("expected comment_count 1, got %d", updatedPost.CommentCount)
	}

	// GetByID
	got, err := comments.GetByID(ctx, comment.CommentID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.CommentContent != "Great post!" {
		t.Errorf("unexpected content: %q", got.CommentContent)
	}

	// GetByID non-existent
	_, err = comments.GetByID(ctx, 9999)
	if !errors.IsCode(err, errors.ErrCommentNotFound) {
		t.Errorf("expected CodeCommentNotFound, got %v", err)
	}

	// GetByPostID
	byPost, err := comments.GetByPostID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByPostID: %v", err)
	}
	if len(byPost) != 1 {
		t.Errorf("expected 1 comment, got %d", len(byPost))
	}

	// Update
	got.CommentContent = "Updated!"
	updated, err := comments.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.CommentContent != "Updated!" {
		t.Errorf("expected 'Updated!', got %q", updated.CommentContent)
	}

	// Unapprove
	if err := comments.Unapprove(ctx, comment.CommentID); err != nil {
		t.Fatalf("Unapprove: %v", err)
	}
	updatedPost, _ = posts.GetByID(ctx, post.ID)
	if updatedPost.CommentCount != 0 {
		t.Errorf("expected comment_count 0 after unapprove, got %d", updatedPost.CommentCount)
	}

	// Unapprove again (idempotent)
	if err := comments.Unapprove(ctx, comment.CommentID); err != nil {
		t.Fatalf("Unapprove again: %v", err)
	}

	// Approve
	if err := comments.Approve(ctx, comment.CommentID); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	updatedPost, _ = posts.GetByID(ctx, post.ID)
	if updatedPost.CommentCount != 1 {
		t.Errorf("expected comment_count 1 after approve, got %d", updatedPost.CommentCount)
	}

	// Approve again (idempotent)
	if err := comments.Approve(ctx, comment.CommentID); err != nil {
		t.Fatalf("Approve again: %v", err)
	}

	// Delete
	deleted, err := comments.Delete(ctx, comment.CommentID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.CommentContent != "Updated!" {
		t.Errorf("expected deleted comment content 'Updated!', got %q", deleted.CommentContent)
	}
	updatedPost, _ = posts.GetByID(ctx, post.ID)
	if updatedPost.CommentCount != 0 {
		t.Errorf("expected comment_count 0 after delete, got %d", updatedPost.CommentCount)
	}
}

func TestCommentsRepository_ListFilters(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post1 := createTestPost(t, ctx, posts)
	post2 := &model.Post{
		PostAuthor: 1, PostTitle: "Post 2", PostName: "post-2",
		PostContent: "test2", PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post2)

	comments.Create(ctx, &model.Comment{
		CommentPostID: post1.ID, CommentAuthor: "Alice",
		CommentContent: "First", CommentApproved: "1", CommentType: "comment",
	})
	comments.Create(ctx, &model.Comment{
		CommentPostID: post1.ID, CommentAuthor: "Bob",
		CommentContent: "Pending", CommentApproved: "0", CommentType: "comment",
	})
	comments.Create(ctx, &model.Comment{
		CommentPostID: post2.ID, CommentAuthor: "Carol",
		CommentContent: "Other post", CommentApproved: "1", CommentType: "comment",
	})

	// Filter by post
	result, err := comments.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "post", Operator: query.Is, Value: post1.ID}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List by post: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 comments on post1, got %d", len(result.Items))
	}

	// Filter by approved
	result, err = comments.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "approved", Operator: query.Is, Value: "1"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List approved: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 approved comments, got %d", len(result.Items))
	}

	// Filter by post AND approved
	result, err = comments.List(ctx, query.Query{
		Filters: []query.Filter{
			{Field: "post", Operator: query.Is, Value: post1.ID},
			{Field: "approved", Operator: query.Is, Value: "0"},
		},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List by both: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 pending comment on post1, got %d", len(result.Items))
	}

	// Pagination
	result, err = comments.List(ctx, query.Query{PerPage: 1, Page: 1})
	if err != nil {
		t.Fatalf("List limited: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 with limit, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
}

func TestCommentsRepository_UnapprovedCreate(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post := createTestPost(t, ctx, posts)

	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Spammer",
		CommentContent: "buy stuff", CommentApproved: "0", CommentType: "comment",
	})

	updatedPost, _ := posts.GetByID(ctx, post.ID)
	if updatedPost.CommentCount != 0 {
		t.Errorf("unapproved comment should not increment count, got %d", updatedPost.CommentCount)
	}
}

func TestCommentsRepository_UpdateSyncsCount(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post := createTestPost(t, ctx, posts)

	c := &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "A",
		CommentContent: "hi", CommentApproved: "1", CommentType: "comment",
	}
	comments.Create(ctx, c)

	p, _ := posts.GetByID(ctx, post.ID)
	if p.CommentCount != 1 {
		t.Fatalf("setup: expected count 1, got %d", p.CommentCount)
	}

	// Unapprove via Update()
	c.CommentApproved = "0"
	comments.Update(ctx, c)
	p, _ = posts.GetByID(ctx, post.ID)
	if p.CommentCount != 0 {
		t.Errorf("expected count 0 after Update unapprove, got %d", p.CommentCount)
	}

	// Re-approve via Update()
	c.CommentApproved = "1"
	comments.Update(ctx, c)
	p, _ = posts.GetByID(ctx, post.ID)
	if p.CommentCount != 1 {
		t.Errorf("expected count 1 after Update approve, got %d", p.CommentCount)
	}
}

func TestCommentsRepository_DeleteReparentsChildren(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post := createTestPost(t, ctx, posts)

	parent := &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Parent",
		CommentContent: "parent", CommentApproved: "1", CommentType: "comment",
	}
	comments.Create(ctx, parent)

	child := &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Child",
		CommentContent: "child", CommentApproved: "1", CommentType: "comment",
		CommentParent: parent.CommentID,
	}
	comments.Create(ctx, child)

	comments.Delete(ctx, parent.CommentID)

	got, err := comments.GetByID(ctx, child.CommentID)
	if err != nil {
		t.Fatalf("GetByID child: %v", err)
	}
	if got.CommentParent != 0 {
		t.Errorf("expected child comment_parent 0 after parent delete, got %d", got.CommentParent)
	}
}

func TestCommentsRepository_TypeFilter(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post := createTestPost(t, ctx, posts)

	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "User",
		CommentContent: "a comment", CommentApproved: "1", CommentType: "comment",
	})
	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Pinger",
		CommentContent: "pingback", CommentApproved: "1", CommentType: "pingback",
	})

	result, err := comments.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "type", Operator: query.Is, Value: "comment"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List by type: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 comment type, got %d", len(result.Items))
	}

	result, _ = comments.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "type", Operator: query.Is, Value: "pingback"}},
		PerPage: 100,
	})
	if len(result.Items) != 1 {
		t.Errorf("expected 1 pingback, got %d", len(result.Items))
	}
}

func TestCommentsRepository_ParentFilter(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post := createTestPost(t, ctx, posts)

	parent := &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Parent",
		CommentContent: "top level", CommentApproved: "1", CommentType: "comment",
	}
	comments.Create(ctx, parent)

	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Child",
		CommentContent: "reply", CommentApproved: "1", CommentType: "comment",
		CommentParent: parent.CommentID,
	})
	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Other",
		CommentContent: "top level 2", CommentApproved: "1", CommentType: "comment",
	})

	// Top-level only
	result, err := comments.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "parent", Operator: query.Is, Value: int64(0)}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List parent=0: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 top-level comments, got %d", len(result.Items))
	}

	// Replies to parent
	result, _ = comments.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "parent", Operator: query.Is, Value: parent.CommentID}},
		PerPage: 100,
	})
	if len(result.Items) != 1 {
		t.Errorf("expected 1 reply, got %d", len(result.Items))
	}
}

func TestCommentsRepository_AuthorEmailFilter(t *testing.T) {
	database := db.SetupTestDB(t)
	comments := repository.NewCommentsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	post := createTestPost(t, ctx, posts)

	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Alice",
		CommentAuthorEmail: "alice@example.com",
		CommentContent: "hi", CommentApproved: "1", CommentType: "comment",
	})
	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Bob",
		CommentAuthorEmail: "bob@example.com",
		CommentContent: "hey", CommentApproved: "1", CommentType: "comment",
	})

	result, err := comments.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "email", Operator: query.Is, Value: "alice@example.com"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List by email: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 comment from alice, got %d", len(result.Items))
	}
}
