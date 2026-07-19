package zenchannel

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/xhanio/zen/pkg/types/entity"
)

// ChannelNotification is the payload of a notifications/claude/channel
// message. Direct port of the ChannelNotification shape in the (deleted) Bun
// plugins/zen/server.ts so existing zen-conversation-watcher SKILL.md still
// applies verbatim.
// Meta is a string map, not map[string]any: Claude Code validates it against
// z.record(z.string(), z.string()), and a non-string value throws a ZodError
// inside the notification handler, which tears down the stdio connection.
type ChannelNotification struct {
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta"`
}

// FormatNotification is a faithful port of formatNotification() from the Bun
// server. Wire compatibility with the previous TS version matters because the
// SPA's zen-conversation-watcher skill describes the exact tag shape.
func FormatNotification(ev *entity.ConversationEvent, conv *entity.Conversation) ChannelNotification {
	hasSelection := ev.SelectionText != nil && *ev.SelectionText != ""

	content := ev.Content
	if hasSelection {
		// Match JS JSON.stringify(sel) — double-quoted, escapes embedded
		// quotes/backslashes/control chars.
		b, _ := json.Marshal(*ev.SelectionText)
		content = "> selected: " + string(b) + "\n\n" + ev.Content
	}

	meta := map[string]string{
		"conversation_id": ev.ConversationID,
		"message_id":      ev.MessageID,
		"has_selection":   strconv.FormatBool(hasSelection),
		"ts":              ev.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if conv.AnchorKind != nil && conv.AnchorID != nil {
		meta["anchor_kind"] = *conv.AnchorKind
		meta["anchor_id"] = *conv.AnchorID
	}
	return ChannelNotification{Content: content, Meta: meta}
}
