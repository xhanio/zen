package delivery

import "github.com/xhanio/zen/pkg/types/model"

type Manager = model.Delivery

var _ Manager = (*manager)(nil)
