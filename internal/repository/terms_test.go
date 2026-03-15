package repository_test

import (
	"context"
	"fmt"
	"testing"

	"press/internal/db"
	"press/internal/errors"
	"press/internal/model"
	"press/internal/query"
	"press/internal/repository"
)

func TestTermsRepository_CRUD(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	// Create
	term := &model.Term{Name: "Tech", Slug: "tech"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	if err := terms.Create(ctx, term, tt); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if term.TermID == 0 {
		t.Fatal("expected term ID to be set")
	}
	if tt.TermTaxonomyID == 0 {
		t.Fatal("expected term_taxonomy ID to be set")
	}

	// GetByID
	got, err := terms.GetByID(ctx, term.TermID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Tech" {
		t.Errorf("expected name 'Tech', got %q", got.Name)
	}

	// GetByID non-existent
	_, err = terms.GetByID(ctx, 9999)
	if !errors.IsCode(err, errors.ErrTermNotFound) {
		t.Errorf("expected CodeTermNotFound, got %v", err)
	}

	// GetBySlug
	got, err = terms.GetBySlug(ctx, "tech")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.TermID != term.TermID {
		t.Error("GetBySlug returned wrong term")
	}

	// GetTaxonomy
	gotTT, err := terms.GetTaxonomy(ctx, term.TermID, "category")
	if err != nil {
		t.Fatalf("GetTaxonomy: %v", err)
	}
	if gotTT.Taxonomy != "category" {
		t.Errorf("expected taxonomy 'category', got %q", gotTT.Taxonomy)
	}

	// List
	result, err := terms.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "taxonomy", Operator: query.Is, Value: "category"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 category, got %d", len(result.Items))
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}

	// Update
	got.Name = "Technology"
	updated, err := terms.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Technology" {
		t.Errorf("expected 'Technology', got %q", updated.Name)
	}

	// UpdateTaxonomy
	gotTT.Description = "All things tech"
	if err := terms.UpdateTaxonomy(ctx, gotTT); err != nil {
		t.Fatalf("UpdateTaxonomy: %v", err)
	}
	gotTT2, _ := terms.GetTaxonomy(ctx, term.TermID, "category")
	if gotTT2.Description != "All things tech" {
		t.Errorf("expected description 'All things tech', got %q", gotTT2.Description)
	}

	// Delete
	deleted, err := terms.Delete(ctx, term.TermID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.Name != "Technology" {
		t.Errorf("expected deleted term name 'Technology', got %q", deleted.Name)
	}
	result, _ = terms.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "taxonomy", Operator: query.Is, Value: "category"}},
		PerPage: 100,
	})
	if result.Total != 0 {
		t.Errorf("expected total 0 after delete, got %d", result.Total)
	}

	// Delete non-existent
	_, err = terms.Delete(ctx, 9999)
	if !errors.IsCode(err, errors.ErrTermNotFound) {
		t.Errorf("expected CodeTermNotFound, got %v", err)
	}
}

func TestTermsRepository_PostTerms(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	term := &model.Term{Name: "Go", Slug: "go"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, term, tt)

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Go Post", PostName: "go-post",
		PostContent: "test", PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post)

	// AddTermToPost
	if err := terms.AddTermToPost(ctx, post.ID, tt.TermTaxonomyID); err != nil {
		t.Fatalf("AddTermToPost: %v", err)
	}

	postTerms, err := terms.GetPostTerms(ctx, post.ID, "category")
	if err != nil {
		t.Fatalf("GetPostTerms: %v", err)
	}
	if len(postTerms) != 1 {
		t.Errorf("expected 1 post term, got %d", len(postTerms))
	}
	if postTerms[0].Name != "Go" {
		t.Errorf("expected term name 'Go', got %q", postTerms[0].Name)
	}

	// Verify count updated
	ttGot, _ := terms.GetTaxonomy(ctx, term.TermID, "category")
	if ttGot.Count != 1 {
		t.Errorf("expected count 1, got %d", ttGot.Count)
	}

	// AddTermToPost again (idempotent)
	terms.AddTermToPost(ctx, post.ID, tt.TermTaxonomyID)

	// RemoveTermFromPost
	if err := terms.RemoveTermFromPost(ctx, post.ID, tt.TermTaxonomyID); err != nil {
		t.Fatalf("RemoveTermFromPost: %v", err)
	}
	postTerms, _ = terms.GetPostTerms(ctx, post.ID, "category")
	if len(postTerms) != 0 {
		t.Errorf("expected 0 post terms after remove, got %d", len(postTerms))
	}

	ttGot, _ = terms.GetTaxonomy(ctx, term.TermID, "category")
	if ttGot.Count != 0 {
		t.Errorf("expected count 0 after remove, got %d", ttGot.Count)
	}

	// Remove non-existent
	err = terms.RemoveTermFromPost(ctx, post.ID, tt.TermTaxonomyID)
	if !errors.IsCode(err, errors.ErrNotFound) {
		t.Errorf("expected CodeNotFound, got %v", err)
	}
}

func TestTermsRepository_SetPostTerms(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	cat1 := &model.Term{Name: "Go", Slug: "go"}
	tt1 := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, cat1, tt1)

	cat2 := &model.Term{Name: "Rust", Slug: "rust"}
	tt2 := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, cat2, tt2)

	cat3 := &model.Term{Name: "Python", Slug: "python"}
	tt3 := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, cat3, tt3)

	post := &model.Post{
		PostAuthor: 1, PostTitle: "Multi-cat", PostName: "multi-cat",
		PostContent: "test", PostStatus: "publish", PostType: "post",
	}
	posts.Create(ctx, post)

	// Set to Go and Rust
	terms.SetPostTerms(ctx, post.ID, []int64{tt1.TermTaxonomyID, tt2.TermTaxonomyID})
	postTerms, _ := terms.GetPostTerms(ctx, post.ID, "category")
	if len(postTerms) != 2 {
		t.Errorf("expected 2 terms, got %d", len(postTerms))
	}

	// Replace with just Python
	terms.SetPostTerms(ctx, post.ID, []int64{tt3.TermTaxonomyID})
	postTerms, _ = terms.GetPostTerms(ctx, post.ID, "category")
	if len(postTerms) != 1 {
		t.Errorf("expected 1 term after replace, got %d", len(postTerms))
	}
	if postTerms[0].Name != "Python" {
		t.Errorf("expected 'Python', got %q", postTerms[0].Name)
	}

	tt1Got, _ := terms.GetTaxonomy(ctx, cat1.TermID, "category")
	if tt1Got.Count != 0 {
		t.Errorf("expected Go count 0 after replace, got %d", tt1Got.Count)
	}
	tt3Got, _ := terms.GetTaxonomy(ctx, cat3.TermID, "category")
	if tt3Got.Count != 1 {
		t.Errorf("expected Python count 1, got %d", tt3Got.Count)
	}
}

func TestTermsRepository_ParentFilter(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	parent := &model.Term{Name: "Programming", Slug: "programming"}
	parentTT := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, parent, parentTT)

	child := &model.Term{Name: "Go", Slug: "go"}
	childTT := &model.TermTaxonomy{Taxonomy: "category", Parent: parentTT.TermTaxonomyID}
	terms.Create(ctx, child, childTT)

	toplevel := &model.Term{Name: "News", Slug: "news"}
	toplevelTT := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, toplevel, toplevelTT)

	// Filter by parent=0 (top level)
	result, err := terms.List(ctx, query.Query{
		Filters: []query.Filter{
			{Field: "taxonomy", Operator: query.Is, Value: "category"},
			{Field: "parent", Operator: query.Is, Value: int64(0)},
		},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List top level: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 top-level categories, got %d", len(result.Items))
	}

	// Filter by parent = parentTT
	result, err = terms.List(ctx, query.Query{
		Filters: []query.Filter{
			{Field: "taxonomy", Operator: query.Is, Value: "category"},
			{Field: "parent", Operator: query.Is, Value: parentTT.TermTaxonomyID},
		},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List children: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 child category, got %d", len(result.Items))
	}
}

func TestTermsRepository_MultipleTaxonomies(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	cat := &model.Term{Name: "Tech", Slug: "tech"}
	catTT := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, cat, catTT)

	linkCat := &model.Term{Name: "Blogroll", Slug: "blogroll"}
	linkTT := &model.TermTaxonomy{Taxonomy: "link_category"}
	terms.Create(ctx, linkCat, linkTT)

	result, _ := terms.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "taxonomy", Operator: query.Is, Value: "category"}},
		PerPage: 100,
	})
	if len(result.Items) != 1 {
		t.Errorf("expected 1 category, got %d", len(result.Items))
	}

	result, _ = terms.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "taxonomy", Operator: query.Is, Value: "link_category"}},
		PerPage: 100,
	})
	if len(result.Items) != 1 {
		t.Errorf("expected 1 link_category, got %d", len(result.Items))
	}

	result, _ = terms.List(ctx, query.Query{PerPage: 100})
	if len(result.Items) != 2 {
		t.Errorf("expected 2 total terms, got %d", len(result.Items))
	}
}

func TestTermsRepository_SlugAutoGeneration(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	term := &model.Term{Name: "Web Development"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	if err := terms.Create(ctx, term, tt); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if term.Slug != "web-development" {
		t.Errorf("expected slug 'web-development', got %q", term.Slug)
	}

	term2 := &model.Term{Name: "Web Development"}
	tt2 := &model.TermTaxonomy{Taxonomy: "category"}
	if err := terms.Create(ctx, term2, tt2); err != nil {
		t.Fatalf("Create duplicate: %v", err)
	}
	if term2.Slug != "web-development-2" {
		t.Errorf("expected slug 'web-development-2', got %q", term2.Slug)
	}
}

func TestTermsRepository_DefaultCategoryProtection(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	options := repository.NewOptionsRepository(database)
	ctx := context.Background()

	term := &model.Term{Name: "Default", Slug: "default"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, term, tt)
	options.Set(ctx, "default_category", fmt.Sprintf("%d", term.TermID), "yes")

	_, err := terms.Delete(ctx, term.TermID)
	if !errors.IsCode(err, repository.ErrDefaultCategory) {
		t.Errorf("expected ErrDefaultCategory, got %v", err)
	}

	got, _ := terms.GetByID(ctx, term.TermID)
	if got == nil {
		t.Fatal("default category should not have been deleted")
	}
}

func TestTermsRepository_DeleteReassignsChildren(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	parent := &model.Term{Name: "Parent", Slug: "parent"}
	parentTT := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, parent, parentTT)

	child := &model.Term{Name: "Child", Slug: "child"}
	childTT := &model.TermTaxonomy{Taxonomy: "category", Parent: parentTT.TermTaxonomyID}
	terms.Create(ctx, child, childTT)

	terms.Delete(ctx, parent.TermID)

	childTTGot, err := terms.GetTaxonomy(ctx, child.TermID, "category")
	if err != nil {
		t.Fatalf("GetTaxonomy: %v", err)
	}
	if childTTGot.Parent != 0 {
		t.Errorf("expected child parent 0 after parent delete, got %d", childTTGot.Parent)
	}
}

func TestTermsRepository_HideEmpty(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	posts := repository.NewPostsRepository(database)
	ctx := context.Background()

	used := &model.Term{Name: "Used", Slug: "used"}
	usedTT := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, used, usedTT)

	empty := &model.Term{Name: "Empty", Slug: "empty"}
	emptyTT := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, empty, emptyTT)

	post := &model.Post{PostAuthor: 1, PostTitle: "P", PostName: "p", PostContent: "x", PostStatus: "publish", PostType: "post"}
	posts.Create(ctx, post)
	terms.AddTermToPost(ctx, post.ID, usedTT.TermTaxonomyID)

	// Without hide_empty
	result, _ := terms.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "taxonomy", Operator: query.Is, Value: "category"}},
		PerPage: 100,
	})
	if len(result.Items) != 2 {
		t.Errorf("expected 2 total categories, got %d", len(result.Items))
	}

	// With hide_empty
	result, err := terms.List(ctx, query.Query{
		Filters: []query.Filter{
			{Field: "taxonomy", Operator: query.Is, Value: "category"},
			{Field: "hide_empty", Operator: query.Is, Value: true},
		},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List hide_empty: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 non-empty category, got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].Name != "Used" {
		t.Errorf("expected 'Used', got %q", result.Items[0].Name)
	}
}

func TestTermsRepository_Search(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	terms.Create(ctx, &model.Term{Name: "Go Programming", Slug: "go-programming"}, &model.TermTaxonomy{Taxonomy: "category"})
	terms.Create(ctx, &model.Term{Name: "Rust Language", Slug: "rust-language"}, &model.TermTaxonomy{Taxonomy: "category"})
	terms.Create(ctx, &model.Term{Name: "Go Modules", Slug: "go-modules"}, &model.TermTaxonomy{Taxonomy: "category"})

	result, err := terms.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "taxonomy", Operator: query.Is, Value: "category"}},
		Search:  "Go",
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 results for 'Go', got %d", len(result.Items))
	}
}

func TestTermsRepository_GetByName(t *testing.T) {
	database := db.SetupTestDB(t)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	terms.Create(ctx, &model.Term{Name: "JavaScript", Slug: "javascript"}, &model.TermTaxonomy{Taxonomy: "category"})

	got, err := terms.GetByName(ctx, "JavaScript", "category")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.Slug != "javascript" {
		t.Errorf("expected slug 'javascript', got %q", got.Slug)
	}

	_, err = terms.GetByName(ctx, "Nonexistent", "category")
	if !errors.IsCode(err, errors.ErrTermNotFound) {
		t.Errorf("expected ErrTermNotFound, got %v", err)
	}
}
