package repository_test

import (
	"context"
	"testing"
	"time"

	"press/internal/db"
	"press/internal/model"
	"press/internal/query"
	"press/internal/repository"
)

func TestPostsRepository_CRUD(t *testing.T) {
	database := db.SetupTestDB(t)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	// Create
	post := &model.Post{
		PostAuthor:  1,
		PostTitle:   "Test Post",
		PostName:    "test-post",
		PostContent: "This is a test.",
		PostStatus:  "publish",
		PostType:    "post",
	}
	if err := posts.Create(ctx, post); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if post.ID == 0 {
		t.Fatal("expected post ID to be set")
	}
	if post.PostDate.IsZero() {
		t.Error("expected PostDate to be auto-set")
	}
	if post.PostModified.IsZero() {
		t.Error("expected PostModified to be auto-set")
	}

	// GetByID
	got, err := posts.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.PostTitle != "Test Post" {
		t.Errorf("expected title 'Test Post', got %q", got.PostTitle)
	}

	// GetBySlug
	got, err = posts.GetBySlug(ctx, "test-post")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.ID != post.ID {
		t.Errorf("expected ID %d, got %d", post.ID, got.ID)
	}

	// GetBySlugAndType
	got, err = posts.GetBySlugAndType(ctx, "test-post", "post")
	if err != nil {
		t.Fatalf("GetBySlugAndType: %v", err)
	}
	if got.ID != post.ID {
		t.Errorf("GetBySlugAndType: expected ID %d, got %d", post.ID, got.ID)
	}

	// GetBySlugAndType with wrong type
	_, err = posts.GetBySlugAndType(ctx, "test-post", "page")
	if err == nil {
		t.Error("expected error for wrong post type")
	}

	// Update
	got.PostTitle = "Updated Post"
	updated, err := posts.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.PostTitle != "Updated Post" {
		t.Errorf("expected title 'Updated Post', got %q", updated.PostTitle)
	}
	got2, _ := posts.GetByID(ctx, post.ID)
	if got2.PostTitle != "Updated Post" {
		t.Errorf("expected title 'Updated Post', got %q", got2.PostTitle)
	}

	// List
	result, err := posts.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "type", Operator: query.Is, Value: "post"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 post, got %d", len(result.Items))
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}

	// Delete
	deleted, err := posts.Delete(ctx, post.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.PostTitle != "Updated Post" {
		t.Errorf("expected deleted post title 'Updated Post', got %q", deleted.PostTitle)
	}

	result, _ = posts.List(ctx, query.Query{PerPage: 100})
	if result.Total != 0 {
		t.Errorf("expected total 0 after delete, got %d", result.Total)
	}

	// Delete non-existent
	_, err = posts.Delete(ctx, 9999)
	if err == nil {
		t.Error("expected error deleting non-existent post")
	}

	// GetByID non-existent
	_, err = posts.GetByID(ctx, 9999)
	if err == nil {
		t.Error("expected error getting non-existent post")
	}
}

func TestPostsRepository_ListFilters(t *testing.T) {
	database := db.SetupTestDB(t)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	now := time.Now().UTC()
	for i, title := range []string{"Alpha", "Beta", "Gamma"} {
		post := &model.Post{
			PostAuthor:  int64(i + 1),
			PostTitle:   title,
			PostName:    title,
			PostContent: "content about " + title,
			PostStatus:  "publish",
			PostType:    "post",
			PostDate:    now.Add(time.Duration(i) * time.Hour),
			PostDateGmt: now.Add(time.Duration(i) * time.Hour),
		}
		if i == 2 {
			post.PostStatus = "draft"
		}
		posts.Create(ctx, post)
	}

	// Filter by status
	result, err := posts.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "status", Operator: query.Is, Value: "publish"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List published: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 published, got %d", len(result.Items))
	}

	// Filter by author
	result, err = posts.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "author", Operator: query.Is, Value: int64(1)}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List by author: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 post by author 1, got %d", len(result.Items))
	}

	// Search
	result, err = posts.List(ctx, query.Query{Search: "Alpha", PerPage: 100})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 post matching search 'Alpha', got %d", len(result.Items))
	}

	// Pagination
	result, err = posts.List(ctx, query.Query{PerPage: 1, Page: 1})
	if err != nil {
		t.Fatalf("List limited: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 post with limit, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if result.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", result.TotalPages)
	}

	result2, err := posts.List(ctx, query.Query{PerPage: 1, Page: 2})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(result2.Items) != 1 {
		t.Errorf("expected 1 post on page 2, got %d", len(result2.Items))
	}
	if len(result.Items) > 0 && len(result2.Items) > 0 && result.Items[0].ID == result2.Items[0].ID {
		t.Error("page 2 should return different post than page 1")
	}

	// Post type filter
	posts.Create(ctx, &model.Post{
		PostAuthor: 1, PostTitle: "About", PostName: "about",
		PostContent: "page", PostStatus: "publish", PostType: "page",
	})
	result, err = posts.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "type", Operator: query.Is, Value: "page"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List pages: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 page, got %d", len(result.Items))
	}
}

func TestPostsRepository_CategoryFilter(t *testing.T) {
	database := db.SetupTestDB(t)
	posts := repository.NewPostsRepository(database)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	// Create category
	term := &model.Term{Name: "Tech", Slug: "tech"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, term, tt)

	// Create posts
	p1 := &model.Post{PostAuthor: 1, PostTitle: "Go is great", PostName: "go-great", PostContent: "x", PostStatus: "publish", PostType: "post"}
	p2 := &model.Post{PostAuthor: 1, PostTitle: "Unrelated", PostName: "unrelated", PostContent: "y", PostStatus: "publish", PostType: "post"}
	posts.Create(ctx, p1)
	posts.Create(ctx, p2)

	terms.AddTermToPost(ctx, p1.ID, tt.TermTaxonomyID)

	// Filter by category
	result, err := posts.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "category", Operator: query.Is, Value: term.TermID}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List by category: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 post in Tech category, got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].PostTitle != "Go is great" {
		t.Errorf("expected 'Go is great', got %q", result.Items[0].PostTitle)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1 for Tech category, got %d", result.Total)
	}
}

func TestPostsRepository_EnsureUniqueSlug(t *testing.T) {
	database := db.SetupTestDB(t)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	posts.Create(ctx, &model.Post{
		PostAuthor: 1, PostTitle: "Hello", PostName: "hello",
		PostContent: "x", PostStatus: "publish", PostType: "post",
	})

	s, err := posts.EnsureUniqueSlug(ctx, "hello", 0)
	if err != nil {
		t.Fatalf("EnsureUniqueSlug: %v", err)
	}
	if s != "hello-2" {
		t.Errorf("expected 'hello-2', got %q", s)
	}

	posts.Create(ctx, &model.Post{
		PostAuthor: 1, PostTitle: "Hello 2", PostName: "hello-2",
		PostContent: "x", PostStatus: "publish", PostType: "post",
	})

	s, err = posts.EnsureUniqueSlug(ctx, "hello", 0)
	if err != nil {
		t.Fatalf("EnsureUniqueSlug: %v", err)
	}
	if s != "hello-3" {
		t.Errorf("expected 'hello-3', got %q", s)
	}

	s, err = posts.EnsureUniqueSlug(ctx, "unique-post", 0)
	if err != nil {
		t.Fatalf("EnsureUniqueSlug unique: %v", err)
	}
	if s != "unique-post" {
		t.Errorf("expected 'unique-post', got %q", s)
	}
}

func TestPostsRepository_DeleteCascade(t *testing.T) {
	database := db.SetupTestDB(t)
	posts := repository.NewPostsRepository(database)
	comments := repository.NewCommentsRepository(database)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Cascade", PostName: "cascade",
		PostContent: "x", PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post)

	comments.Create(ctx, &model.Comment{
		CommentPostID: post.ID, CommentAuthor: "Jane",
		CommentContent: "hi", CommentApproved: "1", CommentType: "comment",
	})

	term := &model.Term{Name: "Cat", Slug: "cat"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, term, tt)
	terms.AddTermToPost(ctx, post.ID, tt.TermTaxonomyID)

	posts.Delete(ctx, post.ID)

	byPost, err := comments.GetByPostID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByPostID: %v", err)
	}
	if len(byPost) != 0 {
		t.Errorf("expected 0 comments after cascade delete, got %d", len(byPost))
	}

	postTerms, err := terms.GetPostTerms(ctx, post.ID, "category")
	if err != nil {
		t.Fatalf("GetPostTerms: %v", err)
	}
	if len(postTerms) != 0 {
		t.Errorf("expected 0 term relationships after cascade delete, got %d", len(postTerms))
	}
}

func TestPostsRepository_Sort(t *testing.T) {
	database := db.SetupTestDB(t)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	posts.Create(ctx, &model.Post{PostAuthor: 1, PostTitle: "Zebra", PostName: "zebra", PostContent: "x", PostStatus: "publish", PostType: "post"})
	posts.Create(ctx, &model.Post{PostAuthor: 1, PostTitle: "Alpha", PostName: "alpha", PostContent: "x", PostStatus: "publish", PostType: "post"})
	posts.Create(ctx, &model.Post{PostAuthor: 1, PostTitle: "Middle", PostName: "middle", PostContent: "x", PostStatus: "publish", PostType: "post"})

	result, err := posts.List(ctx, query.Query{
		Sort:    &query.Sort{Field: "title", Direction: "asc"},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List sorted: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(result.Items))
	}
	if result.Items[0].PostTitle != "Alpha" {
		t.Errorf("expected first post 'Alpha', got %q", result.Items[0].PostTitle)
	}
	if result.Items[2].PostTitle != "Zebra" {
		t.Errorf("expected last post 'Zebra', got %q", result.Items[2].PostTitle)
	}
}
