package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func TestV110Backfill_WritesOneBaselinePerCardAndIsIdempotent(t *testing.T) {
	repo := repository.New(testutil.NewDB(t))
	ctx := context.Background()

	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	updated := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	c := &entity.Card{ID: ulidutil.New(), Title: "old", Content: "body", Format: "markdown",
		GroupID: g.ID, CreatedAt: updated, UpdatedAt: updated}
	if err := repo.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	for i := 0; i < 2; i++ { // twice: the backfill runs on every startup
		if err := repo.RunV110Backfill(ctx); err != nil {
			t.Fatalf("RunV110Backfill (pass %d): %v", i, err)
		}
	}

	got, err := repo.ListCardSnapshots(ctx, api.ListSnapshotsRequest{CardID: &c.ID})
	if err != nil {
		t.Fatalf("ListCardSnapshots: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want exactly 1 baseline after two passes, got %d", len(got))
	}
	if got[0].ChangeKind != "baseline" || got[0].Actor != "system" {
		t.Fatalf("unexpected baseline: kind=%q actor=%q", got[0].ChangeKind, got[0].Actor)
	}
	if got[0].Content != "body" || got[0].Seq != 1 {
		t.Fatalf("baseline must copy current state at seq 1: %+v", got[0])
	}
	if !got[0].CreatedAt.UTC().Truncate(time.Second).Equal(updated) {
		t.Fatalf("created_at = %v, want the card's updated_at %v", got[0].CreatedAt, updated)
	}
	// Backfilled ids must be real ULIDs: snapshot.Get parses the id, so a
	// non-ULID would make every pre-upgrade baseline unfetchable.
	if err := ulidutil.Parse(got[0].ID); err != nil {
		t.Fatalf("backfilled id is not a ulid: %v", err)
	}
}

func TestV110Backfill_SkipsCardsThatAlreadyHaveSnapshots(t *testing.T) {
	repo := repository.New(testutil.NewDB(t))
	ctx := context.Background()

	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	c := &entity.Card{ID: ulidutil.New(), Title: "t", Content: "v2", Format: "markdown",
		GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	existing := &entity.CardSnapshot{ID: ulidutil.New(), CardID: c.ID, Seq: 1, Title: "t",
		Content: "v1", Format: "markdown", Actor: "user", ChangeKind: "create", CreatedAt: time.Now()}
	if err := repo.CreateCardSnapshot(ctx, existing); err != nil {
		t.Fatalf("CreateCardSnapshot: %v", err)
	}

	if err := repo.RunV110Backfill(ctx); err != nil {
		t.Fatalf("RunV110Backfill: %v", err)
	}

	got, err := repo.ListCardSnapshots(ctx, api.ListSnapshotsRequest{CardID: &c.ID})
	if err != nil {
		t.Fatalf("ListCardSnapshots: %v", err)
	}
	if len(got) != 1 || got[0].ID != existing.ID {
		t.Fatalf("a card with history must be left alone; got %d rows", len(got))
	}
}
