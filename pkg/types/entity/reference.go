package entity

import "time"

type Reference struct {
	ID             string    `json:"id"`
	SourceCardID   string    `json:"source_card_id"`
	DerivedCardID  string    `json:"derived_card_id"`
	ConversationID *string   `json:"conversation_id"`
	SelectionText  string    `json:"selection_text"`
	CreatedAt      time.Time `json:"created_at"`
}
