package presence

import (
	"sync"

	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/nameutil"
)

type manager struct {
	name string
	log  log.Logger

	mu sync.RWMutex
	// bySession is the registry: a session has at most one live channel.
	bySession map[string]*registration
	watchers  map[*watcher]struct{}
}

func New(opts ...Option) Manager {
	return newManager(opts...)
}

// newManager returns the concrete manager, the form package tests construct.
func newManager(opts ...Option) *manager {
	m := &manager{
		log:       log.Default,
		bySession: make(map[string]*registration),
		watchers:  make(map[*watcher]struct{}),
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

// Dependencies is empty: presence owns only in-memory state.
func (m *manager) Dependencies() []common.Service { return nil }
