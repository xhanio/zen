package orm

type Tag struct {
	ID      string `gorm:"primaryKey;column:id"`
	GroupID string `gorm:"column:group_id;not null"`
	Name    string `gorm:"column:name;not null"`
}

func (Tag) TableName() string { return "tags" }
