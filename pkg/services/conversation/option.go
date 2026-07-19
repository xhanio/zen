package conversation

import (
	"github.com/xhanio/framingo/pkg/types/model"
	"github.com/xhanio/framingo/pkg/utils/log"
)

type Option func(*manager)

func WithLogger(logger log.Logger) Option {
	return func(m *manager) {
		m.log = logger.By(m)
	}
}

func WithMessageBus(bus model.MessageBus) Option {
	return func(m *manager) {
		m.bus = bus
	}
}
