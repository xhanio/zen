package delivery

import (
	"path"
	"sync"

	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/reflectutil"
)

type manager struct {
	name string
	log  log.Logger

	mu       sync.RWMutex
	watchers map[*watcher]struct{}
}

func New(opts ...Option) Manager {
	m := &manager{
		log:      log.Default,
		watchers: make(map[*watcher]struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	m.log = m.log.By(m)
	return m
}

func (m *manager) Name() string {
	if m.name == "" {
		m.name = path.Join(reflectutil.Locate(m))
	}
	return m.name
}

// Dependencies is empty: delivery owns only in-memory state.
func (m *manager) Dependencies() []common.Service { return nil }
