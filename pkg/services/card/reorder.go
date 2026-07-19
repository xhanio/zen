package card

import (
	"context"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// Reorder moves cardID to `position` inside its parent container. The
// parent stays the same. Positions of siblings whose index changes are
// updated atomically inside a single transaction.
//
// Rules:
//   - cardID must be a valid ULID.
//   - The card must currently have a non-nil parent_card_id. Top-level
//     cards use a different reorder path (the group grid); rejected
//     here with BadRequest.
//   - `position` is clamped to [0, len(liveSiblings) - 1].
//   - If the clamped target equals the card's current position, no
//     writes are issued and the card is returned unchanged.
func (m *manager) Reorder(ctx context.Context, cardID string, position int) (*entity.Card, error) {
	if err := ulidutil.Parse(cardID); err != nil {
		return nil, errors.Wrap(err)
	}
	var updated *entity.Card
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		card, err := m.repo.GetCard(txCtx, cardID)
		if err != nil {
			return errors.Wrap(err)
		}
		if card.ParentCardID == nil {
			return errors.BadRequest.Newf("card %q has no parent; reorder is only valid for section cards", cardID)
		}
		siblings, err := m.repo.ListChildren(txCtx, *card.ParentCardID, false)
		if err != nil {
			return errors.Wrap(err)
		}
		n := len(siblings)
		if n == 0 {
			return errors.NotFound.Newf("no siblings found for card %q", cardID)
		}
		oldIdx := -1
		for i, s := range siblings {
			if s.ID == cardID {
				oldIdx = i
				break
			}
		}
		if oldIdx < 0 {
			return errors.NotFound.Newf("card %q not in its parent's child list", cardID)
		}
		target := position
		if target < 0 {
			target = 0
		}
		if target >= n {
			target = n - 1
		}
		if target == oldIdx {
			updated = card
			return nil
		}
		moved := siblings[oldIdx]
		rest := make([]*entity.Card, 0, n-1)
		rest = append(rest, siblings[:oldIdx]...)
		rest = append(rest, siblings[oldIdx+1:]...)
		final := make([]*entity.Card, 0, n)
		final = append(final, rest[:target]...)
		final = append(final, moved)
		final = append(final, rest[target:]...)
		for i, s := range final {
			if s.Position != i {
				s.Position = i
				if err := m.repo.UpdateCard(txCtx, s); err != nil {
					return errors.Wrap(err)
				}
			}
		}
		updated = moved
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return updated, nil
}
