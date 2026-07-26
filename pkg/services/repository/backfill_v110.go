package repository

import (
	"context"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/orm"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// RunV110Backfill gives every card that has no snapshot a baseline row copying
// its current state, timestamped with the card's updated_at so the history
// reads truthfully. Idempotent: the NOT EXISTS guard means a second run is a
// no-op, which matters because this executes on every startup.
//
// Baselines carry no diff — nothing precedes them. Their purpose is to give
// the first post-upgrade edit something to diff against.
//
// IDs are minted in Go, not SQL. A tempting `lower(hex(randomblob(16)))`
// yields 32 hex chars, which ulid.ParseStrict rejects — every backfilled
// baseline would then 400 on GET /snapshots/:id, and only for pre-upgrade
// cards. Row-at-a-time is irrelevant at this scale: it runs once, over a
// personal knowledge base.
func (m *manager) RunV110Backfill(ctx context.Context) error {
	var rows []orm.Card
	if err := m.db.FromContext(ctx).Model(&orm.Card{}).
		Where("NOT EXISTS (SELECT 1 FROM card_snapshots s WHERE s.card_id = cards.id)").
		Find(&rows).Error; err != nil {
		return errors.DBFailed.Wrap(err)
	}
	for i := range rows {
		c := &rows[i]
		if err := m.db.FromContext(ctx).Create(&orm.CardSnapshot{
			ID:         ulidutil.New(),
			CardID:     c.ID,
			Seq:        1,
			Title:      c.Title,
			Summary:    c.Summary,
			Content:    c.Content,
			Format:     c.Format,
			Actor:      "system",
			ChangeKind: "baseline",
			CreatedAt:  c.UpdatedAt,
		}).Error; err != nil {
			return errors.DBFailed.Wrap(err)
		}
	}
	if len(rows) > 0 {
		m.log.Infof("v1.1.0 backfill: wrote %d baseline snapshot(s)", len(rows))
	}
	return nil
}
