package model

import (
	"context"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Tag interface {
	common.Service
	EnsureByName(ctx context.Context, groupID, name string) (*entity.Tag, error)
	Get(ctx context.Context, id string) (*entity.Tag, error)
	List(ctx context.Context, groupID string) ([]*entity.Tag, error)
	Rename(ctx context.Context, groupID, oldName, newName string) (*entity.Tag, error)
	Delete(ctx context.Context, groupID, name string) error
}
