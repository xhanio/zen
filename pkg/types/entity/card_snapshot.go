package entity

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
