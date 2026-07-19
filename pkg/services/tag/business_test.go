package tag_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// newTagSvc builds a tag service over a fresh DB seeded with one group, and
// returns the repo too so a test can add a second group in the same DB.
func newTagSvc(t *testing.T) (svc model.Tag, repo repository.Repository, groupID string) {
	t.Helper()
	repo = repository.New(testutil.NewDB(t))
	groupID = ulidutil.New()
	if err := repo.CreateGroup(context.Background(), &entity.Group{
		ID: groupID, Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	return tag.New(repo), repo, groupID
}

func TestTag_EnsureByName_CreatesOnFirstCall(t *testing.T) {
	svc, _, gid := newTagSvc(t)
	got, err := svc.EnsureByName(context.Background(), gid, "Go")
	if err != nil {
		t.Fatalf("EnsureByName: %v", err)
	}
	if got.Name != "go" {
		t.Fatalf("expected normalized name 'go', got %q", got.Name)
	}
}

func TestTag_EnsureByName_IdempotentOnSecondCall(t *testing.T) {
	svc, _, gid := newTagSvc(t)
	a, _ := svc.EnsureByName(context.Background(), gid, "go")
	b, err := svc.EnsureByName(context.Background(), gid, "GO")
	if err != nil {
		t.Fatalf("second EnsureByName: %v", err)
	}
	if a.ID != b.ID {
		t.Fatalf("expected same id on idempotent ensure; got %q vs %q", a.ID, b.ID)
	}
}

func TestTag_Rename_MergesOnCollision(t *testing.T) {
	svc, _, gid := newTagSvc(t)
	_, _ = svc.EnsureByName(context.Background(), gid, "golang")
	_, _ = svc.EnsureByName(context.Background(), gid, "go")
	merged, err := svc.Rename(context.Background(), gid, "golang", "go")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if merged.Name != "go" {
		t.Fatalf("expected merged name 'go', got %q", merged.Name)
	}
	all, _ := svc.List(context.Background(), gid)
	if len(all) != 1 {
		t.Fatalf("expected 1 tag after merge, got %d", len(all))
	}
}

func TestTag_Create_EmptyName(t *testing.T) {
	svc, _, gid := newTagSvc(t)
	_, err := svc.EnsureByName(context.Background(), gid, "")
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}

func TestTag_Delete_RemovesTag(t *testing.T) {
	svc, _, gid := newTagSvc(t)
	_, _ = svc.EnsureByName(context.Background(), gid, "throwaway")
	if err := svc.Delete(context.Background(), gid, "throwaway"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	all, _ := svc.List(context.Background(), gid)
	if len(all) != 0 {
		t.Fatalf("expected 0 tags after delete, got %d", len(all))
	}
}

func TestTag_EnsureByName_ScopedToGroup(t *testing.T) {
	svc, repo, gA := newTagSvc(t)
	ctx := context.Background()
	gB := ulidutil.New()
	if err := repo.CreateGroup(ctx, &entity.Group{ID: gB, Name: "B", CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("group B: %v", err)
	}

	a1, err := svc.EnsureByName(ctx, gA, "Draft")
	if err != nil {
		t.Fatalf("ensure A: %v", err)
	}
	a2, _ := svc.EnsureByName(ctx, gA, "draft") // normalized → same tag
	if a1.ID != a2.ID {
		t.Fatalf("expected same tag for A/draft, got %s vs %s", a1.ID, a2.ID)
	}
	b1, _ := svc.EnsureByName(ctx, gB, "draft") // other group → independent
	if b1.ID == a1.ID {
		t.Fatalf("expected independent tag in group B")
	}
	listA, _ := svc.List(ctx, gA)
	if len(listA) != 1 {
		t.Fatalf("group A should have exactly 1 tag, got %d", len(listA))
	}
}
