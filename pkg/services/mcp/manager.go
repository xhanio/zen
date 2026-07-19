package mcp

import (
	"net/http"
	"path"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/reflectutil"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
)

type manager struct {
	name    string
	log     log.Logger
	backend zenbackend.Client

	server  *mcpsdk.Server
	handler http.Handler
}

func New(backend zenbackend.Client, opts ...Option) Manager {
	m := &manager{log: log.Default, backend: backend}
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
	return nil
}

func (m *manager) Handler() http.Handler {
	return m.handler
}
