package api

import "github.com/xhanio/zen/pkg/types/entity"

type ListSnapshotsRequest struct {
	CardID         *string `json:"card_id,omitempty" validate:"omitempty,len=26"`
	ConversationID *string `json:"conversation_id,omitempty" validate:"omitempty,len=26"`
}

type ListSnapshotsResponse struct {
	Snapshots []*entity.CardSnapshot `json:"snapshots"`
}

// SnapshotDetailResponse carries the snapshot plus its predecessor, so the
// client can render a diff — or, when the diff was truncated, both bodies —
// without a second request. Previous is nil for a baseline.
type SnapshotDetailResponse struct {
	Snapshot *entity.CardSnapshot `json:"snapshot"`
	Previous *entity.CardSnapshot `json:"previous"`
}
