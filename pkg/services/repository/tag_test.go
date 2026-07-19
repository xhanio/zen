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

// newGroupID creates a group and returns its id. Tags now carry a NOT NULL
// group_id FK, so every tag test needs a real group to hang tags on.
func newGroupID(t *testing.T, r repository.Repository) string {
	t.Helper()
	gid := ulidutil.New()
	if err := r.CreateGroup(context.Background(), &entity.Group{
		ID: gid, Name: "g-" + gid[:6], CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	return gid
}

func TestCreateAndGetTagByName(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	gid := newGroupID(t, r)
	tag := &entity.Tag{ID: ulidutil.New(), GroupID: gid, Name: "go"}
	if err := r.CreateTag(ctx, tag); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	got, err := r.GetTagByNameInGroup(ctx, gid, "go")
	if err != nil {
		t.Fatalf("GetTagByNameInGroup: %v", err)
	}
	if got.ID != tag.ID {
		t.Fatalf("got id %q, want %q", got.ID, tag.ID)
	}
}

func TestGetTagByName_NotFound(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	gid := newGroupID(t, r)
	_, err := r.GetTagByNameInGroup(context.Background(), gid, "missing")
	if err == nil {
		t.Fatalf("expected error for missing tag")
	}
}

func TestListTags(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	gid := newGroupID(t, r)
	for _, name := range []string{"go", "rust", "python"} {
		_ = r.CreateTag(ctx, &entity.Tag{ID: ulidutil.New(), GroupID: gid, Name: name})
	}
	tags, err := r.ListTags(ctx, gid)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("got %d, want 3", len(tags))
	}
}

func TestDeleteTag(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	gid := newGroupID(t, r)
	tag := &entity.Tag{ID: ulidutil.New(), GroupID: gid, Name: "ephemeral"}
	_ = r.CreateTag(ctx, tag)
	if err := r.DeleteTag(ctx, tag.ID); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}
	if _, err := r.GetTag(ctx, tag.ID); err == nil {
		t.Fatalf("expected NotFound after delete")
	}
}

func TestListTags_PopulatesCardCount(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	gid := newGroupID(t, repo)

	tagGo := &entity.Tag{ID: ulidutil.New(), GroupID: gid, Name: "go"}
	tagRust := &entity.Tag{ID: ulidutil.New(), GroupID: gid, Name: "rust"}
	if err := repo.CreateTag(ctx, tagGo); err != nil {
		t.Fatalf("CreateTag go: %v", err)
	}
	if err := repo.CreateTag(ctx, tagRust); err != nil {
		t.Fatalf("CreateTag rust: %v", err)
	}

	cid := ulidutil.New()
	if err := repo.CreateCard(ctx, &entity.Card{ID: cid, Title: "t", GroupID: gid, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if err := repo.AttachTag(ctx, cid, tagGo.ID); err != nil {
		t.Fatalf("AttachTag: %v", err)
	}

	tags, err := repo.ListTags(ctx, gid)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	got := map[string]int{}
	for _, x := range tags {
		got[x.Name] = x.CardCount
	}
	if got["go"] != 1 {
		t.Fatalf("expected go=1, got %d", got["go"])
	}
	if got["rust"] != 0 {
		t.Fatalf("expected rust=0, got %d", got["rust"])
	}
}

func TestListTags_ExcludesTrashedCards(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	gid := newGroupID(t, repo)

	tagGo := &entity.Tag{ID: ulidutil.New(), GroupID: gid, Name: "go"}
	if err := repo.CreateTag(ctx, tagGo); err != nil {
		t.Fatalf("CreateTag go: %v", err)
	}

	// Two cards, both tagged, then trash one.
	live := ulidutil.New()
	trashed := ulidutil.New()
	for _, cid := range []string{live, trashed} {
		if err := repo.CreateCard(ctx, &entity.Card{ID: cid, Title: "t", GroupID: gid, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
			t.Fatalf("CreateCard: %v", err)
		}
		if err := repo.AttachTag(ctx, cid, tagGo.ID); err != nil {
			t.Fatalf("AttachTag: %v", err)
		}
	}
	if _, err := repo.SoftDelete(ctx, trashed, false); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	tags, err := repo.ListTags(ctx, gid)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	var count int
	for _, x := range tags {
		if x.Name == "go" {
			count = x.CardCount
		}
	}
	if count != 1 {
		t.Fatalf("expected go=1 (trashed card excluded), got %d", count)
	}
}

func TestTag_ScopedByGroup(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	now := time.Now()

	gA := &entity.Group{ID: ulidutil.New(), Name: "A", CreatedAt: now, UpdatedAt: now}
	gB := &entity.Group{ID: ulidutil.New(), Name: "B", CreatedAt: now, UpdatedAt: now}
	if err := r.CreateGroup(ctx, gA); err != nil {
		t.Fatalf("group A: %v", err)
	}
	if err := r.CreateGroup(ctx, gB); err != nil {
		t.Fatalf("group B: %v", err)
	}

	// same name in two groups = two independent tags
	tA := &entity.Tag{ID: ulidutil.New(), GroupID: gA.ID, Name: "draft"}
	tB := &entity.Tag{ID: ulidutil.New(), GroupID: gB.ID, Name: "draft"}
	if err := r.CreateTag(ctx, tA); err != nil {
		t.Fatalf("tag A: %v", err)
	}
	if err := r.CreateTag(ctx, tB); err != nil {
		t.Fatalf("tag B (same name, other group) should be allowed: %v", err)
	}

	gotA, err := r.GetTagByNameInGroup(ctx, gA.ID, "draft")
	if err != nil || gotA.ID != tA.ID {
		t.Fatalf("GetTagByNameInGroup A: %v (%+v)", err, gotA)
	}
	listA, err := r.ListTags(ctx, gA.ID)
	if err != nil {
		t.Fatalf("ListTags A: %v", err)
	}
	if len(listA) != 1 || listA[0].Name != "draft" || listA[0].GroupID != gA.ID {
		t.Fatalf("ListTags A scoped wrong: %+v", listA)
	}
}
