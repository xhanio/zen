package repo

import (
	"context"

	"github.com/xhanio/zen/pkg/types/entity"
)

type Message interface {
	CreateMessage(ctx context.Context, m *entity.Message) error
	GetMessage(ctx context.Context, id string) (*entity.Message, error)
	ListMessages(ctx context.Context, conversationID string, limit int) ([]*entity.Message, error)
	// ListMessagesAfter returns the messages of a conversation newer than
	// afterID, oldest first. An empty afterID means "from the beginning".
	//
	// ULIDs are monotonic and Crockford base32 sorts in time order, so the
	// cursor is a plain id comparison — no join on created_at is needed.
	ListMessagesAfter(ctx context.Context, conversationID, afterID string, limit int) ([]*entity.Message, error)
	LatestMessage(ctx context.Context, conversationID string) (*entity.Message, error)
	// SetMessageSession attributes a message to a session, but only if it has
	// none yet: the UPDATE is guarded by session_id IS NULL, so a dispatch can
	// fill an undelivered message's session once and can never re-point one
	// already set.
	SetMessageSession(ctx context.Context, messageID, sessionID, cwd string) error
	// CountUserMessagesByAnchorCard returns the total number of role="user"
	// messages across all conversations anchored to cardID (anchor_kind="card").
	CountUserMessagesByAnchorCard(ctx context.Context, cardID string) (int, error)
}
