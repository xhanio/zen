package entity

type Tag struct {
	ID        string `json:"id"`
	GroupID   string `json:"group_id"`
	Name      string `json:"name"`
	CardCount int    `json:"card_count"` // only populated by ListTags
}
