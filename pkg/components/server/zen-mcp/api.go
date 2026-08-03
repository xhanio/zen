package zenmcp

import (
	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/types/api"

	"github.com/xhanio/zen/pkg/middlewares/throttle"
	mcpRouter "github.com/xhanio/zen/pkg/routers/mcp"
)

func (m *manager) initAPI() error {
	middlewares := []api.Middleware{
		// See zen-backend's initAPI: attachment lives in router.yaml, the
		// limit in api.<name>.middlewares.
		throttle.New(),
	}
	routers := []api.Router{
		mcpRouter.New(m.mcpSvc, m.log),
	}

	if err := m.api.RegisterMiddlewares(middlewares...); err != nil {
		return errors.Wrap(err)
	}
	if err := m.api.RegisterRouters(routers...); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
