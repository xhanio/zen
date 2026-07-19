package entity

import "time"

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	SelectionText  *string   `json:"selection_text"`
	SessionID      *string   `json:"session_id"`
	SessionCwd     *string   `json:"session_cwd"`
	CreatedAt      time.Time `json:"created_at"`
}
