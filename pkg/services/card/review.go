package card

import (
	"context"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// Valid review grades. Uppercase per the v0.12 spec — API, storage,
// Go constants all agree.
const (
	GradeLGTM     = "LGTM"
	GradeDigested = "DIGESTED"
	GradeGrilled  = "GRILLED"
)

func isValidGrade(g string) bool {
	return g == GradeLGTM || g == GradeDigested || g == GradeGrilled
}

// Review sets the review_grade on cardID and adjusts reviewed_at per
// the v0.12 rules:
//   - grade rising above LGTM: reviewed_at = now()
//   - grade returning to LGTM: reviewed_at = nil
//   - grade unchanged: no writes at all (reviewed_at untouched)
//
// Idempotent. Setting to the current grade returns the card unchanged.
func (m *manager) Review(ctx context.Context, cardID string, grade string) (*entity.Card, error) {
	if err := ulidutil.Parse(cardID); err != nil {
		return nil, errors.Wrap(err)
	}
	if !isValidGrade(grade) {
		return nil, errors.BadRequest.Newf("invalid review grade %q; must be one of LGTM, DIGESTED, GRILLED", grade)
	}
	var updated *entity.Card
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		card, err := m.repo.GetCard(txCtx, cardID)
		if err != nil {
			return errors.Wrap(err)
		}
		if card.ReviewGrade == grade {
			updated = card
			return nil
		}
		card.ReviewGrade = grade
		if grade == GradeLGTM {
			card.ReviewedAt = nil
		} else {
			now := time.Now().UTC()
			card.ReviewedAt = &now
		}
		if err := m.repo.UpdateCard(txCtx, card); err != nil {
			return errors.Wrap(err)
		}
		updated = card
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	// Populate ReviewScore on the returned card so API consumers see the
	// fresh score without a follow-up GET.
	score, err := m.computeReviewScore(ctx, updated)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	updated.ReviewScore = score
	return updated, nil
}
