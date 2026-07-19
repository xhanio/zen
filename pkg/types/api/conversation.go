package api

import "github.com/xhanio/zen/pkg/types/entity"

type CreateConversationRequest struct {
	Title      string  `json:"title" validate:"max=80"`
	// oneof mirrors the conversations CHECK constraint, which migration
	// 010_v06_card_only narrowed to card/group. "document" lingered here after
	// that and was accepted by this validator only to be rejected a layer down
	// by conversation.validateAnchor — a legal-looking value that never worked.
	AnchorKind *string `json:"anchor_kind" validate:"omitempty,oneof=card group"`
	AnchorID   *string `json:"anchor_id" validate:"omitempty,len=26"`
}

type UpdateConversationTitleRequest struct {
	Title string `json:"title" validate:"required,max=80"`
}

type AppendMessageRequest struct {
	Role          string  `json:"role" validate:"required,oneof=user assistant system"`
	Content       string  `json:"content" validate:"required,max=1048576"`
	SelectionText *string `json:"selection_text" validate:"omitempty,max=10000"`

	// TargetSessionID addresses this message at one live Claude Code session.
	// Absent or empty means "nobody" — the message posts undelivered. It is
	// never stored; the router forwards it to the published event only.
	TargetSessionID *string `json:"target_session_id" validate:"omitempty,max=128"`

	// SessionID and SessionCwd let the channel attribute an assistant reply to
	// its own Claude Code session. Ignored for a user message, whose session is
	// the resolved TargetSessionID with cwd taken from the registry. Persisted.
	SessionID  *string `json:"session_id" validate:"omitempty,max=128"`
	SessionCwd *string `json:"session_cwd" validate:"omitempty,max=4096"`
}

// DispatchRequest addresses an already-stored message at a live session.
type DispatchRequest struct {
	TargetSessionID string `json:"target_session_id" validate:"required,max=128"`
}

type ConversationListResponse struct {
	Conversations    []*entity.Conversation `json:"conversations"`
	UnansweredCounts []int                  `json:"unanswered_counts,omitempty"`
}

type MessageListResponse struct {
	Messages []*entity.Message `json:"messages"`
}
