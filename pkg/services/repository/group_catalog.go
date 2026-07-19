package repository

import (
	"context"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/orm"
)

// UnfileCardsForEntry clears level_entry_id on every card in the given
// group that points at the deleted catalog entry. Returns the number of
// rows touched. Used by group.Update when a catalog entry is dropped.
func (m *manager) UnfileCardsForEntry(ctx context.Context, groupID, entryID string) (int, error) {
	res := m.db.FromContext(ctx).
		Model(&orm.Card{}).
		Where("group_id = ? AND level_entry_id = ?", groupID, entryID).
		Update("level_entry_id", nil)
	if res.Error != nil {
		return 0, errors.DBFailed.Wrap(res.Error)
	}
	return int(res.RowsAffected), nil
}
