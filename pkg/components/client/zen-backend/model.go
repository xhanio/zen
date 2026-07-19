package zenbackend

import (
	"context"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

// Client is the aggregated entry point returned by New. Each sub-interface
// covers one REST resource. Methods return errors mapped to xhanio/errors
// categories so callers can use errors.Is(err, errors.NotFound) etc.
type Client interface {
	Group() GroupClient
	Tag() TagClient
	Card() CardClient
	Search() SearchClient
	Conversation() ConversationClient
	Reference() ReferenceClient
}

type ReferenceClient interface {
	Create(ctx context.Context, req api.CreateReferenceRequest) (*entity.Reference, error)
	Get(ctx context.Context, id string) (*entity.Reference, error)
	List(ctx context.Context, f api.ListReferencesRequest) ([]*entity.Reference, error)
	Delete(ctx context.Context, id string) error
}

type GroupClient interface {
	Create(ctx context.Context, req api.CreateGroupRequest) (*entity.Group, error)
	List(ctx context.Context) ([]*entity.Group, error)
	Get(ctx context.Context, id string) (*entity.Group, error)
	Update(ctx context.Context, id string, req api.UpdateGroupRequest) (*entity.Group, error)
	Delete(ctx context.Context, id string, recursive bool) error
}

type TagClient interface {
	List(ctx context.Context, groupID string) ([]*entity.Tag, error)
	Rename(ctx context.Context, groupID, name string, req api.RenameTagRequest) (*entity.Tag, error)
	Delete(ctx context.Context, groupID, name string) error
}

type CardClient interface {
	Create(ctx context.Context, req api.CreateCardRequest) (*entity.Card, error)
	List(ctx context.Context, groupID *string, includeTrashed bool) ([]*entity.Card, error)
	Get(ctx context.Context, id string) (*entity.Card, error)
	Update(ctx context.Context, id string, req api.UpdateCardRequest) (*entity.Card, error)
	Delete(ctx context.Context, id string, cascade bool) error
	Decompose(ctx context.Context, req api.DecomposeRequest) (*api.DecomposeResponse, error)
	Compose(ctx context.Context, req api.ComposeRequest) (*api.ComposeResponse, error)
	Restore(ctx context.Context, id string) (*entity.Card, error)
	Purge(ctx context.Context, id string) error
	Trash(ctx context.Context, limit int) (*api.TrashResponse, error)
	Children(ctx context.Context, parentID string, includeTrashed bool) ([]*entity.Card, error)
	EmptyTrash(ctx context.Context) (*api.EmptyTrashResponse, error)
	Reorder(ctx context.Context, id string, position int) (*entity.Card, error)
	Review(ctx context.Context, id string, grade string) (*entity.Card, error)
}

type SearchClient interface {
	Search(ctx context.Context, query, scope string, limit int) (*api.SearchResponse, error)
}

type ConversationClient interface {
	Create(ctx context.Context, req api.CreateConversationRequest) (*entity.Conversation, error)
	Get(ctx context.Context, id string) (*entity.Conversation, error)
	List(ctx context.Context, anchorKind, anchorID *string, pending bool, limit int) (*api.ConversationListResponse, error)
	UpdateTitle(ctx context.Context, id string, req api.UpdateConversationTitleRequest) (*entity.Conversation, error)
	Delete(ctx context.Context, id string) error
	AppendMessage(ctx context.Context, conversationID string, req api.AppendMessageRequest) (*entity.Message, error)
	GetMessages(ctx context.Context, conversationID string, limit int) (*api.MessageListResponse, error)
}
