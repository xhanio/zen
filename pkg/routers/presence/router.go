package presence

import (
	_ "embed"
	"path"

	fapi "github.com/xhanio/framingo/pkg/types/api"
	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/reflectutil"

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

func New(pres model.Presence, del model.Delivery, logger log.Logger) fapi.Router {
	return &router{log: logger, presence: pres, delivery: del}
}

type RouterForTest = router

// NewForTest supplies a real logger: SessionsWS logs on the write-failure path.
func NewForTest(pres model.Presence, del model.Delivery) *RouterForTest {
	return &router{log: log.Default, presence: pres, delivery: del}
}

func (r *router) Name() string {
	if r.name == "" {
		r.name = path.Join(reflectutil.Locate(r))
	}
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
