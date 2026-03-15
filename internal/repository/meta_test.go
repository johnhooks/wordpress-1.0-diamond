package repository_test

import (
	"context"
	"testing"

	"press/internal/db"
	"press/internal/errors"
	"press/internal/model"
	"press/internal/repository"

	"github.com/jmoiron/sqlx"
)

func TestMetaRepository(t *testing.T) {
	database := db.SetupTestDB(t)
	ctx := context.Background()

	type metaSetup struct {
		name        string
		newMeta     func(db *sqlx.DB) *repository.MetaRepository
		createOwner func(t *testing.T) int64
	}

	// Create owners for each meta type
	posts := repository.NewPostsRepository(database)
	users := repository.NewUsersRepository(database)
	comments := repository.NewCommentsRepository(database)
	terms := repository.NewTermsRepository(database)

	setups := []metaSetup{
		{
			name:    "postmeta",
			newMeta: repository.NewPostMeta,
			createOwner: func(t *testing.T) int64 {
				t.Helper()
				post := &model.Post{
					PostAuthor: 1, PostTitle: "Meta Post", PostName: t.Name(),
					PostContent: "x", PostStatus: "publish", PostType: "post",
				}
				if err := posts.Create(ctx, post); err != nil {
					t.Fatalf("create post: %v", err)
				}
				return post.ID
			},
		},
		{
			name:    "usermeta",
			newMeta: repository.NewUserMeta,
			createOwner: func(t *testing.T) int64 {
				t.Helper()
				user := &model.User{
					UserLogin: t.Name(), UserPass: "x",
					UserEmail: t.Name() + "@test.com",
				}
				if err := users.Create(ctx, user); err != nil {
					t.Fatalf("create user: %v", err)
				}
				return user.ID
			},
		},
		{
			name:    "commentmeta",
			newMeta: repository.NewCommentMeta,
			createOwner: func(t *testing.T) int64 {
				t.Helper()
				post := &model.Post{
					PostAuthor: 1, PostTitle: "CM", PostName: t.Name() + "-post",
					PostContent: "x", PostStatus: "publish", PostType: "post",
				}
				posts.Create(ctx, post)
				comment := &model.Comment{
					CommentPostID: post.ID, CommentAuthor: "Test",
					CommentContent: "test", CommentApproved: "1", CommentType: "comment",
				}
				if err := comments.Create(ctx, comment); err != nil {
					t.Fatalf("create comment: %v", err)
				}
				return comment.CommentID
			},
		},
		{
			name:    "termmeta",
			newMeta: repository.NewTermMeta,
			createOwner: func(t *testing.T) int64 {
				t.Helper()
				term := &model.Term{Name: t.Name(), Slug: t.Name()}
				tt := &model.TermTaxonomy{Taxonomy: "category"}
				if err := terms.Create(ctx, term, tt); err != nil {
					t.Fatalf("create term: %v", err)
				}
				return term.TermID
			},
		},
	}

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			meta := setup.newMeta(database)
			ownerID := setup.createOwner(t)

			t.Run("Set_and_Get", func(t *testing.T) {
				if err := meta.Set(ctx, ownerID, "key1", "value1"); err != nil {
					t.Fatalf("Set: %v", err)
				}
				m, err := meta.Get(ctx, ownerID, "key1")
				if err != nil {
					t.Fatalf("Get: %v", err)
				}
				if m.MetaValue != "value1" {
					t.Errorf("expected 'value1', got %q", m.MetaValue)
				}
				if m.ObjectID != ownerID {
					t.Errorf("expected ObjectID %d, got %d", ownerID, m.ObjectID)
				}
			})

			t.Run("Set_upsert", func(t *testing.T) {
				meta.Set(ctx, ownerID, "upsert_key", "original")
				meta.Set(ctx, ownerID, "upsert_key", "updated")
				m, _ := meta.Get(ctx, ownerID, "upsert_key")
				if m.MetaValue != "updated" {
					t.Errorf("expected 'updated', got %q", m.MetaValue)
				}
			})

			t.Run("Get_not_found", func(t *testing.T) {
				_, err := meta.Get(ctx, ownerID, "nonexistent")
				if !errors.IsCode(err, errors.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			})

			t.Run("GetAll", func(t *testing.T) {
				meta.Set(ctx, ownerID, "all_a", "1")
				meta.Set(ctx, ownerID, "all_b", "2")
				all, err := meta.GetAll(ctx, ownerID)
				if err != nil {
					t.Fatalf("GetAll: %v", err)
				}
				if len(all) < 2 {
					t.Errorf("expected at least 2 metas, got %d", len(all))
				}
			})

			t.Run("Delete", func(t *testing.T) {
				meta.Set(ctx, ownerID, "del_key", "val")
				if err := meta.Delete(ctx, ownerID, "del_key"); err != nil {
					t.Fatalf("Delete: %v", err)
				}
				_, err := meta.Get(ctx, ownerID, "del_key")
				if !errors.IsCode(err, errors.ErrNotFound) {
					t.Errorf("expected ErrNotFound after delete, got %v", err)
				}
			})

			t.Run("Delete_not_found", func(t *testing.T) {
				err := meta.Delete(ctx, ownerID, "nonexistent")
				if !errors.IsCode(err, errors.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			})

			t.Run("DeleteAll", func(t *testing.T) {
				meta.Set(ctx, ownerID, "da_a", "1")
				meta.Set(ctx, ownerID, "da_b", "2")
				if err := meta.DeleteAll(ctx, ownerID); err != nil {
					t.Fatalf("DeleteAll: %v", err)
				}
				all, _ := meta.GetAll(ctx, ownerID)
				if len(all) != 0 {
					t.Errorf("expected 0 metas after DeleteAll, got %d", len(all))
				}
			})
		})
	}
}

func TestTermMetaCascade(t *testing.T) {
	database := db.SetupTestDB(t)
	ctx := context.Background()
	terms := repository.NewTermsRepository(database)
	meta := repository.NewTermMeta(database)

	term := &model.Term{Name: "Doomed", Slug: "doomed-cascade"}
	tt := &model.TermTaxonomy{Taxonomy: "category"}
	terms.Create(ctx, term, tt)

	meta.Set(ctx, term.TermID, "k1", "v1")
	meta.Set(ctx, term.TermID, "k2", "v2")

	terms.Delete(ctx, term.TermID)

	all, _ := meta.GetAll(ctx, term.TermID)
	if len(all) != 0 {
		t.Errorf("expected 0 metas after term delete, got %d", len(all))
	}
}
