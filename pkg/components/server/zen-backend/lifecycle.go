package zenbackend

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/xhanio/errors"
)

func (m *manager) Init(ctx context.Context) error {
	if err := m.initConfig(); err != nil {
		return errors.Wrap(err)
	}

	if err := m.initServices(); err != nil {
		return errors.Wrap(err)
	}

	// infra services
	m.services.Register(
		m.db,
		m.repository,
		m.pubsub,
		m.bus,
	)

	// business services. Every one goes in, including those with no lifecycle
	// work today: the supervisor is also what reports them through Info and
	// the health probes, and an Init added later would otherwise never run.
	m.services.Register(
		m.conversation,
		m.group,
		m.tag,
		m.card,
		m.search,
		m.reference,
		m.snapshot,
		m.presence,
		m.delivery,
	)

	if err := m.services.TopoSort(); err != nil {
		return errors.Wrap(err)
	}

	// API server registered last so it starts after dependencies are up.
	m.services.Register(
		m.api,
	)

	// Register all services with the messagebus; those implementing neither
	// MessageHandler nor RawMessageHandler are skipped automatically.
	for _, svc := range m.services.Services() {
		m.bus.Register(svc)
	}

	if err := m.services.Init(ctx); err != nil {
		return errors.Wrap(err)
	}

	if err := m.runV12PostInit(ctx); err != nil {
		return errors.Wrap(err)
	}

	if err := m.runV110PostInit(ctx); err != nil {
		return errors.Wrap(err)
	}

	if err := m.initAPI(); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (m *manager) Start(ctx context.Context) error {
	if m.cancel != nil {
		m.log.Warn("manager already started, skipping")
		return nil
	}
	pport := m.config.GetUint("pprof.port")
	if pport != 0 {
		go func() {
			m.log.Infof("enable pprof on port %d", pport)
			err := http.ListenAndServe(fmt.Sprintf("localhost:%d", pport), nil)
			if err != nil {
				panic(err)
			}
		}()
	}
	if err := m.services.Start(ctx); err != nil {
		return err
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.listenSignals(m.ctx)
	return nil
}

func (m *manager) Stop(wait bool) error {
	if err := m.services.Stop(wait); err != nil {
		m.log.Error(err)
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	return nil
}

func (m *manager) Info(w io.Writer, debug bool) {
	m.services.Info(w, debug)
}
