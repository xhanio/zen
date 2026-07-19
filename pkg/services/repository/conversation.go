package repository

import (
	"context"
	"time"

	"github.com/xhanio/errors"
	"gorm.io/gorm"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/orm"
)

func ormToEntityConversation(o *orm.Conversation) *entity.Conversation {
	return &entity.Conversation{
		ID:            o.ID,
		Title:         o.Title,
		AnchorKind:    o.AnchorKind,
		AnchorID:      o.AnchorID,
		CreatedAt:     o.CreatedAt,
		LastMessageAt: o.LastMessageAt,
	}
}

func entityToOrmConversation(e *entity.Conversation) *orm.Conversation {
	return &orm.Conversation{
		ID:            e.ID,
		Title:         e.Title,
		AnchorKind:    e.AnchorKind,
		AnchorID:      e.AnchorID,
		CreatedAt:     e.CreatedAt,
		LastMessageAt: e.LastMessageAt,
	}
}

func (m *manager) CreateConversation(ctx context.Context, c *entity.Conversation) error {
	if err := m.db.FromContext(ctx).Create(entityToOrmConversation(c)).Error; err != nil {
		return errors.DBFailed.Wrapf(err, "failed to create conversation")
	}
	return nil
}

func (m *manager) GetConversation(ctx context.Context, id string) (*entity.Conversation, error) {
	var row orm.Conversation
	err := m.db.FromContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound.Newf("conversation %q not found", id)
	}
	if err != nil {
		return nil, errors.DBFailed.Wrapf(err, "failed to get conversation %q", id)
	}
	return ormToEntityConversation(&row), nil
}

func (m *manager) ListConversations(ctx context.Context, anchorKind, anchorID *string, limit int) ([]*entity.Conversation, error) {
	q := m.db.FromContext(ctx).Model(&orm.Conversation{})
	if anchorKind != nil {
		q = q.Where("anchor_kind = ?", *anchorKind)
	}
	if anchorID != nil {
		q = q.Where("anchor_id = ?", *anchorID)
	}
	q = q.Order("last_message_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []orm.Conversation
	if err := q.Find(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Conversation, 0, len(rows))
	for i := range rows {
		out = append(out, ormToEntityConversation(&rows[i]))
	}
	return out, nil
}

func (m *manager) ListConversationsByAnchor(ctx context.Context, anchorKind, anchorID string) ([]*entity.Conversation, error) {
	return m.ListConversations(ctx, &anchorKind, &anchorID, 0)
}

// ListPendingConversations returns conversations whose latest message is from
// role='user'. Returned counts[i] is the number of trailing user messages in
// conversations[i] with no following assistant message.
func (m *manager) ListPendingConversations(ctx context.Context, limit int) ([]*entity.Conversation, []int, error) {
	type row struct {
		orm.Conversation
		UnansweredCount int `gorm:"column:unanswered_count"`
	}
	var rows []row
	q := m.db.FromContext(ctx).
		Table("conversations c").
		Select(`c.*, (
			SELECT COUNT(*) FROM messages m
			WHERE m.conversation_id = c.id
			  AND m.role = 'user'
			  AND m.created_at > IFNULL(
				(SELECT MAX(m2.created_at) FROM messages m2
				 WHERE m2.conversation_id = c.id AND m2.role = 'assistant'),
				'0000-01-01'
			  )
		) AS unanswered_count`).
		Where(`EXISTS (
			SELECT 1 FROM messages m WHERE m.conversation_id = c.id AND m.role = 'user'
			  AND m.created_at > IFNULL(
				(SELECT MAX(m2.created_at) FROM messages m2
				 WHERE m2.conversation_id = c.id AND m2.role = 'assistant'),
				'0000-01-01'
			  )
		)`).
		Order("c.last_message_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, nil, errors.DBFailed.Wrap(err)
	}
	out := make([]*entity.Conversation, 0, len(rows))
	counts := make([]int, 0, len(rows))
	for i := range rows {
		c := rows[i].Conversation
		out = append(out, ormToEntityConversation(&c))
		counts = append(counts, rows[i].UnansweredCount)
	}
	return out, counts, nil
}

func (m *manager) UpdateConversation(ctx context.Context, c *entity.Conversation) error {
	if c.LastMessageAt.IsZero() {
		c.LastMessageAt = time.Now()
	}
	res := m.db.FromContext(ctx).Save(entityToOrmConversation(c))
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to update conversation %q", c.ID)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("conversation %q not found", c.ID)
	}
	return nil
}

func (m *manager) DeleteConversation(ctx context.Context, id string) error {
	res := m.db.FromContext(ctx).Delete(&orm.Conversation{ID: id})
	if res.Error != nil {
		return errors.DBFailed.Wrapf(res.Error, "failed to delete conversation %q", id)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound.Newf("conversation %q not found", id)
	}
	return nil
}
