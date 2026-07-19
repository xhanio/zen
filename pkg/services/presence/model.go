package presence

import "github.com/xhanio/zen/pkg/types/model"

type Manager = model.Presence

var _ Manager = (*manager)(nil)
