package repository_test

import (
	"context"
	"testing"

	"press/internal/db"
	"press/internal/errors"
	"press/internal/repository"
)

func TestOptionsRepository_CRUD(t *testing.T) {
	database := db.SetupTestDB(t)
	options := repository.NewOptionsRepository(database)
	ctx := context.Background()

	// Set
	if err := options.Set(ctx, "blogname", "Test Blog", "yes"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Get
	val, err := options.GetValue(ctx, "blogname")
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	if val != "Test Blog" {
		t.Errorf("expected 'Test Blog', got %q", val)
	}

	// Get full option
	opt, err := options.Get(ctx, "blogname")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if opt.Autoload != "yes" {
		t.Errorf("expected autoload 'yes', got %q", opt.Autoload)
	}

	// Upsert
	if err := options.Set(ctx, "blogname", "Updated Blog", "yes"); err != nil {
		t.Fatalf("Set upsert: %v", err)
	}
	val, _ = options.GetValue(ctx, "blogname")
	if val != "Updated Blog" {
		t.Errorf("expected 'Updated Blog', got %q", val)
	}

	// List
	list, err := options.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 option, got %d", len(list))
	}

	// Delete
	if err := options.Delete(ctx, "blogname"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = options.GetValue(ctx, "blogname")
	if !errors.IsCode(err, errors.ErrOptionNotFound) {
		t.Errorf("expected CodeOptionNotFound after delete, got %v", err)
	}

	// Delete non-existent
	err = options.Delete(ctx, "nonexistent")
	if !errors.IsCode(err, errors.ErrOptionNotFound) {
		t.Errorf("expected CodeOptionNotFound, got %v", err)
	}
}

func TestOptionsRepository_Autoload(t *testing.T) {
	database := db.SetupTestDB(t)
	options := repository.NewOptionsRepository(database)
	ctx := context.Background()

	options.Set(ctx, "blogname", "My Blog", "yes")
	options.Set(ctx, "secret_key", "abc123", "no")
	options.Set(ctx, "blogdescription", "A blog", "yes")

	autoload, err := options.GetAutoloadOptions(ctx)
	if err != nil {
		t.Fatalf("GetAutoloadOptions: %v", err)
	}
	if len(autoload) != 2 {
		t.Errorf("expected 2 autoload options, got %d", len(autoload))
	}
	if autoload["blogname"] != "My Blog" {
		t.Errorf("expected blogname 'My Blog', got %q", autoload["blogname"])
	}
	if _, ok := autoload["secret_key"]; ok {
		t.Error("secret_key should not be in autoload options")
	}
}

func TestOptionsRepository_SetMultiple(t *testing.T) {
	database := db.SetupTestDB(t)
	options := repository.NewOptionsRepository(database)
	ctx := context.Background()

	if err := options.SetMultiple(ctx, map[string]string{"a": "1", "b": "2", "c": "3"}, "yes"); err != nil {
		t.Fatalf("SetMultiple: %v", err)
	}

	list, _ := options.List(ctx)
	if len(list) != 3 {
		t.Errorf("expected 3 options after SetMultiple, got %d", len(list))
	}

	// Upsert via SetMultiple
	if err := options.SetMultiple(ctx, map[string]string{"a": "updated", "d": "4"}, "no"); err != nil {
		t.Fatalf("SetMultiple upsert: %v", err)
	}

	val, _ := options.GetValue(ctx, "a")
	if val != "updated" {
		t.Errorf("expected 'updated', got %q", val)
	}

	list, _ = options.List(ctx)
	if len(list) != 4 {
		t.Errorf("expected 4 options, got %d", len(list))
	}

	opt, _ := options.Get(ctx, "a")
	if opt.Autoload != "no" {
		t.Errorf("expected autoload 'no' after upsert, got %q", opt.Autoload)
	}
}

func TestOptionsRepository_GetNonExistent(t *testing.T) {
	database := db.SetupTestDB(t)
	options := repository.NewOptionsRepository(database)
	ctx := context.Background()

	_, err := options.Get(ctx, "nonexistent")
	if !errors.IsCode(err, errors.ErrOptionNotFound) {
		t.Errorf("expected CodeOptionNotFound, got %v", err)
	}

	_, err = options.GetValue(ctx, "nonexistent")
	if !errors.IsCode(err, errors.ErrOptionNotFound) {
		t.Errorf("expected CodeOptionNotFound for GetValue, got %v", err)
	}
}

func TestOptionsRepository_GetValueDefault(t *testing.T) {
	database := db.SetupTestDB(t)
	options := repository.NewOptionsRepository(database)
	ctx := context.Background()

	// Non-existent returns default
	val, err := options.GetValueDefault(ctx, "nonexistent", "fallback")
	if err != nil {
		t.Fatalf("GetValueDefault: %v", err)
	}
	if val != "fallback" {
		t.Errorf("expected 'fallback', got %q", val)
	}

	// Existing returns actual value
	options.Set(ctx, "blogname", "My Blog", "yes")
	val, err = options.GetValueDefault(ctx, "blogname", "fallback")
	if err != nil {
		t.Fatalf("GetValueDefault existing: %v", err)
	}
	if val != "My Blog" {
		t.Errorf("expected 'My Blog', got %q", val)
	}
}
