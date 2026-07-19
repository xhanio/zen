package entity

import "time"

type LevelEntry struct {
	ID     string  `json:"id"`
	Weight float64 `json:"weight"`
	Name   string  `json:"name"`
}

type Group struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Rule         string       `json:"rule"`
	Position     int          `json:"position"`
	LevelCatalog []LevelEntry `json:"level_catalog"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
