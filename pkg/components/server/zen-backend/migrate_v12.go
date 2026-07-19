package zenbackend

import (
	"context"

	"github.com/xhanio/errors"
)

// runV12PostInit applies the v0.12 grade-backfill pass. Sweeps every card
// that has card-anchored conversations, counts user messages, and raises
// review_grade to the corresponding auto-upgrade floor. Idempotent — cards
// already at or above the floor are left alone, so this is safe on every
// startup.
func (m *manager) runV12PostInit(ctx context.Context) error {
	if err := m.repository.RunV12Backfill(ctx); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
