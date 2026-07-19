package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/xhanio/errors"
	"gorm.io/gorm"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/orm"
)

func ormToEntityCard(o *orm.Card) *entity.Card {
	c := &entity.Card{
		ID:                   o.ID,
		Title:                o.Title,
		Summary:              o.Summary,
		Content:              o.Content,
		Format:               o.Format,
		LevelEntryID:         o.LevelEntryID,
		Genesis:              o.Genesis,
		GroupID:              o.GroupID,
		Position:             o.Position,
		ParentCardID:         o.ParentCardID,
		SourceConversationID: o.SourceConversationID,
		CreatedAt:            o.CreatedAt,
		UpdatedAt:            o.UpdatedAt,
		ReviewGrade:          o.ReviewGrade,
	}
	if o.DeletedAt.Valid {
		t := o.DeletedAt.Time
		c.DeletedAt = &t
	}
	if o.ReviewedAt.Valid {
		t := o.ReviewedAt.Time
		c.ReviewedAt = &t
	}
	return c
}

func entityToOrmCard(e *entity.Card) *orm.Card {
	row := &orm.Card{
		ID:                   e.ID,
		Title:                e.Title,
		Summary:              e.Summary,
		Content:              e.Content,
		Format:               normalizeFormat(e.Format),
		SearchHint:           searchHintFor(e.Format, e.Content),
		LevelEntryID:         e.LevelEntryID,
		Genesis:              e.Genesis,
		GroupID:              e.GroupID,
		Position:             e.Position,
		ParentCardID:         e.ParentCardID,
		SourceConversationID: e.SourceConversationID,
		CreatedAt:            e.CreatedAt,
		UpdatedAt:            e.UpdatedAt,
		ReviewGrade:          e.ReviewGrade,
	}
	if row.ReviewGrade == "" {
		row.ReviewGrade = "LGTM"
	}
	if e.DeletedAt != nil {
		row.DeletedAt = sql.NullTime{Time: *e.DeletedAt, Valid: true}
	}
	if e.ReviewedAt != nil {
		row.ReviewedAt = sql.NullTime{Time: *e.ReviewedAt, Valid: true}
	}
	return row
}

func (m *manager) CreateCard(ctx context.Context, c *entity.Card) error {
	if err := m.db.FromContext(ctx).Create(entityToOrmCard(c)).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to create card")
	}
	return nil
}

func (m *manager) GetCard(ctx context.Context, id string) (*entity.Card, error) {
	var row orm.Card
	err := m.db.FromContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound.Newf("card %q not found", id)
	}
	if err != nil {
		return nil, errors.DBFailed.Wrapf(err, "failed to get card %q", id)
	}
	return ormToEntityCard(&row), nil
}

func (m *manager) ListCards(ctx context.Context, groupID *string, includeTrashed bool) ([]*entity.Card, error) {
	q := m.db.FromContext(ctx)
	if groupID != nil {
		q = q.Where("group_id = ?", *groupID)
	}
	if !includeTrashed {
		q = q.Where("deleted_at IS NULL")
	}
	var rows []orm.Card
	if err := q.Order("position ASC, created_at ASC").Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Card, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityCard(&rows[i]))
	}
	return out, nil
}

func (m *manager) ListChildren(ctx context.Context, parentID string, includeTrashed bool) ([]*entity.Card, error) {
	q := m.db.FromContext(ctx).Where("parent_card_id = ?", parentID)
	if !includeTrashed {
		q = q.Where("deleted_at IS NULL")
	}
	var rows []orm.Card
	if err := q.Order("position ASC, created_at ASC").Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Card, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityCard(&rows[i]))
	}
	return out, nil
}

func (m *manager) UpdateCard(ctx context.Context, c *entity.Card) error {
	c.UpdatedAt = time.Now()
	res := m.db.FromContext(ctx).Save(entityToOrmCard(c))
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to update card %q", c.ID)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("card %q not found", c.ID)
	}
	return nil
}

func (m *manager) DeleteCard(ctx context.Context, id string) error {
	res := m.db.FromContext(ctx).Delete(&orm.Card{ID: id})
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to delete card %q", id)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("card %q not found", id)
	}
	return nil
}

// SoftDelete flips deleted_at on the target card, and — when cascade is
// true — on every live descendant reachable via parent_card_id. The
// cascade uses a recursive CTE so the whole subtree flips atomically in
// a single UPDATE. Cards that are already trashed are skipped; the
// returned count reflects only newly-trashed rows.
func (m *manager) SoftDelete(ctx context.Context, id string, cascade bool) (int, error) {
	now := time.Now()
	if !cascade {
		res := m.db.FromContext(ctx).
			Model(&orm.Card{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Updates(map[string]any{"deleted_at": now, "updated_at": now})
		if res.Error != nil {
			return 0, errors.DBFailed.Wrapf(res.Error, "failed to soft-delete card %q", id)
		}
		return int(res.RowsAffected), nil
	}
	// Recursive CTE finds the id + every live descendant, then the
	// UPDATE flips them all in one statement.
	sql := `
		WITH RECURSIVE subtree(id) AS (
			SELECT id FROM cards WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT c.id FROM cards c JOIN subtree s ON c.parent_card_id = s.id
				WHERE c.deleted_at IS NULL
		)
		UPDATE cards
			SET deleted_at = ?, updated_at = ?
			WHERE id IN (SELECT id FROM subtree)
	`
	res := m.db.FromContext(ctx).Exec(sql, id, now, now)
	if res.Error != nil {
		return 0, errors.DBFailed.Wrapf(res.Error, "failed to soft-delete subtree rooted at %q", id)
	}
	return int(res.RowsAffected), nil
}

func (m *manager) RestoreCard(ctx context.Context, id string) error {
	res := m.db.FromContext(ctx).
		Model(&orm.Card{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil)
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to restore card %q", id)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("card %q not soft-deleted", id)
	}
	return nil
}

func (m *manager) PurgeCard(ctx context.Context, id string) error {
	res := m.db.FromContext(ctx).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Delete(&orm.Card{})
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to purge card %q", id)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("card %q not soft-deleted", id)
	}
	return nil
}

// PurgeAllTrashedCards hard-deletes every soft-deleted card in one shot,
// cascading to card_tags + card_references via existing FK rules. Returns
// the number of cards that were purged.
func (m *manager) PurgeAllTrashedCards(ctx context.Context) (int, error) {
	res := m.db.FromContext(ctx).
		Where("deleted_at IS NOT NULL").
		Delete(&orm.Card{})
	if res.Error != nil {
		return 0, errors.DBFailed.Wrapf(res.Error, "failed to empty trash")
	}
	return int(res.RowsAffected), nil
}

func (m *manager) ListTrashedCards(ctx context.Context, limit int) ([]*entity.Card, error) {
	q := m.db.FromContext(ctx).
		Where("deleted_at IS NOT NULL").
		Order("deleted_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []orm.Card
	if err := q.Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Card, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityCard(&rows[i]))
	}
	return out, nil
}

func (m *manager) DeleteCardsByGroup(ctx context.Context, groupID string) error {
	if err := m.db.FromContext(ctx).
		Where("group_id = ?", groupID).
		Delete(&orm.Card{}).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to delete cards in group %q", groupID)
	}
	return nil
}

// --- CardTag methods ---

func (m *manager) AttachTag(ctx context.Context, cardID, tagID string) error {
	if err := m.db.FromContext(ctx).Create(&orm.CardTag{CardID: cardID, TagID: tagID}).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to attach tag %q to card %q", tagID, cardID)
	}
	return nil
}

func (m *manager) DetachTag(ctx context.Context, cardID, tagID string) error {
	res := m.db.FromContext(ctx).
		Where("card_id = ? AND tag_id = ?", cardID, tagID).
		Delete(&orm.CardTag{})
	if res.Error != nil {
		return errors.DBFailed.Wrap(res.Error)
	}
	return nil
}

func (m *manager) ListTagsForCard(ctx context.Context, cardID string) ([]string, error) {
	var names []string
	err := m.db.FromContext(ctx).
		Table("tags").
		Select("tags.name").
		Joins("JOIN card_tags ON card_tags.tag_id = tags.id").
		Where("card_tags.card_id = ?", cardID).
		Order("tags.name ASC").
		Pluck("tags.name", &names).Error
	if err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	return names, nil
}

func (m *manager) ListCardsForTag(ctx context.Context, tagName string) ([]string, error) {
	var ids []string
	err := m.db.FromContext(ctx).
		Table("card_tags").
		Select("card_tags.card_id").
		Joins("JOIN tags ON tags.id = card_tags.tag_id").
		Where("tags.name = ?", tagName).
		Pluck("card_tags.card_id", &ids).Error
	if err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	return ids, nil
}
