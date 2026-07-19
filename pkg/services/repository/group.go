package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/xhanio/errors"
	"gorm.io/gorm"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/orm"
)

func ormToEntityGroup(o *orm.Group) *entity.Group {
	g := &entity.Group{
		ID:        o.ID,
		Name:      o.Name,
		Rule:      o.Rule,
		Position:  o.Position,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
	raw := strings.TrimSpace(o.LevelCatalog)
	if raw != "" && raw != "[]" {
		if err := json.Unmarshal([]byte(raw), &g.LevelCatalog); err != nil {
			g.LevelCatalog = nil
		}
	}
	if g.LevelCatalog == nil {
		g.LevelCatalog = []entity.LevelEntry{}
	}
	return g
}

func entityToOrmGroup(e *entity.Group) *orm.Group {
	cat := e.LevelCatalog
	if cat == nil {
		cat = []entity.LevelEntry{}
	}
	b, _ := json.Marshal(cat)
	return &orm.Group{
		ID:           e.ID,
		Name:         e.Name,
		Rule:         e.Rule,
		Position:     e.Position,
		LevelCatalog: string(b),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func (m *manager) CreateGroup(ctx context.Context, g *entity.Group) error {
	row := entityToOrmGroup(g)
	if err := m.db.FromContext(ctx).Create(row).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to create group")
	}
	return nil
}

func (m *manager) GetGroup(ctx context.Context, id string) (*entity.Group, error) {
	var row orm.Group
	err := m.db.FromContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound.Newf("group %q not found", id)
	}
	if err != nil {
		return nil, errors.DBFailed.Wrapf(err, "failed to get group %q", id)
	}
	return ormToEntityGroup(&row), nil
}

func (m *manager) ListGroups(ctx context.Context) ([]*entity.Group, error) {
	var rows []orm.Group
	err := m.db.FromContext(ctx).
		Order("position ASC, name ASC").
		Find(&rows).Error
	if err != nil {
		return nil, errors.DBFailed.Wrapf(err, "failed to list groups")
	}
	out := make([]*entity.Group, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityGroup(&rows[i]))
	}
	return out, nil
}

func (m *manager) UpdateGroup(ctx context.Context, g *entity.Group) error {
	g.UpdatedAt = time.Now()
	row := entityToOrmGroup(g)
	res := m.db.FromContext(ctx).Save(row)
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to update group %q", g.ID)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("group %q not found", g.ID)
	}
	return nil
}

func (m *manager) DeleteGroup(ctx context.Context, id string) error {
	res := m.db.FromContext(ctx).Delete(&orm.Group{ID: id})
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to delete group %q", id)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("group %q not found", id)
	}
	return nil
}

func (m *manager) GroupHasContent(ctx context.Context, id string) (bool, error) {
	g := m.db.FromContext(ctx)
	var n int64
	if err := g.Table("cards").Where("group_id = ?", id).Count(&n).Error; err != nil {
		return false, errors.DBFailed.Wrap(err)
	}
	return n > 0, nil
}
