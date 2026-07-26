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

func newSnapCtx(t *testing.T) (repository.Repository, string, string) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	c := &entity.Card{
		ID: ulidutil.New(), Title: "card", Content: "x", Format: "markdown",
		GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	return repo, g.ID, c.ID
}

func mkSnap(t *testing.T, repo repository.Repository, cardID string, conv *string) *entity.CardSnapshot {
	t.Helper()
	ctx := context.Background()
	seq, err := repo.NextSnapshotSeq(ctx, cardID)
	if err != nil {
		t.Fatalf("NextSnapshotSeq: %v", err)
	}
	s := &entity.CardSnapshot{
		ID: ulidutil.New(), CardID: cardID, Seq: seq,
		Title: "card", Content: "x", Format: "markdown",
		Actor: "agent", ConversationID: conv, ChangeKind: "update",
		CreatedAt: time.Now(),
	}
	if err := repo.CreateCardSnapshot(ctx, s); err != nil {
		t.Fatalf("CreateCardSnapshot: %v", err)
	}
	return s
}

func TestSnapshot_SeqIncrementsPerCard(t *testing.T) {
	repo, _, cardID := newSnapCtx(t)
	first := mkSnap(t, repo, cardID, nil)
	second := mkSnap(t, repo, cardID, nil)
	if first.Seq != 1 || second.Seq != 2 {
		t.Fatalf("seq = %d,%d, want 1,2", first.Seq, second.Seq)
	}
}

func TestSnapshot_ListRequiresAFilter(t *testing.T) {
	repo, _, _ := newSnapCtx(t)
	if _, err := repo.ListCardSnapshots(context.Background(), api.ListSnapshotsRequest{}); err == nil {
		t.Fatalf("expected an error when no filter is given")
	}
}

func TestSnapshot_ListByConversationIsOldestFirst(t *testing.T) {
	repo, _, cardID := newSnapCtx(t)
	conv := ulidutil.New()
	a := mkSnap(t, repo, cardID, &conv)
	b := mkSnap(t, repo, cardID, &conv)

	got, err := repo.ListCardSnapshots(context.Background(),
		api.ListSnapshotsRequest{ConversationID: &conv})
	if err != nil {
		t.Fatalf("ListCardSnapshots: %v", err)
	}
	if len(got) != 2 || got[0].ID != a.ID || got[1].ID != b.ID {
		t.Fatalf("order wrong: got %d rows", len(got))
	}
	if got[0].CardTitle != "card" {
		t.Fatalf("CardTitle = %q, want %q", got[0].CardTitle, "card")
	}
}

func TestSnapshot_ListByCardIsNewestFirst(t *testing.T) {
	repo, _, cardID := newSnapCtx(t)
	mkSnap(t, repo, cardID, nil)
	second := mkSnap(t, repo, cardID, nil)

	got, err := repo.ListCardSnapshots(context.Background(),
		api.ListSnapshotsRequest{CardID: &cardID})
	if err != nil {
		t.Fatalf("ListCardSnapshots: %v", err)
	}
	if len(got) != 2 || got[0].ID != second.ID {
		t.Fatalf("card-filtered list must be newest-first; got[0].Seq = %d", got[0].Seq)
	}
}

// The schema cascade does not run in production (no foreign_keys pragma), so
// assert the explicit delete removes the rows. Calling it directly rather than
// purging keeps the test honest: purge would pass here on the pragma alone.
func TestSnapshot_DeleteSnapshotsForCard(t *testing.T) {
	repo, _, cardID := newSnapCtx(t)
	ctx := context.Background()
	mkSnap(t, repo, cardID, nil)
	mkSnap(t, repo, cardID, nil)

	if err := repo.DeleteSnapshotsForCard(ctx, cardID); err != nil {
		t.Fatalf("DeleteSnapshotsForCard: %v", err)
	}
	got, err := repo.ListCardSnapshots(ctx, api.ListSnapshotsRequest{CardID: &cardID})
	if err != nil {
		t.Fatalf("ListCardSnapshots: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d snapshots after delete, want 0", len(got))
	}
}

func TestSnapshot_GetBySeq(t *testing.T) {
	repo, _, cardID := newSnapCtx(t)
	ctx := context.Background()
	first := mkSnap(t, repo, cardID, nil)

	got, err := repo.GetCardSnapshotBySeq(ctx, cardID, 1)
	if err != nil {
		t.Fatalf("GetCardSnapshotBySeq: %v", err)
	}
	if got.ID != first.ID {
		t.Fatalf("got %q, want %q", got.ID, first.ID)
	}
	if _, err := repo.GetCardSnapshotBySeq(ctx, cardID, 99); err == nil {
		t.Fatalf("expected NotFound for a missing seq")
	}
}
