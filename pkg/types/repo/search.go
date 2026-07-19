package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Search interface {
	SearchCards(ctx context.Context, query string, limit int) ([]*entity.SearchHit, error)
	SearchMessages(ctx context.Context, query string, limit int) ([]*entity.SearchHit, error)
}
