package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Tag interface {
	CreateTag(ctx context.Context, t *entity.Tag) error
	GetTag(ctx context.Context, id string) (*entity.Tag, error)
	GetTagByNameInGroup(ctx context.Context, groupID, name string) (*entity.Tag, error)
	ListTags(ctx context.Context, groupID string) ([]*entity.Tag, error)
	UpdateTag(ctx context.Context, t *entity.Tag) error
	DeleteTag(ctx context.Context, id string) error
}
