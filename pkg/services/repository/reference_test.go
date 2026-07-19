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

func newRefCtx(t *testing.T) (repo repository.Repository, srcID, derivedID, convID string) {
	t.Helper()
	repo = repository.New(testutil.NewDB(t))
	ctx := context.Background()
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	mkCard := func(title string) string {
		c := &entity.Card{
			ID: ulidutil.New(), Title: title, Content: "x", Format: "markdown",
			GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := repo.CreateCard(ctx, c); err != nil {
			t.Fatalf("CreateCard: %v", err)
		}
		return c.ID
	}
	srcID, derivedID = mkCard("source"), mkCard("derived")
	conv := &entity.Conversation{ID: ulidutil.New(), Title: "", CreatedAt: time.Now(), LastMessageAt: time.Now()}
	if err := repo.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	convID = conv.ID
	return
}

func TestReference_RoundTrip(t *testing.T) {
	repo, src, der, conv := newRefCtx(t)
	ctx := context.Background()
	r := &entity.Reference{
		ID: ulidutil.New(), SourceCardID: src, DerivedCardID: der, ConversationID: &conv,
		SelectionText: "hello", CreatedAt: time.Now(),
	}
	if err := repo.CreateReference(ctx, r); err != nil {
		t.Fatalf("CreateReference: %v", err)
	}
	got, err := repo.GetReference(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetReference: %v", err)
	}
	if got.SourceCardID != src || got.DerivedCardID != der || got.ConversationID == nil || *got.ConversationID != conv || got.SelectionText != "hello" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestReference_ListBySource(t *testing.T) {
	repo, src, der, conv := newRefCtx(t)
	ctx := context.Background()
	r := &entity.Reference{
		ID: ulidutil.New(), SourceCardID: src, DerivedCardID: der, ConversationID: &conv,
		SelectionText: "x", CreatedAt: time.Now(),
	}
	if err := repo.CreateReference(ctx, r); err != nil {
		t.Fatalf("CreateReference: %v", err)
	}
	got, err := repo.ListReferences(ctx, api.ListReferencesRequest{SourceCardID: &src})
	if err != nil {
		t.Fatalf("ListReferences: %v", err)
	}
	if len(got) != 1 || got[0].ID != r.ID {
		t.Fatalf("expected 1 result, got %+v", got)
	}
}

func TestReference_Delete(t *testing.T) {
	repo, src, der, conv := newRefCtx(t)
	ctx := context.Background()
	r := &entity.Reference{
		ID: ulidutil.New(), SourceCardID: src, DerivedCardID: der, ConversationID: &conv,
		SelectionText: "y", CreatedAt: time.Now(),
	}
	_ = repo.CreateReference(ctx, r)
	if err := repo.DeleteReference(ctx, r.ID); err != nil {
		t.Fatalf("DeleteReference: %v", err)
	}
	if _, err := repo.GetReference(ctx, r.ID); err == nil {
		t.Fatalf("expected NotFound after delete")
	}
}

func TestReference_RoundTrip_NullConversation(t *testing.T) {
	repo, src, der, _ := newRefCtx(t)
	ctx := context.Background()
	r := &entity.Reference{
		ID: ulidutil.New(), SourceCardID: src, DerivedCardID: der,
		ConversationID: nil,
		SelectionText:  "anchor", CreatedAt: time.Now(),
	}
	if err := repo.CreateReference(ctx, r); err != nil {
		t.Fatalf("CreateReference: %v", err)
	}
	got, err := repo.GetReference(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetReference: %v", err)
	}
	if got.ConversationID != nil {
		t.Fatalf("expected ConversationID nil after round-trip, got %v", *got.ConversationID)
	}
	if got.SelectionText != "anchor" {
		t.Fatalf("bad selection_text: %q", got.SelectionText)
	}
}
