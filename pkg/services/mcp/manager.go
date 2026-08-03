package mcp

import (
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/nameutil"

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
	return newManager(backend, opts...)
}

// newManager returns the concrete manager, the form package tests construct.
func newManager(backend zenbackend.Client, opts ...Option) *manager {
	m := &manager{log: log.Default, backend: backend}
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
	return nil
}

func (m *manager) Handler() http.Handler {
	return m.handler
}
