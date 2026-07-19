package zenbackend

import (
	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/types/api"

	cardRouter "github.com/xhanio/zen/pkg/routers/card"
	conversationRouter "github.com/xhanio/zen/pkg/routers/conversation"
	groupRouter "github.com/xhanio/zen/pkg/routers/group"
	healthRouter "github.com/xhanio/zen/pkg/routers/health"
	referenceRouter "github.com/xhanio/zen/pkg/routers/reference"
	searchRouter "github.com/xhanio/zen/pkg/routers/search"
	tagRouter "github.com/xhanio/zen/pkg/routers/tag"
	presenceRouter "github.com/xhanio/zen/pkg/routers/presence"
	trashRouter "github.com/xhanio/zen/pkg/routers/trash"
)

func (m *manager) initAPI() error {
	middlewares := []api.Middleware{}
	routers := []api.Router{
		healthRouter.New(m.db, m.log),
		groupRouter.New(m.group, m.log),
		tagRouter.New(m.tag, m.log),
		cardRouter.New(m.card, m.log),
		conversationRouter.New(m.conversation, m.bus, m.presence, m.delivery, m.log),
		presenceRouter.New(m.presence, m.delivery, m.log),
		searchRouter.New(m.search, m.log),
		trashRouter.New(m.card, m.log),
		referenceRouter.New(m.reference, m.log),
	}

	if err := m.api.RegisterMiddlewares(middlewares...); err != nil {
		return errors.Wrap(err)
	}
	if err := m.api.RegisterRouters(routers...); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
