package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Group interface {
	CreateGroup(ctx context.Context, g *entity.Group) error
	GetGroup(ctx context.Context, id string) (*entity.Group, error)
	ListGroups(ctx context.Context) ([]*entity.Group, error)
	UpdateGroup(ctx context.Context, g *entity.Group) error
	DeleteGroup(ctx context.Context, id string) error
	GroupHasContent(ctx context.Context, id string) (bool, error)
	// UnfileCardsForEntry clears level_entry_id on every card in the given
	// group that points at the deleted catalog entry. Returns rows touched.
	// Used by group.Update when a catalog entry is dropped.
	UnfileCardsForEntry(ctx context.Context, groupID, entryID string) (int, error)
}
