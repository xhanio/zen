package zenmcp

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/services/api/server"
	"github.com/xhanio/framingo/pkg/services/supervisor"
	"github.com/xhanio/framingo/pkg/utils/log"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
	"github.com/xhanio/zen/pkg/services/mcp"
	"github.com/xhanio/zen/pkg/utils/infra"
)

func (m *manager) initServices() error {
	m.log = log.New(
		log.WithLevel(m.config.GetInt("log.level")),
		log.WithFileWriter(
			m.config.GetString("log.file"),
			m.config.GetInt("log.rotation.max_size"),
			m.config.GetInt("log.rotation.max_backups"),
			m.config.GetInt("log.rotation.max_age"),
		),
	)
	infra.Debug = (m.log.Level() == zapcore.DebugLevel)

	m.services = supervisor.New(m.config, supervisor.WithLogger(m.log))

	m.backend = zenbackend.New(
		m.config.GetString("backend.url"),
		zenbackend.WithTimeout(30*time.Second),
	)

	m.mcpSvc = mcp.New(m.backend, mcp.WithLogger(m.log))

	m.api = server.New(server.WithLogger(m.log))

	for name := range m.config.GetStringMap("api") {
		opts := []server.ServerOption{
			server.WithEndpoint(
				m.config.GetString(fmt.Sprintf("api.%s.host", name)),
				m.config.GetUint(fmt.Sprintf("api.%s.port", name)),
				m.config.GetString(fmt.Sprintf("api.%s.prefix", name)),
			),
		}
		if err := m.api.Add(name, opts...); err != nil {
			return errors.Wrap(err)
		}
	}
	return nil
}
