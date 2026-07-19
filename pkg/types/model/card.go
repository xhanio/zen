package model

import (
	"context"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type Card interface {
	common.Service
	Create(ctx context.Context, title, content, groupID string, tagNames []string, parentCardID, sourceConversationID *string, format *string, levelEntryID *string, genesis *string, reference *api.ReferenceSpec, summary *string) (*entity.Card, error)
	Get(ctx context.Context, id string) (*entity.Card, error)
	List(ctx context.Context, groupID *string, includeTrashed bool) ([]*entity.Card, error)
	Children(ctx context.Context, parentID string, includeTrashed bool) ([]*entity.Card, error)
	Update(ctx context.Context, id string, title, content, groupID *string, position *int, tagNames *[]string, format *string, levelEntryID *string, clearLevelEntry bool, genesis *string, summary *string) (*entity.Card, error)
	Delete(ctx context.Context, id string, cascade bool) error
	Restore(ctx context.Context, id string) (*entity.Card, error)
	Purge(ctx context.Context, id string) error
	Trash(ctx context.Context, limit int) ([]*entity.Card, error)
	EmptyTrash(ctx context.Context) (int, error)
	Decompose(ctx context.Context, req api.DecomposeRequest) (*api.DecomposeResponse, error)
	Compose(ctx context.Context, req api.ComposeRequest) (*api.ComposeResponse, error)
	Reorder(ctx context.Context, cardID string, position int) (*entity.Card, error)
	Review(ctx context.Context, cardID string, grade string) (*entity.Card, error)
}
