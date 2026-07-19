package orm

import "time"

type Message struct {
	ID             string    `gorm:"primaryKey;column:id"`
	ConversationID string    `gorm:"column:conversation_id;not null;index"`
	Role           string    `gorm:"column:role;not null"`
	Content        string    `gorm:"column:content;not null"`
	SelectionText  *string   `gorm:"column:selection_text"`
	SessionID      *string   `gorm:"column:session_id"`
	SessionCwd     *string   `gorm:"column:session_cwd"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
}

func (Message) TableName() string { return "messages" }
