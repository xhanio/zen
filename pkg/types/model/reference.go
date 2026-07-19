package model

import (
	"context"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type Reference interface {
	common.Service
	Create(ctx context.Context, req api.CreateReferenceRequest) (*entity.Reference, error)
	Get(ctx context.Context, id string) (*entity.Reference, error)
	List(ctx context.Context, f api.ListReferencesRequest) ([]*entity.Reference, error)
	Delete(ctx context.Context, id string) error
}
