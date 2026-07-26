package zenbackend

import (
	"context"

	"github.com/xhanio/errors"
)

// runV110PostInit gives pre-v1.1.0 cards a baseline snapshot, so their first
// post-upgrade edit has something to diff against. Idempotent — cards that
// already have snapshots are skipped — so it is safe on every startup.
func (m *manager) runV110PostInit(ctx context.Context) error {
	if err := m.repository.RunV110Backfill(ctx); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
