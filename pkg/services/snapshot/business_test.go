package snapshot_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/snapshot"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func newSnapshotSvc(t *testing.T) (snapshot.Manager, repository.Repository, string) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	ctx := context.Background()

	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	c := &entity.Card{ID: ulidutil.New(), Title: "c", Content: "v2", Format: "markdown",
		GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	return snapshot.New(repo), repo, c.ID
}

func TestGet_ReturnsPreviousSnapshot(t *testing.T) {
	svc, repo, cardID := newSnapshotSvc(t)
	ctx := context.Background()

	first := &entity.CardSnapshot{ID: ulidutil.New(), CardID: cardID, Seq: 1, Title: "c",
		Content: "v1", Format: "markdown", Actor: "user", ChangeKind: "create", CreatedAt: time.Now()}
	second := &entity.CardSnapshot{ID: ulidutil.New(), CardID: cardID, Seq: 2, Title: "c",
		Content: "v2", Format: "markdown", Actor: "agent", ChangeKind: "update", CreatedAt: time.Now()}
	for _, s := range []*entity.CardSnapshot{first, second} {
		if err := repo.CreateCardSnapshot(ctx, s); err != nil {
			t.Fatalf("CreateCardSnapshot: %v", err)
		}
	}

	got, err := svc.Get(ctx, second.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Previous == nil || got.Previous.Content != "v1" {
		t.Fatalf("previous snapshot missing or wrong: %+v", got.Previous)
	}

	base, err := svc.Get(ctx, first.ID)
	if err != nil {
		t.Fatalf("Get baseline: %v", err)
	}
	if base.Previous != nil {
		t.Fatalf("a baseline has no predecessor, got %+v", base.Previous)
	}
}

func TestList_RejectsAMalformedFilter(t *testing.T) {
	svc, _, _ := newSnapshotSvc(t)
	bad := "not-a-ulid"
	if _, err := svc.List(context.Background(), api.ListSnapshotsRequest{CardID: &bad}); err == nil {
		t.Fatalf("expected a validation error for a non-ulid filter")
	}
}

func TestList_ReturnsEmptySliceNotNil(t *testing.T) {
	svc, _, _ := newSnapshotSvc(t)
	other := ulidutil.New()
	got, err := svc.List(context.Background(), api.ListSnapshotsRequest{CardID: &other})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Fatalf("List must return an empty slice, not nil — it serializes as [] not null")
	}
}
