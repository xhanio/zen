package entity

import "time"

// CardSelection is one message's selected span in a card's rendered text.
// Only messages that captured real offsets appear here: an underline is a
// claim about an exact span, so there is no text-matching fallback for these
// the way there is for a reference.
type CardSelection struct {
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	SelectionText  string    `json:"selection_text"`
	SelectionStart int       `json:"selection_start"`
	SelectionEnd   int       `json:"selection_end"`
	SelectionSeq   *int      `json:"selection_seq"`
	CreatedAt      time.Time `json:"created_at"`
}
