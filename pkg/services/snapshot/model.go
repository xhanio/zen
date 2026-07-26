package snapshot

import "github.com/xhanio/zen/pkg/types/model"

type Manager = model.Snapshot

var _ Manager = (*manager)(nil)
