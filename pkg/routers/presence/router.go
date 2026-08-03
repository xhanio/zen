package presence

import (
	_ "embed"

	fapi "github.com/xhanio/framingo/pkg/types/api"
	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/nameutil"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/model"
)

var _ fapi.Router = (*router)(nil)

//go:embed router.yaml
var config []byte

type router struct {
	name     string
	log      log.Logger
	presence model.Presence
	delivery model.Delivery
}

func newRouter(pres model.Presence, del model.Delivery, logger log.Logger) *router {
	r := &router{log: logger, presence: pres, delivery: del}
	r.name = nameutil.Name(r)
	return r
}

func New(pres model.Presence, del model.Delivery, logger log.Logger) fapi.Router {
	return newRouter(pres, del, logger)
}

func (r *router) Name() string {
	return r.name
}

func (r *router) Dependencies() []common.Service {
	deps := []common.Service{r.presence}
	if r.delivery != nil {
		deps = append(deps, r.delivery)
	}
	return deps
}

func (r *router) Config() []byte { return config }

func (r *router) Handlers() map[string]any {
	handlers := api.DiscoverHandlers(r)
	r.log.Debugf("router %s parsed %d handler(s)", r.Name(), len(handlers))
	return handlers
}
