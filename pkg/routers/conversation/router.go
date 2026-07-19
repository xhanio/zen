package conversation

import (
	_ "embed"
	"path"
	"time"

	fapi "github.com/xhanio/framingo/pkg/types/api"
	"github.com/xhanio/framingo/pkg/types/common"
	fmodel "github.com/xhanio/framingo/pkg/types/model"
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
	svc      model.Conversation
	bus      fmodel.MessageBus
	presence model.Presence
	delivery model.Delivery

	// regTimeout overrides defaultRegistrationTimeout. Zero means default.
	regTimeout time.Duration
}

func New(svc model.Conversation, bus fmodel.MessageBus, pres model.Presence, del model.Delivery, logger log.Logger) fapi.Router {
	return &router{log: logger, svc: svc, bus: bus, presence: pres, delivery: del}
}

type RouterForTest = router

// Every constructor here supplies a real logger. StreamWS logs on the
// registration-failure path and Handlers() logs its handler count, and a nil
// logger would panic in either — a test-only crash in code the tests exist to
// exercise.
func NewForTest(svc model.Conversation) *RouterForTest {
	return &router{log: log.Default, svc: svc}
}

func NewForTestWithBus(svc model.Conversation, bus fmodel.MessageBus) *RouterForTest {
	return &router{log: log.Default, svc: svc, bus: bus}
}

func NewForTestWithPresence(svc model.Conversation, bus fmodel.MessageBus, pres model.Presence) *RouterForTest {
	return &router{log: log.Default, svc: svc, bus: bus, presence: pres}
}

func NewForTestWithDelivery(svc model.Conversation, bus fmodel.MessageBus, pres model.Presence, del model.Delivery) *RouterForTest {
	return &router{log: log.Default, svc: svc, bus: bus, presence: pres, delivery: del}
}

// SetRegistrationTimeout shrinks the silence budget for tests that assert on an
// unregistered socket. Set it before the server starts; it is read by the
// handler goroutine.
func (r *RouterForTest) SetRegistrationTimeout(d time.Duration) { r.regTimeout = d }

// SetLogger swaps the logger so a test can read what the handler actually
// emitted. Set it before the server starts.
func (r *RouterForTest) SetLogger(l log.Logger) { r.log = l }

func (r *router) Name() string {
	if r.name == "" {
		r.name = path.Join(reflectutil.Locate(r))
	}
	return r.name
}

func (r *router) Dependencies() []common.Service {
	deps := []common.Service{r.svc}
	if r.bus != nil {
		deps = append(deps, r.bus)
	}
	if r.presence != nil {
		deps = append(deps, r.presence)
	}
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
