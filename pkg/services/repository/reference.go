package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/xhanio/errors"
	"gorm.io/gorm"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/orm"
)

func ormToEntityReference(o *orm.Reference) *entity.Reference {
	var conv *string
	if o.ConversationID.Valid {
		v := o.ConversationID.String
		conv = &v
	}
	return &entity.Reference{
		ID:             o.ID,
		SourceCardID:   o.SourceCardID,
		DerivedCardID:  o.DerivedCardID,
		ConversationID: conv,
		SelectionText:  o.SelectionText,
		CreatedAt:      o.CreatedAt,
	}
}

func entityToOrmReference(e *entity.Reference) *orm.Reference {
	var conv sql.NullString
	if e.ConversationID != nil {
		conv = sql.NullString{String: *e.ConversationID, Valid: true}
	}
	return &orm.Reference{
		ID:             e.ID,
		SourceCardID:   e.SourceCardID,
		DerivedCardID:  e.DerivedCardID,
		ConversationID: conv,
		SelectionText:  e.SelectionText,
		CreatedAt:      e.CreatedAt,
	}
}

func (m *manager) CreateReference(ctx context.Context, r *entity.Reference) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	row := entityToOrmReference(r)
	if err := m.db.FromContext(ctx).Create(row).Error; err != nil {
		return errors.DBFailed.Wrap(err)
	}
	return nil
}

func (m *manager) GetReference(ctx context.Context, id string) (*entity.Reference, error) {
	var row orm.Reference
	if err := m.db.FromContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound.Newf("reference %q not found", id)
		}
		return nil, errors.DBFailed.Wrap(err)
	}
	return ormToEntityReference(&row), nil
}

func (m *manager) ListReferences(ctx context.Context, f api.ListReferencesRequest) ([]*entity.Reference, error) {
	q := m.db.FromContext(ctx).Model(&orm.Reference{})
	if f.SourceCardID != nil {
		q = q.Where("source_card_id = ?", *f.SourceCardID)
	}
	if f.DerivedCardID != nil {
		q = q.Where("derived_card_id = ?", *f.DerivedCardID)
	}
	if f.ConversationID != nil {
		q = q.Where("conversation_id = ?", *f.ConversationID)
	}
	q = q.Order("created_at ASC")
	var rows []*orm.Reference
	if err := q.Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Reference, 0, len(rows))
	for _, r := range rows {
		out = append(out, ormToEntityReference(r))
	}
	return out, nil
}

func (m *manager) DeleteReference(ctx context.Context, id string) error {
	res := m.db.FromContext(ctx).Where("id = ?", id).Delete(&orm.Reference{})
	if res.Error != nil {
		return errors.DBFailed.Wrap(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("reference %q not found", id)
	}
	return nil
}

func (m *manager) DeleteReferencesForCard(ctx context.Context, cardID string) error {
	if err := m.db.FromContext(ctx).
		Where("source_card_id = ? OR derived_card_id = ?", cardID, cardID).
		Delete(&orm.Reference{}).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to purge references for card %q", cardID)
	}
	return nil
}

func (m *manager) DeleteReferencesForTrashedCards(ctx context.Context) error {
	sub := "SELECT id FROM cards WHERE deleted_at IS NOT NULL"
	if err := m.db.FromContext(ctx).
		Where("source_card_id IN ("+sub+") OR derived_card_id IN ("+sub+")").
		Delete(&orm.Reference{}).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to purge references for trashed cards")
	}
	return nil
}
