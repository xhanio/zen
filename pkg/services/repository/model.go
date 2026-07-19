package repository

import (
	"context"
	"database/sql"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/repo"
)

type Repository interface {
	common.Service
	Transaction(ctx context.Context, fn func(context.Context) error, opts ...*sql.TxOptions) error
	repo.Group
	repo.Tag
	repo.Card
	repo.CardTag
	repo.Conversation
	repo.Message
	repo.Reference
	repo.Search
	// RunV12Backfill applies the v0.12 auto-upgrade rule to every existing
	// card once (based on prior conversation activity). Idempotent — cards
	// already at or above the computed floor are left alone.
	RunV12Backfill(ctx context.Context) error
}
