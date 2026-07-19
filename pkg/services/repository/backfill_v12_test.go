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

// backfillCtx wires a repo with a single card + optional card-anchored
// conversation with N user messages, spaced 1 second apart.
type backfillCtx struct {
	repo    repository.Repository
	cardID  string
	lastMsg time.Time
}

func newCardWithNUserMessages(t *testing.T, n int) *backfillCtx {
	t.Helper()
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	card := &entity.Card{
		ID: ulidutil.New(), Title: "target", Content: "body", GroupID: g.ID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, card); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if n == 0 {
		return &backfillCtx{repo: repo, cardID: card.ID}
	}
	anchorKind := "card"
	convo := &entity.Conversation{
		ID: ulidutil.New(), AnchorKind: &anchorKind, AnchorID: &card.ID,
		CreatedAt: time.Now(), LastMessageAt: time.Now(),
	}
	if err := repo.CreateConversation(ctx, convo); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	base := time.Now().UTC().Add(-time.Duration(n) * time.Second).Truncate(time.Second)
	var last time.Time
	for i := 0; i < n; i++ {
		at := base.Add(time.Duration(i) * time.Second)
		msg := &entity.Message{
			ID: ulidutil.New(), ConversationID: convo.ID, Role: "user",
			Content: "q", CreatedAt: at,
		}
		if err := repo.CreateMessage(ctx, msg); err != nil {
			t.Fatalf("CreateMessage: %v", err)
		}
		last = at
	}
	return &backfillCtx{repo: repo, cardID: card.ID, lastMsg: last}
}

func TestBackfill_CardWithNoConversation_StaysLGTM(t *testing.T) {
	b := newCardWithNUserMessages(t, 0)
	if err := b.repo.RunV12Backfill(context.Background()); err != nil {
		t.Fatalf("RunV12Backfill: %v", err)
	}
	got, err := b.repo.GetCard(context.Background(), b.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "LGTM" {
		t.Fatalf("expected LGTM, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt != nil {
		t.Fatalf("expected reviewed_at nil, got %v", *got.ReviewedAt)
	}
}

func TestBackfill_CardWithOneUserMessage_BecomesDigested(t *testing.T) {
	b := newCardWithNUserMessages(t, 1)
	if err := b.repo.RunV12Backfill(context.Background()); err != nil {
		t.Fatalf("RunV12Backfill: %v", err)
	}
	got, err := b.repo.GetCard(context.Background(), b.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "DIGESTED" {
		t.Fatalf("expected DIGESTED, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at non-nil")
	}
}

func TestBackfill_CardWithThreeUserMessages_BecomesGrilled(t *testing.T) {
	b := newCardWithNUserMessages(t, 3)
	if err := b.repo.RunV12Backfill(context.Background()); err != nil {
		t.Fatalf("RunV12Backfill: %v", err)
	}
	got, err := b.repo.GetCard(context.Background(), b.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "GRILLED" {
		t.Fatalf("expected GRILLED, got %q", got.ReviewGrade)
	}
}

func TestBackfill_ReviewedAt_MatchesLastQualifyingMessage(t *testing.T) {
	b := newCardWithNUserMessages(t, 2)
	if err := b.repo.RunV12Backfill(context.Background()); err != nil {
		t.Fatalf("RunV12Backfill: %v", err)
	}
	got, err := b.repo.GetCard(context.Background(), b.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at non-nil")
	}
	if diff := got.ReviewedAt.Sub(b.lastMsg).Abs(); diff > time.Second {
		t.Fatalf("expected reviewed_at close to %v, got %v (diff %v)", b.lastMsg, *got.ReviewedAt, diff)
	}
}

func TestBackfill_Idempotent(t *testing.T) {
	b := newCardWithNUserMessages(t, 3)
	ctx := context.Background()
	if err := b.repo.RunV12Backfill(ctx); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := b.repo.RunV12Backfill(ctx); err != nil {
		t.Fatalf("second: %v", err)
	}
	got, err := b.repo.GetCard(ctx, b.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "GRILLED" {
		t.Fatalf("expected GRILLED after 2 backfill passes, got %q", got.ReviewGrade)
	}
}

func TestBackfill_ManualGrilledAboveFloor_Untouched(t *testing.T) {
	// If a card is already GRILLED but has only 1 message (floor DIGESTED),
	// backfill must NOT lower it.
	b := newCardWithNUserMessages(t, 1)
	ctx := context.Background()
	card, err := b.repo.GetCard(ctx, b.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	card.ReviewGrade = "GRILLED"
	now := time.Now().UTC()
	card.ReviewedAt = &now
	if err := b.repo.UpdateCard(ctx, card); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	if err := b.repo.RunV12Backfill(ctx); err != nil {
		t.Fatalf("RunV12Backfill: %v", err)
	}
	got, err := b.repo.GetCard(ctx, b.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "GRILLED" {
		t.Fatalf("expected GRILLED unchanged, got %q", got.ReviewGrade)
	}
}
