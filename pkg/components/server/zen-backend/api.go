package zenbackend

import (
	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/types/api"

	"github.com/xhanio/zen/pkg/middlewares/throttle"
	cardRouter "github.com/xhanio/zen/pkg/routers/card"
	conversationRouter "github.com/xhanio/zen/pkg/routers/conversation"
	groupRouter "github.com/xhanio/zen/pkg/routers/group"
	healthRouter "github.com/xhanio/zen/pkg/routers/health"
	presenceRouter "github.com/xhanio/zen/pkg/routers/presence"
	referenceRouter "github.com/xhanio/zen/pkg/routers/reference"
	searchRouter "github.com/xhanio/zen/pkg/routers/search"
	snapshotRouter "github.com/xhanio/zen/pkg/routers/snapshot"
	tagRouter "github.com/xhanio/zen/pkg/routers/tag"
	trashRouter "github.com/xhanio/zen/pkg/routers/trash"
)

func (m *manager) initAPI() error {
	middlewares := []api.Middleware{
		// Routers opt in through router.yaml, where a handler may also carry
		// its own limit under the middleware's name; the server's middleware
		// defaults (api.<name>.middlewares in config.yaml) cover the rest.
		// With no limit anywhere it passes everything, so attaching it is
		// safe with no config at all.
		throttle.New(),
	}
	routers := []api.Router{
		healthRouter.New(m.services, m.log),
		groupRouter.New(m.group, m.log),
		tagRouter.New(m.tag, m.log),
		cardRouter.New(m.card, m.log),
		conversationRouter.New(m.conversation, m.bus, m.presence, m.delivery, m.log),
		presenceRouter.New(m.presence, m.delivery, m.log),
		searchRouter.New(m.search, m.log),
		trashRouter.New(m.card, m.log),
		referenceRouter.New(m.reference, m.log),
		snapshotRouter.New(m.snapshot, m.log),
	}

	if err := m.api.RegisterMiddlewares(middlewares...); err != nil {
		return errors.Wrap(err)
	}
	if err := m.api.RegisterRouters(routers...); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
