package conversation_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/conversation"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// autoupgradeCtx wires a fresh conversation service + repo, with a
// card-anchored conversation ready for AppendMessage. Returns the card ID
// (target of auto-upgrade) and the conversation ID.
type autoupgradeCtx struct {
	svc      conversation.Manager
	repo     repository.Repository
	cardID   string
	convoID  string
	groupID  string
}

func newAutoupgradeCtx(t *testing.T) *autoupgradeCtx {
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
	svc := conversation.New(repo)
	anchorKind := "card"
	c, err := svc.Create(ctx, "", &anchorKind, &card.ID)
	if err != nil {
		t.Fatalf("Create convo: %v", err)
	}
	return &autoupgradeCtx{svc: svc, repo: repo, cardID: card.ID, convoID: c.ID, groupID: g.ID}
}

func TestAutoUpgrade_ZeroQuestions_StaysLGTM(t *testing.T) {
	c := newAutoupgradeCtx(t)
	got, err := c.repo.GetCard(context.Background(), c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "LGTM" {
		t.Fatalf("expected LGTM, got %q", got.ReviewGrade)
	}
}

func TestAutoUpgrade_OneUserMessage_RaisesToDigested(t *testing.T) {
	c := newAutoupgradeCtx(t)
	if _, err := c.svc.AppendMessage(context.Background(), c.convoID, "user", "why?", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	got, err := c.repo.GetCard(context.Background(), c.cardID)
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

func TestAutoUpgrade_ThreeUserMessages_RaisesToGrilled(t *testing.T) {
	c := newAutoupgradeCtx(t)
	for i := 0; i < 3; i++ {
		if _, err := c.svc.AppendMessage(context.Background(), c.convoID, "user", "q", nil); err != nil {
			t.Fatalf("AppendMessage %d: %v", i, err)
		}
	}
	got, err := c.repo.GetCard(context.Background(), c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "GRILLED" {
		t.Fatalf("expected GRILLED, got %q", got.ReviewGrade)
	}
}

func TestAutoUpgrade_AssistantMessagesDoNotCount(t *testing.T) {
	c := newAutoupgradeCtx(t)
	for i := 0; i < 5; i++ {
		if _, err := c.svc.AppendMessage(context.Background(), c.convoID, "assistant", "answer", nil); err != nil {
			t.Fatalf("AppendMessage %d: %v", i, err)
		}
	}
	got, err := c.repo.GetCard(context.Background(), c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "LGTM" {
		t.Fatalf("expected LGTM (assistant messages skipped), got %q", got.ReviewGrade)
	}
}

func TestAutoUpgrade_ManualGrilledStays_WhenBelowThreshold(t *testing.T) {
	c := newAutoupgradeCtx(t)
	// Manually set to GRILLED first
	card, err := c.repo.GetCard(context.Background(), c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	card.ReviewGrade = "GRILLED"
	now := time.Now().UTC()
	card.ReviewedAt = &now
	if err := c.repo.UpdateCard(context.Background(), card); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	// A single user message would upgrade to DIGESTED; but current grade is
	// already above the floor, so stays GRILLED.
	if _, err := c.svc.AppendMessage(context.Background(), c.convoID, "user", "q", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	got, err := c.repo.GetCard(context.Background(), c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "GRILLED" {
		t.Fatalf("expected GRILLED to stay, got %q", got.ReviewGrade)
	}
}

func TestAutoUpgrade_ManualLGTMAfterAuto_ReUpgradesOnNextTrigger(t *testing.T) {
	c := newAutoupgradeCtx(t)
	ctx := context.Background()
	// First message → DIGESTED
	if _, err := c.svc.AppendMessage(ctx, c.convoID, "user", "q", nil); err != nil {
		t.Fatalf("AppendMessage 1: %v", err)
	}
	// Manual downgrade to LGTM
	card, err := c.repo.GetCard(ctx, c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	card.ReviewGrade = "LGTM"
	card.ReviewedAt = nil
	if err := c.repo.UpdateCard(ctx, card); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	// Another user message → count is now 2, floor is DIGESTED, re-upgrade
	if _, err := c.svc.AppendMessage(ctx, c.convoID, "user", "q2", nil); err != nil {
		t.Fatalf("AppendMessage 2: %v", err)
	}
	got, err := c.repo.GetCard(ctx, c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "DIGESTED" {
		t.Fatalf("expected DIGESTED after re-trigger, got %q", got.ReviewGrade)
	}
}

func TestAutoUpgrade_NonCardAnchoredConversation_NoEffect(t *testing.T) {
	// A group-anchored conversation must not touch any card's grade —
	// the auto-upgrade branch checks anchor_kind == "card" specifically.
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
	svc := conversation.New(repo)
	groupKind := "group"
	c, err := svc.Create(ctx, "", &groupKind, &g.ID)
	if err != nil {
		t.Fatalf("Create convo: %v", err)
	}
	if _, err := svc.AppendMessage(ctx, c.ID, "user", "q", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	got, err := repo.GetCard(ctx, card.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "LGTM" {
		t.Fatalf("expected LGTM (non-card anchor), got %q", got.ReviewGrade)
	}
}

func TestAutoUpgrade_SetsReviewedAt(t *testing.T) {
	c := newAutoupgradeCtx(t)
	before := time.Now().UTC()
	if _, err := c.svc.AppendMessage(context.Background(), c.convoID, "user", "q", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	got, err := c.repo.GetCard(context.Background(), c.cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at non-nil after auto-upgrade")
	}
	if diff := got.ReviewedAt.Sub(before).Abs(); diff > 2*time.Second {
		t.Fatalf("reviewed_at drifted %v from now", diff)
	}
}
