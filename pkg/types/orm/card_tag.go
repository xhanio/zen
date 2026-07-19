package orm

type CardTag struct {
	CardID string `gorm:"primaryKey;column:card_id"`
	TagID  string `gorm:"primaryKey;column:tag_id"`
}

func (CardTag) TableName() string { return "card_tags" }
