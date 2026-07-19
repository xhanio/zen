package repository

import (
	"context"

	"github.com/xhanio/errors"
	"gorm.io/gorm"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/orm"
)

func ormToEntityTag(o *orm.Tag) *entity.Tag {
	return &entity.Tag{ID: o.ID, GroupID: o.GroupID, Name: o.Name}
}

func entityToOrmTag(e *entity.Tag) *orm.Tag {
	return &orm.Tag{ID: e.ID, GroupID: e.GroupID, Name: e.Name}
}

func (m *manager) CreateTag(ctx context.Context, t *entity.Tag) error {
	row := entityToOrmTag(t)
	if err := m.db.FromContext(ctx).Create(row).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to create tag %q", t.Name)
	}
	return nil
}

func (m *manager) GetTag(ctx context.Context, id string) (*entity.Tag, error) {
	var row orm.Tag
	err := m.db.FromContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound.Newf("tag %q not found", id)
	}
	if err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	return ormToEntityTag(&row), nil
}

func (m *manager) GetTagByNameInGroup(ctx context.Context, groupID, name string) (*entity.Tag, error) {
	var row orm.Tag
	err := m.db.FromContext(ctx).First(&row, "group_id = ? AND name = ?", groupID, name).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound.Newf("tag named %q not found in group %q", name, groupID)
	}
	if err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	return ormToEntityTag(&row), nil
}

func (m *manager) ListTags(ctx context.Context, groupID string) ([]*entity.Tag, error) {
	type row struct {
		ID        string
		Name      string
		CardCount int
	}
	var rows []row
	// Scope to the group's own tags. Only live cards contribute to card_count:
	// trashed cards keep their card_tags rows (soft delete leaves them intact),
	// so we LEFT JOIN through cards and filter deleted_at — a zero-card tag in
	// the group's vocabulary still shows.
	err := m.db.FromContext(ctx).
		Table("tags").
		Select("tags.id AS id, tags.name AS name, COUNT(cards.id) AS card_count").
		Joins("LEFT JOIN card_tags ON card_tags.tag_id = tags.id").
		Joins("LEFT JOIN cards ON cards.id = card_tags.card_id AND cards.deleted_at IS NULL").
		Where("tags.group_id = ?", groupID).
		Group("tags.id").
		Order("tags.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, &entity.Tag{ID: r.ID, GroupID: groupID, Name: r.Name, CardCount: r.CardCount})
	}
	return out, nil
}

func (m *manager) UpdateTag(ctx context.Context, t *entity.Tag) error {
	res := m.db.FromContext(ctx).Save(entityToOrmTag(t))
	if res.Error != nil {
		return errors.DBFailed.Wrap(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("tag %q not found", t.ID)
	}
	return nil
}

func (m *manager) DeleteTag(ctx context.Context, id string) error {
	res := m.db.FromContext(ctx).Delete(&orm.Tag{ID: id})
	if res.Error != nil {
		return errors.DBFailed.Wrap(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("tag %q not found", id)
	}
	return nil
}
