package orm

import "time"

type CardSnapshot struct {
	ID             string    `gorm:"primaryKey;column:id"`
	CardID         string    `gorm:"column:card_id;not null;index"`
	Seq            int       `gorm:"column:seq;not null"`
	Title          string    `gorm:"column:title;not null"`
	Summary        string    `gorm:"column:summary;not null;default:''"`
	Content        string    `gorm:"column:content;not null;default:''"`
	Format         string    `gorm:"column:format;not null"`
	Actor          string    `gorm:"column:actor;not null;default:'user'"`
	ConversationID *string   `gorm:"column:conversation_id"`
	ChangeKind     string    `gorm:"column:change_kind;not null"`
	Diff           string    `gorm:"column:diff;not null;default:''"`
	DiffTruncated  bool      `gorm:"column:diff_truncated;not null;default:false"`
	LinesAdded     int       `gorm:"column:lines_added;not null;default:0"`
	LinesRemoved   int       `gorm:"column:lines_removed;not null;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
}

func (CardSnapshot) TableName() string { return "card_snapshots" }
