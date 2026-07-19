package model

import (
	"context"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Search interface {
	common.Service
	Search(ctx context.Context, query, scope string, limit int) (cards, msgs []*entity.SearchHit, err error)
}
