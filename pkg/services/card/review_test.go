package card_test

import (
	"context"
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

// seedReviewCtx builds a fresh card service backed by an in-memory repo with
// one top-level leaf card ready to review.
func seedReviewCtx(t *testing.T) (svc model.Card, leafID string) {
	t.Helper()
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	leaf := &entity.Card{
		ID: ulidutil.New(), Title: "leaf", Content: "body", GroupID: g.ID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, leaf); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	svc = card.New(repo, tag.New(repo), nil)
	return svc, leaf.ID
}

func TestReview_HappyPath_SetsGradeAndReviewedAt(t *testing.T) {
	svc, id := seedReviewCtx(t)
	before := time.Now().UTC()
	got, err := svc.Review(context.Background(), id, "DIGESTED")
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if got.ReviewGrade != "DIGESTED" {
		t.Fatalf("expected DIGESTED, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at non-nil")
	}
	if diff := got.ReviewedAt.Sub(before).Abs(); diff > 2*time.Second {
		t.Fatalf("reviewed_at drifted %v from now", diff)
	}
}

func TestReview_SettingLGTM_ClearsReviewedAt(t *testing.T) {
	svc, id := seedReviewCtx(t)
	if _, err := svc.Review(context.Background(), id, "GRILLED"); err != nil {
		t.Fatalf("first Review: %v", err)
	}
	got, err := svc.Review(context.Background(), id, "LGTM")
	if err != nil {
		t.Fatalf("second Review: %v", err)
	}
	if got.ReviewGrade != "LGTM" {
		t.Fatalf("expected LGTM, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt != nil {
		t.Fatalf("expected reviewed_at nil after LGTM, got %v", *got.ReviewedAt)
	}
}

func TestReview_DowngradeAllowed(t *testing.T) {
	svc, id := seedReviewCtx(t)
	if _, err := svc.Review(context.Background(), id, "GRILLED"); err != nil {
		t.Fatalf("Review GRILLED: %v", err)
	}
	got, err := svc.Review(context.Background(), id, "DIGESTED")
	if err != nil {
		t.Fatalf("Review DIGESTED: %v", err)
	}
	if got.ReviewGrade != "DIGESTED" {
		t.Fatalf("expected DIGESTED, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("downgrade above LGTM keeps reviewed_at set")
	}
}

func TestReview_InvalidGradeRejected(t *testing.T) {
	svc, id := seedReviewCtx(t)
	if _, err := svc.Review(context.Background(), id, "BOGUS"); err == nil {
		t.Fatalf("expected error for bogus grade")
	}
}

func TestReview_NonexistentCardReturns404(t *testing.T) {
	svc, _ := seedReviewCtx(t)
	// Valid ULID that doesn't exist
	if _, err := svc.Review(context.Background(), ulidutil.New(), "DIGESTED"); err == nil {
		t.Fatalf("expected error for missing card")
	}
}

func TestReview_SameGradeIsNoOp(t *testing.T) {
	svc, id := seedReviewCtx(t)
	first, err := svc.Review(context.Background(), id, "DIGESTED")
	if err != nil {
		t.Fatalf("first Review: %v", err)
	}
	if first.ReviewedAt == nil {
		t.Fatalf("first Review should set reviewed_at")
	}
	firstAt := *first.ReviewedAt
	time.Sleep(20 * time.Millisecond)
	got, err := svc.Review(context.Background(), id, "DIGESTED")
	if err != nil {
		t.Fatalf("second Review: %v", err)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("second Review lost reviewed_at")
	}
	if !got.ReviewedAt.Equal(firstAt) {
		t.Fatalf("no-op should not touch reviewed_at; before=%v after=%v", firstAt, *got.ReviewedAt)
	}
}
