package card_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/services/card"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// seedReorderCtx creates a group, a parent card, and `n` live children at
// positions 0..n-1. Returns the card service and the ordered child IDs.
func seedReorderCtx(t *testing.T, n int) (svc model.Card, repo repository.Repository, parentID string, childIDs []string) {
	t.Helper()
	ctx := context.Background()
	repo = repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	parent := &entity.Card{ID: ulidutil.New(), Title: "P", GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateCard(ctx, parent); err != nil {
		t.Fatalf("CreateCard parent: %v", err)
	}
	parentID = parent.ID
	pid := parent.ID
	for i := 0; i < n; i++ {
		id := ulidutil.New()
		c := &entity.Card{
			ID: id, Title: string(rune('A' + i)),
			GroupID: g.ID, ParentCardID: &pid, Position: i,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := repo.CreateCard(ctx, c); err != nil {
			t.Fatalf("CreateCard child %d: %v", i, err)
		}
		childIDs = append(childIDs, id)
	}
	svc = card.New(repo, tag.New(repo), nil)
	return
}

func positionsByID(t *testing.T, r repository.Repository, ids []string) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, id := range ids {
		c, err := r.GetCard(context.Background(), id)
		if err != nil {
			t.Fatalf("GetCard %s: %v", id, err)
		}
		out[id] = c.Position
	}
	return out
}

func TestReorder_MoveMiddleToStart(t *testing.T) {
	svc, r, _, ids := seedReorderCtx(t, 5)
	if _, err := svc.Reorder(context.Background(), ids[2], 0); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	got := positionsByID(t, r, ids)
	want := map[string]int{ids[0]: 1, ids[1]: 2, ids[2]: 0, ids[3]: 3, ids[4]: 4}
	for id, w := range want {
		if got[id] != w {
			t.Fatalf("card %s: position = %d, want %d (all: %+v)", id, got[id], w, got)
		}
	}
}

func TestReorder_MoveStartToEnd(t *testing.T) {
	svc, r, _, ids := seedReorderCtx(t, 5)
	if _, err := svc.Reorder(context.Background(), ids[0], 4); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	got := positionsByID(t, r, ids)
	want := map[string]int{ids[0]: 4, ids[1]: 0, ids[2]: 1, ids[3]: 2, ids[4]: 3}
	for id, w := range want {
		if got[id] != w {
			t.Fatalf("card %s: position = %d, want %d", id, got[id], w)
		}
	}
}

func TestReorder_NoOpWhenSamePosition(t *testing.T) {
	svc, r, _, ids := seedReorderCtx(t, 3)
	if _, err := svc.Reorder(context.Background(), ids[1], 1); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	got := positionsByID(t, r, ids)
	if got[ids[0]] != 0 || got[ids[1]] != 1 || got[ids[2]] != 2 {
		t.Fatalf("positions changed on no-op: %+v", got)
	}
}

func TestReorder_PositionClampedToUpperBound(t *testing.T) {
	svc, r, _, ids := seedReorderCtx(t, 3)
	if _, err := svc.Reorder(context.Background(), ids[0], 999); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	got := positionsByID(t, r, ids)
	if got[ids[0]] != 2 || got[ids[1]] != 0 || got[ids[2]] != 1 {
		t.Fatalf("clamp failed: %+v", got)
	}
}

func TestReorder_RejectsTopLevelCard(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	c := &entity.Card{ID: ulidutil.New(), Title: "top", GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	svc := card.New(repo, tag.New(repo), nil)
	_, err := svc.Reorder(ctx, c.ID, 0)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for top-level card, got %v", err)
	}
}
