package orm

import "time"

type Conversation struct {
	ID            string    `gorm:"primaryKey;column:id"`
	Title         string    `gorm:"column:title;not null"`
	AnchorKind    *string   `gorm:"column:anchor_kind"`
	AnchorID      *string   `gorm:"column:anchor_id"`
	CreatedAt     time.Time `gorm:"column:created_at;not null"`
	LastMessageAt time.Time `gorm:"column:last_message_at;not null"`
}

func (Conversation) TableName() string { return "conversations" }
