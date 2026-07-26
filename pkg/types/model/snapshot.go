package model

import (
	"context"

	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type Snapshot interface {
	common.Service
	List(ctx context.Context, f api.ListSnapshotsRequest) ([]*entity.CardSnapshot, error)
	// Get returns the snapshot together with its predecessor (nil for seq 1),
	// so a client renders a diff — or both bodies, when the diff was
	// truncated — in one round trip.
	Get(ctx context.Context, id string) (*api.SnapshotDetailResponse, error)
}
