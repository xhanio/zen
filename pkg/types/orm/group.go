package orm

import "time"

type Group struct {
	ID           string    `gorm:"primaryKey;column:id"`
	Name         string    `gorm:"column:name;not null"`
	Rule         string    `gorm:"column:rule;not null;default:''"`
	Position     int       `gorm:"column:position;not null;default:0"`
	LevelCatalog string    `gorm:"column:level_catalog;not null;default:'[]'"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (Group) TableName() string { return "groups" }
