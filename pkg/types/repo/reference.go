package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type Reference interface {
	CreateReference(ctx context.Context, r *entity.Reference) error
	GetReference(ctx context.Context, id string) (*entity.Reference, error)
	ListReferences(ctx context.Context, f api.ListReferencesRequest) ([]*entity.Reference, error)
	DeleteReference(ctx context.Context, id string) error
	// DeleteReferencesForCard removes every reference where the given card
	// is either the source or the derived side. Prod SQLite runs without
	// the foreign_keys pragma, so the schema's ON DELETE CASCADE never
	// fires; callers that purge a card must invoke this first.
	DeleteReferencesForCard(ctx context.Context, cardID string) error
	// DeleteReferencesForTrashedCards mirrors the above for a bulk purge:
	// removes every reference whose source or derived side is a
	// soft-deleted card. Used by PurgeAllTrashedCards.
	DeleteReferencesForTrashedCards(ctx context.Context) error
}
