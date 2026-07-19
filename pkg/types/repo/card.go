package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Card interface {
	CreateCard(ctx context.Context, c *entity.Card) error
	GetCard(ctx context.Context, id string) (*entity.Card, error)
	ListCards(ctx context.Context, groupID *string, includeTrashed bool) ([]*entity.Card, error)
	// ListChildren returns cards whose parent_card_id equals parentID,
	// ordered by position ASC then created_at ASC. When !includeTrashed
	// the result excludes soft-deleted rows (deleted_at IS NULL).
	ListChildren(ctx context.Context, parentID string, includeTrashed bool) ([]*entity.Card, error)
	UpdateCard(ctx context.Context, c *entity.Card) error
	DeleteCard(ctx context.Context, id string) error
	// SoftDelete flips the target card's deleted_at to the current time.
	// When cascade=true, every live descendant (transitively linked via
	// parent_card_id) is flipped in the same statement. Returns the
	// number of cards flipped. Already-trashed cards in the subtree are
	// left alone.
	SoftDelete(ctx context.Context, id string, cascade bool) (int, error)
	RestoreCard(ctx context.Context, id string) error
	PurgeCard(ctx context.Context, id string) error
	PurgeAllTrashedCards(ctx context.Context) (int, error)
	ListTrashedCards(ctx context.Context, limit int) ([]*entity.Card, error)
	DeleteCardsByGroup(ctx context.Context, groupID string) error
}
