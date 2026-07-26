package entity

import "time"

// CardSnapshot is a card's complete state at one moment. Rows are never
// mutated. CardTitle is denormalized at read time for list rendering and is
// not stored.
type CardSnapshot struct {
	ID             string    `json:"id"`
	CardID         string    `json:"card_id"`
	CardTitle      string    `json:"card_title,omitempty"`
	Seq            int       `json:"seq"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Content        string    `json:"content"`
	Format         string    `json:"format"`
	Actor          string    `json:"actor"`
	ConversationID *string   `json:"conversation_id"`
	ChangeKind     string    `json:"change_kind"`
	Diff           string    `json:"diff"`
	DiffTruncated  bool      `json:"diff_truncated"`
	LinesAdded     int       `json:"lines_added"`
	LinesRemoved   int       `json:"lines_removed"`
	CreatedAt      time.Time `json:"created_at"`
}

// SnapshotAttribution travels with every card mutation. Actor is derived from
// the transport (the X-Zen-Actor header), never from the request body;
// ConversationID is passed explicitly by the agent and is nil for hand edits.
// The zero value means "user, no conversation", which is the correct default
// for a direct SPA edit.
type SnapshotAttribution struct {
	Actor          string
	ConversationID *string
}

// ActorOrDefault returns the actor, defaulting to "user".
func (a SnapshotAttribution) ActorOrDefault() string {
	if a.Actor == "" {
		return "user"
	}
	return a.Actor
}

// DiffSpan is a run of text inside a line or field value. Op is "eq", "del",
// or "add".
type DiffSpan struct {
	Op   string `json:"op"`
	Text string `json:"text"`
}

// DiffField is a scalar field that changed (title, summary, format).
type DiffField struct {
	Key   string     `json:"key"`
	Spans []DiffSpan `json:"spans"`
}

// DiffLine is one rendered line of the body diff. Op is "ctx", "del", or
// "add". Spans is populated only for del/add lines that pair with an
// opposite-op neighbor; a wholly inserted or deleted line has none.
type DiffLine struct {
	Op    string     `json:"op"`
	Text  string     `json:"text"`
	Spans []DiffSpan `json:"spans,omitempty"`
}

type CardDiff struct {
	Fields []DiffField `json:"fields"`
	Lines  []DiffLine  `json:"lines"`
}
