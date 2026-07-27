package entity

import "time"

type Reference struct {
	ID             string  `json:"id"`
	SourceCardID   string  `json:"source_card_id"`
	DerivedCardID  string  `json:"derived_card_id"`
	ConversationID *string `json:"conversation_id"`
	SelectionText  string  `json:"selection_text"`
	// SelectionStart/End are character offsets into the source card's
	// rendered text. Nil means "no range": the highlight falls back to a
	// unique-text match. SelectionSeq is the card_snapshots.seq the
	// selection was taken against — a label, not a paint condition.
	SelectionStart *int      `json:"selection_start"`
	SelectionEnd   *int      `json:"selection_end"`
	SelectionSeq   *int      `json:"selection_seq"`
	CreatedAt      time.Time `json:"created_at"`
}
