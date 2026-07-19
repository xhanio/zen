package repository

import (
	"context"

	"github.com/xhanio/errors"
	"gorm.io/gorm"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/orm"
)

func ormToEntityMessage(o *orm.Message) *entity.Message {
	return &entity.Message{
		ID:             o.ID,
		ConversationID: o.ConversationID,
		Role:           o.Role,
		Content:        o.Content,
		SelectionText:  o.SelectionText,
		SessionID:      o.SessionID,
		SessionCwd:     o.SessionCwd,
		CreatedAt:      o.CreatedAt,
	}
}

func entityToOrmMessage(e *entity.Message) *orm.Message {
	return &orm.Message{
		ID:             e.ID,
		ConversationID: e.ConversationID,
		Role:           e.Role,
		Content:        e.Content,
		SelectionText:  e.SelectionText,
		SessionID:      e.SessionID,
		SessionCwd:     e.SessionCwd,
		CreatedAt:      e.CreatedAt,
	}
}

func (m *manager) CreateMessage(ctx context.Context, msg *entity.Message) error {
	if err := m.db.FromContext(ctx).Create(entityToOrmMessage(msg)).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to create message")
	}
	return nil
}

func (m *manager) SetMessageSession(ctx context.Context, messageID, sessionID, cwd string) error {
	err := m.db.FromContext(ctx).
		Model(&orm.Message{}).
		Where("id = ? AND session_id IS NULL", messageID).
		Updates(map[string]any{"session_id": sessionID, "session_cwd": cwd}).Error
	if err != nil {
		return errors.DBFailed.Wrapf(err, "failed to set message session")
	}
	return nil
}

func (m *manager) GetMessage(ctx context.Context, id string) (*entity.Message, error) {
	var row orm.Message
	err := m.db.FromContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound.Newf("message %q not found", id)
	}
	if err != nil {
		return nil, errors.DBFailed.Wrapf(err, "failed to get message %q", id)
	}
	return ormToEntityMessage(&row), nil
}

func (m *manager) ListMessages(ctx context.Context, conversationID string, limit int) ([]*entity.Message, error) {
	q := m.db.FromContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []orm.Message
	if err := q.Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Message, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityMessage(&rows[i]))
	}
	return out, nil
}

func (m *manager) ListMessagesAfter(ctx context.Context, conversationID, afterID string, limit int) ([]*entity.Message, error) {
	q := m.db.FromContext(ctx).Where("conversation_id = ?", conversationID)
	if afterID != "" {
		q = q.Where("id > ?", afterID)
	}
	q = q.Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []orm.Message
	if err := q.Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Message, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityMessage(&rows[i]))
	}
	return out, nil
}

func (m *manager) LatestMessage(ctx context.Context, conversationID string) (*entity.Message, error) {
	var row orm.Message
	err := m.db.FromContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound.Newf("no messages in conversation %q", conversationID)
	}
	if err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	return ormToEntityMessage(&row), nil
}

func (m *manager) CountUserMessagesByAnchorCard(ctx context.Context, cardID string) (int, error) {
	var count int64
	err := m.db.FromContext(ctx).
		Table("messages").
		Joins("JOIN conversations c ON c.id = messages.conversation_id").
		Where("c.anchor_kind = ? AND c.anchor_id = ? AND messages.role = ?", "card", cardID, "user").
		Count(&count).Error
	if err != nil {
		return 0, errors.DBFailed.Wrap(err)
	}
	return int(count), nil
}
