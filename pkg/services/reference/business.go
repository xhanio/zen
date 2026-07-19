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
	if req.SelectionText == "" {
		return nil, errors.BadRequest.Newf("selection_text is required")
	}
	if len(req.SelectionText) > maxSelectionLen {
		return nil, errors.BadRequest.Newf("selection_text exceeds %d chars", maxSelectionLen)
	}
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
		SelectionText:  req.SelectionText,
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
