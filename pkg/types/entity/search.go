package entity

// SearchHit is a single result from full-text search. Kind discriminates
// whether the hit comes from cards or conversation messages.
// For kind='message', Title carries the parent conversation title and
// ConversationID points back to it; GroupID stays empty.
type SearchHit struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Title string `json:"title"`
	// TitlePath is the ancestor card titles, root-first, excluding this card
	// itself. Empty for a top-level card or a message hit. The UI renders it as
	// a breadcrumb ahead of Title (e.g. "… > parent > title").
	TitlePath      []string `json:"title_path,omitempty"`
	Snippet        string   `json:"snippet"`
	GroupID        string   `json:"group_id"`
	ConversationID *string  `json:"conversation_id,omitempty"`
}
