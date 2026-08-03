package delivery

import (
	"sync"

	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/nameutil"
)

type manager struct {
	name string
	log  log.Logger

	mu       sync.RWMutex
	watchers map[*watcher]struct{}
}

func New(opts ...Option) Manager {
	return newManager(opts...)
}

// newManager returns the concrete manager, the form package tests construct.
func newManager(opts ...Option) *manager {
	m := &manager{
		log:      log.Default,
		watchers: make(map[*watcher]struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	m.name = nameutil.Name(m)
	m.log = m.log.By(m)
	return m
}

func (m *manager) Name() string {
	return m.name
}

// Dependencies is empty: delivery owns only in-memory state.
func (m *manager) Dependencies() []common.Service { return nil }
