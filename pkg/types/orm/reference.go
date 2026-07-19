package orm

import (
	"database/sql"
	"time"
)

type Reference struct {
	ID             string         `gorm:"primaryKey;column:id"`
	SourceCardID   string         `gorm:"column:source_card_id;not null"`
	DerivedCardID  string         `gorm:"column:derived_card_id;not null"`
	ConversationID sql.NullString `gorm:"column:conversation_id"`
	SelectionText  string         `gorm:"column:selection_text;not null"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null"`
}

func (Reference) TableName() string { return "card_references" }
