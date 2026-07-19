package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func newRepo(t *testing.T) repository.Repository {
	t.Helper()
	return repository.New(testutil.NewDB(t))
}

func TestCreateAndGetGroup(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	g := &entity.Group{
		ID:        ulidutil.New(),
		Name:      "work",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	got, err := r.GetGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.Name != "work" {
		t.Fatalf("got name %q, want %q", got.Name, "work")
	}
}

func TestGroup_RoundTrip_PreservesRule(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	g := &entity.Group{
		ID:        ulidutil.New(),
		Name:      "design",
		Rule:      "Cards here must be translated into Chinese and formatted as HTML.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	got, err := r.GetGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.Rule != g.Rule {
		t.Fatalf("got rule %q, want %q", got.Rule, g.Rule)
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	r := newRepo(t)
	_, err := r.GetGroup(context.Background(), ulidutil.New())
	if err == nil {
		t.Fatalf("expected error for missing group, got nil")
	}
}

func TestGroup_NameUniqueGlobally(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	now := time.Now()
	first := &entity.Group{ID: ulidutil.New(), Name: "design", CreatedAt: now, UpdatedAt: now}
	if err := r.CreateGroup(ctx, first); err != nil {
		t.Fatalf("CreateGroup first: %v", err)
	}
	dup := &entity.Group{ID: ulidutil.New(), Name: "design", CreatedAt: now, UpdatedAt: now}
	if err := r.CreateGroup(ctx, dup); err == nil {
		t.Fatal("expected UNIQUE constraint failure on second insert")
	}
}

func TestListGroups_FlatOrdering(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	for i, name := range []string{"alpha", "beta", "gamma"} {
		g := &entity.Group{
			ID: ulidutil.New(), Name: name, Position: i,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := r.CreateGroup(ctx, g); err != nil {
			t.Fatalf("CreateGroup %s: %v", name, err)
		}
	}
	got, err := r.ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d groups, want 3", len(got))
	}
}

func TestUpdateGroup(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	g.Name = "personal"
	g.UpdatedAt = time.Now()
	if err := r.UpdateGroup(ctx, g); err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	got, err := r.GetGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.Name != "personal" {
		t.Fatalf("got %q, want %q", got.Name, "personal")
	}
}

func TestDeleteGroup(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := r.DeleteGroup(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if _, err := r.GetGroup(ctx, g.ID); err == nil {
		t.Fatalf("expected NotFound after delete")
	}
}

func TestGroupHasContent_False(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	has, err := r.GroupHasContent(ctx, g.ID)
	if err != nil {
		t.Fatalf("GroupHasContent: %v", err)
	}
	if has {
		t.Fatalf("empty group reported as having content")
	}
}

func TestGroup_RoundTrip_PreservesLevelCatalog(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	now := time.Now()
	g := &entity.Group{
		ID: ulidutil.New(), Name: "design",
		LevelCatalog: []entity.LevelEntry{
			{Weight: 0, Name: "原则"},
			{Weight: 0.5, Name: "模式"},
			{Weight: 1, Name: "决策"},
		},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	got, err := r.GetGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if len(got.LevelCatalog) != 3 {
		t.Fatalf("catalog length: got %d want 3 (%+v)", len(got.LevelCatalog), got.LevelCatalog)
	}
	if got.LevelCatalog[1].Weight != 0.5 || got.LevelCatalog[1].Name != "模式" {
		t.Fatalf("entry[1] = %+v", got.LevelCatalog[1])
	}
}

func TestGroup_DefaultCatalogIsEmptySlice(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	now := time.Now()
	g := &entity.Group{ID: ulidutil.New(), Name: "x", CreatedAt: now, UpdatedAt: now}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	got, _ := r.GetGroup(ctx, g.ID)
	if got.LevelCatalog == nil {
		t.Fatal("LevelCatalog should be empty slice, not nil")
	}
	if len(got.LevelCatalog) != 0 {
		t.Fatalf("expected empty catalog, got %+v", got.LevelCatalog)
	}
}
