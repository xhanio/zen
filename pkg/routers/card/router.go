package card

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
	name string
	log  log.Logger
	svc  model.Card
}

func newRouter(svc model.Card, logger log.Logger) *router {
	r := &router{log: logger, svc: svc}
	r.name = nameutil.Name(r)
	return r
}

func New(svc model.Card, logger log.Logger) fapi.Router {
	return newRouter(svc, logger)
}

func (r *router) Name() string {
	return r.name
}

func (r *router) Dependencies() []common.Service {
	return []common.Service{r.svc}
}

func (r *router) Config() []byte { return config }

func (r *router) Handlers() map[string]any {
	handlers := api.DiscoverHandlers(r)
	r.log.Debugf("router %s parsed %d handler(s)", r.Name(), len(handlers))
	return handlers
}
