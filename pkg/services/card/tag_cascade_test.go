package card_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/card"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// tagCascadeCtx sets up a container with two live section children so we
// can verify tag cascade.
type tagCascadeCtx struct {
	svc         model.Card
	repo        repository.Repository
	containerID string
	childAID    string
	childBID    string
}

func newTagCascadeCtx(t *testing.T) *tagCascadeCtx {
	t.Helper()
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	parent := &entity.Card{
		ID: ulidutil.New(), Title: "container", GroupID: g.ID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, parent); err != nil {
		t.Fatalf("CreateCard container: %v", err)
	}
	pid := parent.ID
	childA := &entity.Card{
		ID: ulidutil.New(), Title: "A", Content: "body", GroupID: g.ID,
		ParentCardID: &pid, Position: 0,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	childB := &entity.Card{
		ID: ulidutil.New(), Title: "B", Content: "body", GroupID: g.ID,
		ParentCardID: &pid, Position: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, childA); err != nil {
		t.Fatalf("CreateCard A: %v", err)
	}
	if err := repo.CreateCard(ctx, childB); err != nil {
		t.Fatalf("CreateCard B: %v", err)
	}
	return &tagCascadeCtx{
		svc:         card.New(repo, tag.New(repo), nil),
		repo:        repo,
		containerID: parent.ID,
		childAID:    childA.ID,
		childBID:    childB.ID,
	}
}

func mustGetTags(t *testing.T, r repository.Repository, id string) []string {
	t.Helper()
	names, err := r.ListTagsForCard(context.Background(), id)
	if err != nil {
		t.Fatalf("ListTagsForCard %s: %v", id, err)
	}
	sort.Strings(names)
	return names
}

// TestCascade_EnsuresTagInDescendantGroup covers the per-group invariant: a
// descendant living in a DIFFERENT group than its container must receive a
// same-named tag owned by ITS group (not the container's tag id), so
// tag.group_id == card.group_id always holds.
func TestCascade_EnsuresTagInDescendantGroup(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	gA := &entity.Group{ID: ulidutil.New(), Name: "A", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	gB := &entity.Group{ID: ulidutil.New(), Name: "B", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, gA); err != nil {
		t.Fatalf("group A: %v", err)
	}
	if err := repo.CreateGroup(ctx, gB); err != nil {
		t.Fatalf("group B: %v", err)
	}
	svc := card.New(repo, tag.New(repo), nil)

	container := &entity.Card{ID: ulidutil.New(), Title: "C", GroupID: gA.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateCard(ctx, container); err != nil {
		t.Fatalf("container: %v", err)
	}
	pid := container.ID
	child := &entity.Card{ID: ulidutil.New(), Title: "child", GroupID: gB.ID, ParentCardID: &pid, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateCard(ctx, child); err != nil {
		t.Fatalf("child: %v", err)
	}

	// Add "reviewed" to the container → cascades to the child, ensured in B.
	if _, err := svc.Update(ctx, container.ID, nil, nil, nil, nil, &[]string{"reviewed"}, nil, nil, false, nil, nil); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := mustGetTags(t, repo, child.ID); !equalStringSlices(got, []string{"reviewed"}) {
		t.Fatalf("child did not get cascaded tag, got %v", got)
	}
	if _, err := repo.GetTagByNameInGroup(ctx, gB.ID, "reviewed"); err != nil {
		t.Fatalf("cascaded tag not owned by child's group B: %v", err)
	}
}

func TestUpdateTags_CascadesAddedTagsToDescendants(t *testing.T) {
	c := newTagCascadeCtx(t)
	ctx := context.Background()
	tags := []string{"v0.12", "review"}
	if _, err := c.svc.Update(ctx, c.containerID,
		nil, nil, nil, nil, &tags, nil, nil, false, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := mustGetTags(t, c.repo, c.containerID); !equalStringSlices(got, []string{"review", "v0.12"}) {
		t.Fatalf("container tags = %v", got)
	}
	if got := mustGetTags(t, c.repo, c.childAID); !equalStringSlices(got, []string{"review", "v0.12"}) {
		t.Fatalf("child A tags = %v", got)
	}
	if got := mustGetTags(t, c.repo, c.childBID); !equalStringSlices(got, []string{"review", "v0.12"}) {
		t.Fatalf("child B tags = %v", got)
	}
}

func TestUpdateTags_RemovalDoesNotCascade(t *testing.T) {
	c := newTagCascadeCtx(t)
	ctx := context.Background()
	// First cascade both tags down.
	both := []string{"v0.12", "review"}
	if _, err := c.svc.Update(ctx, c.containerID,
		nil, nil, nil, nil, &both, nil, nil, false, nil, nil); err != nil {
		t.Fatalf("Update 1: %v", err)
	}
	// Now drop "review" from the container only.
	trimmed := []string{"v0.12"}
	if _, err := c.svc.Update(ctx, c.containerID,
		nil, nil, nil, nil, &trimmed, nil, nil, false, nil, nil); err != nil {
		t.Fatalf("Update 2: %v", err)
	}
	if got := mustGetTags(t, c.repo, c.containerID); !equalStringSlices(got, []string{"v0.12"}) {
		t.Fatalf("container tags after removal = %v", got)
	}
	// Children keep the removed tag — removal isn't cascaded.
	if got := mustGetTags(t, c.repo, c.childAID); !equalStringSlices(got, []string{"review", "v0.12"}) {
		t.Fatalf("child A tags after container removal = %v", got)
	}
}

func TestUpdateTags_IdempotentOnDescendantsThatAlreadyHaveTag(t *testing.T) {
	c := newTagCascadeCtx(t)
	ctx := context.Background()
	// Pre-tag child A with "v0.12" directly.
	pre := []string{"v0.12"}
	if _, err := c.svc.Update(ctx, c.childAID,
		nil, nil, nil, nil, &pre, nil, nil, false, nil, nil); err != nil {
		t.Fatalf("Update childA: %v", err)
	}
	// Now cascade "v0.12" + "review" from the container.
	both := []string{"v0.12", "review"}
	if _, err := c.svc.Update(ctx, c.containerID,
		nil, nil, nil, nil, &both, nil, nil, false, nil, nil); err != nil {
		t.Fatalf("Update container: %v", err)
	}
	if got := mustGetTags(t, c.repo, c.childAID); !equalStringSlices(got, []string{"review", "v0.12"}) {
		t.Fatalf("child A tags = %v (should not error on duplicate v0.12)", got)
	}
}

func TestUpdateTags_LeafHasNoDescendantsSoNoCascade(t *testing.T) {
	c := newTagCascadeCtx(t)
	ctx := context.Background()
	tags := []string{"leafy"}
	if _, err := c.svc.Update(ctx, c.childAID,
		nil, nil, nil, nil, &tags, nil, nil, false, nil, nil); err != nil {
		t.Fatalf("Update leaf: %v", err)
	}
	// Sibling child B unaffected.
	if got := mustGetTags(t, c.repo, c.childBID); len(got) != 0 {
		t.Fatalf("sibling B tags = %v (should be empty)", got)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
