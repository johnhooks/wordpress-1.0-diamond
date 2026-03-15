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

func TestLinksRepository_CRUD(t *testing.T) {
	database := db.SetupTestDB(t)
	links := repository.NewLinksRepository(database)
	ctx := context.Background()

	// Create
	link := &model.Link{
		LinkURL:     "https://example.com",
		LinkName:    "Example",
		LinkVisible: "Y",
		LinkOwner:   1,
	}
	if err := links.Create(ctx, link); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if link.LinkID == 0 {
		t.Fatal("expected link ID to be set")
	}

	// GetByID
	got, err := links.GetByID(ctx, link.LinkID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LinkName != "Example" {
		t.Errorf("expected name 'Example', got %q", got.LinkName)
	}

	// GetByID non-existent
	_, err = links.GetByID(ctx, 9999)
	if !errors.IsCode(err, errors.ErrLinkNotFound) {
		t.Errorf("expected CodeLinkNotFound, got %v", err)
	}

	// List
	result, err := links.List(ctx, query.Query{PerPage: 100})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 link, got %d", len(result.Items))
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}

	// Update
	got.LinkName = "Updated"
	updated, err := links.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.LinkName != "Updated" {
		t.Errorf("expected 'Updated', got %q", updated.LinkName)
	}

	// Update non-existent
	ghost := &model.Link{LinkID: 9999, LinkURL: "x", LinkName: "x", LinkVisible: "Y"}
	_, err = links.Update(ctx, ghost)
	if !errors.IsCode(err, errors.ErrLinkNotFound) {
		t.Errorf("expected CodeLinkNotFound on update, got %v", err)
	}

	// Delete
	deleted, err := links.Delete(ctx, link.LinkID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.LinkName != "Updated" {
		t.Errorf("expected deleted link name 'Updated', got %q", deleted.LinkName)
	}
	result, _ = links.List(ctx, query.Query{PerPage: 100})
	if result.Total != 0 {
		t.Errorf("expected total 0, got %d", result.Total)
	}

	// Delete non-existent
	_, err = links.Delete(ctx, 9999)
	if !errors.IsCode(err, errors.ErrLinkNotFound) {
		t.Errorf("expected CodeLinkNotFound on delete, got %v", err)
	}
}

func TestLinksRepository_VisibilityFilter(t *testing.T) {
	database := db.SetupTestDB(t)
	links := repository.NewLinksRepository(database)
	ctx := context.Background()

	links.Create(ctx, &model.Link{LinkURL: "https://visible.com", LinkName: "Visible", LinkVisible: "Y", LinkOwner: 1})
	links.Create(ctx, &model.Link{LinkURL: "https://hidden.com", LinkName: "Hidden", LinkVisible: "N", LinkOwner: 1})

	result, err := links.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "visible", Operator: query.Is, Value: "Y"}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List visible: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 visible link, got %d", len(result.Items))
	}

	result, _ = links.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "visible", Operator: query.Is, Value: "N"}},
		PerPage: 100,
	})
	if len(result.Items) != 1 {
		t.Errorf("expected 1 hidden link, got %d", len(result.Items))
	}
}

func TestLinksRepository_CategoryFilter(t *testing.T) {
	database := db.SetupTestDB(t)
	links := repository.NewLinksRepository(database)
	terms := repository.NewTermsRepository(database)
	ctx := context.Background()

	cat := &model.Term{Name: "Blogroll", Slug: "blogroll"}
	catTT := &model.TermTaxonomy{Taxonomy: "link_category"}
	terms.Create(ctx, cat, catTT)

	link1 := &model.Link{LinkURL: "https://in-blogroll.com", LinkName: "In Blogroll", LinkVisible: "Y", LinkOwner: 1}
	link2 := &model.Link{LinkURL: "https://no-category.com", LinkName: "No Category", LinkVisible: "Y", LinkOwner: 1}
	links.Create(ctx, link1)
	links.Create(ctx, link2)

	terms.AddTermToPost(ctx, link1.LinkID, catTT.TermTaxonomyID)

	result, err := links.List(ctx, query.Query{
		Filters: []query.Filter{{Field: "category", Operator: query.Is, Value: cat.TermID}},
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("List by category: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 link in Blogroll, got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].LinkName != "In Blogroll" {
		t.Errorf("expected 'In Blogroll', got %q", result.Items[0].LinkName)
	}
}
