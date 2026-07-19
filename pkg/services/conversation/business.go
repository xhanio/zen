package conversation

import (
	"context"
	"strings"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

const (
	maxTitleLen     = 80
	maxContentLen   = 1 << 20 // 1 MB
	maxSelectionLen = 10_000
)

func (m *manager) Create(ctx context.Context, title string, anchorKind, anchorID *string) (*entity.Conversation, error) {
	title = strings.TrimSpace(title)
	if len(title) > maxTitleLen {
		return nil, errors.BadRequest.Newf("conversation title must be %d chars or fewer", maxTitleLen)
	}
	if (anchorKind == nil) != (anchorID == nil) {
		return nil, errors.BadRequest.Newf("anchor_kind and anchor_id must both be set or both nil")
	}
	if anchorKind != nil {
		if err := m.checkAnchorExists(ctx, *anchorKind, *anchorID); err != nil {
			return nil, errors.Wrap(err)
		}
	}
	now := time.Now()
	c := &entity.Conversation{
		ID:            ulidutil.New(),
		Title:         title,
		AnchorKind:    anchorKind,
		AnchorID:      anchorID,
		CreatedAt:     now,
		LastMessageAt: now,
	}
	if err := m.repo.CreateConversation(ctx, c); err != nil {
		return nil, errors.Wrap(err)
	}
	return c, nil
}

func (m *manager) checkAnchorExists(ctx context.Context, kind, id string) error {
	if err := ulidutil.Parse(id); err != nil {
		return errors.Wrap(err)
	}
	switch kind {
	case "card":
		_, err := m.repo.GetCard(ctx, id)
		return errors.Wrap(err)
	case "group":
		_, err := m.repo.GetGroup(ctx, id)
		return errors.Wrap(err)
	default:
		return errors.BadRequest.Newf("anchor_kind %q is invalid", kind)
	}
}

func (m *manager) Get(ctx context.Context, id string) (*entity.Conversation, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}
	return m.repo.GetConversation(ctx, id)
}

func (m *manager) List(ctx context.Context, anchorKind, anchorID *string, pending bool, limit int) ([]*entity.Conversation, []int, error) {
	if pending {
		return m.repo.ListPendingConversations(ctx, limit)
	}
	cs, err := m.repo.ListConversations(ctx, anchorKind, anchorID, limit)
	return cs, nil, err
}

func (m *manager) UpdateTitle(ctx context.Context, id, newTitle string) (*entity.Conversation, error) {
	newTitle = strings.TrimSpace(newTitle)
	if newTitle == "" {
		return nil, errors.BadRequest.Newf("title cannot be empty")
	}
	if len(newTitle) > maxTitleLen {
		return nil, errors.BadRequest.Newf("title must be %d chars or fewer", maxTitleLen)
	}
	c, err := m.repo.GetConversation(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	c.Title = newTitle
	if err := m.repo.UpdateConversation(ctx, c); err != nil {
		return nil, errors.Wrap(err)
	}
	return c, nil
}

func (m *manager) Delete(ctx context.Context, id string) error {
	if err := ulidutil.Parse(id); err != nil {
		return errors.Wrap(err)
	}
	return m.repo.DeleteConversation(ctx, id)
}

func (m *manager) AppendMessage(ctx context.Context, conversationID, role, content string, selectionText *string, opts ...model.AppendOption) (*entity.Message, error) {
	var appendOpts model.AppendOptions
	for _, opt := range opts {
		opt(&appendOpts)
	}
	if err := ulidutil.Parse(conversationID); err != nil {
		return nil, errors.Wrap(err)
	}
	switch role {
	case "user", "assistant", "system":
	default:
		return nil, errors.BadRequest.Newf("role %q is invalid", role)
	}
	if len(content) > maxContentLen {
		return nil, errors.BadRequest.Newf("message content exceeds 1 MB limit")
	}
	if selectionText != nil {
		if role != "user" {
			return nil, errors.BadRequest.Newf("selection_text is only allowed on user messages")
		}
		if len(*selectionText) > maxSelectionLen {
			return nil, errors.BadRequest.Newf("selection_text must be %d chars or fewer", maxSelectionLen)
		}
	}

	var msg *entity.Message
	var prevID string
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		c, err := m.repo.GetConversation(txCtx, conversationID)
		if err != nil {
			return errors.Wrap(err)
		}

		// The previous message in this conversation, if any. LatestMessage
		// returns NotFound on an empty conversation, which is not an error here:
		// the first message simply has no predecessor.
		if prev, err := m.repo.LatestMessage(txCtx, conversationID); err == nil {
			prevID = prev.ID
		} else if !errors.Is(err, errors.NotFound) {
			return errors.Wrap(err)
		}

		now := time.Now()
		msg = &entity.Message{
			ID:             ulidutil.New(),
			ConversationID: conversationID,
			Role:           role,
			Content:        content,
			SelectionText:  selectionText,
			CreatedAt:      now,
		}
		if appendOpts.SessionID != "" {
			sid := appendOpts.SessionID
			msg.SessionID = &sid
			if appendOpts.SessionCwd != "" {
				cwd := appendOpts.SessionCwd
				msg.SessionCwd = &cwd
			}
		}
		if err := m.repo.CreateMessage(txCtx, msg); err != nil {
			return errors.Wrap(err)
		}
		c.LastMessageAt = now
		if c.Title == "" && role == "user" {
			c.Title = truncateForTitle(content, maxTitleLen)
		}
		if err := m.repo.UpdateConversation(txCtx, c); err != nil {
			return errors.Wrap(err)
		}

		// v0.12 auto-upgrade: after a user message on a card-anchored
		// conversation, recount user messages for that card and raise the
		// grade floor. Monotonic — never lowers. Manual set above the floor
		// is preserved. Trashed anchor cards are skipped defensively.
		if role == "user" && c.AnchorKind != nil && *c.AnchorKind == "card" && c.AnchorID != nil {
			anchor, err := m.repo.GetCard(txCtx, *c.AnchorID)
			if err == nil && anchor.DeletedAt == nil {
				count, err := m.repo.CountUserMessagesByAnchorCard(txCtx, *c.AnchorID)
				if err != nil {
					return errors.Wrap(err)
				}
				floor := autoFloorForCount(count)
				if gradeRank(anchor.ReviewGrade) < gradeRank(floor) {
					anchor.ReviewGrade = floor
					anchorAt := now.UTC()
					anchor.ReviewedAt = &anchorAt
					if err := m.repo.UpdateCard(txCtx, anchor); err != nil {
						return errors.Wrap(err)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if m.bus != nil {
		ev := &entity.ConversationEvent{
			ConversationID:  msg.ConversationID,
			MessageID:       msg.ID,
			Role:            msg.Role,
			Content:         msg.Content,
			SelectionText:   msg.SelectionText,
			TargetSessionID: appendOpts.TargetSessionID,
			SessionID:       appendOpts.SessionID,
			SessionCwd:      appendOpts.SessionCwd,
			PrevMessageID:   prevID,
			CreatedAt:       msg.CreatedAt,
		}
		m.bus.SendRawMessage(ctx, m, entity.ConversationEventKind, ev)
	}
	return msg, nil
}

func (m *manager) GetMessagesAfter(ctx context.Context, conversationID, afterID string, limit int) ([]*entity.Message, error) {
	if err := ulidutil.Parse(conversationID); err != nil {
		return nil, errors.Wrap(err)
	}
	if afterID != "" {
		if err := ulidutil.Parse(afterID); err != nil {
			return nil, errors.BadRequest.Wrap(err)
		}
	}
	return m.repo.ListMessagesAfter(ctx, conversationID, afterID, limit)
}

func (m *manager) DispatchMessage(ctx context.Context, conversationID, messageID, targetSessionID, targetCwd string) error {
	if err := ulidutil.Parse(conversationID); err != nil {
		return errors.Wrap(err)
	}
	if err := ulidutil.Parse(messageID); err != nil {
		return errors.Wrap(err)
	}
	if targetSessionID == "" {
		return errors.BadRequest.Newf("target_session_id is required")
	}
	if m.bus == nil {
		return errors.Newf("message bus is not configured")
	}

	msg, err := m.repo.GetMessage(ctx, messageID)
	if err != nil {
		return errors.Wrap(err)
	}
	// GetMessage keys on the message id alone, so a caller could otherwise
	// dispatch a message that belongs to a different conversation.
	if msg.ConversationID != conversationID {
		return errors.BadRequest.Newf("message %s is not in conversation %s", messageID, conversationID)
	}
	// Only a user message is delivered to a session. An assistant reply travels
	// the other way and has no target.
	if msg.Role != "user" {
		return errors.BadRequest.Newf("only user messages can be dispatched, got %q", msg.Role)
	}

	// Set-once attribution: an undelivered message (posted with no session) is
	// attributed the first time it is dispatched. A message that already has a
	// session keeps it, so resending to another session for delivery leaves the
	// record of who it was first addressed to intact.
	sessionID, sessionCwd := targetSessionID, targetCwd
	if msg.SessionID == nil {
		if err := m.repo.SetMessageSession(ctx, messageID, targetSessionID, targetCwd); err != nil {
			return errors.Wrap(err)
		}
	} else {
		sessionID = *msg.SessionID
		sessionCwd = ""
		if msg.SessionCwd != nil {
			sessionCwd = *msg.SessionCwd
		}
	}

	// Republish, never insert. The row already exists; this is the same message
	// finding a session, not a new one.
	//
	// No PrevMessageID: this republishes a message the consumer already has, and
	// a prev from the past would look like a gap.
	m.bus.SendRawMessage(ctx, m, entity.ConversationEventKind, &entity.ConversationEvent{
		ConversationID:  msg.ConversationID,
		MessageID:       msg.ID,
		Role:            msg.Role,
		Content:         msg.Content,
		SelectionText:   msg.SelectionText,
		TargetSessionID: targetSessionID,
		SessionID:       sessionID,
		SessionCwd:      sessionCwd,
		CreatedAt:       msg.CreatedAt,
	})
	return nil
}

func truncateForTitle(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func (m *manager) GetMessages(ctx context.Context, conversationID string, limit int) ([]*entity.Message, error) {
	if err := ulidutil.Parse(conversationID); err != nil {
		return nil, errors.Wrap(err)
	}
	return m.repo.ListMessages(ctx, conversationID, limit)
}

func (m *manager) DeleteByAnchor(ctx context.Context, anchorKind, anchorID string) error {
	cs, err := m.repo.ListConversationsByAnchor(ctx, anchorKind, anchorID)
	if err != nil {
		return errors.Wrap(err)
	}
	return m.repo.Transaction(ctx, func(txCtx context.Context) error {
		for _, c := range cs {
			if err := m.repo.DeleteConversation(txCtx, c.ID); err != nil {
				return errors.Wrap(err)
			}
		}
		return nil
	})
}
