package group

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
	name string
	log  log.Logger
	svc  model.Group
}

func New(svc model.Group, logger log.Logger) fapi.Router {
	return &router{
		log: logger,
		svc: svc,
	}
}

// RouterForTest is a type alias exposing the unexported router for HTTP-level
// unit tests in this package's _test.go files.
type RouterForTest = router

// NewForTest constructs a router for unit tests without requiring a logger.
func NewForTest(svc model.Group) *RouterForTest {
	return &router{svc: svc}
}

func (r *router) Name() string {
	if r.name == "" {
		r.name = path.Join(reflectutil.Locate(r))
	}
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
