package entity

import "time"

type Card struct {
	ID                   string     `json:"id"`
	Title                string     `json:"title"`
	Summary              string     `json:"summary"`
	Content              string     `json:"content"`
	Format               string     `json:"format"`
	LevelEntryID         *string    `json:"level_entry_id"`
	Genesis              string     `json:"genesis"`
	DeletedAt            *time.Time `json:"deleted_at"`
	GroupID              string     `json:"group_id"`
	Position             int        `json:"position"`
	Tags                 []string   `json:"tags"`
	ParentCardID         *string    `json:"parent_card_id"`
	SourceConversationID *string    `json:"source_conversation_id"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	ReviewGrade string     `json:"review_grade"`
	ReviewScore *float64   `json:"review_score"`
	ReviewedAt  *time.Time `json:"reviewed_at"`

	References []*Reference `json:"references"`
}
