package conversation

import (
	"github.com/xhanio/framingo/pkg/types/common"
	framodel "github.com/xhanio/framingo/pkg/types/model"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/nameutil"

	"github.com/xhanio/zen/pkg/services/repository"
)

type manager struct {
	name string
	log  log.Logger
	repo repository.Repository
	bus  framodel.MessageBus
}

func New(repo repository.Repository, opts ...Option) Manager {
	return newManager(repo, opts...)
}

// newManager returns the concrete manager, the form package tests construct.
func newManager(repo repository.Repository, opts ...Option) *manager {
	m := &manager{
		log:  log.Default,
		repo: repo,
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

func (m *manager) Dependencies() []common.Service {
	deps := []common.Service{m.repo}
	if m.bus != nil {
		deps = append(deps, m.bus)
	}
	return deps
}
