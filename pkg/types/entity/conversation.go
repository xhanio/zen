package entity

import "time"

type Conversation struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	AnchorKind    *string   `json:"anchor_kind"`
	AnchorID      *string   `json:"anchor_id"`
	CreatedAt     time.Time `json:"created_at"`
	LastMessageAt time.Time `json:"last_message_at"`
}
