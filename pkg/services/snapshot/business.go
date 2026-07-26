package snapshot

import (
	"context"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func (m *manager) List(ctx context.Context, f api.ListSnapshotsRequest) ([]*entity.CardSnapshot, error) {
	for _, id := range []*string{f.CardID, f.ConversationID} {
		if id == nil {
			continue
		}
		if err := ulidutil.Parse(*id); err != nil {
			return nil, errors.Wrap(err)
		}
	}
	out, err := m.repo.ListCardSnapshots(ctx, f)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if out == nil {
		out = []*entity.CardSnapshot{}
	}
	return out, nil
}

func (m *manager) Get(ctx context.Context, id string) (*api.SnapshotDetailResponse, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}
	s, err := m.repo.GetCardSnapshot(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	resp := &api.SnapshotDetailResponse{Snapshot: s}
	if s.Seq > 1 {
		prev, err := m.repo.GetCardSnapshotBySeq(ctx, s.CardID, s.Seq-1)
		if err != nil {
			// A gap in the chain is not fatal: render what we have rather
			// than failing the whole request.
			if !errors.Is(err, errors.NotFound) {
				return nil, errors.Wrap(err)
			}
		} else {
			resp.Previous = prev
		}
	}
	return resp, nil
}
