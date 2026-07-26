package repository

import (
	"context"
	"time"

	"github.com/xhanio/errors"
	"gorm.io/gorm"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/orm"
)

func ormToEntityCardSnapshot(o *orm.CardSnapshot) *entity.CardSnapshot {
	return &entity.CardSnapshot{
		ID:             o.ID,
		CardID:         o.CardID,
		Seq:            o.Seq,
		Title:          o.Title,
		Summary:        o.Summary,
		Content:        o.Content,
		Format:         o.Format,
		Actor:          o.Actor,
		ConversationID: o.ConversationID,
		ChangeKind:     o.ChangeKind,
		Diff:           o.Diff,
		DiffTruncated:  o.DiffTruncated,
		LinesAdded:     o.LinesAdded,
		LinesRemoved:   o.LinesRemoved,
		CreatedAt:      o.CreatedAt,
	}
}

func entityToOrmCardSnapshot(e *entity.CardSnapshot) *orm.CardSnapshot {
	return &orm.CardSnapshot{
		ID:             e.ID,
		CardID:         e.CardID,
		Seq:            e.Seq,
		Title:          e.Title,
		Summary:        e.Summary,
		Content:        e.Content,
		Format:         e.Format,
		Actor:          e.Actor,
		ConversationID: e.ConversationID,
		ChangeKind:     e.ChangeKind,
		Diff:           e.Diff,
		DiffTruncated:  e.DiffTruncated,
		LinesAdded:     e.LinesAdded,
		LinesRemoved:   e.LinesRemoved,
		CreatedAt:      e.CreatedAt,
	}
}

func (m *manager) CreateCardSnapshot(ctx context.Context, s *entity.CardSnapshot) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if err := m.db.FromContext(ctx).Create(entityToOrmCardSnapshot(s)).Error; err != nil {
		return errors.DBFailed.Wrap(err)
	}
	return nil
}

func (m *manager) NextSnapshotSeq(ctx context.Context, cardID string) (int, error) {
	var max *int
	if err := m.db.FromContext(ctx).Model(&orm.CardSnapshot{}).
		Where("card_id = ?", cardID).
		Select("MAX(seq)").Scan(&max).Error; err != nil {
		return 0, errors.DBFailed.Wrap(err)
	}
	if max == nil {
		return 1, nil
	}
	return *max + 1, nil
}

func (m *manager) GetCardSnapshot(ctx context.Context, id string) (*entity.CardSnapshot, error) {
	var row orm.CardSnapshot
	if err := m.db.FromContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound.Newf("snapshot %q not found", id)
		}
		return nil, errors.DBFailed.Wrap(err)
	}
	return ormToEntityCardSnapshot(&row), nil
}

func (m *manager) GetCardSnapshotBySeq(ctx context.Context, cardID string, seq int) (*entity.CardSnapshot, error) {
	var row orm.CardSnapshot
	if err := m.db.FromContext(ctx).
		Where("card_id = ? AND seq = ?", cardID, seq).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound.Newf("snapshot %d of card %q not found", seq, cardID)
		}
		return nil, errors.DBFailed.Wrap(err)
	}
	return ormToEntityCardSnapshot(&row), nil
}

func (m *manager) ListCardSnapshots(ctx context.Context, f api.ListSnapshotsRequest) ([]*entity.CardSnapshot, error) {
	if f.CardID == nil && f.ConversationID == nil {
		return nil, errors.BadRequest.Newf("at least one of card_id, conversation_id is required")
	}
	q := m.db.FromContext(ctx).Model(&orm.CardSnapshot{})
	if f.CardID != nil {
		// One card's chain reads newest-first: the list is a history view.
		q = q.Where("card_id = ?", *f.CardID).Order("seq DESC")
	}
	if f.ConversationID != nil {
		// A conversation's records read oldest-first: they merge into a
		// timeline that runs forward in time.
		q = q.Where("conversation_id = ?", *f.ConversationID).Order("created_at ASC, id ASC")
	}
	var rows []orm.CardSnapshot
	if err := q.Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.CardSnapshot, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityCardSnapshot(&rows[i]))
	}
	return m.fillCardTitles(ctx, out)
}

// fillCardTitles denormalizes the current card title onto each snapshot so a
// timeline row can name its card without a per-row fetch.
func (m *manager) fillCardTitles(ctx context.Context, snaps []*entity.CardSnapshot) ([]*entity.CardSnapshot, error) {
	if len(snaps) == 0 {
		return snaps, nil
	}
	ids := make([]string, 0, len(snaps))
	seen := map[string]struct{}{}
	for _, s := range snaps {
		if _, ok := seen[s.CardID]; ok {
			continue
		}
		seen[s.CardID] = struct{}{}
		ids = append(ids, s.CardID)
	}
	var rows []orm.Card
	if err := m.db.FromContext(ctx).Model(&orm.Card{}).
		Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	titles := make(map[string]string, len(rows))
	for i := range rows {
		titles[rows[i].ID] = rows[i].Title
	}
	for _, s := range snaps {
		s.CardTitle = titles[s.CardID]
	}
	return snaps, nil
}

func (m *manager) DeleteSnapshotsForCard(ctx context.Context, cardID string) error {
	if err := m.db.FromContext(ctx).
		Where("card_id = ?", cardID).Delete(&orm.CardSnapshot{}).Error; err != nil {
		return errors.DBFailed.Wrap(err)
	}
	return nil
}

func (m *manager) DeleteSnapshotsForTrashedCards(ctx context.Context) error {
	if err := m.db.FromContext(ctx).
		Where("card_id IN (SELECT id FROM cards WHERE deleted_at IS NOT NULL)").
		Delete(&orm.CardSnapshot{}).Error; err != nil {
		return errors.DBFailed.Wrap(err)
	}
	return nil
}
