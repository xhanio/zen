package reference

import "github.com/xhanio/framingo/pkg/utils/log"

type Option func(*manager)

func WithLogger(l log.Logger) Option {
	return func(m *manager) { m.log = l }
}
