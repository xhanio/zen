package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Conversation interface {
	CreateConversation(ctx context.Context, c *entity.Conversation) error
	GetConversation(ctx context.Context, id string) (*entity.Conversation, error)
	ListConversations(ctx context.Context, anchorKind, anchorID *string, limit int) ([]*entity.Conversation, error)
	ListConversationsByAnchor(ctx context.Context, anchorKind, anchorID string) ([]*entity.Conversation, error)
	ListPendingConversations(ctx context.Context, limit int) ([]*entity.Conversation, []int, error)
	UpdateConversation(ctx context.Context, c *entity.Conversation) error
	DeleteConversation(ctx context.Context, id string) error
}
