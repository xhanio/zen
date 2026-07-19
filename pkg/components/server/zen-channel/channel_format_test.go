package zenchannel

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/types/entity"
)

func strptr(s string) *string { return &s }

func TestFormatNotification_BasicNoSelection(t *testing.T) {
	ts := time.Date(2026, 6, 27, 19, 30, 0, 0, time.UTC)
	ev := &entity.ConversationEvent{
		ConversationID: "01CONV",
		MessageID:      "01MSG",
		Role:           "user",
		Content:        "what does X mean?",
		CreatedAt:      ts,
	}
	conv := &entity.Conversation{ID: "01CONV", Title: "t", AnchorKind: strptr("card"), AnchorID: strptr("01CARD")}

	got := FormatNotification(ev, conv)
	if got.Content != "what does X mean?" {
		t.Fatalf("content: %q", got.Content)
	}
	b, _ := json.Marshal(got.Meta)
	want := `{"anchor_id":"01CARD","anchor_kind":"card","conversation_id":"01CONV","has_selection":"false","message_id":"01MSG","ts":"2026-06-27T19:30:00Z"}`
	if string(b) != want {
		t.Fatalf("meta:\n got %s\nwant %s", string(b), want)
	}
}

func TestFormatNotification_SelectionPrependsQuotedBlock(t *testing.T) {
	ev := &entity.ConversationEvent{
		ConversationID: "01CONV",
		MessageID:      "01MSG",
		Role:           "user",
		Content:        "what does this mean?",
		SelectionText:  strptr("FTS5's snippet() helper"),
		CreatedAt:      time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC),
	}
	conv := &entity.Conversation{ID: "01CONV", Title: "t", AnchorKind: strptr("card"), AnchorID: strptr("01CARD")}

	got := FormatNotification(ev, conv)
	want := "> selected: \"FTS5's snippet() helper\"\n\nwhat does this mean?"
	if got.Content != want {
		t.Fatalf("content:\n got %q\nwant %q", got.Content, want)
	}
	if got.Meta["has_selection"] != "true" {
		t.Fatalf("has_selection should be true; meta=%v", got.Meta)
	}
}

// Claude Code validates params.meta against z.record(z.string(), z.string()),
// so a non-string value (e.g. a bare `true`) makes the notification handler
// throw a ZodError and drop the whole stdio connection.
func TestFormatNotification_MetaValuesAreAllStrings(t *testing.T) {
	ev := &entity.ConversationEvent{
		ConversationID: "01CONV", MessageID: "01MSG", Role: "user", Content: "hi",
		SelectionText: strptr("some selection"),
		CreatedAt:     time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC),
	}
	conv := &entity.Conversation{ID: "01CONV", Title: "t", AnchorKind: strptr("card"), AnchorID: strptr("01CARD")}

	b, err := json.Marshal(FormatNotification(ev, conv).Meta)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	for k, v := range raw {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			t.Errorf("meta[%q] = %s; must be a JSON string", k, v)
		}
	}
}

func TestFormatNotification_NoAnchorOmitsAnchorFields(t *testing.T) {
	ev := &entity.ConversationEvent{
		ConversationID: "01CONV", MessageID: "01MSG", Role: "user", Content: "hello",
		CreatedAt: time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC),
	}
	conv := &entity.Conversation{ID: "01CONV", Title: "t"}

	got := FormatNotification(ev, conv)
	if _, ok := got.Meta["anchor_kind"]; ok {
		t.Fatal("anchor_kind should be absent")
	}
	if _, ok := got.Meta["anchor_id"]; ok {
		t.Fatal("anchor_id should be absent")
	}
}
