package zenchannel

import (
	"context"
	"fmt"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
	"github.com/xhanio/zen/pkg/types/api"
)

// ReplyFunc posts an assistant message back to a Zen conversation. It returns
// a short success line ("sent (<msg id>)") for inclusion in the
// tools/call result text, matching what the Bun reply tool returned.
type ReplyFunc func(ctx context.Context, conversationID, content string) (string, error)

// NewReply binds a ReplyFunc to an existing zen-backend client, tagging each
// assistant message with the channel's own Claude Code session id and cwd so
// the reply is attributed to the session that produced it. Direct port of
// buildReplyTool() from plugins/zen/server.ts.
func NewReply(backend zenbackend.Client, sessionID, cwd string) ReplyFunc {
	return func(ctx context.Context, conversationID, content string) (string, error) {
		sid, wd := sessionID, cwd
		msg, err := backend.Conversation().AppendMessage(ctx, conversationID, api.AppendMessageRequest{
			Role:       "assistant",
			Content:    content,
			SessionID:  &sid,
			SessionCwd: &wd,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("sent (%s)", msg.ID), nil
	}
}
