package entity

import "time"

// ConversationEvent is the payload published on the messagebus when a
// message is appended to a conversation. Kind on the bus envelope is
// always ConversationEventKind; Role distinguishes user/assistant/system.
type ConversationEvent struct {
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	SelectionText  *string   `json:"selection_text,omitempty"`

	// TargetSessionID addresses this event to exactly one Claude Code session.
	// It rides on the wire and is never persisted (unlike SessionID below): the
	// messages table must not learn who a message was *routed* to. Empty means
	// the event is addressed to nobody — which is not the same as "everybody",
	// and must never be treated as a broadcast.
	TargetSessionID string `json:"target_session_id,omitempty"`

	// SessionID and SessionCwd are the persisted attribution of the message: the
	// session it belongs to and that session's working directory. Unlike
	// TargetSessionID they ARE stored on the row; here they ride along so a
	// consumer can render the badge without a refetch.
	SessionID  string `json:"session_id,omitempty"`
	SessionCwd string `json:"session_cwd,omitempty"`

	// PrevMessageID names the message immediately before this one in the same
	// conversation. A consumer whose cursor is not this id knows the bus dropped
	// something in between and refetches the span.
	//
	// Empty means "no gap possible": the first message of a conversation, and
	// every dispatched re-publish, carry none. It is never persisted.
	PrevMessageID string `json:"prev_message_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// ConversationEventKind is the Kind value used when publishing a
// ConversationEvent through messagebus.SendRawMessage.
const ConversationEventKind = "conversation_event"
