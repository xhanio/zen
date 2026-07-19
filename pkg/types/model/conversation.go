package model

import (
	"context"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/entity"
)

// AppendOptions carries the parts of an append that are not message content.
type AppendOptions struct {
	// TargetSessionID addresses the published event to one Claude Code session.
	// Never persisted.
	TargetSessionID string

	// SessionID and SessionCwd attribute the stored message to a session and
	// snapshot its working directory. Persisted on the row and echoed on the
	// event. Empty SessionID means the message carries no attribution.
	SessionID  string
	SessionCwd string
}

type AppendOption func(*AppendOptions)

// WithTargetSession addresses the resulting event to one session. Without it,
// the event is addressed to nobody.
func WithTargetSession(sessionID string) AppendOption {
	return func(o *AppendOptions) { o.TargetSessionID = sessionID }
}

// WithSession attributes the stored message to a session and snapshots its cwd.
func WithSession(sessionID, cwd string) AppendOption {
	return func(o *AppendOptions) { o.SessionID = sessionID; o.SessionCwd = cwd }
}

type Conversation interface {
	common.Service
	Create(ctx context.Context, title string, anchorKind, anchorID *string) (*entity.Conversation, error)
	Get(ctx context.Context, id string) (*entity.Conversation, error)
	List(ctx context.Context, anchorKind, anchorID *string, pending bool, limit int) ([]*entity.Conversation, []int, error)
	UpdateTitle(ctx context.Context, id, newTitle string) (*entity.Conversation, error)
	Delete(ctx context.Context, id string) error
	AppendMessage(ctx context.Context, conversationID, role, content string, selectionText *string, opts ...AppendOption) (*entity.Message, error)
	GetMessages(ctx context.Context, conversationID string, limit int) ([]*entity.Message, error)

	// GetMessagesAfter returns the messages newer than afterID, oldest first.
	// An empty afterID means "from the beginning".
	GetMessagesAfter(ctx context.Context, conversationID, afterID string, limit int) ([]*entity.Message, error)

	// DispatchMessage re-publishes an already-stored user message as an event
	// addressed to one session. It never inserts a row: this serves both the
	// "no session was connected when I posted" path and the "resend to another
	// session" path, and neither may duplicate the message.
	DispatchMessage(ctx context.Context, conversationID, messageID, targetSessionID, targetCwd string) error
	DeleteByAnchor(ctx context.Context, anchorKind, anchorID string) error
}
