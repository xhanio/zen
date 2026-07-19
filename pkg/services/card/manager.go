package card

import (
	"path"

	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/reflectutil"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/types/model"
)

type manager struct {
	name string
	log  log.Logger
	repo repository.Repository
	tags model.Tag
	conv model.Conversation
}

func New(repo repository.Repository, tags model.Tag, conv model.Conversation, opts ...Option) Manager {
	m := &manager{
		log:  log.Default,
		repo: repo,
		tags: tags,
		conv: conv,
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

func (m *manager) Dependencies() []common.Service {
	deps := []common.Service{m.repo, m.tags}
	if m.conv != nil {
		deps = append(deps, m.conv)
	}
	return deps
}
