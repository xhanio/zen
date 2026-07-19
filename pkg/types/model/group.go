package model

import (
	"context"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Group interface {
	common.Service
	Create(ctx context.Context, name string, levelCatalog []entity.LevelEntry, rule string) (*entity.Group, error)
	Get(ctx context.Context, id string) (*entity.Group, error)
	List(ctx context.Context) ([]*entity.Group, error)
	Update(ctx context.Context, id string, name *string, position *int, levelCatalog *[]entity.LevelEntry, rule *string) (*entity.Group, error)
	Delete(ctx context.Context, id string, recursive bool) error
}
