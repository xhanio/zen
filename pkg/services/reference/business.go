package reference

import (
	"context"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

const maxSelectionLen = 5000

func (m *manager) Create(ctx context.Context, req api.CreateReferenceRequest) (*entity.Reference, error) {
	if req.SourceCardID == req.DerivedCardID {
		return nil, errors.BadRequest.Newf("source and derived card must differ")
	}
	if err := ulidutil.Parse(req.SourceCardID); err != nil {
		return nil, errors.BadRequest.Wrap(err)
	}
	if err := ulidutil.Parse(req.DerivedCardID); err != nil {
		return nil, errors.BadRequest.Wrap(err)
	}
	if err := ulidutil.Parse(req.ConversationID); err != nil {
		return nil, errors.BadRequest.Wrap(err)
	}

	// Inherit from the message when one is named. The SPA captured that copy
	// at drag time together with its offsets; a caller-supplied excerpt is a
	// retype, and any normalization in it (smart quotes, collapsed spaces,
	// a trimmed newline) yields a highlight that never paints.
	selectionText := req.SelectionText
	var start, end, seq *int
	if req.MessageID != nil {
		if err := ulidutil.Parse(*req.MessageID); err != nil {
			return nil, errors.BadRequest.Wrap(err)
		}
		msg, err := m.repo.GetMessage(ctx, *req.MessageID)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if msg.ConversationID != req.ConversationID {
			return nil, errors.BadRequest.Newf(
				"message %q belongs to conversation %q, not %q",
				*req.MessageID, msg.ConversationID, req.ConversationID)
		}
		if msg.SelectionText != nil && *msg.SelectionText != "" {
			selectionText = *msg.SelectionText
		}
		start, end, seq = msg.SelectionStart, msg.SelectionEnd, msg.SelectionSeq
	}

	if selectionText == "" {
		return nil, errors.BadRequest.Newf("selection_text is required when no message_id supplies one")
	}
	if len(selectionText) > maxSelectionLen {
		return nil, errors.BadRequest.Newf("selection_text exceeds %d chars", maxSelectionLen)
	}

	if _, err := m.cards.Get(ctx, req.SourceCardID); err != nil {
		return nil, errors.Wrap(err)
	}
	if _, err := m.cards.Get(ctx, req.DerivedCardID); err != nil {
		return nil, errors.Wrap(err)
	}
	if _, err := m.conv.Get(ctx, req.ConversationID); err != nil {
		return nil, errors.Wrap(err)
	}
	convID := req.ConversationID
	r := &entity.Reference{
		ID:             ulidutil.New(),
		SourceCardID:   req.SourceCardID,
		DerivedCardID:  req.DerivedCardID,
		ConversationID: &convID,
		SelectionText:  selectionText,
		SelectionStart: start,
		SelectionEnd:   end,
		SelectionSeq:   seq,
		CreatedAt:      time.Now(),
	}
	if err := m.repo.CreateReference(ctx, r); err != nil {
		return nil, errors.Wrap(err)
	}
	return r, nil
}

func (m *manager) Get(ctx context.Context, id string) (*entity.Reference, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.BadRequest.Wrap(err)
	}
	return m.repo.GetReference(ctx, id)
}

func (m *manager) List(ctx context.Context, f api.ListReferencesRequest) ([]*entity.Reference, error) {
	if f.SourceCardID == nil && f.DerivedCardID == nil && f.ConversationID == nil {
		return nil, errors.BadRequest.Newf("at least one filter is required")
	}
	return m.repo.ListReferences(ctx, f)
}

func (m *manager) Delete(ctx context.Context, id string) error {
	if err := ulidutil.Parse(id); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	return m.repo.DeleteReference(ctx, id)
}
