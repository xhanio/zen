package orm

import (
	"database/sql"
	"time"
)

type Card struct {
	ID                   string       `gorm:"primaryKey;column:id"`
	Title                string       `gorm:"column:title;not null"`
	Content              string       `gorm:"column:content;not null;default:''"`
	Summary              string       `gorm:"column:summary;not null;default:''"`
	Format               string       `gorm:"column:format;not null;default:'markdown'"`
	SearchHint           string       `gorm:"column:search_hint;not null;default:''"`
	LevelEntryID         *string      `gorm:"column:level_entry_id"`
	Genesis              string       `gorm:"column:genesis;not null;default:''"`
	DeletedAt            sql.NullTime `gorm:"column:deleted_at"`
	GroupID              string       `gorm:"column:group_id;not null"`
	Position             int          `gorm:"column:position;not null;default:0"`
	ParentCardID         *string      `gorm:"column:parent_card_id"`
	SourceConversationID *string      `gorm:"column:source_conversation_id"`
	CreatedAt            time.Time    `gorm:"column:created_at;not null"`
	UpdatedAt            time.Time    `gorm:"column:updated_at;not null"`
	ReviewGrade          string       `gorm:"column:review_grade;not null;default:'LGTM'"`
	ReviewedAt           sql.NullTime `gorm:"column:reviewed_at"`
}

func (Card) TableName() string { return "cards" }
