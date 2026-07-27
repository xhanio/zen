package entity

import "time"

type Message struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	Role           string  `json:"role"`
	Content        string  `json:"content"`
	SelectionText  *string `json:"selection_text"`
	// Character offsets into the anchored card's rendered text, captured by
	// the SPA at drag time. A reference created with this message's id
	// inherits them, along with SelectionText.
	SelectionStart *int      `json:"selection_start"`
	SelectionEnd   *int      `json:"selection_end"`
	SelectionSeq   *int      `json:"selection_seq"`
	SessionID      *string   `json:"session_id"`
	SessionCwd     *string   `json:"session_cwd"`
	CreatedAt      time.Time `json:"created_at"`
}
