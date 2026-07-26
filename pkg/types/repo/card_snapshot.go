package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type CardSnapshot interface {
	CreateCardSnapshot(ctx context.Context, s *entity.CardSnapshot) error
	// NextSnapshotSeq returns max(seq)+1 for a card, or 1 when it has none.
	// Call inside the mutation's transaction; the unique index on
	// (card_id, seq) is the backstop against a racing writer.
	NextSnapshotSeq(ctx context.Context, cardID string) (int, error)
	GetCardSnapshot(ctx context.Context, id string) (*entity.CardSnapshot, error)
	GetCardSnapshotBySeq(ctx context.Context, cardID string, seq int) (*entity.CardSnapshot, error)
	// ListCardSnapshots requires at least one filter. Card-filtered results
	// are newest-first; conversation-filtered results are oldest-first, which
	// is the order the conversation timeline merges in.
	ListCardSnapshots(ctx context.Context, f api.ListSnapshotsRequest) ([]*entity.CardSnapshot, error)
	// DeleteSnapshotsForCard removes a card's snapshots. Production SQLite
	// runs without the foreign_keys pragma, so the schema cascade never
	// fires; purge MUST call this explicitly.
	DeleteSnapshotsForCard(ctx context.Context, cardID string) error
	// DeleteSnapshotsForTrashedCards mirrors the above for empty-trash.
	DeleteSnapshotsForTrashedCards(ctx context.Context) error
}
